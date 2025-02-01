package internal

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type zipModel struct {
	commonState *commonState

	textInput  textinput.Model
	inputValid bool
	error      string
	keyMap     zipKeyMap
	help       help.Model

	success       bool
	successDialog *dialogModel
}

func newZipModel(commonState *commonState) *zipModel {
	zt := textinput.New()
	zt.CharLimit = 250
	zt.Width = 40
	zt.Focus()

	help := help.New()
	help.Styles = commonState.styles.HelpStyles
	help.ShowAll = true

	return &zipModel{
		commonState:   commonState,
		textInput:     zt,
		inputValid:    true,
		error:         "",
		keyMap:        defaultZipKeyMap(),
		help:          help,
		success:       false,
		successDialog: nil,
	}
}

func (m *zipModel) Init() tea.Cmd {
	return nil
}

func (m *zipModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.success {
		switch msg := msg.(type) {
		case dialogDone:
			return m, returnFromZip()
		default:
			return m, m.successDialog.Update(msg)
		}
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Confirm):
			file := m.textInput.Value()
			if file == "" {
				m.inputValid = false
			} else {
				// TODO show error message?
				absFile, err := getAbsPath(file)
				if err != nil {
					m.inputValid = false
				} else {
					m.inputValid = true
					cmd = zipBackupDir(m.commonState.backupDir, absFile)
				}
			}
		case key.Matches(msg, m.keyMap.Cancel):
			cmd = returnFromZip()
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	case zipResult:
		r := msg.result
		if r.exitCode == 0 {
			m.success = true
			m.successDialog = newDialogModel(m.commonState.styles, dialogOptionWithSingleButton())
		} else {
			m.error = r.err.Error()
		}
	default:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m *zipModel) View() string {
	styles := m.commonState.styles

	if m.success {
		return m.successDialog.View("Zip successful!", "back to main menu", "")
	}

	content := fmt.Sprintf(
		"%s\n\n%s\n%s\n",
		styles.TitleStyle.Render("Zip Backup Directory"),
		styles.NormalTextStyle.Render("Enter filename"),
		m.textInput.View(),
	)
	if !m.inputValid {
		content = fmt.Sprintf("%s\n%s\n", content, styles.ErrorTextStyle.Render("invalid filename"))
	} else if m.error != "" {
		content = fmt.Sprintf("%s\n%s\n", content, styles.ErrorTextStyle.Render(m.error))
	}
	content = fmt.Sprintf("%s\n%s\n", content, m.help.View(m.keyMap))
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
	Confirm key.Binding
	Cancel  key.Binding
}

func defaultZipKeyMap() zipKeyMap {
	return zipKeyMap{
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

func (m zipKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Cancel, m.Confirm}
}

func (m zipKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.Cancel}, {m.Confirm}}
}
