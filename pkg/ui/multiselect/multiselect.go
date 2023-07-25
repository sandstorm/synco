package multiselect

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/utils/strings/slices"
	"os"
)

func Exec(question string, options []string, defaultOptions []string) []string {
	items := make([]item, 0, len(options))
	for _, option := range options {
		items = append(items, item{
			text:    option,
			checked: slices.Contains(defaultOptions, option),
		})
	}

	p := tea.NewProgram(&model{
		question: question,
		options:  items,
		item:     0,
	})

	// Run returns the model as a tea.Model.
	m, err := p.Run()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}

	mo := m.(*model)
	// Assert the final tea.Model to our local model and print the choice.
	selectedItems := make([]string, 0, len(options))
	for _, item := range mo.options {
		if item.checked {
			selectedItems = append(selectedItems, item.text)
		}
	}
	return selectedItems
}

type model struct {
	question string
	options  []item
	item     int
}

type item struct {
	text    string
	checked bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return m, m.handleKeyMsg(typed)
	}
	return m, nil
}

func (m *model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		return tea.Quit
	case "enter":
		return tea.Quit
	case " ":
		m.options[m.item].checked = !m.options[m.item].checked
		break
	case "up":
		if m.item > 0 {
			m.item--
		}
		break
	case "down":
		if m.item+1 < len(m.options) {
			m.item++
		}
		break
	}
	return nil
}

func (m *model) View() string {
	return m.renderList(m.question, m.options, m.item)
}

func (m *model) renderList(header string, items []item, selected int) string {
	out := "\n~ " + header + ":\n"
	for i, item := range items {
		sel := " "
		if i == selected {
			sel = ">"
		}
		check := " "
		if items[i].checked {
			check = "✓"
		}
		out += fmt.Sprintf("%s [%s] %s\n", sel, check, item.text)
	}
	out += "\n"
	out += "  (arrow keys / space to select, enter to continue)"
	return out
}
