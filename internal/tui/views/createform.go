package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/Felipalds/rancher-saddle/internal/workflow"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FieldType represents the type of form field
type FieldType int

const (
	FieldText FieldType = iota
	FieldSelect
)

// FormField represents a single form field
type FormField struct {
	fieldType   FieldType
	label       string
	input       textinput.Model
	options     []string // For select fields
	selected    int      // Selected option index
	placeholder string
	hidden      bool
}

// CreateFormModel represents the simplified cluster creation form
type CreateFormModel struct {
	width  int
	height int

	// Form fields
	fields      []FormField
	focusIndex  int
	credentials *credentials.CloudCredentials
	profiles    *config.ProfilesConfig
	amis        *config.AMIsConfig

	// Scroll state for tall forms
	scrollOffset       int
	showProfileSelect  bool
	profileSelectIndex int
}

// NewCreateFormModel creates a new simplified creation form
func NewCreateFormModel() CreateFormModel {
	m := CreateFormModel{
		width:        80,
		height:       20,
		focusIndex:   0,
		scrollOffset: 0,
	}

	m.initFields()

	return m
}

// Init initializes the form
func (m CreateFormModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.loadCredentials(),
		m.loadProfiles(),
		m.loadAMIsForForm(),
	)
}

// SetSize updates dimensions
func (m *CreateFormModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *CreateFormModel) initFields() {
	m.fields = []FormField{
		// Provider selection
		{
			fieldType: FieldSelect,
			label:     "Provider",
			options:   []string{"AWS", "Azure (Coming soon)", "GCP (Coming soon)"},
			selected:  0,
		},
		// Credentials selection (will be populated when credentials load)
		{
			fieldType:   FieldSelect,
			label:       "Credentials",
			options:     []string{"Loading..."},
			selected:    0,
		},
		// Kubernetes Distribution
		{
			fieldType: FieldSelect,
			label:     "Kubernetes Distribution",
			options:   []string{"RKE2", "K3s"},
			selected:  0,
		},
		// Cluster Name
		{
			fieldType:   FieldText,
			label:       "Cluster Name",
			input:       m.createTextInput("my-cluster", 50, 40),
			placeholder: "my-cluster",
		},
		// Node Prefix
		{
			fieldType:   FieldText,
			label:       "Node Prefix",
			input:       m.createTextInput("k8s-node", 30, 40),
			placeholder: "k8s-node",
		},
		// Region
		{
			fieldType:   FieldText,
			label:       "Region",
			input:       m.createTextInput("us-east-1", 20, 40),
			placeholder: "us-east-1",
		},
		// Subnet ID
		{
			fieldType:   FieldText,
			label:       "Subnet ID",
			input:       m.createTextInput("subnet-xxxxx", 50, 50),
			placeholder: "subnet-xxxxx",
		},
		// Security Group ID
		{
			fieldType:   FieldText,
			label:       "Security Group ID",
			input:       m.createTextInput("sg-xxxxx", 50, 50),
			placeholder: "sg-xxxxx",
		},
		// OS Image (distro picker — resolves to an AMI ID at submit time)
		{
			fieldType: FieldSelect,
			label:     "OS Image",
			options:   []string{"Ubuntu 22.04 LTS", "RHEL 9", "SLES 15 SP5", "Custom"},
			selected:  0,
		},
		// Custom AMI ID (only visible when OS Image == "Custom")
		{
			fieldType:   FieldText,
			label:       "Custom AMI ID",
			input:       m.createTextInput("ami-xxxxx", 50, 50),
			placeholder: "ami-xxxxx",
			hidden:      true,
		},
		// Instance Type
		{
			fieldType:   FieldText,
			label:       "Instance Type",
			input:       m.createTextInput("t3.xlarge", 20, 40),
			placeholder: "t3.xlarge",
		},
		// Instance Count
		{
			fieldType:   FieldText,
			label:       "Instance Count",
			input:       m.createTextInput("3", 2, 10),
			placeholder: "3",
		},
		// SSH Key Name
		{
			fieldType:   FieldText,
			label:       "SSH Key Name",
			input:       m.createTextInput("my-key", 50, 40),
			placeholder: "my-key",
		},
		// SSH Private Key Path
		{
			fieldType:   FieldText,
			label:       "SSH Private Key Path",
			input:       m.createTextInput("~/.ssh/my-key.pem", 100, 60),
			placeholder: "~/.ssh/my-key.pem",
		},
		// SSH User
		{
			fieldType:   FieldText,
			label:       "SSH User",
			input:       m.createTextInput("ubuntu", 20, 30),
			placeholder: "ubuntu",
		},
		// K8s Version
		{
			fieldType:   FieldText,
			label:       "K8s Version",
			input:       m.createTextInput("v1.33.7+rke2r1", 30, 40),
			placeholder: "v1.33.7+rke2r1",
		},
		// Deploy Rancher
		{
			fieldType: FieldSelect,
			label:     "Deploy Rancher",
			options:   []string{"No", "Yes"},
			selected:  0,
		},
		// Rancher Prime
		{
			fieldType: FieldSelect,
			label:     "Rancher Prime",
			options:   []string{"No", "Yes"},
			selected:  0,
		},
		// Rancher Version
		{
			fieldType:   FieldText,
			label:       "Rancher Version",
			input:       m.createTextInput("2.11.7", 20, 30),
			placeholder: "2.11.7",
		},
		// Bootstrap Password
		{
			fieldType:   FieldText,
			label:       "Bootstrap Password",
			input:       m.createTextInput("admin", 50, 40),
			placeholder: "admin",
		},
		// Image Tag (hotfix)
		{
			fieldType:   FieldText,
			label:       "Image Tag (hotfix)",
			input:       m.createTextInput("", 60, 50),
			placeholder: "e.g. v0.0.0-hotfix-abc123.1",
		},
		// Debug Mode
		{
			fieldType: FieldSelect,
			label:     "Debug Mode",
			options:   []string{"No", "Yes"},
			selected:  0,
		},
	}

	// Focus first field
	if len(m.fields) > 0 && m.fields[0].fieldType == FieldText {
		m.fields[0].input.Focus()
	}
}

func (m *CreateFormModel) createTextInput(placeholder string, charLimit int, width int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = charLimit
	input.Width = width
	return input
}

// Update handles messages
func (m CreateFormModel) Update(msg tea.Msg) (CreateFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle special keys only - let normal typing pass through
		key := msg.String()

		// Only intercept specific navigation/control keys
		switch key {
		case "esc":
			// Close profile selector if open
			if m.showProfileSelect {
				m.showProfileSelect = false
				return m, nil
			}
			// Cancel and return to cluster list
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateClusterList}
			}

		case "ctrl+p":
			// Toggle profile selector (changed from 'p' to ctrl+p)
			if m.profiles != nil && m.profiles.HasProfiles() {
				m.showProfileSelect = !m.showProfileSelect
				m.profileSelectIndex = 0
			}
			return m, nil

		case "enter":
			// Handle profile selection
			if m.showProfileSelect {
				profileNames := m.profiles.ListProfiles()
				if m.profileSelectIndex < len(profileNames) {
					m.loadProfileIntoForm(profileNames[m.profileSelectIndex])
					m.showProfileSelect = false
				}
				return m, nil
			}
			// Submit only when the Apply button is focused
			if m.focusIndex == len(m.fields) {
				return m.handleSubmit()
			}
			// Otherwise, move to next field
			m.focusIndex = m.nextVisibleFieldOrApply(m.focusIndex)
			return m, m.updateFocus()

		case "tab", "down":
			// Handle profile selector navigation
			if m.showProfileSelect {
				profileNames := m.profiles.ListProfiles()
				m.profileSelectIndex++
				if m.profileSelectIndex >= len(profileNames) {
					m.profileSelectIndex = 0
				}
				return m, nil
			}
			// Move to next visible field (including Apply button)
			m.focusIndex = m.nextVisibleFieldOrApply(m.focusIndex)
			return m, m.updateFocus()

		case "shift+tab", "up":
			// Handle profile selector navigation
			if m.showProfileSelect {
				profileNames := m.profiles.ListProfiles()
				m.profileSelectIndex--
				if m.profileSelectIndex < 0 {
					m.profileSelectIndex = len(profileNames) - 1
				}
				return m, nil
			}
			// Move to previous visible field (including Apply button)
			m.focusIndex = m.prevVisibleFieldOrApply(m.focusIndex)
			return m, m.updateFocus()

		case "left":
			// For select fields, move to previous option
			if m.focusIndex < len(m.fields) && m.fields[m.focusIndex].fieldType == FieldSelect {
				m.fields[m.focusIndex].selected--
				if m.fields[m.focusIndex].selected < 0 {
					m.fields[m.focusIndex].selected = len(m.fields[m.focusIndex].options) - 1
				}
				if m.focusIndex == 8 {
					m.syncCustomAMIVisibility()
				}
				return m, nil
			}
			// Let text inputs handle left arrow

		case "right":
			// For select fields, move to next option
			if m.focusIndex < len(m.fields) && m.fields[m.focusIndex].fieldType == FieldSelect {
				m.fields[m.focusIndex].selected++
				if m.fields[m.focusIndex].selected >= len(m.fields[m.focusIndex].options) {
					m.fields[m.focusIndex].selected = 0
				}
				if m.focusIndex == 8 {
					m.syncCustomAMIVisibility()
				}
				return m, nil
			}
			// Let text inputs handle right arrow
		}

	case credentialsLoadedForWizardMsg:
		m.credentials = msg.credentials
		// Update credentials field options
		if m.credentials != nil && m.credentials.HasAWSCredentials() {
			credNames := m.credentials.ListAWSCredentials()
			if len(credNames) > 0 {
				m.fields[1].options = credNames
				m.fields[1].selected = 0
			}
		}
		return m, nil

	case clusterCreatedMsg:
		// Cluster created successfully, return to cluster list
		return m, func() tea.Msg {
			return StateChangeMsg{NewState: StateClusterList}
		}

	case clusterCreationErrorMsg:
		// TODO: Show error message to user
		fmt.Printf("Error creating cluster: %v\n", msg.err)
		return m, nil

	case profilesLoadedForFormMsg:
		m.profiles = msg.profiles
		return m, nil

	case amisLoadedForFormMsg:
		m.amis = msg.amis
		if m.amis != nil {
			distros := m.amis.ListDistros()
			options := append(distros, "Custom")
			m.fields[8].options = options
			// Keep selection in bounds
			if m.fields[8].selected >= len(options) {
				m.fields[8].selected = 0
			}
			m.syncCustomAMIVisibility()
		}
		return m, nil
	}

	// Update focused text input
	if m.focusIndex < len(m.fields) && m.fields[m.focusIndex].fieldType == FieldText {
		var cmd tea.Cmd
		m.fields[m.focusIndex].input, cmd = m.fields[m.focusIndex].input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CreateFormModel) handleSubmit() (CreateFormModel, tea.Cmd) {
	// Collect form data (indices updated for new fields)
	provider := m.fields[0].options[m.fields[0].selected]
	credential := ""
	if len(m.fields[1].options) > 0 {
		credential = m.fields[1].options[m.fields[1].selected]
	}
	k8sDistro := m.fields[2].options[m.fields[2].selected]
	clusterName := m.fields[3].input.Value()
	nodePrefix := m.fields[4].input.Value()
	region := m.fields[5].input.Value()
	subnet := m.fields[6].input.Value()
	securityGroup := m.fields[7].input.Value()
	osImageOption := m.fields[8].options[m.fields[8].selected]
	var ami string
	if osImageOption == "Custom" {
		ami = m.fields[9].input.Value()
	} else if m.amis != nil {
		resolvedRegion := region
		if resolvedRegion == "" {
			resolvedRegion = "us-east-1"
		}
		ami, _ = m.amis.GetAMI(osImageOption, resolvedRegion)
	}
	instanceType := m.fields[10].input.Value()
	instanceCountStr := m.fields[11].input.Value()
	sshKeyName := m.fields[12].input.Value()
	sshPrivateKeyPath := m.fields[13].input.Value()
	sshUser := m.fields[14].input.Value()
	k8sVersion := m.fields[15].input.Value()
	deployRancher := m.fields[16].options[m.fields[16].selected] == "Yes"
	rancherPrime := m.fields[17].options[m.fields[17].selected] == "Yes"
	rancherVersion := m.fields[18].input.Value()
	bootstrapPassword := m.fields[19].input.Value()
	imageTag := m.fields[20].input.Value()
	debugMode := m.fields[21].options[m.fields[21].selected] == "Yes"

	// Validate required fields
	if clusterName == "" {
		// TODO: Show error message
		return m, nil
	}
	if subnet == "" {
		// TODO: Show error message
		return m, nil
	}
	if securityGroup == "" {
		// TODO: Show error message
		return m, nil
	}
	if ami == "" {
		// TODO: Show error message
		return m, nil
	}
	if sshKeyName == "" {
		// TODO: Show error message
		return m, nil
	}
	if sshPrivateKeyPath == "" {
		// TODO: Show error message
		return m, nil
	}

	// Set defaults
	if nodePrefix == "" {
		nodePrefix = "k8s-node"
	}
	if region == "" {
		region = "us-east-1"
	}
	if instanceType == "" {
		instanceType = "t3.xlarge"
	}
	if sshUser == "" {
		sshUser = "ubuntu"
	}
	if k8sVersion == "" {
		if k8sDistro == "RKE2" {
			k8sVersion = "v1.33.7+rke2r1"
		} else {
			k8sVersion = "v1.33.7+k3s1"
		}
	}

	if rancherVersion == "" {
		rancherVersion = "2.11.7"
	}
	if bootstrapPassword == "" {
		bootstrapPassword = "admin"
	}

	instanceCount := 3
	if instanceCountStr != "" {
		fmt.Sscanf(instanceCountStr, "%d", &instanceCount)
	}

	return m, m.createCluster(clusterName, provider, credential, k8sDistro, k8sVersion,
		region, subnet, securityGroup, ami, instanceType, nodePrefix, instanceCount,
		sshKeyName, sshPrivateKeyPath, sshUser, deployRancher, rancherPrime, rancherVersion, bootstrapPassword,
		imageTag, debugMode)
}

func (m CreateFormModel) createCluster(name, provider, credential, distro, version,
	region, subnet, securityGroup, ami, instanceType, nodePrefix string, instanceCount int,
	sshKeyName, sshPrivateKeyPath, sshUser string, deployRancher, rancherPrime bool, rancherVersion, bootstrapPassword string,
	imageTag string, debugMode bool) tea.Cmd {
	return func() tea.Msg {
		// Load clusters config
		cfg, err := config.LoadClustersConfig("config.yaml")
		if err != nil {
			return clusterCreationErrorMsg{err: err}
		}

		// Load credentials to get credential details
		creds, err := credentials.LoadCredentials("cloud-credentials.yaml")
		if err != nil {
			return clusterCreationErrorMsg{err: fmt.Errorf("failed to load credentials: %w", err)}
		}

		awsCred, err := creds.GetAWSCredential(credential)
		if err != nil {
			return clusterCreationErrorMsg{err: fmt.Errorf("credential not found: %w", err)}
		}

		// Create cluster configuration
		cluster := &config.ClusterConfig{
			Provider: config.ProviderSection{
				Type: strings.ToLower(provider),
				Config: map[string]interface{}{
					"region":            region,
					"instance_type":     instanceType,
					"subnet_id":         subnet,
					"security_group_id": securityGroup,
					"ami":               ami,
					"access_key":        awsCred.AccessKey,
					"secret_key":        awsCred.SecretKey,
				},
			},
			Kubernetes: config.KubernetesSection{
				Distribution: strings.ToLower(distro),
				Config: map[string]interface{}{
					"version": version,
				},
			},
			SSH: config.SSHSection{
				KeyName:        sshKeyName,
				PrivateKeyPath: sshPrivateKeyPath,
				User:           sshUser,
			},
			Cluster: config.ClusterSection{
				NodePrefix:    nodePrefix,
				InstanceCount: instanceCount,
			},
			Rancher: config.RancherSection{
				Deploy:            deployRancher,
				Version:           rancherVersion,
				Prime:             rancherPrime,
				BootstrapPassword: bootstrapPassword,
				ImageTag:          imageTag,
				Debug:             debugMode,
			},
			Status: "creating",
		}

		// Add cluster to config
		cfg.AddCluster(name, cluster)

		// Save config
		if err := cfg.Save("config.yaml"); err != nil {
			return clusterCreationErrorMsg{err: fmt.Errorf("failed to save config: %w", err)}
		}

		// Trigger cluster deployment in background
		go deployCluster(name, cluster)

		return clusterCreatedMsg{name: name}
	}
}

func (m *CreateFormModel) updateFocus() tea.Cmd {
	for i := range m.fields {
		if m.fields[i].fieldType == FieldText {
			if i == m.focusIndex && !m.fields[i].hidden {
				m.fields[i].input.Focus()
			} else {
				m.fields[i].input.Blur()
			}
		}
	}
	// Blur all text inputs when on the Apply button
	if m.focusIndex == len(m.fields) {
		for i := range m.fields {
			if m.fields[i].fieldType == FieldText {
				m.fields[i].input.Blur()
			}
		}
	}
	return nil
}

// syncCustomAMIVisibility shows/hides field[9] based on the OS Image selection.
// "Custom" is always the last option in field[8].
func (m *CreateFormModel) syncCustomAMIVisibility() {
	customIdx := len(m.fields[8].options) - 1
	isCustom := m.fields[8].selected == customIdx
	m.fields[9].hidden = !isCustom
	if m.fields[9].hidden {
		m.fields[9].input.Blur()
	}
}

// nextVisibleField returns the next non-hidden field index after `from`,
// wrapping around to 0 when the end is reached.
func (m *CreateFormModel) nextVisibleField(from int) int {
	n := len(m.fields)
	for step := 1; step <= n; step++ {
		idx := (from + step) % n
		if !m.fields[idx].hidden {
			return idx
		}
	}
	return from
}

// prevVisibleField returns the previous non-hidden field index before `from`,
// wrapping around to the last field when the start is reached.
func (m *CreateFormModel) prevVisibleField(from int) int {
	n := len(m.fields)
	for step := 1; step <= n; step++ {
		idx := (from - step + n) % n
		if !m.fields[idx].hidden {
			return idx
		}
	}
	return from
}

// nextVisibleFieldOrApply advances to the next visible field, treating
// len(m.fields) as the Apply button position at the end.
func (m *CreateFormModel) nextVisibleFieldOrApply(from int) int {
	// If we're on the Apply button, wrap to the first visible field
	if from == len(m.fields) {
		return m.nextVisibleField(len(m.fields) - 1)
	}
	// Try to find a visible field after `from`
	n := len(m.fields)
	for step := 1; step <= n; step++ {
		idx := from + step
		if idx >= n {
			// Reached the end — land on the Apply button
			return n
		}
		if !m.fields[idx].hidden {
			return idx
		}
	}
	return n
}

// prevVisibleFieldOrApply goes back to the previous visible field, treating
// len(m.fields) as the Apply button position at the end.
func (m *CreateFormModel) prevVisibleFieldOrApply(from int) int {
	// If we're on the Apply button, go to the last visible field
	if from == len(m.fields) {
		for i := len(m.fields) - 1; i >= 0; i-- {
			if !m.fields[i].hidden {
				return i
			}
		}
		return 0
	}
	// If we're on the first visible field, wrap to Apply button
	first := m.nextVisibleField(len(m.fields) - 1)
	if from == first {
		return len(m.fields)
	}
	return m.prevVisibleField(from)
}

func (m CreateFormModel) loadCredentials() tea.Cmd {
	return func() tea.Msg {
		creds, _ := credentials.LoadCredentials("cloud-credentials.yaml")
		return credentialsLoadedForWizardMsg{credentials: creds}
	}
}

func (m CreateFormModel) loadProfiles() tea.Cmd {
	return func() tea.Msg {
		profiles, _ := config.LoadProfiles("profiles.yaml")
		return profilesLoadedForFormMsg{profiles: profiles}
	}
}

func (m CreateFormModel) loadAMIsForForm() tea.Cmd {
	return func() tea.Msg {
		amis, _ := config.LoadAMIs("amis.yaml")
		return amisLoadedForFormMsg{amis: amis}
	}
}

func (m *CreateFormModel) loadProfileIntoForm(profileName string) {
	profile, err := m.profiles.GetProfile(profileName)
	if err != nil {
		return
	}

	// Load profile values into form fields
	m.fields[5].input.SetValue(profile.Region)
	m.fields[6].input.SetValue(profile.SubnetID)
	m.fields[7].input.SetValue(profile.SecurityGroupID)

	// Reverse-lookup the AMI to see if it matches a known distro
	customIdx := len(m.fields[8].options) - 1 // "Custom" is always last
	if m.amis != nil {
		if distro, found := m.amis.FindDistro(profile.AMI, profile.Region); found {
			for i, opt := range m.fields[8].options {
				if opt == distro {
					m.fields[8].selected = i
					break
				}
			}
			m.fields[9].input.SetValue("")
		} else {
			m.fields[8].selected = customIdx
			m.fields[9].input.SetValue(profile.AMI)
		}
	} else {
		m.fields[8].selected = customIdx
		m.fields[9].input.SetValue(profile.AMI)
	}
	m.syncCustomAMIVisibility()

	m.fields[10].input.SetValue(profile.InstanceType)
	m.fields[12].input.SetValue(profile.SSHKeyName)
	m.fields[13].input.SetValue(profile.SSHPrivateKeyPath)
	m.fields[14].input.SetValue(profile.SSHUser)
}

// View renders the form
func (m CreateFormModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(22)

	title := titleStyle.Render("Create New Cluster")
	content := title + "\n\n"

	// Render all fields
	for i, field := range m.fields {
		if field.hidden {
			continue
		}
		focused := i == m.focusIndex
		label := labelStyle.Render(field.label + ":")

		var fieldView string

		if field.fieldType == FieldSelect {
			// Render select field
			selectedValue := ""
			if len(field.options) > 0 && field.selected >= 0 && field.selected < len(field.options) {
				selectedValue = field.options[field.selected]
			}

			if focused {
				fieldView = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("▶ ") + label + " "
				fieldView += lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(selectedValue)
				fieldView += lipgloss.NewStyle().Faint(true).Render(" ◀ ▶")
			} else {
				fieldView = "  " + label + " " + selectedValue
			}
		} else {
			// Render text input field
			if focused {
				fieldView = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("▶ ") + label + " " + field.input.View()
			} else {
				fieldView = "  " + label + " " + field.input.View()
			}
		}

		content += fieldView + "\n"
	}

	// Apply button
	if m.focusIndex == len(m.fields) {
		content += "\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("▶ [ Apply ]") + "\n"
	} else {
		content += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  [ Apply ]") + "\n"
	}

	help := lipgloss.NewStyle().
		Faint(true).
		Render("\nenter: next/apply • ctrl+p: load profile • esc: cancel • tab/↓: next • shift+tab/↑: prev • ◀/▶: select")

	view := lipgloss.NewStyle().
		Padding(2, 4).
		Render(content + help)

	// Show profile selector overlay if active
	if m.showProfileSelect && m.profiles != nil {
		view = m.renderProfileSelector(view)
	}

	return view
}

func (m CreateFormModel) renderProfileSelector(baseView string) string {
	profileNames := m.profiles.ListProfiles()
	if len(profileNames) == 0 {
		return baseView
	}

	selectorStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Background(lipgloss.Color("235"))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	selector := titleStyle.Render("Load Profile") + "\n\n"

	for i, name := range profileNames {
		if i == m.profileSelectIndex {
			selector += lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Render("▶ " + name + "\n")
		} else {
			selector += "  " + name + "\n"
		}
	}

	selector += "\n" + lipgloss.NewStyle().Faint(true).Render("enter: load • esc: cancel")

	selectorBox := selectorStyle.Render(selector)

	// Overlay the selector in the center
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		selectorBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// deployCluster runs the cluster deployment in background using the actual workflow
func deployCluster(name string, clusterConfig *config.ClusterConfig) {
	// Create log file for this deployment
	logPath := fmt.Sprintf("logs/%s.log", name)
	os.MkdirAll("logs", 0755)

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		updateClusterStatus(name, "failed")
		return
	}
	defer logFile.Close()

	// Helper to write logs
	writeLog := func(message string) {
		timestamp := getTimestamp()
		logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)
		logFile.WriteString(logLine)
		logFile.Sync() // Flush to disk immediately
	}

	writeLog(fmt.Sprintf("=== Starting deployment for cluster: %s ===", name))
	writeLog("Converting cluster configuration...")

	// Convert ClusterConfig to config.Config for the workflow
	cfg := clusterConfig.ToModernConfig()
	cfg.ClusterName = name

	writeLog(fmt.Sprintf("Provider: %s", cfg.Provider))
	writeLog(fmt.Sprintf("Orchestrator: %s", cfg.Orchestrator))
	writeLog(fmt.Sprintf("Instance Count: %d", cfg.InstanceCount))

	// Create build directory for this cluster
	buildDir := filepath.Join("clusters", name)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		writeLog(fmt.Sprintf("ERROR: Failed to create build directory: %v", err))
		updateClusterStatus(name, "failed")
		return
	}

	writeLog(fmt.Sprintf("Build directory: %s", buildDir))

	// Update cluster config with build directory
	clusterConfig.BuildDir = buildDir
	clustersConfig, _ := config.LoadClustersConfig("config.yaml")
	clustersConfig.AddCluster(name, clusterConfig)
	clustersConfig.Save("config.yaml")

	// Create the workflow runner
	writeLog("Creating workflow runner...")
	runner, err := workflow.NewModularRunner(cfg, core.GlobalRegistry)
	if err != nil {
		writeLog(fmt.Sprintf("ERROR: Failed to create runner: %v", err))
		updateClusterStatus(name, "failed")
		return
	}

	// Redirect runner output to log file
	writeLog("Starting infrastructure deployment...")

	// Run the deployment with output capture
	if err := runDeploymentWithLogging(runner, buildDir, logFile, writeLog); err != nil {
		writeLog(fmt.Sprintf("ERROR: Deployment failed: %v", err))
		updateClusterStatus(name, "failed")
		return
	}

	writeLog(fmt.Sprintf("✓ Cluster %s deployed successfully!", name))

	// Get outputs and update cluster config
	// TODO: Pass proper context instead of nil
	if outputs, err := runner.Provider.GetOutputs(nil, buildDir); err == nil {
		clusterConfig.InstanceIPs = outputs.InstanceIPs
		clusterConfig.InstanceDNS = outputs.InstanceDNSNames
		if len(outputs.InstanceDNSNames) > 0 {
			clusterConfig.RancherURL = fmt.Sprintf("https://%s", outputs.InstanceDNSNames[0])
		}
		clustersConfig.AddCluster(name, clusterConfig)
		clustersConfig.Save("config.yaml")

		writeLog(fmt.Sprintf("Instance IPs: %v", outputs.InstanceIPs))
		writeLog(fmt.Sprintf("Instance DNS: %v", outputs.InstanceDNSNames))
	}

	// Update cluster status to running
	updateClusterStatus(name, "running")
}

// runDeploymentWithLogging runs the deployment and captures all output
func runDeploymentWithLogging(runner *workflow.ModularRunner, buildDir string, logFile *os.File, writeLog func(string)) error {
	// Create a multi-writer to write to both log file and capture output
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Channel to signal completion
	done := make(chan error, 1)

	// Goroutine to copy output to log file
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				logFile.Write(buf[:n])
				logFile.Sync()
			}
			if err != nil {
				break
			}
		}
	}()

	// Run the deployment
	go func() {
		done <- runner.RunWithBuildDir(buildDir)
	}()

	// Wait for completion
	err := <-done

	// Restore stdout/stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return err
}

// getTimestamp returns a formatted timestamp
func getTimestamp() string {
	return time.Now().Format("15:04:05")
}

// updateClusterStatus updates the status of a cluster in config.yaml
func updateClusterStatus(name, status string) {
	cfg, err := config.LoadClustersConfig("config.yaml")
	if err != nil {
		return
	}

	cluster, exists := cfg.GetCluster(name)
	if !exists {
		return
	}

	cluster.Status = status
	cfg.AddCluster(name, cluster)
	cfg.Save("config.yaml")
}

// Message types
type credentialsLoadedForWizardMsg struct {
	credentials *credentials.CloudCredentials
}

type clusterCreatedMsg struct {
	name string
}

type clusterCreationErrorMsg struct {
	err error
}

type profilesLoadedForFormMsg struct {
	profiles *config.ProfilesConfig
}

type amisLoadedForFormMsg struct {
	amis *config.AMIsConfig
}
