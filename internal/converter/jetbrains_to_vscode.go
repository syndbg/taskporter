package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskporter/internal/config"
)

// JetBrainsToVSCodeConverter converts JetBrains run configurations to VSCode tasks
type JetBrainsToVSCodeConverter struct {
	projectRoot string
	outputPath  string
	verbose     bool
}

// NewJetBrainsToVSCodeConverter creates a new converter
func NewJetBrainsToVSCodeConverter(projectRoot, outputPath string, verbose bool) *JetBrainsToVSCodeConverter {
	return &JetBrainsToVSCodeConverter{
		projectRoot: projectRoot,
		outputPath:  outputPath,
		verbose:     verbose,
	}
}

// VSCodeTasksFile represents the structure of tasks.json
type VSCodeTasksFile struct {
	Version string       `json:"version"`
	Tasks   []VSCodeTask `json:"tasks"`
}

// VSCodeTask represents a single task in tasks.json
type VSCodeTask struct {
	Label          string             `json:"label"`
	Type           string             `json:"type"`
	Command        string             `json:"command,omitempty"`
	Args           []string           `json:"args,omitempty"`
	Group          interface{}        `json:"group,omitempty"`
	Options        *VSCodeTaskOptions `json:"options,omitempty"`
	ProblemMatcher []string           `json:"problemMatcher,omitempty"`
}

// VSCodeTaskOptions represents task options
type VSCodeTaskOptions struct {
	Cwd string            `json:"cwd,omitempty"`
	Env map[string]string `json:"env,omitempty"`
}

// ConvertTasks converts JetBrains tasks to VSCode tasks.json format
func (c *JetBrainsToVSCodeConverter) ConvertTasks(tasks []*config.Task, dryRun bool) error {
	if c.verbose {
		fmt.Printf("🔄 Converting %d JetBrains configurations to VSCode tasks format...\n", len(tasks))
	}

	// Filter only JetBrains tasks
	jetBrainsTasks := make([]*config.Task, 0)
	for _, task := range tasks {
		if task.Type == config.TypeJetBrains {
			jetBrainsTasks = append(jetBrainsTasks, task)
		}
	}

	if len(jetBrainsTasks) == 0 {
		fmt.Printf("⚠️  No JetBrains configurations found to convert\n")
		return nil
	}

	if c.verbose {
		fmt.Printf("📋 Converting %d JetBrains configurations\n", len(jetBrainsTasks))
	}

	// Convert tasks
	vscodeTasksFile := &VSCodeTasksFile{
		Version: "2.0.0",
		Tasks:   make([]VSCodeTask, 0, len(jetBrainsTasks)),
	}

	for _, task := range jetBrainsTasks {
		vscodeTask, err := c.convertSingleTask(task)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert task '%s': %v\n", task.Name, err)
			continue
		}

		vscodeTasksFile.Tasks = append(vscodeTasksFile.Tasks, *vscodeTask)
	}

	// Determine output path
	outputPath := c.outputPath
	if outputPath == "" {
		outputPath = filepath.Join(c.projectRoot, ".vscode", "tasks.json")
	}

	if c.verbose {
		fmt.Printf("📁 Output file: %s\n", outputPath)
	}

	if dryRun {
		fmt.Printf("   [DRY RUN] Would create: %s\n", outputPath)
		fmt.Printf("📝 Preview of tasks.json content:\n")

		jsonData, _ := json.MarshalIndent(vscodeTasksFile, "", "    ")
		fmt.Printf("%s\n", string(jsonData))
	} else {
		// Create output directory
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write tasks.json file
		if err := c.writeVSCodeTasksFile(vscodeTasksFile, outputPath); err != nil {
			return fmt.Errorf("failed to write tasks.json: %w", err)
		}

		if c.verbose {
			fmt.Printf("✅ Successfully created %s\n", outputPath)
		}
	}

	fmt.Printf("✅ Successfully converted %d/%d JetBrains configurations\n", len(vscodeTasksFile.Tasks), len(jetBrainsTasks))

	return nil
}

// convertSingleTask converts a single JetBrains task to VSCode format
func (c *JetBrainsToVSCodeConverter) convertSingleTask(task *config.Task) (*VSCodeTask, error) {
	vscodeTask := &VSCodeTask{
		Label: task.Name,
		Type:  "shell", // Default to shell type
	}

	// Convert based on the task command and structure
	if err := c.determineVSCodeTaskDetails(task, vscodeTask); err != nil {
		return nil, err
	}

	// Set working directory (convert JetBrains variables)
	if task.Cwd != "" {
		if vscodeTask.Options == nil {
			vscodeTask.Options = &VSCodeTaskOptions{}
		}

		vscodeTask.Options.Cwd = c.convertJetBrainsVariables(task.Cwd)
	}

	// Convert environment variables
	if len(task.Env) > 0 {
		if vscodeTask.Options == nil {
			vscodeTask.Options = &VSCodeTaskOptions{}
		}

		vscodeTask.Options.Env = make(map[string]string)
		for key, value := range task.Env {
			vscodeTask.Options.Env[key] = c.convertJetBrainsVariables(value)
		}
	}

	// Set task group based on common patterns
	vscodeTask.Group = c.determineTaskGroup(task)

	return vscodeTask, nil
}

// determineVSCodeTaskDetails sets command and args based on the JetBrains task
func (c *JetBrainsToVSCodeConverter) determineVSCodeTaskDetails(task *config.Task, vscodeTask *VSCodeTask) error {
	// Parse the command from task.Command which might contain the full command line
	parts := strings.Fields(task.Command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command in task '%s'", task.Name)
	}

	vscodeTask.Command = parts[0]

	// Combine command arguments with task arguments
	allArgs := make([]string, 0)
	if len(parts) > 1 {
		allArgs = append(allArgs, parts[1:]...)
	}

	allArgs = append(allArgs, task.Args...)

	if len(allArgs) > 0 {
		vscodeTask.Args = allArgs
	}

	return nil
}

// determineTaskGroup determines the appropriate VSCode task group
func (c *JetBrainsToVSCodeConverter) determineTaskGroup(task *config.Task) interface{} {
	taskName := strings.ToLower(task.Name)
	command := strings.ToLower(task.Command)

	// Check for common build patterns
	if strings.Contains(taskName, "build") || strings.Contains(command, "build") ||
		strings.Contains(command, "gradle") && (strings.Contains(command, "build") || strings.Contains(command, "assemble")) ||
		strings.Contains(command, "mvn") && strings.Contains(command, "compile") ||
		strings.Contains(command, "make") {
		return map[string]interface{}{
			"kind":      "build",
			"isDefault": strings.Contains(taskName, "build") && !strings.Contains(taskName, "test"),
		}
	}

	// Check for test patterns
	if strings.Contains(taskName, "test") || strings.Contains(command, "test") ||
		strings.Contains(command, "gradle") && strings.Contains(command, "test") ||
		strings.Contains(command, "mvn") && strings.Contains(command, "test") ||
		strings.Contains(command, "npm") && strings.Contains(command, "test") {
		return "test"
	}

	// Default to none for other tasks
	return "none"
}

// convertJetBrainsVariables converts JetBrains variables to VSCode format
func (c *JetBrainsToVSCodeConverter) convertJetBrainsVariables(input string) string {
	result := input

	// Convert JetBrains variables to VSCode equivalents
	result = strings.ReplaceAll(result, "$PROJECT_DIR$", "${workspaceFolder}")
	result = strings.ReplaceAll(result, "$MODULE_DIR$", "${workspaceFolder}")
	result = strings.ReplaceAll(result, "$FileDir$", "${fileDirname}")
	result = strings.ReplaceAll(result, "$FileName$", "${fileBasename}")
	result = strings.ReplaceAll(result, "$FilePath$", "${file}")

	return result
}

// writeVSCodeTasksFile writes the VSCode tasks file
func (c *JetBrainsToVSCodeConverter) writeVSCodeTasksFile(tasksFile *VSCodeTasksFile, outputPath string) error {
	jsonData, err := json.MarshalIndent(tasksFile, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks.json: %w", err)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
