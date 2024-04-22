package main

import (
	"backuper/github"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
* don't need confirm dialog for backup and zip
* do we need it for github? probably
* refactoring
    * more styling
        * figure out how to handle windowresizes
            * for mainMenu
                * we want to set the available height, we will now the necessary height
                    * because we know how many lines the main menu list item delegate will use
            * just make some useful defaults, don't try to handle every case, if someone tries to run this in a window with only 2 lines fuck em
    * error handling
        * how to get back better errors from commands? we probably have to read out/err streams?
        * also for zip command, parse error message better
* testing
    * think a bit about testing and how to do it
    * we could create a model and feed it articial events and see what commands and new state it returns?
    * i.e. just testing the state machine
* rename module to backup instead of backuper?
* optional
    * a header at the top always that shows the current step or progress like 1/4?
    * validate backup dir that it would be writable? don't really need this, keep this on the user
    * mainMenuKeyMap, backupDirKeyMap and zipKeyMap all through a single type with functions that return a keymap?
* what could be changed in the future?
    * the way we handle esc, returning to main menu, should the inner model handle this how it likes? -> probably
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
	BackupDir
	Zip
	Inner
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
	zipError      string
	zipKeyMap     zipKeyMap

	inner tea.Model

	helpView help.Model

	styles Styles
}

func NewModel() Model {
	keyMap := defaultMainMenuKeyMap()
	backupDir := defaultBackupDir()
	styles := defaultStyles()

	items := []list.Item{
		item("github"),
		item("backup"),
		item("zip"),
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
		inner:                nil,
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
	// handle global messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state != ConfirmGoBack && m.state != MainMenu {
				m.setState(ConfirmGoBack)
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		// TODO yes this is what we need, need to do this everywhere we have a list
		h, _ := m.styles.ViewStyle.GetFrameSize()
		// m.list.SetSize(msg.Width-h, msg.Height-v)
		m.mainMenuList.SetSize(msg.Width-h, 9)
		// TODO forward these to inner?
	}

	var cmd tea.Cmd

	switch m.state {

	case MainMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// TODO disable filtering on list
				s := string(m.mainMenuList.SelectedItem().(item))
				switch s {
				// TODO should have these as constants mainMenuItemGithub
				case "github":
					m.setState(Inner)
					m.inner = github.NewModel(m.backupDir, github.Styles{
						ViewStyle:             m.styles.ViewStyle,
						TitleStyle:            m.styles.TitleStyle,
						NormalTextStyle:       m.styles.NormalTextStyle,
						ErrorTextStyle:        m.styles.ErrorTextStyle,
						SelectedListItemStyle: m.styles.SelectedListItemStyle,
						HelpStyles:            m.styles.HelpStyles,
					})
					cmd = m.inner.Init()
				case "backup":
					m.setState(BackupDir)
					m.backupDirTextInput.Reset()
					m.backupDirTextInput.Placeholder = m.backupDir
					m.backupDirTextInput.Focus()
					m.backupDirInputValid = true
				case "zip":
					m.setState(Zip)
					m.zipTextInput.Reset()
					m.zipTextInput.Focus()
					m.zipInputValid = true
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
				m.setState(MainMenu)
				m.inner = nil
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
				// TODO check if valid
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
	case Inner:
		m.inner, cmd = m.inner.Update(msg)
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
			"%s\n",
			m.styles.NormalTextStyle.Render("Do you want to return to the main menu? <y> to confirm, <n> to cancel"),
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
	case Inner:
		content = m.inner.View()
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
	return func() tea.Msg {
		c := exec.Command("sh", "-c", fmt.Sprintf("zip -r %s %s", file, dir))
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return zipResult{err: err}
		})()
	}
}

type mainMenuKeyMap struct {
	CursorUp   key.Binding
	CursorDown key.Binding
	// ShowFullHelp  key.Binding
	// CloseFullHelp key.Binding
	Select key.Binding
	Exit   key.Binding
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
		// ShowFullHelp: key.NewBinding(
		// 	key.WithKeys("?"),
		// 	key.WithHelp("?", "more"),
		// ),
		// CloseFullHelp: key.NewBinding(
		// 	key.WithKeys("?"),
		// 	key.WithHelp("?", "less"),
		// ),
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
	case "github":
		title = "GitHub"
		description = "Backup your repos"
	case "backup":
		title = "Change Backup Directory"
		description = d.backupDir
	case "zip":
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
	// TODO change these a bit
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#909090",
		Dark:  "#626262",
	})

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#4A4A4A",
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
