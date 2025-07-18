package vscode

// VSCodeLaunchConfig represents a single launch configuration in VSCode launch.json
type VSCodeLaunchConfig struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Request       string            `json:"request"`
	Mode          string            `json:"mode,omitempty"`
	Program       string            `json:"program,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Cwd           string            `json:"cwd,omitempty"`
	Console       string            `json:"console,omitempty"`
	StopOnEntry   bool              `json:"stopOnEntry,omitempty"`
	JustMyCode    bool              `json:"justMyCode,omitempty"`
	PreLaunchTask string            `json:"preLaunchTask,omitempty"`
	ProcessId     interface{}       `json:"processId,omitempty"`
}
