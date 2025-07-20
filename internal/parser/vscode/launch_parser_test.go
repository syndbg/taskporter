package vscode

import (
	"path/filepath"
	"testing"

	"taskporter/internal/config"

	"github.com/stretchr/testify/require"
)

func TestLaunchParser(t *testing.T) {
	t.Run("NewLaunchParser", func(t *testing.T) {
		parser := NewLaunchParser("/test/project")
		require.NotNil(t, parser)
		require.Equal(t, "/test/project", parser.projectRoot)
	})

	t.Run("ParseLaunchConfigs", func(t *testing.T) {
		t.Run("should parse all launch configs from testdata", func(t *testing.T) {
			// Use our testdata
			testDataPath := filepath.Join("..", "..", "test", "testdata", ".vscode", "launch.json")
			projectRoot := filepath.Join("..", "..", "test", "testdata")

			parser := NewLaunchParser(projectRoot)
			tasks, err := parser.ParseLaunchConfigs(testDataPath)

			require.NoError(t, err)
			require.NotEmpty(t, tasks)

			// Verify we have the expected launch configs from our testdata
			expectedConfigs := map[string]bool{
				"Launch taskporter":     false,
				"Debug taskporter list": false,
				"Debug taskporter run":  false,
				"Debug taskporter port": false,
			}

			for _, task := range tasks {
				require.Equal(t, config.TypeVSCodeLaunch, task.Type)

				if _, expected := expectedConfigs[task.Name]; expected {
					expectedConfigs[task.Name] = true
				}

				// Verify all launch configs have required fields
				require.NotEmpty(t, task.Name, "Launch config name should not be empty")
				require.NotEmpty(t, task.Command, "Launch config %s should have a command", task.Name)
				require.NotEmpty(t, task.Source, "Launch config %s should have a source file", task.Name)
			}

			// Check that expected configs were found (excluding "Attach to Process" as it's not supported yet)
			for configName, found := range expectedConfigs {
				require.True(t, found, "Expected launch config '%s' not found", configName)
			}
		})

		t.Run("should parse specific launch config properties", func(t *testing.T) {
			testDataPath := filepath.Join("..", "..", "test", "testdata", ".vscode", "launch.json")
			projectRoot := filepath.Join("..", "..", "test", "testdata")

			parser := NewLaunchParser(projectRoot)
			tasks, err := parser.ParseLaunchConfigs(testDataPath)

			require.NoError(t, err)

			// Find specific configs and verify their properties
			var launchTaskporter, debugPort *config.Task

			for _, task := range tasks {
				switch task.Name {
				case "Launch taskporter":
					launchTaskporter = task
				case "Debug taskporter port":
					debugPort = task
				}
			}

			t.Run("Launch taskporter properties", func(t *testing.T) {
				require.NotNil(t, launchTaskporter, "Launch taskporter config not found")
				require.Equal(t, "go", launchTaskporter.Command)
				require.Contains(t, launchTaskporter.Args, "run")
				require.Contains(t, launchTaskporter.Args, "list")
				require.Contains(t, launchTaskporter.Args, "--verbose")
				require.Equal(t, "launch", launchTaskporter.Group)
				require.NotNil(t, launchTaskporter.Env)
				require.Equal(t, "true", launchTaskporter.Env["DEBUG"])
			})

			t.Run("Debug taskporter port properties", func(t *testing.T) {
				require.NotNil(t, debugPort, "Debug taskporter port config not found")
				require.Equal(t, "go", debugPort.Command)
				require.Contains(t, debugPort.Args, "run")
				require.Contains(t, debugPort.Args, "port")
				require.Contains(t, debugPort.Args, "--from")
				require.Contains(t, debugPort.Args, "vscode-tasks")
				require.Contains(t, debugPort.Args, "--to")
				require.Contains(t, debugPort.Args, "jetbrains")
				require.Contains(t, debugPort.Args, "--dry-run")
				require.NotNil(t, debugPort.Env)
				require.Equal(t, "1", debugPort.Env["VERBOSE"])
			})
		})
	})

	t.Run("convertLaunchConfig", func(t *testing.T) {
		projectRoot := "/test/project"
		parser := NewLaunchParser(projectRoot)

		t.Run("Go launch configuration", func(t *testing.T) {
			vscodeConfig := VSCodeLaunchConfig{
				Name:    "test-go-launch",
				Type:    "go",
				Request: "launch",
				Mode:    "auto",
				Program: "${workspaceFolder}",
				Args:    []string{"--flag", "value"},
				Env: map[string]string{
					"GO_ENV": "test",
				},
				Cwd: "${workspaceFolder}/subdir",
			}

			task, err := parser.convertLaunchConfig(vscodeConfig, "/test/launch.json")
			require.NoError(t, err)

			require.Equal(t, "test-go-launch", task.Name)
			require.Equal(t, config.TypeVSCodeLaunch, task.Type)
			require.Equal(t, "go", task.Command)
			require.Contains(t, task.Args, "run")
			require.Contains(t, task.Args, projectRoot)
			require.Contains(t, task.Args, "--flag")
			require.Contains(t, task.Args, "value")
			require.Equal(t, "launch", task.Group)
			require.Equal(t, filepath.Join(projectRoot, "subdir"), task.Cwd)
			require.Equal(t, "test", task.Env["GO_ENV"])
		})

		t.Run("Node.js launch configuration", func(t *testing.T) {
			vscodeConfig := VSCodeLaunchConfig{
				Name:    "test-node-launch",
				Type:    "node",
				Request: "launch",
				Program: "${workspaceFolder}/app.js",
				Args:    []string{"--port", "8080"},
				Env: map[string]string{
					"NODE_ENV": "development",
				},
			}

			task, err := parser.convertLaunchConfig(vscodeConfig, "/test/launch.json")
			require.NoError(t, err)

			require.Equal(t, "test-node-launch", task.Name)
			require.Equal(t, "node", task.Command)
			require.Equal(t, filepath.Join(projectRoot, "app.js"), task.Args[0])
			require.Contains(t, task.Args, "--port")
			require.Contains(t, task.Args, "8080")
			require.Equal(t, "development", task.Env["NODE_ENV"])
		})

		t.Run("Python launch configuration", func(t *testing.T) {
			vscodeConfig := VSCodeLaunchConfig{
				Name:    "test-python-launch",
				Type:    "python",
				Request: "launch",
				Program: "${workspaceFolder}/script.py",
				Args:    []string{"--input", "data.csv"},
				Env: map[string]string{
					"PYTHONPATH": "${workspaceFolder}",
				},
			}

			task, err := parser.convertLaunchConfig(vscodeConfig, "/test/launch.json")
			require.NoError(t, err)

			require.Equal(t, "test-python-launch", task.Name)
			require.Equal(t, "python", task.Command)
			require.Equal(t, filepath.Join(projectRoot, "script.py"), task.Args[0])
			require.Contains(t, task.Args, "--input")
			require.Contains(t, task.Args, "data.csv")
			require.Equal(t, projectRoot, task.Env["PYTHONPATH"])
		})

		t.Run("unsupported launch type", func(t *testing.T) {
			vscodeConfig := VSCodeLaunchConfig{
				Name:    "test-unsupported",
				Type:    "cpp",
				Request: "launch",
			}

			task, err := parser.convertLaunchConfig(vscodeConfig, "/test/launch.json")
			require.Error(t, err)
			require.Nil(t, task)
			require.Contains(t, err.Error(), "unsupported launch type: cpp")
		})

		t.Run("attach request type", func(t *testing.T) {
			vscodeConfig := VSCodeLaunchConfig{
				Name:    "test-attach",
				Type:    "go",
				Request: "attach",
			}

			task, err := parser.convertLaunchConfig(vscodeConfig, "/test/launch.json")
			require.Error(t, err)
			require.Nil(t, task)
			require.Contains(t, err.Error(), "attach mode not yet supported")
		})
	})

	t.Run("resolveWorkspacePath", func(t *testing.T) {
		projectRoot := "/home/user/project"
		parser := NewLaunchParser(projectRoot)

		tests := []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "workspaceFolder variable",
				path:     "${workspaceFolder}/src/main.go",
				expected: "/home/user/project/src/main.go",
			},
			{
				name:     "workspaceRoot variable",
				path:     "${workspaceRoot}/app.js",
				expected: "/home/user/project/app.js",
			},
			{
				name:     "relative path",
				path:     "scripts/build.py",
				expected: "/home/user/project/scripts/build.py",
			},
			{
				name:     "absolute path",
				path:     "/usr/bin/python",
				expected: "/usr/bin/python",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := parser.resolveWorkspacePath(tt.path)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("GetPreLaunchTask", func(t *testing.T) {
		testDataPath := filepath.Join("..", "..", "test", "testdata", ".vscode", "launch.json")
		parser := NewLaunchParser("/test")

		t.Run("config with preLaunchTask", func(t *testing.T) {
			preLaunchTask, err := parser.GetPreLaunchTask(testDataPath, "Launch taskporter")
			require.NoError(t, err)
			require.Equal(t, "build", preLaunchTask)
		})

		t.Run("config without preLaunchTask", func(t *testing.T) {
			preLaunchTask, err := parser.GetPreLaunchTask(testDataPath, "Debug taskporter list")
			require.NoError(t, err)
			require.Empty(t, preLaunchTask)
		})

		t.Run("nonexistent config", func(t *testing.T) {
			preLaunchTask, err := parser.GetPreLaunchTask(testDataPath, "Nonexistent Config")
			require.Error(t, err)
			require.Empty(t, preLaunchTask)
			require.Contains(t, err.Error(), "not found")
		})
	})
}
