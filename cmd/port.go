package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"taskporter/internal/config"
	"taskporter/internal/converter"
	"taskporter/internal/parser/vscode"

	"github.com/spf13/cobra"
)

func NewPortCommand(verbose *bool, configPath *string) *cobra.Command {
	var fromFormat string
	var toFormat string
	var dryRun bool
	var outputPath string

	portCmd := &cobra.Command{
		Use:   "port",
		Short: "Convert task configurations between IDE formats",
		Long: `Port (migrate) task configurations between different IDE formats.

Supports conversion between:
- VSCode tasks.json ↔ JetBrains run configurations
- VSCode launch.json ↔ JetBrains run configurations

This command helps bridge development workflows when switching between editors
or working in mixed-IDE teams. Like a porter carrying cargo between stations!

Examples:
  # Convert VSCode tasks to JetBrains format
  taskporter port --from vscode-tasks --to jetbrains

  # Convert JetBrains configs to VSCode launch format
  taskporter port --from jetbrains --to vscode-launch

  # Dry run to preview changes
  taskporter port --from vscode-tasks --to jetbrains --dry-run

  # Specify output path
  taskporter port --from vscode-tasks --to jetbrains --output .idea/runConfigurations/

Establishing cross-platform development strand...`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runPortCommand(fromFormat, toFormat, *verbose, *configPath, dryRun, outputPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add flags
	portCmd.Flags().StringVar(&fromFormat, "from", "", "source format (vscode-tasks, vscode-launch, jetbrains)")
	portCmd.Flags().StringVar(&toFormat, "to", "", "target format (vscode-tasks, vscode-launch, jetbrains)")
	portCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing files")
	portCmd.Flags().StringVar(&outputPath, "output", "", "output path (default: auto-detect based on target format)")

	// Mark required flags
	_ = portCmd.MarkFlagRequired("from")
	_ = portCmd.MarkFlagRequired("to")

	// Add completion for format flags
	formatOptions := []string{"vscode-tasks", "vscode-launch", "jetbrains"}
	_ = portCmd.RegisterFlagCompletionFunc("from", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return formatOptions, cobra.ShellCompDirectiveNoFileComp
	})
	_ = portCmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return formatOptions, cobra.ShellCompDirectiveNoFileComp
	})

	return portCmd
}

func runPortCommand(fromFormat, toFormat string, verbose bool, configPath string, dryRun bool, outputPath string) error {
	if verbose {
		fmt.Printf("🚛 Preparing to port configurations...\n")
		fmt.Printf("📤 From: %s\n", fromFormat)
		fmt.Printf("📥 To: %s\n", toFormat)
		if dryRun {
			fmt.Printf("🔍 Mode: Dry run (preview only)\n")
		}
		fmt.Println()
	}

	// Validate format combinations
	if err := validateFormatCombination(fromFormat, toFormat); err != nil {
		return err
	}

	// Determine project root
	projectRoot := "."
	if configPath != "" {
		projectRoot = filepath.Dir(configPath)
	}

	// Execute the conversion based on format combination
	switch {
	case fromFormat == "vscode-tasks" && toFormat == "jetbrains":
		return convertVSCodeTasksToJetBrains(projectRoot, outputPath, verbose, dryRun)
	default:
		fmt.Printf("🚧 Conversion from %s to %s is not yet implemented!\n", fromFormat, toFormat)
		fmt.Printf("📋 Planned conversion: %s → %s\n", fromFormat, toFormat)

		if dryRun {
			fmt.Printf("✅ Dry run completed - no files were modified\n")
		} else {
			fmt.Printf("📡 Strand connection established... migration ready for implementation!\n")
		}
	}

	return nil
}

// convertVSCodeTasksToJetBrains handles the conversion from VSCode tasks to JetBrains
func convertVSCodeTasksToJetBrains(projectRoot, outputPath string, verbose, dryRun bool) error {
	// Initialize project detector
	detector := config.NewProjectDetector(projectRoot)
	projectConfig, err := detector.DetectProject()
	if err != nil {
		return fmt.Errorf("failed to detect project configuration: %w", err)
	}

	if !projectConfig.HasVSCode {
		return fmt.Errorf("no VSCode configuration found in project")
	}

	// Parse VSCode tasks
	tasksPath := detector.GetVSCodeTasksPath()
	if tasksPath == "" {
		return fmt.Errorf("no VSCode tasks.json found")
	}

	if verbose {
		fmt.Printf("📋 Reading VSCode tasks from: %s\n", tasksPath)
	}

	parser := vscode.NewTasksParser(projectConfig.ProjectRoot)
	tasks, err := parser.ParseTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to parse VSCode tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Printf("⚠️  No tasks found in %s\n", tasksPath)
		return nil
	}

	if verbose {
		fmt.Printf("✅ Found %d VSCode tasks to convert\n", len(tasks))
	}

	// Create converter and perform conversion
	conv := converter.NewVSCodeToJetBrainsConverter(projectRoot, outputPath, verbose)
	return conv.ConvertTasks(tasks, dryRun)
}

func validateFormatCombination(from, to string) error {
	validFormats := map[string]bool{
		"vscode-tasks":  true,
		"vscode-launch": true,
		"jetbrains":     true,
	}

	if !validFormats[from] {
		return fmt.Errorf("invalid source format '%s'. Valid options: vscode-tasks, vscode-launch, jetbrains", from)
	}

	if !validFormats[to] {
		return fmt.Errorf("invalid target format '%s'. Valid options: vscode-tasks, vscode-launch, jetbrains", to)
	}

	if from == to {
		return fmt.Errorf("source and target formats cannot be the same")
	}

	// Check for supported conversion paths
	supportedConversions := map[string][]string{
		"vscode-tasks":  {"jetbrains"},
		"vscode-launch": {"jetbrains"},
		"jetbrains":     {"vscode-tasks", "vscode-launch"},
	}

	if supported, exists := supportedConversions[from]; exists {
		for _, validTarget := range supported {
			if validTarget == to {
				return nil
			}
		}
	}

	return fmt.Errorf("conversion from '%s' to '%s' is not yet supported", from, to)
}
