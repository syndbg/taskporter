package config

import (
	"os"
	"path/filepath"
)

// ProjectDetector handles detection of IDE configuration files
type ProjectDetector struct {
	projectRoot string
}

// NewProjectDetector creates a new project detector for the given directory
func NewProjectDetector(projectRoot string) *ProjectDetector {
	if projectRoot == "" {
		projectRoot = "."
	}

	// Convert to absolute path
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		abs = projectRoot
	}

	return &ProjectDetector{
		projectRoot: abs,
	}
}

// DetectProject scans for IDE configuration files and returns project config
func (pd *ProjectDetector) DetectProject() (*ProjectConfig, error) {
	config := &ProjectConfig{
		ProjectRoot: pd.projectRoot,
		Tasks:       []*Task{},
	}

	// Check for VSCode configurations
	vscodeDir := filepath.Join(pd.projectRoot, ".vscode")
	if pd.dirExists(vscodeDir) {
		config.HasVSCode = true
	}

	// Check for JetBrains configurations
	ideaDir := filepath.Join(pd.projectRoot, ".idea")
	if pd.dirExists(ideaDir) {
		runConfigsDir := filepath.Join(ideaDir, "runConfigurations")
		if pd.dirExists(runConfigsDir) {
			config.HasJetBrains = true
		}
	}

	return config, nil
}

// GetVSCodeTasksPath returns the path to VSCode tasks.json if it exists
func (pd *ProjectDetector) GetVSCodeTasksPath() string {
	path := filepath.Join(pd.projectRoot, ".vscode", "tasks.json")
	if pd.fileExists(path) {
		return path
	}
	return ""
}

// GetVSCodeLaunchPath returns the path to VSCode launch.json if it exists
func (pd *ProjectDetector) GetVSCodeLaunchPath() string {
	path := filepath.Join(pd.projectRoot, ".vscode", "launch.json")
	if pd.fileExists(path) {
		return path
	}
	return ""
}

// GetJetBrainsRunConfigPaths returns paths to all JetBrains run configuration files
func (pd *ProjectDetector) GetJetBrainsRunConfigPaths() []string {
	var paths []string
	runConfigsDir := filepath.Join(pd.projectRoot, ".idea", "runConfigurations")

	if !pd.dirExists(runConfigsDir) {
		return paths
	}

	entries, err := os.ReadDir(runConfigsDir)
	if err != nil {
		return paths
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".xml" {
			paths = append(paths, filepath.Join(runConfigsDir, entry.Name()))
		}
	}

	return paths
}

// Helper functions
func (pd *ProjectDetector) fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (pd *ProjectDetector) dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
