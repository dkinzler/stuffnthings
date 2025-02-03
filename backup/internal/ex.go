package internal

import (
	"backup/internal/exec"
	"backup/internal/style"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type exModel struct {
	index      int
	cmdInput   textinput.Model
	outputView viewport.Model
	errView    viewport.Model

	exitCode int
	err      error

	styles style.Styles
}

func newExModel(styles style.Styles) *exModel {
	bt := textinput.New()
	bt.Placeholder = ""
	bt.CharLimit = 250
	bt.Width = 40
	bt.Focus()

	outputView := viewport.New(100, 10)
	outputView.SetContent("This is a test")

	errView := viewport.New(100, 10)
	errView.SetContent("This is a test")

	return &exModel{
		index:      0,
		cmdInput:   bt,
		outputView: outputView,
		errView:    errView,
		exitCode:   -1,
		err:        nil,
		styles:     styles,
	}
}

func (m *exModel) Init() tea.Cmd {
	return nil
}

func (m *exModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "n" {
			m.index = (m.index + 1) % 3
			return m, nil
		}
	case commandExecuted:
		m.exitCode = msg.result.ExitCode
		m.err = msg.result.Err
		m.outputView.SetContent(msg.result.Stdout)
		m.errView.SetContent(msg.result.Stderr)
		fmt.Println(msg.result.Stdout)
		fmt.Println(msg.result.Stderr)
		return m, nil
	}

	switch m.index {
	case 0:
		if msg, ok := msg.(tea.KeyMsg); ok {
			if msg.String() == "enter" {
				m.exitCode = -1
				m.err = nil
				m.outputView.SetContent("")
				m.errView.SetContent("")
				parts := strings.Split(m.cmdInput.Value(), " ")
				return m, exec.Foreground(parts, func(er exec.Result) tea.Msg {
					return commandExecuted{result: er}
				}, exec.DefaultOptions())
			}
		}
		m.cmdInput, cmd = m.cmdInput.Update(msg)
	case 1:
		m.outputView, cmd = m.outputView.Update(msg)
	case 2:
		m.errView, cmd = m.errView.Update(msg)
	}
	return m, cmd
}

type commandExecuted struct {
	result exec.Result
}

func (m *exModel) View() string {
	styles := m.styles
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		styles.TitleStyle.Render("Command Test"),
		m.cmdInput.View(),
		m.outputView.View(),
		m.errView.View(),
	)
	if m.exitCode != -1 {
		content = fmt.Sprintf("%s\nExit Code: %v\n", content, m.exitCode)
	}
	if m.err != nil {
		content = fmt.Sprintf("%s\nError: %s\n", content, m.err.Error())
	}
	return content
}
