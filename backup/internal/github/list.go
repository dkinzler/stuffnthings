package github

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectReposList struct {
	repos []Repo

	list         list.Model
	listDelegate *selectReposItemDelegate

	keyMap keyMap
}

func newSelectReposList(repos []Repo, keyMap keyMap) *selectReposList {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = r
	}
	// initially select all items
	listDelegate := newSelectReposItemDelegate()
	for _, repo := range repos {
		listDelegate.selected[repo.Id] = struct{}{}
	}
	list := list.New(items, listDelegate, 0, 0)
	list.SetFilteringEnabled(false)
	list.SetShowHelp(false)
	list.DisableQuitKeybindings()
	list.SetShowStatusBar(false)
	list.SetShowPagination(true)
	list.SetShowTitle(false)
	list.KeyMap = keyMap.listKeyMap()

	return &selectReposList{
		repos:        repos,
		list:         list,
		listDelegate: listDelegate,
		keyMap:       keyMap,
	}
}

func (l *selectReposList) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, l.keyMap.Select):
			repo, ok := l.list.SelectedItem().(Repo)
			if ok {
				_, selected := l.listDelegate.selected[repo.Id]
				if selected {
					delete(l.listDelegate.selected, repo.Id)
				} else {
					l.listDelegate.selected[repo.Id] = struct{}{}
				}
			}
		case key.Matches(msg, l.keyMap.SelectAll):
			if len(l.listDelegate.selected) > 0 {
				// unselect all
				l.listDelegate.selected = map[int]struct{}{}
			} else {
				// select all
				for _, repo := range l.repos {
					l.listDelegate.selected[repo.Id] = struct{}{}
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

func (l *selectReposList) SetSize(width, height int) {
	l.list.SetSize(width, height)
}

func (l *selectReposList) View() string {
	return l.list.View()
}

func (l *selectReposList) Selected() []Repo {
	var selected []Repo
	for _, r := range l.repos {
		if _, ok := l.listDelegate.selected[r.Id]; ok {
			selected = append(selected, r)
		}
	}
	return selected
}

type selectReposItemDelegate struct {
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style

	selected map[int]struct{}
}

func newSelectReposItemDelegate() *selectReposItemDelegate {
	return &selectReposItemDelegate{
		itemStyle:         lipgloss.NewStyle().PaddingLeft(4),
		selectedItemStyle: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")),
		selected:          map[int]struct{}{},
	}
}

func (d selectReposItemDelegate) Height() int {
	return 1
}

func (d selectReposItemDelegate) Spacing() int {
	return 0
}

func (d selectReposItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d selectReposItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	repo, ok := listItem.(Repo)
	if !ok {
		return
	}

	_, selected := d.selected[repo.Id]

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

type cloneResultList struct {
	repos []Repo

	list         list.Model
	listDelegate *cloneResultItemDelegate

	keyMap keyMap
}

func newCloneResultList(repos []Repo, cloneResult map[int]bool, keyMap keyMap) *cloneResultList {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = r
	}

	listDelegate := newCloneResultItemDelegate(cloneResult)
	list := list.New(items, listDelegate, 0, 0)
	list.SetFilteringEnabled(false)
	list.SetShowHelp(false)
	list.DisableQuitKeybindings()
	list.SetShowStatusBar(false)
	list.SetShowPagination(true)
	list.SetShowTitle(false)
	list.KeyMap = keyMap.listKeyMap()
	list.KeyMap.CursorUp.SetEnabled(false)
	list.KeyMap.CursorDown.SetEnabled(false)

	return &cloneResultList{
		repos: repos,

		list:         list,
		listDelegate: listDelegate,

		keyMap: keyMap,
	}
}

func (l *cloneResultList) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.list, cmd = l.list.Update(msg)
	return cmd
}

func (l *cloneResultList) SetSize(width, height int) {
	l.list.SetSize(width, height)
}

func (l *cloneResultList) View() string {
	return l.list.View()
}

type cloneResultItemDelegate struct {
	itemStyle lipgloss.Style

	cloneResult map[int]bool
}

func newCloneResultItemDelegate(cloneResult map[int]bool) *cloneResultItemDelegate {
	return &cloneResultItemDelegate{
		itemStyle:   lipgloss.NewStyle().PaddingLeft(4),
		cloneResult: cloneResult,
	}
}

func (d cloneResultItemDelegate) Height() int {
	return 1
}

func (d cloneResultItemDelegate) Spacing() int {
	return 0
}

func (d cloneResultItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

var checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ef542")).Render("✓")
var cross = lipgloss.NewStyle().Foreground(lipgloss.Color("#de0d18")).Render("x")

func (d cloneResultItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	repo, ok := listItem.(Repo)
	if !ok {
		return
	}

	var s string

	success, ok := d.cloneResult[repo.Id]
	if !ok {
		s = fmt.Sprintf("%s  ?", repo.FullName)
	} else if success {
		s = fmt.Sprintf("%s  %s", repo.FullName, checkmark)
	} else {
		s = fmt.Sprintf("%s  %s", repo.FullName, cross)
	}

	s = d.itemStyle.Render(s)
	fmt.Fprintf(w, s)
}
