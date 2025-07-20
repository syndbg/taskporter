package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizer(t *testing.T) {
	t.Run("NewSanitizer", func(t *testing.T) {
		sanitizer := NewSanitizer("/test/project")
		require.NotNil(t, sanitizer)
		require.Equal(t, "/test/project", sanitizer.projectRoot)
	})

	t.Run("SanitizeCommand", func(t *testing.T) {
		sanitizer := NewSanitizer("/test/project")

		t.Run("should allow valid commands", func(t *testing.T) {
			validCommands := []string{
				"go", "node", "python", "java", "gradle", "mvn",
				"npm", "yarn", "make", "cargo", "rustc",
				"./script.sh", "/usr/bin/gcc",
			}

			for _, cmd := range validCommands {
				err := sanitizer.SanitizeCommand(cmd)
				require.NoError(t, err, "Valid command should pass: %s", cmd)
			}
		})

		t.Run("should reject dangerous commands", func(t *testing.T) {
			dangerousCommands := []string{
				"rm -rf /",
				"curl -s http://evil.com",
				"cmd; rm -rf *",
				"echo $(whoami)",
				"ls | grep secret",
				"cat /etc/passwd",
			}

			for _, cmd := range dangerousCommands {
				err := sanitizer.SanitizeCommand(cmd)
				require.Error(t, err, "Dangerous command should be rejected: %s", cmd)
			}
		})

		t.Run("should reject empty command", func(t *testing.T) {
			err := sanitizer.SanitizeCommand("")
			require.Error(t, err)
			require.Contains(t, err.Error(), "cannot be empty")
		})

		t.Run("should reject commands with invalid characters", func(t *testing.T) {
			invalidCommands := []string{
				"cmd<script>",
				"cmd>output",
				"cmd&background",
				"cmd$var",
			}

			for _, cmd := range invalidCommands {
				err := sanitizer.SanitizeCommand(cmd)
				require.Error(t, err, "Command with invalid chars should be rejected: %s", cmd)
			}
		})
	})

	t.Run("SanitizeArgs", func(t *testing.T) {
		sanitizer := NewSanitizer("/test/project")

		t.Run("should allow valid arguments", func(t *testing.T) {
			validArgs := []string{"build", "--verbose", "-o", "output.bin", "src/main.go"}
			result, err := sanitizer.SanitizeArgs(validArgs)
			require.NoError(t, err)
			require.Equal(t, validArgs, result)
		})

		t.Run("should reject dangerous arguments", func(t *testing.T) {
			dangerousArgs := []string{
				"--exec rm -rf /",
				"-c malicious_command",
				"$(whoami)",
				"`ls -la`",
				"arg; rm file",
			}

			for _, arg := range dangerousArgs {
				_, err := sanitizer.SanitizeArgs([]string{arg})
				require.Error(t, err, "Dangerous argument should be rejected: %s", arg)
			}
		})

		t.Run("should filter empty arguments", func(t *testing.T) {
			args := []string{"build", "", "--verbose", "", "main.go"}
			result, err := sanitizer.SanitizeArgs(args)
			require.NoError(t, err)
			expected := []string{"build", "--verbose", "main.go"}
			require.Equal(t, expected, result)
		})

		t.Run("should handle empty argument list", func(t *testing.T) {
			result, err := sanitizer.SanitizeArgs([]string{})
			require.NoError(t, err)
			require.Empty(t, result)
		})
	})

	t.Run("SanitizeEnvironment", func(t *testing.T) {
		sanitizer := NewSanitizer("/test/project")

		t.Run("should allow valid environment variables", func(t *testing.T) {
			validEnv := map[string]string{
				"DEBUG":      "true",
				"NODE_ENV":   "development",
				"BUILD_TYPE": "release",
				"PORT":       "8080",
			}

			result, err := sanitizer.SanitizeEnvironment(validEnv)
			require.NoError(t, err)
			require.Equal(t, validEnv, result)
		})

		t.Run("should reject dangerous environment variable keys", func(t *testing.T) {
			dangerousKeys := []string{
				"PATH", "LD_LIBRARY_PATH", "SHELL", "IFS",
			}

			for _, key := range dangerousKeys {
				env := map[string]string{key: "value"}
				_, err := sanitizer.SanitizeEnvironment(env)
				require.Error(t, err, "Dangerous env key should be rejected: %s", key)
			}
		})

		t.Run("should reject invalid environment variable names", func(t *testing.T) {
			invalidEnvs := map[string]string{
				"invalid-name": "value", // hyphens not allowed
				"123INVALID":   "value", // cannot start with number
				"invalid name": "value", // spaces not allowed
				"":             "value", // empty key
			}

			for key, value := range invalidEnvs {
				env := map[string]string{key: value}
				_, err := sanitizer.SanitizeEnvironment(env)
				require.Error(t, err, "Invalid env name should be rejected: %s", key)
			}
		})

		t.Run("should reject dangerous environment variable values", func(t *testing.T) {
			dangerousValues := []string{
				"$(whoami)",
				"`ls -la`",
				"value; rm file",
				"../../../etc/passwd",
			}

			for _, value := range dangerousValues {
				env := map[string]string{"TEST_VAR": value}
				_, err := sanitizer.SanitizeEnvironment(env)
				require.Error(t, err, "Dangerous env value should be rejected: %s", value)
			}
		})

		t.Run("should handle empty environment", func(t *testing.T) {
			result, err := sanitizer.SanitizeEnvironment(map[string]string{})
			require.NoError(t, err)
			require.Empty(t, result)
		})
	})

	t.Run("SanitizePath", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "sanitizer_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		sanitizer := NewSanitizer(tempDir)

		t.Run("should allow valid paths", func(t *testing.T) {
			validPaths := []string{
				"src/main.go",
				"./build",
				"bin/output",
			}

			for _, path := range validPaths {
				result, err := sanitizer.SanitizePath(path)
				require.NoError(t, err, "Valid path should pass: %s", path)
				require.NotEmpty(t, result)
			}
		})

		t.Run("should reject directory traversal attempts", func(t *testing.T) {
			dangerousPaths := []string{
				"../../../etc/passwd",
				"..\\..\\..\\windows\\system32",
				"./../../secret",
			}

			for _, path := range dangerousPaths {
				_, err := sanitizer.SanitizePath(path)
				require.Error(t, err, "Directory traversal should be rejected: %s", path)
			}
		})

		t.Run("should handle absolute paths", func(t *testing.T) {
			absPath := "/usr/bin/git"
			result, err := sanitizer.SanitizePath(absPath)
			require.NoError(t, err)
			require.Equal(t, absPath, result)
		})

		t.Run("should reject empty path", func(t *testing.T) {
			_, err := sanitizer.SanitizePath("")
			require.Error(t, err)
			require.Contains(t, err.Error(), "cannot be empty")
		})

		t.Run("should convert relative paths to absolute within project", func(t *testing.T) {
			relativePath := "src/main.go"
			result, err := sanitizer.SanitizePath(relativePath)
			require.NoError(t, err)
			expectedPath := filepath.Join(tempDir, relativePath)
			require.Equal(t, expectedPath, result)
		})
	})

	t.Run("ValidateTaskName", func(t *testing.T) {
		sanitizer := NewSanitizer("/test/project")

		t.Run("should allow valid task names", func(t *testing.T) {
			validNames := []string{
				"build",
				"test-unit",
				"run_dev",
				"Deploy to Production",
				"Task with spaces",
			}

			for _, name := range validNames {
				err := sanitizer.ValidateTaskName(name)
				require.NoError(t, err, "Valid task name should pass: %s", name)
			}
		})

		t.Run("should reject dangerous task names", func(t *testing.T) {
			dangerousNames := []string{
				"task & rm -rf /",
				"task; malicious",
				"task|pipe",
				"task$(cmd)",
				"task`backtick`",
				"task\"quote",
				"task'single",
			}

			for _, name := range dangerousNames {
				err := sanitizer.ValidateTaskName(name)
				require.Error(t, err, "Dangerous task name should be rejected: %s", name)
			}
		})

		t.Run("should reject empty task name", func(t *testing.T) {
			err := sanitizer.ValidateTaskName("")
			require.Error(t, err)
			require.Contains(t, err.Error(), "cannot be empty")
		})

		t.Run("should reject overly long task names", func(t *testing.T) {
			longName := string(make([]byte, 101)) // 101 characters
			for i := range longName {
				longName = longName[:i] + "a" + longName[i+1:]
			}
			err := sanitizer.ValidateTaskName(longName)
			require.Error(t, err)
			require.Contains(t, err.Error(), "too long")
		})
	})

	t.Run("ValidateConfigPath", func(t *testing.T) {
		// Create temporary files for testing
		tempDir, err := os.MkdirTemp("", "sanitizer_config_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		validConfigFile := filepath.Join(tempDir, "valid.json")
		err = os.WriteFile(validConfigFile, []byte("{}"), 0644)
		require.NoError(t, err)

		configDir := filepath.Join(tempDir, "config_dir")
		err = os.Mkdir(configDir, 0755)
		require.NoError(t, err)

		sanitizer := NewSanitizer(tempDir)

		t.Run("should allow valid config file", func(t *testing.T) {
			err := sanitizer.ValidateConfigPath(validConfigFile)
			require.NoError(t, err)
		})

		t.Run("should allow empty config path", func(t *testing.T) {
			err := sanitizer.ValidateConfigPath("")
			require.NoError(t, err)
		})

		t.Run("should reject non-existent file", func(t *testing.T) {
			err := sanitizer.ValidateConfigPath(filepath.Join(tempDir, "nonexistent.json"))
			require.Error(t, err)
		})

		t.Run("should reject directory as config file", func(t *testing.T) {
			err := sanitizer.ValidateConfigPath(configDir)
			require.Error(t, err)
			require.Contains(t, err.Error(), "directory")
		})

		t.Run("should reject directory traversal in config path", func(t *testing.T) {
			err := sanitizer.ValidateConfigPath("../../../etc/passwd")
			require.Error(t, err)
		})
	})

	t.Run("ValidateOutputPath", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sanitizer_output_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		sanitizer := NewSanitizer(tempDir)

		t.Run("should allow empty output path", func(t *testing.T) {
			err := sanitizer.ValidateOutputPath("")
			require.NoError(t, err)
		})

		t.Run("should allow valid output path", func(t *testing.T) {
			outputPath := filepath.Join(tempDir, "output")
			err := sanitizer.ValidateOutputPath(outputPath)
			require.NoError(t, err)
		})

		t.Run("should reject directory traversal in output path", func(t *testing.T) {
			err := sanitizer.ValidateOutputPath("../../../dangerous/path")
			require.Error(t, err)
		})
	})
}
