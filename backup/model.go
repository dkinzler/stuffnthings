package main

import (
	"backup/config/backupdir"
	"backup/github"
	"backup/styles"
	"backup/zip"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type State int

const (
	MainMenu State = iota
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
	state State

	mainMenuList         list.Model
	mainMenuItemDelegate *mainMenuItemDelegate

	backupDir      string
	backupDirModel *backupdir.Model

	zipModel *zip.Model

	githubModel *github.Model

	keyMap keyMap

	help help.Model

	styles styles.Styles

	viewWidth, viewHeight int
}

func NewModel() Model {
	keyMap := defaultKeyMap()
	backupDir := defaultBackupDir()
	styles := styles.DefaultStyles()

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

	helpView := help.New()
	helpView.Styles = styles.HelpStyles
	helpView.ShowAll = true

	return Model{
		state:                MainMenu,
		mainMenuList:         list,
		mainMenuItemDelegate: itemDelegate,
		keyMap:               keyMap,
		backupDir:            backupDir,
		backupDirModel:       nil,
		zipModel:             nil,
		githubModel:          nil,
		help:                 helpView,
		styles:               styles,
	}
}

func defaultBackupDir() string {
	date := time.Now().Format(time.DateOnly)
	return filepath.Join("~/backup", date)
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
			switch {
			case key.Matches(msg, m.keyMap.Select):
				s := string(m.mainMenuList.SelectedItem().(item))
				switch s {
				case MainMenuItemGithub:
					m.state = Github
					m.githubModel = github.NewModel(m.backupDir, m.styles)
					m.githubModel.SetSize(m.viewWidth, m.viewHeight)
					cmd = m.githubModel.Init()
				case MainMenuItemBackupDir:
					m.state = BackupDir
					m.backupDirModel = backupdir.NewModel(m.backupDir, m.styles)
					cmd = m.backupDirModel.Init()
				case MainMenuItemZip:
					m.state = Zip
					m.zipModel = zip.NewModel(m.backupDir, m.styles)
					cmd = m.zipModel.Init()
				}
				return m, cmd
			}
		}
		m.mainMenuList, cmd = m.mainMenuList.Update(msg)

	case BackupDir:
		switch msg := msg.(type) {
		case backupdir.BackupDirChanged:
			m.backupDir = msg.BackupDir
			m.mainMenuItemDelegate.backupDir = m.backupDir
			m.backupDirModel = nil
			m.state = MainMenu
		case backupdir.Done:
			m.backupDirModel = nil
			m.state = MainMenu
		default:
			_, cmd = m.backupDirModel.Update(msg)
		}
	case Zip:
		switch msg := msg.(type) {
		case zip.Done:
			m.zipModel = nil
			m.state = MainMenu
		default:
			_, cmd = m.zipModel.Update(msg)
		}
	case Github:
		switch msg := msg.(type) {
		case github.Done:
			m.githubModel = nil
			m.state = MainMenu
		default:
			_, cmd = m.githubModel.Update(msg)
		}
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
			m.help.View(m.keyMap),
		)
	case BackupDir:
		content = m.backupDirModel.View()
	case Zip:
		content = m.zipModel.View()
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

type keyMap struct {
	CursorUp   key.Binding
	CursorDown key.Binding
	Select     key.Binding
	Exit       key.Binding
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
		Exit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
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

func (m keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.CursorUp,
		m.CursorDown,
		m.Select,
		m.Exit,
	}
}

func (m keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.CursorUp, m.CursorDown},
		{m.Select, m.Exit},
	}
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
