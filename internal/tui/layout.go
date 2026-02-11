package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// RenderWithSidebar combines sidebar and content into a two-column layout
func RenderWithSidebar(sidebar SidebarModel, content string) string {
	sidebarStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 1).
		Width(sidebar.GetWidth())

	contentStyle := lipgloss.NewStyle().
		Padding(1, 2)

	sidebarView := sidebarStyle.Render(sidebar.View())
	contentView := contentStyle.Render(content)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarView,
		contentView,
	)
}
