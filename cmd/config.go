package cmd

import (
	"encoding/json"
	"fmt"
	"gcli/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Grafana API configurations",
}

// addCmd adds a new configuration profile.
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new Grafana configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		user, _ := cmd.Flags().GetString("user")
		pass, _ := cmd.Flags().GetString("pass")
		if name == "" || url == "" || user == "" || pass == "" {
			return fmt.Errorf("all flags --name, --url, --user, --pass are required")
		}
		profile := config.Profile{
			Name: name,
			URL:  url,
			User: user,
			Pass: pass,
		}
		return config.SaveProfile(profile)
	},
}

// listCmd lists all saved profiles.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved Grafana configurations",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := config.LoadAll()
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(profiles, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

// useCmd selects a profile for subsequent requests.
var useCmd = &cobra.Command{
	Use:   "use [profile-name]",
	Short: "Select a configuration profile to use",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return config.SetActive(name)
	},
}

func init() {
	configCmd.AddCommand(addCmd)
	configCmd.AddCommand(listCmd)
	configCmd.AddCommand(useCmd)

	// Flags for `add`.
	addCmd.Flags().String("name", "", "Profile name")
	addCmd.Flags().String("url", "", "Grafana base URL")
	addCmd.Flags().String("user", "", "Basic auth username")
	addCmd.Flags().String("pass", "", "Basic auth password")
	addCmd.MarkFlagRequired("name")
	addCmd.MarkFlagRequired("url")
	addCmd.MarkFlagRequired("user")
	addCmd.MarkFlagRequired("pass")
}
