package converter

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/syndbg/taskporter/internal/config"
	"github.com/syndbg/taskporter/internal/parser/vscode"

	"github.com/stretchr/testify/require"
)

func TestVSCodeToJetBrainsConverter(t *testing.T) {
	// Create temporary directory for test outputs
	tempDir := t.TempDir()

	t.Run("NewVSCodeToJetBrainsConverter", func(t *testing.T) {
		converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, true)

		require.NotNil(t, converter)
		require.Equal(t, "/test/project", converter.projectRoot)
		require.Equal(t, tempDir, converter.outputPath)
		require.True(t, converter.verbose)
	})

	t.Run("ConvertTasks", func(t *testing.T) {
		t.Run("should convert Java tasks correctly", func(t *testing.T) {
			tasks := loadTestTasks(t, "java-tasks.json")
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)

			// Check that XML files were created
			compileFile := filepath.Join(tempDir, "compile-java.xml")
			runFile := filepath.Join(tempDir, "run-java-app.xml")

			require.FileExists(t, compileFile)
			require.FileExists(t, runFile)

			// Validate XML content
			validateJavaCompileXML(t, compileFile) // javac command -> ShellScript
			validateJavaRunXML(t, runFile)         // java command -> Application
		})

		t.Run("should convert Gradle tasks correctly", func(t *testing.T) {
			tasks := loadTestTasks(t, "gradle-tasks.json")
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)

			// Check that XML files were created
			buildFile := filepath.Join(tempDir, "gradle-build.xml")
			testFile := filepath.Join(tempDir, "gradle-test.xml")

			require.FileExists(t, buildFile)
			require.FileExists(t, testFile)

			// Validate that it's recognized as Gradle task
			validateGradleXML(t, buildFile, "build --info")
		})

		t.Run("should convert Node.js tasks correctly", func(t *testing.T) {
			tasks := loadTestTasks(t, "nodejs-tasks.json")
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)

			// Check generated files
			npmInstallFile := filepath.Join(tempDir, "npm-install.xml")
			nodeServerFile := filepath.Join(tempDir, "node-server.xml")

			require.FileExists(t, npmInstallFile)
			require.FileExists(t, nodeServerFile)

			// Validate Node.js specific configuration (node-server task uses 'node' command)
			validateNodeJSXML(t, nodeServerFile)
		})

		t.Run("should convert Python tasks correctly", func(t *testing.T) {
			tasks := loadTestTasks(t, "python-tasks.json")
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)

			// Check generated files
			pythonRunFile := filepath.Join(tempDir, "python-run.xml")
			pytestFile := filepath.Join(tempDir, "pytest.xml")

			require.FileExists(t, pythonRunFile)
			require.FileExists(t, pytestFile)

			// Validate Python specific configuration
			validatePythonXML(t, pythonRunFile)
		})

		t.Run("should convert Maven tasks correctly", func(t *testing.T) {
			tasks := loadTestTasks(t, "maven-tasks.json")
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)

			// Check generated files
			mavenCompileFile := filepath.Join(tempDir, "maven-compile.xml")

			require.FileExists(t, mavenCompileFile)

			// Validate Maven specific configuration
			validateMavenXML(t, mavenCompileFile, "compile")
		})

		t.Run("should handle dry run mode", func(t *testing.T) {
			tasks := loadTestTasks(t, "java-tasks.json")

			// Use a different temp directory for dry run to avoid conflicts
			dryRunDir := filepath.Join(tempDir, "dry-run")
			converter := NewVSCodeToJetBrainsConverter("/test/project", dryRunDir, true)

			err := converter.ConvertTasks(tasks, true)
			require.NoError(t, err)

			// Check that NO XML files were created in dry run
			compileFile := filepath.Join(dryRunDir, "compile-java.xml")
			require.NoFileExists(t, compileFile)
		})

		t.Run("should handle empty task list", func(t *testing.T) {
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks([]*config.Task{}, false)
			require.NoError(t, err)
		})

		t.Run("should handle edge cases", func(t *testing.T) {
			tasks := loadTestTasks(t, "edge-cases.json")
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)

			// Check that files were created for all tasks (!@# are not invalid filename chars)
			specialCharsFile := filepath.Join(tempDir, "task-with-special-chars!@#.xml")
			emptyCommandFile := filepath.Join(tempDir, "empty-command.xml")
			noArgsFile := filepath.Join(tempDir, "no-args-task.xml")
			complexGradleFile := filepath.Join(tempDir, "complex-gradle-task.xml")

			require.FileExists(t, specialCharsFile)
			require.FileExists(t, emptyCommandFile)
			require.FileExists(t, noArgsFile)
			require.FileExists(t, complexGradleFile)

			// Validate special character handling
			validateEdgeCaseXML(t, specialCharsFile, "task-with-special-chars!@#")

			// Validate complex Gradle task
			validateComplexGradleXML(t, complexGradleFile)
		})

		t.Run("should handle nil tasks gracefully", func(t *testing.T) {
			converter := NewVSCodeToJetBrainsConverter("/test/project", tempDir, false)

			// This should not panic
			err := converter.ConvertTasks(nil, false)
			require.NoError(t, err)
		})

		t.Run("should handle invalid output directory", func(t *testing.T) {
			// Try to write to a non-existent directory without creating it
			invalidDir := filepath.Join(tempDir, "nonexistent", "deeply", "nested")
			converter := NewVSCodeToJetBrainsConverter("/test/project", invalidDir, false)

			tasks := loadTestTasks(t, "java-tasks.json")

			// This should still work because ConvertTasks creates the directory
			err := converter.ConvertTasks(tasks, false)
			require.NoError(t, err)
		})
	})

	t.Run("convertVSCodeVariables", func(t *testing.T) {
		converter := NewVSCodeToJetBrainsConverter("/test/project", "", false)

		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "workspace folder",
				input:    "${workspaceFolder}/src",
				expected: "$PROJECT_DIR$/src",
			},
			{
				name:     "workspace root",
				input:    "${workspaceRoot}/build",
				expected: "$PROJECT_DIR$/build",
			},
			{
				name:     "file path",
				input:    "${file}",
				expected: "$FilePath$",
			},
			{
				name:     "multiple variables",
				input:    "${workspaceFolder}/src/${fileBasename}",
				expected: "$PROJECT_DIR$/src/$FileName$",
			},
			{
				name:     "no variables",
				input:    "/absolute/path",
				expected: "/absolute/path",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := converter.convertVSCodeVariables(tc.input)
				require.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("determineConfigType", func(t *testing.T) {
		converter := NewVSCodeToJetBrainsConverter("/test/project", "", false)

		testCases := []struct {
			name     string
			command  string
			expected string
		}{
			{
				name:     "Java application",
				command:  "java",
				expected: "Application",
			},
			{
				name:     "Gradle build",
				command:  "gradle",
				expected: "GradleRunTask",
			},
			{
				name:     "Maven build",
				command:  "mvn",
				expected: "MavenRunConfiguration",
			},
			{
				name:     "Node.js app",
				command:  "node",
				expected: "NodeJS",
			},
			{
				name:     "Python script",
				command:  "python",
				expected: "PythonConfigurationType",
			},
			{
				name:     "Shell script",
				command:  "sh",
				expected: "ShellScript",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				task := &config.Task{Command: tc.command}
				result := converter.determineConfigType(task)
				require.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("sanitizeFilename", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "normal name",
				input:    "build-task",
				expected: "build-task",
			},
			{
				name:     "spaces to underscores",
				input:    "compile java app",
				expected: "compile_java_app",
			},
			{
				name:     "invalid characters",
				input:    "task/with:invalid*chars",
				expected: "task_with_invalid_chars",
			},
			{
				name:     "complex filename",
				input:    "Test: Run All <Tests>",
				expected: "Test__Run_All__Tests_",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := sanitizeFilename(tc.input)
				require.Equal(t, tc.expected, result)
			})
		}
	})
}

// Helper functions for loading test data and validation

func loadTestTasks(t *testing.T, filename string) []*config.Task {
	testdataPath := filepath.Join("testdata", filename)
	parser := vscode.NewTasksParser("/test/project")

	tasks, err := parser.ParseTasks(testdataPath)
	require.NoError(t, err)
	require.NotEmpty(t, tasks)

	return tasks
}

func validateJavaCompileXML(t *testing.T, filename string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "compile-java", config.Name)
	require.Equal(t, "ShellScript", config.Type) // javac command should be ShellScript, not Application

	// Check for script text option
	scriptOption := findOption(config.Options, "SCRIPT_TEXT")
	require.NotNil(t, scriptOption)
	require.Contains(t, scriptOption.Value, "javac")
	require.Contains(t, scriptOption.Value, "Main.java")

	// Check working directory (VSCode parser resolves ${workspaceFolder} to absolute path)
	workingDirOption := findOption(config.Options, "WORKING_DIRECTORY")
	require.NotNil(t, workingDirOption)
	require.Equal(t, "/test/project", workingDirOption.Value)

	// Check environment variables
	require.NotNil(t, config.EnvVars)
	require.Len(t, config.EnvVars.EnvVars, 2) // JAVA_HOME and DEBUG

	javaHomeEnv := findEnvVar(config.EnvVars.EnvVars, "JAVA_HOME")
	require.NotNil(t, javaHomeEnv)
	require.Equal(t, "/usr/lib/jvm/java-11-openjdk", javaHomeEnv.Value)
}

func validateJavaRunXML(t *testing.T, filename string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "run-java-app", config.Name)
	require.Equal(t, "Application", config.Type) // java command should be detected as Application

	// Check working directory conversion (VSCode parser resolves ${workspaceFolder} to absolute path)
	workingDirOption := findOption(config.Options, "WORKING_DIRECTORY")
	require.NotNil(t, workingDirOption)
	require.Equal(t, "/test/project/build", workingDirOption.Value)
}

func validateGradleXML(t *testing.T, filename string, expectedTaskName string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "GradleRunTask", config.Type)

	// Check task name option
	taskNameOption := findOption(config.Options, "TASK_NAME")
	require.NotNil(t, taskNameOption)
	require.Equal(t, expectedTaskName, taskNameOption.Value)
}

func validateNodeJSXML(t *testing.T, filename string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "NodeJS", config.Type)

	// Check script text contains node command
	scriptOption := findOption(config.Options, "SCRIPT_TEXT")
	require.NotNil(t, scriptOption)
	require.Contains(t, scriptOption.Value, "node server.js")
	require.Contains(t, scriptOption.Value, "--port 8080")

	// Check working directory (VSCode parser resolves ${workspaceFolder} to absolute path)
	workingDirOption := findOption(config.Options, "WORKING_DIRECTORY")
	require.NotNil(t, workingDirOption)
	require.Equal(t, "/test/project/src", workingDirOption.Value)
}

func validatePythonXML(t *testing.T, filename string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "PythonConfigurationType", config.Type)

	// Check environment variables
	require.NotNil(t, config.EnvVars)

	pythonPathEnv := findEnvVar(config.EnvVars.EnvVars, "PYTHONPATH")
	require.NotNil(t, pythonPathEnv)
	require.Contains(t, pythonPathEnv.Value, "$PROJECT_DIR$/src")
}

func validateMavenXML(t *testing.T, filename string, expectedGoals string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "MavenRunConfiguration", config.Type)

	// Check goals option
	goalsOption := findOption(config.Options, "GOALS")
	require.NotNil(t, goalsOption)
	require.Equal(t, expectedGoals, goalsOption.Value)
}

func validateEdgeCaseXML(t *testing.T, filename string, originalTaskName string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, originalTaskName, config.Name)
	require.Equal(t, "ShellScript", config.Type)

	// Check that script text exists
	scriptOption := findOption(config.Options, "SCRIPT_TEXT")
	require.NotNil(t, scriptOption)
}

func validateComplexGradleXML(t *testing.T, filename string) {
	xmlData, err := os.ReadFile(filename)
	require.NoError(t, err)

	var component JetBrainsComponent

	err = xml.Unmarshal(xmlData, &component)
	require.NoError(t, err)

	config := component.Configuration
	require.Equal(t, "complex-gradle-task", config.Name)
	require.Equal(t, "GradleRunTask", config.Type)

	// Check task arguments
	taskNameOption := findOption(config.Options, "TASK_NAME")
	require.NotNil(t, taskNameOption)
	require.Contains(t, taskNameOption.Value, "clean build")
	require.Contains(t, taskNameOption.Value, "--parallel")

	// Check environment variables
	require.NotNil(t, config.EnvVars)

	gradleOptsEnv := findEnvVar(config.EnvVars.EnvVars, "GRADLE_OPTS")
	require.NotNil(t, gradleOptsEnv)
	require.Contains(t, gradleOptsEnv.Value, "-Xmx4g")

	// Check working directory is converted
	workingDirOption := findOption(config.Options, "WORKING_DIRECTORY")
	require.NotNil(t, workingDirOption)
	require.Equal(t, "/test/project/subproject", workingDirOption.Value)
}

// Helper functions to find options and environment variables

func findOption(options []JetBrainsOption, name string) *JetBrainsOption {
	for _, option := range options {
		if option.Name == name {
			return &option
		}
	}

	return nil
}

func findEnvVar(envVars []JetBrainsEnvVar, name string) *JetBrainsEnvVar {
	for _, envVar := range envVars {
		if envVar.Name == name {
			return &envVar
		}
	}

	return nil
}
