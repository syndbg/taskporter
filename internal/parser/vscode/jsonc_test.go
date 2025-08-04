package vscode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripJSONComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no comments",
			input:    `{"name": "test", "value": 123}`,
			expected: `{"name": "test", "value": 123}`,
		},
		{
			name: "line comment at end",
			input: `{
				"name": "test", // This is a comment
				"value": 123
			}`,
			expected: `{
				"name": "test", 
				"value": 123
			}`,
		},
		{
			name: "line comment on separate line",
			input: `{
				"name": "test",
				// This is a comment line
				"value": 123
			}`,
			expected: `{
				"name": "test",
				
				"value": 123
			}`,
		},
		{
			name: "block comment",
			input: `{
				"name": "test", /* block comment */
				"value": 123
			}`,
			expected: `{
				"name": "test", 
				"value": 123
			}`,
		},
		{
			name: "multiline block comment",
			input: `{
				"name": "test",
				/* This is a
				   multiline
				   comment */
				"value": 123
			}`,
			expected: `{
				"name": "test",
				
				"value": 123
			}`,
		},
		{
			name: "comment-like strings should be preserved",
			input: `{
				"name": "test // not a comment",
				"url": "http://example.com",
				"note": "/* not a comment */"
			}`,
			expected: `{
				"name": "test // not a comment",
				"url": "http://example.com",
				"note": "/* not a comment */"
			}`,
		},
		{
			name: "escaped quotes in strings",
			input: `{
				"name": "test \"quoted\" // not a comment",
				"value": 123 // actual comment
			}`,
			expected: `{
				"name": "test \"quoted\" // not a comment",
				"value": 123 
			}`,
		},
		{
			name: "mixed comments",
			input: `{
				// Line comment at start
				"name": "test", /* inline block */
				"value": 123, // line comment
				/* Another block comment */
				"enabled": true
			}`,
			expected: `{
				
				"name": "test", 
				"value": 123, 
				
				"enabled": true
			}`,
		},
		{
			name: "comments in array",
			input: `{
				"items": [
					"first", // comment 1
					"second", /* comment 2 */
					"third"
				]
			}`,
			expected: `{
				"items": [
					"first", 
					"second", 
					"third"
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripJSONComments(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSONC(t *testing.T) {
	t.Run("parse valid JSONC with comments", func(t *testing.T) {
		jsonc := `{
			// This is a configuration file
			"version": "0.2.0",
			"configurations": [
				{
					"name": "Launch Program", // Launch configuration
					"type": "go",
					"request": "launch",
					/* Multi-line comment
					   about the program */
					"program": "${workspaceFolder}",
					"args": ["--verbose"] // Command arguments
				}
			]
		}`

		var result struct {
			Version        string `json:"version"`
			Configurations []struct {
				Name    string   `json:"name"`
				Type    string   `json:"type"`
				Request string   `json:"request"`
				Program string   `json:"program"`
				Args    []string `json:"args"`
			} `json:"configurations"`
		}

		err := parseJSONC([]byte(jsonc), &result)
		require.NoError(t, err)

		require.Equal(t, "0.2.0", result.Version)
		require.Len(t, result.Configurations, 1)
		require.Equal(t, "Launch Program", result.Configurations[0].Name)
		require.Equal(t, "go", result.Configurations[0].Type)
		require.Equal(t, "launch", result.Configurations[0].Request)
		require.Equal(t, "${workspaceFolder}", result.Configurations[0].Program)
		require.Equal(t, []string{"--verbose"}, result.Configurations[0].Args)
	})

	t.Run("parse VSCode tasks.json with comments", func(t *testing.T) {
		jsonc := `{
			// VSCode tasks configuration
			"version": "2.0.0",
			"tasks": [
				{
					"label": "build", // Build task
					"type": "shell",
					"command": "go",
					"args": ["build", "-o", "bin/app"], /* Build arguments */
					"group": "build"
				},
				// Test task
				{
					"label": "test",
					"type": "shell",
					"command": "go",
					"args": ["test", "./..."],
					"group": "test"
				}
			]
		}`

		var result struct {
			Version string `json:"version"`
			Tasks   []struct {
				Label   string   `json:"label"`
				Type    string   `json:"type"`
				Command string   `json:"command"`
				Args    []string `json:"args"`
				Group   string   `json:"group"`
			} `json:"tasks"`
		}

		err := parseJSONC([]byte(jsonc), &result)
		require.NoError(t, err)

		require.Equal(t, "2.0.0", result.Version)
		require.Len(t, result.Tasks, 2)

		// Check build task
		require.Equal(t, "build", result.Tasks[0].Label)
		require.Equal(t, "shell", result.Tasks[0].Type)
		require.Equal(t, "go", result.Tasks[0].Command)
		require.Equal(t, []string{"build", "-o", "bin/app"}, result.Tasks[0].Args)
		require.Equal(t, "build", result.Tasks[0].Group)

		// Check test task
		require.Equal(t, "test", result.Tasks[1].Label)
		require.Equal(t, "go", result.Tasks[1].Command)
		require.Equal(t, []string{"test", "./..."}, result.Tasks[1].Args)
	})

	t.Run("parse JSON without comments should still work", func(t *testing.T) {
		regularJSON := `{
			"version": "0.2.0",
			"configurations": [
				{
					"name": "Launch Program",
					"type": "go",
					"request": "launch"
				}
			]
		}`

		var result struct {
			Version        string `json:"version"`
			Configurations []struct {
				Name    string `json:"name"`
				Type    string `json:"type"`
				Request string `json:"request"`
			} `json:"configurations"`
		}

		err := parseJSONC([]byte(regularJSON), &result)
		require.NoError(t, err)

		require.Equal(t, "0.2.0", result.Version)
		require.Len(t, result.Configurations, 1)
		require.Equal(t, "Launch Program", result.Configurations[0].Name)
	})

	t.Run("invalid JSON should return error", func(t *testing.T) {
		invalidJSON := `{
			// This is invalid JSON
			"name": "test",
			"invalid": // missing value
		}`

		var result map[string]interface{}

		err := parseJSONC([]byte(invalidJSON), &result)
		require.Error(t, err)
	})
}
