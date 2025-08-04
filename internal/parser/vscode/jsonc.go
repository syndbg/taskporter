package vscode

import (
	"encoding/json"
	"strings"
)

// parseJSONC parses JSON with comments (JSONC format) commonly used by VSCode
func parseJSONC(data []byte, v interface{}) error {
	// Strip comments from the JSON data
	stripped := stripJSONComments(string(data))

	// Parse the cleaned JSON
	return json.Unmarshal([]byte(stripped), v)
}

// stripJSONComments removes both line comments (//) and block comments (/* */)
// from JSON data while preserving strings that might contain comment-like sequences
func stripJSONComments(jsonStr string) string {
	var (
		result   strings.Builder
		inString bool
		escaped  bool
	)

	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]

		// Handle escape sequences in strings
		if inString && escaped {
			result.WriteByte(char)

			escaped = false

			continue
		}

		// Handle string boundaries
		if char == '"' && !escaped {
			inString = !inString

			result.WriteByte(char)

			continue
		}

		// Handle escape character
		if inString && char == '\\' {
			escaped = true

			result.WriteByte(char)

			continue
		}

		// If we're inside a string, just copy the character
		if inString {
			result.WriteByte(char)
			continue
		}

		// Handle line comments (//)
		if char == '/' && i+1 < len(jsonStr) && jsonStr[i+1] == '/' {
			// Skip until end of line
			for i < len(jsonStr) && jsonStr[i] != '\n' && jsonStr[i] != '\r' {
				i++
			}
			// Don't increment i again at the end of the loop
			i--

			continue
		}

		// Handle block comments (/* */)
		if char == '/' && i+1 < len(jsonStr) && jsonStr[i+1] == '*' {
			// Skip until we find the closing */
			i += 2 // Skip /*
			for i+1 < len(jsonStr) {
				if jsonStr[i] == '*' && jsonStr[i+1] == '/' {
					i += 2 // Skip */
					break
				}

				i++
			}
			// Don't increment i again at the end of the loop
			i--

			continue
		}

		// Copy regular characters
		result.WriteByte(char)
	}

	return result.String()
}
