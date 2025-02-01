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

	singleButton bool

	styles styles
}

// TODO keep this with just a single option or add use argument to newDialogModel
// or are there any other options we would want?
type dialogOption func(*dialogModel)

func dialogOptionWithSingleButton() dialogOption {
	return func(dm *dialogModel) {
		dm.singleButton = true
	}
}

// TODO rename to yesnodialog? or something similar?
func newDialogModel(styles styles, options ...dialogOption) *dialogModel {
	help := help.New()
	help.Styles = styles.HelpStyles
	help.ShowAll = true

	dm := &dialogModel{
		keyMap:       defaultDialogKeyMap(),
		helpView:     help,
		singleButton: false,
		styles:       styles,
	}
	for _, opt := range options {
		opt(dm)
	}

	if dm.singleButton {
		dm.keyMap.No.SetEnabled(false)
	}
	return dm
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
			if !m.singleButton {
				cmd = returnFromDialog(false)
			}
		}
	}
	return cmd
}

// TODO can we maybe do this better? options? but this is also fine
// there has to be a better way to do all this with yesText, noText and co
// the problem is basically we want to probably only define the text when we render not when we create the model?
func (m *dialogModel) View(content, yesText, noText string) string {
	keyMap := m.keyMap
	keyMap.Yes.SetHelp(keyMap.Yes.Help().Key, yesText)
	keyMap.No.SetHelp(keyMap.No.Help().Key, noText)

	return fmt.Sprintf(
		"%s\n\n%s\n",
		content,
		m.helpView.View(keyMap),
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
