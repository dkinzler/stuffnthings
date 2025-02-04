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
	stateExTest
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
	exModel        *exModel

	styles style.Styles
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

		// Note: instead of keeping track of each possible child/nested model, we could have used a single generic "innerModel" field of type tea.Model,
		// but it can sometimes be useful to know the concrete types e.g. to call a method not part of the tea.Model interface
		dirSelectModel: nil,
		zipModel:       nil,
		githubModel:    nil,
		exModel:        nil,

		styles: styles,
	}
}

func (m *model) Init() tea.Cmd {
	// TODO do we need to call Init on list and helpview?
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
				// TODO call setSize for inner models that need it
				case mainMenuItemDirSelect:
					m.state = stateDirSelect
					m.dirSelectModel = dirselect.NewModel(m.config.BackupDir, m.styles)
					cmd = m.dirSelectModel.Init()
				case mainMenuItemExTest:
					m.state = stateExTest
					m.exModel = newExModel(m.styles)
					cmd = m.exModel.Init()
				case mainMenuItemZip:
					m.state = stateZip
					m.zipModel = zip.NewModel(m.config.BackupDir, m.styles)
					cmd = m.zipModel.Init()
				case mainMenuItemGithub:
					m.state = stateGithub
					m.githubModel = github.NewModel(m.config.BackupDir, m.config.Github.Token, m.styles)
					// TODO we need to store viewWidth and height here?
					// m.githubModel.SetSize(m.viewWidth, m.viewHeight)
					// this is a hack
					cmd = tea.Batch(m.githubModel.Init(), tea.WindowSize())
				}
				return m, cmd
			}
		}
		m.mainMenu, cmd = m.mainMenu.Update(msg)

	case stateDirSelect:
		switch msg := msg.(type) {
		case dirselect.Done:
			if msg.BackupDir != "" {
				m.config.BackupDir = msg.BackupDir
				// TODO this would be less annoying with a shared pointer to a struct, but oh well
				m.mainMenuItemDelegate.backupDir = m.config.BackupDir
			}
			m.dirSelectModel = nil
			m.state = stateMainMenu
		default:
			_, cmd = m.dirSelectModel.Update(msg)
		}
	case stateExTest:
		switch msg := msg.(type) {
		default:
			_, cmd = m.exModel.Update(msg)
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
	// title takes 2 lines, help takes 3 lines, including empty lines and global margin/padding takes some space
	w, h := m.styles.ViewStyle.GetFrameSize()
	// TODO name this innerHeight and innerWidth?
	innerHeight := height - h - 5
	if innerHeight > 9 {
		innerHeight = 9
	}
	if innerHeight < 0 {
		innerHeight = 2
	}
	innerWidth := width - w
	m.mainMenu.SetSize(innerWidth, innerHeight)
	m.helpView.Width = innerWidth
	if m.githubModel != nil {
		m.githubModel.SetSize(width-w, height-h)
	}
	// TODO call for other models that have SetSize
}

func (m *model) View() string {
	var content string
	styles := m.styles

	if m.confirmQuit {
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			// TODO better title text?
			styles.TitleStyle.Render("Confirm Quit"),
			styles.NormalTextStyle.Render("Do you really want to quit?"),
			m.helpView.ShortHelpView(m.keyMap.confirmQuitKeys()),
		)
		return styles.ViewStyle.Render(content)
	}

	switch m.state {
	case stateMainMenu:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			styles.TitleStyle.Render("Backup"),
			m.mainMenu.View(),
			m.helpView.FullHelpView(m.keyMap.mainMenuKeys()),
		)
	case stateConfigError:
		var errText string
		if m.configError != nil {
			// TODO any fance error messages processing here
			errText = m.configError.Error()
		}
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			styles.TitleStyle.Render("Config Error"),
			styles.ErrorTextStyle.Render(errText),
			m.helpView.ShortHelpView(m.keyMap.configErrorKeys()),
		)
	case stateDirSelect:
		content = m.dirSelectModel.View()
	case stateExTest:
		content = m.exModel.View()
	case stateZip:
		content = m.zipModel.View()
	case stateGithub:
		content = m.githubModel.View()
	}
	return styles.ViewStyle.Render(content)
}

// TODO unexport these
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
	mainMenuItemExTest
)

// We won't actually need any of the methods since we use a custom delegate to draw the list items and don't enable filtering.
type mainMenuItem int

func (i mainMenuItem) FilterValue() string {
	return ""
}

func (i mainMenuItem) Title() string {
	return ""
}

func (i mainMenuItem) Description() string {
	return ""
}

var mainMenuItems = []list.Item{
	mainMenuItem(mainMenuItemDirSelect),
	mainMenuItem(mainMenuItemZip),
	mainMenuItem(mainMenuItemGithub),
	// TODO delete this
	mainMenuItem(mainMenuItemExTest),
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

	// TODO make these more consistent, i.e. add all these classes already to Styles struct
	itemTitleStyle := d.styles.NormalTextStyle
	itemDescriptionStyle := d.styles.NormalTextStyle.Faint(true)
	selectedItemStyle := d.styles.SelectedListItemStyle

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
	case mainMenuItemExTest:
		title = "ExTest"
		description = "Great Test"
	default:
		return
	}

	s := fmt.Sprintf("%s\n%s", itemTitleStyle.Render(title), itemDescriptionStyle.Render(description))
	if index == m.Index() {
		s = selectedItemStyle.Render(s)
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
		absPath, err := fs.GetAbsPath(config.BackupDir)
		if err != nil {
			return config, fmt.Errorf("invalid directory: %w", err)
		}
		config.BackupDir = absPath
	}

	return config, nil
}
