package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func main() {
	f, err := tea.LogToFile("/home/slartibartfast/Documents/bubbletea/log.txt", "log")
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
					m.inner = NewGithubModel(returnCmd())
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
