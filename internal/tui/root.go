package tui

import (
	"fmt"

	"github.com/Felipalds/go-kubernetes-helper/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
)

// RootModel is the main state machine that routes between different views
type RootModel struct {
	state        views.AppState
	width        int
	height       int
	header       HeaderModel
	footer       FooterModel
	help         HelpModel
	clusterList  views.ClusterListModel
	createForm   views.CreateFormModel
	deleteModal  views.DeleteModalModel
	ready        bool
}

// NewRootModel creates a new root model with initial state
func NewRootModel() RootModel {
	return RootModel{
		state:       views.StateClusterList,
		width:       80,
		height:      24,
		header:      NewHeaderModel(),
		footer:      NewFooterModel(),
		help:        NewHelpModel(),
		clusterList: views.NewClusterListModel(),
		createForm:  views.NewCreateFormModel(),
		deleteModal: views.NewDeleteModalModel(),
		ready:       false,
	}
}

// Init initializes the root model
func (m RootModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.clusterList.Init(),
	)
}

// Update handles all messages and routes them appropriately
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update all child components with new size
		m.header.SetWidth(m.width)
		m.footer.SetWidth(m.width)

		// Calculate content height (total - header - footer)
		contentHeight := m.height - 3 - 3 // 3 lines for header, 3 for footer
		m.clusterList.SetSize(m.width, contentHeight)
		m.createForm.SetSize(m.width, contentHeight)
		m.deleteModal.SetSize(m.width, contentHeight)

		return m, nil

	case tea.KeyMsg:
		// Global keybindings
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			// Only quit from cluster list view
			if m.state == views.StateClusterList {
				return m, tea.Quit
			}

		case "?":
			// Toggle help overlay
			if m.help.IsVisible() {
				m.help.Toggle()
			} else {
				m.help.Toggle()
			}
			return m, nil

		case "esc":
			// Close help if open
			if m.help.IsVisible() {
				m.help.Toggle()
				return m, nil
			}
			// Otherwise, navigate back to cluster list
			if m.state != views.StateClusterList {
				m.state = views.StateClusterList
				return m, m.clusterList.Init()
			}
		}

		// Route to active view if help is not visible
		if !m.help.IsVisible() {
			return m.routeUpdate(msg)
		}

		// Handle help overlay
		if m.help.IsVisible() {
			m.help, cmd = m.help.Update(msg)
			return m, cmd
		}

	case views.StateChangeMsg:
		// Handle state changes from child views
		m.state = msg.NewState

		// Initialize the new view
		switch msg.NewState {
		case views.StateCreateForm:
			return m, m.createForm.Init()
		case views.StateDeleteConfirm:
			m.deleteModal.SetCluster(msg.Data.(string))
			return m, nil
		case views.StateClusterList:
			return m, m.clusterList.Init()
		}
		return m, nil

	case views.ClusterDeletedMsg:
		// Refresh cluster list after deletion
		m.state = views.StateClusterList
		return m, m.clusterList.Init()
	}

	// Route to active view
	return m.routeUpdate(msg)
}

// routeUpdate routes messages to the active view
func (m RootModel) routeUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case views.StateClusterList:
		m.clusterList, cmd = m.clusterList.Update(msg)
	case views.StateCreateForm:
		m.createForm, cmd = m.createForm.Update(msg)
	case views.StateDeleteConfirm:
		m.deleteModal, cmd = m.deleteModal.Update(msg)
	}

	return m, cmd
}

// View renders the entire application
func (m RootModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Show help overlay if visible
	if m.help.IsVisible() {
		return m.help.View()
	}

	// Build the layout
	header := m.header.View()
	footer := m.footer.ViewForState(m.state)

	// Get content from active view
	var content string
	switch m.state {
	case views.StateClusterList:
		content = m.clusterList.View()
	case views.StateCreateForm:
		content = m.createForm.View()
	case views.StateDeleteConfirm:
		// Show cluster list in background with modal overlay
		content = m.clusterList.View()
		content = m.deleteModal.ViewOver(content)
	default:
		content = fmt.Sprintf("State: %s (not implemented)", m.state)
	}

	// Combine all parts
	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}
