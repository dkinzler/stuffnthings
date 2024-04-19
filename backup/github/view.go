package github

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var style = lipgloss.NewStyle().
		Bold(true).
		BorderStyle(lipgloss.RoundedBorder()).
		MaxHeight(30).
		Padding(1, 4).
		Margin(1, 1)

	var content string

	if m.confirmClose {
		content = "Are you sure you want to return? Press y to confim, n to cancel."
	} else {
		switch m.state {
		case Unauthenticated:
			content = m.viewUnauthenticated()
		case Authenticating:
			content = m.viewAuthenticating()
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
	}

	return style.Render(content)
}

func (m Model) viewUnauthenticated() string {
	if m.authenticationError != nil {
		return fmt.Sprintf("Authentication failed: %v\nPress <enter> to try again\n", m.authenticationError)
	} else {
		return "Unauthenticated"
	}
}

func (m Model) viewAuthenticating() string {
	return "Authenticating..."
}

func (m Model) viewLoadingRepos() string {
	return "Authenticated! Loading Repos..."
}

func (m Model) viewLoadingReposeError() string {
	if m.loadingReposError != nil {
		return fmt.Sprintf("Loading repos failed: %v\nPress <enter> to try again\n", m.loadingReposError)
	} else {
		return "Loading repos failed"
	}
}

func (m Model) viewReposLoaded() string {
	return m.reposList.View()
}

func (m Model) viewCloningRepos() string {
	s := "Cloning repos...\n\n"
	for repo, ok := range m.cloneResult {
		if ok {
			s += fmt.Sprintf("%v  ✓\n", repo)
		} else {
			s += fmt.Sprintf("%v  x\n", repo)
		}
	}
	return s
}

func (m Model) viewReposCloned() string {
	s := "Repos Cloned\n"
	for repo, ok := range m.cloneResult {
		if ok {
			s += fmt.Sprintf("%v  ✓\n", repo)
		} else {
			s += fmt.Sprintf("%v  x\n", repo)
		}
	}
	return s
}
