package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"taskporter/internal/config"
	"taskporter/internal/parser/jetbrains"
	"taskporter/internal/parser/vscode"

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
		if err := runListCommand(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runListCommand() error {
	if verbose {
		fmt.Println("🔍 Scanning for configuration files...")
	}

	// Determine project root
	projectRoot := "."
	if configPath != "" {
		projectRoot = filepath.Dir(configPath)
	}

	// Initialize project detector
	detector := config.NewProjectDetector(projectRoot)
	projectConfig, err := detector.DetectProject()
	if err != nil {
		return fmt.Errorf("failed to detect project configuration: %w", err)
	}

	if verbose {
		fmt.Printf("📁 Project root: %s\n", projectConfig.ProjectRoot)
		fmt.Printf("🔧 VSCode detected: %v\n", projectConfig.HasVSCode)
		fmt.Printf("🧠 JetBrains detected: %v\n", projectConfig.HasJetBrains)
	}

	var allTasks []*config.Task

	// Parse VSCode tasks
	if projectConfig.HasVSCode {
		if tasksPath := detector.GetVSCodeTasksPath(); tasksPath != "" {
			if verbose {
				fmt.Printf("📋 Parsing VSCode tasks from: %s\n", tasksPath)
			}

			parser := vscode.NewTasksParser(projectConfig.ProjectRoot)
			tasks, err := parser.ParseTasks(tasksPath)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: failed to parse VSCode tasks: %v\n", err)
				}
			} else {
				allTasks = append(allTasks, tasks...)
				if verbose {
					fmt.Printf("✅ Found %d VSCode tasks\n", len(tasks))
				}
			}
		}

		// Parse VSCode launch configurations
		if launchPath := detector.GetVSCodeLaunchPath(); launchPath != "" {
			if verbose {
				fmt.Printf("🚀 Parsing VSCode launch configs from: %s\n", launchPath)
			}

			launchParser := vscode.NewLaunchParser(projectConfig.ProjectRoot)
			launchTasks, err := launchParser.ParseLaunchConfigs(launchPath)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: failed to parse VSCode launch configs: %v\n", err)
				}
			} else {
				allTasks = append(allTasks, launchTasks...)
				if verbose {
					fmt.Printf("✅ Found %d VSCode launch configurations\n", len(launchTasks))
				}
			}
		}
	}

	// Parse JetBrains configurations
	if projectConfig.HasJetBrains {
		jetbrainsPaths := detector.GetJetBrainsRunConfigPaths()
		if verbose && len(jetbrainsPaths) > 0 {
			fmt.Printf("🧠 Parsing JetBrains configurations from: %d files\n", len(jetbrainsPaths))
		}

		parser := jetbrains.NewRunConfigurationParser(projectConfig.ProjectRoot)
		for _, configPath := range jetbrainsPaths {
			if verbose {
				fmt.Printf("   📄 %s\n", configPath)
			}

			task, err := parser.ParseRunConfiguration(configPath)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: failed to parse JetBrains config %s: %v\n", configPath, err)
				}
			} else {
				allTasks = append(allTasks, task)
			}
		}

		if verbose && len(jetbrainsPaths) > 0 {
			jetbrainsTaskCount := 0
			for _, task := range allTasks {
				if task.Type == config.TypeJetBrains {
					jetbrainsTaskCount++
				}
			}
			fmt.Printf("✅ Found %d JetBrains configurations\n", jetbrainsTaskCount)
		}
	}

	// Display results
	return displayTasks(allTasks)
}

func displayTasks(tasks []*config.Task) error {
	if outputFormat == "json" {
		return displayTasksJSON(tasks)
	}
	return displayTasksText(tasks)
}

func displayTasksText(tasks []*config.Task) error {
	fmt.Println("📦 Available Tasks & Launch Configurations:")
	fmt.Println()

	if len(tasks) == 0 {
		fmt.Println("No configurations found. Ensure you're in a project directory with:")
		fmt.Println("  • .vscode/tasks.json or .vscode/launch.json")
		fmt.Println("  • .idea/runConfigurations/*.xml")
		fmt.Println()
		fmt.Println("📡 Strand connection pending... no active configurations detected.")
		return nil
	}

	// Group tasks by type
	tasksByType := make(map[config.TaskType][]*config.Task)
	for _, task := range tasks {
		tasksByType[task.Type] = append(tasksByType[task.Type], task)
	}

	// Display VSCode tasks
	if vscTasks := tasksByType[config.TypeVSCodeTask]; len(vscTasks) > 0 {
		fmt.Printf("🔧 VSCode Tasks (%d):\n", len(vscTasks))
		for _, task := range vscTasks {
			fmt.Printf("  • %s", task.Name)
			if task.Group != "" {
				fmt.Printf(" [%s]", task.Group)
			}
			fmt.Printf(" - %s", task.Command)
			if len(task.Args) > 0 {
				fmt.Printf(" %v", task.Args)
			}
			fmt.Println()
			if task.Description != "" {
				fmt.Printf("    %s\n", task.Description)
			}
		}
		fmt.Println()
	}

	// Display VSCode launch configs
	if vscLaunches := tasksByType[config.TypeVSCodeLaunch]; len(vscLaunches) > 0 {
		fmt.Printf("🚀 VSCode Launch Configurations (%d):\n", len(vscLaunches))
		for _, task := range vscLaunches {
			fmt.Printf("  • %s", task.Name)
			if task.Group != "" {
				fmt.Printf(" [%s]", task.Group)
			}
			fmt.Printf(" - %s", task.Command)
			if len(task.Args) > 0 {
				fmt.Printf(" %v", task.Args)
			}
			fmt.Println()
			if task.Description != "" {
				fmt.Printf("    %s\n", task.Description)
			}
		}
		fmt.Println()
	}

	// Display JetBrains configs (when implemented)
	if jbTasks := tasksByType[config.TypeJetBrains]; len(jbTasks) > 0 {
		fmt.Printf("🧠 JetBrains Run Configurations (%d):\n", len(jbTasks))
		for _, task := range jbTasks {
			fmt.Printf("  • %s - %s %v\n", task.Name, task.Command, task.Args)
		}
		fmt.Println()
	}

	fmt.Println("📡 Strand established! Use 'taskporter run <task-name>' to execute.")
	return nil
}

func displayTasksJSON(tasks []*config.Task) error {
	output := map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// setupListCommand configures the list command
func setupListCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(listCmd)
}
