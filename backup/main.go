package main

import (
	bexec "backup/exec"
	"backup/github"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/*
TODO
* retry cloning?
* testing
    * think a bit about testing and how to do it
    * we could create a model and feed it articial events and see what commands and new state it returns?
    * i.e. just testing the state machine
* optional
    * mainMenuKeyMap, backupDirKeyMap and zipKeyMap all through a single type with functions that return a keymap?
        * how could this be done?
* what could be changed in the future?
    * the way we handle esc, returning to main menu, should the inner model handle this how it likes? -> probably
    * we don't try to handle every case, if someone tries to run this in a window with only 2 lines they gonna have a bad time
    * a better way to handle window resizes?
        * right now we have to explicitely pass the size through to every model and update it whenever we create a new model and co
        * maybe another way is to periodically trigger a windowSizeMsg? can we get the current size somewhere or
*/

func main() {
	f, err := tea.LogToFile("log.txt", "log")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	log.SetOutput(f)

	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
	}
}

type State int

const (
	MainMenu State = iota
	ConfirmGoBack
	Github
	BackupDir
	Zip
)

const (
	MainMenuItemGithub    string = "github"
	MainMenuItemBackupDir string = "backup"
	MainMenuItemZip       string = "zip"
)

type Model struct {
	lastState State
	state     State

	mainMenuList         list.Model
	mainMenuItemDelegate *mainMenuItemDelegate
	mainMenuKeyMap       mainMenuKeyMap

	backupDirTextInput  textinput.Model
	backupDirInputValid bool
	backupDir           string
	backupDirKeyMap     backupDirKeyMap

	zipTextInput  textinput.Model
	zipInputValid bool

	zipError  string
	zipKeyMap zipKeyMap

	githubModel *github.Model

	helpView help.Model

	styles Styles

	viewWidth, viewHeight int
}

func NewModel() Model {
	keyMap := defaultMainMenuKeyMap()
	backupDir := defaultBackupDir()
	styles := defaultStyles()

	items := []list.Item{
		item(MainMenuItemGithub),
		item(MainMenuItemBackupDir),
		item(MainMenuItemZip),
	}
	itemDelegate := &mainMenuItemDelegate{backupDir: backupDir}
	itemDelegate.itemTitleStyle = styles.NormalTextStyle
	itemDelegate.itemDescriptionStyle = styles.NormalTextStyle.Faint(true)
	itemDelegate.selectedItemStyle = styles.SelectedListItemStyle
	list := list.New(items, itemDelegate, 0, 0)
	list.SetFilteringEnabled(false)
	list.SetShowHelp(false)
	list.DisableQuitKeybindings()
	list.SetShowStatusBar(false)
	list.SetShowPagination(false)
	list.SetShowTitle(false)
	list.KeyMap = keyMap.listKeyMap()

	bt := textinput.New()
	bt.Placeholder = backupDir
	bt.CharLimit = 250
	bt.Width = 40

	zt := textinput.New()
	zt.CharLimit = 250
	zt.Width = 40

	helpView := help.New()
	helpView.Styles = styles.HelpStyles
	helpView.ShowAll = true

	return Model{
		lastState:            MainMenu,
		state:                MainMenu,
		mainMenuList:         list,
		mainMenuItemDelegate: itemDelegate,
		mainMenuKeyMap:       keyMap,
		backupDirTextInput:   bt,
		backupDirInputValid:  true,
		backupDir:            backupDir,
		backupDirKeyMap:      defaultBackupDirKeyMap(),
		zipTextInput:         zt,
		zipInputValid:        true,
		zipError:             "",
		zipKeyMap:            defaultZipKeyMap(),
		githubModel:          nil,
		helpView:             helpView,
		styles:               styles,
	}
}

func defaultBackupDir() string {
	date := time.Now().Format(time.DateOnly)
	return filepath.Join("~/backup", date)
}

func (m *Model) setState(state State) {
	m.lastState = m.state
	m.state = state
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == BackupDir || m.state == Zip {
				m.setState(MainMenu)
			} else if m.state == Github {
				m.setState(ConfirmGoBack)
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		// title takes 2 lines, help takes 3 lines, including empty lines, global margin/padding takes some space
		// the rest we can use for the list
		w, h := m.styles.ViewStyle.GetFrameSize()
		m.viewWidth = msg.Width - w
		m.viewHeight = msg.Height - h
		height := msg.Height - h - 5
		if height > 9 {
			height = 9
		}
		if height < 0 {
			height = 2
		}
		m.mainMenuList.SetSize(msg.Width-w, height)
		if m.githubModel != nil {
			m.githubModel.SetSize(m.viewWidth, m.viewHeight)
		}
	}

	var cmd tea.Cmd

	switch m.state {

	case MainMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				s := string(m.mainMenuList.SelectedItem().(item))
				switch s {
				case MainMenuItemGithub:
					m.setState(Github)
					m.githubModel = github.NewModel(m.backupDir, github.Styles{
						ViewStyle:             m.styles.ViewStyle,
						TitleStyle:            m.styles.TitleStyle,
						NormalTextStyle:       m.styles.NormalTextStyle,
						ErrorTextStyle:        m.styles.ErrorTextStyle,
						SelectedListItemStyle: m.styles.SelectedListItemStyle,
						HelpStyles:            m.styles.HelpStyles,
					})
					m.githubModel.SetSize(m.viewWidth, m.viewHeight)
					cmd = m.githubModel.Init()
				case MainMenuItemBackupDir:
					m.setState(BackupDir)
					m.backupDirTextInput.Reset()
					m.backupDirTextInput.Placeholder = m.backupDir
					m.backupDirTextInput.Focus()
					m.backupDirInputValid = true
				case MainMenuItemZip:
					m.setState(Zip)
					m.zipTextInput.Reset()
					m.zipTextInput.Focus()
					m.zipInputValid = true
					m.zipError = ""
				}
				return m, cmd
			case "?":
				m.helpView.ShowAll = !m.helpView.ShowAll
			}
		}
		m.mainMenuList, cmd = m.mainMenuList.Update(msg)

	case ConfirmGoBack:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y":
				if m.state == Github {
					m.githubModel = nil
				}
				m.setState(MainMenu)
				return m, nil
			case "n":
				m.setState(m.lastState)
				return m, nil
			}
		}

	case BackupDir:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				dir := m.backupDirTextInput.Value()
				if dir == "" {
					m.backupDirInputValid = false
				} else {
					m.backupDir = dir
					m.mainMenuItemDelegate.backupDir = dir
					m.backupDirTextInput.Reset()
					m.backupDirInputValid = true
					m.setState(MainMenu)
				}
				return m, cmd
			}
		}
		m.backupDirTextInput, cmd = m.backupDirTextInput.Update(msg)
	case Zip:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				file := m.zipTextInput.Value()
				if file == "" {
					m.zipInputValid = false
				} else {
					m.zipInputValid = true
					cmd = zip(m.backupDir, file)
				}
				return m, cmd
			}
		case zipResult:
			if msg.err == nil {
				m.zipError = ""
				m.zipTextInput.Reset()
				m.setState(MainMenu)
				return m, cmd
			} else {
				m.zipError = msg.err.Error()
			}
		}
		m.zipTextInput, cmd = m.zipTextInput.Update(msg)
	case Github:
		_, cmd = m.githubModel.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	var content string

	switch m.state {
	case MainMenu:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			m.styles.TitleStyle.Render("Backup"),
			m.mainMenuList.View(),
			m.helpView.View(m.mainMenuKeyMap),
		)
	case ConfirmGoBack:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			m.styles.TitleStyle.Render("Backup"),
			m.styles.NormalTextStyle.Render("Do you want to return to the main menu?"),
			m.styles.NormalTextStyle.Render("<y> to confirm, <n> to cancel"),
		)
	case BackupDir:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n",
			m.styles.TitleStyle.Render("Change Backup Directory"),
			m.backupDirTextInput.View(),
		)
		if !m.backupDirInputValid {
			content = fmt.Sprintf("%s%s\n\n", content, m.styles.ErrorTextStyle.Render("invalid directory"))
		}
		content = fmt.Sprintf("%s%s\n", content, m.helpView.View(m.backupDirKeyMap))
	case Zip:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n",
			m.styles.TitleStyle.Render("Zip Backup Directory"),
			m.zipTextInput.View(),
		)
		if !m.zipInputValid {
			content = fmt.Sprintf("%s%s\n\n", content, m.styles.ErrorTextStyle.Render("invalid filename"))
		} else if m.zipError != "" {
			content = fmt.Sprintf("%s%s\n\n", content, m.styles.ErrorTextStyle.Render(m.zipError))
		}
		content = fmt.Sprintf("%s%s\n", content, m.helpView.View(m.zipKeyMap))
	case Github:
		content = m.githubModel.View()
	}
	return m.styles.ViewStyle.Render(content)
}

type item string

func (i item) FilterValue() string {
	return string(i)
}

func (i item) Title() string {
	return string(i)
}

func (i item) Description() string {
	return ""
}

type zipResult struct {
	err error
}

func zip(dir string, file string) tea.Cmd {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("zip -r %s %s", file, dir))
	// note that zip prints errors to stdout
	return bexec.Exec(cmd, func(err error, s string) tea.Msg {
		if err != nil {
			s = strings.TrimSpace(s)
			e := fmt.Errorf("%v: %v", err, s)
			return zipResult{err: e}
		}
		return zipResult{err: nil}
	}, true)
}

type mainMenuKeyMap struct {
	CursorUp   key.Binding
	CursorDown key.Binding
	Select     key.Binding
	Exit       key.Binding
}

func defaultMainMenuKeyMap() mainMenuKeyMap {
	return mainMenuKeyMap{
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
		Exit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (m mainMenuKeyMap) listKeyMap() list.KeyMap {
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

func (m mainMenuKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.CursorUp,
		m.CursorDown,
		m.Select,
		m.Exit,
	}
}

func (m mainMenuKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.CursorUp, m.CursorDown},
		{m.Select, m.Exit},
	}
}

type backupDirKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func defaultBackupDirKeyMap() backupDirKeyMap {
	return backupDirKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func (m backupDirKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Cancel, m.Confirm}
}

func (m backupDirKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.Cancel}, {m.Confirm}}
}

type zipKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func defaultZipKeyMap() zipKeyMap {
	return zipKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func (m zipKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Cancel, m.Confirm}
}

func (m zipKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.Cancel}, {m.Confirm}}
}

type mainMenuItemDelegate struct {
	itemTitleStyle       lipgloss.Style
	itemDescriptionStyle lipgloss.Style
	selectedItemStyle    lipgloss.Style

	backupDir string
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
	item := string(listItem.(item))

	var title string
	var description string

	switch item {
	case MainMenuItemGithub:
		title = "GitHub"
		description = "Backup your repos"
	case MainMenuItemBackupDir:
		title = "Change Backup Directory"
		description = d.backupDir
	case MainMenuItemZip:
		title = "Zip"
		description = "Zip Backup Directory"
	default:
		return
	}

	s := fmt.Sprintf("%s\n%s", d.itemTitleStyle.Render(title), d.itemDescriptionStyle.Render(description))
	if index == m.Index() {
		s = d.selectedItemStyle.Render(s)
	}

	fmt.Fprintf(w, s)
}

type Styles struct {
	ViewStyle             lipgloss.Style
	TitleStyle            lipgloss.Style
	NormalTextStyle       lipgloss.Style
	ErrorTextStyle        lipgloss.Style
	SelectedListItemStyle lipgloss.Style
	HelpStyles            help.Styles
}

func defaultStyles() Styles {
	// copied these styles from the charmbracelet/bubbles/help package
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#909090",
		Dark:  "#828282",
	})

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#626262",
	})

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#DDDADA",
		Dark:  "#3C3C3C",
	})
	helpStyles := help.Styles{
		Ellipsis:       sepStyle,
		ShortKey:       keyStyle,
		ShortDesc:      descStyle,
		ShortSeparator: sepStyle,
		FullKey:        keyStyle,
		FullDesc:       descStyle,
		FullSeparator:  sepStyle,
	}

	return Styles{
		ViewStyle:             lipgloss.NewStyle().Margin(1),
		TitleStyle:            lipgloss.NewStyle().Bold(true).Padding(0, 3).Background(lipgloss.Color("#fc03ec")).Foreground(lipgloss.Color("#ffffff")),
		NormalTextStyle:       lipgloss.NewStyle(),
		ErrorTextStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		SelectedListItemStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fc03ec")),
		HelpStyles:            helpStyles,
	}
}
