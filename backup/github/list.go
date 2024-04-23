package github

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type List struct {
	repos             []Repo
	reposList         list.Model
	reposListDelegate *itemDelegate
	keyMap            reposLoadedKeyMap
}

func NewList(repos []Repo, keyMap reposLoadedKeyMap) *List {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = r
	}
	// initially select all items
	reposListDelegate := NewItemDelegate()
	for _, repo := range repos {
		reposListDelegate.Selected[repo.Id] = struct{}{}
	}
	reposList := list.New(items, reposListDelegate, 0, 0)
	reposList.SetFilteringEnabled(false)
	reposList.SetShowHelp(false)
	reposList.DisableQuitKeybindings()
	reposList.SetShowStatusBar(false)
	reposList.SetShowPagination(true)
	reposList.SetShowTitle(false)
	reposList.KeyMap = keyMap.listKeyMap()

	return &List{
		repos:             repos,
		reposList:         reposList,
		reposListDelegate: reposListDelegate,
		keyMap:            keyMap,
	}
}

func (l *List) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, l.keyMap.Select):
			repo, ok := l.reposList.SelectedItem().(Repo)
			if ok {
				_, selected := l.reposListDelegate.Selected[repo.Id]
				if selected {
					delete(l.reposListDelegate.Selected, repo.Id)
				} else {
					l.reposListDelegate.Selected[repo.Id] = struct{}{}
				}
			}
		case key.Matches(msg, l.keyMap.SelectAll):
			if len(l.reposListDelegate.Selected) > 0 {
				// unselect all
				l.reposListDelegate.Selected = map[string]struct{}{}
			} else {
				// select all
				for _, repo := range l.repos {
					l.reposListDelegate.Selected[repo.Id] = struct{}{}
				}
			}
		default:
			l.reposList, cmd = l.reposList.Update(msg)
		}
	default:
		l.reposList, cmd = l.reposList.Update(msg)
	}
	return cmd
}

func (l *List) SetSize(w, h int) {
	l.reposList.SetSize(w, h)
}

func (l *List) View() string {
	return l.reposList.View()
}

func (l *List) Selected() []Repo {
	var selected []Repo
	for _, r := range l.repos {
		if _, ok := l.reposListDelegate.Selected[r.Id]; ok {
			selected = append(selected, r)
		}
	}
	return selected
}

func (r Repo) FilterValue() string {
	return r.Id
}

type itemDelegate struct {
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style

	Selected map[string]struct{}
}

func NewItemDelegate() *itemDelegate {
	return &itemDelegate{
		itemStyle:         lipgloss.NewStyle().PaddingLeft(4),
		selectedItemStyle: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")),
		Selected:          map[string]struct{}{},
	}
}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	repo, ok := listItem.(Repo)
	if !ok {
		return
	}

	_, selected := d.Selected[repo.Id]

	var s string
	if selected {
		s = fmt.Sprintf("[x] %v", repo.Name)
	} else {
		s = fmt.Sprintf("[ ] %v", repo.Name)
	}

	if index == m.Index() {
		s = d.selectedItemStyle.Render("> " + s)
	} else {
		s = d.itemStyle.Render(s)
	}

	fmt.Fprintf(w, s)
}
