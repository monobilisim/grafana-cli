package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"gcli/internal/config"

	"github.com/spf13/cobra"
)

var requestCmd = &cobra.Command{
	Use:   "request [METHOD] [PATH]",
	Short: "Make a request to the selected Grafana instance",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		method := strings.ToUpper(args[0])
		path := args[1]
		// Ensure path starts with a slash
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		profile, err := config.GetActive()
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("no active profile set; use 'gcli config use <name>' first")
		}
		url := strings.TrimRight(profile.URL, "/") + path
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(profile.User, profile.Pass)

		// Set Org ID header if active org is selected
		activeOrg, _ := config.GetActiveOrg()
		if activeOrg != "" {
			req.Header.Set("X-Grafana-Org-Id", activeOrg)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		fmt.Printf("Status: %s\n", resp.Status)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	},
}

func init() {
	// requestCmd is added to root in root.go's init()
	// No extra flags needed for now.
}
