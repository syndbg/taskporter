package config

// ProjectConfig represents the overall project configuration
type ProjectConfig struct {
	ProjectRoot  string  `json:"project_root"`
	Tasks        []*Task `json:"tasks"`
	HasVSCode    bool    `json:"has_vscode"`
	HasJetBrains bool    `json:"has_jetbrains"`
}
