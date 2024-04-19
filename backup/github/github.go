package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"

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

	authenticationError error

	repos             []Repo
	loadingReposError error

	reposList *List

	reposToClone []Repo

	cloneResult map[string]bool

	onExit tea.Cmd
}

func NewModel(onExit tea.Cmd) Model {
	return Model{
		state:               Authenticating,
		authenticationError: nil,
		repos:               nil,
		loadingReposError:   nil,
		reposList:           nil,
		reposToClone:        nil,
		cloneResult:         map[string]bool{},
		onExit:              onExit,
	}
}

func (m Model) Init() tea.Cmd {
	return authenticate()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {

	case Authenticating:
		switch msg := msg.(type) {
		case authenticationResult:
			if msg.err == nil {
				m.state = Authenticated
				m.authenticationError = nil
			} else {
				m.state = AuthenticationError
				m.authenticationError = msg.err
			}
		}

	case AuthenticationError:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			if key == "enter" {
				m.state = Authenticating
				m.authenticationError = nil
				cmd = authenticate()
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
			}
			// TODO else if key == "l" here call login command to use another account?
			// and after that we need to check authentication again
		}

	case LoadingRepos:
		switch msg := msg.(type) {
		case loadReposResult:
			if msg.err == nil {
				m.state = ReposLoaded
				m.repos = msg.repos
				m.loadingReposError = nil
				m.reposList = NewList(m.repos)
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

				m.state = CloningRepos
				var cmds []tea.Cmd
				for _, r := range m.reposToClone {
					cmds = append(cmds, cloneRepo(r))
				}
				cmd = tea.Batch(cmds...)
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
				m.cloneResult[msg.repo.Id] = true
			} else {
				m.cloneResult[msg.repo.Id] = false
			}

			if len(m.cloneResult) == len(m.reposToClone) {
				m.state = ReposCloned
			}
		}

	case ReposCloned:
		// TODO can press enter here to retry failed ones?
		// how would we handle this with reposToClone and cloneResult?
		// maybe have another list that keeps those currently in progress of cloning?

	}
	return m, cmd
}

type authenticationResult struct {
	err error
}

// TODO split this into separate functions, a function sthat checks if we are authenticated and if so
// return the account and information about it
// another functions for login?
// or have both of these as commands and then we can define authenticate as a sequence command?
// but that won't work because?
// probably use functions and then define commands based on that?
// authenticate() does checking and then login

// TODO to check if we are logged in already can do
// gh auth status -h github.com   -> will exit with code 1 if not logged in, 0 otherwise
// could also use this to read the current authenticated username?
func authenticate() tea.Cmd {
	return func() tea.Msg {
		c := exec.Command("gh", "auth", "status", "-h", "github.com")
		err := c.Run()
		if err == nil {
			return authenticationResult{err: nil}
		}
		// TODO could also run this manually
		c = exec.Command("gh", "auth", "login")
		return tea.ExecProcess(c, func(err error) tea.Msg {
			log.Println(err)
			return authenticationResult{err: err}
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
		cmd := exec.Command("gh", "repo", "list", "--json", "id,name,url")

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
	repo Repo
	err  error
}

func cloneRepo(repo Repo) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "repo", "clone", repo.Url, fmt.Sprintf("backup/%v", repo.Name))

		err := cmd.Run()
		return cloneRepoResult{repo: repo, err: err}
	}
}
