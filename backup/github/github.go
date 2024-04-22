package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

/*
TODO
* could make view code a bit more dry by setting title/helpView globally and just return the content from methods
*/

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
}

func NewModel(backupRoot string, styles Styles) Model {
	helpView := help.New()
	helpView.Styles = styles.HelpStyles
	helpView.ShowAll = true

	return Model{
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

func (m Model) Init() tea.Cmd {
	return checkAuthentication()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

type authenticationResult struct {
	status string
	err    error
}

func checkAuthentication() tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("gh", "auth", "status", "-h", "github.com")
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
	return func() tea.Msg {
		c := exec.Command("gh", "auth", "login")
		return tea.ExecProcess(c, func(err error) tea.Msg {
			log.Println(err)
			return loginResult{err: err}
		})()
	}
}

func switchUser() tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("gh", "auth", "switch")
		return tea.ExecProcess(c, func(err error) tea.Msg {
			log.Println(err)
			return loginResult{err: err}
		})()
	}
}

type loadReposResult struct {
	repos []Repo
	err   error
}

// TODO can use "gh repo list dkinzler --json "name,id,url""
// we can run a exec.Cmd directly to get the output and parse it as json
// we also want to read the status code of the command to see if the output was valid?
// can then do classic json parsing
// can we unmarshal into a []Repo?
// TODO use CommandContext? to cancel the process after a certain time?
func loadRepos() tea.Cmd {
	return func() tea.Msg {
		// TODO add argument here to increase max num of results
		cmd := exec.Command("gh", "repo", "list", "--json", "id,name,nameWithOwner,url")

		out, err := cmd.Output()
		if err != nil {
			return loadReposResult{repos: nil, err: errors.New("could not load repos")}
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
		// run the command through sh, otherwise e.g. ~ in the path won't get expanded
		cmd := exec.Command("sh", "-c", fmt.Sprintf("gh repo clone %s %s", repo.Url, filepath.Join(dir, repo.Name)))

		err := cmd.Run()
		return cloneRepoResult{id: repo.Id, err: err}
	}
}
