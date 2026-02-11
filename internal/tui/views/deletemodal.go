package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DeleteModalModel represents the delete confirmation modal
type DeleteModalModel struct {
	width       int
	height      int
	clusterName string
}

// NewDeleteModalModel creates a new delete modal
func NewDeleteModalModel() DeleteModalModel {
	return DeleteModalModel{
		width:  80,
		height: 20,
	}
}

// SetSize updates the modal dimensions
func (m *DeleteModalModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetCluster sets the cluster to be deleted
func (m *DeleteModalModel) SetCluster(name string) {
	m.clusterName = name
}

// Update handles messages
func (m DeleteModalModel) Update(msg tea.Msg) (DeleteModalModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "enter":
			// TODO: Actually delete the cluster
			return m, func() tea.Msg {
				return ClusterDeletedMsg{ClusterName: m.clusterName}
			}

		case "n", "esc":
			// Cancel deletion
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateClusterList}
			}
		}
	}

	return m, nil
}

// ViewOver renders the modal over existing content
func (m DeleteModalModel) ViewOver(content string) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(50).
		Background(lipgloss.Color("235"))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Render("⚠ Delete Cluster")

	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("Are you sure you want to delete:\n\n  " + m.clusterName + "\n\nThis action cannot be undone.")

	actions := lipgloss.NewStyle().
		Faint(true).
		Render("\n[y] Confirm  [n] Cancel")

	modal := modalStyle.Render(title + "\n\n" + message + actions)

	// Overlay on top of content
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
	)
}
