package internal

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type dialogModel struct {
	keyMap dialogKeyMap

	helpView help.Model

	styles styles
}

// TODO rename to yesnodialog?
func newDialogModel(styles styles) *dialogModel {
	help := help.New()
	help.Styles = styles.HelpStyles
	help.ShowAll = true

	return &dialogModel{
		keyMap:   defaultDialogKeyMap(),
		helpView: help,
		styles:   styles,
	}
}

func (m *dialogModel) Init() tea.Cmd {
	return nil
}

func (m *dialogModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Yes):
			cmd = returnFromDialog(true)
		case key.Matches(msg, m.keyMap.No):
			cmd = returnFromDialog(false)
		}
	}
	return cmd
}

func (m *dialogModel) View(content string) string {
	return fmt.Sprintf(
		"%s\n\n%s\n",
		content,
		m.helpView.View(m.keyMap),
	)
}

type dialogDone struct {
	confirmed bool
}

func returnFromDialog(confirmed bool) tea.Cmd {
	return func() tea.Msg {
		return dialogDone{confirmed: confirmed}
	}
}

type dialogKeyMap struct {
	Yes key.Binding
	No  key.Binding
}

func defaultDialogKeyMap() dialogKeyMap {
	return dialogKeyMap{
		Yes: key.NewBinding(
			key.WithKeys("y", "enter"),
			key.WithHelp("y/enter", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/esc", "no"),
		),
	}
}

func (m dialogKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Yes, m.No}
}

func (m dialogKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Yes, m.No},
	}
}
