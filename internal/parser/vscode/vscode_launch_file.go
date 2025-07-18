package vscode

// VSCodeLaunchFile represents the structure of VSCode launch.json
type VSCodeLaunchFile struct {
	Version        string               `json:"version"`
	Configurations []VSCodeLaunchConfig `json:"configurations"`
}
