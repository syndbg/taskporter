package jetbrains

import (
	"path/filepath"
	"testing"

	"taskporter/internal/config"

	"github.com/stretchr/testify/require"
)

func TestRunConfigurationParser(t *testing.T) {
	t.Run("NewRunConfigurationParser", func(t *testing.T) {
		parser := NewRunConfigurationParser("/test/project")
		require.NotNil(t, parser)
		require.Equal(t, "/test/project", parser.projectRoot)
	})

	t.Run("ParseRunConfiguration", func(t *testing.T) {
		t.Run("should parse Application configuration from testdata", func(t *testing.T) {
			testDataPath := filepath.Join("..", "..", "..", "test", "jetbrains-testdata", ".idea", "runConfigurations", "Application.xml")
			projectRoot := filepath.Join("..", "..", "..", "test", "jetbrains-testdata")

			parser := NewRunConfigurationParser(projectRoot)
			task, err := parser.ParseRunConfiguration(testDataPath)

			require.NoError(t, err)
			require.NotNil(t, task)

			require.Equal(t, "Application", task.Name)
			require.Equal(t, config.TypeJetBrains, task.Type)
			require.Equal(t, "java", task.Command)
			require.Equal(t, "run", task.Group)
			require.Contains(t, task.Args, "com.example.Main")
			require.Contains(t, task.Args, "-Xmx1024m")
			require.Contains(t, task.Args, "--port")
			require.Contains(t, task.Args, "8080")
			require.Equal(t, testDataPath, task.Source)
			require.NotNil(t, task.Env)
			require.Equal(t, "true", task.Env["DEBUG"])
		})

		t.Run("should parse Gradle configuration from testdata", func(t *testing.T) {
			testDataPath := filepath.Join("..", "..", "..", "test", "jetbrains-testdata", ".idea", "runConfigurations", "Gradle_Build.xml")
			projectRoot := filepath.Join("..", "..", "..", "test", "jetbrains-testdata")

			parser := NewRunConfigurationParser(projectRoot)
			task, err := parser.ParseRunConfiguration(testDataPath)

			require.NoError(t, err)
			require.NotNil(t, task)

			require.Equal(t, "Gradle Build", task.Name)
			require.Equal(t, config.TypeJetBrains, task.Type)
			require.Equal(t, "gradle", task.Command)
			require.Equal(t, "build", task.Group)
			require.Contains(t, task.Args, "build")
			require.Equal(t, testDataPath, task.Source)
		})
	})

	t.Run("handleApplicationConfig", func(t *testing.T) {
		projectRoot := "/test/project"
		parser := NewRunConfigurationParser(projectRoot)

		t.Run("should handle basic Application configuration", func(t *testing.T) {
			jetbrainsConfig := JetBrainsRunConfiguration{
				Name: "Test App",
				Type: "Application",
				Options: []JetBrainsOption{
					{Name: "MAIN_CLASS_NAME", Value: "com.test.Main"},
					{Name: "VM_PARAMETERS", Value: "-Xmx512m"},
					{Name: "PROGRAM_PARAMETERS", Value: "--config test.properties"},
					{Name: "WORKING_DIRECTORY", Value: "$PROJECT_DIR$/subdir"},
				},
			}

			task := &config.Task{}
			err := parser.handleApplicationConfig(jetbrainsConfig, task)

			require.NoError(t, err)
			require.Equal(t, "java", task.Command)
			require.Equal(t, "run", task.Group)
			require.Contains(t, task.Args, "-Xmx512m")
			require.Contains(t, task.Args, "com.test.Main")
			require.Contains(t, task.Args, "--config")
			require.Contains(t, task.Args, "test.properties")
			require.Equal(t, filepath.Join(projectRoot, "subdir"), task.Cwd)
		})

		t.Run("should handle Application configuration with environment variables", func(t *testing.T) {
			jetbrainsConfig := JetBrainsRunConfiguration{
				Name: "Test App",
				Type: "Application",
				Options: []JetBrainsOption{
					{Name: "MAIN_CLASS_NAME", Value: "com.test.Main"},
					{
						Name: "ENV_VARIABLES",
						Map: &JetBrainsMap{
							Entries: []JetBrainsEntry{
								{Key: "DEBUG", Value: "true"},
								{Key: "ENV", Value: "test"},
							},
						},
					},
				},
			}

			task := &config.Task{}
			err := parser.handleApplicationConfig(jetbrainsConfig, task)

			require.NoError(t, err)
			require.NotNil(t, task.Env)
			require.Equal(t, "true", task.Env["DEBUG"])
			require.Equal(t, "test", task.Env["ENV"])
		})

		t.Run("should fail without MAIN_CLASS_NAME", func(t *testing.T) {
			jetbrainsConfig := JetBrainsRunConfiguration{
				Name: "Test App",
				Type: "Application",
				Options: []JetBrainsOption{
					{Name: "VM_PARAMETERS", Value: "-Xmx512m"},
				},
			}

			task := &config.Task{}
			err := parser.handleApplicationConfig(jetbrainsConfig, task)

			require.Error(t, err)
			require.Contains(t, err.Error(), "MAIN_CLASS_NAME is required")
		})
	})

	t.Run("handleGradleConfig", func(t *testing.T) {
		projectRoot := "/test/project"
		parser := NewRunConfigurationParser(projectRoot)

		t.Run("should handle basic Gradle configuration", func(t *testing.T) {
			jetbrainsConfig := JetBrainsRunConfiguration{
				Name: "Gradle Build",
				Type: "GradleRunConfiguration",
				ExternalSystemSettings: &JetBrainsExternalSystemSettings{
					Options: []JetBrainsOption{
						{
							Name: "taskNames",
							List: &JetBrainsList{
								Options: []JetBrainsListOption{
									{Value: "clean"},
									{Value: "build"},
								},
							},
						},
						{Name: "scriptParameters", Value: "--info --stacktrace"},
					},
				},
			}

			task := &config.Task{}
			err := parser.handleGradleConfig(jetbrainsConfig, task)

			require.NoError(t, err)
			require.Equal(t, "gradle", task.Command)
			require.Equal(t, "build", task.Group)
			require.Contains(t, task.Args, "clean")
			require.Contains(t, task.Args, "build")
			require.Contains(t, task.Args, "--info")
			require.Contains(t, task.Args, "--stacktrace")
		})
	})

	t.Run("parseParameters", func(t *testing.T) {
		parser := NewRunConfigurationParser("/test")

		tests := []struct {
			name     string
			input    string
			expected []string
		}{
			{
				name:     "simple parameters",
				input:    "--port 8080 --debug",
				expected: []string{"--port", "8080", "--debug"},
			},
			{
				name:     "quoted parameters",
				input:    `--config "test file.properties" --name 'My App'`,
				expected: []string{"--config", "test file.properties", "--name", "My App"},
			},
			{
				name:     "mixed parameters",
				input:    `-Xmx512m -Dprop="quoted value" --flag`,
				expected: []string{"-Xmx512m", "-Dprop=quoted value", "--flag"},
			},
			{
				name:     "empty string",
				input:    "",
				expected: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := parser.parseParameters(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("resolveJetBrainsPath", func(t *testing.T) {
		projectRoot := "/home/user/project"
		parser := NewRunConfigurationParser(projectRoot)

		tests := []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "PROJECT_DIR variable",
				path:     "$PROJECT_DIR$/src/main/java",
				expected: "/home/user/project/src/main/java",
			},
			{
				name:     "MODULE_DIR variable",
				path:     "$MODULE_DIR$/target",
				expected: "/home/user/project/target",
			},
			{
				name:     "relative path",
				path:     "build/libs",
				expected: "/home/user/project/build/libs",
			},
			{
				name:     "absolute path",
				path:     "/usr/local/bin",
				expected: "/usr/local/bin",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := parser.resolveJetBrainsPath(tt.path)
				require.Equal(t, tt.expected, result)
			})
		}
	})
}
