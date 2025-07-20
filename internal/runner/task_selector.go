package runner

import (
	"fmt"
	"strings"

	"taskporter/internal/config"

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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginTop(1)

	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2).
			MarginTop(1)
)

// TaskSelectorModel represents the Bubble Tea model for task selection
type TaskSelectorModel struct {
	tasks    []config.Task
	cursor   int
	selected *config.Task
	quitting bool
	width    int
	height   int
}

// NewTaskSelectorModel creates a new task selector model
func NewTaskSelectorModel(tasks []config.Task) *TaskSelectorModel {
	return &TaskSelectorModel{
		tasks:  tasks,
		cursor: 0,
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
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}

		case "enter", " ":
			if len(m.tasks) > 0 {
				m.selected = &m.tasks[m.cursor]
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
	b.WriteString(headerStyle.Render(fmt.Sprintf("Found %d configurations", len(m.tasks))))
	b.WriteString("\n\n")

	// Task list
	for i, task := range m.tasks {
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

	// Help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ Navigate • Enter: Run Task • q: Quit"))

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
