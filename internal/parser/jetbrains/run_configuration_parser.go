package jetbrains

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskporter/internal/config"
)

// RunConfigurationParser handles parsing of JetBrains run configuration XML files
type RunConfigurationParser struct {
	projectRoot string
}

// NewRunConfigurationParser creates a new JetBrains run configuration parser
func NewRunConfigurationParser(projectRoot string) *RunConfigurationParser {
	return &RunConfigurationParser{
		projectRoot: projectRoot,
	}
}

// ParseRunConfiguration parses a JetBrains run configuration XML file and returns internal Task structure
func (p *RunConfigurationParser) ParseRunConfiguration(configFilePath string) (*config.Task, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	var jetbrainsConfig JetBrainsConfiguration
	if err := xml.Unmarshal(data, &jetbrainsConfig); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Convert JetBrains configuration to our internal Task structure
	task, err := p.convertRunConfiguration(jetbrainsConfig.Configuration, configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert configuration: %w", err)
	}

	return task, nil
}

// convertRunConfiguration converts a JetBrains run config to our internal Task structure
func (p *RunConfigurationParser) convertRunConfiguration(jetbrainsConfig JetBrainsRunConfiguration, sourceFile string) (*config.Task, error) {
	task := &config.Task{
		Name:        jetbrainsConfig.Name,
		Type:        config.TypeJetBrains,
		Source:      sourceFile,
		Description: fmt.Sprintf("JetBrains %s configuration", jetbrainsConfig.Type),
	}

	// Handle different configuration types
	switch jetbrainsConfig.Type {
	case "Application":
		if err := p.handleApplicationConfig(jetbrainsConfig, task); err != nil {
			return nil, err
		}
	case "GradleRunConfiguration":
		if err := p.handleGradleConfig(jetbrainsConfig, task); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported JetBrains configuration type: %s", jetbrainsConfig.Type)
	}

	// Set default working directory to project root if not specified
	if task.Cwd == "" {
		task.Cwd = p.projectRoot
	}

	return task, nil
}

// handleApplicationConfig handles Java Application run configurations
func (p *RunConfigurationParser) handleApplicationConfig(jetbrainsConfig JetBrainsRunConfiguration, task *config.Task) error {
	task.Command = "java"
	task.Group = "run"

	var (
		mainClass         string
		vmParameters      string
		programParameters string
		workingDirectory  string
		envVars           map[string]string
	)

	// Parse options

	for _, option := range jetbrainsConfig.Options {
		switch option.Name {
		case "MAIN_CLASS_NAME":
			mainClass = option.Value
		case "VM_PARAMETERS":
			vmParameters = option.Value
		case "PROGRAM_PARAMETERS":
			programParameters = option.Value
		case "WORKING_DIRECTORY":
			workingDirectory = option.Value
		case "ENV_VARIABLES":
			if option.Map != nil {
				envVars = make(map[string]string)
				for _, entry := range option.Map.Entries {
					envVars[entry.Key] = entry.Value
				}
			}
		}
	}

	// Build command arguments
	var args []string

	// Add VM parameters
	if vmParameters != "" {
		vmArgs := p.parseParameters(vmParameters)
		args = append(args, vmArgs...)
	}

	// Add main class
	if mainClass == "" {
		return fmt.Errorf("MAIN_CLASS_NAME is required for Application configuration")
	}

	args = append(args, mainClass)

	// Add program parameters
	if programParameters != "" {
		progArgs := p.parseParameters(programParameters)
		args = append(args, progArgs...)
	}

	task.Args = args

	// Set working directory
	if workingDirectory != "" {
		task.Cwd = p.resolveJetBrainsPath(workingDirectory)
	}

	// Set environment variables
	if envVars != nil {
		task.Env = envVars
	}

	return nil
}

// handleGradleConfig handles Gradle run configurations
func (p *RunConfigurationParser) handleGradleConfig(jetbrainsConfig JetBrainsRunConfiguration, task *config.Task) error {
	task.Command = "gradle"
	task.Group = "build"

	if jetbrainsConfig.ExternalSystemSettings == nil {
		return fmt.Errorf("ExternalSystemSettings is required for Gradle configuration")
	}

	var (
		taskNames        []string
		scriptParameters string
	)

	// Parse external system settings

	for _, option := range jetbrainsConfig.ExternalSystemSettings.Options {
		switch option.Name {
		case "taskNames":
			if option.List != nil {
				for _, listOption := range option.List.Options {
					taskNames = append(taskNames, listOption.Value)
				}
			}
		case "scriptParameters":
			scriptParameters = option.Value
		}
	}

	// Build command arguments
	var args []string

	args = append(args, taskNames...)

	// Add script parameters
	if scriptParameters != "" {
		scriptArgs := p.parseParameters(scriptParameters)
		args = append(args, scriptArgs...)
	}

	task.Args = args

	return nil
}

// parseParameters parses a parameter string and splits it into individual arguments
func (p *RunConfigurationParser) parseParameters(params string) []string {
	if params == "" {
		return nil
	}

	// Parse parameters with quoted string support
	var (
		args      []string
		current   strings.Builder
		inQuote   bool
		quoteChar rune
	)

	for _, char := range params {
		switch {
		case !inQuote && (char == '"' || char == '\''):
			// Start of quoted string - don't include the quote in output
			inQuote = true
			quoteChar = char
		case inQuote && char == quoteChar:
			// End of quoted string - don't include the quote in output
			inQuote = false
		case !inQuote && char == ' ':
			// Space outside quotes - end current argument
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			// Regular character or space inside quotes
			current.WriteRune(char)
		}
	}

	// Add final argument
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// resolveJetBrainsPath resolves JetBrains variables in paths
func (p *RunConfigurationParser) resolveJetBrainsPath(path string) string {
	// Replace common JetBrains variables
	resolved := strings.ReplaceAll(path, "$PROJECT_DIR$", p.projectRoot)
	resolved = strings.ReplaceAll(resolved, "$MODULE_DIR$", p.projectRoot)

	// Handle relative paths
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(p.projectRoot, resolved)
	}

	return resolved
}
