package zip

import (
	"backup/internal/exec"
	"backup/internal/fs"
	"backup/internal/style"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateInput state = iota
	// otherwise it might be possible to e.g. start multiple zip commands by spamming the enter key
	stateZipping
	stateSuccess
	stateError
)

type Model struct {
	state      state
	backupDir  string
	inputError error
	result     zipResult

	keyMap keyMap

	textInput textinput.Model
	help      help.Model

	styles style.Styles
}

func NewModel(backupDir string, styles style.Styles) *Model {
	zt := textinput.New()
	zt.CharLimit = 250
	zt.Width = 40
	zt.Focus()

	help := help.New()
	help.Styles = styles.HelpStyles

	return &Model{
		state:      stateInput,
		backupDir:  backupDir,
		inputError: nil,
		keyMap:     defaultKeyMap(),
		textInput:  zt,
		help:       help,
		styles:     styles,
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
			case key.Matches(msg, m.keyMap.inputConfirm):
				file := m.textInput.Value()
				if file == "" {
					m.inputError = errors.New("please type something, anything, I beg you")
				} else {
					absFile, err := fs.AbsPath(file)
					if err != nil {
						m.inputError = err
					} else {
						m.inputError = nil
						m.result = zipResult{}
						m.state = stateZipping
						cmd = zipBackupDir(m.backupDir, absFile)
					}
				}
			case key.Matches(msg, m.keyMap.inputCancel):
				cmd = returnFromZip()
			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	case stateZipping:
		switch msg := msg.(type) {
		case zipResult:
			if msg.result.ExitCode == 0 {
				m.state = stateSuccess
			} else {
				m.state = stateError
			}
			m.result = msg
		}
	case stateSuccess:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keyMap.successContinue) {
			cmd = returnFromZip()
		}
	case stateError:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keyMap.errorContinue) {
			m.state = stateInput
		}
	}

	return m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.help.Width = width
}

func (m *Model) View() string {
	styles := m.styles
	var content string

	switch m.state {
	case stateInput:
		parts := []string{
			styles.TitleStyle.Render("Zip Backup Directory"),
			"",
			styles.NormalTextStyle.Render("Enter filename"),
			"",
			m.textInput.View(),
			"",
		}
		if m.inputError != nil {
			parts = append(parts, styles.ErrorTextStyle.Render(m.inputError.Error()), "")
		}
		parts = append(parts, m.help.ShortHelpView(m.keyMap.inputKeys()))
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			parts...,
		)
	case stateZipping:
		content = ""
	case stateSuccess:
		duration := m.result.result.Time.Round(time.Second)
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Success"),
			"",
			styles.NormalTextStyle.Render(fmt.Sprintf("Zipped %s in %s", m.result.file, duration)),
			styles.NormalTextStyle.Render(fmt.Sprintf("File size %s", fileSizeString(m.result.size))),
			"",
			m.help.ShortHelpView(m.keyMap.successKeys()),
		)
	case stateError:
		// TODO do error page
		var errText string
		if m.result.result.Err != nil {
			errText = m.result.result.Err.Error()
		} else {
			errText = m.result.result.Stdout
		}
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			styles.TitleStyle.Render("Error"),
			styles.NormalTextStyle.Render(errText),
			m.help.ShortHelpView(m.keyMap.errorKeys()),
		)
	}

	return content
}

func fileSizeString(size int64) string {
	if size >= 1024*1024 {
		return fmt.Sprintf("%vM", size/(1024*1024))
	} else if size >= 1024 {
		return fmt.Sprintf("%vK", size/1024)
	} else {
		return fmt.Sprintf("%v", size)
	}
}

type zipResult struct {
	result exec.Result
	file   string
	size   int64
}

func zipBackupDir(dir string, file string) tea.Cmd {
	// should work even if dir is /
	base := fs.BasePath(dir)
	parent := fs.ParentPath(dir)
	// why change directory? because otherwise the zip file will contain all the parent directories of files
	// e.g. when you unzip you will get home/username/backup/somefile instead of just backup/somefile
	// sh starts a new shell, so we do not have to worry about changing directory back
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && zip -r %s %s", parent, file, base)}
	return exec.Foreground(cmd, func(er exec.Result) tea.Msg {
		result := zipResult{
			result: er,
			file:   file,
			size:   -1,
		}
		if er.ExitCode == 0 {
			size, err := fs.FileSize(file)
			if err == nil {
				result.size = size
			}
		}
		return result
	})
}

type Done struct{}

func returnFromZip() tea.Cmd {
	return func() tea.Msg {
		return Done{}
	}
}

type keyMap struct {
	inputConfirm key.Binding
	inputCancel  key.Binding

	successContinue key.Binding

	errorContinue key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		inputConfirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		inputCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		successContinue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
		errorContinue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
	}
}

func (m keyMap) inputKeys() []key.Binding {
	return []key.Binding{m.inputCancel, m.inputConfirm}
}

func (m keyMap) successKeys() []key.Binding {
	return []key.Binding{m.successContinue}
}

func (m keyMap) errorKeys() []key.Binding {
	return []key.Binding{m.errorContinue}
}
