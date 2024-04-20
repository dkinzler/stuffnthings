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

	return style.Render(content)
}

func (m Model) viewAuthenticating() string {
	return "Authenticating..."
}

func (m Model) viewAuthenticationError() string {
	if m.loginError != nil {
		return "Login failed."
	}
	return fmt.Sprintf("Not authenticated: %v\nPress <enter> to try again\n", m.authenticationError)
}

func (m Model) viewAuthenticated() string {
	s := "Authenticated! Press <enter> to continue.\n" + m.authenticationStatus
	return s
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
	for _, repo := range m.reposToClone {
		success, ok := m.cloneResult[repo.Id]
		if !ok {
			continue
		}
		if success {
			s += fmt.Sprintf("%v  ✓\n", repo.NameWithOwner)
		} else {
			s += fmt.Sprintf("%v  x\n", repo.NameWithOwner)
		}

	}
	return s
}

func (m Model) viewReposCloned() string {
	s := "Repos Cloned\n"
	for _, repo := range m.reposToClone {
		success, ok := m.cloneResult[repo.Id]
		if !ok {
			continue
		}
		if success {
			s += fmt.Sprintf("%v  ✓\n", repo.NameWithOwner)
		} else {
			s += fmt.Sprintf("%v  x\n", repo.NameWithOwner)
		}

	}
	return s
}
