package textinput

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"fmt"
	"github.com/pterm/pterm"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func Exec(question string) string {
	p := tea.NewProgram(initialModel(question))
	m, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}
	mCasted := m.(model)
	return mCasted.textInput.Value()
}

type (
	errMsg error
)

type model struct {
	question  string
	textInput textinput.Model
	err       error
}

func initialModel(question string) model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return model{
		question:  question,
		textInput: ti,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			pterm.Warning.Println("Exiting")
			os.Exit(1)
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.question,
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}
