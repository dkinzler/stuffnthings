package github

import (
	"backup/internal/fs"
	"backup/internal/style"
	"log"

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

// TODO do we need a confirm dialog? maybe not, can just load stuff again, no biggy?
// although it might be annoying on repo select screen if you did a bunch of selections and then misclick
// -> probably keep

type Model struct {
	state state

	confirmQuit bool

	token string

	repos             []repo
	loadingReposError error

	reposList       *RepoList
	validationError string

	reposToClone []repo

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
			spinner.WithStyle(styles.ListItemSelectedStyle),
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

type Done struct{}

// TODO we could name that function done() as well in other models if we don't have that already -> check it out
// yesyes
func done() tea.Cmd {
	return func() tea.Msg {
		return Done{}
	}
}
