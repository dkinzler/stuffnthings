package internal

import (
	"backup/internal/dirselect"
	"backup/internal/fs"
	"backup/internal/github"
	"backup/internal/style"
	"backup/internal/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Notes on design
//   - we use a separate package for every UI component to avoid naming conflicts, UI components will typically have types "state", "model", "keyMap" etc.
//     with a single package we would need to add a qualifier to every name e.g. mainMenuKeyMap, githubKeyMap ...
//   - we could have defined a common component to handle all dialogs e.g. to confirm an action or show an error message
//     but overall it seems easier to just create them from scratch every time
//     it does not take substantially more code and we always have full control over behavior and looks, especially since dialogs often need to be slightly different
//     e.g. one or two actions, naming of actions, coloring of text etc.
//   - components that need to know the current window size get updates via a SetSize(width, height) method
//     the alternative would be to forward the tea.WindowSizeMsg messages to nested components

type state int

const (
	stateMainMenu state = iota
	stateConfigError
	stateDirSelect
	stateZip
	stateGithub
)

type model struct {
	config config

	state state
	// if true a dialog to confirm quit will be shown
	confirmQuit bool
	configError error

	mainMenu             list.Model
	mainMenuItemDelegate *mainMenuItemDelegate
	helpView             help.Model
	keyMap               keyMap

	dirSelectModel *dirselect.Model
	zipModel       *zip.Model
	githubModel    *github.Model

	styles style.Styles

	// we need to keep track of the current window size so that we can pass it to nested models when they are created
	width  int
	height int
}

func NewModel(configFile string) *model {
	initialState := stateMainMenu

	config, configErr := loadConfig(configFile)
	if configErr != nil {
		initialState = stateConfigError
	}

	styles := style.DefaultStyles()

	keyMap := defaultKeyMap()

	itemDelegate := &mainMenuItemDelegate{backupDir: config.BackupDir, styles: styles}
	list := list.New(mainMenuItems, itemDelegate, 0, 0)
	list.SetFilteringEnabled(false)
	list.SetShowHelp(false)
	list.DisableQuitKeybindings()
	list.SetShowStatusBar(false)
	list.SetShowPagination(false)
	list.SetShowTitle(false)
	list.KeyMap = keyMap.listKeyMap()

	helpView := help.New()
	helpView.Styles = styles.HelpStyles

	return &model{
		config: config,

		state:       initialState,
		confirmQuit: false,
		configError: configErr,

		mainMenu:             list,
		mainMenuItemDelegate: itemDelegate,
		helpView:             helpView,
		keyMap:               keyMap,

		// instead of keeping track of each child/nested model we could have used a single generic "innerModel" field of type tea.Model
		// but sometimes it can be useful to have the concrete types e.g. to call a method not part of the tea.Model interface
		dirSelectModel: nil,
		zipModel:       nil,
		githubModel:    nil,

		styles: styles,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirmQuit {
		// process any other messages as usual e.g. inner models might have async commands running that will return a message while the confirm quit dialog is still open
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.ConfirmQuit):
				return m, tea.Quit
			case key.Matches(msg, m.keyMap.CancelQuit):
				m.confirmQuit = false
				return m, nil
			default:
				// return, otherwise key message would be passed through to inner model
				return m, nil
			}
		}
	} else if m.state == stateConfigError {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, m.keyMap.ConfigErrorContinue) {
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			m.confirmQuit = true
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd

	switch m.state {

	case stateMainMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.Select):
				s := int(m.mainMenu.SelectedItem().(mainMenuItem))
				switch s {
				case mainMenuItemDirSelect:
					m.state = stateDirSelect
					m.dirSelectModel = dirselect.NewModel(m.config.BackupDir, m.styles)
					cmd = m.dirSelectModel.Init()
				case mainMenuItemZip:
					m.state = stateZip
					m.zipModel = zip.NewModel(m.config.BackupDir, m.styles)
					cmd = m.zipModel.Init()
				case mainMenuItemGithub:
					m.state = stateGithub
					m.githubModel = github.NewModel(m.config.BackupDir, m.config.Github.Token, m.styles)
					cmd = m.githubModel.Init()
				}
				// will call SetSize on the nested model we just created
				m.SetSize(m.width, m.height)
				return m, cmd
			}
		}
		m.mainMenu, cmd = m.mainMenu.Update(msg)

	case stateDirSelect:
		switch msg := msg.(type) {
		case dirselect.Done:
			if msg.BackupDir != "" {
				m.config.BackupDir = msg.BackupDir
				m.mainMenuItemDelegate.backupDir = m.config.BackupDir
			}
			m.dirSelectModel = nil
			m.state = stateMainMenu
		default:
			_, cmd = m.dirSelectModel.Update(msg)
		}
	case stateZip:
		switch msg := msg.(type) {
		case zip.Done:
			m.zipModel = nil
			m.state = stateMainMenu
		default:
			_, cmd = m.zipModel.Update(msg)
		}
	case stateGithub:
		switch msg := msg.(type) {
		case github.Done:
			m.githubModel = nil
			m.state = stateMainMenu
		default:
			_, cmd = m.githubModel.Update(msg)
		}
	}
	return m, cmd
}

func (m *model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// margin, border, padding
	w, h := m.styles.ViewStyle.GetFrameSize()

	// available to nested models
	innerWidth := width - w
	innerHeight := height - h

	// title takes 2 lines, help takes 3, rest is available for main menu list
	mainMenuHeight := innerHeight - 5
	// if there is a lot of vertical space we don't want the list to fill it all
	// otherwise most of it will be empty and key help will be all the way at the bottom
	// TODO can we achieve this otherwise, that list will not use all the space? -> we could truncate the output
	// from Render()? that way we would mostly get the right results? think about itagain
	if mainMenuHeight > len(mainMenuItems)*3 {
		mainMenuHeight = len(mainMenuItems) * 3
	}
	if mainMenuHeight < 0 {
		mainMenuHeight = 2
	}
	m.mainMenu.SetSize(innerWidth, mainMenuHeight)

	m.helpView.Width = innerWidth

	if m.dirSelectModel != nil {
		m.dirSelectModel.SetSize(innerWidth, innerHeight)
	}
	if m.zipModel != nil {
		m.zipModel.SetSize(innerWidth, innerHeight)
	}
	if m.githubModel != nil {
		m.githubModel.SetSize(innerWidth, innerHeight)
	}
}

func (m *model) View() string {
	var content string
	styles := m.styles

	if m.confirmQuit {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Quit"),
			"",
			styles.NormalTextStyle.Render("Do you really want to quit?"),
			"",
			m.helpView.ShortHelpView(m.keyMap.confirmQuitKeys()),
		)
		return styles.ViewStyle.Render(content)
	}

	switch m.state {
	case stateMainMenu:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Backup"),
			"",
			m.mainMenu.View(),
			m.helpView.FullHelpView(m.keyMap.mainMenuKeys()),
		)
	case stateConfigError:
		var errText string
		if m.configError != nil {
			errText = m.configError.Error()
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Config Error"),
			"",
			styles.ErrorTextStyle.Render(errText),
			"",
			m.helpView.ShortHelpView(m.keyMap.configErrorKeys()),
		)
	case stateDirSelect:
		content = m.dirSelectModel.View()
	case stateZip:
		content = m.zipModel.View()
	case stateGithub:
		content = m.githubModel.View()
	}
	return styles.ViewStyle.Render(content)
}

type keyMap struct {
	CursorUp            key.Binding
	CursorDown          key.Binding
	Select              key.Binding
	Quit                key.Binding
	ConfirmQuit         key.Binding
	CancelQuit          key.Binding
	ConfigErrorContinue key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		CursorUp: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		ConfirmQuit: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "yes"),
		),
		CancelQuit: key.NewBinding(
			key.WithKeys("esc", "n"),
			key.WithHelp("esc/n", "no"),
		),
		ConfigErrorContinue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "quit"),
		),
	}
}

func (m keyMap) listKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp:             m.CursorUp,
		CursorDown:           m.CursorDown,
		PrevPage:             key.NewBinding(key.WithDisabled()),
		NextPage:             key.NewBinding(key.WithDisabled()),
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

func (m keyMap) mainMenuKeys() [][]key.Binding {
	return [][]key.Binding{
		{m.CursorUp, m.CursorDown},
		{m.Select, m.Quit},
	}
}

func (m keyMap) confirmQuitKeys() []key.Binding {
	return []key.Binding{m.CancelQuit, m.ConfirmQuit}
}

func (m keyMap) configErrorKeys() []key.Binding {
	return []key.Binding{m.ConfigErrorContinue}
}

const (
	mainMenuItemDirSelect int = iota
	mainMenuItemZip
	mainMenuItemGithub
)

type mainMenuItem int

// satisfy list.Item interface, we won't actually need this since filtering is disabled
func (i mainMenuItem) FilterValue() string {
	return ""
}

var mainMenuItems = []list.Item{
	mainMenuItem(mainMenuItemDirSelect),
	mainMenuItem(mainMenuItemZip),
	mainMenuItem(mainMenuItemGithub),
}

type mainMenuItemDelegate struct {
	backupDir string
	styles    style.Styles
}

func (d mainMenuItemDelegate) Height() int {
	return 2
}

func (d mainMenuItemDelegate) Spacing() int {
	return 1
}

func (d mainMenuItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d mainMenuItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item := int(listItem.(mainMenuItem))

	styles := d.styles

	var title string
	var description string

	switch item {
	case mainMenuItemDirSelect:
		title = "Change Backup Directory"
		description = d.backupDir
	case mainMenuItemZip:
		title = "Zip"
		description = "Zip Backup Directory"
	case mainMenuItemGithub:
		title = "GitHub"
		description = "Backup your repos"
	default:
		return
	}

	s := fmt.Sprintf("%s\n%s", styles.ListItemTitleStyle.Render(title), styles.ListItemDescriptionStyle.Render(description))
	if index == m.Index() {
		s = styles.ListItemSelectedStyle.Render(s)
	}

	fmt.Fprintf(w, s)
}

type config struct {
	BackupDir string `json:"backupDir"`
	Github    struct {
		Token string `json:"token"`
	} `json:"github"`
}

func loadConfig(configFile string) (config, error) {
	var config config

	if configFile != "" {
		s, err := os.ReadFile(configFile)
		if err != nil {
			return config, fmt.Errorf("could not read file: %w", err)
		}
		err = json.Unmarshal(s, &config)
		if err != nil {
			return config, fmt.Errorf("could not decode json: %w", err)
		}
	}

	// validate and set defaults
	if config.BackupDir == "" {
		backupDir, err := fs.DefaultBackupDir()
		if err != nil {
			return config, fmt.Errorf("could not get default backup directory: %w", err)
		}
		config.BackupDir = backupDir
	} else {
		absPath, err := fs.AbsPath(config.BackupDir)
		if err != nil {
			return config, fmt.Errorf("invalid directory: %w", err)
		}
		config.BackupDir = absPath
	}

	return config, nil
}
