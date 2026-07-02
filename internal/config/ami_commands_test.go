package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func amiPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "amis.yaml")
}

func seedAMIs(t *testing.T, path string, entries ...AMIEntry) {
	t.Helper()
	cfg := &AMIsConfig{AMIs: entries}
	require.NoError(t, cfg.Save(path))
}

// ── ListAMIsCLI ──────────────────────────────────────────────────────────────

func TestListAMIsCLI_Empty(t *testing.T) {
	path := amiPath(t)
	// Write an empty AMIs config so LoadAMIs does not seed defaults.
	require.NoError(t, (&AMIsConfig{AMIs: []AMIEntry{}}).Save(path))

	err := ListAMIsCLI(path)
	assert.NoError(t, err)
}

func TestListAMIsCLI_ShowsEntries(t *testing.T) {
	path := amiPath(t)
	seedAMIs(t, path,
		AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"},
		AMIEntry{Distro: "RHEL 9", Region: "us-east-1", AMIID: "ami-0002"},
	)
	err := ListAMIsCLI(path)
	assert.NoError(t, err)
}

// ── AddAMICLI ────────────────────────────────────────────────────────────────

func TestAddAMICLI_CreatesEntry(t *testing.T) {
	path := amiPath(t)

	err := AddAMICLI(path, AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0abc"})
	require.NoError(t, err)

	amis, err := LoadAMIs(path)
	require.NoError(t, err)
	id, ok := amis.GetAMI("Ubuntu 22.04 LTS", "us-east-1")
	require.True(t, ok)
	assert.Equal(t, "ami-0abc", id)
}

func TestAddAMICLI_ReplacesExisting(t *testing.T) {
	path := amiPath(t)
	seedAMIs(t, path, AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-old"})

	err := AddAMICLI(path, AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-new"})
	require.NoError(t, err)

	amis, err := LoadAMIs(path)
	require.NoError(t, err)
	id, _ := amis.GetAMI("Ubuntu 22.04 LTS", "us-east-1")
	assert.Equal(t, "ami-new", id)
}

func TestAddAMICLI_ValidationErrors(t *testing.T) {
	path := amiPath(t)

	tests := []struct {
		name  string
		entry AMIEntry
	}{
		{"missing distro", AMIEntry{Region: "us-east-1", AMIID: "ami-0abc"}},
		{"missing region", AMIEntry{Distro: "Ubuntu 22.04 LTS", AMIID: "ami-0abc"}},
		{"missing ami id", AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, AddAMICLI(path, tt.entry))
		})
	}
}

// ── DeleteAMICLI ─────────────────────────────────────────────────────────────

func TestDeleteAMICLI_RemovesEntry(t *testing.T) {
	path := amiPath(t)
	seedAMIs(t, path,
		AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0001"},
		AMIEntry{Distro: "RHEL 9", Region: "us-east-1", AMIID: "ami-0002"},
	)

	err := DeleteAMICLI(path, "Ubuntu 22.04 LTS", "us-east-1", true)
	require.NoError(t, err)

	amis, err := LoadAMIs(path)
	require.NoError(t, err)
	_, ok := amis.GetAMI("Ubuntu 22.04 LTS", "us-east-1")
	assert.False(t, ok)
	// Other entry must still be there.
	_, ok = amis.GetAMI("RHEL 9", "us-east-1")
	assert.True(t, ok)
}

func TestDeleteAMICLI_NotFound(t *testing.T) {
	path := amiPath(t)
	require.NoError(t, (&AMIsConfig{AMIs: []AMIEntry{}}).Save(path))

	err := DeleteAMICLI(path, "Ghost Distro", "us-east-1", true)
	assert.Error(t, err)
}

func TestDeleteAMICLI_MissingFlags(t *testing.T) {
	path := amiPath(t)
	assert.Error(t, DeleteAMICLI(path, "", "us-east-1", true))
	assert.Error(t, DeleteAMICLI(path, "Ubuntu 22.04 LTS", "", true))
}

// ── EditAMICLI ───────────────────────────────────────────────────────────────

func TestEditAMICLI_UpdatesAMIID(t *testing.T) {
	path := amiPath(t)
	seedAMIs(t, path, AMIEntry{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-old"})

	err := EditAMICLI(path, "Ubuntu 22.04 LTS", "us-east-1", "ami-newid")
	require.NoError(t, err)

	amis, err := LoadAMIs(path)
	require.NoError(t, err)
	id, ok := amis.GetAMI("Ubuntu 22.04 LTS", "us-east-1")
	require.True(t, ok)
	assert.Equal(t, "ami-newid", id)
}

func TestEditAMICLI_NotFound(t *testing.T) {
	path := amiPath(t)
	require.NoError(t, (&AMIsConfig{AMIs: []AMIEntry{}}).Save(path))

	err := EditAMICLI(path, "Ghost Distro", "us-east-1", "ami-xyz")
	assert.Error(t, err)
}

func TestEditAMICLI_MissingFlags(t *testing.T) {
	path := amiPath(t)
	assert.Error(t, EditAMICLI(path, "", "us-east-1", "ami-xyz"))
	assert.Error(t, EditAMICLI(path, "Ubuntu 22.04 LTS", "", "ami-xyz"))
	assert.Error(t, EditAMICLI(path, "Ubuntu 22.04 LTS", "us-east-1", ""))
}
