package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// CreateFormModel represents the cluster creation form
type CreateFormModel struct {
	width  int
	height int
}

// NewCreateFormModel creates a new create form
func NewCreateFormModel() CreateFormModel {
	return CreateFormModel{
		width:  80,
		height: 20,
	}
}

// Init initializes the form
func (m CreateFormModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the form dimensions
func (m *CreateFormModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages
func (m CreateFormModel) Update(msg tea.Msg) (CreateFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Go back to cluster list
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateClusterList}
			}
		}
	}

	return m, nil
}

// View renders the form
func (m CreateFormModel) View() string {
	return "Create Cluster Form (TODO: Implement multi-step wizard)\n\nPress ESC to go back"
}
