package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"taskporter/internal/config"
	"taskporter/internal/parser/jetbrains"
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

		// Parse VSCode launch configurations
		if launchPath := detector.GetVSCodeLaunchPath(); launchPath != "" {
			if verbose {
				fmt.Printf("🚀 Scanning VSCode launch configs from: %s\n", launchPath)
			}

			launchParser := vscode.NewLaunchParser(projectConfig.ProjectRoot)
			launchTasks, err := launchParser.ParseLaunchConfigs(launchPath)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: failed to parse VSCode launch configs: %v\n", err)
				}
			} else {
				allTasks = append(allTasks, launchTasks...)
			}
		}
	}

	// Parse JetBrains configurations
	if projectConfig.HasJetBrains {
		jetbrainsPaths := detector.GetJetBrainsRunConfigPaths()
		if verbose && len(jetbrainsPaths) > 0 {
			fmt.Printf("🧠 Scanning JetBrains configurations from: %d files\n", len(jetbrainsPaths))
		}

		parser := jetbrains.NewRunConfigurationParser(projectConfig.ProjectRoot)
		for _, configPath := range jetbrainsPaths {
			task, err := parser.ParseRunConfiguration(configPath)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: failed to parse JetBrains config %s: %v\n", configPath, err)
				}
			} else {
				allTasks = append(allTasks, task)
			}
		}
	}

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

	// Check for preLaunchTask if this is a launch configuration
	if task.Type == config.TypeVSCodeLaunch {
		if err := runPreLaunchTask(task, allTasks, projectConfig, detector, finder, verbose); err != nil {
			return fmt.Errorf("preLaunchTask failed: %w", err)
		}
	}

	// Execute the main task
	taskRunner := runner.NewTaskRunner(verbose)
	if err := taskRunner.RunTask(task); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

// runPreLaunchTask executes a preLaunchTask if specified in a launch configuration
func runPreLaunchTask(launchTask *config.Task, allTasks []*config.Task, projectConfig *config.ProjectConfig, detector *config.ProjectDetector, finder *runner.TaskFinder, verbose bool) error {
	// Only check VSCode launch configurations for preLaunchTask
	if launchTask.Type != config.TypeVSCodeLaunch {
		return nil
	}

	// Get the launch file path
	launchPath := detector.GetVSCodeLaunchPath()
	if launchPath == "" {
		return nil // No launch.json file found
	}

	// Create launch parser to get preLaunchTask name
	launchParser := vscode.NewLaunchParser(projectConfig.ProjectRoot)
	preLaunchTaskName, err := launchParser.GetPreLaunchTask(launchPath, launchTask.Name)
	if err != nil {
		if verbose {
			fmt.Printf("⚠️  Warning: failed to get preLaunchTask for %s: %v\n", launchTask.Name, err)
		}
		return nil // Continue without preLaunchTask
	}

	// If no preLaunchTask specified, continue
	if preLaunchTaskName == "" {
		return nil
	}

	if verbose {
		fmt.Printf("🔗 Launch configuration has preLaunchTask: %s\n", preLaunchTaskName)
	}

	// Find the preLaunchTask
	preLaunchTask, err := finder.FindTask(preLaunchTaskName, allTasks)
	if err != nil {
		return fmt.Errorf("preLaunchTask '%s' not found: %w", preLaunchTaskName, err)
	}

	if verbose {
		fmt.Printf("🔧 Executing preLaunchTask: %s (%s)\n", preLaunchTask.Name, preLaunchTask.Type)
		fmt.Println()
	}

	// Execute the preLaunchTask
	taskRunner := runner.NewTaskRunner(verbose)
	if err := taskRunner.RunTask(preLaunchTask); err != nil {
		return fmt.Errorf("preLaunchTask '%s' execution failed: %w", preLaunchTaskName, err)
	}

	if verbose {
		fmt.Printf("✅ PreLaunchTask '%s' completed successfully\n", preLaunchTaskName)
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
