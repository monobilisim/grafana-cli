package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gcliv2/internal/config"
	"gcliv2/internal/datasource"

	"github.com/spf13/cobra"
)

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "Manage Grafana dashboards",
}

// dash list
var dashListCmd = &cobra.Command{
	Use:   "list",
	Short: "List dashboards for the active profile and organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}

		// Use the search API to list dashboards
		url := fmt.Sprintf("%s/api/search?type=dash-db", profile.URL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)

		// Set Org ID header
		activeOrg, _ := config.GetActiveOrg()
		if activeOrg != "" {
			req.Header.Set("X-Grafana-Org-Id", activeOrg)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("list failed: %s %s", resp.Status, string(body))
		}

		details, _ := cmd.Flags().GetBool("details")
		if details {
			pretty, _ := datasource.PrettyPrintJSON(body)
			fmt.Printf("Status: %s\n%s\n", resp.Status, string(pretty))
			return nil
		}

		var items []struct {
			UID         string   `json:"uid"`
			Title       string   `json:"title"`
			FolderTitle string   `json:"folderTitle"`
			Tags        []string `json:"tags"`
		}
		if err := json.Unmarshal(body, &items); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("%-40s %-30s %-20s %s\n", "UID", "Title", "Folder", "Tags")
		fmt.Println("------------------------------------------------------------------------------------------------------------------------")
		for _, item := range items {
			tags := ""
			if len(item.Tags) > 0 {
				tags = fmt.Sprintf("%v", item.Tags)
			}
			folder := item.FolderTitle
			if folder == "" {
				folder = "General"
			}
			fmt.Printf("%-40s %-30s %-20s %s\n", item.UID, item.Title, folder, tags)
		}

		return nil
	},
}

// dash read [UID]
var dashReadCmd = &cobra.Command{
	Use:   "read [UID]",
	Short: "Read a dashboard definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uid := args[0]
		external, _ := cmd.Flags().GetBool("external")

		profile, err := config.GetActive()
		if err != nil {
			return err
		}

		activeOrg, _ := config.GetActiveOrg()

		if external {
			// For external export, we use the export API
			url := fmt.Sprintf("%s/api/dashboards/uid/%s", profile.URL, uid)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			req.SetBasicAuth(profile.User, profile.Pass)
			if activeOrg != "" {
				req.Header.Set("X-Grafana-Org-Id", activeOrg)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("read failed: %s %s", resp.Status, string(body))
			}

			var dashData struct {
				Dashboard json.RawMessage `json:"dashboard"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&dashData); err != nil {
				return err
			}

			var dashObj map[string]interface{}
			if err := json.Unmarshal(dashData.Dashboard, &dashObj); err != nil {
				return fmt.Errorf("manual export failed: could not parse dashboard JSON: %w", err)
			}

			// 1. Fetch all datasources to map UIDs to names/types
			dsURL := fmt.Sprintf("%s/api/datasources", profile.URL)
			dsReq, _ := http.NewRequest(http.MethodGet, dsURL, nil)
			dsReq.SetBasicAuth(profile.User, profile.Pass)
			if activeOrg != "" {
				dsReq.Header.Set("X-Grafana-Org-Id", activeOrg)
			}
			dsResp, err := http.DefaultClient.Do(dsReq)
			if err != nil {
				return fmt.Errorf("failed to fetch datasources for mapping: %w", err)
			}
			defer dsResp.Body.Close()
			var allDS []struct {
				UID  string `json:"uid"`
				Name string `json:"name"`
				Type string `json:"type"`
			}
			json.NewDecoder(dsResp.Body).Decode(&allDS)

			dsMap := make(map[string]struct{ Name, Type string })
			for _, ds := range allDS {
				dsMap[ds.UID] = struct{ Name, Type string }{Name: ds.Name, Type: ds.Type}
			}

			// 2. Discover all datasource UIDs used in the dashboard
			usedUIDs := make(map[string]bool)
			discoverDatasourceUIDs(dashObj, usedUIDs)

			// 3. Prepare __inputs and perform replacements
			var inputs []map[string]interface{}
			requires := []map[string]interface{}{
				{"type": "grafana", "id": "grafana", "name": "Grafana", "version": "1.0.0"}, // Dummy version
			}
			pluginMap := make(map[string]bool)

			jsonStr := string(dashData.Dashboard)
			for uid := range usedUIDs {
				if ds, ok := dsMap[uid]; ok {
					varName := "DS_" + strings.ToUpper(strings.ReplaceAll(ds.Name, "-", "_"))
					varName = strings.ReplaceAll(varName, " ", "_")

					inputs = append(inputs, map[string]interface{}{
						"name":     varName,
						"label":    ds.Name,
						"type":     "datasource",
						"pluginId": ds.Type,
					})

					if !pluginMap[ds.Type] {
						requires = append(requires, map[string]interface{}{
							"type":    "datasource",
							"id":      ds.Type,
							"name":    ds.Type,
							"version": "1.0.0",
						})
						pluginMap[ds.Type] = true
					}

					// Replace UID with ${VAR_NAME}
					jsonStr = strings.ReplaceAll(jsonStr, fmt.Sprintf(`"%s"`, uid), fmt.Sprintf(`"${%s}"`, varName))
				}
			}

			// 4. Final assembly
			var finalDash map[string]interface{}
			json.Unmarshal([]byte(jsonStr), &finalDash)

			// Strip instance-specifics
			delete(finalDash, "id")
			delete(finalDash, "uid")
			delete(finalDash, "version")

			exportOutput := map[string]interface{}{
				"__inputs":   inputs,
				"__requires": requires,
			}
			// Merge everything back
			for k, v := range finalDash {
				exportOutput[k] = v
			}

			pretty, _ := json.MarshalIndent(exportOutput, "", "  ")
			fmt.Println(string(pretty))
			return nil
		}

		// Standard read
		url := fmt.Sprintf("%s/api/dashboards/uid/%s", profile.URL, uid)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)
		if activeOrg != "" {
			req.Header.Set("X-Grafana-Org-Id", activeOrg)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("read failed: %s %s", resp.Status, string(body))
		}

		// Typically we want just the dashboard metadata, not the whole wrapper
		var wrapper struct {
			Dashboard json.RawMessage `json:"dashboard"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			// Fallback to printing whole thing if we can't parse wrapper
			pretty, _ := datasource.PrettyPrintJSON(body)
			fmt.Println(string(pretty))
			return nil
		}

		pretty, _ := datasource.PrettyPrintJSON(wrapper.Dashboard)
		fmt.Println(string(pretty))

		return nil
	},
}

// dash rm [UID]
var dashRmCmd = &cobra.Command{
	Use:   "rm [UID]",
	Short: "Delete a dashboard by UID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uid := args[0]

		profile, err := config.GetActive()
		if err != nil {
			return err
		}

		activeOrg, _ := config.GetActiveOrg()

		url := fmt.Sprintf("%s/api/dashboards/uid/%s", profile.URL, uid)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)
		if activeOrg != "" {
			req.Header.Set("X-Grafana-Org-Id", activeOrg)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("delete failed: %s %s", resp.Status, string(body))
		}

		fmt.Printf("Dashboard deleted: %s\n", uid)
		return nil
	},
}

// dash update [UID]
var dashUpdateCmd = &cobra.Command{
	Use:   "update [UID]",
	Short: "Update a dashboard interactively",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uid := args[0]
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		activeOrg, _ := config.GetActiveOrg()

		// Fetch current dashboard
		url := fmt.Sprintf("%s/api/dashboards/uid/%s", profile.URL, uid)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.SetBasicAuth(profile.User, profile.Pass)
		if activeOrg != "" {
			req.Header.Set("X-Grafana-Org-Id", activeOrg)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch dashboard: %s", resp.Status)
		}

		var wrapper struct {
			Dashboard json.RawMessage `json:"dashboard"`
			Metadata  struct {
				FolderUID string `json:"folderUid"`
			} `json:"meta"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
			return err
		}

		pretty, _ := datasource.PrettyPrintJSON(wrapper.Dashboard)
		content := string(pretty)

		var lastError string
		for {
			tmpFile, err := os.CreateTemp("", "gcli-dash-*.json")
			if err != nil {
				return err
			}
			defer os.Remove(tmpFile.Name())

			if lastError != "" {
				tmpFile.WriteString("// ERROR: " + lastError + "\n")
				tmpFile.WriteString("// Fix the error below and save to retry.\n\n")
			}
			tmpFile.WriteString(content)
			tmpFile.Close()

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			ecmd := exec.Command(editor, tmpFile.Name())
			ecmd.Stdin = os.Stdin
			ecmd.Stdout = os.Stdout
			ecmd.Stderr = os.Stderr
			if err := ecmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			// Read back
			updatedBytes, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				return err
			}

			// Strip comments
			var cleanLines []string
			lines := bytes.Split(updatedBytes, []byte("\n"))
			for _, line := range lines {
				trimmed := bytes.TrimSpace(line)
				if bytes.HasPrefix(trimmed, []byte("//")) || bytes.HasPrefix(trimmed, []byte("#")) {
					continue
				}
				cleanLines = append(cleanLines, string(line))
			}
			cleanJSON := strings.Join(cleanLines, "\n")
			if strings.TrimSpace(cleanJSON) == "" {
				fmt.Println("No content, skipping update.")
				return nil
			}

			// Verify JSON
			var dashObj map[string]interface{}
			if err := json.Unmarshal([]byte(cleanJSON), &dashObj); err != nil {
				lastError = err.Error()
				content = cleanJSON
				continue
			}

			// Prepare update payload
			payload := map[string]interface{}{
				"dashboard": dashObj,
				"overwrite": true,
			}
			if wrapper.Metadata.FolderUID != "" {
				payload["folderUid"] = wrapper.Metadata.FolderUID
			}

			payloadBytes, _ := json.Marshal(payload)
			updateURL := fmt.Sprintf("%s/api/dashboards/db", profile.URL)
			ureq, _ := http.NewRequest(http.MethodPost, updateURL, bytes.NewReader(payloadBytes))
			ureq.Header.Set("Content-Type", "application/json")
			ureq.SetBasicAuth(profile.User, profile.Pass)
			if activeOrg != "" {
				ureq.Header.Set("X-Grafana-Org-Id", activeOrg)
			}

			uresp, err := http.DefaultClient.Do(ureq)
			if err != nil {
				return err
			}
			defer uresp.Body.Close()

			ubody, _ := io.ReadAll(uresp.Body)
			if uresp.StatusCode != http.StatusOK {
				lastError = fmt.Sprintf("%s: %s", uresp.Status, string(ubody))
				content = cleanJSON
				continue
			}

			fmt.Printf("Dashboard updated successfully.\n%s\n", string(ubody))
			break
		}

		return nil
	},
}

// dash create --file [PATH]
var dashCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a dashboard from a file",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return fmt.Errorf("--file flag is required")
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		var dashRaw map[string]interface{}
		if err := json.Unmarshal(data, &dashRaw); err != nil {
			return fmt.Errorf("invalid dashboard JSON: %w", err)
		}

		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		activeOrg, _ := config.GetActiveOrg()

		// Check for external template inputs (exported dashboards often have this)
		if inputs, ok := dashRaw["__inputs"].([]interface{}); ok && len(inputs) > 0 {
			fmt.Println("This dashboard is an external template and requires datasource mapping.")

			// Map to hold our final mappings
			mappings := make(map[string]string)

			// Fetch available datasources for the active org
			dsURL := fmt.Sprintf("%s/api/datasources", profile.URL)
			dsReq, _ := http.NewRequest(http.MethodGet, dsURL, nil)
			dsReq.SetBasicAuth(profile.User, profile.Pass)
			if activeOrg != "" {
				dsReq.Header.Set("X-Grafana-Org-Id", activeOrg)
			}
			dsResp, err := http.DefaultClient.Do(dsReq)
			if err != nil {
				return err
			}
			defer dsResp.Body.Close()
			var availableDS []struct {
				UID  string `json:"uid"`
				Name string `json:"name"`
				Type string `json:"type"`
			}
			json.NewDecoder(dsResp.Body).Decode(&availableDS)

			reader := bufio.NewReader(os.Stdin)

			for _, input := range inputs {
				im := input.(map[string]interface{})
				inputType, _ := im["type"].(string)
				inputName, _ := im["name"].(string)
				inputLabel, _ := im["label"].(string)
				pluginID, _ := im["pluginId"].(string)

				if inputType == "datasource" {
					fmt.Printf("\nSelect datasource for '%s' (%s, plugin: %s):\n", inputLabel, inputName, pluginID)

					// Filter datasources by pluginID (type)
					var filtered []int
					for i, ds := range availableDS {
						if ds.Type == pluginID {
							fmt.Printf("[%d] %s (UID: %s)\n", len(filtered)+1, ds.Name, ds.UID)
							filtered = append(filtered, i)
						}
					}

					if len(filtered) == 0 {
						return fmt.Errorf("no datasources found for type %s", pluginID)
					}

					for {
						fmt.Print("Enter number: ")
						inputStr, _ := reader.ReadString('\n')
						inputStr = strings.TrimSpace(inputStr)
						idx, err := strconv.Atoi(inputStr)
						if err == nil && idx > 0 && idx <= len(filtered) {
							mappings[inputName] = availableDS[filtered[idx-1]].UID
							break
						}
						fmt.Println("Invalid selection. Please try again.")
					}
				}
			}

			// Now we need to replace occurrences of these template variables in the dashboard spec.
			// Actually, external templates use ${VAR_NAME} syntax.
			// We can do a string replacement on the entire raw JSON for simplicity if we are careful,
			// but better to walk the tree or use a regex.
			// For simplicity and speed in a CLI, string replacement of ${NAME} is common for Grafana templates.

			jsonStr := string(data)
			for name, uid := range mappings {
				target := fmt.Sprintf("${%s}", name)
				jsonStr = strings.ReplaceAll(jsonStr, target, uid)
			}

			// Re-parse it
			if err := json.Unmarshal([]byte(jsonStr), &dashRaw); err != nil {
				return fmt.Errorf("failed to re-parse dashboard after mapping: %w", err)
			}

			// Remove __inputs and __requires as they are for templates, not for direct import
			delete(dashRaw, "__inputs")
			delete(dashRaw, "__requires")
		}

		// Interactive Prompts for Title and UID
		fmt.Println() // New line for spacing
		reader := bufio.NewReader(os.Stdin)

		currTitle := dashRaw["title"]
		fmt.Printf("Change title? (current: %v) [y/N]: ", currTitle)
		ans, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) == "y" {
			fmt.Print("Enter new title: ")
			newName, _ := reader.ReadString('\n')
			dashRaw["title"] = strings.TrimSpace(newName)
		}

		currUID := dashRaw["uid"]
		displayUID := currUID
		if displayUID == nil || displayUID == "" {
			displayUID = "(none, will be auto-generated)"
		}
		fmt.Printf("Change UID? (current: %v) [y/N]: ", displayUID)
		ans, _ = reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) == "y" {
			fmt.Print("Enter new UID: ")
			newUID, _ := reader.ReadString('\n')
			dashRaw["uid"] = strings.TrimSpace(newUID)
		}

		// Prepare create payload
		payload := map[string]interface{}{
			"dashboard": dashRaw,
			"overwrite": false,
		}

		payloadBytes, _ := json.Marshal(payload)
		createURL := fmt.Sprintf("%s/api/dashboards/db", profile.URL)
		req, _ := http.NewRequest(http.MethodPost, createURL, bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(profile.User, profile.Pass)
		if activeOrg != "" {
			req.Header.Set("X-Grafana-Org-Id", activeOrg)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("create failed: %s %s", resp.Status, string(body))
		}

		fmt.Printf("Dashboard created successfully.\n%s\n", string(body))
		return nil
	},
}

func init() {
	dashCmd.AddCommand(dashListCmd)
	dashCmd.AddCommand(dashReadCmd)
	dashCmd.AddCommand(dashRmCmd)
	dashCmd.AddCommand(dashUpdateCmd)
	dashCmd.AddCommand(dashCreateCmd)
	dashListCmd.Flags().Bool("details", false, "Show detailed JSON output")
	dashReadCmd.Flags().Bool("external", false, "Export dashboard for sharing (external template)")
	dashCreateCmd.Flags().String("file", "", "JSON file containing dashboard definition")
}

func discoverDatasourceUIDs(v interface{}, uids map[string]bool) {
	switch val := v.(type) {
	case map[string]interface{}:
		// Check for "datasource" or "datasource": { "uid": "..." }
		if ds, ok := val["datasource"]; ok {
			switch dsv := ds.(type) {
			case string:
				if dsv != "" && dsv != "grafana" && !strings.HasPrefix(dsv, "$") {
					uids[dsv] = true
				}
			case map[string]interface{}:
				if uid, ok := dsv["uid"].(string); ok {
					if uid != "" && uid != "grafana" && !strings.HasPrefix(uid, "$") {
						uids[uid] = true
					}
				}
			}
		}
		for _, v2 := range val {
			discoverDatasourceUIDs(v2, uids)
		}
	case []interface{}:
		for _, v2 := range val {
			discoverDatasourceUIDs(v2, uids)
		}
	}
}
