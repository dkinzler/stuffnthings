package zip

import (
	"backup/internal/exec"
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
	// TODO could have had a separate stateZipping but not really necessary?
	// maybe for completeness so that we can't spam keys and cause another zip to happen?
	// although that is unlikely, but why not do it
	stateSuccess
	stateError
)

type Model struct {
	state      state
	backupDir  string
	inputError error
	result     exec.Result

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
	// TODO do we need to init help or textinput?
	return nil
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
					// TODO better text
					m.inputError = errors.New("cannot be empty")
				} else {
					absFile, err := fs.GetAbsPath(file)
					if err != nil {
						m.inputError = err
					} else {
						m.inputError = nil
						cmd = zipBackupDir(m.backupDir, absFile)
					}
				}
			case key.Matches(msg, m.keyMap.inputCancel):
				cmd = returnFromZip()
			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		case zipResult:
			r := msg.result
			if r.ExitCode == 0 {
				m.state = stateSuccess
			} else {
				m.state = stateError
				m.result = r
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
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

func (m *Model) View() string {
	styles := m.styles
	content := ""

	switch m.state {
	case stateInput:
		content = fmt.Sprintf(
			"%s\n\n%s\n%s\n",
			styles.TitleStyle.Render("Zip Backup Directory"),
			styles.NormalTextStyle.Render("Enter filename"),
			m.textInput.View(),
		)
		if m.inputError != nil {
			content = fmt.Sprintf("%s\n%s\n", content, styles.ErrorTextStyle.Render(m.inputError.Error()))
		}
		content = fmt.Sprintf("%s\n%s\n", content, m.help.ShortHelpView(m.keyMap.inputKeys()))
	case stateSuccess:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			// TODO as content could maybe include here the stats, maybe how long it took, how big the resulting file is?
			styles.TitleStyle.Render("Zip successful!"),
			"",
			m.help.ShortHelpView(m.keyMap.successKeys()),
		)
	case stateError:
		// TODO prettier error message, should be show complete stdout?
		var errText string
		if m.result.Err != nil {
			errText = m.result.Err.Error()
		} else {
			errText = m.result.Stdout
		}
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			styles.TitleStyle.Render("Zip error!"),
			styles.NormalTextStyle.Render(errText),
			m.help.ShortHelpView(m.keyMap.errorKeys()),
		)
	}

	return content
}

type zipResult struct {
	result exec.Result
}

// TODO fix
func zipBackupDir(dir string, file string) tea.Cmd {
	// sh starts a new shell, so we do not have to worry about changing directory back
	// why in new shell?
	// we need the cd because otherwise zip file will contain the full path to every file
	// i.e. when you unpack you will get something like home/username/abc/backup/somefile
	// instead of just backup/somefile
	// TODO yeah we probably have to do this
	// cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %s && zip -r %s .", dir, file))

	// note that zip prints errors to stdout
	// return bexec.Exec(cmd, func(err error, s string) tea.Msg {
	// 	if err != nil {
	// 		s = strings.TrimSpace(s)
	// 		e := fmt.Errorf("%v: %v", err, s)
	// 		log.Println(e)
	// 		return zipResult{err: e}
	// 	}
	// 	return zipResult{err: nil}
	// }, true)

	// TODO does this always work?
	// basically if the dir is not just a single segment like / then it should work
	// what if it is / then?
	// TODO probably just forbid using / as backup dir -> yes yes and show a cheeky error message that this is a bad idea and you better do this outside of this program, I'm not taking responsibility for this
	// TODO what happens if we pass something like / to zip as the output file, would this work?
	// or more generally if we pass an existing directory, or will it always add an .zip ending? try it out
	// that should be fine, it will add .zip and if there already is a dir with the name .zip it will try to use it and fail
	base := fs.GetBasePath(dir)
	parent := fs.GetParentPath(dir)
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && zip -r %s %s", parent, file, base)}
	return exec.Foreground(cmd, func(er exec.Result) tea.Msg {
		return zipResult{result: er}
	}, exec.DefaultOptions())
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
