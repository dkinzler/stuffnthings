package main

import (
	"backuper/github"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/*
TODO what to do next?
maybe we could split view building into a separate file, or just put view code at the end?
what should we change about functionality?
* github
    * refactoring
        * turn switch around to
    * authentication
        * have an Authentication screen where we show the currently authenticated user
        * and maybe the other available ones
        * and on this screen we can either continue or login with a new account, or switch accounts
        * rename Authenticating state to CheckingAuthentication this will load the data from gh
        * when we e.g. login we run the gh command and then move back to the checking authentcation state
        * and eventually to the Authenticated state
        * when we press continue in authenticated we move to loading repos
    * list repos
        * maybe our own list for repos without list class?
            * this will probably be even simpler than using the predefined one
            * and it will have selection integrated directly
            * ability to select/unselect all with a button, a?
            * default nothing selected?
        * but problem with that is that we won't have pagination and scrolling and co
        * so maybe keep the list but customize it?
        * can also customize the help text at the bottom so we can use the same style everywhere
    * cloning
        * in result just return the id, that is enough
    * a header at the top always that shows the current step or progress like 1/4?
* main screen
    * can just quit, no confirmation needed
    * ability to change directory for backup
        * for submodules we don't really need this, just put it in a subfolder, e.g. "github" inside backup dir
        * how do we implement that?
        * maybe just a menu point that shows the current backup folder
            * if we press enter we open a screen with a textfield where we can change and then return
    * add a command to zip the backup folder up quickly
        * for that command we will first enter zip file name
    * to keep it simple whenever we enter a submodule we create that module new, i.e. we don't keep old one around
    * rename onExit command to just exit, or return?
* styling
    * have a simple style, we don't need a border?
    * have a heading in each screen, and some margin, padding
    * a consistent bottom
    * there are also lipgloss place horizontal and placevertical methods that might be interesting?
    * we probably want to use the terminal width/height or size changed events
        * can then propogate this down to our other models to use correct sizes?
* testing
    * think a bit about testing and how to do it
    * we could create a model and feed it articial events and see what commands and new state it returns?
    * i.e. just testing the state machine
* rename module to backup instead of backuper?
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

type Model struct {
	list  list.Model
	inner tea.Model
}

func NewModel() Model {
	items := []list.Item{
		item("Github"),
	}
	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.SetFilteringEnabled(false)

	return Model{
		list:  list,
		inner: nil,
	}
}

type returnMsg struct{}

func returnCmd() tea.Cmd {
	return func() tea.Msg {
		return returnMsg{}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		//TODO have to take care to not overlap keybinds with other views
		case "ctrl+c", "q":
			return m, tea.Quit
			// TODO handle returning to main screen here
		// case "esc":
		// 	m.confirmClose = true
		// case "y":
		// 	if m.confirmClose {
		// 		cmd = m.onExit
		// 	}
		// case "n":
		// 	if m.confirmClose {
		// 		m.confirmClose = false
		// 	}
		case "enter":
			// TODO bug
			// when we are filtering we want to be able to press enter to apply the filer
			// how could we distinguish here if we are currently filtering
			// maybe with list.FilterState there is filtering, unfiltered, filtered?
			// or we can just disable filtering
			if m.inner == nil {
				_, ok := m.list.SelectedItem().(item)
				if ok {
					// TODO should we call init here on that model and return that command? probably?
					m.inner = github.NewModel(returnCmd())
					cmd = m.inner.Init()
				}
				return m, cmd
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case returnMsg:
		m.inner = nil
		return m, nil
	}
	if m.inner != nil {
		m.inner, cmd = m.inner.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.inner != nil {
		return m.inner.View()
	} else {
		return m.list.View()
	}
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
