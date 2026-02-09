package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Felipalds/go-kubernetes-helper/internal/model"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type Model struct {
	focusIndex int
	inputs     []textinput.Model
	config     *model.Config
	done       bool
	quitting   bool
}

func NewModel(cfg *model.Config) Model {
	m := Model{
		inputs: make([]textinput.Model, 13),
		config: cfg,
	}

	for i := range m.inputs {
		t := textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 64 // increased limit

		switch i {
		case 0:
			t.Prompt = "AWS Access Key: "
			t.Placeholder = "AKIA..."
			t.SetValue(cfg.AWSAccessKey)
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Prompt = "AWS Secret Key: "
			t.Placeholder = "Secret..."
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
			t.SetValue(cfg.AWSSecretKey)
		case 2:
			t.Prompt = "AWS Region: "
			t.Placeholder = "us-east-1"
			t.SetValue(cfg.AWSRegion)
		case 3:
			t.Prompt = "Subnet ID: "
			t.Placeholder = "subnet-..."
			t.SetValue(cfg.SubnetID)
		case 4:
			t.Prompt = "Security Group ID: "
			t.Placeholder = "sg-..."
			t.SetValue(cfg.SecurityGroupID)
		case 5:
			t.Prompt = "SSH Key Name: "
			t.Placeholder = "my-key-pair"
			t.SetValue(cfg.SSHKeyName)
		case 6:
			t.Prompt = "Private Key Path: "
			t.Placeholder = "/home/user/.ssh/id_rsa"
			t.SetValue(cfg.SSHPrivateKeyPath)
		case 7:
			t.Prompt = "Node Prefix: "
			t.Placeholder = "rancher-node"
			t.SetValue(cfg.NodePrefix)
		case 8:
			t.Prompt = "AMI ID: "
			t.Placeholder = "ami-..."
			t.SetValue(cfg.AMI)
		case 9:
			t.Prompt = "Instance Count: "
			t.Placeholder = "1"
			t.SetValue(strconv.Itoa(cfg.InstanceCount))
			t.Validate = func(s string) error {
				_, err := strconv.Atoi(s)
				return err
			}
		case 10:
			t.Prompt = "Root Volume Size (GB): "
			t.Placeholder = "20"
			t.SetValue(strconv.Itoa(cfg.RootVolumeSize))
			t.Validate = func(s string) error {
				val, err := strconv.Atoi(s)
				if err != nil {
					return err
				}
				if val < 10 {
					return fmt.Errorf("minimum 10 GB required")
				}
				return nil
			}
		case 11:
			t.Prompt = "RKE2 Version: "
			t.Placeholder = "v1.x.x"
			t.SetValue(cfg.RKE2Version)
		case 12:
			t.Prompt = "Rancher Version: "
			t.Placeholder = "2.x.x"
			t.SetValue(cfg.RancherVersion)
		}

		m.inputs[i] = t
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		// Change focus
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			if s == "enter" && m.focusIndex == len(m.inputs) {
				m.done = true
				m.updateConfig()
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only update the focused input
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex], cmds[m.focusIndex] = m.inputs[m.focusIndex].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.done {
		return "Deploying Rancher...\n"
	}
	if m.quitting {
		return "Aborted.\n"
	}

	var b strings.Builder

	b.WriteString("Forge Config\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render("cursor mode is "))
	b.WriteString(cursorModeHelpStyle.Render(m.inputs[0].Cursor.Mode().String()))
	b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))

	return b.String()
}

func (m *Model) updateConfig() {
	m.config.AWSAccessKey = m.inputs[0].Value()
	m.config.AWSSecretKey = m.inputs[1].Value()
	m.config.AWSRegion = m.inputs[2].Value()
	m.config.SubnetID = m.inputs[3].Value()
	m.config.SecurityGroupID = m.inputs[4].Value()
	m.config.SSHKeyName = m.inputs[5].Value()
	m.config.SSHPrivateKeyPath = m.inputs[6].Value()
	m.config.NodePrefix = m.inputs[7].Value()
	m.config.AMI = m.inputs[8].Value()

	if i, err := strconv.Atoi(m.inputs[9].Value()); err == nil {
		m.config.InstanceCount = i
	}

	if i, err := strconv.Atoi(m.inputs[10].Value()); err == nil {
		m.config.RootVolumeSize = i
	}

	m.config.RKE2Version = m.inputs[11].Value()
	m.config.RancherVersion = m.inputs[12].Value()
}

func (m Model) Done() bool {
	return m.done
}
