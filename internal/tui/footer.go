package tui

import (
	"github.com/Felipalds/go-kubernetes-helper/internal/tui/views"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// FooterModel represents the persistent footer with context-aware keybindings
type FooterModel struct {
	width int
	help  help.Model
}

// keyMap defines keybindings for each state
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	New      key.Binding
	Delete   key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// ShortHelp returns keybindings to show in the mini help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.New, k.Delete, k.Refresh},
		{k.Back, k.Help, k.Quit},
	}
}

var (
	clusterListKeys = keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		New: key.NewBinding(
			key.WithKeys("n", "c"),
			key.WithHelp("n", "new cluster"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}

	createFormKeys = keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "shift+tab"),
			key.WithHelp("↑/shift+tab", "prev field"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "tab"),
			key.WithHelp("↓/tab", "next field"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit/next"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
	}

	deleteModalKeys = keyMap{
		Enter: key.NewBinding(
			key.WithKeys("y", "enter"),
			key.WithHelp("y/enter", "confirm"),
		),
		Back: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
	}
)

// NewFooterModel creates a new footer
func NewFooterModel() FooterModel {
	h := help.New()
	h.ShowAll = false
	return FooterModel{
		width: 80,
		help:  h,
	}
}

// SetWidth updates the footer width
func (f *FooterModel) SetWidth(width int) {
	f.width = width
	f.help.Width = width - 4
}

// ViewForState renders the footer with context-aware keybindings
func (f FooterModel) ViewForState(state views.AppState) string {
	footerStyle := lipgloss.NewStyle().
		Width(f.width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("250")).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("62")).
		BorderTop(true)

	var keys keyMap
	switch state {
	case views.StateClusterList:
		keys = clusterListKeys
	case views.StateCreateForm:
		keys = createFormKeys
	case views.StateDeleteConfirm:
		keys = deleteModalKeys
	default:
		keys = clusterListKeys
	}

	helpView := f.help.ShortHelpView([]key.Binding{
		keys.Up, keys.Down, keys.Enter, keys.New, keys.Delete,
		keys.Refresh, keys.Back, keys.Help, keys.Quit,
	})

	return footerStyle.Render(helpView)
}
