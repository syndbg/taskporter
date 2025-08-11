package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndbg/taskporter/internal/config"
)

func TestTaskRunner_WorkspaceVariableResolution(t *testing.T) {
	// Create a temporary directory to act as workspace
	tmpDir, err := os.MkdirTemp("", "taskporter-workspace-test")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	t.Run("resolves VSCode workspace variables in task execution", func(t *testing.T) {
		runner := NewTaskRunnerWithProjectRoot(false, tmpDir)

		// Create a VSCode task with workspace variables
		task := &config.Task{
			Name:    "test-task",
			Type:    config.TypeVSCodeTask,
			Command: "echo",
			Args: []string{
				"${workspaceFolder}/src",
				"${workspaceFolder}/build",
				"${env:HOME}",
			},
			Cwd: "${workspaceFolder}/subdir",
			Env: map[string]string{
				"PROJECT_ROOT": "${workspaceFolder}",
				"BUILD_DIR":    "${workspaceFolder}/build",
			},
		}

		// Resolve variables for execution
		resolvedTask := runner.resolveTaskVariables(task)

		// Verify variables were resolved
		require.Equal(t, "echo", resolvedTask.Command)
		require.Contains(t, resolvedTask.Args, filepath.Join(tmpDir, "src"))
		require.Contains(t, resolvedTask.Args, filepath.Join(tmpDir, "build"))
		require.Equal(t, filepath.Join(tmpDir, "subdir"), resolvedTask.Cwd)
		require.Equal(t, tmpDir, resolvedTask.Env["PROJECT_ROOT"])
		require.Equal(t, filepath.Join(tmpDir, "build"), resolvedTask.Env["BUILD_DIR"])

		// HOME environment variable should be resolved
		homeDir, _ := os.UserHomeDir()
		require.Contains(t, resolvedTask.Args, homeDir)
	})

	t.Run("preserves non-VSCode tasks unchanged", func(t *testing.T) {
		runner := NewTaskRunnerWithProjectRoot(false, tmpDir)

		// Create a JetBrains task (should not be modified)
		task := &config.Task{
			Name:    "jetbrains-task",
			Type:    config.TypeJetBrains,
			Command: "echo",
			Args:    []string{"$PROJECT_DIR$/src"},
			Cwd:     "$PROJECT_DIR$/build",
		}

		// Resolve variables for execution
		resolvedTask := runner.resolveTaskVariables(task)

		// Verify task was not modified (JetBrains variables preserved)
		require.Equal(t, task.Command, resolvedTask.Command)
		require.Equal(t, task.Args, resolvedTask.Args)
		require.Equal(t, task.Cwd, resolvedTask.Cwd)
	})

	t.Run("handles complex variable combinations", func(t *testing.T) {
		runner := NewTaskRunnerWithProjectRoot(false, tmpDir)

		// Set test environment variable
		_ = os.Setenv("BUILD_MODE", "production")

		defer os.Unsetenv("BUILD_MODE")

		task := &config.Task{
			Name:    "complex-task",
			Type:    config.TypeVSCodeLaunch,
			Command: "go",
			Args: []string{
				"build",
				"-o",
				"${workspaceFolder}/dist/app-${env:BUILD_MODE}",
				"${workspaceFolder}/cmd/main.go",
			},
			Cwd: "${workspaceFolder}",
			Env: map[string]string{
				"CGO_ENABLED": "0",
				"OUTPUT_DIR":  "${workspaceFolder}/dist",
			},
		}

		resolvedTask := runner.resolveTaskVariables(task)

		// Verify complex resolution
		expectedOutputPath := filepath.Join(tmpDir, "dist", "app-production")
		expectedMainPath := filepath.Join(tmpDir, "cmd", "main.go")

		require.Contains(t, resolvedTask.Args, expectedOutputPath)
		require.Contains(t, resolvedTask.Args, expectedMainPath)
		require.Equal(t, tmpDir, resolvedTask.Cwd)
		require.Equal(t, filepath.Join(tmpDir, "dist"), resolvedTask.Env["OUTPUT_DIR"])
	})

	t.Run("handles missing environment variables gracefully", func(t *testing.T) {
		runner := NewTaskRunnerWithProjectRoot(false, tmpDir)

		task := &config.Task{
			Name:    "env-test",
			Type:    config.TypeVSCodeTask,
			Command: "echo",
			Args:    []string{"${env:NONEXISTENT_VAR}"},
		}

		resolvedTask := runner.resolveTaskVariables(task)

		// Nonexistent environment variables should resolve to empty string
		require.Contains(t, resolvedTask.Args, "")
	})
}

func TestTaskRunner_WorkspaceVariableResolutionIntegration(t *testing.T) {
	// Create a temporary workspace
	tmpDir, err := os.MkdirTemp("", "taskporter-integration-test")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	// Create some test files
	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(srcDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	t.Run("executes task with resolved workspace variables", func(t *testing.T) {
		runner := NewTaskRunnerWithProjectRoot(false, tmpDir)

		// Create a task that lists contents of the workspace src directory
		task := &config.Task{
			Name:    "list-src",
			Type:    config.TypeVSCodeTask,
			Command: "ls",
			Args:    []string{"-la", "${workspaceFolder}/src"},
			Cwd:     "${workspaceFolder}",
		}

		// This would normally execute the command, but for testing we just verify resolution
		resolvedTask := runner.resolveTaskVariables(task)

		require.Equal(t, "ls", resolvedTask.Command)
		require.Contains(t, resolvedTask.Args, filepath.Join(tmpDir, "src"))
		require.Equal(t, tmpDir, resolvedTask.Cwd)
	})
}
