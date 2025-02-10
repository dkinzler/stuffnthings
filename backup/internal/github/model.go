package github

import (
	"backup/internal/fs"
	"backup/internal/style"
	"errors"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Config struct {
	// personal access token to authenticate API requests
	Token string `json:"token"`
}

type state int

const (
	stateNoToken state = iota
	stateLoadingRepos
	stateLoadingReposError
	stateReposLoaded
	stateCloningRepos
	stateReposCloned
)

type Model struct {
	state       state
	confirmBack bool

	backupDir string
	config    Config

	repos             []repo
	loadingReposError error

	reposToClone []repo
	cloneResult  map[int]bool
	clonesFailed int

	selectReposList *selectReposList
	validationError error
	cloneResultList *cloneResultList

	spinner  spinner.Model
	helpView help.Model
	keyMap   keyMap

	styles style.Styles

	width, height int
}

func NewModel(backupDir string, config Config, styles style.Styles) *Model {
	helpView := help.New()
	helpView.Styles = styles.HelpStyles

	state := stateLoadingRepos
	if config.Token == "" {
		state = stateNoToken
	}

	return &Model{
		state:             state,
		confirmBack:       false,
		backupDir:         fs.JoinPath(backupDir, "github"),
		config:            config,
		repos:             nil,
		loadingReposError: nil,
		reposToClone:      nil,
		cloneResult:       map[int]bool{},
		clonesFailed:      0,

		selectReposList: nil,
		validationError: nil,
		cloneResultList: nil,
		spinner: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(styles.ListItemSelectedStyle),
		),
		helpView: helpView,
		keyMap:   defaultKeyMap(),

		styles: styles,
	}
}

func (m *Model) Init() tea.Cmd {
	if m.state == stateNoToken {
		return nil
	}
	return tea.Batch(loadRepos(m.config.Token), m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keyMap.Back) {
		m.confirmBack = true
		return m, nil
	}

	if m.confirmBack {
		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keyMap.ConfirmBack) {
				return m, done()
			} else if key.Matches(msg, m.keyMap.CancelBack) {
				m.confirmBack = false
			}
			return m, nil
		}
		// other messages can still be processed e.g. results of async operations
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
			if msg.err == nil {
				m.state = stateReposLoaded
				m.repos = msg.repos
				m.loadingReposError = nil
				m.selectReposList = newSelectReposList(m.repos, m.keyMap)
				m.setListSize()
				m.validationError = nil
			} else {
				m.state = stateLoadingReposError
				m.repos = nil
				m.loadingReposError = msg.err
			}
		case spinner.TickMsg:
			m.spinner, cmd = m.spinner.Update(msg)
		}

	case stateLoadingReposError:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.ErrorBack):
				m.state = stateLoadingRepos
				m.repos = nil
				m.loadingReposError = nil
				cmd = loadRepos(m.config.Token)
			}
		}

	case stateReposLoaded:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.Continue):
				reposToClone := m.selectReposList.Selected()
				if len(reposToClone) == 0 {
					m.validationError = errors.New("no repos selected")
				} else {
					m.state = stateCloningRepos
					m.reposToClone = reposToClone
					m.cloneResult = map[int]bool{}
					m.clonesFailed = 0
					m.cloneResultList = newCloneResultList(m.repos, m.cloneResult, m.keyMap)
					m.setListSize()
					m.validationError = nil
					var cmds []tea.Cmd
					for _, r := range m.reposToClone {
						cmds = append(cmds, cloneRepo(r, m.backupDir, m.config.Token))
					}
					cmds = append(cmds, m.spinner.Tick)
					cmd = tea.Batch(cmds...)
				}
			default:
				cmd = m.selectReposList.Update(msg)
			}
		default:
			cmd = m.selectReposList.Update(msg)
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
		default:
			cmd = m.cloneResultList.Update(msg)
		}

	case stateReposCloned:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.CloneRetry):
				if m.clonesFailed > 0 {
					cmds := []tea.Cmd{m.spinner.Tick}
					for _, r := range m.reposToClone {
						if success, ok := m.cloneResult[r.Id]; ok && !success {
							delete(m.cloneResult, r.Id)
							cmds = append(cmds, cloneRepo(r, m.backupDir, m.config.Token))
						}
					}
					cmd = tea.Batch(cmds...)
					m.state = stateCloningRepos
					m.clonesFailed = 0
				}
			case key.Matches(msg, m.keyMap.CloneReturn):
				cmd = done()
			default:
				cmd = m.cloneResultList.Update(msg)
			}
		default:
			cmd = m.cloneResultList.Update(msg)
		}
	}
	return m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.setListSize()
	m.helpView.Width = width
}

func (m *Model) setListSize() {
	if m.selectReposList != nil {
		// 4 lines for title and header and 4 lines for help text, plus 2 empty lines
		listHeight := m.height - 10
		if listHeight < 2 {
			listHeight = 2
		}
		m.selectReposList.SetSize(m.width, listHeight)
	}
	if m.cloneResultList != nil {
		// 4 lines for title and header and 4 lines for help text, plus 2 empty lines
		listHeight := m.height - 10
		if listHeight < 2 {
			listHeight = 2
		}
		m.cloneResultList.SetSize(m.width, listHeight)
	}
}

type Done struct{}

func done() tea.Cmd {
	return func() tea.Msg {
		return Done{}
	}
}
