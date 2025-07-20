package converter

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskporter/internal/config"
)

// VSCodeToJetBrainsConverter converts VSCode tasks to JetBrains run configurations
type VSCodeToJetBrainsConverter struct {
	projectRoot string
	outputPath  string
	verbose     bool
}

// NewVSCodeToJetBrainsConverter creates a new converter
func NewVSCodeToJetBrainsConverter(projectRoot, outputPath string, verbose bool) *VSCodeToJetBrainsConverter {
	return &VSCodeToJetBrainsConverter{
		projectRoot: projectRoot,
		outputPath:  outputPath,
		verbose:     verbose,
	}
}

// ConvertTasks converts VSCode tasks to JetBrains run configurations
func (c *VSCodeToJetBrainsConverter) ConvertTasks(tasks []*config.Task, dryRun bool) error {
	if c.verbose {
		fmt.Printf("🔄 Converting %d VSCode tasks to JetBrains format...\n", len(tasks))
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

	for _, task := range tasks {
		// Only convert VSCode tasks (not launch configs)
		if !strings.HasPrefix(string(task.Type), "vscode-task") {
			if c.verbose {
				fmt.Printf("⏭️  Skipping non-VSCode task: %s (type: %s)\n", task.Name, string(task.Type))
			}

			continue
		}

		jetbrainsConfig, err := c.convertSingleTask(task)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert task '%s': %v\n", task.Name, err)
			continue
		}

		// Generate filename (sanitize name for filesystem)
		filename := sanitizeFilename(task.Name) + ".xml"
		filepath := filepath.Join(outputDir, filename)

		if c.verbose {
			fmt.Printf("📝 Converting task: %s → %s\n", task.Name, filename)
		}

		if dryRun {
			fmt.Printf("   [DRY RUN] Would create: %s\n", filepath)
		} else {
			if err := c.writeJetBrainsConfig(jetbrainsConfig, filepath); err != nil {
				fmt.Printf("⚠️  Warning: failed to write config for '%s': %v\n", task.Name, err)
				continue
			}
		}

		convertedCount++
	}

	if c.verbose {
		fmt.Printf("✅ Successfully converted %d/%d tasks\n", convertedCount, len(tasks))
	}

	return nil
}

// convertSingleTask converts a single VSCode task to JetBrains format
func (c *VSCodeToJetBrainsConverter) convertSingleTask(task *config.Task) (*JetBrainsRunConfiguration, error) {
	// Determine configuration type based on task
	configType := c.determineConfigType(task)

	config := &JetBrainsRunConfiguration{
		Name:    task.Name,
		Type:    configType,
		Options: make([]JetBrainsOption, 0),
		EnvVars: nil,
	}

	// Add options based on task type (type was already determined by determineConfigType)
	switch config.Type {
	case "Application":
		mainClass := c.extractMainClass(task)

		config.Options = append(config.Options, JetBrainsOption{
			Name:  "MAIN_CLASS_NAME",
			Value: mainClass,
		})
		if len(task.Args) > 0 {
			config.Options = append(config.Options, JetBrainsOption{
				Name:  "PROGRAM_PARAMETERS",
				Value: strings.Join(task.Args, " "),
			})
		}
	case "GradleRunTask":
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "TASK_NAME",
			Value: strings.Join(task.Args, " "),
		})
	case "MavenRunConfiguration":
		config.Options = append(config.Options, JetBrainsOption{
			Name:  "GOALS",
			Value: strings.Join(task.Args, " "),
		})
	default:
		// Generic shell/external tool configuration or other types
		scriptText := task.Command
		if len(task.Args) > 0 {
			scriptText += " " + strings.Join(task.Args, " ")
		}

		config.Options = append(config.Options, JetBrainsOption{
			Name:  "SCRIPT_TEXT",
			Value: scriptText,
		})
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
		envVars := make([]JetBrainsEnvVar, 0, len(task.Env))
		for key, value := range task.Env {
			envVars = append(envVars, JetBrainsEnvVar{
				Name:  key,
				Value: c.convertVSCodeVariables(value),
			})
		}

		config.EnvVars = &JetBrainsEnvVars{
			EnvVars: envVars,
		}
	}

	return config, nil
}

// determineConfigType determines the best JetBrains configuration type for a task
func (c *VSCodeToJetBrainsConverter) determineConfigType(task *config.Task) string {
	command := strings.ToLower(task.Command)

	switch {
	case command == "java":
		return "Application"
	case strings.Contains(command, "gradle"):
		return "GradleRunTask"
	case strings.Contains(command, "maven") || strings.Contains(command, "mvn"):
		return "MavenRunConfiguration"
	case strings.Contains(command, "npm") || strings.Contains(command, "node"):
		return "NodeJS"
	case strings.Contains(command, "python") || strings.Contains(command, "py"):
		return "PythonConfigurationType"
	default:
		return "ShellScript"
	}
}

// extractMainClass attempts to extract a main class from Java-related tasks
func (c *VSCodeToJetBrainsConverter) extractMainClass(task *config.Task) string {
	// Look for main class in args
	for i := 0; i < len(task.Args); i++ {
		arg := task.Args[i]
		if strings.Contains(arg, ".") && !strings.HasPrefix(arg, "-") {
			// Likely a class name
			return arg
		}

		if arg == "-cp" || arg == "--class-path" {
			// Skip classpath argument and its value
			if i+1 < len(task.Args) {
				i++ // Skip the classpath value
			}

			continue
		}
	}

	// Default fallback
	return "Main"
}

// convertVSCodeVariables converts VSCode variables to JetBrains equivalents
func (c *VSCodeToJetBrainsConverter) convertVSCodeVariables(path string) string {
	// Common VSCode → JetBrains variable mappings
	replacements := map[string]string{
		"${workspaceFolder}": "$PROJECT_DIR$",
		"${workspaceRoot}":   "$PROJECT_DIR$",
		"${file}":            "$FilePath$",
		"${fileBasename}":    "$FileName$",
		"${fileDirname}":     "$FileDir$",
		"${fileExtname}":     "$FileExt$",
	}

	result := path
	for vscode, jetbrains := range replacements {
		result = strings.ReplaceAll(result, vscode, jetbrains)
	}

	return result
}

// writeJetBrainsConfig writes the JetBrains configuration to an XML file
func (c *VSCodeToJetBrainsConverter) writeJetBrainsConfig(config *JetBrainsRunConfiguration, filepath string) error {
	// Create the root component structure that JetBrains expects
	component := &JetBrainsComponent{
		Name:          "ProjectRunConfigurationManager",
		Configuration: *config,
	}

	// Marshal to XML with proper formatting
	xmlData, err := xml.MarshalIndent(component, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML declaration
	xmlContent := []byte(xml.Header + string(xmlData))

	// Write to file
	if err := os.WriteFile(filepath, xmlContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	// Replace invalid filename characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Replace spaces with underscores for consistency
	result = strings.ReplaceAll(result, " ", "_")

	return result
}

// JetBrains XML structures for run configurations
type JetBrainsComponent struct {
	XMLName       xml.Name                  `xml:"component"`
	Name          string                    `xml:"name,attr"`
	Configuration JetBrainsRunConfiguration `xml:"configuration"`
}

type JetBrainsRunConfiguration struct {
	XMLName xml.Name          `xml:"configuration"`
	Name    string            `xml:"name,attr"`
	Type    string            `xml:"type,attr"`
	Options []JetBrainsOption `xml:"option"`
	EnvVars *JetBrainsEnvVars `xml:"envs,omitempty"`
}

type JetBrainsOption struct {
	XMLName xml.Name `xml:"option"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

type JetBrainsEnvVars struct {
	XMLName xml.Name          `xml:"envs"`
	EnvVars []JetBrainsEnvVar `xml:"env"`
}

type JetBrainsEnvVar struct {
	XMLName xml.Name `xml:"env"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}
