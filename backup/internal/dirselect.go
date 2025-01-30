package internal

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type dirSelectModel struct {
	commonState *commonState

	textInput  textinput.Model
	filepicker filepicker.Model
	inputValid bool
	keyMap     dirSelectKeyMap
	helpView   help.Model
}

func newDirSelectModel(commonState *commonState) *dirSelectModel {
	bt := textinput.New()
	bt.Placeholder = commonState.backupDir
	bt.CharLimit = 250
	bt.Width = 40
	bt.Focus()

	fp := filepicker.New()
	fp.AllowedTypes = []string{}
	// TODO handle this error? -> if that happens probably just panic
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.ShowSize = false
	// TODO keep this?
	fp.ShowPermissions = false
	fp.Height = 15
	// TODO need to set height dynamically in SetSize whenever ic hanges

	help := help.New()
	help.Styles = commonState.styles.HelpStyles
	help.ShowAll = true

	return &dirSelectModel{
		commonState: commonState,
		textInput:   bt,
		filepicker:  fp,
		inputValid:  true,
		keyMap:      defaultDirSelectKeyMap(),
		helpView:    help,
	}
}

func (m *dirSelectModel) Init() tea.Cmd {
	// TODO need to call init on Help?
	// return textinput.Blink
	return m.filepicker.Init()
}

func (m *dirSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// switch msg := msg.(type) {
	// case tea.KeyMsg:
	// 	switch {
	// 	case key.Matches(msg, m.keyMap.Confirm):
	// 		dir := m.textInput.Value()
	// 		if dir == "" {
	// 			m.inputValid = false
	// 		} else {
	// 			m.inputValid = true
	// 			cmd = returnBackupDir(dir)
	// 		}
	// 	case key.Matches(msg, m.keyMap.Cancel):
	// 		cmd = returnBackupDir("")
	// 	default:
	// 		m.textInput, cmd = m.textInput.Update(msg)
	// 	}
	// default:
	// 	m.textInput, cmd = m.textInput.Update(msg)
	// }
	m.filepicker, cmd = m.filepicker.Update(msg)
	return m, cmd
}

func (m *dirSelectModel) View() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n",
		m.commonState.styles.TitleStyle.Render("Change Backup Directory"),
		// m.textInput.View(),
		m.filepicker.View(),
	)
	// if !m.inputValid {
	// 	content = fmt.Sprintf("%s\n%s\n", content, m.commonState.styles.ErrorTextStyle.Render("invalid directory"))
	// }
	// content = fmt.Sprintf("%s\n%s\n", content, m.helpView.View(m.keyMap))
	// TODO use custom keymap -> yes have to to show help, need to copy binds over or best set up our own keymap from the start and update the one in filepicker
	// content = fmt.Sprintf("%s\n%s\n", content, m.helpView.View(m.filepickerkeyMap))
	return content
}

type dirSelectDone struct {
	// empty if backup directory was not changed
	backupDir string
}

func returnBackupDir(backupDir string) tea.Cmd {
	return func() tea.Msg {
		return dirSelectDone{backupDir: backupDir}
	}
}

type dirSelectKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func defaultDirSelectKeyMap() dirSelectKeyMap {
	return dirSelectKeyMap{
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

func (m dirSelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Cancel, m.Confirm}
}

func (m dirSelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.Cancel}, {m.Confirm}}
}
