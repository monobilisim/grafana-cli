package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gcli",
	Short: "CLI tool to interact with Grafana API",
	Long:  `gcli provides commands to manage Grafana API configurations and make requests.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("gcli: use subcommands like config or request")
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		// In a real CLI you might os.Exit(1)
	}
}

func init() {
	// Add subcommands defined in other files.
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(orgCmd)
	rootCmd.AddCommand(dsCmd)
	rootCmd.AddCommand(dashCmd)
	rootCmd.AddCommand(requestCmd)
}
