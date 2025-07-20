package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskporter/internal/config"
)

// LaunchParser handles parsing of VSCode launch.json files
type LaunchParser struct {
	projectRoot string
}

// NewLaunchParser creates a new VSCode launch parser
func NewLaunchParser(projectRoot string) *LaunchParser {
	return &LaunchParser{
		projectRoot: projectRoot,
	}
}

// ParseLaunchConfigs parses a VSCode launch.json file and returns internal Task structures
func (p *LaunchParser) ParseLaunchConfigs(launchFilePath string) ([]*config.Task, error) {
	data, err := os.ReadFile(launchFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read launch file %s: %w", launchFilePath, err)
	}

	var launchFile VSCodeLaunchFile
	if err := json.Unmarshal(data, &launchFile); err != nil {
		return nil, fmt.Errorf("failed to parse launch JSON: %w", err)
	}

	var tasks []*config.Task
	for _, vscodeConfig := range launchFile.Configurations {
		task, err := p.convertLaunchConfig(vscodeConfig, launchFilePath)
		if err != nil {
			// Log error but continue with other configs
			fmt.Printf("Warning: failed to convert launch config %s: %v\n", vscodeConfig.Name, err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// convertLaunchConfig converts a VSCode launch config to our internal Task structure
func (p *LaunchParser) convertLaunchConfig(vscodeConfig VSCodeLaunchConfig, sourceFile string) (*config.Task, error) {
	task := &config.Task{
		Name:        vscodeConfig.Name,
		Type:        config.TypeVSCodeLaunch,
		Source:      sourceFile,
		Description: fmt.Sprintf("%s %s configuration", vscodeConfig.Type, vscodeConfig.Request),
	}

	// Handle different launch types
	switch vscodeConfig.Type {
	case "go":
		if err := p.handleGoLaunchConfig(vscodeConfig, task); err != nil {
			return nil, err
		}
	case "node":
		if err := p.handleNodeLaunchConfig(vscodeConfig, task); err != nil {
			return nil, err
		}
	case "python":
		if err := p.handlePythonLaunchConfig(vscodeConfig, task); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported launch type: %s", vscodeConfig.Type)
	}

	// Handle common properties
	if vscodeConfig.Cwd != "" {
		task.Cwd = p.resolveWorkspacePath(vscodeConfig.Cwd)
	}

	// Set default working directory to project root if not specified
	if task.Cwd == "" {
		task.Cwd = p.projectRoot
	}

	// Handle environment variables
	if vscodeConfig.Env != nil {
		task.Env = make(map[string]string)
		for k, v := range vscodeConfig.Env {
			// Only resolve workspace variables, leave other values as-is
			if strings.Contains(v, "${workspace") {
				task.Env[k] = p.resolveWorkspacePath(v)
			} else {
				task.Env[k] = v
			}
		}
	}

	// Set group based on request type
	switch vscodeConfig.Request {
	case "launch":
		task.Group = "launch"
	case "attach":
		task.Group = "debug"
	default:
		task.Group = "launch"
	}

	return task, nil
}

// resolveWorkspacePath resolves VSCode workspace variables in paths
func (p *LaunchParser) resolveWorkspacePath(path string) string {
	// Replace common VSCode variables
	resolved := strings.ReplaceAll(path, "${workspaceFolder}", p.projectRoot)
	resolved = strings.ReplaceAll(resolved, "${workspaceRoot}", p.projectRoot)

	// Handle relative paths
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(p.projectRoot, resolved)
	}

	return resolved
}

// GetPreLaunchTask returns the preLaunchTask name if specified
func (p *LaunchParser) GetPreLaunchTask(launchFilePath string, configName string) (string, error) {
	data, err := os.ReadFile(launchFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read launch file: %w", err)
	}

	var launchFile VSCodeLaunchFile
	if err := json.Unmarshal(data, &launchFile); err != nil {
		return "", fmt.Errorf("failed to parse launch JSON: %w", err)
	}

	for _, config := range launchFile.Configurations {
		if config.Name == configName {
			return config.PreLaunchTask, nil
		}
	}

	return "", fmt.Errorf("launch configuration '%s' not found", configName)
}

// handleGoLaunchConfig handles Go-specific launch configuration
func (p *LaunchParser) handleGoLaunchConfig(vscodeConfig VSCodeLaunchConfig, task *config.Task) error {
	switch vscodeConfig.Request {
	case "launch":
		task.Command = "go"
		if vscodeConfig.Mode == "debug" {
			// For debug mode, we could use delve, but for simplicity we'll use go run
			task.Args = []string{"run"}
		} else {
			task.Args = []string{"run"}
		}

		// Add program path
		if vscodeConfig.Program != "" {
			programPath := p.resolveWorkspacePath(vscodeConfig.Program)
			task.Args = append(task.Args, programPath)
		} else {
			task.Args = append(task.Args, ".")
		}

		// Add arguments
		if len(vscodeConfig.Args) > 0 {
			task.Args = append(task.Args, vscodeConfig.Args...)
		}

	case "attach":
		return fmt.Errorf("go attach mode not yet supported")

	default:
		return fmt.Errorf("unsupported Go request type: %s", vscodeConfig.Request)
	}

	return nil
}

// handleNodeLaunchConfig handles Node.js-specific launch configuration
func (p *LaunchParser) handleNodeLaunchConfig(vscodeConfig VSCodeLaunchConfig, task *config.Task) error {
	switch vscodeConfig.Request {
	case "launch":
		task.Command = "node"

		// Add program path
		if vscodeConfig.Program != "" {
			programPath := p.resolveWorkspacePath(vscodeConfig.Program)
			task.Args = []string{programPath}
		} else {
			return fmt.Errorf("node.js launch config requires program path")
		}

		// Add arguments
		if len(vscodeConfig.Args) > 0 {
			task.Args = append(task.Args, vscodeConfig.Args...)
		}

	case "attach":
		return fmt.Errorf("node.js attach mode not yet supported")

	default:
		return fmt.Errorf("unsupported Node.js request type: %s", vscodeConfig.Request)
	}

	return nil
}

// handlePythonLaunchConfig handles Python-specific launch configuration
func (p *LaunchParser) handlePythonLaunchConfig(vscodeConfig VSCodeLaunchConfig, task *config.Task) error {
	switch vscodeConfig.Request {
	case "launch":
		task.Command = "python"

		// Add program path
		if vscodeConfig.Program != "" {
			programPath := p.resolveWorkspacePath(vscodeConfig.Program)
			task.Args = []string{programPath}
		} else {
			return fmt.Errorf("python launch config requires program path")
		}

		// Add arguments
		if len(vscodeConfig.Args) > 0 {
			task.Args = append(task.Args, vscodeConfig.Args...)
		}

	case "attach":
		return fmt.Errorf("python attach mode not yet supported")

	default:
		return fmt.Errorf("unsupported Python request type: %s", vscodeConfig.Request)
	}

	return nil
}
