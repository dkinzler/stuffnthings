package dialog

import (
	"backup/styles"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	KeyMap KeyMap

	help help.Model

	styles styles.Styles
}

func NewModel(styles styles.Styles) *Model {
	help := help.New()
	help.Styles = styles.HelpStyles
	help.ShowAll = true

	return &Model{
		KeyMap: DefaultKeyMap(),
		help:   help,
		styles: styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Yes):
			cmd = done(true)
		case key.Matches(msg, m.KeyMap.No):
			cmd = done(false)
		}
	}
	return cmd
}

func (m *Model) View(text string) string {
	return fmt.Sprintf(
		"%s\n\n%s\n",
		m.styles.NormalTextStyle.Render(text),
		m.help.View(m.KeyMap),
	)
}

type DialogResult struct {
	Confirmed bool
}

func done(confirmed bool) tea.Cmd {
	return func() tea.Msg {
		return DialogResult{Confirmed: confirmed}
	}
}

type KeyMap struct {
	Yes key.Binding
	No  key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
	}
}

func (m KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Yes, m.No}
}

func (m KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Yes, m.No},
	}
}
