package github

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	ViewStyle             lipgloss.Style
	TitleStyle            lipgloss.Style
	NormalTextStyle       lipgloss.Style
	ErrorTextStyle        lipgloss.Style
	SelectedListItemStyle lipgloss.Style
	HelpStyles            help.Styles
}

func (m *Model) View() string {
	var content string

	switch m.state {
	case Authenticating:
		content = m.viewAuthenticating()
	case AuthenticationError:
		content = m.viewAuthenticationError()
	case Authenticated:
		content = m.viewAuthenticated()
	case LoadingRepos:
		content = m.viewLoadingRepos()
	case LoadingReposError:
		content = m.viewLoadingReposeError()
	case ReposLoaded:
		content = m.viewReposLoaded()
	case CloningRepos:
		content = m.viewCloningRepos()
	case ReposCloned:
		content = m.viewReposCloned()
	}

	return content
}

func (m *Model) viewAuthenticating() string {
	return fmt.Sprintf(
		"%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Authenticating..."),
	)
}

func (m *Model) viewAuthenticationError() string {
	var err error
	if m.loginError != nil {
		err = m.loginError
	} else {
		err = m.authenticationError
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.ErrorTextStyle.Render("Ups, something went wrong. You are not authenticated."),
		m.styles.ErrorTextStyle.Render(err.Error()),
		m.helpView.View(m.errorKeyMap),
	)
}

func (m *Model) viewAuthenticated() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("You are authenticated!"),
		m.authenticationStatus,
		m.helpView.View(m.authenticatedKeyMap),
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
		m.helpView.View(m.errorKeyMap),
	)
}

func (m *Model) viewReposLoaded() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Select repos to backup"),
		m.reposList.View(),
		m.helpView.View(m.reposLoadedKeyMap),
	)
}

var checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ef542")).Render("✓")
var cross = lipgloss.NewStyle().Foreground(lipgloss.Color("#de0d18")).Render("x")

func (m *Model) viewCloningRepos() string {
	var s string
	for _, repo := range m.reposToClone {
		success, ok := m.cloneResult[repo.Id]
		if !ok {
			s += fmt.Sprintf("%s  ?\n", repo.NameWithOwner)
		} else if success {
			s += fmt.Sprintf("%s  %s\n", repo.NameWithOwner, checkmark)
		} else {
			s += fmt.Sprintf("%s  %s\n", repo.NameWithOwner, cross)
		}

	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Cloning Repos"),
		m.styles.NormalTextStyle.Render(s),
	)
}

func (m *Model) viewReposCloned() string {
	var s string
	for _, repo := range m.reposToClone {
		success, ok := m.cloneResult[repo.Id]
		if !ok {
			s += fmt.Sprintf("%s  ?\n", repo.NameWithOwner)
		} else if success {
			s += fmt.Sprintf("%s  %s\n", repo.NameWithOwner, checkmark)
		} else {
			s += fmt.Sprintf("%s  %s\n", repo.NameWithOwner, cross)
		}
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n",
		m.styles.TitleStyle.Render("GitHub"),
		m.styles.NormalTextStyle.Render("Repos Cloned!"),
		m.styles.NormalTextStyle.Render(s),
		m.helpView.View(m.reposClonedKeyMap),
	)
}

type errorKeyMap struct {
	Retry  key.Binding
	Cancel key.Binding
}

func defaultErrorKeyMap() errorKeyMap {
	return errorKeyMap{
		Retry: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "try again"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func (m errorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.Retry,
		m.Cancel,
	}
}

func (m errorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Retry, m.Cancel},
	}
}

type authenticatedKeyMap struct {
	Login    key.Binding
	Switch   key.Binding
	Continue key.Binding
}

func defaultAuthenticatedKeyMap() authenticatedKeyMap {
	return authenticatedKeyMap{
		Login: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "login"),
		),
		Switch: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "switch"),
		),
		Continue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
	}
}

func (m authenticatedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.Login,
		m.Switch,
		m.Continue,
	}
}

func (m authenticatedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Login, m.Switch, m.Continue},
	}
}

type reposLoadedKeyMap struct {
	CursorUp      key.Binding
	CursorDown    key.Binding
	PrevPage      key.Binding
	NextPage      key.Binding
	Select        key.Binding
	SelectAll     key.Binding
	Continue      key.Binding
	ShowFullHelp  key.Binding
	CloseFullHelp key.Binding
}

func defaultReposLoadedKeyMap() reposLoadedKeyMap {
	return reposLoadedKeyMap{
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
	}
}

func (m reposLoadedKeyMap) listKeyMap() list.KeyMap {
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

func (m reposLoadedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.CursorUp,
		m.CursorDown,
		m.PrevPage,
		m.NextPage,
		m.Select,
		m.SelectAll,
		m.Continue,
	}
}

func (m reposLoadedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.CursorUp, m.CursorDown, m.PrevPage, m.NextPage},
		{m.Select, m.SelectAll, m.Continue},
	}
}

type reposClonedKeyMap struct {
	Return key.Binding
	Retry  key.Binding
}

func defaultReposClonedKeyMap() reposClonedKeyMap {
	return reposClonedKeyMap{
		Return: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "return to main menu"),
		),
		Retry: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "retry failed"),
		),
	}
}

func (m reposClonedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.Return, m.Retry,
	}
}

func (m reposClonedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Return, m.Retry},
	}
}
