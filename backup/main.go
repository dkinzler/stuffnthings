package main

import (
	"backuper/github"
	"fmt"
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
* refactoring
    * add confirm view to state -> ConfirmGoBack
    * save current state, have a method to switch state, which sets last and current state
    * when confirm view gets accepted can move back to last state
    * do some renames so that it is clerar which variable is from which state
        * rename list to mainMenuList
        * rename keyMap to mainMenukeyMap and helpView to mainMenuHelpView
        * move those up to mainMenuList
    * backup
        * create keymap keep it in Model with backup fields
        * create helpmodel
    * zip
        * create keymap and help model
        * make zipError a string, because we want to display a string
            * so whatever we get from result we need to transform or we transform it already in zip function?
    * create all the models when in NewModel()
        * and then reset them when appropriate
        * i.e. do this for backup and zip, for mainMenu we already do it
    * mainMenu create custom item delegate to control rendering
        * have a description for each
        * for backup we want to show the currently selected dir
        * we either have to set this is a var in the delegate and update it
            * or we can somehow access the model?
    * styling
        * define Styles struct with different fields
        * have a method defaultStyles() that returns a Styles instance
        * save this in models and pass it to GithubModel
        * what styles? keep it very simple
            * ViewStyle for the content wrapper
            * HelpStyle for help text everywhere
            * ListItem Style
            * SelectedListItem Style
            * TitleStyle for headers
                * can we make text a bit bigger or only e.g. bold?
            * ErrorStyle in red?
            * NormalTextStyle might be same as ListItemStyle?
        * need to figure out how to set these everywhere
            * for list we do it with the delegate?
            * for help we need to pass it to help.Model -> but hthis already requires certain styles?
    * refactor View()
        * is lipgloss.PlaceHorizontal and PlaceVertical useful?
        * can use fmt.Sprintf everywhere, as a sequence of rendering different things
            * title, content, help, maybe an error
        * apply ViewStyle globally at the end
    * more styling
        * figure out how to handle windowresizes
            * for mainMenu
                * we want to set the available height, we will now the necessary height
                    * because we know how many lines the main menu list item delegate will use
            * just make some useful defaults, don't try to handle every case, if someone tries to run this in a window with only 2 lines fuck em
    * github refactor
        * can we handle cloningRepos and reposCloned differently?
            * just show a done text at the end if we are done but not a separate reposCloned state?
            * just keep it like this
        * do we need to group or rename any model fields?
        * create keymaps and help models for the different screens
            * need one for Authenticated and AuthenticationError
            * need one for ListRepos, same thing as for mainMenu list = needs a conversion function
        * update ListRepo screen
            * add custom help view
            * need to run the same functions as on main menu to remove title and co
        * cloningrepos screen
            * show all repos initially with a "?", need to make sure that we are consistent
                * in the iteration order
        * refactor view and styling
            * can basically use same styles everywhere, don't need anything special
    * error handling
        * how to get back better errors from commands? we probably have to read out/err streams?
* testing
    * think a bit about testing and how to do it
    * we could create a model and feed it articial events and see what commands and new state it returns?
    * i.e. just testing the state machine
* rename module to backup instead of backuper?
* optional
    * a header at the top always that shows the current step or progress like 1/4?
    * validate backup dir that it would be writable? don't really need this, keep this on the user
*/

var docStyle = lipgloss.NewStyle().Margin(1, 2)

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
	BackupDir
	Zip
	Inner
)

type Model struct {
	state             State
	confirmViewActive bool

	list list.Model

	inner tea.Model

	backupDir string

	backupDirTextInput    textinput.Model
	backupDirInputInvalid bool

	zipTextInput    textinput.Model
	zipInputInvalid bool
	zipError        error

	keyMap   mainMenuKeyMap
	helpView help.Model
}

func NewModel() Model {
	keyMap := defaultMainMenuKeyMap()

	items := []list.Item{
		item("Github"),
		item("Backup"),
		item("Zip"),
	}
	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.SetFilteringEnabled(false)
	list.SetShowHelp(false)
	list.DisableQuitKeybindings()
	list.SetShowStatusBar(false)
	list.SetShowPagination(false)
	list.SetShowTitle(false)
	list.KeyMap = keyMap.listKeyMap()

	return Model{
		state:             MainMenu,
		confirmViewActive: false,
		list:              list,
		inner:             nil,
		backupDir:         defaultBackupDir(),
		keyMap:            keyMap,
		helpView:          help.New(),
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
	// handle global messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if !m.confirmViewActive && m.state != MainMenu {
				m.confirmViewActive = true
			}
			return m, nil
		case "y":
			if m.confirmViewActive {
				m.confirmViewActive = false
				m.state = MainMenu
				m.inner = nil
				return m, nil
			}
		case "n":
			if m.confirmViewActive {
				m.confirmViewActive = false
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		// TODO yes this is what we need, need to do this everywhere we have a list
		h, _ := docStyle.GetFrameSize()
		// m.list.SetSize(msg.Width-h, msg.Height-v)
		m.list.SetSize(msg.Width-h, 12)
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
				s := string(m.list.SelectedItem().(item))
				switch s {
				case "Github":
					m.state = Inner
					m.inner = github.NewModel(m.backupDir)
					cmd = m.inner.Init()
				case "Backup":
					m.state = BackupDir
					t := textinput.New()
					t.Placeholder = m.backupDir
					t.Focus()
					t.CharLimit = 250
					t.Width = 40
					m.backupDirTextInput = t
					m.backupDirInputInvalid = false
				case "Zip":
					m.state = Zip
					t := textinput.New()
					t.Focus()
					t.CharLimit = 250
					t.Width = 40
					m.zipTextInput = t
					m.zipInputInvalid = false
				}
				return m, cmd
			case "?":
				m.helpView.ShowAll = !m.helpView.ShowAll
			}
		}
		m.list, cmd = m.list.Update(msg)
	case BackupDir:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// TODO check if valid
				dir := m.backupDirTextInput.Value()
				if dir == "" {
					m.backupDirInputInvalid = true
				} else {
					m.backupDir = dir
					m.backupDirTextInput = textinput.Model{}
					m.backupDirInputInvalid = false
					m.state = MainMenu
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
					m.zipInputInvalid = true
				} else {
					m.zipInputInvalid = false
					cmd = zip(m.backupDir, file)
				}
				return m, cmd
			}
		case zipResult:
			if msg.err == nil {
				m.zipError = nil
				m.zipTextInput = textinput.Model{}
				m.state = MainMenu
				return m, cmd
			} else {
				m.zipError = msg.err
			}
		}
		m.zipTextInput, cmd = m.zipTextInput.Update(msg)
	case Inner:
		m.inner, cmd = m.inner.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.confirmViewActive {
		return docStyle.Render("Do you really want to return to main screen?")
	}
	switch m.state {
	case MainMenu:
		return docStyle.Render(fmt.Sprintf("%s\n\n%s\n\n", m.list.View(), m.helpView.View(m.keyMap)))
	case BackupDir:
		if m.backupDirInputInvalid {
			return fmt.Sprintf(
				"Enter new backup directory:\n\n%s\n\n%s\n\n%s\n",
				m.backupDirTextInput.View(),
				"Invalid directory.",
				"(enter) to confirm",
			)
		}
		return fmt.Sprintf(
			"Enter new backup directory:\n\n%s\n\n%s\n",
			m.backupDirTextInput.View(),
			"(enter) to confirm",
		)
	case Zip:
		if m.zipInputInvalid {
			return fmt.Sprintf(
				"Zip output file:\n\n%s\n\n%s\n\n%s\n",
				m.zipTextInput.View(),
				"Invalid input.",
				"(enter) to zip",
			)
		}
		if m.zipError == nil {
			return fmt.Sprintf(
				"Zip output file:\n\n%s\n\n%s\n",
				m.zipTextInput.View(),
				"(enter) to zip",
			)
		} else {
			return fmt.Sprintf(
				"Zip output file:\n\n%s\n\n%s\n\n %s: %v\n",
				m.zipTextInput.View(),
				"zip failed",
				m.zipError,
				"(enter) to try again",
			)
		}
	case Inner:
		return m.inner.View()
	}
	return ""
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
	CursorUp      key.Binding
	CursorDown    key.Binding
	ShowFullHelp  key.Binding
	CloseFullHelp key.Binding
	Select        key.Binding
	Exit          key.Binding
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
		ShowFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),
		CloseFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "less"),
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
		ShowFullHelp:         m.ShowFullHelp,
		CloseFullHelp:        m.CloseFullHelp,
	}
}

func (m mainMenuKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.CursorUp,
		m.CursorDown,
		m.Select,
		m.ShowFullHelp,
	}
}

func (m mainMenuKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.CursorUp, m.CursorDown, m.Select},
		{m.ShowFullHelp, m.CloseFullHelp, m.Exit},
	}
}
