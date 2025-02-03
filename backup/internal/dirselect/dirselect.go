package dirselect

import (
	"backup/internal/fs"
	"backup/internal/style"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateInput state = iota
	stateWarning
)

const (
	warningDirNotEmpty = iota
	warningParentNotExists
)

type Model struct {
	state      state
	backupDir  string
	inputValid bool
	inputError error
	warning    int

	textInput textinput.Model
	keyMap    keyMap
	helpView  help.Model
	styles    style.Styles
}

func NewModel(backupDir string, styles style.Styles) *Model {
	bt := textinput.New()
	bt.Placeholder = backupDir
	bt.CharLimit = 250
	bt.Width = 40
	bt.Focus()

	helpView := help.New()
	helpView.Styles = styles.HelpStyles
	helpView.ShowAll = true

	return &Model{
		backupDir:  backupDir,
		state:      stateInput,
		inputValid: true,
		inputError: nil,
		warning:    -1,

		textInput: bt,
		keyMap:    defaultKeyMap(),
		helpView:  helpView,

		styles: styles,
	}
}

func (m *Model) Init() tea.Cmd {
	// TODO need to call init on Help?
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case stateInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.confirm):
				path := m.textInput.Value()
				if path == "" {
					m.inputValid = false
					m.inputError = errors.New("path cannot be empty")
				} else if absPath, err := fs.GetAbsPath(path); err != nil {
					m.inputValid = false
					m.inputError = err
				} else {
					empty, err := fs.IsDirEmpty(absPath)
					if err != nil {
						m.inputValid = false
						m.inputError = err
					} else if empty {
						exists, err := fs.DirExists(fs.GetParentPath(absPath))
						if err != nil {
							m.inputValid = false
							m.inputError = err
						} else if !exists {
							m.inputValid = true
							m.inputError = nil
							m.state = stateWarning
							m.warning = warningParentNotExists
						} else {
							m.inputValid = true
							m.inputError = nil
							// TODO is it possible that this command gets run async and that in the meantime the user can press enter again and cause the same thing again?
							// wouldn't really be a big problem but still
							// to avoid this we could use a separate finished state that we switch into here that just does nothing with a msg
							// of course how likely is that on a modern machine to happen? can you even press that quickly?
							cmd = returnBackupDir(absPath)
						}
					} else {
						// TODO factor this out into a common method
						// probably want to refactor this whole code anyway all these if elses are kinda ugly, what is a better way to do it?
						m.state = stateWarning
						m.warning = warningDirNotEmpty
					}
				}
			case key.Matches(msg, m.keyMap.cancel):
				cmd = returnBackupDir("")
			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	case stateWarning:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.warningConfirm):
				// TODO don't recompute abs path again, need to store it above
				absPath, _ := fs.GetAbsPath(m.textInput.Value())
				cmd = returnBackupDir(absPath)
			case key.Matches(msg, m.keyMap.warningCancel):
				m.state = stateInput
			}
		}
	}
	return m, cmd
}

func (m *Model) View() string {
	// TODO a common type of rendering is with a title content and help at the bottom
	// maybe make a function for that, so we don't have to repeat the same pattern over and over?
	styles := m.styles
	content := ""

	switch m.state {
	case stateInput:
		content = fmt.Sprintf(
			"%s\n\n%s\n%s\n",
			styles.TitleStyle.Render("Change Backup Directory"),
			styles.NormalTextStyle.Render(fmt.Sprintf("Current: %s", m.backupDir)),
			m.textInput.View(),
		)
		if !m.inputValid {
			content = fmt.Sprintf("%s\n%s\n", content, styles.ErrorTextStyle.Render(m.inputError.Error()))
		}
		content = fmt.Sprintf("%s\n%s\n", content, m.helpView.ShortHelpView(m.keyMap.inputKeys()))
	case stateWarning:
		// TODO better text here pls
		// TODO probably we should show the currently selected abs path also here -> create a field on model
		var warningText string
		// TODO we might also get into problems here with no text wrapping?
		if m.warning == warningDirNotEmpty {
			warningText = "Selected directory is not empty, if you continue files may be overwritten."
		} else if m.warning == warningParentNotExists {
			// TODO maybe don't have the might be a typo part? because sometimes you might want to backup in a subfolder of a new folder
			// or keep it but formulate it differently, you sure you didn't mistype?
			warningText = "Parent directory does not exist, might be a typo."
		} else {
			// TODO this shouldn't happen, just do the above in the else clause here?
		}
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			styles.TitleStyle.Render("Warning!"),
			styles.ErrorTextStyle.Render(fmt.Sprintf("%s\nDo you want to continue?", warningText)),
			m.helpView.ShortHelpView(m.keyMap.warningKeys()),
		)
	}
	return content
}

type Done struct {
	// empty if backup directory was not changed
	BackupDir string
}

func returnBackupDir(backupDir string) tea.Cmd {
	return func() tea.Msg {
		return Done{BackupDir: backupDir}
	}
}

type keyMap struct {
	confirm key.Binding
	cancel  key.Binding

	warningConfirm key.Binding
	warningCancel  key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		warningConfirm: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "yes"),
		),
		warningCancel: key.NewBinding(
			key.WithKeys("esc", "n"),
			key.WithHelp("esc/n", "no"),
		),
	}
}

func (m keyMap) inputKeys() []key.Binding {
	return []key.Binding{m.cancel, m.confirm}
}

func (m keyMap) warningKeys() []key.Binding {
	return []key.Binding{m.warningCancel, m.warningConfirm}
}
