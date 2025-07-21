package converter

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskporter/internal/config"

	"github.com/stretchr/testify/require"
)

func TestJetBrainsToVSCodeLaunchConverter_ConvertToLaunch(t *testing.T) {
	t.Run("Go configuration", func(t *testing.T) {
		// Load JetBrains Go config
		jetbrainsConfig := loadJetBrainsTestData(t, "jetbrains-go.xml")

		// Convert to task
		task := jetbrainsConfigToTask(jetbrainsConfig, "Go")

		// Convert to VSCode launch
		converter := NewJetBrainsToVSCodeLaunchConverter("/test/project", "", false)

		require.True(t, converter.canConvertToLaunch(task))

		launchConfig, err := converter.convertSingleTaskToLaunch(task)
		require.NoError(t, err)

		// Verify Go-specific configuration
		require.Equal(t, "go", launchConfig.Type)
		require.Equal(t, "launch", launchConfig.Request)
		require.Equal(t, "Run Go App", launchConfig.Name)
		// For Go, program can be "." (current directory) or workspace folder
		require.True(t, launchConfig.Program == "." || strings.Contains(launchConfig.Program, "workspaceFolder"),
			"Program should be current directory or workspace folder, got: %s", launchConfig.Program)
		require.Contains(t, launchConfig.Args, "--verbose")
		require.Contains(t, launchConfig.Args, "--output")
		require.Len(t, launchConfig.Env, 2)
		require.Equal(t, "development", launchConfig.Env["GO_ENV"])

		// Verify against golden file for exact output
		verifyVSCodeLaunchConfigGolden(t, launchConfig, "jetbrains_go_to_vscode_expected.json")
	})

	t.Run("Java configuration", func(t *testing.T) {
		// Load JetBrains Java config
		jetbrainsConfig := loadJetBrainsTestData(t, "jetbrains-java.xml")

		// Convert to task
		task := jetbrainsConfigToTask(jetbrainsConfig, "Java")

		// Convert to VSCode launch
		converter := NewJetBrainsToVSCodeLaunchConverter("/test/project", "", false)

		require.True(t, converter.canConvertToLaunch(task))

		launchConfig, err := converter.convertSingleTaskToLaunch(task)
		require.NoError(t, err)

		// Verify Java-specific configuration
		require.Equal(t, "java", launchConfig.Type)
		require.Equal(t, "launch", launchConfig.Request)
		require.Equal(t, "Java Application", launchConfig.Name)
		require.Equal(t, "com.example.Application", launchConfig.MainClass)
		require.Contains(t, launchConfig.Args, "--spring.profiles.active=dev")
		require.Len(t, launchConfig.Env, 2)

		// Verify against golden file for exact output
		verifyVSCodeLaunchConfigGolden(t, launchConfig, "jetbrains_java_to_vscode_expected.json")
	})

	t.Run("Node.js configuration", func(t *testing.T) {
		// Load JetBrains Node.js config
		jetbrainsConfig := loadJetBrainsTestData(t, "jetbrains-nodejs.xml")

		// Convert to task
		task := jetbrainsConfigToTask(jetbrainsConfig, "NodeJS")

		// Convert to VSCode launch
		converter := NewJetBrainsToVSCodeLaunchConverter("/test/project", "", false)

		require.True(t, converter.canConvertToLaunch(task))

		launchConfig, err := converter.convertSingleTaskToLaunch(task)
		require.NoError(t, err)

		// Verify Node.js-specific configuration
		require.Equal(t, "node", launchConfig.Type)
		require.Equal(t, "launch", launchConfig.Request)
		require.Equal(t, "Node.js App", launchConfig.Name)
		require.Contains(t, launchConfig.Program, "src/index.js")
		require.Contains(t, launchConfig.Args, "--env")
		require.Contains(t, launchConfig.Args, "development")
		require.Len(t, launchConfig.Env, 2)

		// Verify against golden file for exact output
		verifyVSCodeLaunchConfigGolden(t, launchConfig, "jetbrains_nodejs_to_vscode_expected.json")
	})

	t.Run("Python configuration", func(t *testing.T) {
		// Load JetBrains Python config
		jetbrainsConfig := loadJetBrainsTestData(t, "jetbrains-python.xml")

		// Convert to task
		task := jetbrainsConfigToTask(jetbrainsConfig, "Python")

		// Convert to VSCode launch
		converter := NewJetBrainsToVSCodeLaunchConverter("/test/project", "", false)

		require.True(t, converter.canConvertToLaunch(task))

		launchConfig, err := converter.convertSingleTaskToLaunch(task)
		require.NoError(t, err)

		// Verify Python-specific configuration
		require.Equal(t, "python", launchConfig.Type)
		require.Equal(t, "launch", launchConfig.Request)
		require.Equal(t, "Python App", launchConfig.Name)
		require.Contains(t, launchConfig.Program, "src/main.py")
		require.Contains(t, launchConfig.Args, "--verbose")
		require.Contains(t, launchConfig.Args, "--config")
		require.Len(t, launchConfig.Env, 2)

		// Verify against golden file for exact output
		verifyVSCodeLaunchConfigGolden(t, launchConfig, "jetbrains_python_to_vscode_expected.json")
	})
}

func TestJetBrainsToVSCodeLaunchConverter_LanguageDetection(t *testing.T) {
	converter := NewJetBrainsToVSCodeLaunchConverter("/test/project", "", false)

	testCases := []struct {
		name         string
		task         *config.Task
		canConvert   bool
		expectedType string
	}{
		{
			name: "Go application by description",
			task: &config.Task{
				Name:        "Go App",
				Description: "GoApplicationRunConfiguration",
				Command:     "go run .",
			},
			canConvert:   true,
			expectedType: "go",
		},
		{
			name: "Java application",
			task: &config.Task{
				Name:    "Java App",
				Command: "java com.example.Main",
			},
			canConvert:   true,
			expectedType: "java",
		},
		{
			name: "Node.js by description",
			task: &config.Task{
				Name:        "Node App",
				Description: "NodeJSConfigurationType",
				Command:     "node index.js",
			},
			canConvert:   true,
			expectedType: "node",
		},
		{
			name: "Python by description",
			task: &config.Task{
				Name:        "Python App",
				Description: "PythonConfigurationType",
				Command:     "python main.py",
			},
			canConvert:   true,
			expectedType: "python",
		},
		{
			name: "Gradle build (not convertible)",
			task: &config.Task{
				Name:    "Gradle Build",
				Command: "gradle build",
			},
			canConvert: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canConvert := converter.canConvertToLaunch(tc.task)
			require.Equal(t, tc.canConvert, canConvert)

			if canConvert {
				launchConfig := &VSCodeLaunchConfig{
					Name:    tc.task.Name,
					Request: "launch",
				}

				err := converter.determineLaunchType(tc.task, launchConfig)
				require.NoError(t, err)
				require.Equal(t, tc.expectedType, launchConfig.Type)
			}
		})
	}
}

func TestJetBrainsToVSCodeLaunchConverter_BidirectionalConsistency(t *testing.T) {
	// Test that converting VSCode → JetBrains → VSCode maintains language consistency
	testCases := []struct {
		name         string
		vscodeFile   string
		originalType string
	}{
		{
			name:         "Go round-trip",
			vscodeFile:   "vscode-launch-go.json",
			originalType: "go",
		},
		{
			name:         "Java round-trip",
			vscodeFile:   "vscode-launch-java.json",
			originalType: "java",
		},
		{
			name:         "Node.js round-trip",
			vscodeFile:   "vscode-launch-nodejs.json",
			originalType: "node",
		},
		{
			name:         "Python round-trip",
			vscodeFile:   "vscode-launch-python.json",
			originalType: "python",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load original VSCode config
			launchFile := loadVSCodeLaunchTestData(t, tc.vscodeFile)
			originalTasks := parseVSCodeLaunchDataToTasks(t, launchFile)
			require.NotEmpty(t, originalTasks)

			// Find launch configs (not attach)
			var launchTasks []*config.Task

			for _, task := range originalTasks {
				if !containsString(task.Description, "attach") {
					launchTasks = append(launchTasks, task)
				}
			}

			require.NotEmpty(t, launchTasks)

			for _, originalTask := range launchTasks {
				// Convert VSCode → JetBrains
				vscodeToJB := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)
				jetbrainsConfig, err := vscodeToJB.convertSingleLaunchConfig(originalTask)
				require.NoError(t, err)

				// Convert JetBrains back to task
				jetbrainsTask := jetbrainsConfigToTask(jetbrainsConfig, tc.originalType)

				// Convert JetBrains → VSCode
				jbToVSCode := NewJetBrainsToVSCodeLaunchConverter("/test/project", "", false)
				require.True(t, jbToVSCode.canConvertToLaunch(jetbrainsTask))

				finalLaunchConfig, err := jbToVSCode.convertSingleTaskToLaunch(jetbrainsTask)
				require.NoError(t, err)

				// Verify language consistency
				require.Equal(t, tc.originalType, finalLaunchConfig.Type,
					"Language type should be consistent through round-trip conversion")

				// Verify against golden file for exact round-trip output
				goldenFileName := tc.originalType + "_roundtrip_expected.json"
				verifyVSCodeLaunchConfigGolden(t, finalLaunchConfig, goldenFileName)
			}
		})
	}
}

// Helper functions

// loadJetBrainsTestData loads JetBrains XML test data
func loadJetBrainsTestData(t *testing.T, filename string) *JetBrainsRunConfiguration {
	t.Helper()

	testDataPath := filepath.Join("testdata", filename)
	data, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file: %s", filename)

	var component JetBrainsComponent

	err = xml.Unmarshal(data, &component)
	require.NoError(t, err, "Failed to parse test data XML: %s", filename)

	return &component.Configuration
}

// jetbrainsConfigToTask converts JetBrains config to internal task representation
func jetbrainsConfigToTask(jbConfig *JetBrainsRunConfiguration, language string) *config.Task {
	task := &config.Task{
		Name:        jbConfig.Name,
		Type:        config.TypeJetBrains,
		Description: jbConfig.Type,
		Env:         make(map[string]string),
	}

	// Extract options and build command/args based on type
	var (
		command string
		args    []string
	)

	switch jbConfig.Type {
	case "GoApplicationRunConfiguration":
		command = "go"
		packagePath := "."

		var programParams []string

		for _, option := range jbConfig.Options {
			switch option.Name {
			case "PACKAGE":
				packagePath = option.Value
			case "PROGRAM_PARAMETERS":
				programParams = parseSpaceSeparatedArgs(option.Value)
			case "WORKING_DIRECTORY":
				task.Cwd = option.Value
			}
		}

		args = append([]string{"run", packagePath}, programParams...)

	case "Application":
		command = "java"

		var (
			mainClass     string
			programParams []string
		)

		for _, option := range jbConfig.Options {
			switch option.Name {
			case "MAIN_CLASS_NAME":
				mainClass = option.Value
			case "PROGRAM_PARAMETERS":
				programParams = parseSpaceSeparatedArgs(option.Value)
			case "WORKING_DIRECTORY":
				task.Cwd = option.Value
			}
		}

		if mainClass != "" {
			args = append([]string{mainClass}, programParams...)
		}

	case "NodeJSConfigurationType":
		command = "node"

		var (
			jsFile    string
			appParams []string
		)

		for _, option := range jbConfig.Options {
			switch option.Name {
			case "PATH_TO_JS_FILE":
				jsFile = option.Value
			case "APPLICATION_PARAMETERS":
				appParams = parseSpaceSeparatedArgs(option.Value)
			case "WORKING_DIRECTORY":
				task.Cwd = option.Value
			}
		}

		if jsFile != "" {
			args = append([]string{jsFile}, appParams...)
		}

	case "PythonConfigurationType":
		command = "python"

		var (
			scriptName string
			params     []string
		)

		for _, option := range jbConfig.Options {
			switch option.Name {
			case "SCRIPT_NAME":
				scriptName = option.Value
			case "PARAMETERS":
				params = parseSpaceSeparatedArgs(option.Value)
			case "WORKING_DIRECTORY":
				task.Cwd = option.Value
			}
		}

		// Check if this is a module execution (dummy script name + -m in parameters)
		if scriptName == "python" && len(params) >= 2 && params[0] == "-m" {
			// This is a Python module execution
			args = params
		} else if scriptName != "" {
			// Regular script execution
			args = append([]string{scriptName}, params...)
		}
	}

	task.Command = command
	task.Args = args

	// Extract environment variables
	if jbConfig.EnvVars != nil {
		for _, envVar := range jbConfig.EnvVars.EnvVars {
			task.Env[envVar.Name] = envVar.Value
		}
	}

	return task
}

// parseSpaceSeparatedArgs parses space-separated argument string
func parseSpaceSeparatedArgs(input string) []string {
	if input == "" {
		return nil
	}

	// Simple space splitting - could be enhanced for quoted args if needed
	args := strings.Fields(input)

	return args
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
