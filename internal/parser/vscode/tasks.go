package vscode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syndbg/taskporter/internal/config"
)

// VSCodeTaskFile represents the structure of VSCode tasks.json
type VSCodeTaskFile struct {
	Version string       `json:"version"`
	Tasks   []VSCodeTask `json:"tasks"`
}

// VSCodeTask represents a single task in VSCode tasks.json
type VSCodeTask struct {
	Label          string                  `json:"label"`
	Type           string                  `json:"type"`
	Command        string                  `json:"command,omitempty"`
	Args           []string                `json:"args,omitempty"`
	Group          interface{}             `json:"group,omitempty"` // Can be string or object
	Options        *VSCodeTaskOptions      `json:"options,omitempty"`
	Presentation   *VSCodeTaskPresentation `json:"presentation,omitempty"`
	ProblemMatcher interface{}             `json:"problemMatcher,omitempty"`
	DependsOn      interface{}             `json:"dependsOn,omitempty"`
	Detail         string                  `json:"detail,omitempty"`
}

// VSCodeTaskOptions represents task execution options
type VSCodeTaskOptions struct {
	Cwd string            `json:"cwd,omitempty"`
	Env map[string]string `json:"env,omitempty"`
}

// VSCodeTaskPresentation represents task presentation options
type VSCodeTaskPresentation struct {
	Echo   bool   `json:"echo,omitempty"`
	Reveal string `json:"reveal,omitempty"`
	Focus  bool   `json:"focus,omitempty"`
	Panel  string `json:"panel,omitempty"`
}

// VSCodeTaskGroup represents task group information
type VSCodeTaskGroup struct {
	Kind      string `json:"kind"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

// TasksParser handles parsing of VSCode tasks.json files
type TasksParser struct {
	projectRoot string
}

// NewTasksParser creates a new VSCode tasks parser
func NewTasksParser(projectRoot string) *TasksParser {
	return &TasksParser{
		projectRoot: projectRoot,
	}
}

// ParseTasks parses a VSCode tasks.json file and returns internal Task structures
func (p *TasksParser) ParseTasks(tasksFilePath string) ([]*config.Task, error) {
	data, err := os.ReadFile(tasksFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file %s: %w", tasksFilePath, err)
	}

	var taskFile VSCodeTaskFile
	if err := parseJSONC(data, &taskFile); err != nil {
		return nil, fmt.Errorf("failed to parse tasks JSON: %w", err)
	}

	var tasks []*config.Task

	for _, vscodeTask := range taskFile.Tasks {
		task, err := p.convertTask(vscodeTask, tasksFilePath)
		if err != nil {
			// Log error but continue with other tasks
			fmt.Printf("Warning: failed to convert task %s: %v\n", vscodeTask.Label, err)
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// convertTask converts a VSCode task to our internal Task structure
func (p *TasksParser) convertTask(vscodeTask VSCodeTask, sourceFile string) (*config.Task, error) {
	task := &config.Task{
		Name:        vscodeTask.Label,
		Type:        config.TypeVSCodeTask,
		Command:     vscodeTask.Command,
		Args:        vscodeTask.Args,
		Description: vscodeTask.Detail,
		Source:      sourceFile,
	}

	// Handle group information
	task.Group = p.parseGroup(vscodeTask.Group)

	// Handle options (cwd and env)
	if vscodeTask.Options != nil {
		if vscodeTask.Options.Cwd != "" {
			task.Cwd = p.resolveWorkspacePath(vscodeTask.Options.Cwd)
		}

		if vscodeTask.Options.Env != nil {
			task.Env = make(map[string]string)
			for k, v := range vscodeTask.Options.Env {
				task.Env[k] = v
			}
		}
	}

	// Set default working directory to project root if not specified
	if task.Cwd == "" {
		task.Cwd = p.projectRoot
	}

	return task, nil
}

// parseGroup extracts group information from VSCode task group field
func (p *TasksParser) parseGroup(group interface{}) string {
	if group == nil {
		return ""
	}

	switch g := group.(type) {
	case string:
		return g
	case map[string]interface{}:
		if kind, ok := g["kind"].(string); ok {
			return kind
		}
	}

	return ""
}

// resolveWorkspacePath resolves VSCode workspace variables in paths
func (p *TasksParser) resolveWorkspacePath(path string) string {
	// Replace common VSCode variables
	resolved := strings.ReplaceAll(path, "${workspaceFolder}", p.projectRoot)
	resolved = strings.ReplaceAll(resolved, "${workspaceRoot}", p.projectRoot)

	// Handle relative paths
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(p.projectRoot, resolved)
	}

	return resolved
}
