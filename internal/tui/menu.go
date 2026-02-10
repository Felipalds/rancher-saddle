package tui

import (
	"fmt"
	"strings"

	"github.com/Felipalds/go-kubernetes-helper/internal/config"
	"github.com/Felipalds/go-kubernetes-helper/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuAction int

const (
	MenuList MenuAction = iota
	MenuCreate
	MenuDelete
	MenuExit
)

type MenuModel struct {
	choices  []string
	cursor   int
	selected MenuAction
	done     bool
	config   *model.Config
}

func NewMenuModel(cfg *model.Config) MenuModel {
	return MenuModel{
		choices: []string{
			"📋 List Clusters",
			"✨ Create New Cluster",
			"🗑️  Delete Cluster",
			"🚪 Exit",
		},
		cursor:   0,
		selected: -1,
		done:     false,
		config:   cfg,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.done = true
			m.selected = MenuExit
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = MenuAction(m.cursor)
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m MenuModel) View() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("🚀 Go Kubernetes Helper") + "\n\n")
	s.WriteString("What would you like to do?\n\n")

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = "▶"
			choice = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Render(choice)
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().
		Faint(true).
		Render("↑/↓: Navigate • Enter: Select • q/Esc: Exit"))
	s.WriteString("\n")

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(s.String())
}

func (m MenuModel) Done() bool {
	return m.done
}

func (m MenuModel) SelectedAction() MenuAction {
	return m.selected
}

// DeleteMenuModel shows list of clusters to delete
type DeleteMenuModel struct {
	clusters []string
	cursor   int
	selected int
	done     bool
	canceled bool
}

func NewDeleteMenuModel() (DeleteMenuModel, error) {
	cfg, err := config.LoadClustersConfig("config.yaml")
	if err != nil {
		return DeleteMenuModel{}, err
	}

	names := cfg.ListClusters()

	return DeleteMenuModel{
		clusters: names,
		cursor:   0,
		selected: -1,
		done:     false,
		canceled: false,
	}, nil
}

func (m DeleteMenuModel) Init() tea.Cmd {
	return nil
}

func (m DeleteMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.done = true
			m.canceled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.clusters)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m DeleteMenuModel) View() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("🗑️  Delete Cluster") + "\n\n")

	if len(m.clusters) == 0 {
		s.WriteString("No clusters found.\n\n")
		s.WriteString(lipgloss.NewStyle().
			Faint(true).
			Render("Press any key to return to menu..."))
		return lipgloss.NewStyle().
			Padding(1, 2).
			Render(s.String())
	}

	s.WriteString("Select a cluster to delete:\n\n")

	for i, cluster := range m.clusters {
		cursor := " "
		if m.cursor == i {
			cursor = "▶"
			cluster = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196")).
				Render(cluster)
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, cluster))
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().
		Faint(true).
		Render("↑/↓: Navigate • Enter: Delete • q/Esc: Cancel"))
	s.WriteString("\n")

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(s.String())
}

func (m DeleteMenuModel) Done() bool {
	return m.done
}

func (m DeleteMenuModel) Canceled() bool {
	return m.canceled
}

func (m DeleteMenuModel) SelectedCluster() string {
	if m.selected >= 0 && m.selected < len(m.clusters) {
		return m.clusters[m.selected]
	}
	return ""
}
