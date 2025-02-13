package dirselect

import (
	"backup/internal/fs"
	"backup/internal/style"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateInput state = iota
	stateWarning
	stateDone
)

type Model struct {
	state        state
	oldBackupDir string
	newBackupDir string

	inputError error
	warning    int

	textInput textinput.Model
	keyMap    keyMap
	helpView  help.Model

	styles style.Styles
}

func NewModel(backupDir string, styles style.Styles) *Model {
	bt := textinput.New()
	bt.CharLimit = 250
	bt.Width = 40
	bt.Focus()

	helpView := help.New()
	helpView.Styles = styles.HelpStyles

	return &Model{
		oldBackupDir: backupDir,
		state:        stateInput,

		inputError: nil,
		warning:    0,

		textInput: bt,
		keyMap:    defaultKeyMap(),
		helpView:  helpView,

		styles: styles,
	}
}

func (m *Model) Init() tea.Cmd {
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
				absPath, err, warning := validatePath(m.textInput.Value())
				if err != nil {
					m.inputError = err
					m.warning = 0
				} else if warning != 0 {
					m.inputError = nil
					m.state = stateWarning
					m.warning = warning
					m.newBackupDir = absPath
				} else {
					m.newBackupDir = absPath
					// why the extra done state?
					// another key message might get here before the done command is processed
					// e.g. if the user spams the enter key
					// whether or not that is actually possible depends on the implementation of the event loop of bubbletea, I didn't check
					m.state = stateDone
					cmd = done(m.newBackupDir)
				}
			case key.Matches(msg, m.keyMap.back):
				m.state = stateDone
				cmd = done("")
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
				cmd = done(m.newBackupDir)
			case key.Matches(msg, m.keyMap.warningCancel):
				m.state = stateInput
			}
		}
	}
	return m, cmd
}

const (
	warningDirNotEmpty = iota + 1
	warningParentNotExists
)

func validatePath(path string) (string, error, int) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path cannot be empty"), 0
	}

	absPath, err := fs.AbsPath(path)
	if err != nil {
		return "", err, 0
	}

	exists, err := fs.DirExists(fs.ParentPath(absPath))
	if err != nil {
		return "", err, 0
	}
	if !exists {
		return absPath, nil, warningParentNotExists
	}

	empty, err := fs.IsDirEmpty(absPath)
	if err != nil {
		return "", err, 0
	}
	if !empty {
		return absPath, nil, warningDirNotEmpty
	}

	return absPath, nil, 0
}

func (m *Model) SetSize(width, height int) {
	m.helpView.Width = width
}

func (m *Model) View() string {
	styles := m.styles
	var content string

	switch m.state {
	case stateInput:
		parts := []string{
			styles.TitleStyle.Render("Change Backup Directory"),
			"",
			styles.NormalTextStyle.Render(fmt.Sprintf("Current: %s", m.oldBackupDir)),
			"",
			m.textInput.View(),
			"",
		}
		if m.inputError != nil {
			parts = append(parts, styles.ErrorTextStyle.Render(m.inputError.Error()), "")
		}
		parts = append(parts, m.helpView.ShortHelpView(m.keyMap.inputKeys()))
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		)
	case stateWarning:
		var warningText string
		if m.warning == warningDirNotEmpty {
			warningText = "Directory is not empty, if you continue files might get overwritten."
		} else if m.warning == warningParentNotExists {
			warningText = "Parent directory does not exist, might be a typo."
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Warning"),
			"",
			styles.NormalTextStyle.Render(fmt.Sprintf("Selected: %s", m.newBackupDir)),
			"",
			styles.ErrorTextStyle.Render(fmt.Sprintf("%s\nDo you want to continue?", warningText)),
			"",
			m.helpView.ShortHelpView(m.keyMap.warningKeys()),
		)
	}
	return content
}

type Done struct {
	// empty if backup directory was not changed
	BackupDir string
}

func done(backupDir string) tea.Cmd {
	return func() tea.Msg {
		return Done{BackupDir: backupDir}
	}
}

type keyMap struct {
	confirm key.Binding
	back    key.Binding

	warningConfirm key.Binding
	warningCancel  key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		warningConfirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		warningCancel: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
	}
}

func (m keyMap) inputKeys() []key.Binding {
	return []key.Binding{m.back, m.confirm}
}

func (m keyMap) warningKeys() []key.Binding {
	return []key.Binding{m.warningCancel, m.warningConfirm}
}
