package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gcliv2/internal/config"

	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage Grafana organizations",
}

// org list
var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organizations for the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}
		url := fmt.Sprintf("%s/api/orgs", profile.URL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

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
		var orgs []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &orgs); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("%-5s %s\n", "ID", "Name")
		fmt.Println("----------------------------------------")
		for _, org := range orgs {
			fmt.Printf("%-5d %s\n", org.ID, org.Name)
		}
		return nil
	},
}

// org use <orgID>
var orgUseCmd = &cobra.Command{
	Use:   "use [orgID]",
	Short: "Select an organization for subsequent operations",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgIDOrName := args[0]
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}
		// Fetch organization list
		url := fmt.Sprintf("%s/api/orgs", profile.URL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch org list: %s", resp.Status)
		}
		body, _ := io.ReadAll(resp.Body)
		var orgs []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &orgs); err != nil {
			return fmt.Errorf("failed to parse org list: %w", err)
		}
		found := false
		var resolvedID int
		for _, o := range orgs {
			if fmt.Sprintf("%d", o.ID) == orgIDOrName || o.Name == orgIDOrName {
				found = true
				resolvedID = o.ID
				break
			}
		}
		if !found {
			return fmt.Errorf("organization %s not found", orgIDOrName)
		}
		if err := config.SetActiveOrg(fmt.Sprintf("%d", resolvedID)); err != nil {
			return err
		}
		fmt.Printf("Active organization set to %s (ID: %d)\n", orgIDOrName, resolvedID)
		return nil
	},
}

// org create --name <name>
var orgCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new organization by specifying its name",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name flag is required")
		}
		// Build JSON payload
		payload := fmt.Sprintf(`{"name":"%s"}`, name)
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}
		url := fmt.Sprintf("%s/api/orgs", profile.URL)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(payload)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(profile.User, profile.Pass)
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

// org rm <orgID>
var orgRmCmd = &cobra.Command{
	Use:   "rm [orgID]",
	Short: "Delete an organization by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgIDOrName := args[0]
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}
		// Fetch organization list to verify existence
		listURL := fmt.Sprintf("%s/api/orgs", profile.URL)
		listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			return err
		}
		listReq.SetBasicAuth(profile.User, profile.Pass)
		listResp, err := http.DefaultClient.Do(listReq)
		if err != nil {
			return err
		}
		defer listResp.Body.Close()
		if listResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch org list: %s", listResp.Status)
		}
		listBody, _ := io.ReadAll(listResp.Body)
		var orgs []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(listBody, &orgs); err != nil {
			return fmt.Errorf("failed to parse org list: %w", err)
		}
		found := false
		var orgID string
		for _, o := range orgs {
			if fmt.Sprintf("%d", o.ID) == orgIDOrName || o.Name == orgIDOrName {
				found = true
				orgID = fmt.Sprintf("%d", o.ID)
				break
			}
		}
		if !found {
			return fmt.Errorf("organization %s not found", orgIDOrName)
		}
		// Proceed to delete using the numeric ID
		url := fmt.Sprintf("%s/api/orgs/%s", profile.URL, orgID)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)
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

// org update [orgID|name] --name [new-name]
var orgUpdateCmd = &cobra.Command{
	Use:   "update [orgID|name]",
	Short: "Update an organization name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgIDOrName := args[0]
		newName, _ := cmd.Flags().GetString("name")
		if newName == "" {
			return fmt.Errorf("--name flag is required")
		}

		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}

		// Fetch organization list to verify existence and get ID
		listURL := fmt.Sprintf("%s/api/orgs", profile.URL)
		listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			return err
		}
		listReq.SetBasicAuth(profile.User, profile.Pass)
		listResp, err := http.DefaultClient.Do(listReq)
		if err != nil {
			return err
		}
		defer listResp.Body.Close()
		if listResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch org list: %s", listResp.Status)
		}

		listBody, _ := io.ReadAll(listResp.Body)
		var orgs []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(listBody, &orgs); err != nil {
			return fmt.Errorf("failed to parse org list: %w", err)
		}

		found := false
		var orgID string
		for _, o := range orgs {
			if fmt.Sprintf("%d", o.ID) == orgIDOrName || o.Name == orgIDOrName {
				found = true
				orgID = fmt.Sprintf("%d", o.ID)
				break
			}
		}
		if !found {
			return fmt.Errorf("organization %s not found", orgIDOrName)
		}

		// Proceed to update
		url := fmt.Sprintf("%s/api/orgs/%s", profile.URL, orgID)
		payload := fmt.Sprintf(`{"name":"%s"}`, newName)
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader([]byte(payload)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(profile.User, profile.Pass)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("update failed: %s %s", resp.Status, string(body))
		}

		fmt.Printf("Organization updated: %s -> %s (ID: %s)\n", orgIDOrName, newName, orgID)
		return nil
	},
}

func init() {
	// Register org subcommands
	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgUseCmd)
	orgCmd.AddCommand(orgCreateCmd)
	orgCmd.AddCommand(orgRmCmd)
	orgCmd.AddCommand(orgUpdateCmd)

	// Flag for create command
	orgCreateCmd.Flags().String("name", "", "Name of the organization to create")
	orgCreateCmd.MarkFlagRequired("name")

	// Flag for update command
	orgUpdateCmd.Flags().String("name", "", "New name of the organization")
	orgUpdateCmd.MarkFlagRequired("name")

	// List command flags
	orgListCmd.Flags().Bool("details", false, "Show detailed JSON output")
}
