package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/syndbg/taskporter/internal/config"
	"github.com/syndbg/taskporter/internal/security"
)

// TaskRunner handles execution of tasks
type TaskRunner struct {
	verbose      bool
	paranoidMode bool
	sanitizer    *security.Sanitizer
}

// NewTaskRunner creates a new task runner
func NewTaskRunner(verbose bool) *TaskRunner {
	return &TaskRunner{
		verbose:      verbose,
		paranoidMode: false,                      // Default: trust user configurations
		sanitizer:    security.NewSanitizer("."), // Will be updated with proper project root
	}
}

// NewTaskRunnerWithProjectRoot creates a new task runner with a specific project root
func NewTaskRunnerWithProjectRoot(verbose bool, projectRoot string) *TaskRunner {
	return &TaskRunner{
		verbose:      verbose,
		paranoidMode: false, // Default: trust user configurations
		sanitizer:    security.NewSanitizer(projectRoot),
	}
}

// NewTaskRunnerWithOptions creates a new task runner with all options
func NewTaskRunnerWithOptions(verbose bool, projectRoot string, paranoidMode bool) *TaskRunner {
	return &TaskRunner{
		verbose:      verbose,
		paranoidMode: paranoidMode,
		sanitizer:    security.NewSanitizer(projectRoot),
	}
}

// RunTask executes a given task with proper environment and working directory setup
func (tr *TaskRunner) RunTask(task *config.Task) error {
	if tr.verbose {
		fmt.Printf("🚀 Executing task: %s\n", task.Name)
		fmt.Printf("📋 Type: %s\n", task.Type)
		fmt.Printf("💻 Command: %s %v\n", task.Command, task.Args)
		fmt.Printf("📁 Working directory: %s\n", task.Cwd)

		if len(task.Env) > 0 {
			fmt.Printf("🌐 Environment variables: %v\n", task.Env)
		}

		if tr.paranoidMode {
			fmt.Printf("🛡️ Paranoid mode: Performing security validation...\n")
		} else {
			fmt.Printf("🤝 Trust mode: Executing user configuration as-is (like IDEs)\n")
		}

		fmt.Println("⚡ Starting execution...")
		fmt.Println()
	}

	// Security validation (only in paranoid mode)
	if tr.paranoidMode {
		if err := tr.validateTaskSecurity(task); err != nil {
			return fmt.Errorf("security validation failed for task '%s': %w", task.Name, err)
		}

		if tr.verbose {
			fmt.Printf("✅ Security validation passed\n")
		}
	}

	// Create the command with optional sanitization
	var args []string

	var err error

	if tr.paranoidMode {
		args, err = tr.sanitizer.SanitizeArgs(task.Args)
		if err != nil {
			return fmt.Errorf("failed to sanitize arguments for task '%s': %w", task.Name, err)
		}
	} else {
		args = task.Args // Use original arguments as-is
	}

	cmd := exec.Command(task.Command, args...)

	// Set working directory (with optional validation)
	if task.Cwd != "" {
		if tr.paranoidMode {
			sanitizedCwd, err := tr.sanitizer.SanitizePath(task.Cwd)
			if err != nil {
				return fmt.Errorf("failed to sanitize working directory for task '%s': %w", task.Name, err)
			}

			cmd.Dir = sanitizedCwd
		} else {
			cmd.Dir = task.Cwd // Use original path as-is
		}
	}

	// Set up environment variables (with optional validation)
	env, err := tr.buildEnvironment(task.Env)
	if err != nil {
		return fmt.Errorf("failed to build environment for task '%s': %w", task.Name, err)
	}

	cmd.Env = env

	// Set up input/output
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("task '%s' failed: %w", task.Name, err)
	}

	if tr.verbose {
		fmt.Println()
		fmt.Printf("✅ Task '%s' completed successfully\n", task.Name)
		fmt.Println("📡 Strand connection maintained... delivery complete!")
	}

	return nil
}

// validateTaskSecurity performs comprehensive security validation on a task (paranoid mode only)
func (tr *TaskRunner) validateTaskSecurity(task *config.Task) error {
	// Validate task name
	if err := tr.sanitizer.ValidateTaskName(task.Name); err != nil {
		return fmt.Errorf("invalid task name: %w", err)
	}

	// Validate command
	if err := tr.sanitizer.SanitizeCommand(task.Command); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	// Validate arguments
	if _, err := tr.sanitizer.SanitizeArgs(task.Args); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate working directory
	if task.Cwd != "" {
		if _, err := tr.sanitizer.SanitizePath(task.Cwd); err != nil {
			return fmt.Errorf("invalid working directory: %w", err)
		}
	}

	// Validate environment variables
	if _, err := tr.sanitizer.SanitizeEnvironment(task.Env); err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}

	return nil
}

// buildEnvironment creates the environment for task execution with optional security validation
func (tr *TaskRunner) buildEnvironment(taskEnv map[string]string) ([]string, error) {
	// Start with current environment
	env := os.Environ()

	// Handle task-specific environment variables
	if len(taskEnv) > 0 {
		if tr.paranoidMode {
			// Validate and sanitize in paranoid mode
			sanitizedEnv, err := tr.sanitizer.SanitizeEnvironment(taskEnv)
			if err != nil {
				return nil, fmt.Errorf("failed to sanitize environment variables: %w", err)
			}

			// Add sanitized task-specific environment variables
			for key, value := range sanitizedEnv {
				env = append(env, fmt.Sprintf("%s=%s", key, value))
			}
		} else {
			// Use original environment variables as-is (trust mode)
			for key, value := range taskEnv {
				env = append(env, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}

	return env, nil
}

// TaskFinder helps find tasks by name from a list
type TaskFinder struct{}

// NewTaskFinder creates a new task finder
func NewTaskFinder() *TaskFinder {
	return &TaskFinder{}
}

// FindTask searches for a task by name in the given list
func (tf *TaskFinder) FindTask(taskName string, tasks []*config.Task) (*config.Task, error) {
	// Exact match first
	for _, task := range tasks {
		if task.Name == taskName {
			return task, nil
		}
	}

	// Case-insensitive match
	taskNameLower := strings.ToLower(taskName)

	for _, task := range tasks {
		if strings.ToLower(task.Name) == taskNameLower {
			return task, nil
		}
	}

	// Partial match (if unique)
	var matches []*config.Task

	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Name), taskNameLower) {
			matches = append(matches, task)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		var names []string
		for _, match := range matches {
			names = append(names, match.Name)
		}

		return nil, fmt.Errorf("multiple tasks match '%s': %s", taskName, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("task '%s' not found", taskName)
}
