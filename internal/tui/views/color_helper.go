package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// statusColor returns the color for a given status
func statusColor(status string) lipgloss.Color {
	switch status {
	case "running":
		return lipgloss.Color("82")
	case "pending":
		return lipgloss.Color("220")
	case "failed":
		return lipgloss.Color("196")
	case "creating":
		return lipgloss.Color("39")
	case "deleting":
		return lipgloss.Color("240")
	case "upgrading":
		return lipgloss.Color("39")
	case "upgrade-failed":
		return lipgloss.Color("196")
	case "delete-failed":
		return lipgloss.Color("196")
	default:
		return lipgloss.Color("240")
	}
}

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
	case "upgrading":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("⟳ upgrading")
	case "upgrade-failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ upgrade-failed")
	case "delete-failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ delete-failed")
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
