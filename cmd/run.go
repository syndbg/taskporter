package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"taskporter/internal/config"
	"taskporter/internal/parser/vscode"
	"taskporter/internal/runner"

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
		if err := runTaskCommand(taskName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runTaskCommand(taskName string) error {
	if verbose {
		fmt.Printf("🔍 Searching for task: %s\n", taskName)
	}

	// Determine project root
	projectRoot := "."
	if configPath != "" {
		projectRoot = filepath.Dir(configPath)
	}

	// Initialize project detector and find all tasks
	detector := config.NewProjectDetector(projectRoot)
	projectConfig, err := detector.DetectProject()
	if err != nil {
		return fmt.Errorf("failed to detect project configuration: %w", err)
	}

	if verbose {
		fmt.Printf("📁 Project root: %s\n", projectConfig.ProjectRoot)
	}

	var allTasks []*config.Task

	// Parse VSCode tasks
	if projectConfig.HasVSCode {
		if tasksPath := detector.GetVSCodeTasksPath(); tasksPath != "" {
			if verbose {
				fmt.Printf("📋 Scanning VSCode tasks from: %s\n", tasksPath)
			}

			parser := vscode.NewTasksParser(projectConfig.ProjectRoot)
			tasks, err := parser.ParseTasks(tasksPath)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: failed to parse VSCode tasks: %v\n", err)
				}
			} else {
				allTasks = append(allTasks, tasks...)
			}
		}

		// TODO: Parse VSCode launch configurations
		// if launchPath := detector.GetVSCodeLaunchPath(); launchPath != "" {
		//     // Parse launch configs when implemented
		// }
	}

	// TODO: Parse JetBrains configurations
	// if projectConfig.HasJetBrains {
	//     // Parse JetBrains configs when implemented
	// }

	if len(allTasks) == 0 {
		fmt.Println("❌ No tasks found in this project.")
		fmt.Println()
		fmt.Println("Use 'taskporter list' to see available tasks and launch configurations.")
		fmt.Println("📡 Strand connection failed... no active configurations detected.")
		return nil
	}

	// Find the requested task
	finder := runner.NewTaskFinder()
	task, err := finder.FindTask(taskName, allTasks)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println()
		fmt.Println("Available tasks:")
		for _, t := range allTasks {
			fmt.Printf("  • %s", t.Name)
			if t.Group != "" {
				fmt.Printf(" [%s]", t.Group)
			}
			fmt.Println()
		}
		fmt.Println()
		fmt.Println("📡 Strand connection failed... task not in network.")
		return nil
	}

	if verbose {
		fmt.Printf("✅ Found task: %s (%s)\n", task.Name, task.Type)
		fmt.Println()
	}

	// Execute the task
	taskRunner := runner.NewTaskRunner(verbose)
	if err := taskRunner.RunTask(task); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
