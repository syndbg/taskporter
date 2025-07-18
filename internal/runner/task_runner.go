package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"taskporter/internal/config"
)

// TaskRunner handles execution of tasks
type TaskRunner struct {
	verbose bool
}

// NewTaskRunner creates a new task runner
func NewTaskRunner(verbose bool) *TaskRunner {
	return &TaskRunner{
		verbose: verbose,
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
		fmt.Println("⚡ Starting execution...")
		fmt.Println()
	}

	// Create the command
	cmd := exec.Command(task.Command, task.Args...)

	// Set working directory
	if task.Cwd != "" {
		cmd.Dir = task.Cwd
	}

	// Set up environment variables
	cmd.Env = tr.buildEnvironment(task.Env)

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

// buildEnvironment creates the environment for task execution
func (tr *TaskRunner) buildEnvironment(taskEnv map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Add task-specific environment variables
	for key, value := range taskEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
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
