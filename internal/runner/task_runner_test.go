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

	t.Run("NewTaskRunnerWithOptions", func(t *testing.T) {
		runner := NewTaskRunnerWithOptions(true, "/test/project", true)
		require.NotNil(t, runner)
		require.True(t, runner.verbose)
		require.True(t, runner.paranoidMode)
		require.NotNil(t, runner.sanitizer)
	})

	t.Run("buildEnvironment", func(t *testing.T) {
		t.Run("trust mode", func(t *testing.T) {
			runner := NewTaskRunner(false) // paranoidMode = false by default

			t.Run("with task environment variables", func(t *testing.T) {
				taskEnv := map[string]string{
					"DEBUG":    "true",
					"NODE_ENV": "development",
					"PATH":     "/custom/bin:$PATH", // Should be allowed in trust mode
				}

				env, err := runner.buildEnvironment(taskEnv)
				require.NoError(t, err)
				require.NotEmpty(t, env)

				// Check that our task env vars are present (including PATH)
				var foundDebug, foundNodeEnv, foundPath bool

				for _, envVar := range env {
					if envVar == "DEBUG=true" {
						foundDebug = true
					}

					if envVar == "NODE_ENV=development" {
						foundNodeEnv = true
					}

					if envVar == "PATH=/custom/bin:$PATH" {
						foundPath = true
					}
				}

				require.True(t, foundDebug, "DEBUG environment variable should be set")
				require.True(t, foundNodeEnv, "NODE_ENV environment variable should be set")
				require.True(t, foundPath, "PATH environment variable should be allowed in trust mode")
			})
		})

		t.Run("paranoid mode", func(t *testing.T) {
			runner := NewTaskRunnerWithOptions(false, "/test/project", true)

			t.Run("with valid environment variables", func(t *testing.T) {
				taskEnv := map[string]string{
					"DEBUG":      "true",
					"BUILD_TYPE": "release",
				}

				env, err := runner.buildEnvironment(taskEnv)
				require.NoError(t, err)
				require.NotEmpty(t, env)
			})

			t.Run("with invalid environment variables", func(t *testing.T) {
				taskEnv := map[string]string{
					"PATH": "/malicious/path", // Should be rejected in paranoid mode
				}

				_, err := runner.buildEnvironment(taskEnv)
				require.Error(t, err)
				require.Contains(t, err.Error(), "PATH")
			})
		})
	})

	t.Run("RunTask modes", func(t *testing.T) {
		t.Run("trust mode allows shell operators", func(t *testing.T) {
			runner := NewTaskRunner(false) // paranoidMode = false by default
			task := &config.Task{
				Name:    "complex-build",
				Command: "echo",
				Args:    []string{"build && test"}, // Would be blocked in paranoid mode
				Type:    config.TypeVSCodeTask,
			}

			// This should work in trust mode
			err := runner.RunTask(task)
			require.NoError(t, err)
		})

		t.Run("paranoid mode blocks dangerous patterns", func(t *testing.T) {
			runner := NewTaskRunnerWithOptions(false, ".", true) // paranoidMode = true
			task := &config.Task{
				Name:    "malicious",
				Command: "rm -rf /", // This should be blocked by command validation
				Args:    []string{},
				Type:    config.TypeVSCodeTask,
			}

			// This should be blocked in paranoid mode
			err := runner.RunTask(task)
			require.Error(t, err)
			require.Contains(t, err.Error(), "security validation failed")
		})
	})

	t.Run("RunTask", func(t *testing.T) {
		t.Run("successful command execution", func(t *testing.T) {
			runner := NewTaskRunner(false)
			task := &config.Task{
				Name:    "test-echo",
				Command: "echo",
				Args:    []string{"hello", "world"},
				Type:    config.TypeVSCodeTask,
			}

			err := runner.RunTask(task)
			require.NoError(t, err)
		})

		t.Run("command not found", func(t *testing.T) {
			runner := NewTaskRunner(false)
			task := &config.Task{
				Name:    "test-nonexistent",
				Command: "nonexistent-command-12345",
				Args:    []string{},
				Type:    config.TypeVSCodeTask,
			}

			err := runner.RunTask(task)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failed")
		})

		t.Run("with environment variables", func(t *testing.T) {
			runner := NewTaskRunner(false)
			task := &config.Task{
				Name:    "test-env",
				Command: "sh",
				Args:    []string{"-c", "echo $TEST_VAR"},
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
				Type: config.TypeVSCodeTask,
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
