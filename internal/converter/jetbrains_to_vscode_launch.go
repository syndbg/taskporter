package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskporter/internal/config"
)

// JetBrainsToVSCodeLaunchConverter converts JetBrains run configurations to VSCode launch configs
type JetBrainsToVSCodeLaunchConverter struct {
	projectRoot string
	outputPath  string
	verbose     bool
}

// NewJetBrainsToVSCodeLaunchConverter creates a new launch converter
func NewJetBrainsToVSCodeLaunchConverter(projectRoot, outputPath string, verbose bool) *JetBrainsToVSCodeLaunchConverter {
	return &JetBrainsToVSCodeLaunchConverter{
		projectRoot: projectRoot,
		outputPath:  outputPath,
		verbose:     verbose,
	}
}

// VSCodeLaunchFile represents the structure of launch.json
type VSCodeLaunchFile struct {
	Version        string               `json:"version"`
	Configurations []VSCodeLaunchConfig `json:"configurations"`
}

// VSCodeLaunchConfig represents a single launch configuration in launch.json
type VSCodeLaunchConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Request     string            `json:"request"`
	Program     string            `json:"program,omitempty"`
	MainClass   string            `json:"mainClass,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Console     string            `json:"console,omitempty"`
	StopOnEntry bool              `json:"stopOnEntry,omitempty"`
}

// ConvertToLaunch converts JetBrains tasks to VSCode launch.json format
func (c *JetBrainsToVSCodeLaunchConverter) ConvertToLaunch(tasks []*config.Task, dryRun bool) error {
	if c.verbose {
		fmt.Printf("🔄 Converting %d JetBrains configurations to VSCode launch format...\n", len(tasks))
	}

	// Filter only JetBrains tasks that can be converted to launch configs
	jetBrainsTasks := make([]*config.Task, 0)
	for _, task := range tasks {
		if task.Type == config.TypeJetBrains && c.canConvertToLaunch(task) {
			jetBrainsTasks = append(jetBrainsTasks, task)
		}
	}

	if len(jetBrainsTasks) == 0 {
		fmt.Printf("⚠️  No JetBrains configurations suitable for launch conversion found\n")
		fmt.Printf("💡 Note: Only Application-type JetBrains configs can be converted to launch configurations\n")
		return nil
	}

	if c.verbose {
		fmt.Printf("📋 Converting %d suitable JetBrains configurations\n", len(jetBrainsTasks))
	}

	// Convert tasks
	launchFile := &VSCodeLaunchFile{
		Version:        "0.2.0",
		Configurations: make([]VSCodeLaunchConfig, 0, len(jetBrainsTasks)),
	}

	for _, task := range jetBrainsTasks {
		launchConfig, err := c.convertSingleTaskToLaunch(task)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert task '%s': %v\n", task.Name, err)
			continue
		}
		launchFile.Configurations = append(launchFile.Configurations, *launchConfig)
	}

	// Determine output path
	outputPath := c.outputPath
	if outputPath == "" {
		outputPath = filepath.Join(c.projectRoot, ".vscode", "launch.json")
	}

	if c.verbose {
		fmt.Printf("📁 Output file: %s\n", outputPath)
	}

	if dryRun {
		fmt.Printf("   [DRY RUN] Would create: %s\n", outputPath)
		fmt.Printf("📝 Preview of launch.json content:\n")

		jsonData, _ := json.MarshalIndent(launchFile, "", "    ")
		fmt.Printf("%s\n", string(jsonData))
	} else {
		// Create output directory
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write launch.json file
		if err := c.writeVSCodeLaunchFile(launchFile, outputPath); err != nil {
			return fmt.Errorf("failed to write launch.json: %w", err)
		}

		if c.verbose {
			fmt.Printf("✅ Successfully created %s\n", outputPath)
		}
	}

	fmt.Printf("✅ Successfully converted %d/%d JetBrains configurations to launch configs\n", len(launchFile.Configurations), len(jetBrainsTasks))
	return nil
}

// canConvertToLaunch determines if a JetBrains task can be converted to a launch config
func (c *JetBrainsToVSCodeLaunchConverter) canConvertToLaunch(task *config.Task) bool {
	command := strings.ToLower(task.Command)

	// Only convert Application-type configurations or those with executable commands
	return strings.Contains(command, "java") ||
		strings.Contains(command, "node") ||
		strings.Contains(command, "python") ||
		strings.Contains(task.Name, "Application") ||
		(strings.Contains(task.Command, " ") && !strings.Contains(command, "gradle") && !strings.Contains(command, "mvn"))
}

// convertSingleTaskToLaunch converts a single JetBrains task to VSCode launch format
func (c *JetBrainsToVSCodeLaunchConverter) convertSingleTaskToLaunch(task *config.Task) (*VSCodeLaunchConfig, error) {
	launchConfig := &VSCodeLaunchConfig{
		Name:    task.Name,
		Request: "launch",
	}

	// Determine launch type and configuration based on command
	if err := c.determineLaunchType(task, launchConfig); err != nil {
		return nil, err
	}

	// Set working directory (convert JetBrains variables)
	if task.Cwd != "" {
		launchConfig.Cwd = c.convertJetBrainsVariables(task.Cwd)
	} else {
		launchConfig.Cwd = "${workspaceFolder}"
	}

	// Convert environment variables
	if len(task.Env) > 0 {
		launchConfig.Env = make(map[string]string)
		for key, value := range task.Env {
			launchConfig.Env[key] = c.convertJetBrainsVariables(value)
		}
	}

	return launchConfig, nil
}

// determineLaunchType sets the appropriate launch type and configuration
func (c *JetBrainsToVSCodeLaunchConverter) determineLaunchType(task *config.Task, launchConfig *VSCodeLaunchConfig) error {
	command := strings.ToLower(task.Command)

	if strings.Contains(command, "java") {
		// Java application
		launchConfig.Type = "java"

		// Extract main class from command or args
		mainClass := c.extractJavaMainClass(task)
		if mainClass == "" {
			return fmt.Errorf("could not determine main class for Java application '%s'", task.Name)
		}
		launchConfig.MainClass = mainClass

		// Add program arguments (excluding main class)
		args := c.extractJavaArgs(task, mainClass)
		if len(args) > 0 {
			launchConfig.Args = args
		}

	} else if strings.Contains(command, "node") {
		// Node.js application
		launchConfig.Type = "node"

		// Extract program path
		program := c.extractNodeProgram(task)
		if program == "" {
			return fmt.Errorf("could not determine program for Node.js application '%s'", task.Name)
		}
		launchConfig.Program = c.convertJetBrainsVariables(program)

		// Add arguments
		args := c.extractNodeArgs(task)
		if len(args) > 0 {
			launchConfig.Args = args
		}

	} else if strings.Contains(command, "python") {
		// Python application
		launchConfig.Type = "python"

		// Extract program path
		program := c.extractPythonProgram(task)
		if program == "" {
			return fmt.Errorf("could not determine program for Python application '%s'", task.Name)
		}
		launchConfig.Program = c.convertJetBrainsVariables(program)

		// Add arguments
		args := c.extractPythonArgs(task)
		if len(args) > 0 {
			launchConfig.Args = args
		}

	} else {
		// Generic external tool - use node as fallback
		launchConfig.Type = "node"

		// Try to extract program from command
		parts := strings.Fields(task.Command)
		if len(parts) > 0 {
			// Use the first part as program, rest as args
			launchConfig.Program = c.convertJetBrainsVariables(parts[0])
			if len(parts) > 1 {
				launchConfig.Args = append(parts[1:], task.Args...)
			} else if len(task.Args) > 0 {
				launchConfig.Args = task.Args
			}
		} else {
			return fmt.Errorf("could not determine program for task '%s'", task.Name)
		}
	}

	return nil
}

// extractJavaMainClass extracts the main class from Java command
func (c *JetBrainsToVSCodeLaunchConverter) extractJavaMainClass(task *config.Task) string {
	// Look in command arguments for class name
	parts := strings.Fields(task.Command)
	for _, part := range parts {
		if strings.Contains(part, ".") && !strings.HasPrefix(part, "-") {
			return part
		}
	}

	// Look in task args
	for _, arg := range task.Args {
		if strings.Contains(arg, ".") && !strings.HasPrefix(arg, "-") && !strings.HasSuffix(arg, ".jar") {
			return arg
		}
	}

	return "Main" // fallback
}

// extractJavaArgs extracts Java program arguments (excluding main class)
func (c *JetBrainsToVSCodeLaunchConverter) extractJavaArgs(task *config.Task, mainClass string) []string {
	var args []string

	// Add args from task, excluding main class
	for _, arg := range task.Args {
		if arg != mainClass {
			args = append(args, arg)
		}
	}

	return args
}

// extractNodeProgram extracts the Node.js program path
func (c *JetBrainsToVSCodeLaunchConverter) extractNodeProgram(task *config.Task) string {
	parts := strings.Fields(task.Command)

	// Look for the script file in command parts
	for i, part := range parts {
		if i > 0 && (strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts")) {
			return part
		}
	}

	// Look in args
	for _, arg := range task.Args {
		if strings.HasSuffix(arg, ".js") || strings.HasSuffix(arg, ".ts") {
			return arg
		}
	}

	// Fallback to a common pattern
	return "${workspaceFolder}/index.js"
}

// extractNodeArgs extracts Node.js program arguments
func (c *JetBrainsToVSCodeLaunchConverter) extractNodeArgs(task *config.Task) []string {
	var args []string

	// Extract args from command (skip 'node' and program file)
	parts := strings.Fields(task.Command)
	foundProgram := false
	for i, part := range parts {
		if i == 0 { // skip 'node'
			continue
		}
		if !foundProgram && (strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts")) {
			foundProgram = true
			continue
		}
		if foundProgram {
			args = append(args, part)
		}
	}

	// Add task args
	args = append(args, task.Args...)

	return args
}

// extractPythonProgram extracts the Python program path
func (c *JetBrainsToVSCodeLaunchConverter) extractPythonProgram(task *config.Task) string {
	parts := strings.Fields(task.Command)

	// Look for the script file in command parts
	for i, part := range parts {
		if i > 0 && strings.HasSuffix(part, ".py") {
			return part
		}
	}

	// Look in args
	for _, arg := range task.Args {
		if strings.HasSuffix(arg, ".py") {
			return arg
		}
	}

	// Fallback
	return "${workspaceFolder}/main.py"
}

// extractPythonArgs extracts Python program arguments
func (c *JetBrainsToVSCodeLaunchConverter) extractPythonArgs(task *config.Task) []string {
	var args []string

	// Extract args from command (skip 'python' and program file)
	parts := strings.Fields(task.Command)
	foundProgram := false
	for i, part := range parts {
		if i == 0 { // skip 'python'
			continue
		}
		if !foundProgram && strings.HasSuffix(part, ".py") {
			foundProgram = true
			continue
		}
		if foundProgram {
			args = append(args, part)
		}
	}

	// Add task args
	args = append(args, task.Args...)

	return args
}

// convertJetBrainsVariables converts JetBrains variables to VSCode format (same as in jetbrains_to_vscode.go)
func (c *JetBrainsToVSCodeLaunchConverter) convertJetBrainsVariables(input string) string {
	result := input

	// Convert JetBrains variables to VSCode equivalents
	result = strings.ReplaceAll(result, "$PROJECT_DIR$", "${workspaceFolder}")
	result = strings.ReplaceAll(result, "$MODULE_DIR$", "${workspaceFolder}")
	result = strings.ReplaceAll(result, "$FileDir$", "${fileDirname}")
	result = strings.ReplaceAll(result, "$FileName$", "${fileBasename}")
	result = strings.ReplaceAll(result, "$FilePath$", "${file}")

	return result
}

// writeVSCodeLaunchFile writes the VSCode launch file
func (c *JetBrainsToVSCodeLaunchConverter) writeVSCodeLaunchFile(launchFile *VSCodeLaunchFile, outputPath string) error {
	jsonData, err := json.MarshalIndent(launchFile, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal launch.json: %w", err)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
