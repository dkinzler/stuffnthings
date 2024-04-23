package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	bexec "backup/exec"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

type State int

const (
	Authenticating State = iota
	AuthenticationError
	Authenticated
	LoadingRepos
	LoadingReposError
	ReposLoaded
	CloningRepos
	ReposCloned
)

type Repo struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	Url           string `json:"url"`
}

type Model struct {
	state State

	authenticationStatus string
	authenticationError  error
	loginError           error

	repos             []Repo
	loadingReposError error

	reposList       *List
	validationError string

	reposToClone []Repo

	cloneResult map[string]bool

	backupDir string

	errorKeyMap         errorKeyMap
	authenticatedKeyMap authenticatedKeyMap
	reposLoadedKeyMap   reposLoadedKeyMap
	reposClonedKeyMap   reposClonedKeyMap

	helpView help.Model

	styles Styles

	viewWidth, viewHeight int
}

func NewModel(backupRoot string, styles Styles) *Model {
	helpView := help.New()
	helpView.Styles = styles.HelpStyles
	helpView.ShowAll = true

	return &Model{
		state:                Authenticating,
		authenticationStatus: "",
		authenticationError:  nil,
		loginError:           nil,
		repos:                nil,
		loadingReposError:    nil,
		reposList:            nil,
		validationError:      "",
		reposToClone:         nil,
		cloneResult:          map[string]bool{},
		backupDir:            filepath.Join(backupRoot, "github"),

		errorKeyMap:         defaultErrorKeyMap(),
		authenticatedKeyMap: defaultAuthenticatedKeyMap(),
		reposLoadedKeyMap:   defaultReposLoadedKeyMap(),
		reposClonedKeyMap:   defaultReposClonedKeyMap(),

		styles:   styles,
		helpView: helpView,
	}
}

func (m *Model) Init() tea.Cmd {
	return checkAuthentication()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {

	case Authenticating:
		switch msg := msg.(type) {
		case authenticationResult:
			if msg.err == nil {
				m.state = Authenticated
				m.authenticationStatus = msg.status
				m.authenticationError = nil
			} else {
				m.state = AuthenticationError
				m.authenticationStatus = ""
				m.authenticationError = msg.err
			}
		case loginResult:
			m.loginError = msg.err
			cmd = checkAuthentication()
		}

	case AuthenticationError:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			if key == "enter" {
				m.state = Authenticating
				m.authenticationError = nil
				cmd = login()
			}
		}

	case Authenticated:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.state = LoadingRepos
				m.repos = nil
				m.loadingReposError = nil
				cmd = loadRepos()
			case "l":
				m.state = Authenticating
				m.authenticationStatus = ""
				m.authenticationError = nil
				m.loginError = nil
				cmd = login()
			case "s":
				m.state = Authenticating
				m.authenticationStatus = ""
				m.authenticationError = nil
				m.loginError = nil
				cmd = switchUser()
			}
		}

	case LoadingRepos:
		switch msg := msg.(type) {
		case loadReposResult:
			if msg.err == nil {
				m.state = ReposLoaded
				m.repos = msg.repos
				m.loadingReposError = nil
				m.reposList = NewList(m.repos, m.reposLoadedKeyMap.listKeyMap())
				m.setListSize()
				m.validationError = ""
			} else {
				m.state = LoadingReposError
				m.repos = nil
				m.loadingReposError = msg.err
			}
		}

	case LoadingReposError:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.state = LoadingRepos
				m.repos = nil
				m.loadingReposError = nil
				cmd = loadRepos()
			}
		}

	case ReposLoaded:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.reposToClone = m.reposList.Selected()
				if len(m.reposToClone) == 0 {
					m.validationError = "No repos selected"
				} else {
					m.state = CloningRepos
					m.validationError = ""
					var cmds []tea.Cmd
					for _, r := range m.reposToClone {
						cmds = append(cmds, cloneRepo(r, m.backupDir))
					}
					cmd = tea.Batch(cmds...)
				}
			default:
				cmd = m.reposList.Update(msg)
			}
		default:
			cmd = m.reposList.Update(msg)
		}
	case CloningRepos:
		switch msg := msg.(type) {
		case cloneRepoResult:
			if msg.err == nil {
				m.cloneResult[msg.id] = true
			} else {
				m.cloneResult[msg.id] = false
			}

			if len(m.cloneResult) == len(m.reposToClone) {
				m.state = ReposCloned
			}
		}

	case ReposCloned:
	}
	return m, cmd
}

func (m *Model) SetSize(w, h int) {
	m.viewWidth = w
	m.viewHeight = h
	m.setListSize()
}

func (m *Model) setListSize() {
	if m.reposList != nil {
		// use 4 lines for title and text and 4 lines for help text, plus 2 empty lines
		height := m.viewHeight - 10
		if height < 2 {
			height = 2
		}
		if height > 15 {
			height = 15
		}
		m.reposList.SetSize(m.viewWidth, height)
	}
}

type authenticationResult struct {
	status string
	err    error
}

func checkAuthentication() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		c := exec.CommandContext(ctx, "gh", "auth", "status", "-h", "github.com")
		out, err := c.CombinedOutput()
		if err != nil {
			return authenticationResult{err: fmt.Errorf("Not authenticated: %v", string(out))}
		}
		return authenticationResult{status: string(out), err: nil}
	}
}

type loginResult struct {
	err error
}

func login() tea.Cmd {
	// interactive, don't use context with timeout here
	cmd := exec.Command("gh", "auth", "login")
	return bexec.Exec(cmd, func(err error, s string) tea.Msg {
		if err != nil {
			e := fmt.Errorf("%v: %v", err, s)
			return loginResult{err: e}
		}
		return loginResult{err: nil}
	}, false)
}

func switchUser() tea.Cmd {
	// interactive, don't use context with timeout here
	cmd := exec.Command("gh", "auth", "switch")
	return bexec.Exec(cmd, func(err error, s string) tea.Msg {
		if err != nil {
			e := fmt.Errorf("%v: %v", err, s)
			return loginResult{err: e}
		}
		return loginResult{err: nil}
	}, false)
}

type loadReposResult struct {
	repos []Repo
	err   error
}

func loadRepos() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		cmd := exec.CommandContext(ctx, "gh", "repo", "list", "--limit", "200", "--json", "id,name,nameWithOwner,url")

		out, err := cmd.CombinedOutput()
		if err != nil {
			return loadReposResult{repos: nil, err: fmt.Errorf("could not load repos: %s", out)}
		}

		var repos []Repo
		err = json.Unmarshal(out, &repos)
		if err != nil {
			return loadReposResult{repos: nil, err: errors.New("could not unmarshal json")}
		}

		return loadReposResult{
			repos: repos,
			err:   nil,
		}
	}
}

type cloneRepoResult struct {
	id  string
	err error
}

func cloneRepo(repo Repo, dir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		// run the command through sh, otherwise e.g. ~ in the path won't get expanded
		cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("gh repo clone %s %s", repo.Url, filepath.Join(dir, repo.Name)))

		err := cmd.Run()
		return cloneRepoResult{id: repo.Id, err: err}
	}
}
