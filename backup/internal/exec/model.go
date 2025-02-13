package exec

import (
	"backup/internal/style"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ErrorModel struct {
	result Result

	help   help.Model
	keyMap keyMap

	styles style.Styles

	width  int
	height int
}

func NewErrorModel(result Result, styles style.Styles) *ErrorModel {
	help := help.New()
	help.Styles = styles.HelpStyles

	keyMap := defaultKeyMap()
	if len(result.Stdout) == 0 {
		keyMap.ShowStdout.SetEnabled(false)
	}
	if len(result.Stderr) == 0 {
		keyMap.ShowStderr.SetEnabled(false)
	}

	return &ErrorModel{
		result: result,
		help:   help,
		keyMap: keyMap,
		styles: styles,
	}
}

func (m *ErrorModel) Init() tea.Cmd {
	return nil
}

func (m *ErrorModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.ShowStdout):
			cmd = ForegroundCmd(
				[]string{"less"},
				func(r Result) tea.Msg {
					return nil
				},
				WithStdout(false),
				WithStderr(false),
				WithStdin(m.result.Stdout),
			)
		case key.Matches(msg, m.keyMap.ShowStderr):
			cmd = ForegroundCmd(
				[]string{"less"},
				func(r Result) tea.Msg {
					return nil
				},
				WithStdout(false),
				WithStderr(false),
				WithStdin(m.result.Stderr),
			)
		case key.Matches(msg, m.keyMap.Return):
			cmd = done()
		}
	}

	return cmd
}

func (m *ErrorModel) View() string {
	styles := m.styles
	result := m.result

	content := []string{
		styles.TitleStyle.Render("Ups, something went wrong."),
		"",
	}
	if result.Err != nil {
		content = append(
			content,
			styles.NormalTextStyle.Render(fmt.Sprintf("Command: %s", commandString(result.Cmd))),
			"",
			styles.NormalTextStyle.Render(result.Err.Error()),
		)
	} else {
		content = append(
			content,
			styles.NormalTextStyle.Render(fmt.Sprintf("Command: %s", commandString(result.Cmd))),
			"",
			styles.NormalTextStyle.Render(fmt.Sprintf("Exit Code: %v", result.ExitCode)),
		)
		if len(result.Stdout) > 0 {
			content = append(
				content,
				styles.NormalTextStyle.Render(fmt.Sprintf("Stdout: %v bytes", len(result.Stdout))),
			)
		} else {
			content = append(
				content,
				styles.NormalTextStyle.Render("Stdout: no output"),
			)
		}
		if len(result.Stderr) > 0 {
			content = append(
				content,
				styles.NormalTextStyle.Render(fmt.Sprintf("Stderr: %v bytes", len(result.Stderr))),
			)
		} else {
			content = append(
				content,
				styles.NormalTextStyle.Render("Stderr: no output"),
			)
		}
	}
	content = append(content, "", m.help.ShortHelpView(m.keyMap.keys()))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content...,
	)
}

func commandString(cmd []string) string {
	if len(cmd) == 0 {
		return ""
	}

	s := cmd[0]
	for _, c := range cmd[1:] {
		if strings.Contains(s, " ") {
			s = fmt.Sprintf("%s \"%s\"", s, c)
		} else {
			s = fmt.Sprintf("%s %s", s, c)
		}
	}
	return s
}

func (m *ErrorModel) SetSize(width, height int) {
	m.help.Width = width
	m.width = width
	m.height = height
}

type Done struct{}

func done() tea.Cmd {
	return func() tea.Msg {
		return Done{}
	}
}

type keyMap struct {
	ShowStdout key.Binding
	ShowStderr key.Binding

	Return key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		ShowStdout: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "show stdout"),
		),
		ShowStderr: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "show stderr"),
		),
		Return: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "return"),
		),
	}
}

func (m keyMap) keys() []key.Binding {
	return []key.Binding{
		m.ShowStdout,
		m.ShowStderr,
		m.Return,
	}
}
