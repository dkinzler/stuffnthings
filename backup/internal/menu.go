package internal

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// How to add a new component
// - model should be returned as a pointer and create a "Done" message to exit back to the main menu, see dirselect.go for a simple example
//   - if your model needs window size updates, add a SetSize method and call it in mainMenuModel.Update(), you will also need to call SetSize once when the child model is created
// - add a state constant, a field for the model pointer in the mainMenuModel struct and add appropriate cases to the Update method
// - add a mainMenuItem constant, update the global mainMenuItems variable and update the Render method of mainMenuItemDelegate

type state int

const (
	mainMenu state = iota
	dirSelect
	zip
	github
)

// Most UI models/components will need this information.
// Note:
// - another approach would be to pass these values individually whenever a new child model is created
// - cannot have a child model in a separate package since that would create an import cycle, would need to move commonState to its own package
type commonState struct {
	backupDir string
	styles    styles
}

type mainMenuModel struct {
	state       state
	commonState *commonState

	mainMenuList         list.Model
	mainMenuItemDelegate *mainMenuItemDelegate
	helpView             help.Model
	keyMap               mainMenuKeyMap

	dirSelectModel *dirSelectModel
}

func NewMainMenuModel() *mainMenuModel {
	backupDir, err := defaultBackupDir()
	if err != nil {
		panic(err)
	}
	styles := defaultStyles()
	commonState := &commonState{
		backupDir: backupDir,
		styles:    styles,
	}

	keyMap := defaultMainMenuKeyMap()

	itemDelegate := &mainMenuItemDelegate{commonState: commonState}
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
	helpView.ShowAll = true

	return &mainMenuModel{
		state:       mainMenu,
		commonState: commonState,

		mainMenuList:         list,
		mainMenuItemDelegate: itemDelegate,
		helpView:             helpView,
		keyMap:               keyMap,

		// Note: instead of keeping track of each possible child model, we could have used a single generic "innerModel" field of type tea.Model,
		// but it can sometimes be useful to know the concrete types e.g. to call a method not part of the tea.Model interface.
		dirSelectModel: nil,
	}
}

func (m *mainMenuModel) Init() tea.Cmd {
	// TODO do we need to call Init on list and helpview?
	return nil
}

func (m *mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Note: instead of using SetSize on child models we could have passed the WindowSizeMsg through to them
		m.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd

	switch m.state {

	case mainMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.Select):
				s := int(m.mainMenuList.SelectedItem().(mainMenuItem))
				switch s {
				// TODO call setSize for inner models that need it
				case mainMenuItemDirSelect:
					m.state = dirSelect
					m.dirSelectModel = newDirSelectModel(m.commonState)
					cmd = m.dirSelectModel.Init()
				case mainMenuItemZip:
					// m.state = Zip
					// m.zipModel = zip.NewModel(m.backupDir, m.styles)
					// cmd = m.zipModel.Init()
				case mainMenuItemGithub:
					// m.state = Github
					// m.githubModel = github.NewModel(m.backupDir, m.styles)
					// m.githubModel.SetSize(m.viewWidth, m.viewHeight)
					// cmd = m.githubModel.Init()
				}
				return m, cmd
			}
		}
		m.mainMenuList, cmd = m.mainMenuList.Update(msg)

	case dirSelect:
		switch msg := msg.(type) {
		case dirSelectDone:
			if msg.backupDir != "" {
				m.commonState.backupDir = msg.backupDir
			}
			m.dirSelectModel = nil
			m.state = mainMenu
		default:
			_, cmd = m.dirSelectModel.Update(msg)
		}
	case zip:
		// switch msg := msg.(type) {
		// case zip.Done:
		// 	m.zipModel = nil
		// 	m.state = MainMenu
		// default:
		// 	_, cmd = m.zipModel.Update(msg)
		// }
	case github:
		// switch msg := msg.(type) {
		// case github.Done:
		// 	m.githubModel = nil
		// 	m.state = MainMenu
		// default:
		// 	_, cmd = m.githubModel.Update(msg)
		// }
	}
	return m, cmd
}

func (m *mainMenuModel) SetSize(width, height int) {
	// title takes 2 lines, help takes 3 lines, including empty lines and global margin/padding takes some space
	_, h := m.commonState.styles.ViewStyle.GetFrameSize()
	height = height - h - 5
	if height > 9 {
		height = 9
	}
	if height < 0 {
		height = 2
	}
	m.mainMenuList.SetSize(width, height)
	// TODO if in inner model call SetSize on child model
}

func (m *mainMenuModel) View() string {
	var content string
	styles := m.commonState.styles

	switch m.state {
	case mainMenu:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			styles.TitleStyle.Render("Backup"),
			m.mainMenuList.View(),
			m.helpView.View(m.keyMap),
		)
	case dirSelect:
		content = m.dirSelectModel.View()
	case zip:
		// content = m.zipModel.View()
	case github:
		// content = m.githubModel.View()
	}
	return styles.ViewStyle.Render(content)
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

const (
	mainMenuItemDirSelect int = iota
	mainMenuItemZip
	mainMenuItemGithub
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
}

type mainMenuItemDelegate struct {
	commonState *commonState
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
	itemTitleStyle := d.commonState.styles.NormalTextStyle
	itemDescriptionStyle := d.commonState.styles.NormalTextStyle.Faint(true)
	selectedItemStyle := d.commonState.styles.SelectedListItemStyle

	var title string
	var description string

	switch item {
	case mainMenuItemDirSelect:
		title = "Change Backup Directory"
		description = d.commonState.backupDir
	case mainMenuItemZip:
		title = "Zip"
		description = "Zip Backup Directory"
	case mainMenuItemGithub:
		title = "GitHub"
		description = "Backup your repos"
	default:
		return
	}

	s := fmt.Sprintf("%s\n%s", itemTitleStyle.Render(title), itemDescriptionStyle.Render(description))
	if index == m.Index() {
		s = selectedItemStyle.Render(s)
	}

	fmt.Fprintf(w, s)
}
