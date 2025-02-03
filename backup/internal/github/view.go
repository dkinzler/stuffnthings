package github

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.confirmQuit {
		return m.viewConfirmQuit()
	}

	var content string

	switch m.state {
	case stateNoToken:
		content = m.viewNoToken()
	case stateLoadingRepos:
		content = m.viewLoadingRepos()
	case stateLoadingReposError:
		content = m.viewLoadingReposeError()
	case stateReposLoaded:
		content = m.viewReposLoaded()
	case stateCloningRepos:
		content = m.viewCloningRepos()
	case stateReposCloned:
		content = m.viewReposCloned()
	}

	return content
}

func (m *Model) viewNoToken() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n\n",
		m.styles.TitleStyle.Render("GitHub"),
		// TODO better text
		m.styles.NormalTextStyle.Render("No token provided."),
		m.helpView.ShortHelpView(m.keyMap.noTokenKeys()),
	)
}

func (m *Model) viewConfirmQuit() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Do you really want to return to the main menu?"),
		m.helpView.ShortHelpView(m.keyMap.confirmQuitKeys()),
	)
}

func (m *Model) viewLoadingRepos() string {
	return fmt.Sprintf(
		"%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Loading repos..."),
	)
}

func (m *Model) viewLoadingReposeError() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.ErrorTextStyle.Render("Ups, loading repos failed."),
		m.styles.ErrorTextStyle.Render(m.loadingReposError.Error()),
		m.helpView.ShortHelpView(m.keyMap.errorKeys()),
	)
}

func (m *Model) viewReposLoaded() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Select repos to backup"),
		m.reposList.View(),
		m.helpView.FullHelpView(m.keyMap.reposLoadedKeys()),
	)
}

var checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ef542")).Render("✓")
var cross = lipgloss.NewStyle().Foreground(lipgloss.Color("#de0d18")).Render("x")

func (m *Model) viewCloningRepos() string {
	var s string
	for _, repo := range m.reposToClone {
		success, ok := m.cloneResult[repo.Id]
		if !ok {
			s += fmt.Sprintf("%s  ?\n", repo.FullName)
		} else if success {
			s += fmt.Sprintf("%s  %s\n", repo.FullName, checkmark)
		} else {
			s += fmt.Sprintf("%s  %s\n", repo.FullName, cross)
		}

	}

	return fmt.Sprintf(
		"%s\n\n%s %s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Cloning Repos"),
		m.spinner.View(),
		m.styles.NormalTextStyle.Render(s),
	)
}

func (m *Model) viewReposCloned() string {
	var s string
	for _, repo := range m.reposToClone {
		success, ok := m.cloneResult[repo.Id]
		if !ok {
			s += fmt.Sprintf("%s  ?\n", repo.FullName)
		} else if success {
			s += fmt.Sprintf("%s  %s\n", repo.FullName, checkmark)
		} else {
			s += fmt.Sprintf("%s  %s\n", repo.FullName, cross)
		}
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Repos Cloned!"),
		m.styles.NormalTextStyle.Render(s),
		m.helpView.ShortHelpView(m.keyMap.reposClonedKeys()),
	)
}

type keyMap struct {
	NoTokenReturn key.Binding

	CursorUp   key.Binding
	CursorDown key.Binding
	PrevPage   key.Binding
	NextPage   key.Binding
	Select     key.Binding
	SelectAll  key.Binding
	Continue   key.Binding
	Quit       key.Binding

	ErrorRetry  key.Binding
	ErrorCancel key.Binding

	CloneReturn key.Binding
	CloneRetry  key.Binding

	ConfirmQuit key.Binding
	CancelQuit  key.Binding
}

// TODO some of these keys are a bit fucked up still, should we be able to return with esc? -> probably need to somehow, but ask for confirmation
func defaultKeyMap() keyMap {
	return keyMap{
		NoTokenReturn: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "return"),
		),
		CursorUp: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "down"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "prev page"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "next page"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all"),
		),
		Continue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		ErrorRetry: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "retry"),
		),
		ErrorCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		CloneReturn: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "return to main menu"),
		),
		CloneRetry: key.NewBinding(
			key.WithKeys("enter"),
			// TODO bad text
			key.WithHelp("enter", "retry failed"),
		),
		ConfirmQuit: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "yes"),
		),
		CancelQuit: key.NewBinding(
			key.WithKeys("esc", "n"),
			key.WithHelp("esc/n", "no"),
		),
	}
}

func (m keyMap) errorKeys() []key.Binding {
	return []key.Binding{m.ErrorCancel, m.ErrorRetry}
}

func (m keyMap) listKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp:             m.CursorUp,
		CursorDown:           m.CursorDown,
		PrevPage:             m.PrevPage,
		NextPage:             m.NextPage,
		GoToStart:            key.NewBinding(key.WithDisabled()),
		GoToEnd:              key.NewBinding(key.WithDisabled()),
		Filter:               key.NewBinding(key.WithDisabled()),
		ClearFilter:          key.NewBinding(key.WithDisabled()),
		CancelWhileFiltering: key.NewBinding(key.WithDisabled()),
		AcceptWhileFiltering: key.NewBinding(key.WithDisabled()),
		ShowFullHelp:         key.NewBinding(key.WithDisabled()),
		CloseFullHelp:        key.NewBinding(key.WithDisabled()),
	}
}

func (m keyMap) reposLoadedKeys() [][]key.Binding {
	return [][]key.Binding{
		{m.CursorUp, m.CursorDown, m.PrevPage, m.NextPage},
		{m.Select, m.SelectAll, m.Continue, m.Quit},
	}
}

func (m keyMap) reposClonedKeys() []key.Binding {
	return []key.Binding{
		m.CloneReturn, m.CloneRetry,
	}
}

func (m keyMap) confirmQuitKeys() []key.Binding {
	return []key.Binding{
		m.CancelQuit, m.ConfirmQuit,
	}
}

func (m keyMap) noTokenKeys() []key.Binding {
	return []key.Binding{m.NoTokenReturn}
}
