package backupdir

import (
	"backup/styles"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	textInput  textinput.Model
	inputValid bool
	backupDir  string
	keyMap     keyMap
	styles     styles.Styles
	help       help.Model
}

func NewModel(backupDir string, styles styles.Styles) *Model {
	bt := textinput.New()
	bt.Placeholder = backupDir
	bt.CharLimit = 250
	bt.Width = 40
	bt.Focus()

	help := help.New()
	help.Styles = styles.HelpStyles
	help.ShowAll = true

	return &Model{
		textInput:  bt,
		inputValid: true,
		backupDir:  backupDir,
		keyMap:     defaultKeyMap(),
		styles:     styles,
		help:       help,
	}
}

func (m *Model) KeyMap() help.KeyMap {
	return m.keyMap
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Confirm):
			dir := m.textInput.Value()
			if dir == "" {
				m.inputValid = false
			} else {
				m.backupDir = dir
				m.inputValid = true
				cmd = returnBackupDir(dir)
			}
		case key.Matches(msg, m.keyMap.Cancel):
			cmd = done()
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	default:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) View() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n",
		m.styles.TitleStyle.Render("Change Backup Directory"),
		m.textInput.View(),
	)
	if !m.inputValid {
		content = fmt.Sprintf("%s\n%s\n", content, m.styles.ErrorTextStyle.Render("invalid directory"))
	}
	content = fmt.Sprintf("%s\n%s\n", content, m.help.View(m.keyMap))
	return content
}

type BackupDirChanged struct {
	BackupDir string
}

func returnBackupDir(backupDir string) tea.Cmd {
	return func() tea.Msg {
		return BackupDirChanged{BackupDir: backupDir}
	}
}

type Done struct{}

func done() tea.Cmd {
	return func() tea.Msg {
		return Done{}
	}
}

type keyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func (m keyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Cancel, m.Confirm}
}

func (m keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.Cancel}, {m.Confirm}}
}
