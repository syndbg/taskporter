package vscode

import (
	"path/filepath"
	"testing"

	"taskporter/internal/config"

	"github.com/stretchr/testify/require"
)

func TestTasksParser(t *testing.T) {
	t.Run("NewTasksParser", func(t *testing.T) {
		parser := NewTasksParser("/test/project")
		require.NotNil(t, parser)
		require.Equal(t, "/test/project", parser.projectRoot)
	})

	t.Run("ParseTasks", func(t *testing.T) {
		t.Run("should parse all tasks from testdata", func(t *testing.T) {
			// Use our testdata
			testDataPath := filepath.Join("..", "..", "test", "testdata", ".vscode", "tasks.json")
			projectRoot := filepath.Join("..", "..", "test", "testdata")

			parser := NewTasksParser(projectRoot)
			tasks, err := parser.ParseTasks(testDataPath)

			require.NoError(t, err)
			require.NotEmpty(t, tasks)

			// Verify we have the expected tasks from our testdata
			expectedTasks := map[string]bool{
				"build":        false,
				"test":         false,
				"lint":         false,
				"clean":        false,
				"run-dev":      false,
				"install-deps": false,
			}

			for _, task := range tasks {
				require.Equal(t, config.TypeVSCodeTask, task.Type)

				if _, expected := expectedTasks[task.Name]; expected {
					expectedTasks[task.Name] = true
				}

				// Verify all tasks have required fields
				require.NotEmpty(t, task.Name, "Task name should not be empty")
				require.NotEmpty(t, task.Command, "Task %s should have a command", task.Name)
				require.NotEmpty(t, task.Source, "Task %s should have a source file", task.Name)
			}

			// Check that all expected tasks were found
			for taskName, found := range expectedTasks {
				require.True(t, found, "Expected task '%s' not found", taskName)
			}
		})

		t.Run("should parse specific task properties correctly", func(t *testing.T) {
			testDataPath := filepath.Join("..", "..", "test", "testdata", ".vscode", "tasks.json")
			projectRoot := filepath.Join("..", "..", "test", "testdata")

			parser := NewTasksParser(projectRoot)
			tasks, err := parser.ParseTasks(testDataPath)

			require.NoError(t, err)

			// Find specific tasks and verify their properties
			var buildTask, runDevTask *config.Task
			for _, task := range tasks {
				switch task.Name {
				case "build":
					buildTask = task
				case "run-dev":
					runDevTask = task
				}
			}

			t.Run("build task properties", func(t *testing.T) {
				require.NotNil(t, buildTask, "Build task not found")
				require.Equal(t, "go", buildTask.Command)
				require.GreaterOrEqual(t, len(buildTask.Args), 3)
				require.Equal(t, "build", buildTask.Group)
			})

			t.Run("run-dev task properties", func(t *testing.T) {
				require.NotNil(t, runDevTask, "run-dev task not found")
				require.Equal(t, "go", runDevTask.Command)
				require.NotNil(t, runDevTask.Env)
				require.Equal(t, "true", runDevTask.Env["DEBUG"])
			})
		})
	})

	t.Run("convertTask", func(t *testing.T) {
		projectRoot := "/test/project"
		parser := NewTasksParser(projectRoot)

		vscodeTask := VSCodeTask{
			Label:   "test-task",
			Type:    "shell",
			Command: "echo",
			Args:    []string{"hello", "world"},
			Detail:  "A test task",
			Group:   "test",
			Options: &VSCodeTaskOptions{
				Cwd: "${workspaceFolder}/subdir",
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
		}

		task, err := parser.convertTask(vscodeTask, "/test/tasks.json")
		require.NoError(t, err)

		t.Run("basic properties", func(t *testing.T) {
			require.Equal(t, "test-task", task.Name)
			require.Equal(t, config.TypeVSCodeTask, task.Type)
			require.Equal(t, "echo", task.Command)
			require.Equal(t, []string{"hello", "world"}, task.Args)
			require.Equal(t, "A test task", task.Description)
			require.Equal(t, "test", task.Group)
			require.Equal(t, "/test/tasks.json", task.Source)
		})

		t.Run("workspace path resolution", func(t *testing.T) {
			expectedCwd := filepath.Join(projectRoot, "subdir")
			require.Equal(t, expectedCwd, task.Cwd)
		})

		t.Run("environment variables", func(t *testing.T) {
			require.NotNil(t, task.Env)
			require.Equal(t, "test_value", task.Env["TEST_VAR"])
		})
	})

	t.Run("parseGroup", func(t *testing.T) {
		parser := NewTasksParser("/test")

		tests := []struct {
			name     string
			group    interface{}
			expected string
		}{
			{
				name:     "nil group",
				group:    nil,
				expected: "",
			},
			{
				name:     "string group",
				group:    "build",
				expected: "build",
			},
			{
				name: "object group",
				group: map[string]interface{}{
					"kind":      "test",
					"isDefault": true,
				},
				expected: "test",
			},
			{
				name: "object group without kind",
				group: map[string]interface{}{
					"isDefault": true,
				},
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := parser.parseGroup(tt.group)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("resolveWorkspacePath", func(t *testing.T) {
		projectRoot := "/home/user/project"
		parser := NewTasksParser(projectRoot)

		tests := []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "workspaceFolder variable",
				path:     "${workspaceFolder}/src",
				expected: "/home/user/project/src",
			},
			{
				name:     "workspaceRoot variable",
				path:     "${workspaceRoot}/build",
				expected: "/home/user/project/build",
			},
			{
				name:     "relative path",
				path:     "relative/path",
				expected: "/home/user/project/relative/path",
			},
			{
				name:     "absolute path",
				path:     "/absolute/path",
				expected: "/absolute/path",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := parser.resolveWorkspacePath(tt.path)
				require.Equal(t, tt.expected, result)
			})
		}
	})
}
