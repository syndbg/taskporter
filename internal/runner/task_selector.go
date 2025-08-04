package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/syndbg/taskporter/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the interactive selector
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DD3FC")).
			Bold(true).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Bold(true).
			MarginBottom(1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#7DD3FC")).
				Bold(true).
				Padding(0, 1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Padding(0, 1)

	sourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Italic(true)

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Background(lipgloss.Color("#064E3B")).
			Padding(0, 1).
			Bold(true)

	searchPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34D399")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginTop(1)

	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2).
			MarginTop(1)
)

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}

	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}

	if b <= c {
		return b
	}

	return c
}

// taskMatch represents a task with its relevance score
type taskMatch struct {
	task  config.Task
	score float64
}

// calculateRelevanceScore calculates a relevance score for a task name against a query
func calculateRelevanceScore(query, taskName string) float64 {
	if query == "" {
		return 1.0 // All tasks are equally relevant for empty query
	}

	queryLower := strings.ToLower(query)
	taskNameLower := strings.ToLower(taskName)

	// Exact match gets the highest score
	if queryLower == taskNameLower {
		return 1.0
	}

	// Exact substring match gets very high score
	if strings.Contains(taskNameLower, queryLower) {
		// Score based on how much of the task name the query represents
		return 0.9 * (float64(len(queryLower)) / float64(len(taskNameLower)))
	}

	// For other cases, use Levenshtein distance
	distance := levenshteinDistance(queryLower, taskNameLower)
	maxLen := max(len(queryLower), len(taskNameLower))

	if distance > maxLen {
		return 0.0 // Too different
	}

	// Convert distance to similarity score (0-1)
	similarity := 1.0 - (float64(distance) / float64(maxLen))

	// Apply threshold - only return matches with reasonable similarity
	if similarity < 0.5 {
		return 0.0
	}

	return similarity * 0.8 // Cap at 0.8 to prioritize exact/substring matches
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// fuzzyMatch performs fuzzy matching using Levenshtein distance
func fuzzyMatch(query string, taskName string) bool {
	return calculateRelevanceScore(query, taskName) > 0.0
}

// filterTasks filters tasks based on the search input using Levenshtein distance scoring
func (m *TaskSelectorModel) filterTasks() {
	if m.searchInput == "" {
		m.filteredTasks = m.tasks
		return
	}

	// Calculate relevance scores for all tasks
	var matches []taskMatch

	for _, task := range m.tasks {
		score := calculateRelevanceScore(m.searchInput, task.Name)
		if score > 0.0 {
			matches = append(matches, taskMatch{
				task:  task,
				score: score,
			})
		}
	}

	// Sort by relevance score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Extract the tasks from sorted matches
	m.filteredTasks = make([]config.Task, len(matches))
	for i, match := range matches {
		m.filteredTasks[i] = match.task
	}

	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.filteredTasks) {
		m.cursor = 0
	}
}

// TaskSelectorModel represents the Bubble Tea model for task selection
type TaskSelectorModel struct {
	tasks         []config.Task
	filteredTasks []config.Task
	cursor        int
	selected      *config.Task
	quitting      bool
	width         int
	height        int
	searchInput   string
	searchMode    bool
}

// NewTaskSelectorModel creates a new task selector model
func NewTaskSelectorModel(tasks []config.Task) *TaskSelectorModel {
	return &TaskSelectorModel{
		tasks:         tasks,
		filteredTasks: tasks, // Initially show all tasks
		cursor:        0,
		searchMode:    false,
	}
}

// Init implements the tea.Model interface
func (m *TaskSelectorModel) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (m *TaskSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case tea.KeyMsg:
		// Handle global quit commands
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Handle search mode
		if m.searchMode {
			switch msg.String() {
			case "esc":
				// Exit search mode and clear search
				m.searchMode = false
				m.searchInput = ""
				m.filterTasks()

				return m, nil

			case "enter":
				// Exit search mode and keep current filter
				if len(m.filteredTasks) > 0 {
					m.searchMode = false
					return m, nil
				}

			case "backspace":
				// Remove last character from search input
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
					m.filterTasks()
				}

			default:
				// Add character to search input (printable characters only)
				if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
					m.searchInput += msg.String()
					m.filterTasks()
				}
			}

			return m, nil
		}

		// Handle navigation mode
		switch msg.String() {
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "/":
			// Enter search mode
			m.searchMode = true
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.filteredTasks)-1 {
				m.cursor++
			}

		case "enter", " ":
			if len(m.filteredTasks) > 0 {
				m.selected = &m.filteredTasks[m.cursor]
				m.quitting = true

				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View implements the tea.Model interface
func (m *TaskSelectorModel) View() string {
	if m.quitting {
		if m.selected != nil {
			return fmt.Sprintf("🎯 Strand established! Running task: %s\n", m.selected.Name)
		}

		return "👋 Porter mission cancelled. Until next time!\n"
	}

	if len(m.tasks) == 0 {
		return containerStyle.Render(
			titleStyle.Render("🎮 Taskporter - Task Selection") + "\n\n" +
				"❌ No tasks or launch configurations found.\n" +
				"Make sure you're in a project with .vscode/ or .idea/ directories.\n\n" +
				helpStyle.Render("Press q to quit"),
		)
	}

	// Header
	var b strings.Builder
	b.WriteString(titleStyle.Render("🎮 Taskporter - Select Task to Run"))
	b.WriteString("\n")

	// Search input display
	if m.searchMode {
		searchPrompt := searchPromptStyle.Render("Search: ")
		searchInput := searchStyle.Render(m.searchInput + "█") // Add cursor
		b.WriteString(searchPrompt + searchInput + "\n")
		b.WriteString(headerStyle.Render(fmt.Sprintf("Showing %d of %d tasks", len(m.filteredTasks), len(m.tasks))))
	} else {
		if m.searchInput != "" {
			searchPrompt := searchPromptStyle.Render("Filter: ")
			searchInput := sourceStyle.Render(m.searchInput)
			b.WriteString(searchPrompt + searchInput + "\n")
			b.WriteString(headerStyle.Render(fmt.Sprintf("Showing %d of %d tasks", len(m.filteredTasks), len(m.tasks))))
		} else {
			b.WriteString(headerStyle.Render(fmt.Sprintf("Found %d configurations", len(m.tasks))))
		}
	}

	b.WriteString("\n\n")

	// Task list (using filtered tasks)
	if len(m.filteredTasks) == 0 {
		b.WriteString("🔍 No tasks match your search.\n")

		if m.searchInput != "" {
			b.WriteString("Try a different search term or press Esc to clear.\n")
		}
	} else {
		for i, task := range m.filteredTasks {
			cursor := "  "
			if i == m.cursor {
				cursor = "▶ "
			}

			// Format task line
			line := fmt.Sprintf("%s%s", cursor, task.Name)

			// Add source and type info
			source := getTaskSource(task)
			taskType := getTaskType(task)
			info := fmt.Sprintf(" [%s - %s]", source, taskType)

			if i == m.cursor {
				line = selectedItemStyle.Render(line) + sourceStyle.Render(info)
			} else {
				line = normalItemStyle.Render(line) + sourceStyle.Render(info)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Help text
	b.WriteString("\n")

	if m.searchMode {
		b.WriteString(helpStyle.Render("Type to search • Enter: Exit search • Esc: Clear search • Ctrl+C: Quit"))
	} else {
		b.WriteString(helpStyle.Render("↑/↓ Navigate • Enter: Run Task • /: Search • q: Quit"))
	}

	return containerStyle.Render(b.String())
}

// getTaskSource returns a human-readable source for the task
func getTaskSource(task config.Task) string {
	switch task.Source {
	case "vscode-tasks":
		return "VSCode Task"
	case "vscode-launch":
		return "VSCode Launch"
	case "jetbrains":
		return "JetBrains"
	default:
		return task.Source
	}
}

// getTaskType returns a human-readable type for the task
func getTaskType(task config.Task) string {
	if task.Group != "" {
		return task.Group
	}

	switch task.Source {
	case "vscode-launch":
		return "launch"
	case "jetbrains":
		return "run"
	default:
		return "task"
	}
}

// RunInteractiveTaskSelector runs the interactive task selector and returns the selected task
func RunInteractiveTaskSelector(tasks []config.Task) (*config.Task, error) {
	model := NewTaskSelectorModel(tasks)
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run interactive selector: %w", err)
	}

	selectorModel := finalModel.(*TaskSelectorModel)

	return selectorModel.selected, nil
}
