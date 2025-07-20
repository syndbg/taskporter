package runner

import (
	"testing"

	"taskporter/internal/config"

	"github.com/stretchr/testify/require"
)

func TestTaskRunner(t *testing.T) {
	t.Run("NewTaskRunner", func(t *testing.T) {
		runner := NewTaskRunner(false)
		require.NotNil(t, runner)
		require.False(t, runner.verbose)
	})

	t.Run("NewTaskRunnerWithProjectRoot", func(t *testing.T) {
		runner := NewTaskRunnerWithProjectRoot(true, "/test/project")
		require.NotNil(t, runner)
		require.True(t, runner.verbose)
		require.NotNil(t, runner.sanitizer) // Just check that sanitizer is initialized
	})

	t.Run("buildEnvironment", func(t *testing.T) {
		runner := NewTaskRunner(false)

		t.Run("with no task environment", func(t *testing.T) {
			env, err := runner.buildEnvironment(nil)
			require.NoError(t, err)
			require.NotEmpty(t, env) // Should have system environment
		})

		t.Run("with task environment variables", func(t *testing.T) {
			taskEnv := map[string]string{
				"DEBUG":    "true",
				"NODE_ENV": "development",
			}

			env, err := runner.buildEnvironment(taskEnv)
			require.NoError(t, err)
			require.NotEmpty(t, env)

			// Check that our task env vars are present
			var foundDebug, foundNodeEnv bool
			for _, envVar := range env {
				if envVar == "DEBUG=true" {
					foundDebug = true
				}
				if envVar == "NODE_ENV=development" {
					foundNodeEnv = true
				}
			}

			require.True(t, foundDebug, "DEBUG environment variable should be set")
			require.True(t, foundNodeEnv, "NODE_ENV environment variable should be set")
		})

		t.Run("with invalid environment variables", func(t *testing.T) {
			taskEnv := map[string]string{
				"PATH": "/malicious/path", // Should be rejected by security validation
			}

			_, err := runner.buildEnvironment(taskEnv)
			require.Error(t, err)
			require.Contains(t, err.Error(), "PATH")
		})
	})

	t.Run("validateTaskSecurity", func(t *testing.T) {
		runner := NewTaskRunner(false)

		t.Run("valid task should pass", func(t *testing.T) {
			task := &config.Task{
				Name:    "build",
				Command: "go",
				Args:    []string{"build", "-o", "bin/app"},
				Cwd:     ".",
				Env: map[string]string{
					"CGO_ENABLED": "0",
				},
			}

			err := runner.validateTaskSecurity(task)
			require.NoError(t, err)
		})

		t.Run("task with dangerous command should be rejected", func(t *testing.T) {
			task := &config.Task{
				Name:    "malicious",
				Command: "rm -rf /",
				Args:    []string{},
			}

			err := runner.validateTaskSecurity(task)
			require.Error(t, err)
			require.Contains(t, err.Error(), "dangerous")
		})

		t.Run("task with dangerous arguments should be rejected", func(t *testing.T) {
			task := &config.Task{
				Name:    "test",
				Command: "echo",
				Args:    []string{"$(whoami)"},
			}

			err := runner.validateTaskSecurity(task)
			require.Error(t, err)
		})

		t.Run("task with invalid environment should be rejected", func(t *testing.T) {
			task := &config.Task{
				Name:    "test",
				Command: "echo",
				Args:    []string{"hello"},
				Env: map[string]string{
					"PATH": "/malicious",
				},
			}

			err := runner.validateTaskSecurity(task)
			require.Error(t, err)
		})
	})

	t.Run("RunTask", func(t *testing.T) {
		t.Run("successful command execution", func(t *testing.T) {
			runner := NewTaskRunner(false)

			task := &config.Task{
				Name:    "echo-test",
				Type:    config.TypeVSCodeTask,
				Command: "echo",
				Args:    []string{"hello", "world"},
				Cwd:     "",
			}

			err := runner.RunTask(task)
			require.NoError(t, err)
		})

		t.Run("command not found", func(t *testing.T) {
			runner := NewTaskRunner(false)

			task := &config.Task{
				Name:    "nonexistent-command",
				Type:    config.TypeVSCodeTask,
				Command: "this-command-does-not-exist-12345",
				Args:    []string{},
				Cwd:     "",
			}

			err := runner.RunTask(task)
			require.Error(t, err)
			require.Contains(t, err.Error(), "task 'nonexistent-command' failed")
		})

		t.Run("with environment variables", func(t *testing.T) {
			runner := NewTaskRunner(false)

			// Use a command that will show environment variables
			task := &config.Task{
				Name:    "env-test",
				Type:    config.TypeVSCodeTask,
				Command: "sh",
				Args:    []string{"-c", "echo $TEST_VAR"},
				Cwd:     "",
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			}

			err := runner.RunTask(task)
			require.NoError(t, err)
		})
	})
}

func TestTaskFinder(t *testing.T) {
	t.Run("NewTaskFinder", func(t *testing.T) {
		finder := NewTaskFinder()
		require.NotNil(t, finder)
	})

	t.Run("FindTask", func(t *testing.T) {
		tasks := []*config.Task{
			{
				Name:    "build",
				Type:    config.TypeVSCodeTask,
				Command: "go",
				Args:    []string{"build"},
			},
			{
				Name:    "test",
				Type:    config.TypeVSCodeTask,
				Command: "go",
				Args:    []string{"test"},
			},
			{
				Name:    "build-docker",
				Type:    config.TypeVSCodeTask,
				Command: "docker",
				Args:    []string{"build"},
			},
		}

		finder := NewTaskFinder()

		t.Run("exact match", func(t *testing.T) {
			task, err := finder.FindTask("build", tasks)
			require.NoError(t, err)
			require.NotNil(t, task)
			require.Equal(t, "build", task.Name)
		})

		t.Run("case insensitive match", func(t *testing.T) {
			task, err := finder.FindTask("BUILD", tasks)
			require.NoError(t, err)
			require.NotNil(t, task)
			require.Equal(t, "build", task.Name)
		})

		t.Run("partial match - unique", func(t *testing.T) {
			task, err := finder.FindTask("test", tasks)
			require.NoError(t, err)
			require.NotNil(t, task)
			require.Equal(t, "test", task.Name)
		})

		t.Run("partial match - multiple matches", func(t *testing.T) {
			task, err := finder.FindTask("build", tasks)
			require.NoError(t, err)
			require.NotNil(t, task)
			// Should return exact match "build", not partial match "build-docker"
			require.Equal(t, "build", task.Name)
		})

		t.Run("partial match - ambiguous", func(t *testing.T) {
			// If we search for something that matches multiple tasks partially
			task, err := finder.FindTask("buil", tasks)
			require.Error(t, err)
			require.Nil(t, task)
			require.Contains(t, err.Error(), "multiple tasks match")
		})

		t.Run("no match", func(t *testing.T) {
			task, err := finder.FindTask("nonexistent", tasks)
			require.Error(t, err)
			require.Nil(t, task)
			require.Contains(t, err.Error(), "task 'nonexistent' not found")
		})

		t.Run("empty task list", func(t *testing.T) {
			task, err := finder.FindTask("build", []*config.Task{})
			require.Error(t, err)
			require.Nil(t, task)
			require.Contains(t, err.Error(), "task 'build' not found")
		})
	})
}
