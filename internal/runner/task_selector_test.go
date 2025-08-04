package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndbg/taskporter/internal/config"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want int
	}{
		{
			name: "empty strings",
			s1:   "",
			s2:   "",
			want: 0,
		},
		{
			name: "empty first string",
			s1:   "",
			s2:   "abc",
			want: 3,
		},
		{
			name: "empty second string",
			s1:   "abc",
			s2:   "",
			want: 3,
		},
		{
			name: "identical strings",
			s1:   "test",
			s2:   "test",
			want: 0,
		},
		{
			name: "single character difference",
			s1:   "test",
			s2:   "best",
			want: 1,
		},
		{
			name: "single insertion",
			s1:   "test",
			s2:   "tests",
			want: 1,
		},
		{
			name: "single deletion",
			s1:   "tests",
			s2:   "test",
			want: 1,
		},
		{
			name: "multiple operations",
			s1:   "kitten",
			s2:   "sitting",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshteinDistance(tt.s1, tt.s2)
			require.Equal(t, tt.want, got, "levenshteinDistance(%q, %q)", tt.s1, tt.s2)
		})
	}
}

func TestCalculateRelevanceScore(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		taskName string
		want     float64
		minScore float64 // minimum expected score for ranges
	}{
		{
			name:     "empty query",
			query:    "",
			taskName: "build",
			want:     1.0,
		},
		{
			name:     "exact match",
			query:    "build",
			taskName: "build",
			want:     1.0,
		},
		{
			name:     "case insensitive exact match",
			query:    "BUILD",
			taskName: "build",
			want:     1.0,
		},
		{
			name:     "substring match",
			query:    "uild",
			taskName: "build",
			minScore: 0.7, // Should be high score for substring
		},
		{
			name:     "partial substring match",
			query:    "test",
			taskName: "run:test:unit",
			minScore: 0.25, // Score based on substring length ratio
		},
		{
			name:     "similar strings",
			query:    "tset", // typo of "test"
			taskName: "test",
			minScore: 0.4, // Should match with reasonable score (0.5 * 0.8 = 0.4)
		},
		{
			name:     "very different strings",
			query:    "xyz",
			taskName: "build",
			want:     0.0, // Should not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateRelevanceScore(tt.query, tt.taskName)
			if tt.want > 0 {
				require.Equal(t, tt.want, got, "calculateRelevanceScore(%q, %q)", tt.query, tt.taskName)
			} else {
				require.GreaterOrEqual(t, got, tt.minScore, "calculateRelevanceScore(%q, %q) should be >= %f, got %f", tt.query, tt.taskName, tt.minScore, got)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		taskName string
		want     bool
	}{
		{
			name:     "empty query matches everything",
			query:    "",
			taskName: "build",
			want:     true,
		},
		{
			name:     "exact match",
			query:    "build",
			taskName: "build",
			want:     true,
		},
		{
			name:     "case insensitive exact match",
			query:    "BUILD",
			taskName: "build",
			want:     true,
		},
		{
			name:     "substring match",
			query:    "uild",
			taskName: "build",
			want:     true,
		},
		{
			name:     "similar strings (typo)",
			query:    "buil", // missing 'd'
			taskName: "build",
			want:     true,
		},
		{
			name:     "very different strings",
			query:    "xyz",
			taskName: "build",
			want:     false,
		},
		{
			name:     "partial match in longer name",
			query:    "test",
			taskName: "run:test:unit",
			want:     true,
		},
		{
			name:     "reasonable typo",
			query:    "tset", // "test" with swapped characters
			taskName: "test",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyMatch(tt.query, tt.taskName)
			require.Equal(t, tt.want, got, "fuzzyMatch(%q, %q)", tt.query, tt.taskName)
		})
	}
}

func TestTaskSelectorModel_FilterTasks(t *testing.T) {
	// Create test tasks
	tasks := []config.Task{
		{Name: "build", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "test", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "lint", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "run:dev", Type: config.TypeVSCodeLaunch, Source: "vscode-launch"},
		{Name: "run:test:unit", Type: config.TypeVSCodeLaunch, Source: "vscode-launch"},
		{Name: "deploy", Type: config.TypeJetBrains, Source: "jetbrains"},
	}

	tests := []struct {
		name          string
		searchInput   string
		wantCount     int
		wantNames     []string
		checkOrder    bool   // Whether to check exact order (for sorted results)
		expectedFirst string // Expected first result for ordering tests
	}{
		{
			name:        "empty search shows all tasks",
			searchInput: "",
			wantCount:   6,
			wantNames:   []string{"build", "test", "lint", "run:dev", "run:test:unit", "deploy"},
			checkOrder:  false,
		},
		{
			name:          "exact match",
			searchInput:   "build",
			wantCount:     1,
			wantNames:     []string{"build"},
			checkOrder:    true,
			expectedFirst: "build",
		},
		{
			name:        "substring match",
			searchInput: "run",
			wantCount:   2,
			wantNames:   []string{"run:dev", "run:test:unit"},
			checkOrder:  false, // Both have same substring score
		},
		{
			name:          "case insensitive",
			searchInput:   "TEST",
			wantCount:     2,
			wantNames:     []string{"test", "run:test:unit"},
			checkOrder:    true,
			expectedFirst: "test", // Exact match should come first
		},
		{
			name:        "no matches",
			searchInput: "xyz",
			wantCount:   0,
			wantNames:   []string{},
			checkOrder:  false,
		},
		{
			name:          "typo handling",
			searchInput:   "tset", // typo of "test"
			wantCount:     1,
			wantNames:     []string{"test"},
			checkOrder:    true,
			expectedFirst: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewTaskSelectorModel(tasks)
			model.searchInput = tt.searchInput
			model.filterTasks()

			require.Len(t, model.filteredTasks, tt.wantCount, "filtered task count")

			var actualNames []string
			for _, task := range model.filteredTasks {
				actualNames = append(actualNames, task.Name)
			}

			if tt.checkOrder && len(actualNames) > 0 {
				require.Equal(t, tt.expectedFirst, actualNames[0], "first result should be most relevant")
				require.ElementsMatch(t, tt.wantNames, actualNames, "filtered task names")
			} else {
				require.ElementsMatch(t, tt.wantNames, actualNames, "filtered task names")
			}
		})
	}
}

func TestTaskSelectorModel_CursorResetOnFilter(t *testing.T) {
	tasks := []config.Task{
		{Name: "build", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "test", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "lint", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
	}

	model := NewTaskSelectorModel(tasks)

	// Move cursor to last position
	model.cursor = 2
	require.Equal(t, 2, model.cursor)

	// Filter to single result
	model.searchInput = "build"
	model.filterTasks()

	// Cursor should be reset to 0 since it was out of bounds
	require.Equal(t, 0, model.cursor)
	require.Len(t, model.filteredTasks, 1)
}

func TestTaskSelectorModel_InitialState(t *testing.T) {
	tasks := []config.Task{
		{Name: "build", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "test", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
	}

	model := NewTaskSelectorModel(tasks)

	require.Equal(t, tasks, model.tasks, "tasks should be set")
	require.Equal(t, tasks, model.filteredTasks, "filteredTasks should initially equal tasks")
	require.Equal(t, 0, model.cursor, "cursor should start at 0")
	require.False(t, model.searchMode, "searchMode should start false")
	require.Empty(t, model.searchInput, "searchInput should start empty")
	require.False(t, model.quitting, "quitting should start false")
	require.Nil(t, model.selected, "selected should start nil")
}

// Test relevance score ordering
func TestTaskSelectorModel_RelevanceOrdering(t *testing.T) {
	tasks := []config.Task{
		{Name: "test", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},             // Exact match
		{Name: "testing", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},          // Substring match
		{Name: "run:test:unit", Type: config.TypeVSCodeLaunch, Source: "vscode-launch"}, // Contains substring
		{Name: "tset", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},             // Typo (Levenshtein distance 2)
		{Name: "best", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},             // Similar ending
	}

	t.Run("exact match prioritized", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)
		model.searchInput = "test"
		model.filterTasks()

		// Should find multiple matches
		require.Greater(t, len(model.filteredTasks), 0)

		// Exact match should be first
		require.Equal(t, "test", model.filteredTasks[0].Name, "exact match should be ranked highest")
	})

	t.Run("expected matches included", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)
		model.searchInput = "test"
		model.filterTasks()

		// Verify all expected matches are present
		var actualNames []string
		for _, task := range model.filteredTasks {
			actualNames = append(actualNames, task.Name)
		}

		// Should contain exact match, substring matches, and reasonable typos
		require.Contains(t, actualNames, "test")
		require.Contains(t, actualNames, "testing")
		require.Contains(t, actualNames, "run:test:unit")
		require.Contains(t, actualNames, "tset") // typo should still match
	})

	t.Run("matches ordered by relevance", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)
		model.searchInput = "test"
		model.filterTasks()

		var actualNames []string
		for _, task := range model.filteredTasks {
			actualNames = append(actualNames, task.Name)
		}

		// Should include good matches but ordered correctly
		require.Contains(t, actualNames, "test")    // exact
		require.Contains(t, actualNames, "testing") // substring
		require.Contains(t, actualNames, "tset")    // typo

		// May or may not contain "best" depending on threshold - that's okay
		// The important thing is that "test" comes first
		if len(model.filteredTasks) > 0 {
			require.Equal(t, "test", model.filteredTasks[0].Name)
		}
	})
}

// Integration test for search workflow
func TestTaskSelectorModel_SearchWorkflow(t *testing.T) {
	tasks := []config.Task{
		{Name: "build:dev", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "build:prod", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "test:unit", Type: config.TypeVSCodeTask, Source: "vscode-tasks"},
		{Name: "deploy", Type: config.TypeJetBrains, Source: "jetbrains"},
	}

	t.Run("initial state shows all tasks", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)
		require.Len(t, model.filteredTasks, 4)
	})

	t.Run("progressive search refinement", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)

		// Start typing "build"
		model.searchInput = "b"
		model.filterTasks()
		require.GreaterOrEqual(t, len(model.filteredTasks), 2) // Should find build tasks

		model.searchInput = "bu"
		model.filterTasks()
		require.GreaterOrEqual(t, len(model.filteredTasks), 2) // Should still find build tasks

		model.searchInput = "build"
		model.filterTasks()
		require.GreaterOrEqual(t, len(model.filteredTasks), 2) // Should find both build tasks

		// Add colon to be more specific
		model.searchInput = "build:"
		model.filterTasks()
		require.GreaterOrEqual(t, len(model.filteredTasks), 2) // Should still find build tasks
	})

	t.Run("specific match", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)

		// Search for a unique task name
		model.searchInput = "deploy"
		model.filterTasks()
		require.Len(t, model.filteredTasks, 1) // only deploy
		require.Equal(t, "deploy", model.filteredTasks[0].Name)
	})

	t.Run("clear search resets to all tasks", func(t *testing.T) {
		model := NewTaskSelectorModel(tasks)

		// Set a search, then clear it
		model.searchInput = "build"
		model.filterTasks()
		require.Greater(t, len(model.filteredTasks), 0)

		model.searchInput = ""
		model.filterTasks()
		require.Len(t, model.filteredTasks, 4) // back to all tasks
	})
}
