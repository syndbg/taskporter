package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"taskporter/internal/config"

	"github.com/stretchr/testify/require"
)

func TestVSCodeLaunchToJetBrainsConverter_ConvertLaunchConfigs(t *testing.T) {
	t.Run("Go launch configuration", func(t *testing.T) {
		// Load VSCode Go launch config
		launchFile := loadVSCodeLaunchTestData(t, "vscode-launch-go.json")

		// Parse to tasks
		tasks := parseVSCodeLaunchDataToTasks(t, launchFile)
		require.Len(t, tasks, 3)

		// Test only launch configs (not attach)
		launchTask := tasks[0] // "Launch Go Package"
		require.Equal(t, "Launch Go Package", launchTask.Name)
		require.Equal(t, config.TypeVSCodeLaunch, launchTask.Type)

		// Convert to JetBrains
		converter := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)
		jetbrainsConfig, err := converter.convertSingleLaunchConfig(launchTask)
		require.NoError(t, err)

		// Verify Go-specific configuration
		require.Equal(t, "GoApplicationRunConfiguration", jetbrainsConfig.Type)
		require.Equal(t, "Launch Go Package", jetbrainsConfig.Name)

		// Check Go-specific options
		hasPackageOption := false
		hasRunKindOption := false
		hasProgramParams := false

		for _, option := range jetbrainsConfig.Options {
			switch option.Name {
			case "PACKAGE":
				hasPackageOption = true

				require.Equal(t, ".", option.Value)
			case "RUN_KIND":
				hasRunKindOption = true

				require.Equal(t, "PACKAGE", option.Value)
			case "PROGRAM_PARAMETERS":
				hasProgramParams = true

				require.Contains(t, option.Value, "--verbose")
				require.Contains(t, option.Value, "--output")
			}
		}

		require.True(t, hasPackageOption, "Should have PACKAGE option")
		require.True(t, hasRunKindOption, "Should have RUN_KIND option")
		require.True(t, hasProgramParams, "Should have PROGRAM_PARAMETERS option")

		// Verify environment variables
		require.NotNil(t, jetbrainsConfig.EnvVars)
		require.Len(t, jetbrainsConfig.EnvVars.EnvVars, 2)
	})

	t.Run("Java launch configuration", func(t *testing.T) {
		// Load VSCode Java launch config
		launchFile := loadVSCodeLaunchTestData(t, "vscode-launch-java.json")

		// Parse to tasks
		tasks := parseVSCodeLaunchDataToTasks(t, launchFile)
		require.Len(t, tasks, 3)

		// Test first launch config
		launchTask := tasks[0] // "Launch Java App"
		require.Equal(t, "Launch Java App", launchTask.Name)

		// Convert to JetBrains
		converter := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)
		jetbrainsConfig, err := converter.convertSingleLaunchConfig(launchTask)
		require.NoError(t, err)

		// Verify Java-specific configuration
		require.Equal(t, "Application", jetbrainsConfig.Type)
		require.Equal(t, "Launch Java App", jetbrainsConfig.Name)

		// Check Java-specific options
		hasMainClass := false
		hasProgramParams := false

		for _, option := range jetbrainsConfig.Options {
			switch option.Name {
			case "MAIN_CLASS_NAME":
				hasMainClass = true

				require.Equal(t, "com.example.Application", option.Value)
			case "PROGRAM_PARAMETERS":
				hasProgramParams = true

				require.Contains(t, option.Value, "--spring.profiles.active=dev")
			}
		}

		require.True(t, hasMainClass, "Should have MAIN_CLASS_NAME option")
		require.True(t, hasProgramParams, "Should have PROGRAM_PARAMETERS option")
	})

	t.Run("Node.js launch configuration", func(t *testing.T) {
		// Load VSCode Node.js launch config
		launchFile := loadVSCodeLaunchTestData(t, "vscode-launch-nodejs.json")

		// Parse to tasks
		tasks := parseVSCodeLaunchDataToTasks(t, launchFile)
		require.Len(t, tasks, 3)

		// Test only launch configs (not attach)
		launchTask := tasks[0] // "Launch Node.js App"
		require.Equal(t, "Launch Node.js App", launchTask.Name)

		// Convert to JetBrains
		converter := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)
		jetbrainsConfig, err := converter.convertSingleLaunchConfig(launchTask)
		require.NoError(t, err)

		// Verify Node.js-specific configuration
		require.Equal(t, "NodeJSConfigurationType", jetbrainsConfig.Type)
		require.Equal(t, "Launch Node.js App", jetbrainsConfig.Name)

		// Check Node.js-specific options
		hasJSPath := false
		hasAppParams := false

		for _, option := range jetbrainsConfig.Options {
			switch option.Name {
			case "PATH_TO_JS_FILE":
				hasJSPath = true

				require.Contains(t, option.Value, "src/index.js")
			case "APPLICATION_PARAMETERS":
				hasAppParams = true

				require.Contains(t, option.Value, "--env development")
			}
		}

		require.True(t, hasJSPath, "Should have PATH_TO_JS_FILE option")
		require.True(t, hasAppParams, "Should have APPLICATION_PARAMETERS option")
	})

	t.Run("Python launch configuration", func(t *testing.T) {
		// Load VSCode Python launch config
		launchFile := loadVSCodeLaunchTestData(t, "vscode-launch-python.json")

		// Parse to tasks
		tasks := parseVSCodeLaunchDataToTasks(t, launchFile)
		require.Len(t, tasks, 3)

		// Test first launch config
		launchTask := tasks[0] // "Launch Python App"
		require.Equal(t, "Launch Python App", launchTask.Name)

		// Convert to JetBrains
		converter := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)
		jetbrainsConfig, err := converter.convertSingleLaunchConfig(launchTask)
		require.NoError(t, err)

		// Verify Python-specific configuration
		require.Equal(t, "PythonConfigurationType", jetbrainsConfig.Type)
		require.Equal(t, "Launch Python App", jetbrainsConfig.Name)

		// Check Python-specific options
		hasScriptName := false
		hasParams := false

		for _, option := range jetbrainsConfig.Options {
			switch option.Name {
			case "SCRIPT_NAME":
				hasScriptName = true

				require.Contains(t, option.Value, "src/main.py")
			case "PARAMETERS":
				hasParams = true

				require.Contains(t, option.Value, "--verbose")
				require.Contains(t, option.Value, "--config")
			}
		}

		require.True(t, hasScriptName, "Should have SCRIPT_NAME option")
		require.True(t, hasParams, "Should have PARAMETERS option")
	})
}

func TestVSCodeLaunchToJetBrainsConverter_LanguageDetection(t *testing.T) {
	converter := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)

	testCases := []struct {
		name         string
		task         *config.Task
		expectedType string
	}{
		{
			name: "Go launch by description",
			task: &config.Task{
				Name:        "Launch Go App",
				Description: "go launch config",
				Command:     "dlv",
			},
			expectedType: "GoApplicationRunConfiguration",
		},
		{
			name: "Go launch by command",
			task: &config.Task{
				Name:        "Debug Go",
				Description: "debug config",
				Command:     "go",
			},
			expectedType: "GoApplicationRunConfiguration",
		},
		{
			name: "Java by command",
			task: &config.Task{
				Name:        "Java App",
				Description: "java application",
				Command:     "java -cp /path com.example.Main",
			},
			expectedType: "Application",
		},
		{
			name: "Node.js by command",
			task: &config.Task{
				Name:        "Node App",
				Description: "node application",
				Command:     "node",
			},
			expectedType: "NodeJSConfigurationType",
		},
		{
			name: "Python by command",
			task: &config.Task{
				Name:        "Python App",
				Description: "python application",
				Command:     "python",
			},
			expectedType: "PythonConfigurationType",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configType, err := converter.determineJetBrainsConfigType(tc.task)
			require.NoError(t, err)
			require.Equal(t, tc.expectedType, configType)
		})
	}
}

func TestVSCodeLaunchToJetBrainsConverter_ArgumentExtraction(t *testing.T) {
	converter := NewVSCodeLaunchToJetBrainsConverter("/test/project", "", false)

	t.Run("extractGoPackageFromLaunch", func(t *testing.T) {
		testCases := []struct {
			name     string
			task     *config.Task
			expected string
		}{
			{
				name: "package in args after run",
				task: &config.Task{
					Args: []string{"run", ".", "--verbose"},
				},
				expected: ".",
			},
			{
				name: "package path in args",
				task: &config.Task{
					Args: []string{"run", "./cmd/main", "--debug"},
				},
				expected: "./cmd/main",
			},
			{
				name: "no package specified",
				task: &config.Task{
					Args: []string{"--verbose"},
				},
				expected: ".",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := converter.extractGoPackageFromLaunch(tc.task)
				require.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("filterGoArgsFromLaunch", func(t *testing.T) {
		task := &config.Task{
			Args: []string{"run", ".", "--verbose", "--output", "file.txt"},
		}

		result := converter.filterGoArgsFromLaunch(task)
		expected := []string{"--verbose", "--output", "file.txt"}
		require.Equal(t, expected, result)
	})
}

// Helper function to load VSCode launch test data
func loadVSCodeLaunchTestData(t *testing.T, filename string) map[string]interface{} {
	t.Helper()

	testDataPath := filepath.Join("testdata", filename)
	data, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file: %s", filename)

	var launchFile map[string]interface{}

	err = json.Unmarshal(data, &launchFile)
	require.NoError(t, err, "Failed to parse test data JSON: %s", filename)

	return launchFile
}

// parseVSCodeLaunchDataToTasks converts test launch data directly to tasks
func parseVSCodeLaunchDataToTasks(t *testing.T, launchFile map[string]interface{}) []*config.Task {
	t.Helper()

	configurations, ok := launchFile["configurations"].([]interface{})
	require.True(t, ok, "launch file should have configurations array")

	var tasks []*config.Task

	for i, configInterface := range configurations {
		configMap, ok := configInterface.(map[string]interface{})
		require.True(t, ok, "configuration %d should be a map", i)

		// Extract basic properties
		name, _ := configMap["name"].(string)
		launchType, _ := configMap["type"].(string)
		request, _ := configMap["request"].(string)
		program, _ := configMap["program"].(string)
		module, _ := configMap["module"].(string)
		mainClass, _ := configMap["mainClass"].(string)
		cwd, _ := configMap["cwd"].(string)

		// Extract args
		var args []string

		if argsInterface, ok := configMap["args"].([]interface{}); ok {
			for _, arg := range argsInterface {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}

		// Extract environment variables
		env := make(map[string]string)
		if envInterface, ok := configMap["env"].(map[string]interface{}); ok {
			for key, value := range envInterface {
				if valueStr, ok := value.(string); ok {
					env[key] = valueStr
				}
			}
		}

		// Create description with type information for language detection
		description := fmt.Sprintf("%s %s config", launchType, request)

		// Create command based on type and properties
		var command string

		switch launchType {
		case "go":
			command = "go"

			if program != "" {
				args = append([]string{"run", program}, args...)
			} else {
				args = append([]string{"run", "."}, args...)
			}
		case "java":
			command = "java"

			if mainClass != "" {
				args = append([]string{mainClass}, args...)
			}
		case "node":
			command = "node"

			if program != "" {
				args = append([]string{program}, args...)
			}
		case "python":
			command = "python"

			if program != "" {
				args = append([]string{program}, args...)
			} else if module != "" {
				// Handle Python module execution (python -m module)
				args = append([]string{"-m", module}, args...)
			}
		default:
			command = launchType
		}

		task := &config.Task{
			Name:        name,
			Type:        config.TypeVSCodeLaunch,
			Description: description,
			Command:     command,
			Args:        args,
			Cwd:         cwd,
			Env:         env,
		}

		tasks = append(tasks, task)
	}

	return tasks
}
