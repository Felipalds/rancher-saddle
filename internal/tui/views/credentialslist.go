package views

import (
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CredentialsListModel displays all cloud credentials
type CredentialsListModel struct {
	table         table.Model
	width         int
	height        int
	credentials   *credentials.CloudCredentials
	credNames     []string
	pendingDelete string // name of credential awaiting delete confirmation
}

// NewCredentialsListModel creates a new credentials list view
func NewCredentialsListModel() CredentialsListModel {
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Provider", Width: 12},
		{Title: "Region", Width: 15},
		{Title: "Access Key", Width: 20},
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

	return CredentialsListModel{
		table:  t,
		width:  80,
		height: 20,
	}
}

// Init initializes the credentials list
func (m CredentialsListModel) Init() tea.Cmd {
	return m.loadCredentials()
}

// SetSize updates the table dimensions
func (m *CredentialsListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(height - 4)
	m.table.SetWidth(width - 4)
}

// Update handles messages
func (m CredentialsListModel) Update(msg tea.Msg) (CredentialsListModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If a delete is pending, only handle confirmation keys.
		if m.pendingDelete != "" {
			switch msg.String() {
			case "y", "enter":
				name := m.pendingDelete
				m.pendingDelete = ""
				return m, m.deleteCredential(name)
			case "n", "esc":
				m.pendingDelete = ""
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "n", "c":
			return m, func() tea.Msg {
				return StateChangeMsg{
					NewState: StateCredentialsForm,
					Data:     nil,
				}
			}

		case "d":
			if len(m.credNames) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.credNames) {
					m.pendingDelete = m.credNames[selectedRow]
					return m, nil
				}
			}

		case "enter":
			if len(m.credNames) > 0 {
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.credNames) {
					credName := m.credNames[selectedRow]
					return m, func() tea.Msg {
						return StateChangeMsg{
							NewState: StateCredentialsForm,
							Data:     credName,
						}
					}
				}
			}

		case "esc":
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateClusterList}
			}
		}

	case credentialsLoadedMsg:
		m.credentials = msg.credentials
		m.credNames = msg.names
		m.updateTable()
		return m, nil

	case credentialDeletedMsg:
		return m, m.loadCredentials()
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the credentials list
func (m CredentialsListModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	title := titleStyle.Render("Cloud Provider Credentials")
	tableView := title + "\n" + baseStyle.Render(m.table.View())

	if m.credentials == nil || len(m.credNames) == 0 {
		tableView = m.emptyState()
	}

	if m.pendingDelete != "" {
		return m.deleteConfirmView(tableView)
	}

	return tableView
}

// deleteConfirmView renders the delete confirmation modal over the table.
func (m CredentialsListModel) deleteConfirmView(behind string) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(50).
		Background(lipgloss.Color("235"))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Render("⚠ Delete Credential")

	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("Are you sure you want to delete:\n\n  " + m.pendingDelete + "\n\nThis action cannot be undone.")

	actions := lipgloss.NewStyle().
		Faint(true).
		Render("\n[y] Confirm  [n] Cancel")

	modal := modalStyle.Render(title + "\n\n" + message + actions)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
	)
}

// emptyState shows a message when no credentials exist
func (m CredentialsListModel) emptyState() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return emptyStyle.Render("No credentials configured.\n\nPress 'n' to add AWS credentials.")
}

// updateTable refreshes the table rows with current credentials
func (m *CredentialsListModel) updateTable() {
	rows := []table.Row{}

	for _, name := range m.credNames {
		cred, err := m.credentials.GetAWSCredential(name)
		if err != nil {
			continue
		}

		rows = append(rows, table.Row{
			cred.Name,
			"AWS",
			cred.DefaultRegion,
			credentials.MaskKey(cred.AccessKey),
		})
	}

	m.table.SetRows(rows)
}

// loadCredentials loads credentials from file
func (m CredentialsListModel) loadCredentials() tea.Cmd {
	return func() tea.Msg {
		creds, err := credentials.LoadCredentials("cloud-credentials.yaml")
		if err != nil {
			return credentialsLoadedMsg{
				credentials: &credentials.CloudCredentials{AWS: []credentials.AWSCredential{}},
				names:       []string{},
			}
		}

		names := creds.ListAWSCredentials()
		return credentialsLoadedMsg{
			credentials: creds,
			names:       names,
		}
	}
}

// deleteCredential deletes a credential
func (m CredentialsListModel) deleteCredential(name string) tea.Cmd {
	return func() tea.Msg {
		creds, err := credentials.LoadCredentials("cloud-credentials.yaml")
		if err != nil {
			return credentialDeletedMsg{err: err}
		}

		if err := creds.DeleteAWSCredential(name); err != nil {
			return credentialDeletedMsg{err: err}
		}

		if err := creds.Save("cloud-credentials.yaml"); err != nil {
			return credentialDeletedMsg{err: err}
		}

		return credentialDeletedMsg{name: name}
	}
}

// Message types
type credentialsLoadedMsg struct {
	credentials *credentials.CloudCredentials
	names       []string
}

type credentialDeletedMsg struct {
	name string
	err  error
}
