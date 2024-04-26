package zip

import (
	bexec "backup/exec"
	"backup/styles"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	backupDir  string
	textInput  textinput.Model
	inputValid bool
	error      string
	keyMap     keyMap
	styles     styles.Styles
	help       help.Model
}

func NewModel(backupDir string, styles styles.Styles) *Model {
	zt := textinput.New()
	zt.CharLimit = 250
	zt.Width = 40
	zt.Focus()

	help := help.New()
	help.Styles = styles.HelpStyles
	help.ShowAll = true

	return &Model{
		backupDir:  backupDir,
		textInput:  zt,
		inputValid: true,
		error:      "",
		keyMap:     defaultKeyMap(),
		styles:     styles,
		help:       help,
	}
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
			file := m.textInput.Value()
			if file == "" {
				m.inputValid = false
			} else {
				m.inputValid = true
				cmd = zip(m.backupDir, file)
			}
		case key.Matches(msg, m.keyMap.Cancel):
			cmd = done()
		default:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	case zipResult:
		if msg.err == nil {
			m.error = ""
			cmd = done()
		} else {
			m.error = msg.err.Error()
		}
	default:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) View() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n",
		m.styles.TitleStyle.Render("Zip Backup Directory"),
		m.textInput.View(),
	)
	if !m.inputValid {
		content = fmt.Sprintf("%s\n%s\n", content, m.styles.ErrorTextStyle.Render("invalid filename"))
	} else if m.error != "" {
		content = fmt.Sprintf("%s\n%s\n", content, m.styles.ErrorTextStyle.Render(m.error))
	}
	content = fmt.Sprintf("%s\n%s\n", content, m.help.View(m.keyMap))
	return content
}

func (m *Model) KeyMap() help.KeyMap {
	return m.keyMap
}

type zipResult struct {
	err error
}

func zip(dir string, file string) tea.Cmd {
	// sh starts a new shell, so we do not have to worry about changing directory back
	cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %s && zip -r %s .", dir, file))
	// note that zip prints errors to stdout
	return bexec.Exec(cmd, func(err error, s string) tea.Msg {
		if err != nil {
			s = strings.TrimSpace(s)
			e := fmt.Errorf("%v: %v", err, s)
			log.Println(e)
			return zipResult{err: e}
		}
		return zipResult{err: nil}
	}, true)
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
