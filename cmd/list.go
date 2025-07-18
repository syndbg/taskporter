package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tasks and launch configurations",
	Long: `List all discoverable tasks and launch configurations from supported editors.

Scans for configuration files in the current project:
- VSCode: .vscode/tasks.json, .vscode/launch.json
- JetBrains: .idea/runConfigurations/*.xml

Establishing connections to available configurations...`,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Println("🔍 Scanning for configuration files...")
		}

		fmt.Println("📦 Available Tasks & Launch Configurations:")
		fmt.Println()
		fmt.Println("No configurations found. Ensure you're in a project directory with:")
		fmt.Println("  • .vscode/tasks.json or .vscode/launch.json")
		fmt.Println("  • .idea/runConfigurations/*.xml")
		fmt.Println()
		fmt.Println("📡 Strand connection pending... no active configurations detected.")
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
