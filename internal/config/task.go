package config

// TaskType represents the type of task or configuration
type TaskType string

const (
	TypeVSCodeTask   TaskType = "vscode-task"
	TypeVSCodeLaunch TaskType = "vscode-launch"
	TypeJetBrains    TaskType = "jetbrains"
)

// Task represents a unified task or launch configuration
type Task struct {
	Name        string            `json:"name"`
	Type        TaskType          `json:"type"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Group       string            `json:"group,omitempty"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source"` // Path to the source configuration file
}
