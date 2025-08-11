package vscode

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// WorkspaceResolver handles resolution of VSCode workspace variables [[memory:3927270]]
type WorkspaceResolver struct {
	projectRoot string
}

// NewWorkspaceResolver creates a new workspace variable resolver
func NewWorkspaceResolver(projectRoot string) *WorkspaceResolver {
	return &WorkspaceResolver{
		projectRoot: projectRoot,
	}
}

// ResolveVariables resolves all VSCode workspace variables in a string
// Supports all predefined variables from https://code.visualstudio.com/docs/reference/variables-reference
func (wr *WorkspaceResolver) ResolveVariables(input string) string {
	if input == "" {
		return input
	}

	resolved := input

	// Workspace folder variables
	resolved = strings.ReplaceAll(resolved, "${workspaceFolder}", wr.projectRoot)
	resolved = strings.ReplaceAll(resolved, "${workspaceRoot}", wr.projectRoot) // Legacy alias for workspaceFolder
	resolved = strings.ReplaceAll(resolved, "${workspaceFolderBasename}", filepath.Base(wr.projectRoot))

	// File variables (use project root as fallback since we don't have active file context)
	projectBasename := filepath.Base(wr.projectRoot)
	projectExt := filepath.Ext(projectBasename)
	projectBasenameNoExt := strings.TrimSuffix(projectBasename, projectExt)

	resolved = strings.ReplaceAll(resolved, "${file}", wr.projectRoot)
	resolved = strings.ReplaceAll(resolved, "${fileWorkspaceFolder}", wr.projectRoot)
	resolved = strings.ReplaceAll(resolved, "${relativeFile}", ".")
	resolved = strings.ReplaceAll(resolved, "${relativeFileDirname}", ".")
	resolved = strings.ReplaceAll(resolved, "${fileBasename}", projectBasename)
	resolved = strings.ReplaceAll(resolved, "${fileBasenameNoExtension}", projectBasenameNoExt)
	resolved = strings.ReplaceAll(resolved, "${fileExtname}", projectExt)
	resolved = strings.ReplaceAll(resolved, "${fileDirname}", wr.projectRoot)
	resolved = strings.ReplaceAll(resolved, "${fileDirnameBasename}", projectBasename)

	// Line and selection variables (fallback to empty since we don't have editor context)
	resolved = strings.ReplaceAll(resolved, "${lineNumber}", "1")
	resolved = strings.ReplaceAll(resolved, "${selectedText}", "")

	// Execution context variables
	if cwd, err := os.Getwd(); err == nil {
		resolved = strings.ReplaceAll(resolved, "${cwd}", cwd)
		resolved = strings.ReplaceAll(resolved, "${execPath}", cwd) // Current working directory as exec path
	}

	// Environment variables
	if userHome, err := os.UserHomeDir(); err == nil {
		resolved = strings.ReplaceAll(resolved, "${userHome}", userHome)
	}

	// Path separator
	resolved = strings.ReplaceAll(resolved, "${pathSeparator}", string(filepath.Separator))

	// Configuration and extension variables (provide reasonable defaults)
	resolved = strings.ReplaceAll(resolved, "${config:terminal.external.linuxExec}", "x-terminal-emulator")
	resolved = strings.ReplaceAll(resolved, "${config:terminal.external.osxExec}", "Terminal.app")
	resolved = strings.ReplaceAll(resolved, "${config:terminal.external.windowsExec}", "cmd")

	// Default shell based on OS
	defaultShell := wr.getDefaultShell()
	resolved = strings.ReplaceAll(resolved, "${defaultBuildTask}", "build")

	// Language-specific defaults
	resolved = strings.ReplaceAll(resolved, "${env:PATH}", os.Getenv("PATH"))
	resolved = strings.ReplaceAll(resolved, "${env:HOME}", os.Getenv("HOME"))
	resolved = strings.ReplaceAll(resolved, "${env:SHELL}", defaultShell)

	// Handle ${env:VARIABLE_NAME} pattern for all environment variables
	resolved = wr.resolveEnvironmentVariables(resolved)

	// Handle ${config:...} patterns
	resolved = wr.resolveConfigVariables(resolved)

	// Handle relative paths after variable resolution
	// Only convert to absolute paths if:
	// 1. The result is not already absolute AND
	// 2. We actually resolved variables (input changed) AND
	// 3. The resolved result looks like a standalone path (not an argument with embedded path)
	if !filepath.IsAbs(resolved) && resolved != input {
		// Check if this looks like a standalone path that should be made absolute
		// Only convert if it starts with "." or doesn't contain "=" (to avoid --flag=path scenarios)
		if strings.HasPrefix(resolved, ".") || (strings.Contains(resolved, string(filepath.Separator)) && !strings.Contains(resolved, "=")) {
			resolved = filepath.Join(wr.projectRoot, resolved)
		}
	}

	return resolved
}

// ResolveStringSlice resolves workspace variables in all strings in a slice
func (wr *WorkspaceResolver) ResolveStringSlice(input []string) []string {
	if len(input) == 0 {
		return input
	}

	resolved := make([]string, len(input))
	for i, str := range input {
		resolved[i] = wr.ResolveVariables(str)
	}

	return resolved
}

// ResolveStringMap resolves workspace variables in all values in a string map
func (wr *WorkspaceResolver) ResolveStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return input
	}

	resolved := make(map[string]string, len(input))
	for key, value := range input {
		// Resolve variables in both key and value for completeness
		resolvedKey := wr.ResolveVariables(key)
		resolvedValue := wr.ResolveVariables(value)
		resolved[resolvedKey] = resolvedValue
	}

	return resolved
}

// HasWorkspaceVariables checks if a string contains any VSCode workspace variables
func (wr *WorkspaceResolver) HasWorkspaceVariables(input string) bool {
	return strings.Contains(input, "${workspace") ||
		strings.Contains(input, "${file") ||
		strings.Contains(input, "${userHome}") ||
		strings.Contains(input, "${cwd}") ||
		strings.Contains(input, "${pathSeparator}") ||
		strings.Contains(input, "${env:") ||
		strings.Contains(input, "${config:") ||
		strings.Contains(input, "${command:") ||
		strings.Contains(input, "${input:")
}

// resolveEnvironmentVariables resolves ${env:VARIABLE_NAME} patterns
func (wr *WorkspaceResolver) resolveEnvironmentVariables(input string) string {
	// Pattern: ${env:VARIABLE_NAME}
	result := input

	// Find all ${env:...} patterns
	for {
		start := strings.Index(result, "${env:")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}

		// Extract variable name
		envVar := result[start+6 : start+end] // Skip "${env:"
		envValue := os.Getenv(envVar)

		// Replace the pattern with the environment variable value
		pattern := result[start : start+end+1] // Include the closing "}"
		result = strings.ReplaceAll(result, pattern, envValue)
	}

	return result
}

// getDefaultShell returns the default shell for the current OS
func (wr *WorkspaceResolver) getDefaultShell() string {
	switch runtime.GOOS {
	case "windows":
		// Check for PowerShell first, then cmd
		if _, err := os.Stat("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"); err == nil {
			return "powershell"
		}

		return "cmd"
	case "darwin":
		// macOS defaults to zsh since Big Sur, bash before
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell
		}

		return "/bin/zsh"
	default:
		// Linux and other Unix-like systems
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell
		}

		return "/bin/bash"
	}
}

// resolveConfigVariables resolves ${config:...} patterns
func (wr *WorkspaceResolver) resolveConfigVariables(input string) string {
	result := input

	// Find all ${config:...} patterns
	for {
		start := strings.Index(result, "${config:")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}

		// Extract config path
		configPath := result[start+9 : start+end] // Skip "${config:"
		configValue := wr.ResolveConfigVariable(configPath)

		// Replace the pattern with the config value
		pattern := result[start : start+end+1] // Include the closing "}"
		result = strings.ReplaceAll(result, pattern, configValue)
	}

	return result
}

// ResolveConfigVariable resolves ${config:...} variables with reasonable defaults
func (wr *WorkspaceResolver) ResolveConfigVariable(configPath string) string {
	// Common configuration variables with defaults
	configDefaults := map[string]string{
		"terminal.external.linuxExec":   "x-terminal-emulator",
		"terminal.external.osxExec":     "Terminal.app",
		"terminal.external.windowsExec": "cmd",
		"editor.fontSize":               "14",
		"editor.tabSize":                "4",
		"files.eol":                     "\n",
	}

	if value, exists := configDefaults[configPath]; exists {
		return value
	}

	// Return empty string for unknown config variables
	return ""
}

// ResolvePathVariables resolves variables in paths and always makes relative paths absolute
// This is used for path contexts like cwd, program paths, etc. where relative paths should be relative to workspace
func (wr *WorkspaceResolver) ResolvePathVariables(input string) string {
	resolved := wr.ResolveVariables(input)

	// Always make relative paths absolute for path contexts
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(wr.projectRoot, resolved)
	}

	return resolved
}
