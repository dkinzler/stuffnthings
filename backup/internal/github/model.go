package github

import (
	"backup/internal/exec"
	"backup/internal/fs"
	"backup/internal/style"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateNoToken state = iota
	stateLoadingRepos
	stateLoadingReposError
	stateReposLoaded
	stateCloningRepos
	stateReposCloned
)

// TODO any other fields we should add, maybe if it is private or not?
type Repo struct {
	// TODO is this the right id field or should we use something else? is it unique?
	Id       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	CloneUrl string `json:"clone_url"`
}

// TODO do we need a confirm dialog? maybe not, can just load stuff again, no biggy?
// although it might be annoying on repo select screen if you did a bunch of selections and then misclick
// -> probably keep

type Model struct {
	state state

	confirmQuit bool

	token string

	repos             []Repo
	loadingReposError error

	reposList       *RepoList
	validationError string

	reposToClone []Repo

	cloneResult  map[int]bool
	clonesFailed int

	backupDir string

	keyMap keyMap

	helpView help.Model

	styles style.Styles

	// TODO rename to width and height? and have a SetSize method?
	viewWidth, viewHeight int

	spinner spinner.Model
}

func NewModel(backupDir string, token string, styles style.Styles) *Model {
	helpView := help.New()
	helpView.Styles = styles.HelpStyles

	// TODO or should we have an initialized state or similar here when we have a token but are notloading yet?
	// because right now we assume that whoever uses this will call Init() so we kick off the loading process
	state := stateLoadingRepos
	if token == "" {
		state = stateNoToken
	}

	return &Model{
		state:             state,
		confirmQuit:       false,
		token:             token,
		repos:             nil,
		loadingReposError: nil,
		reposList:         nil,
		validationError:   "",
		reposToClone:      nil,
		cloneResult:       map[int]bool{},
		clonesFailed:      0,
		backupDir:         fs.JoinPath(backupDir, "github"),

		keyMap: defaultKeyMap(),

		styles:   styles,
		helpView: helpView,
		spinner: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(styles.SelectedListItemStyle),
		),
	}
}

func (m *Model) Init() tea.Cmd {
	// TODO does this make sense, it might be possible that we are not in loading state but call loadRepos so stuff won't get updated or simlar?
	// probably don'tw orry to much about this, nobody else will use this
	if m.state == stateNoToken {
		return nil
	}
	return loadRepos(m.token)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keyMap.Quit) {
		m.confirmQuit = true
		return m, nil
	}

	if m.confirmQuit {
		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keyMap.ConfirmQuit) {
				return m, done()
			} else if key.Matches(msg, m.keyMap.CancelQuit) {
				m.confirmQuit = false
			}
			return m, nil
		}
		// other events than tea.KeyMsg can be further processes e.g. results of async operations
	}

	var cmd tea.Cmd

	switch m.state {

	case stateNoToken:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, m.keyMap.NoTokenReturn) {
				cmd = done()
			}
		}

	case stateLoadingRepos:
		switch msg := msg.(type) {
		case loadReposResult:
			// TODO remove all debug log messages here later
			if msg.err == nil {
				log.Println("repos loaded successfully")
				m.state = stateReposLoaded
				m.repos = msg.repos
				m.loadingReposError = nil
				m.reposList = NewList(m.repos, m.keyMap)
				m.setListSize()
				m.validationError = ""
			} else {
				log.Println("repos loading failed", msg.err)
				m.state = stateLoadingReposError
				m.repos = nil
				m.loadingReposError = msg.err
			}
		}

	case stateLoadingReposError:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.ErrorRetry):
				m.state = stateLoadingRepos
				m.repos = nil
				m.loadingReposError = nil
				cmd = loadRepos(m.token)
			}
		}

	case stateReposLoaded:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.Continue):
				reposToClone := m.reposList.Selected()
				if len(reposToClone) == 0 {
					m.validationError = "No repos selected"
				} else {
					m.state = stateCloningRepos
					m.reposToClone = reposToClone
					m.cloneResult = map[int]bool{}
					m.clonesFailed = 0
					m.validationError = ""
					var cmds []tea.Cmd
					for _, r := range m.reposToClone {
						cmds = append(cmds, cloneRepo(r, m.backupDir, m.token))
					}
					cmds = append(cmds, m.spinner.Tick)
					cmd = tea.Batch(cmds...)
				}
			default:
				cmd = m.reposList.Update(msg)
			}
		default:
			cmd = m.reposList.Update(msg)
		}
	case stateCloningRepos:
		switch msg := msg.(type) {
		case cloneRepoResult:
			if msg.err == nil {
				m.cloneResult[msg.id] = true
			} else {
				m.cloneResult[msg.id] = false
				m.clonesFailed += 1
			}

			if len(m.cloneResult) == len(m.reposToClone) {
				m.state = stateReposCloned
				if m.clonesFailed > 0 {
					m.keyMap.CloneRetry.SetEnabled(true)
				} else {
					m.keyMap.CloneRetry.SetEnabled(false)
				}
			}
		case spinner.TickMsg:
			m.spinner, cmd = m.spinner.Update(msg)
		}

	case stateReposCloned:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, m.keyMap.CloneRetry) {
				if m.clonesFailed > 0 {
					var cmds []tea.Cmd
					for _, r := range m.reposToClone {
						if success, ok := m.cloneResult[r.Id]; ok && !success {
							delete(m.cloneResult, r.Id)
							cmds = append(cmds, cloneRepo(r, m.backupDir, m.token))
						}
					}
					cmd = tea.Batch(cmds...)
					m.state = stateCloningRepos
					m.clonesFailed = 0
				}
			} else if key.Matches(msg, m.keyMap.CloneReturn) {
				cmd = done()
			}
		}
	}
	return m, cmd
}

func (m *Model) SetSize(w, h int) {
	m.viewWidth = w
	m.viewHeight = h
	m.setListSize()
}

// TODO move this into SetSize
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

// TODO add log messages to this
// instead of showing detailed errors can also just give the bare minimum and refer to logs, that is probably easier because usually things shouldn't go wrong

type loadReposResult struct {
	repos []Repo
	err   error
}

func loadRepos(token string) tea.Cmd {
	return func() tea.Msg {
		var result loadReposResult

		var repos []Repo

		// TODO is this the right way or shoudl we use REquestWithContext?
		client := &http.Client{Timeout: time.Second * 10}

		req, err := http.NewRequest("GET", "https://api.github.com/user/repos", nil)
		if err != nil {
			result.err = err
			return result
		}
		req.Header.Add("Accept", "application/vnd.github+json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		// could be omitted to always use most recent version
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
		q := req.URL.Query()
		// TODO only get your own repos, keep it like that?
		q.Add("affiliation", "owner")
		q.Add("per_page", "100")
		req.URL.RawQuery = q.Encode()

		// TODO do pagination
		// see https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api?apiVersion=2022-11-28
		// basically check if the next header ist there

		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			result.err = err
			return result
		}

		if resp.StatusCode != http.StatusOK {
			result.err = err
			return result
		}

		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&repos)
		if err != nil {
			result.err = err
			return result
		}

		result.repos = repos
		result.err = nil
		return result
	}
}

type cloneRepoResult struct {
	id  int
	err error
}

func cloneRepo(repo Repo, dir string, token string) tea.Cmd {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	// defer cancel()
	// // run the command through sh, otherwise e.g. ~ in the path won't get expanded
	// cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("gh repo clone %s %s", repo.Url, filepath.Join(dir, repo.Name)))
	//
	// out, err := cmd.CombinedOutput()
	// if err != nil {
	// 	e := fmt.Errorf("%s: %s", err, out)
	// 	log.Println(e)
	// 	return loadReposResult{repos: nil, err: e}
	// }
	// return cloneRepoResult{id: repo.Id, err: nil}

	// that works
	// but maybe do it with username and token in url, that seems more robust, and run like this commands won't get logged?
	// TODO do we have to be careful with repo names? can they contain weird chars so that we shouldn't use them as dir name?
	cmd := []string{"git", "clone", repo.CloneUrl, fs.JoinPath(dir, repo.Name)}
	// cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("gh repo clone %s %s", repo.Url, filepath.Join(dir, repo.Name)))
	options := exec.DefaultOptions()
	options.Stdin = fmt.Sprintf("%s\n%s\n", repo.Owner.Login, token)

	return exec.Background(cmd, func(r exec.Result) tea.Msg {
		if r.ExitCode == 0 {
			return cloneRepoResult{id: repo.Id, err: nil}
		} else {
			if r.Err != nil {
				return cloneRepoResult{id: repo.Id, err: r.Err}
			} else {
				// TODO this is not good, what about stderr and stdout, does git write to stderr?
				return cloneRepoResult{id: repo.Id, err: errors.New("something went wrong")}
			}
		}
	}, options)
}

type Done struct{}

// TODO we could name that function done() as well in other models if we don't have that already -> check it out
func done() tea.Cmd {
	return func() tea.Msg {
		return Done{}
	}
}

// func cloneRepo(repo Repo, token string) {
// 	// that works
// 	// but maybe do it with username and token in url, that seems more robust, and run like this commands won't get logged?
// 	cmd := []string{"git", "clone", repo.CloneUrl, "clonetest"}
// 	// cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("gh repo clone %s %s", repo.Url, filepath.Join(dir, repo.Name)))
// 	options := exec
// 	options.stdin = fmt.Sprintf("%s\n%s\n", repo.Owner.Login, token)
// 	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
// 	defer cancel()
//
// 	var name string
// 	var args []string
// 	if len(cmd) > 0 {
// 		name = cmd[0]
// 		args = cmd[1:]
// 	}
//
// 	c := exec.CommandContext(ctx, name, args...)
// 	if options.stdin != "" {
// 		c.Stdin = strings.NewReader(options.stdin)
// 	}
// 	var outBuffer *strings.Builder
// 	if options.returnStdout {
// 		outBuffer = &strings.Builder{}
// 	}
// 	c.Stdout = outBuffer
// 	var errBuffer *strings.Builder
// 	if options.returnStderr {
// 		errBuffer = &strings.Builder{}
// 	}
// 	c.Stderr = errBuffer
//
// 	err := c.Run()
//
// 	var result exec.Result
// 	result.Cmd = cmd
// 	if err != nil {
// 		result.ExitCode = -1
// 		e, ok := err.(*exec.ExitError)
// 		if ok {
// 			if e.Exited() {
// 				result.ExitCode = e.ExitCode()
// 			} else {
// 				result.Err = err
// 			}
// 		} else {
// 			result.Err = err
// 		}
// 	}
// 	if options.returnStdout {
// 		result.Stdout = outBuffer.String()
// 	}
// 	if options.returnStderr {
// 		result.Stderr = errBuffer.String()
// 	}
//
// 	if result.ExitCode != 0 {
// 		panic(fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n", result.ExitCode, result.Err.Error(), result.Stderr, result.Stdout))
// 	}
//
// 	// run the command through sh, otherwise e.g. ~ in the path won't get expanded
// 	// cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("gh repo clone %s %s", repo.Url, filepath.Join(dir, repo.Name)))
// }
