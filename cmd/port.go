package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"taskporter/internal/config"
	"taskporter/internal/converter"
	"taskporter/internal/parser/jetbrains"
	"taskporter/internal/parser/vscode"
	"taskporter/internal/security"

	"github.com/spf13/cobra"
)

func NewPortCommand(verbose *bool, configPath *string) *cobra.Command {
	var fromFormat string
	var toFormat string
	var dryRun bool
	var outputPath string
	var paranoidMode bool

	portCmd := &cobra.Command{
		Use:   "port",
		Short: "Convert task configurations between IDE formats",
		Long: `Port (migrate) task configurations between different IDE formats.

Supports conversion between:
- VSCode tasks.json ↔ JetBrains run configurations
- VSCode launch.json ↔ JetBrains run configurations

This command helps bridge development workflows when switching between editors
or working in mixed-IDE teams. Like a porter carrying cargo between stations!

By default, taskporter trusts input configurations and processes them as-is.
Use --paranoid-mode for additional security validation of paths and content.

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
			if err := runPortCommand(fromFormat, toFormat, *verbose, *configPath, dryRun, outputPath, paranoidMode); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add flags
	portCmd.Flags().StringVar(&fromFormat, "from", "", "source format (vscode-tasks, vscode-launch, jetbrains)")
	portCmd.Flags().StringVar(&toFormat, "to", "", "target format (vscode-tasks, vscode-launch, jetbrains)")
	portCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing files")
	portCmd.Flags().StringVar(&outputPath, "output", "", "output directory (default: auto-detect)")
	portCmd.Flags().BoolVar(&paranoidMode, "paranoid-mode", false, "Enable security validation of paths and content")

	// Mark required flags
	_ = portCmd.MarkFlagRequired("from")
	_ = portCmd.MarkFlagRequired("to")

	// Add completion for format flags
	_ = portCmd.RegisterFlagCompletionFunc("from", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"vscode-tasks", "vscode-launch", "jetbrains"}, cobra.ShellCompDirectiveNoFileComp
	})

	_ = portCmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"vscode-tasks", "vscode-launch", "jetbrains"}, cobra.ShellCompDirectiveNoFileComp
	})

	return portCmd
}

func runPortCommand(fromFormat, toFormat string, verbose bool, configPath string, dryRun bool, outputPath string, paranoidMode bool) error {
	// Create sanitizer for input validation (only used in paranoid mode)
	sanitizer := security.NewSanitizer(".")

	// Only validate inputs in paranoid mode
	if paranoidMode {
		// Validate config path if provided
		if err := sanitizer.ValidateConfigPath(configPath); err != nil {
			return fmt.Errorf("invalid config path: %w", err)
		}

		// Validate output path if provided
		if err := sanitizer.ValidateOutputPath(outputPath); err != nil {
			return fmt.Errorf("invalid output path: %w", err)
		}
	}

	if verbose {
		fmt.Printf("🚛 Preparing to port configurations...\n")
		fmt.Printf("📤 From: %s\n", fromFormat)
		fmt.Printf("📥 To: %s\n", toFormat)
		if dryRun {
			fmt.Printf("🔍 Mode: Dry run (preview only)\n")
		}
		if paranoidMode {
			fmt.Printf("🛡️ Paranoid mode: Security validation enabled\n")
		} else {
			fmt.Printf("🤝 Trust mode: Processing configurations as-is\n")
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
	case fromFormat == "jetbrains" && toFormat == "vscode-tasks":
		return convertJetBrainsToVSCodeTasks(projectRoot, outputPath, verbose, dryRun)
	case fromFormat == "jetbrains" && toFormat == "vscode-launch":
		return convertJetBrainsToVSCodeLaunch(projectRoot, outputPath, verbose, dryRun)
	case fromFormat == "vscode-launch" && toFormat == "jetbrains":
		return convertVSCodeLaunchToJetBrains(projectRoot, outputPath, verbose, dryRun)
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

// convertJetBrainsToVSCodeTasks handles the conversion from JetBrains to VSCode tasks
func convertJetBrainsToVSCodeTasks(projectRoot, outputPath string, verbose, dryRun bool) error {
	// Initialize project detector
	detector := config.NewProjectDetector(projectRoot)
	projectConfig, err := detector.DetectProject()
	if err != nil {
		return fmt.Errorf("failed to detect project configuration: %w", err)
	}

	if !projectConfig.HasJetBrains {
		return fmt.Errorf("no JetBrains configuration found in project")
	}

	// Parse JetBrains configurations
	jetbrainsPaths := detector.GetJetBrainsRunConfigPaths()
	if len(jetbrainsPaths) == 0 {
		return fmt.Errorf("no JetBrains run configurations found")
	}

	if verbose {
		fmt.Printf("📋 Reading JetBrains configurations from %d files\n", len(jetbrainsPaths))
	}

	parser := jetbrains.NewRunConfigurationParser(projectConfig.ProjectRoot)
	var allTasks []*config.Task

	for _, configPath := range jetbrainsPaths {
		task, err := parser.ParseRunConfiguration(configPath)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️  Warning: failed to parse %s: %v\n", configPath, err)
			}
			continue
		}
		allTasks = append(allTasks, task)
	}

	if len(allTasks) == 0 {
		fmt.Printf("⚠️  No valid JetBrains configurations found to convert\n")
		return nil
	}

	if verbose {
		fmt.Printf("✅ Found %d JetBrains configurations to convert\n", len(allTasks))
	}

	// Create converter and perform conversion
	conv := converter.NewJetBrainsToVSCodeConverter(projectRoot, outputPath, verbose)
	return conv.ConvertTasks(allTasks, dryRun)
}

// convertJetBrainsToVSCodeLaunch handles the conversion from JetBrains to VSCode launch
func convertJetBrainsToVSCodeLaunch(projectRoot, outputPath string, verbose, dryRun bool) error {
	// Initialize project detector
	detector := config.NewProjectDetector(projectRoot)
	projectConfig, err := detector.DetectProject()
	if err != nil {
		return fmt.Errorf("failed to detect project configuration: %w", err)
	}

	if !projectConfig.HasJetBrains {
		return fmt.Errorf("no JetBrains configuration found in project")
	}

	// Parse JetBrains configurations
	jetbrainsPaths := detector.GetJetBrainsRunConfigPaths()
	if len(jetbrainsPaths) == 0 {
		return fmt.Errorf("no JetBrains run configurations found")
	}

	if verbose {
		fmt.Printf("📋 Reading JetBrains configurations from %d files\n", len(jetbrainsPaths))
	}

	parser := jetbrains.NewRunConfigurationParser(projectConfig.ProjectRoot)
	var allTasks []*config.Task

	for _, configPath := range jetbrainsPaths {
		task, err := parser.ParseRunConfiguration(configPath)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️  Warning: failed to parse %s: %v\n", configPath, err)
			}
			continue
		}
		allTasks = append(allTasks, task)
	}

	if len(allTasks) == 0 {
		fmt.Printf("⚠️  No valid JetBrains configurations found to convert\n")
		return nil
	}

	if verbose {
		fmt.Printf("✅ Found %d JetBrains configurations to convert\n", len(allTasks))
	}

	// Create converter and perform conversion
	conv := converter.NewJetBrainsToVSCodeLaunchConverter(projectRoot, outputPath, verbose)
	return conv.ConvertToLaunch(allTasks, dryRun)
}

// convertVSCodeLaunchToJetBrains handles the conversion from VSCode launch to JetBrains
func convertVSCodeLaunchToJetBrains(projectRoot, outputPath string, verbose, dryRun bool) error {
	// Initialize project detector
	detector := config.NewProjectDetector(projectRoot)
	projectConfig, err := detector.DetectProject()
	if err != nil {
		return fmt.Errorf("failed to detect project configuration: %w", err)
	}

	if !projectConfig.HasVSCode {
		return fmt.Errorf("no VSCode configuration found in project")
	}

	// Parse VSCode launch configurations
	launchPath := detector.GetVSCodeLaunchPath()
	if launchPath == "" {
		return fmt.Errorf("no VSCode launch.json found")
	}

	if verbose {
		fmt.Printf("📋 Reading VSCode launch configs from: %s\n", launchPath)
	}

	launchParser := vscode.NewLaunchParser(projectConfig.ProjectRoot)
	launchTasks, err := launchParser.ParseLaunchConfigs(launchPath)
	if err != nil {
		return fmt.Errorf("failed to parse VSCode launch configs: %w", err)
	}

	if len(launchTasks) == 0 {
		fmt.Printf("⚠️  No launch configurations found in %s\n", launchPath)
		return nil
	}

	if verbose {
		fmt.Printf("✅ Found %d VSCode launch configurations to convert\n", len(launchTasks))
	}

	// Create converter and perform conversion
	conv := converter.NewVSCodeLaunchToJetBrainsConverter(projectRoot, outputPath, verbose)
	return conv.ConvertLaunchConfigs(launchTasks, dryRun)
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
