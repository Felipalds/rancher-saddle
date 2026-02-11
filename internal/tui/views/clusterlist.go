package views

import (
	"fmt"
	"time"

	"github.com/Felipalds/go-kubernetes-helper/internal/config"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// ClusterListModel displays all clusters in a table
type ClusterListModel struct {
	table         table.Model
	width         int
	height        int
	clusters      map[string]*config.ClusterConfig
	clusterNames  []string
	lastRefresh   time.Time
	autoRefresh   bool
	refreshTicker *time.Ticker
}

// NewClusterListModel creates a new cluster list view
func NewClusterListModel() ClusterListModel {
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
		{Title: "Nodes", Width: 7},
		{Title: "Provider", Width: 10},
		{Title: "Region", Width: 12},
		{Title: "Age", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return ClusterListModel{
		table:       t,
		width:       80,
		height:      20,
		autoRefresh: true,
		lastRefresh: time.Now(),
	}
}

// Init initializes the cluster list
func (m ClusterListModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadClusters(),
		m.tickRefresh(),
	)
}

// SetSize updates the table dimensions
func (m *ClusterListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(height - 4) // Account for borders and padding
	m.table.SetWidth(width - 4)
}

// Update handles messages
func (m ClusterListModel) Update(msg tea.Msg) (ClusterListModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n", "c":
			// Navigate to create form
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateCreateForm}
			}

		case "d":
			// Delete selected cluster
			if len(m.clusterNames) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.clusterNames) {
					clusterName := m.clusterNames[selectedRow]
					return m, func() tea.Msg {
						return StateChangeMsg{
							NewState: StateDeleteConfirm,
							Data:     clusterName,
						}
					}
				}
			}

		case "r":
			// Manual refresh
			return m, m.loadClusters()

		case "enter":
			// Show cluster details
			if len(m.clusterNames) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.clusterNames) {
					// TODO: Implement details view
					return m, nil
				}
			}
		}

	case clustersLoadedMsg:
		m.clusters = msg.clusters
		m.clusterNames = msg.names
		m.lastRefresh = time.Now()
		m.updateTable()
		return m, nil

	case tickMsg:
		// Auto-refresh every 5 seconds
		if m.autoRefresh {
			return m, tea.Batch(m.loadClusters(), m.tickRefresh())
		}
		return m, m.tickRefresh()
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the cluster list
func (m ClusterListModel) View() string {
	if len(m.clusters) == 0 {
		return m.emptyState()
	}

	refreshIndicator := ""
	if m.autoRefresh {
		refreshIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Render(fmt.Sprintf(" ● Auto-refresh (5s) • Last: %s", formatDuration(time.Since(m.lastRefresh))))
	}

	return baseStyle.Render(m.table.View()) + "\n" + refreshIndicator
}

// emptyState shows a message when no clusters exist
func (m ClusterListModel) emptyState() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	message := "No clusters found.\n\nPress 'n' to create a new cluster."
	return emptyStyle.Render(message)
}

// updateTable refreshes the table rows with current cluster data
func (m *ClusterListModel) updateTable() {
	rows := []table.Row{}

	for _, name := range m.clusterNames {
		cluster := m.clusters[name]

		// Format status with color
		status := formatStatus(cluster.Status)

		// Get region
		region := "-"
		if r, ok := cluster.Provider.Config["region"].(string); ok {
			region = r
		}

		// Calculate age
		age := formatAge(cluster.CreatedAt)

		rows = append(rows, table.Row{
			name,
			status,
			fmt.Sprintf("%d", cluster.Cluster.InstanceCount),
			cluster.Provider.Type,
			region,
			age,
		})
	}

	m.table.SetRows(rows)
}

// loadClusters loads clusters from config
func (m ClusterListModel) loadClusters() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadClustersConfig("config.yaml")
		if err != nil {
			return clustersLoadedMsg{
				clusters: make(map[string]*config.ClusterConfig),
				names:    []string{},
			}
		}

		names := cfg.ListClusters()
		return clustersLoadedMsg{
			clusters: cfg.Clusters,
			names:    names,
		}
	}
}

// tickRefresh creates a ticker for auto-refresh
func (m ClusterListModel) tickRefresh() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Message types
type clustersLoadedMsg struct {
	clusters map[string]*config.ClusterConfig
	names    []string
}

type tickMsg time.Time

// formatStatus returns a colored status string
func formatStatus(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("● running")
	case "pending":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("⚠ pending")
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ failed")
	case "creating":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("⟳ creating")
	case "deleting":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("◐ deleting")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○ " + status)
	}
}

// formatAge converts time to human-readable age
func formatAge(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)
	hours := int(duration.Hours())

	if hours < 1 {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm", minutes)
	} else if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	} else {
		days := hours / 24
		return fmt.Sprintf("%dd", days)
	}
}

// formatDuration formats a duration to human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}
