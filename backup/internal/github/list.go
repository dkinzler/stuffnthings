package github

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RepoList struct {
	repos        []repo
	list         list.Model
	listDelegate *itemDelegate
	keyMap       keyMap
}

func NewList(repos []repo, keyMap keyMap) *RepoList {
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

	return &RepoList{
		repos:        repos,
		list:         reposList,
		listDelegate: reposListDelegate,
		keyMap:       keyMap,
	}
}

func (l *RepoList) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, l.keyMap.Select):
			repo, ok := l.list.SelectedItem().(repo)
			if ok {
				_, selected := l.listDelegate.Selected[repo.Id]
				if selected {
					delete(l.listDelegate.Selected, repo.Id)
				} else {
					l.listDelegate.Selected[repo.Id] = struct{}{}
				}
			}
		case key.Matches(msg, l.keyMap.SelectAll):
			if len(l.listDelegate.Selected) > 0 {
				// unselect all
				l.listDelegate.Selected = map[int]struct{}{}
			} else {
				// select all
				for _, repo := range l.repos {
					l.listDelegate.Selected[repo.Id] = struct{}{}
				}
			}
		default:
			l.list, cmd = l.list.Update(msg)
		}
	default:
		l.list, cmd = l.list.Update(msg)
	}
	return cmd
}

func (l *RepoList) SetSize(w, h int) {
	l.list.SetSize(w, h)
}

func (l *RepoList) View() string {
	return l.list.View()
}

func (l *RepoList) Selected() []repo {
	var selected []repo
	for _, r := range l.repos {
		if _, ok := l.listDelegate.Selected[r.Id]; ok {
			selected = append(selected, r)
		}
	}
	return selected
}

func (r repo) FilterValue() string {
	return r.Name
}

type itemDelegate struct {
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style

	Selected map[int]struct{}
}

func NewItemDelegate() *itemDelegate {
	return &itemDelegate{
		itemStyle:         lipgloss.NewStyle().PaddingLeft(4),
		selectedItemStyle: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")),
		Selected:          map[int]struct{}{},
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
	repo, ok := listItem.(repo)
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
