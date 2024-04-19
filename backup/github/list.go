package github

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type List struct {
	// TODO in general, see also the TODO below we probably want to wrap all this in a custom struct
	// that internally has this list and delegate
	// that model can then handle the space key event, we don't have to handle it here
	// and we have a simple method to get a list of the selected repos
	// yesyes
	repos     []Repo
	reposList *list.Model
	// TODO might want to use a pointer here? so we can e.g. reset selected items more easily
	// otherwise assigning a new map to reposListDelegate.Selected might not have an effect on the
	// copy passed to reposList
	reposListDelegate *itemDelegate
}

func NewList(repos []Repo) *List {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = r
	}
	reposListDelegate := NewItemDelegate()
	// TODO what to do about these values, how can we make it fit where we don't just show a single item?
	reposList := list.New(items, reposListDelegate, 0, 0)
	return &List{
		repos:             repos,
		reposList:         &reposList,
		reposListDelegate: &reposListDelegate,
	}
}

func (l *List) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			repo, ok := l.reposList.SelectedItem().(Repo)
			if ok {
				_, selected := l.reposListDelegate.Selected[repo.Id]
				if selected {
					delete(l.reposListDelegate.Selected, repo.Id)
				} else {
					l.reposListDelegate.Selected[repo.Id] = struct{}{}
				}
			}
		}
	}
	return cmd
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

func NewItemDelegate() itemDelegate {
	return itemDelegate{
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
