package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TODO what to do next?
// * make lists to select repos to clone and result of cloning, that one can be static
// * implement authenticate cmd
// * implement loadingRepos cmd
// * implement cloningRepos cmd
// * test and think about refactoring
//
// maybe we could split view building into a separate file, or just put view code at the end?

type State int

const (
	Unauthenticated State = iota
	Authenticating
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

type GithubModel struct {
	state               State
	authenticationError error
	loadingReposError   error

	repos []Repo
	// TODO in general, see also the TODO below we probably want to wrap all this in a custom struct
	// that internally has this list and delegate
	// that model can then handle the space key event, we don't have to handle it here
	// and we have a simple method to get a list of the selected repos
	// yesyes
	reposList *list.Model
	// TODO might want to use a pointer here? so we can e.g. reset selected items more easily
	// otherwise assigning a new map to reposListDelegate.Selected might not have an effect on the
	// copy passed to reposList
	reposListDelegate *itemDelegate

	cloneResult map[string]bool

	onExit tea.Cmd
}

func NewGithubModel(onExit tea.Cmd) GithubModel {
	return GithubModel{
		state:               Authenticating,
		authenticationError: nil,
		loadingReposError:   nil,
		repos:               nil,
		reposList:           nil,
		reposListDelegate:   nil,
		cloneResult:         map[string]bool{},
		onExit:              onExit,
	}
}

func (m GithubModel) Init() tea.Cmd {
	return authenticate()
}

func (m GithubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// TODO switch this around?
	// have an outer switch over the current state, and then for each state an inner switch
	// over the msg
	// yes, this is more clear what is going on although probably a bit more verbose
	// maybe we can handle global keypresses at the top? like esc?

	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cmd = m.onExit
		case "enter":
			if m.state == ReposLoaded {
				m.state = CloningRepos
				var cmds []tea.Cmd
				for _, r := range m.repos {
					if _, ok := m.reposListDelegate.Selected[r.Id]; ok {
						cmds = append(cmds, cloneRepo(r))
					}
				}
				cmd = tea.Batch(cmds...)
			} else if m.state == LoadingReposError {
				m.state = LoadingRepos
				m.loadingReposError = nil
				cmd = loadRepos()
			} else if m.state == Unauthenticated {
				m.state = Authenticating
				m.authenticationError = nil
				cmd = authenticate()
			}
		case " ":
			if m.state == ReposLoaded {
				repo, ok := m.reposList.SelectedItem().(Repo)
				if !ok {
					// TODO this shouldn't happen
					return m, cmd
				}
				_, selected := m.reposListDelegate.Selected[repo.Id]
				if selected {
					delete(m.reposListDelegate.Selected, repo.Id)
				} else {
					m.reposListDelegate.Selected[repo.Id] = struct{}{}
				}
			}
		default:
			if m.state == ReposLoaded {
				lm, lCmd := m.reposList.Update(msg)
				m.reposList = &lm
				cmd = lCmd
			}
		}

	case authenticationResult:
		if m.state != Authenticating {
			return m, cmd
		}
		if msg.err == nil {
			m.state = LoadingRepos
			m.authenticationError = nil
			m.loadingReposError = nil
			cmd = loadRepos()
		} else {
			m.state = Unauthenticated
			m.authenticationError = msg.err
		}

	case loadReposResult:
		if m.state != LoadingRepos {
			return m, cmd
		}

		if msg.err == nil {
			m.state = ReposLoaded
			m.loadingReposError = nil
			m.repos = msg.repos
			items := make([]list.Item, len(m.repos))
			for i, r := range msg.repos {
				items[i] = r
			}
			reposListDelegate := NewItemDelegate()
			// TODO what to do about these values, how can we make it fit where we don't just show a single item?
			reposList := list.New(items, reposListDelegate, 0, 20)
			m.reposListDelegate = &reposListDelegate
			m.reposList = &reposList
		} else {
			m.state = LoadingReposError
			m.loadingReposError = msg.err
		}

	case cloneRepoResult:
		if m.state != CloningRepos {
			// TODO unexpected state, log here this shouldn't happen
			return m, cmd
		}

		if msg.err == nil {
			m.cloneResult[msg.repo.Id] = true
		} else {
			m.cloneResult[msg.repo.Id] = false
		}

		// TODO we should make this better, maybe we should add a model field that holds the selected repos for cloning?
		// that field gets set when we move to cloning repos state
		if len(m.cloneResult) == len(m.reposListDelegate.Selected) {
			m.state = ReposCloned
		}

	default:
		if m.state == ReposLoaded {
			lm, lCmd := m.reposList.Update(msg)
			m.reposList = &lm
			cmd = lCmd
		}
	}
	return m, cmd
}

func (m GithubModel) View() string {
	var style = lipgloss.NewStyle().
		Bold(true).
		BorderStyle(lipgloss.RoundedBorder()).
		MaxHeight(30).
		Padding(1, 4).
		Margin(1, 1)

	var content string

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

	return style.Render(content)
}

func (m GithubModel) viewUnauthenticated() string {
	if m.authenticationError != nil {
		return fmt.Sprintf("Authentication failed: %v\nPress <enter> to try again\n", m.authenticationError)
	} else {
		return "Unauthenticated"
	}
}

func (m GithubModel) viewAuthenticating() string {
	return "Authenticating..."
}

func (m GithubModel) viewLoadingRepos() string {
	return "Authenticated! Loading Repos..."
}

func (m GithubModel) viewLoadingReposeError() string {
	if m.loadingReposError != nil {
		return fmt.Sprintf("Loading repos failed: %v\nPress <enter> to try again\n", m.loadingReposError)
	} else {
		return "Loading repos failed"
	}
}

func (m GithubModel) viewReposLoaded() string {
	return m.reposList.View()
}

func (m GithubModel) viewCloningRepos() string {
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

func (m GithubModel) viewReposCloned() string {
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

type authenticationResult struct {
	err error
}

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
		time.Sleep(time.Second * time.Duration((rand.Intn(4) + 1)))
		var err error
		if repo.Id == "Repo 123" || repo.Id == "Repo b" {
			err = errors.New("could not clone repo")
		}
		return cloneRepoResult{repo: repo, err: err}
	}
}

func (r Repo) FilterValue() string {
	return r.Id
}

type itemDelegate struct {
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style

	Selected map[string]struct{}
}

func NewItemDelegate() itemDelegate {
	return itemDelegate{
		itemStyle:         lipgloss.NewStyle().PaddingLeft(4),
		selectedItemStyle: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")),
		Selected:          map[string]struct{}{},
	}
}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	repo, ok := listItem.(Repo)
	if !ok {
		return
	}

	_, selected := d.Selected[repo.Id]

	var s string
	if selected {
		s = fmt.Sprintf("[x] %v", repo.Name)
	} else {
		s = fmt.Sprintf("[ ] %v", repo.Name)
	}

	if index == m.Index() {
		s = d.selectedItemStyle.Render("> " + s)
	} else {
		s = d.itemStyle.Render(s)
	}

	fmt.Fprintf(w, s)
}
