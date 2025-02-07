package github

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.confirmBack {
		return m.viewConfirmBack()
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
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		m.styles.NormalTextStyle.Render("No personal access token provided. Update your config file and try again."),
		"",
		m.helpView.ShortHelpView(m.keyMap.noTokenKeys()),
	)
}

func (m *Model) viewConfirmBack() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		m.styles.NormalTextStyle.Render("Do you really want to go back to the main menu?"),
		"",
		m.helpView.ShortHelpView(m.keyMap.confirmBackKeys()),
	)
}

func (m *Model) viewLoadingRepos() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		fmt.Sprintf(
			"%s %s",
			m.styles.NormalTextStyle.UnsetWidth().Render("Loading repos"),
			m.spinner.View(),
		),
	)
}

func (m *Model) viewLoadingReposeError() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		m.styles.ErrorTextStyle.Render("Ups, loading repos failed."),
		m.styles.ErrorTextStyle.Render(m.loadingReposError.Error()),
		"",
		m.helpView.ShortHelpView(m.keyMap.errorKeys()),
	)
}

func (m *Model) viewReposLoaded() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		m.styles.NormalTextStyle.Render("Select repos to backup"),
		"",
		m.selectReposList.View(),
		"",
		m.helpView.FullHelpView(m.keyMap.reposLoadedKeys()),
	)
}

func (m *Model) viewCloningRepos() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		fmt.Sprintf(
			"%s %s",
			m.styles.NormalTextStyle.UnsetWidth().Render("Cloning repos"),
			m.spinner.View(),
		),
		"",
		m.cloneResultList.View(),
		"",
		m.helpView.ShortHelpView(m.keyMap.cloningReposKeys()),
	)
}

// TODO actually add the log statement to cloneRepo
func (m *Model) viewReposCloned() string {
	var content string
	if m.clonesFailed == 0 {
		content = m.styles.NormalTextStyle.Render("All repos cloned successfully!")
	} else {
		content = m.styles.ErrorTextStyle.Render("Some repos could not be cloned, check the logs for more information. Try again?")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.TitleStyle.Render("GitHub"),
		"",
		content,
		"",
		m.cloneResultList.View(),
		"",
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
	Back       key.Binding

	ErrorRetry  key.Binding
	ErrorCancel key.Binding

	CloneReturn key.Binding
	CloneRetry  key.Binding

	ConfirmBack key.Binding
	CancelBack  key.Binding
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
		// TODO should this be named toggle select?
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
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
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
			key.WithKeys("enter"),
			key.WithHelp("enter", "return"),
		),
		CloneRetry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
		),
		ConfirmBack: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "yes"),
		),
		CancelBack: key.NewBinding(
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
		{m.Select, m.SelectAll, m.Continue, m.Back},
	}
}

func (m keyMap) cloningReposKeys() []key.Binding {
	return []key.Binding{
		m.PrevPage, m.NextPage,
	}
}

func (m keyMap) reposClonedKeys() []key.Binding {
	return []key.Binding{
		m.PrevPage, m.NextPage, m.CloneReturn, m.CloneRetry,
	}
}

func (m keyMap) confirmBackKeys() []key.Binding {
	return []key.Binding{
		m.CancelBack, m.ConfirmBack,
	}
}

func (m keyMap) noTokenKeys() []key.Binding {
	return []key.Binding{m.NoTokenReturn}
}
