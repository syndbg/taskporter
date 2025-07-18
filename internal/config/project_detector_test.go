package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectDetector(t *testing.T) {
	t.Run("NewProjectDetector", func(t *testing.T) {
		tests := []struct {
			name        string
			projectRoot string
			wantAbsPath bool
		}{
			{
				name:        "empty root defaults to current directory",
				projectRoot: "",
				wantAbsPath: true,
			},
			{
				name:        "relative path gets converted to absolute",
				projectRoot: ".",
				wantAbsPath: true,
			},
			{
				name:        "specific path",
				projectRoot: "/tmp/test",
				wantAbsPath: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				detector := NewProjectDetector(tt.projectRoot)

				require.NotNil(t, detector)

				if tt.wantAbsPath {
					require.True(t, filepath.IsAbs(detector.projectRoot),
						"Expected absolute path, got: %s", detector.projectRoot)
				}
			})
		}
	})

	t.Run("DetectProject", func(t *testing.T) {
		t.Run("with VSCode and JetBrains configs", func(t *testing.T) {
			// Create a temporary directory structure for testing
			tempDir := t.TempDir()

			// Create VSCode directory structure
			vscodeDir := filepath.Join(tempDir, ".vscode")
			require.NoError(t, os.MkdirAll(vscodeDir, 0755))

			// Create JetBrains directory structure
			jetbrainsDir := filepath.Join(tempDir, ".idea", "runConfigurations")
			require.NoError(t, os.MkdirAll(jetbrainsDir, 0755))

			detector := NewProjectDetector(tempDir)
			config, err := detector.DetectProject()

			require.NoError(t, err)
			require.NotNil(t, config)
			require.True(t, config.HasVSCode)
			require.True(t, config.HasJetBrains)
			require.Len(t, config.Tasks, 0)

			expectedRoot, _ := filepath.Abs(tempDir)
			require.Equal(t, expectedRoot, config.ProjectRoot)
		})

		t.Run("without configs", func(t *testing.T) {
			// Create a temporary directory without any config directories
			tempDir := t.TempDir()

			detector := NewProjectDetector(tempDir)
			config, err := detector.DetectProject()

			require.NoError(t, err)
			require.False(t, config.HasVSCode)
			require.False(t, config.HasJetBrains)
		})
	})

	t.Run("GetVSCodeTasksPath", func(t *testing.T) {
		t.Run("when tasks.json exists", func(t *testing.T) {
			tempDir := t.TempDir()

			vscodeDir := filepath.Join(tempDir, ".vscode")
			require.NoError(t, os.MkdirAll(vscodeDir, 0755))

			tasksFile := filepath.Join(vscodeDir, "tasks.json")
			require.NoError(t, os.WriteFile(tasksFile, []byte("{}"), 0644))

			detector := NewProjectDetector(tempDir)
			path := detector.GetVSCodeTasksPath()

			require.NotEmpty(t, path)
			require.Equal(t, tasksFile, path)
		})

		t.Run("when tasks.json doesn't exist", func(t *testing.T) {
			tempDir := t.TempDir()
			detector := NewProjectDetector(tempDir)
			path := detector.GetVSCodeTasksPath()

			require.Empty(t, path)
		})
	})

	t.Run("GetVSCodeLaunchPath", func(t *testing.T) {
		t.Run("when launch.json exists", func(t *testing.T) {
			tempDir := t.TempDir()

			vscodeDir := filepath.Join(tempDir, ".vscode")
			require.NoError(t, os.MkdirAll(vscodeDir, 0755))

			launchFile := filepath.Join(vscodeDir, "launch.json")
			require.NoError(t, os.WriteFile(launchFile, []byte("{}"), 0644))

			detector := NewProjectDetector(tempDir)
			path := detector.GetVSCodeLaunchPath()

			require.NotEmpty(t, path)
			require.Equal(t, launchFile, path)
		})

		t.Run("when launch.json doesn't exist", func(t *testing.T) {
			tempDir := t.TempDir()
			detector := NewProjectDetector(tempDir)
			path := detector.GetVSCodeLaunchPath()

			require.Empty(t, path)
		})
	})

	t.Run("GetJetBrainsRunConfigPaths", func(t *testing.T) {
		t.Run("with XML configuration files", func(t *testing.T) {
			tempDir := t.TempDir()

			// Create JetBrains run configurations directory
			runConfigsDir := filepath.Join(tempDir, ".idea", "runConfigurations")
			require.NoError(t, os.MkdirAll(runConfigsDir, 0755))

			// Create test XML files
			xmlFiles := []string{"config1.xml", "config2.xml", "config3.xml"}
			for _, filename := range xmlFiles {
				xmlPath := filepath.Join(runConfigsDir, filename)
				require.NoError(t, os.WriteFile(xmlPath, []byte("<configuration/>"), 0644))
			}

			// Create a non-XML file (should be ignored)
			txtPath := filepath.Join(runConfigsDir, "readme.txt")
			require.NoError(t, os.WriteFile(txtPath, []byte("readme"), 0644))

			detector := NewProjectDetector(tempDir)
			paths := detector.GetJetBrainsRunConfigPaths()

			require.Len(t, paths, 3)

			// Verify all returned paths are XML files
			for _, path := range paths {
				require.Equal(t, ".xml", filepath.Ext(path))
			}
		})

		t.Run("without runConfigurations directory", func(t *testing.T) {
			tempDir := t.TempDir()

			detector := NewProjectDetector(tempDir)
			paths := detector.GetJetBrainsRunConfigPaths()

			require.Len(t, paths, 0)
		})
	})
}
