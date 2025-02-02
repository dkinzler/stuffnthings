package internal

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type zipState int

const (
	zipStateInput zipState = iota
	// TODO could have had a separate state zipStateZipping but not really necessary?
	// maybe for completeness so that we can't spam keys and cause another zip to happen?
	// although that is unlikely, but why not do it
	zipStateSuccess
	zipStateError
)

type zipModel struct {
	commonState *commonState
	state       zipState
	inputError  error
	zipResult   execResult

	keyMap zipKeyMap

	textInput textinput.Model
	help      help.Model
}

func newZipModel(commonState *commonState) *zipModel {
	zt := textinput.New()
	zt.CharLimit = 250
	zt.Width = 40
	zt.Focus()

	help := help.New()
	help.Styles = commonState.styles.HelpStyles

	return &zipModel{
		commonState: commonState,
		state:       zipStateInput,
		inputError:  nil,
		keyMap:      defaultZipKeyMap(),
		textInput:   zt,
		help:        help,
	}
}

func (m *zipModel) Init() tea.Cmd {
	// TODO do we need to init help or textinput?
	return nil
}

func (m *zipModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case zipStateInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.InputConfirm):
				file := m.textInput.Value()
				if file == "" {
					// TODO better text
					m.inputError = errors.New("cannot be empty")
				} else {
					absFile, err := getAbsPath(file)
					if err != nil {
						m.inputError = err
					} else {
						m.inputError = nil
						cmd = zipBackupDir(m.commonState.backupDir, absFile)
					}
				}
			case key.Matches(msg, m.keyMap.InputCancel):
				cmd = returnFromZip()
			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		case zipResult:
			r := msg.result
			if r.exitCode == 0 {
				m.state = zipStateSuccess
			} else {
				m.state = zipStateError
				m.zipResult = r
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	case zipStateSuccess:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keyMap.SuccessContinue) {
			cmd = returnFromZip()
		}
	case zipStateError:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keyMap.ErrorContinue) {
			m.state = zipStateInput
		}
	}

	return m, cmd
}

func (m *zipModel) View() string {
	styles := m.commonState.styles
	content := ""

	switch m.state {
	case zipStateInput:
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
	case zipStateSuccess:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			// TODO as content could maybe include here the stats, maybe how long it took, how big the resulting file is?
			styles.TitleStyle.Render("Zip successful!"),
			"",
			m.help.ShortHelpView(m.keyMap.successKeys()),
		)
	case zipStateError:
		// TODO prettier error message, should be show complete stdout?
		var errText string
		if m.zipResult.err != nil {
			errText = m.zipResult.err.Error()
		} else {
			errText = m.zipResult.stdout
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
	result execResult
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
	base := getBasePath(dir)
	parent := getParentPath(dir)
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && zip -r %s %s", parent, file, base)}
	return execForeground(cmd, func(er execResult) tea.Msg {
		return zipResult{result: er}
	}, defaultExecOptions())
}

type zipDone struct{}

func returnFromZip() tea.Cmd {
	return func() tea.Msg {
		return zipDone{}
	}
}

type zipKeyMap struct {
	InputConfirm key.Binding
	InputCancel  key.Binding

	SuccessContinue key.Binding

	ErrorContinue key.Binding
}

func defaultZipKeyMap() zipKeyMap {
	return zipKeyMap{
		InputConfirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		InputCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		SuccessContinue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
		ErrorContinue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
	}
}

func (m zipKeyMap) inputKeys() []key.Binding {
	return []key.Binding{m.InputCancel, m.InputConfirm}
}

func (m zipKeyMap) successKeys() []key.Binding {
	return []key.Binding{m.SuccessContinue}
}

func (m zipKeyMap) errorKeys() []key.Binding {
	return []key.Binding{m.ErrorContinue}
}
