package vscode

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceResolver(t *testing.T) {
	projectRoot := "/test/project"
	resolver := NewWorkspaceResolver(projectRoot)

	t.Run("workspace folder variables", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "workspaceFolder",
				input:    "${workspaceFolder}",
				expected: projectRoot,
			},
			{
				name:     "workspaceRoot legacy alias",
				input:    "${workspaceRoot}",
				expected: projectRoot,
			},
			{
				name:     "workspaceFolderBasename",
				input:    "${workspaceFolderBasename}",
				expected: "project",
			},
			{
				name:     "workspaceFolder in path",
				input:    "${workspaceFolder}/src/main.go",
				expected: "/test/project/src/main.go",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := resolver.ResolveVariables(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("file variables", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "file",
				input:    "${file}",
				expected: projectRoot,
			},
			{
				name:     "fileWorkspaceFolder",
				input:    "${fileWorkspaceFolder}",
				expected: projectRoot,
			},
			{
				name:     "relativeFile",
				input:    "${relativeFile}",
				expected: "/test/project", // "." gets resolved to project root
			},
			{
				name:     "fileBasename",
				input:    "${fileBasename}",
				expected: "project",
			},
			{
				name:     "fileBasenameNoExtension",
				input:    "${fileBasenameNoExtension}",
				expected: "project",
			},
			{
				name:     "fileExtname",
				input:    "${fileExtname}",
				expected: "",
			},
			{
				name:     "fileDirname",
				input:    "${fileDirname}",
				expected: projectRoot,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := resolver.ResolveVariables(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("environment variables", func(t *testing.T) {
		// Set a test environment variable
		_ = os.Setenv("TEST_VAR", "test_value")

		defer os.Unsetenv("TEST_VAR")

		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "env variable",
				input:    "${env:TEST_VAR}",
				expected: "test_value",
			},
			{
				name:     "env PATH",
				input:    "${env:PATH}",
				expected: os.Getenv("PATH"),
			},
			{
				name:  "userHome",
				input: "${userHome}",
				expected: func() string {
					home, _ := os.UserHomeDir()
					return home
				}(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := resolver.ResolveVariables(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("path separator", func(t *testing.T) {
		result := resolver.ResolveVariables("${pathSeparator}")
		require.Equal(t, string(filepath.Separator), result)
	})

	t.Run("config variables", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "terminal.external.linuxExec",
				input:    "${config:terminal.external.linuxExec}",
				expected: "x-terminal-emulator",
			},
			{
				name:     "editor.fontSize",
				input:    "${config:editor.fontSize}",
				expected: "14",
			},
			{
				name:     "unknown config",
				input:    "${config:unknown.setting}",
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := resolver.ResolveVariables(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("multiple variables in one string", func(t *testing.T) {
		input := "${workspaceFolder}/bin/${fileBasename}"
		expected := "/test/project/bin/project"
		result := resolver.ResolveVariables(input)
		require.Equal(t, expected, result)
	})

	t.Run("relative path handling", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "already absolute",
				input:    "/absolute/path",
				expected: "/absolute/path",
			},
			{
				name:     "no variables, stays relative",
				input:    "relative/path",
				expected: "relative/path",
			},
			{
				name:     "variable resolved to relative becomes absolute",
				input:    "${relativeFile}/src",
				expected: "/test/project/src",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := resolver.ResolveVariables(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestResolveStringSlice(t *testing.T) {
	projectRoot := "/test/project"
	resolver := NewWorkspaceResolver(projectRoot)

	input := []string{
		"${workspaceFolder}/src",
		"--config=${workspaceFolder}/config.json",
		"regular-arg",
	}

	expected := []string{
		"/test/project/src",
		"--config=/test/project/config.json",
		"regular-arg",
	}

	result := resolver.ResolveStringSlice(input)
	require.Equal(t, expected, result)
}

func TestResolveStringMap(t *testing.T) {
	projectRoot := "/test/project"
	resolver := NewWorkspaceResolver(projectRoot)

	input := map[string]string{
		"CONFIG_PATH": "${workspaceFolder}/config",
		"HOME_DIR":    "${userHome}",
		"STATIC":      "static-value",
	}

	result := resolver.ResolveStringMap(input)

	require.Equal(t, "/test/project/config", result["CONFIG_PATH"])
	require.NotEmpty(t, result["HOME_DIR"]) // userHome should resolve to something
	require.Equal(t, "static-value", result["STATIC"])
}

func TestHasWorkspaceVariables(t *testing.T) {
	resolver := NewWorkspaceResolver("/test")

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "has workspace variable",
			input:    "${workspaceFolder}/src",
			expected: true,
		},
		{
			name:     "has file variable",
			input:    "${file}",
			expected: true,
		},
		{
			name:     "has env variable",
			input:    "${env:PATH}",
			expected: true,
		},
		{
			name:     "has config variable",
			input:    "${config:setting}",
			expected: true,
		},
		{
			name:     "no variables",
			input:    "/regular/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.HasWorkspaceVariables(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultShell(t *testing.T) {
	resolver := NewWorkspaceResolver("/test")
	shell := resolver.getDefaultShell()

	switch runtime.GOOS {
	case "windows":
		require.True(t, shell == "powershell" || shell == "cmd")
	case "darwin":
		require.True(t, strings.Contains(shell, "zsh") || strings.Contains(shell, "bash") || shell == "/bin/zsh")
	default:
		require.True(t, strings.Contains(shell, "bash") || strings.Contains(shell, "zsh") || shell == "/bin/bash")
	}
}

func TestResolveEnvironmentVariables(t *testing.T) {
	resolver := NewWorkspaceResolver("/test")

	// Set test environment variables
	_ = os.Setenv("TEST_VAR1", "value1")
	_ = os.Setenv("TEST_VAR2", "value2")

	defer func() {
		_ = os.Unsetenv("TEST_VAR1")
		_ = os.Unsetenv("TEST_VAR2")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single env var",
			input:    "${env:TEST_VAR1}",
			expected: "value1",
		},
		{
			name:     "multiple env vars",
			input:    "${env:TEST_VAR1}-${env:TEST_VAR2}",
			expected: "value1-value2",
		},
		{
			name:     "env var in path",
			input:    "/path/${env:TEST_VAR1}/file",
			expected: "/path/value1/file",
		},
		{
			name:     "nonexistent env var",
			input:    "${env:NONEXISTENT}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.resolveEnvironmentVariables(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveConfigVariables(t *testing.T) {
	resolver := NewWorkspaceResolver("/test")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "known config",
			input:    "${config:editor.fontSize}",
			expected: "14",
		},
		{
			name:     "multiple configs",
			input:    "${config:editor.fontSize}-${config:editor.tabSize}",
			expected: "14-4",
		},
		{
			name:     "unknown config",
			input:    "${config:unknown.setting}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.resolveConfigVariables(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestComplexVariableResolution(t *testing.T) {
	projectRoot := "/test/my-project"
	resolver := NewWorkspaceResolver(projectRoot)

	// Set test environment variable
	_ = os.Setenv("BUILD_MODE", "production")

	defer os.Unsetenv("BUILD_MODE")

	// Test a complex command with multiple variable types
	input := "${workspaceFolder}/build.sh --mode=${env:BUILD_MODE} --config=${config:editor.tabSize} --output=${workspaceFolder}/dist"
	result := resolver.ResolveVariables(input)

	expected := "/test/my-project/build.sh --mode=production --config=4 --output=/test/my-project/dist"
	require.Equal(t, expected, result)
}
