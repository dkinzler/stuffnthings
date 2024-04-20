package main

import (
	"backuper/github"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/*
TODO
what should we change about functionality?
* HUGE BUG
    * dir handling, if we pass "~/.." it will create a new directory named "~" in the current one
    * this is because the argument to commands are passed in "~", in the cli that handles the same
    * how to fix?
        * convert to absolute path somehow?
        * replace ~ with $HOME? -> this would work, but there are probably other cases that are still buggy
* how to do a help text
* define style of main menu
    * define global styles, that can be passed to others
    * have a global "wrapper" style that we use in main model .view that adds a bit of padding/margin
* split of main menu view code
* github
    * list repos
        * can customize the help text at the bottom so we can use the same style everywhere
* backup dir config
    * do some validation checks, e.g. don't allow empty directory
* zip
    * do some validation, e.g. empty file, or have a default backup.zip if it is empty
* figure out how to do the bubbles/help thing at the bottom
    * can have it consistently everywhere
* error handling
    * how to get back better errors from commands? we probably have to read out/err streams?
* ui and styling
    * to fix ui go screen by screen
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

	backupDirTextInput      textinput.Model
	backupDirTextInputValid bool

	zipTextInput textinput.Model
	zipError     error
}

func NewModel() Model {
	items := []list.Item{
		item("Github"),
		item("Backup"),
		item("Zip"),
	}
	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.SetFilteringEnabled(false)

	return Model{
		state:             MainMenu,
		confirmViewActive: false,
		list:              list,
		inner:             nil,
		backupDir:         defaultBackupDir(),
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
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
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
					m.backupDirTextInputValid = true
				case "Zip":
					m.state = Zip
					t := textinput.New()
					t.Focus()
					t.CharLimit = 250
					t.Width = 40
					m.zipTextInput = t
				}
				return m, cmd
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
				m.backupDir = dir
				m.backupDirTextInput = textinput.Model{}
				m.backupDirTextInputValid = true
				m.state = MainMenu
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
				cmd = zip(m.backupDir, file)
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
		return m.list.View()
	case BackupDir:
		return fmt.Sprintf(
			"Enter new backup directory:\n\n%s\n\n%s\n",
			m.backupDirTextInput.View(),
			"(enter) to confirm",
		)
	case Zip:
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
		c := exec.Command("zip", "-r", file, dir)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return zipResult{err: err}
		})()
	}
}
