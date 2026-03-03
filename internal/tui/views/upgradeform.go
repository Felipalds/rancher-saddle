package views

import (
	"fmt"
	"os"
	"time"

	"github.com/Felipalds/rancher-corral/internal/config"
	"github.com/Felipalds/rancher-corral/internal/upgrade"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UpgradeFormModel is the TUI form for upgrading Rancher on an existing cluster.
type UpgradeFormModel struct {
	width       int
	height      int
	clusterName string
	cluster     *config.ClusterConfig

	// Form fields: 0=version, 1=replicas, 2=auditLogLevel (text inputs)
	inputs     []textinput.Model
	focusIndex int

	// Select field for audit log toggle
	auditLogEnabled bool
}

// NewUpgradeFormModel creates a new upgrade form.
func NewUpgradeFormModel() UpgradeFormModel {
	m := UpgradeFormModel{
		width:  80,
		height: 20,
		inputs: make([]textinput.Model, 3),
	}

	// Target Version
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "2.11.7"
	m.inputs[0].CharLimit = 20
	m.inputs[0].Width = 30

	// Replicas
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "1"
	m.inputs[1].CharLimit = 2
	m.inputs[1].Width = 10

	// Audit Log Level
	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "1"
	m.inputs[2].CharLimit = 1
	m.inputs[2].Width = 10

	return m
}

// Init returns the blink command for cursor.
func (m UpgradeFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize updates dimensions.
func (m *UpgradeFormModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetCluster configures the form for a specific cluster and pre-fills fields.
func (m *UpgradeFormModel) SetCluster(clusterName string) tea.Cmd {
	m.clusterName = clusterName
	m.focusIndex = 0

	return func() tea.Msg {
		cfg, err := config.LoadClustersConfig("config.yaml")
		if err != nil {
			return upgradeClusterLoadedMsg{err: err}
		}
		cluster, exists := cfg.GetCluster(clusterName)
		if !exists {
			return upgradeClusterLoadedMsg{err: fmt.Errorf("cluster %q not found", clusterName)}
		}
		return upgradeClusterLoadedMsg{cluster: cluster}
	}
}

// Update handles messages.
func (m UpgradeFormModel) Update(msg tea.Msg) (UpgradeFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateClusterList}
			}

		case "left", "right":
			// Toggle audit log when focused on the select row (index 2)
			if m.focusIndex == 2 {
				m.auditLogEnabled = !m.auditLogEnabled
				return m, nil
			}

		case "tab", "down":
			m.focusIndex++
			if m.focusIndex > 4 { // 0=version, 1=replicas, 2=auditLogSelect, 3=auditLogLevel, 4=submit
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 4
			}
			return m, m.updateFocus()

		case "enter":
			if m.focusIndex == 4 {
				// Submit
				return m, m.startUpgrade()
			}
			// Move to next field
			m.focusIndex++
			if m.focusIndex > 4 {
				m.focusIndex = 0
			}
			return m, m.updateFocus()
		}

	case upgradeClusterLoadedMsg:
		if msg.err != nil {
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: StateClusterList}
			}
		}
		m.cluster = msg.cluster
		m.inputs[0].SetValue(msg.cluster.Rancher.Version)
		m.inputs[1].SetValue("1")
		m.inputs[2].SetValue("1")
		m.auditLogEnabled = msg.cluster.Rancher.AuditLog
		if msg.cluster.Rancher.AuditLogLevel > 0 {
			m.inputs[2].SetValue(fmt.Sprintf("%d", msg.cluster.Rancher.AuditLogLevel))
		}
		m.inputs[0].Focus()
		return m, nil

	case upgradeFinishedMsg:
		return m, func() tea.Msg {
			return StateChangeMsg{NewState: StateClusterList}
		}
	}

	// Update focused text input
	cmd := m.updateInputs(msg)
	return m, cmd
}

// View renders the upgrade form.
func (m UpgradeFormModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(22)

	primeLabel := ""
	if m.cluster != nil && m.cluster.Rancher.Prime {
		primeLabel = " (Prime)"
	}

	title := titleStyle.Render(fmt.Sprintf("Upgrade Rancher%s: %s", primeLabel, m.clusterName))

	form := title + "\n\n"

	// Field 0: Target Version
	label := labelStyle.Render("Target Version:")
	if m.focusIndex == 0 {
		form += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("▶ ") + label + " " + m.inputs[0].View() + "\n"
	} else {
		form += "  " + label + " " + m.inputs[0].View() + "\n"
	}

	// Field 1: Replicas
	label = labelStyle.Render("Replicas:")
	if m.focusIndex == 1 {
		form += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("▶ ") + label + " " + m.inputs[1].View() + "\n"
	} else {
		form += "  " + label + " " + m.inputs[1].View() + "\n"
	}

	// Field 2: Audit Log (select)
	label = labelStyle.Render("Audit Log:")
	auditValue := "No"
	if m.auditLogEnabled {
		auditValue = "Yes"
	}
	if m.focusIndex == 2 {
		form += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("▶ ") + label + " "
		form += lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(auditValue)
		form += lipgloss.NewStyle().Faint(true).Render(" ◀ ▶") + "\n"
	} else {
		form += "  " + label + " " + auditValue + "\n"
	}

	// Field 3: Audit Log Level
	label = labelStyle.Render("Audit Log Level:")
	if m.focusIndex == 3 {
		form += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("▶ ") + label + " " + m.inputs[2].View() + "\n"
	} else {
		form += "  " + label + " " + m.inputs[2].View() + "\n"
	}

	// Submit button
	if m.focusIndex == 4 {
		form += "\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("▶ [ Upgrade ]") + "\n"
	} else {
		form += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  [ Upgrade ]") + "\n"
	}

	form += "\n" + lipgloss.NewStyle().Faint(true).Render("tab/enter: next • shift+tab/↑: prev • ◀/▶: toggle • esc: cancel")

	return lipgloss.NewStyle().Padding(2, 4).Render(form)
}

func (m *UpgradeFormModel) updateFocus() tea.Cmd {
	// Text inputs are at focus indices 0, 1, 3. Index 2 is the select, index 4 is submit.
	inputMap := map[int]int{0: 0, 1: 1, 3: 2} // focusIndex → input index

	for fi, ii := range inputMap {
		if m.focusIndex == fi {
			m.inputs[ii].Focus()
		} else {
			m.inputs[ii].Blur()
		}
	}
	// Blur all if on select(2) or submit(4)
	if m.focusIndex == 2 || m.focusIndex == 4 {
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
	}
	return nil
}

func (m *UpgradeFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	// Only update the focused text input
	inputMap := map[int]int{0: 0, 1: 1, 3: 2}
	if ii, ok := inputMap[m.focusIndex]; ok {
		var cmd tea.Cmd
		m.inputs[ii], cmd = m.inputs[ii].Update(msg)
		return cmd
	}
	return nil
}

func (m UpgradeFormModel) startUpgrade() tea.Cmd {
	clusterName := m.clusterName
	targetVersion := m.inputs[0].Value()
	replicasStr := m.inputs[1].Value()
	auditLog := m.auditLogEnabled
	auditLogLevel := 1
	fmt.Sscanf(m.inputs[2].Value(), "%d", &auditLogLevel)

	replicas := 1
	fmt.Sscanf(replicasStr, "%d", &replicas)

	return func() tea.Msg {
		// Load cluster config
		cfg, err := config.LoadClustersConfig("config.yaml")
		if err != nil {
			return upgradeFinishedMsg{err: err}
		}
		cluster, exists := cfg.GetCluster(clusterName)
		if !exists {
			return upgradeFinishedMsg{err: fmt.Errorf("cluster %q not found", clusterName)}
		}

		// Set status to upgrading
		cluster.Status = "upgrading"
		cfg.AddCluster(clusterName, cluster)
		cfg.Save("config.yaml")

		// Build hostname from DNS or IP
		hostname := ""
		if len(cluster.InstanceDNS) > 0 {
			hostname = cluster.InstanceDNS[0]
		} else if len(cluster.InstanceIPs) > 0 {
			hostname = cluster.InstanceIPs[0]
		}

		initIP := ""
		if len(cluster.InstanceIPs) > 0 {
			initIP = cluster.InstanceIPs[0]
		}

		// Run upgrade in background goroutine
		go runUpgrade(clusterName, upgrade.UpgradeConfig{
			ClusterName:       clusterName,
			Distribution:      cluster.Kubernetes.Distribution,
			InitIP:            initIP,
			SSHPrivateKeyPath: cluster.SSH.PrivateKeyPath,
			SSHUser:           cluster.SSH.User,
			Hostname:          hostname,
			RancherVersion:    targetVersion,
			BootstrapPassword: cluster.Rancher.BootstrapPassword,
			Prime:             cluster.Rancher.Prime,
			Replicas:          replicas,
			AuditLog:          auditLog,
			AuditLogLevel:     auditLogLevel,
		})

		return upgradeFinishedMsg{}
	}
}

// runUpgrade runs the actual upgrade in a background goroutine.
func runUpgrade(clusterName string, cfg upgrade.UpgradeConfig) {
	logPath := fmt.Sprintf("logs/%s-upgrade.log", clusterName)
	os.MkdirAll("logs", 0755)
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	writeLog := func(message string) {
		if logFile != nil {
			fmt.Fprintf(logFile, "[%s] %s\n", time.Now().Format("15:04:05"), message)
			logFile.Sync()
		}
	}

	writeLog(fmt.Sprintf("=== Starting Rancher upgrade for cluster: %s ===", clusterName))
	writeLog(fmt.Sprintf("Target version: %s, Prime: %v", cfg.RancherVersion, cfg.Prime))

	runner := upgrade.NewRunner(cfg)
	if err := runner.Run(logFile); err != nil {
		writeLog(fmt.Sprintf("ERROR: Upgrade failed: %v", err))
		updateClusterStatus(clusterName, "upgrade-failed")
		if logFile != nil {
			logFile.Close()
		}
		return
	}

	writeLog("Upgrade completed successfully!")

	// Update config.yaml with new version and audit log settings
	clustersConfig, err := config.LoadClustersConfig("config.yaml")
	if err == nil {
		if cluster, exists := clustersConfig.GetCluster(clusterName); exists {
			cluster.Rancher.Version = cfg.RancherVersion
			cluster.Rancher.AuditLog = cfg.AuditLog
			cluster.Rancher.AuditLogLevel = cfg.AuditLogLevel
			cluster.Status = "running"
			clustersConfig.AddCluster(clusterName, cluster)
			clustersConfig.Save("config.yaml")
		}
	}

	if logFile != nil {
		logFile.Close()
	}
}

// Message types
type upgradeClusterLoadedMsg struct {
	cluster *config.ClusterConfig
	err     error
}

type upgradeFinishedMsg struct {
	err error
}
