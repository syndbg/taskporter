package converter

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/syndbg/taskporter/internal/config"
)

// VSCodeLaunchToJetBrainsConverter converts VSCode launch configurations to JetBrains run configurations
type VSCodeLaunchToJetBrainsConverter struct {
	projectRoot string
	outputPath  string
	verbose     bool
}

// NewVSCodeLaunchToJetBrainsConverter creates a new launch to JetBrains converter
func NewVSCodeLaunchToJetBrainsConverter(projectRoot, outputPath string, verbose bool) *VSCodeLaunchToJetBrainsConverter {
	return &VSCodeLaunchToJetBrainsConverter{
		projectRoot: projectRoot,
		outputPath:  outputPath,
		verbose:     verbose,
	}
}

// ConvertLaunchConfigs converts VSCode launch configurations to JetBrains run configurations
func (c *VSCodeLaunchToJetBrainsConverter) ConvertLaunchConfigs(tasks []*config.Task, dryRun bool) error {
	if c.verbose {
		fmt.Printf("🔄 Converting %d VSCode launch configurations to JetBrains format...\n", len(tasks))
	}

	// Filter only VSCode launch tasks
	launchTasks := make([]*config.Task, 0)
	for _, task := range tasks {
		if task.Type == config.TypeVSCodeLaunch {
			launchTasks = append(launchTasks, task)
		}
	}

	if len(launchTasks) == 0 {
		fmt.Printf("⚠️  No VSCode launch configurations found to convert\n")
		return nil
	}

	if c.verbose {
		fmt.Printf("📋 Converting %d VSCode launch configurations\n", len(launchTasks))
	}

	// Determine output directory
	outputDir := c.outputPath
	if outputDir == "" {
		outputDir = filepath.Join(c.projectRoot, ".idea", "runConfigurations")
	}

	if c.verbose {
		fmt.Printf("📁 Output directory: %s\n", outputDir)
	}

	// Create output directory if not in dry-run mode
	if !dryRun {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	convertedCount := 0

	for _, task := range launchTasks {
		config, err := c.convertSingleLaunchConfig(task)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert launch config '%s': %v\n", task.Name, err)
			continue
		}

		// Generate filename (sanitize task name)
		filename := c.sanitizeFilename(task.Name) + ".xml"
		outputPath := filepath.Join(outputDir, filename)

		if dryRun {
			fmt.Printf("   [DRY RUN] Would create: %s\n", outputPath)

			// Show XML preview
			xmlData, _ := xml.MarshalIndent(config, "", "  ")
			fmt.Printf("📝 Preview of %s:\n%s\n\n", filename, string(xmlData))
		} else {
			if err := c.writeJetBrainsRunConfig(config, outputPath); err != nil {
				fmt.Printf("⚠️  Warning: failed to write config '%s': %v\n", task.Name, err)
				continue
			}

			if c.verbose {
				fmt.Printf("✅ Created: %s\n", outputPath)
			}
		}

		convertedCount++
	}

	fmt.Printf("✅ Successfully converted %d/%d VSCode launch configurations\n", convertedCount, len(launchTasks))

	return nil
}

// convertSingleLaunchConfig converts a single VSCode launch config to JetBrains format
func (c *VSCodeLaunchToJetBrainsConverter) convertSingleLaunchConfig(task *config.Task) (*JetBrainsRunConfiguration, error) {
	// Determine JetBrains configuration type based on VSCode launch type
	configType, err := c.determineJetBrainsConfigType(task)
	if err != nil {
		return nil, err
	}

	config := &JetBrainsRunConfiguration{
		Name:    task.Name,
		Type:    configType,
		Options: make([]JetBrainsOption, 0),
		EnvVars: nil,
	}

	// Add configuration options based on type
	if err := c.addConfigurationOptions(task, config); err != nil {
		return nil, err
	}

	// Set working directory (convert VSCode variables)
	workingDir := task.Cwd
	if workingDir == "" {
		workingDir = "$PROJECT_DIR$"
	} else {
		workingDir = c.convertVSCodeVariables(workingDir)
	}

	config.Options = append(config.Options, JetBrainsOption{
		Name:  "WORKING_DIRECTORY",
		Value: workingDir,
	})

	// Convert environment variables
	if len(task.Env) > 0 {
		// Sort keys for deterministic ordering
		keys := make([]string, 0, len(task.Env))
		for key := range task.Env {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		envVars := make([]JetBrainsEnvVar, 0, len(task.Env))
		for _, key := range keys {
			envVars = append(envVars, JetBrainsEnvVar{
				Name:  key,
				Value: c.convertVSCodeVariables(task.Env[key]),
			})
		}

		config.EnvVars = &JetBrainsEnvVars{EnvVars: envVars}
	}

	return config, nil
}

// determineJetBrainsConfigType determines the appropriate JetBrains config type
func (c *VSCodeLaunchToJetBrainsConverter) determineJetBrainsConfigType(task *config.Task) (string, error) {
	// Extract the launch type from task description (contains "go launch", "node launch", etc.)
	description := strings.ToLower(task.Description)
	command := strings.ToLower(task.Command)

	// Check for Go applications (priority check)
	if strings.Contains(description, "go launch") || strings.Contains(description, "go attach") || command == "go" {
		return "GoApplicationRunConfiguration", nil
	}

	// Check for Node.js applications
	if strings.Contains(description, "node launch") || strings.Contains(description, "node attach") ||
		command == "node" || strings.Contains(task.Command, ".js") || strings.Contains(task.Command, ".ts") {
		return "NodeJSConfigurationType", nil
	}

	// Check for Python applications
	if strings.Contains(description, "python launch") || strings.Contains(description, "python attach") ||
		command == "python" || strings.Contains(task.Command, ".py") {
		return "PythonConfigurationType", nil
	}

	// Check for Java applications
	if strings.Contains(command, "java") || strings.Contains(task.Command, "mainClass") {
		return "Application", nil
	}

	// Default to Application for generic executables
	return "Application", nil
}

// addConfigurationOptions adds type-specific options to the JetBrains configuration
func (c *VSCodeLaunchToJetBrainsConverter) addConfigurationOptions(task *config.Task, config *JetBrainsRunConfiguration) error {
	switch config.Type {
	case "GoApplicationRunConfiguration":
		return c.addGoApplicationOptions(task, config)
	case "Application":
		return c.addJavaApplicationOptions(task, config)
	case "NodeJSConfigurationType":
		return c.addNodeJSOptions(task, config)
	case "PythonConfigurationType":
		return c.addPythonOptions(task, config)
	default:
		return c.addGenericOptions(task, config)
	}
}

// addJavaApplicationOptions adds Java-specific options
func (c *VSCodeLaunchToJetBrainsConverter) addJavaApplicationOptions(task *config.Task, config *JetBrainsRunConfiguration) error {
	// Extract main class - look for it in command or args
	mainClass := c.extractMainClassFromLaunch(task)
	if mainClass == "" {
		return fmt.Errorf("could not determine main class for Java application '%s'", task.Name)
	}

	config.Options = append(config.Options, JetBrainsOption{
		Name:  "MAIN_CLASS_NAME",
		Value: mainClass,
	})

	// Add program parameters (excluding main class)
	args := c.filterArgsExcluding(task.Args, mainClass)
	if len(args) > 0 {
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "PROGRAM_PARAMETERS",
			Value: strings.Join(args, " "),
		})
	}

	return nil
}

// addGoApplicationOptions adds Go-specific options
func (c *VSCodeLaunchToJetBrainsConverter) addGoApplicationOptions(task *config.Task, config *JetBrainsRunConfiguration) error {
	// For Go applications, extract the package path and arguments
	packagePath := c.extractGoPackageFromLaunch(task)
	if packagePath == "" {
		packagePath = "."
	}

	config.Options = append(config.Options, JetBrainsOption{
		Name:  "PACKAGE",
		Value: packagePath,
	})

	// Add Go run kind (package vs file)
	config.Options = append(config.Options, JetBrainsOption{
		Name:  "RUN_KIND",
		Value: "PACKAGE",
	})

	// Add program arguments (exclude "run" and package path)
	args := c.filterGoArgsFromLaunch(task)
	if len(args) > 0 {
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "PROGRAM_PARAMETERS",
			Value: strings.Join(args, " "),
		})
	}

	return nil
}

// addNodeJSOptions adds Node.js-specific options
func (c *VSCodeLaunchToJetBrainsConverter) addNodeJSOptions(task *config.Task, config *JetBrainsRunConfiguration) error {
	// Extract JavaScript file path
	program := c.extractProgramFromLaunch(task)
	if program == "" {
		return fmt.Errorf("could not determine program for Node.js application '%s'", task.Name)
	}

	config.Options = append(config.Options, JetBrainsOption{
		Name:  "PATH_TO_JS_FILE",
		Value: c.convertVSCodeVariables(program),
	})

	// Add application parameters
	if len(task.Args) > 0 {
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "APPLICATION_PARAMETERS",
			Value: strings.Join(task.Args, " "),
		})
	}

	return nil
}

// addPythonOptions adds Python-specific options
func (c *VSCodeLaunchToJetBrainsConverter) addPythonOptions(task *config.Task, config *JetBrainsRunConfiguration) error {
	// Check if this is a Python module execution (python -m module)
	if len(task.Args) >= 2 && task.Args[0] == "-m" {
		// For module execution, we need to set SCRIPT_NAME to a dummy value
		// and put the module execution in PARAMETERS
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "SCRIPT_NAME",
			Value: "python", // Dummy script name for module execution
		})

		// The parameters should include the full module execution
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "PARAMETERS",
			Value: strings.Join(task.Args, " "),
		})

		return nil
	}

	// Extract Python script path for regular script execution
	program := c.extractProgramFromLaunch(task)
	if program == "" {
		return fmt.Errorf("could not determine program for Python application '%s'", task.Name)
	}

	config.Options = append(config.Options, JetBrainsOption{
		Name:  "SCRIPT_NAME",
		Value: c.convertVSCodeVariables(program),
	})

	// Add parameters
	if len(task.Args) > 0 {
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "PARAMETERS",
			Value: strings.Join(task.Args, " "),
		})
	}

	return nil
}

// addGenericOptions adds generic executable options
func (c *VSCodeLaunchToJetBrainsConverter) addGenericOptions(task *config.Task, config *JetBrainsRunConfiguration) error {
	// Use command as the executable
	if task.Command != "" {
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "PROGRAM_PARAMETERS",
			Value: task.Command,
		})
	}

	// Add arguments
	if len(task.Args) > 0 {
		existing := ""

		for i, opt := range config.Options {
			if opt.Name == "PROGRAM_PARAMETERS" {
				existing = opt.Value
				config.Options[i].Value = existing + " " + strings.Join(task.Args, " ")

				return nil
			}
		}

		config.Options = append(config.Options, JetBrainsOption{
			Name:  "PROGRAM_PARAMETERS",
			Value: strings.Join(task.Args, " "),
		})
	}

	return nil
}

// extractMainClassFromLaunch extracts main class from VSCode launch config
func (c *VSCodeLaunchToJetBrainsConverter) extractMainClassFromLaunch(task *config.Task) string {
	// Check if mainClass is specified in command (common pattern)
	if strings.Contains(task.Command, "mainClass") {
		// Parse command that might contain "mainClass": "com.example.Main"
		parts := strings.Fields(task.Command)
		for i, part := range parts {
			if part == "mainClass" && i+1 < len(parts) {
				return strings.Trim(parts[i+1], `"`)
			}
		}
	}

	// Look for class-like names in command
	parts := strings.Fields(task.Command)
	for _, part := range parts {
		if strings.Contains(part, ".") && !strings.HasPrefix(part, "-") && !strings.HasSuffix(part, ".jar") {
			return part
		}
	}

	// Look in args
	for _, arg := range task.Args {
		if strings.Contains(arg, ".") && !strings.HasPrefix(arg, "-") && !strings.HasSuffix(arg, ".jar") {
			return arg
		}
	}

	return ""
}

// extractProgramFromLaunch extracts program path from VSCode launch config
func (c *VSCodeLaunchToJetBrainsConverter) extractProgramFromLaunch(task *config.Task) string {
	// Check if program is specified in command
	if strings.Contains(task.Command, "program") {
		// Parse command that might contain "program": "/path/to/file"
		parts := strings.Fields(task.Command)
		for i, part := range parts {
			if part == "program" && i+1 < len(parts) {
				return strings.Trim(parts[i+1], `"`)
			}
		}
	}

	// Look for file paths in command
	parts := strings.Fields(task.Command)
	for _, part := range parts {
		if strings.Contains(part, "/") || strings.Contains(part, "\\") ||
			strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts") ||
			strings.HasSuffix(part, ".py") {
			return part
		}
	}

	// Look in args
	for _, arg := range task.Args {
		if strings.Contains(arg, "/") || strings.Contains(arg, "\\") ||
			strings.HasSuffix(arg, ".js") || strings.HasSuffix(arg, ".ts") ||
			strings.HasSuffix(arg, ".py") {
			return arg
		}
	}

	return ""
}

// filterArgsExcluding filters out specific values from args
func (c *VSCodeLaunchToJetBrainsConverter) filterArgsExcluding(args []string, exclude string) []string {
	var filtered []string

	for _, arg := range args {
		if arg != exclude {
			filtered = append(filtered, arg)
		}
	}

	return filtered
}

// convertVSCodeVariables converts VSCode variables to JetBrains format (reuse from vscode_to_jetbrains.go)
func (c *VSCodeLaunchToJetBrainsConverter) convertVSCodeVariables(input string) string {
	result := input

	// Convert VSCode variables to JetBrains equivalents
	result = strings.ReplaceAll(result, "${workspaceFolder}", "$PROJECT_DIR$")
	result = strings.ReplaceAll(result, "${workspaceRoot}", "$PROJECT_DIR$")
	result = strings.ReplaceAll(result, "${fileDirname}", "$FileDir$")
	result = strings.ReplaceAll(result, "${fileBasename}", "$FileName$")
	result = strings.ReplaceAll(result, "${file}", "$FilePath$")
	result = strings.ReplaceAll(result, "${relativeFile}", "$FilePathRelativeToProjectRoot$")

	return result
}

// sanitizeFilename removes invalid characters from filename (reuse from vscode_to_jetbrains.go)
func (c *VSCodeLaunchToJetBrainsConverter) sanitizeFilename(name string) string {
	// Replace invalid filename characters with underscores
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}

	result := name

	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}

// writeJetBrainsRunConfig writes the JetBrains run configuration XML (reuse from vscode_to_jetbrains.go)
func (c *VSCodeLaunchToJetBrainsConverter) writeJetBrainsRunConfig(config *JetBrainsRunConfiguration, outputPath string) error {
	// Create the XML structure
	component := JetBrainsComponent{
		Name:          "ProjectRunConfigurationManager",
		Configuration: *config,
	}

	// Marshal to XML with proper formatting
	xmlData, err := xml.MarshalIndent(component, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML declaration
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(xmlData)

	// Write to file
	if err := os.WriteFile(outputPath, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractGoPackageFromLaunch extracts the Go package path from launch task
func (c *VSCodeLaunchToJetBrainsConverter) extractGoPackageFromLaunch(task *config.Task) string {
	// Look for package path in args after "run"
	for i, arg := range task.Args {
		if arg == "run" && i+1 < len(task.Args) {
			packagePath := task.Args[i+1]
			// Convert VSCode variables
			packagePath = c.convertVSCodeVariables(packagePath)
			// If it's the current directory, return "."
			if packagePath == "$PROJECT_DIR$" {
				return "."
			}

			return packagePath
		}
	}

	// Default to current directory
	return "."
}

// filterGoArgsFromLaunch filters out go command and package path, returning only program arguments
func (c *VSCodeLaunchToJetBrainsConverter) filterGoArgsFromLaunch(task *config.Task) []string {
	var filtered []string

	skipNext := false

	for _, arg := range task.Args {
		if skipNext {
			skipNext = false
			continue
		}

		// Skip "run" command and the package path that follows it
		if arg == "run" {
			skipNext = true
			continue
		}

		// Include everything else as program arguments
		filtered = append(filtered, arg)
	}

	return filtered
}
