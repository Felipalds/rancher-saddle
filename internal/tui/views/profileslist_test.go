package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// keyPress builds a tea.KeyMsg for a single character or named key.
func keyPress(key string) tea.Msg {
	switch key {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// modelWithProfiles returns a ProfilesListModel pre-seeded with profile names.
func modelWithProfiles(names ...string) ProfilesListModel {
	m := NewProfilesListModel()
	m.profileNames = names
	return m
}

// ── delete confirmation flow ──────────────────────────────────────────────────

func TestProfilesListModel_DKeySetsConfirmPending(t *testing.T) {
	m := modelWithProfiles("prod", "dev")

	m, cmd := m.Update(keyPress("d"))

	assert.Equal(t, "prod", m.pendingDelete, "first row should be pending delete")
	assert.Nil(t, cmd, "no command should be issued before confirmation")
}

func TestProfilesListModel_NoProfilesIgnoresDKey(t *testing.T) {
	m := NewProfilesListModel() // no profiles

	m, cmd := m.Update(keyPress("d"))

	assert.Empty(t, m.pendingDelete)
	assert.Nil(t, cmd)
}

func TestProfilesListModel_YKeyConfirmsDelete(t *testing.T) {
	m := modelWithProfiles("prod")
	m.pendingDelete = "prod"

	m, cmd := m.Update(keyPress("y"))

	assert.Empty(t, m.pendingDelete, "pendingDelete must be cleared after confirmation")
	assert.NotNil(t, cmd, "a delete command must be issued")
}

func TestProfilesListModel_EnterKeyConfirmsDelete(t *testing.T) {
	m := modelWithProfiles("prod")
	m.pendingDelete = "prod"

	m, cmd := m.Update(keyPress("enter"))

	assert.Empty(t, m.pendingDelete)
	assert.NotNil(t, cmd)
}

func TestProfilesListModel_NKeyCancelsDelete(t *testing.T) {
	m := modelWithProfiles("prod")
	m.pendingDelete = "prod"

	m, cmd := m.Update(keyPress("n"))

	assert.Empty(t, m.pendingDelete, "pendingDelete must be cleared on cancel")
	assert.Nil(t, cmd, "no command should be issued on cancel")
}

func TestProfilesListModel_EscKeyCancelsDelete(t *testing.T) {
	m := modelWithProfiles("prod")
	m.pendingDelete = "prod"

	m, cmd := m.Update(keyPress("esc"))

	assert.Empty(t, m.pendingDelete)
	assert.Nil(t, cmd)
}

func TestProfilesListModel_OtherKeysBlockedWhilePending(t *testing.T) {
	m := modelWithProfiles("prod")
	m.pendingDelete = "prod"

	// 'c' would normally open create form; must be blocked while confirmation is pending.
	mAfter, cmd := m.Update(keyPress("c"))

	assert.Equal(t, "prod", mAfter.pendingDelete, "pendingDelete must not change")
	assert.Nil(t, cmd)
}

func TestProfilesListModel_ViewShowsModalWhenPending(t *testing.T) {
	m := modelWithProfiles("prod")
	m.pendingDelete = "prod"

	view := m.View()

	assert.Contains(t, view, "prod", "modal must mention the pending profile name")
	assert.Contains(t, view, "Delete Profile")
}

func TestProfilesListModel_ViewNormalWhenNoPending(t *testing.T) {
	m := NewProfilesListModel()

	view := m.View()

	assert.NotContains(t, view, "Delete Profile")
}
