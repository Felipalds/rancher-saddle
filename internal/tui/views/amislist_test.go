package views

import (
	"testing"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/stretchr/testify/assert"
)

// modelWithAMIs returns an AMIsListModel pre-seeded with AMI entries.
func modelWithAMIs(entries ...config.AMIEntry) AMIsListModel {
	m := NewAMIsListModel()
	m.entries = entries
	return m
}

// ── hasPendingDelete ─────────────────────────────────────────────────────────

func TestAMIsListModel_NoPendingByDefault(t *testing.T) {
	m := NewAMIsListModel()
	assert.False(t, m.hasPendingDelete())
}

// ── delete confirmation flow ──────────────────────────────────────────────────

func TestAMIsListModel_DKeySetsConfirmPending(t *testing.T) {
	m := modelWithAMIs(
		config.AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"},
		config.AMIEntry{Distro: "RHEL 9", Region: "us-east-1", AMIID: "ami-0002"},
	)

	m, cmd := m.Update(keyPress("d"))

	assert.True(t, m.hasPendingDelete())
	assert.Equal(t, "Ubuntu 22.04 LTS", m.pendingDeleteEntry.Distro)
	assert.Equal(t, "us-east-1", m.pendingDeleteEntry.Region)
	assert.Nil(t, cmd, "no command should be issued before confirmation")
}

func TestAMIsListModel_NoEntriesIgnoresDKey(t *testing.T) {
	m := NewAMIsListModel() // no entries

	m, cmd := m.Update(keyPress("d"))

	assert.False(t, m.hasPendingDelete())
	assert.Nil(t, cmd)
}

func TestAMIsListModel_YKeyConfirmsDelete(t *testing.T) {
	m := modelWithAMIs(config.AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"})
	m.pendingDeleteEntry = m.entries[0]

	m, cmd := m.Update(keyPress("y"))

	assert.False(t, m.hasPendingDelete(), "pending entry must be cleared after confirmation")
	assert.NotNil(t, cmd, "a delete command must be issued")
}

func TestAMIsListModel_EnterKeyConfirmsDelete(t *testing.T) {
	m := modelWithAMIs(config.AMIEntry{Distro: "RHEL 9", Region: "us-west-2", AMIID: "ami-0003"})
	m.pendingDeleteEntry = m.entries[0]

	m, cmd := m.Update(keyPress("enter"))

	assert.False(t, m.hasPendingDelete())
	assert.NotNil(t, cmd)
}

func TestAMIsListModel_NKeyCancelsDelete(t *testing.T) {
	m := modelWithAMIs(config.AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"})
	m.pendingDeleteEntry = m.entries[0]

	m, cmd := m.Update(keyPress("n"))

	assert.False(t, m.hasPendingDelete(), "pending entry must be cleared on cancel")
	assert.Nil(t, cmd, "no command should be issued on cancel")
}

func TestAMIsListModel_EscKeyCancelsDelete(t *testing.T) {
	m := modelWithAMIs(config.AMIEntry{Distro: "RHEL 9", Region: "us-east-1", AMIID: "ami-0002"})
	m.pendingDeleteEntry = m.entries[0]

	m, cmd := m.Update(keyPress("esc"))

	assert.False(t, m.hasPendingDelete())
	assert.Nil(t, cmd)
}

func TestAMIsListModel_OtherKeysBlockedWhilePending(t *testing.T) {
	m := modelWithAMIs(config.AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"})
	m.pendingDeleteEntry = m.entries[0]

	// 'c' would normally open the create form; must be blocked while confirmation is pending.
	mAfter, cmd := m.Update(keyPress("c"))

	assert.True(t, mAfter.hasPendingDelete(), "pending entry must not change")
	assert.Nil(t, cmd)
}

func TestAMIsListModel_ViewShowsModalWhenPending(t *testing.T) {
	m := modelWithAMIs(config.AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"})
	m.pendingDeleteEntry = m.entries[0]

	view := m.View()

	assert.Contains(t, view, "Ubuntu 22.04 LTS")
	assert.Contains(t, view, "Delete AMI Entry")
}

func TestAMIsListModel_ViewNormalWhenNoPending(t *testing.T) {
	m := NewAMIsListModel()

	view := m.View()

	assert.NotContains(t, view, "Delete AMI Entry")
}
