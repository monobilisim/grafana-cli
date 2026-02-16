package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"gcli/internal/config"
	"gcli/internal/datasource"

	"github.com/spf13/cobra"
)

var dsCmd = &cobra.Command{
	Use:   "ds",
	Short: "Manage Grafana data sources",
}

// ds list
var dsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List data sources for the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {

		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}

		url := fmt.Sprintf("%s/api/datasources", profile.URL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)

		// Set Org ID header if active org is selected
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
			return fmt.Errorf("list failed: %s", resp.Status)
		}

		details, _ := cmd.Flags().GetBool("details")
		if details {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, body, "", "  "); err != nil {
				fmt.Printf("Status: %s\n%s\n", resp.Status, string(body))
			} else {
				fmt.Printf("Status: %s\n%s\n", resp.Status, pretty.String())
			}
			return nil
		}

		// Tabular output
		var dss []struct {
			ID    int    `json:"id"`
			OrgID int    `json:"orgId"`
			Name  string `json:"name"`
			Type  string `json:"type"`
			URL   string `json:"url"`
		}
		if err := json.Unmarshal(body, &dss); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("%-5s %-5s %-30s %-15s %s\n", "ID", "OrgID", "Name", "Type", "URL")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, ds := range dss {
			fmt.Printf("%-5d %-5d %-30s %-15s %s\n", ds.ID, ds.OrgID, ds.Name, ds.Type, ds.URL)
		}
		return nil
	},
}

// ds rm <name|id>
var dsRmCmd = &cobra.Command{
	Use:   "rm [NAME|ID]",
	Short: "Delete a data source by name or ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrName := args[0]

		// Resolve ID first to check existence and get numeric ID
		// Note: Grafana delete API typically uses ID (integer) or name (GET /api/datasources/name/:name)
		// But deletion via ID is safer and standard.
		// ResolveID fetches list and finds by name or ID string match.
		// It returns int.

		// However, ResolveID is in internal package, let's use it.
		// Need to import "gcli/internal/datasource" -- done above.

		// Actually, I need to check how to call ResolveID from here.
		// Since I implemented it in `internal/datasource`, I can use `datasource.ResolveID`.
		// But wait, ResolveID needs config access, which uses `config.GetActive`. That's fine.

		// However, I need to pass the resolved ID to the delete call.
		// Wait, ResolveID returns the int ID.

		// Let's implement generic logic here.

		// First, try resolving.
		// Since ResolveID is in internal package which imports config, and config is internal, that's fine.
		// But wait, `datasource.ResolveID` is what I implemented previously.

		// Let's call it.
		id, err := datasource.ResolveID(idOrName)
		if err != nil {
			return err
		}

		profile, _ := config.GetActive() // Checked inside ResolveID, but need it for URL building here too.
		// Actually need to re-fetch config or trust ResolveID succeed meant config exists.
		// Safer to re-fetch.
		if profile == nil {
			// Should not happen if ResolveID succeeded
			return fmt.Errorf("active profile lost")
		}

		url := fmt.Sprintf("%s/api/datasources/%d", profile.URL, id)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("delete failed: %s %s", resp.Status, string(body))
		}

		fmt.Printf("Data source deleted: %s (ID: %d)\n", idOrName, id)
		return nil
	},
}

// ds create --file <path> | --name ...
var dsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")

		var payload []byte

		if file != "" {
			var err error
			payload, err = ioutil.ReadFile(file)
			if err != nil {
				return err
			}
		} else {
			name, _ := cmd.Flags().GetString("name")
			dsType, _ := cmd.Flags().GetString("type")
			urlFlag, _ := cmd.Flags().GetString("url")
			access, _ := cmd.Flags().GetString("access")
			basicAuth, _ := cmd.Flags().GetBool("basicAuth")

			if name == "" || dsType == "" || urlFlag == "" || access == "" {
				return fmt.Errorf("required flags missing: --name, --type, --url, --access (or use --file)")
			}

			data := map[string]interface{}{
				"name":      name,
				"type":      dsType,
				"url":       urlFlag,
				"access":    access,
				"basicAuth": basicAuth,
			}
			var err error
			payload, err = json.Marshal(data)
			if err != nil {
				return err
			}
		}

		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set")
		}

		url := fmt.Sprintf("%s/api/datasources", profile.URL)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
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
			return fmt.Errorf("create failed: %s %s", resp.Status, string(body))
		}

		fmt.Printf("Status: %s\n%s\n", resp.Status, string(body))
		return nil
	},
}

// ds read <name|id>
var dsReadCmd = &cobra.Command{
	Use:   "read [NAME|ID]",
	Short: "Read a data source definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrName := args[0]
		id, err := datasource.ResolveID(idOrName)
		if err != nil {
			return err
		}

		profile, _ := config.GetActive() // Checked inside ResolveID

		url := fmt.Sprintf("%s/api/datasources/%d", profile.URL, id)
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
			return fmt.Errorf("read failed: %s %s", resp.Status, string(body))
		}

		pretty, err := datasource.PrettyPrintJSON(body)
		if err != nil {
			fmt.Println(string(body))
		} else {
			fmt.Println(string(pretty))
		}
		return nil
	},
}

// ds update [NAME|ID] --file <path> OR interactive
var dsUpdateCmd = &cobra.Command{
	Use:   "update [NAME|ID]",
	Short: "Update a data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")

		var dsID int
		var payload []byte
		var err error

		if file != "" {
			payload, err = ioutil.ReadFile(file)
			if err != nil {
				return err
			}

			// Try to extract ID from file content
			var data map[string]interface{}
			if err := json.Unmarshal(payload, &data); err != nil {
				return fmt.Errorf("invalid json file: %w", err)
			}

			if idVal, ok := data["id"]; ok {
				// Handle float64 (default for json numbers) or int
				switch v := idVal.(type) {
				case float64:
					dsID = int(v)
				case int:
					dsID = v
				case string:
					// Should be int but if string, try parsing? No, let's assume valid JSON int.
					return fmt.Errorf("id in file must be a number")
				}
			}

			// If ID not found in file and argument provided, resolve argument
			if dsID == 0 && len(args) > 0 {
				dsID, err = datasource.ResolveID(args[0])
				if err != nil {
					return err
				}
			} else if dsID == 0 {
				return fmt.Errorf("cannot determine datasource ID from file or arguments")
			}

		} else {
			// Interactive mode
			if len(args) == 0 {
				return fmt.Errorf("please specify datasource name/ID for interactive update or use --file")
			}

			dsID, err = datasource.ResolveID(args[0])
			if err != nil {
				return err
			}

			profile, _ := config.GetActive()

			// Fetch current config
			readURL := fmt.Sprintf("%s/api/datasources/%d", profile.URL, dsID)
			req, err := http.NewRequest(http.MethodGet, readURL, nil)
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
				return fmt.Errorf("failed to fetch current config: %s", resp.Status)
			}

			pretty, _ := datasource.PrettyPrintJSON(body)

			// Open in editor
			tmpFile, err := ioutil.TempFile("", "gcli-ds-*.json")
			if err != nil {
				return err
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(pretty); err != nil {
				return err
			}
			if err := tmpFile.Close(); err != nil {
				return err
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			// Invoke editor
			cmd := exec.Command(editor, tmpFile.Name())
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			// Read modified content
			payload, err = ioutil.ReadFile(tmpFile.Name())
			if err != nil {
				return err
			}
		}

		profile, err := config.GetActive()
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/datasources/%d", profile.URL, dsID)
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
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
		fmt.Printf("Status: %s\n%s\n", resp.Status, string(body))

		return nil
	},
}

func init() {
	dsCmd.AddCommand(dsListCmd)
	dsCmd.AddCommand(dsRmCmd)
	dsCmd.AddCommand(dsCreateCmd)
	dsCmd.AddCommand(dsReadCmd)
	dsCmd.AddCommand(dsUpdateCmd)

	// List command flags
	dsListCmd.Flags().Bool("details", false, "Show detailed JSON output")

	// Create command flags
	dsCreateCmd.Flags().String("file", "", "JSON file containing data source definition")
	dsCreateCmd.Flags().String("name", "", "Name of data source")
	dsCreateCmd.Flags().String("type", "", "Type of data source (e.g., graphite, prometheus)")
	dsCreateCmd.Flags().String("url", "", "URL of data source")
	dsCreateCmd.Flags().String("access", "proxy", "Access mode (proxy or direct)")
	dsCreateCmd.Flags().Bool("basicAuth", false, "Enable basic auth")

	// Update command flags
	dsUpdateCmd.Flags().String("file", "", "JSON file containing data source update definitions")
}
