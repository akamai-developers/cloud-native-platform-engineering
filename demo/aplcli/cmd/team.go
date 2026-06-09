package cmd

import (
	//"fmt"

	//"github.com/ruckus-voxi/aplcli/internal/client"
	"github.com/spf13/cobra"
)

// teamCmd represents the team command
var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Add or Remove teams to APL",
	PreRun: func(cmd *cobra.Command, args []string) {
		// PreRun code goes here...
		// Example: keycloak credentials for obtaining api tokens
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Command logic and imports from client.go...
	},
}

func init() {
	rootCmd.AddCommand(teamCmd)

	// List teams
	teamCmd.Flags().BoolP("list", "l", false, "List teams")

	// Add team
	teamCmd.Flags().StringP("add", "a", "", "Add team")

	// Remove team
	teamCmd.Flags().StringP("delete", "d", "", "Remove team")
}
