package internal

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type dirSelectModel struct {
	commonState       *commonState
	inputValid        bool
	inputError        error
	showWarningDialog bool
	warning           int

	textInput     textinput.Model
	warningDialog *dialogModel
	keyMap        dirSelectKeyMap
	helpView      help.Model
}

const (
	warningDirNotEmpty = iota
	warningParentNotExists
)

func newDirSelectModel(commonState *commonState) *dirSelectModel {
	bt := textinput.New()
	bt.Placeholder = commonState.backupDir
	bt.CharLimit = 250
	bt.Width = 40
	bt.Focus()

	helpView := help.New()
	helpView.Styles = commonState.styles.HelpStyles
	helpView.ShowAll = true

	return &dirSelectModel{
		commonState:       commonState,
		inputValid:        true,
		inputError:        nil,
		showWarningDialog: false,

		textInput:     bt,
		warningDialog: nil,
		keyMap:        defaultDirSelectKeyMap(),
		helpView:      helpView,
	}
}

func (m *dirSelectModel) Init() tea.Cmd {
	// TODO need to call init on Help?
	return textinput.Blink
}

func (m *dirSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if !m.showWarningDialog {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.Confirm):
				path := m.textInput.Value()
				if path == "" {
					m.inputValid = false
					m.inputError = errors.New("path cannot be empty")
				} else if absPath, err := getAbsPath(path); err != nil {
					m.inputValid = false
					m.inputError = err
				} else {
					empty, err := isDirEmpty(absPath)
					if err != nil {
						m.inputValid = false
						m.inputError = err
					} else if empty {
						m.inputValid = true
						exists, err := dirExists(getParentPath(absPath))
						if err != nil {
							m.inputValid = false
							m.inputError = err
						} else if !exists {
							m.showWarningDialog = true
							m.warning = warningParentNotExists
							m.warningDialog = newDialogModel(m.commonState.styles)
						} else {
							m.inputValid = true
							// TODO is it possible that this command gets run async and that in the meantime the user can press enter again and cause the same thing again?
							// wouldn't really be a big problem but still
							// to avoid this we could use a separate finished state that we switch into here that just does nothing with a msg
							// of course how likely is that on a modern machine to happen? can you even press that quickly?
							cmd = returnBackupDir(absPath)
						}
					} else {
						// TODO factor this out into a common method
						// probably want to refactor this whole code anyway all these if elses are kinda ugly, what is a better way to do it?
						m.showWarningDialog = true
						m.warning = warningDirNotEmpty
						m.warningDialog = newDialogModel(m.commonState.styles)
					}
				}
			case key.Matches(msg, m.keyMap.Cancel):
				cmd = returnBackupDir("")
			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	} else {
		switch msg := msg.(type) {
		case dialogDone:
			if msg.confirmed {
				// TODO don't recompute abs path again, need to store it above
				absPath, _ := getAbsPath(m.textInput.Value())
				cmd = returnBackupDir(absPath)
			} else {
				m.showWarningDialog = false
				m.warningDialog = nil
			}
		default:
			cmd = m.warningDialog.Update(msg)
		}
	}
	return m, cmd
}

func (m *dirSelectModel) View() string {
	styles := m.commonState.styles
	if m.showWarningDialog {
		// TODO better text here pls
		// TODO probably we should show the currently selected abs path also here -> create a field on model
		var warningText string
		if m.warning == warningDirNotEmpty {
			warningText = "selected directory is not empty, if you continue files may be overwritten"
		} else if m.warning == warningParentNotExists {
			// TODO maybe don't have the might be a typo part? because sometimes you might want to backup in a subfolder of a new folder
			// or keep it but formulate it differently, you sure you didn't mistype?
			warningText = "parent directory does not exist, might be a typo"
		} else {
			// TODO this shouldn't happen, just do the above in the else clause here?
		}
		content := styles.ErrorTextStyle.Render(fmt.Sprintf("Warning: %s\nDo you want to continue?", warningText))
		return m.warningDialog.View(content)
	} else {
		content := fmt.Sprintf(
			"%s\n\n%s\n%s\n",
			styles.TitleStyle.Render("Change Backup Directory"),
			styles.NormalTextStyle.Render(fmt.Sprintf("Current: %s", m.commonState.backupDir)),
			m.textInput.View(),
		)
		if !m.inputValid {
			content = fmt.Sprintf("%s\n%s\n", content, styles.ErrorTextStyle.Render(m.inputError.Error()))
		}
		content = fmt.Sprintf("%s\n%s\n", content, m.helpView.View(m.keyMap))
		return content
	}
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
