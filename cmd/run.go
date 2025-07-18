package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <task-name>",
	Short: "Execute a task or launch configuration",
	Long: `Execute a specified task or launch configuration from any supported editor.

The task name should match exactly as it appears in the configuration files.
Supports tasks from:
- VSCode tasks.json
- VSCode launch.json
- JetBrains run configurations

Preparing to establish execution strand...`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskName := args[0]

		if verbose {
			fmt.Printf("🚀 Preparing to execute task: %s\n", taskName)
		}

		fmt.Printf("❌ Task '%s' not found.\n", taskName)
		fmt.Println()
		fmt.Println("Use 'taskporter list' to see available tasks and launch configurations.")
		fmt.Println("📡 Strand connection failed... task not in network.")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
