package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Sanitizer provides security sanitization for user inputs and command execution
type Sanitizer struct {
	projectRoot string
}

// NewSanitizer creates a new security sanitizer
func NewSanitizer(projectRoot string) *Sanitizer {
	return &Sanitizer{
		projectRoot: projectRoot,
	}
}

// SanitizeCommand validates and sanitizes a command for safe execution
func (s *Sanitizer) SanitizeCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Block dangerous commands and characters
	dangerousPatterns := []string{
		// Shell injection attempts
		";", "|", "&", "&&", "||", "$(",
		// Redirection operators that could be dangerous
		">>", "<<",
		// Process substitution
		"<(", ">(",
		// Command substitution
		"`",
		// Dangerous commands
		"rm -rf /", "rm -rf /*", ":(){ :|:& };:",
		// Network commands that could be dangerous
		"curl -s", "wget -q",
	}

	commandLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(commandLower, pattern) {
			return fmt.Errorf("potentially dangerous command pattern detected: %s", pattern)
		}
	}

	// Allow only alphanumeric characters, dashes, underscores, dots, and forward slashes for paths
	validCommandPattern := regexp.MustCompile(`^[a-zA-Z0-9\-_./\\]+$`)
	if !validCommandPattern.MatchString(command) {
		return fmt.Errorf("command contains invalid characters: %s", command)
	}

	return nil
}

// SanitizeArgs validates and sanitizes command arguments
func (s *Sanitizer) SanitizeArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	sanitizedArgs := make([]string, 0, len(args))

	for _, arg := range args {
		// Skip empty arguments
		if arg == "" {
			continue
		}

		// Check for dangerous patterns in arguments
		if err := s.validateArgument(arg); err != nil {
			return nil, fmt.Errorf("invalid argument '%s': %w", arg, err)
		}

		sanitizedArgs = append(sanitizedArgs, arg)
	}

	return sanitizedArgs, nil
}

// validateArgument validates a single command argument
func (s *Sanitizer) validateArgument(arg string) error {
	// Block dangerous argument patterns
	dangerousPatterns := []string{
		"--exec", "--evaluate", "--command",
		"-e ", "-c ", "/c ", "/k ",
		"$(", "`", "${",
		"&", "|", ";",
	}

	argLower := strings.ToLower(arg)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(argLower, pattern) {
			return fmt.Errorf("potentially dangerous argument pattern: %s", pattern)
		}
	}

	return nil
}

// SanitizeEnvironment validates and sanitizes environment variables
func (s *Sanitizer) SanitizeEnvironment(env map[string]string) (map[string]string, error) {
	if len(env) == 0 {
		return env, nil
	}

	sanitizedEnv := make(map[string]string)

	for key, value := range env {
		// Validate environment variable key
		if err := s.validateEnvKey(key); err != nil {
			return nil, fmt.Errorf("invalid environment variable key '%s': %w", key, err)
		}

		// Validate environment variable value
		if err := s.validateEnvValue(value); err != nil {
			return nil, fmt.Errorf("invalid environment variable value for '%s': %w", key, err)
		}

		sanitizedEnv[key] = value
	}

	return sanitizedEnv, nil
}

// validateEnvKey validates an environment variable key
func (s *Sanitizer) validateEnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	// Environment variable keys should follow standard naming conventions
	validKeyPattern := regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	if !validKeyPattern.MatchString(key) {
		return fmt.Errorf("environment variable key must contain only uppercase letters, numbers, and underscores")
	}

	// Block dangerous environment variables
	dangerousKeys := []string{
		"PATH", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH", "LD_PRELOAD",
		"SHELL", "IFS", "PS1", "PS2", "PS3", "PS4",
	}

	for _, dangerous := range dangerousKeys {
		if key == dangerous {
			return fmt.Errorf("modifying system environment variable '%s' is not allowed", key)
		}
	}

	return nil
}

// validateEnvValue validates an environment variable value
func (s *Sanitizer) validateEnvValue(value string) error {
	// Check for dangerous patterns in environment values
	dangerousPatterns := []string{
		"$(", "`", "${",
		";", "|", "&",
		"../", "..\\",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(value, pattern) {
			return fmt.Errorf("potentially dangerous pattern in value: %s", pattern)
		}
	}

	return nil
}

// SanitizePath validates and sanitizes file paths to prevent directory traversal
func (s *Sanitizer) SanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("directory traversal detected in path: %s", path)
	}

	// Convert to absolute path if it's relative
	var absPath string
	var err error

	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		// Make relative paths relative to project root
		absPath, err = filepath.Abs(filepath.Join(s.projectRoot, cleanPath))
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path: %w", err)
		}
	}

	// Ensure the path is within the project root (for relative paths)
	if !filepath.IsAbs(path) {
		projectAbs, err := filepath.Abs(s.projectRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve project root: %w", err)
		}

		if !strings.HasPrefix(absPath, projectAbs) {
			return "", fmt.Errorf("path escapes project directory: %s", path)
		}
	}

	return absPath, nil
}

// ValidateTaskName validates a task name for safety
func (s *Sanitizer) ValidateTaskName(taskName string) error {
	if taskName == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	// Task names should be reasonable length
	if len(taskName) > 100 {
		return fmt.Errorf("task name too long (max 100 characters)")
	}

	// Check for dangerous characters in task names
	if strings.ContainsAny(taskName, "<>|&;$`\"'") {
		return fmt.Errorf("task name contains dangerous characters")
	}

	return nil
}

// ValidateConfigPath validates a configuration file path
func (s *Sanitizer) ValidateConfigPath(configPath string) error {
	if configPath == "" {
		return nil // Empty is allowed (means auto-detect)
	}

	// Sanitize the path
	sanitizedPath, err := s.SanitizePath(configPath)
	if err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	// Check if the path exists and is readable
	if _, err := os.Stat(sanitizedPath); err != nil {
		return fmt.Errorf("config path is not accessible: %w", err)
	}

	// Ensure it's a file, not a directory
	fileInfo, err := os.Stat(sanitizedPath)
	if err != nil {
		return fmt.Errorf("cannot access config file: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("config path is a directory, not a file")
	}

	return nil
}

// ValidateOutputPath validates an output path for safety
func (s *Sanitizer) ValidateOutputPath(outputPath string) error {
	if outputPath == "" {
		return nil // Empty is allowed (means auto-detect)
	}

	// Sanitize the path
	_, err := s.SanitizePath(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	return nil
}
