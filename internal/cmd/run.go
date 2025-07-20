package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"taskporter/internal/config"
	"taskporter/internal/parser/jetbrains"
	"taskporter/internal/parser/vscode"
	"taskporter/internal/runner"
	"taskporter/internal/security"

	"github.com/spf13/cobra"
)

// getAllTasksQuiet gets all tasks without verbose output for completion
func getAllTasksQuiet() ([]*config.Task, error) {
	// Determine project root
	projectRoot := "."

	// Initialize project detector
	detector := config.NewProjectDetector(projectRoot)

	projectConfig, err := detector.DetectProject()
	if err != nil {
		return nil, err
	}

	var allTasks []*config.Task

	// Parse VSCode tasks
	if projectConfig.HasVSCode {
		if tasksPath := detector.GetVSCodeTasksPath(); tasksPath != "" {
			parser := vscode.NewTasksParser(projectConfig.ProjectRoot)

			tasks, err := parser.ParseTasks(tasksPath)
			if err == nil {
				allTasks = append(allTasks, tasks...)
			}
		}

		// Parse VSCode launch configurations
		if launchPath := detector.GetVSCodeLaunchPath(); launchPath != "" {
			launchParser := vscode.NewLaunchParser(projectConfig.ProjectRoot)

			launchTasks, err := launchParser.ParseLaunchConfigs(launchPath)
			if err == nil {
				allTasks = append(allTasks, launchTasks...)
			}
		}
	}

	// Parse JetBrains configurations
	if projectConfig.HasJetBrains {
		jetbrainsPaths := detector.GetJetBrainsRunConfigPaths()
		if len(jetbrainsPaths) > 0 {
			parser := jetbrains.NewRunConfigurationParser(projectConfig.ProjectRoot)
			for _, path := range jetbrainsPaths {
				task, err := parser.ParseRunConfiguration(path)
				if err == nil {
					allTasks = append(allTasks, task)
				}
			}
		}
	}

	return allTasks, nil
}

// validTaskNames provides dynamic completion for task names
func validTaskNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		// Only complete the first argument (task name)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get the project configurations to find available tasks
	tasks, err := getAllTasksQuiet()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var taskNames []string
	for _, task := range tasks {
		taskNames = append(taskNames, task.Name)
	}

	return taskNames, cobra.ShellCompDirectiveNoFileComp
}

func NewRunCommand(verbose *bool, configPath *string) *cobra.Command {
	var (
		noInteractive bool
		paranoidMode  bool
	)

	runCmd := &cobra.Command{
		Use:   "run [task-name]",
		Short: "Execute a task or launch configuration",
		Long: `Execute a specified task or launch configuration from any supported editor.

If no task name is provided, an interactive selector will be shown.
Use --no-interactive flag to disable interactive mode (useful for CI/CD).

The task name should match exactly as it appears in the configuration files.
Supports tasks from:
- VSCode tasks.json
- VSCode launch.json
- JetBrains run configurations

By default, taskporter trusts user configurations and executes them as-is (like IDEs).
Use --paranoid-mode for additional security validation of commands and arguments.

Preparing to establish execution strand...`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: validTaskNames,
		Run: func(cmd *cobra.Command, args []string) {
			var taskName string
			if len(args) > 0 {
				taskName = args[0]
			}
			if err := runTaskCommand(taskName, *verbose, *configPath, noInteractive, paranoidMode); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	runCmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Disable interactive mode (useful for CI/CD)")
	runCmd.Flags().BoolVar(&paranoidMode, "paranoid-mode", false, "Enable security validation (default: trust user configurations)")

	return runCmd
}

func runTaskCommand(taskName string, verbose bool, configPath string, noInteractive bool, paranoidMode bool) error {
	// Create sanitizer for input validation (only used in paranoid mode)
	sanitizer := security.NewSanitizer(".")

	// Only validate inputs in paranoid mode
	if paranoidMode {
		// Validate task name if provided
		if taskName != "" {
			if err := sanitizer.ValidateTaskName(taskName); err != nil {
				return fmt.Errorf("invalid task name: %w", err)
			}
		}

		// Validate config path if provided
		if err := sanitizer.ValidateConfigPath(configPath); err != nil {
			return fmt.Errorf("invalid config path: %w", err)
		}
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

	// Convert to value type for interactive selector
	tasks := make([]config.Task, len(allTasks))
	for i, taskPtr := range allTasks {
		tasks[i] = *taskPtr
	}

	// If no task name provided, run interactive mode (unless disabled)
	if taskName == "" {
		if noInteractive {
			fmt.Println("❌ No task name provided and interactive mode is disabled.")
			fmt.Println()
			fmt.Println("Available tasks:")

			for _, taskPtr := range allTasks {
				fmt.Printf("  • %s", taskPtr.Name)

				if taskPtr.Group != "" {
					fmt.Printf(" [%s]", taskPtr.Group)
				}

				fmt.Printf(" - %s", getTaskSourceDisplay(taskPtr))
				fmt.Println()
			}

			fmt.Println()
			fmt.Println("Usage: taskporter run <task-name>")
			fmt.Println("   or: taskporter run (for interactive mode)")
			fmt.Println("📡 Strand connection failed... no task specified.")

			return nil
		}

		if verbose {
			fmt.Printf("🎮 Starting interactive task selector...\n")
		}

		selectedTask, err := runner.RunInteractiveTaskSelector(tasks)
		if err != nil {
			return fmt.Errorf("interactive selection failed: %w", err)
		}

		if selectedTask == nil {
			// User cancelled
			return nil
		}
		// Use the selected task
		task := selectedTask

		return executeSelectedTask(task, allTasks, projectConfig, detector, verbose, paranoidMode)
	}

	if verbose {
		fmt.Printf("🔍 Searching for task: %s\n", taskName)
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

	return executeSelectedTask(task, allTasks, projectConfig, detector, verbose, paranoidMode)
}

// executeSelectedTask executes a task with proper preLaunchTask handling
func executeSelectedTask(task *config.Task, allTasks []*config.Task, projectConfig *config.ProjectConfig, detector *config.ProjectDetector, verbose bool, paranoidMode bool) error {
	// Check for preLaunchTask if this is a launch configuration
	if task.Type == config.TypeVSCodeLaunch {
		finder := runner.NewTaskFinder()
		if err := runPreLaunchTask(task, allTasks, projectConfig, detector, finder, verbose, paranoidMode); err != nil {
			return fmt.Errorf("preLaunchTask failed: %w", err)
		}
	}

	// Execute the main task with paranoid mode option
	taskRunner := runner.NewTaskRunnerWithOptions(verbose, projectConfig.ProjectRoot, paranoidMode)
	if err := taskRunner.RunTask(task); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

// runPreLaunchTask executes a preLaunchTask if specified in a launch configuration
func runPreLaunchTask(launchTask *config.Task, allTasks []*config.Task, projectConfig *config.ProjectConfig, detector *config.ProjectDetector, finder *runner.TaskFinder, verbose bool, paranoidMode bool) error {
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

	// Execute the preLaunchTask with paranoid mode option
	taskRunner := runner.NewTaskRunnerWithOptions(verbose, projectConfig.ProjectRoot, paranoidMode)
	if err := taskRunner.RunTask(preLaunchTask); err != nil {
		return fmt.Errorf("preLaunchTask '%s' execution failed: %w", preLaunchTaskName, err)
	}

	if verbose {
		fmt.Printf("✅ PreLaunchTask '%s' completed successfully\n", preLaunchTaskName)
		fmt.Println()
	}

	return nil
}

// getTaskSourceDisplay returns a display-friendly source name for a task
func getTaskSourceDisplay(task *config.Task) string {
	switch task.Type {
	case config.TypeVSCodeTask:
		return "VSCode Task"
	case config.TypeVSCodeLaunch:
		return "VSCode Launch"
	case config.TypeJetBrains:
		return "JetBrains"
	default:
		return string(task.Type)
	}
}
