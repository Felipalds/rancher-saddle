package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAMIs(t *testing.T) {
	cfg := DefaultAMIs()

	assert.True(t, cfg.HasAMIs())
	assert.Equal(t, 36, len(cfg.AMIs), "should have 36 default entries (3 distros x 12 regions)")
}

func TestDefaultAMIs_Distros(t *testing.T) {
	cfg := DefaultAMIs()
	distros := cfg.ListDistros()

	assert.Contains(t, distros, "Ubuntu 22.04 LTS")
	assert.Contains(t, distros, "RHEL 9")
	assert.Contains(t, distros, "SLES 15 SP5")
	assert.Equal(t, 3, len(distros))
}

func TestAMIsConfig_GetAMI(t *testing.T) {
	cfg := DefaultAMIs()

	tests := []struct {
		name    string
		distro  string
		region  string
		wantID  string
		wantOK  bool
	}{
		{
			name:   "Ubuntu us-east-1",
			distro: "Ubuntu 22.04 LTS",
			region: "us-east-1",
			wantID: "ami-0c7217cdde317cfec",
			wantOK: true,
		},
		{
			name:   "RHEL us-east-1",
			distro: "RHEL 9",
			region: "us-east-1",
			wantID: "ami-0fe630eb857a6ec83",
			wantOK: true,
		},
		{
			name:   "not found",
			distro: "Fedora",
			region: "us-east-1",
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := cfg.GetAMI(tt.distro, tt.region)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestAMIsConfig_FindDistro(t *testing.T) {
	cfg := DefaultAMIs()

	distro, ok := cfg.FindDistro("ami-0c7217cdde317cfec", "us-east-1")
	assert.True(t, ok)
	assert.Equal(t, "Ubuntu 22.04 LTS", distro)

	_, ok = cfg.FindDistro("ami-nonexistent", "us-east-1")
	assert.False(t, ok)
}

func TestAMIsConfig_AddEntry(t *testing.T) {
	cfg := &AMIsConfig{AMIs: []AMIEntry{}}

	cfg.AddEntry(AMIEntry{Distro: "TestOS", Region: "us-east-1", AMIID: "ami-111"})
	assert.Equal(t, 1, len(cfg.AMIs))

	// Replace existing
	cfg.AddEntry(AMIEntry{Distro: "TestOS", Region: "us-east-1", AMIID: "ami-222"})
	assert.Equal(t, 1, len(cfg.AMIs))
	id, ok := cfg.GetAMI("TestOS", "us-east-1")
	assert.True(t, ok)
	assert.Equal(t, "ami-222", id)

	// Add different region
	cfg.AddEntry(AMIEntry{Distro: "TestOS", Region: "eu-west-1", AMIID: "ami-333"})
	assert.Equal(t, 2, len(cfg.AMIs))
}

func TestAMIsConfig_DeleteEntry(t *testing.T) {
	cfg := &AMIsConfig{AMIs: []AMIEntry{
		{Distro: "TestOS", Region: "us-east-1", AMIID: "ami-111"},
		{Distro: "TestOS", Region: "eu-west-1", AMIID: "ami-222"},
	}}

	err := cfg.DeleteEntry("TestOS", "us-east-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(cfg.AMIs))

	err = cfg.DeleteEntry("TestOS", "us-east-1")
	assert.Error(t, err, "deleting nonexistent entry should error")
}

func TestAMIsConfig_ListDistros(t *testing.T) {
	cfg := &AMIsConfig{AMIs: []AMIEntry{
		{Distro: "Zulu", Region: "us-east-1", AMIID: "ami-1"},
		{Distro: "Alpha", Region: "us-east-1", AMIID: "ami-2"},
		{Distro: "Alpha", Region: "eu-west-1", AMIID: "ami-3"},
	}}

	distros := cfg.ListDistros()
	assert.Equal(t, []string{"Alpha", "Zulu"}, distros, "should be sorted and deduplicated")
}

func TestAMIsConfig_HasAMIs(t *testing.T) {
	empty := &AMIsConfig{AMIs: []AMIEntry{}}
	assert.False(t, empty.HasAMIs())

	nonempty := &AMIsConfig{AMIs: []AMIEntry{{Distro: "X", Region: "Y", AMIID: "Z"}}}
	assert.True(t, nonempty.HasAMIs())
}

func TestAMIsConfig_LoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "amis.yaml")

	original := &AMIsConfig{AMIs: []AMIEntry{
		{Distro: "TestOS", Region: "us-east-1", AMIID: "ami-111"},
		{Distro: "TestOS", Region: "eu-west-1", AMIID: "ami-222"},
	}}

	err := original.Save(path)
	require.NoError(t, err)

	loaded, err := LoadAMIs(path)
	require.NoError(t, err)

	assert.Equal(t, 2, len(loaded.AMIs))
	id, ok := loaded.GetAMI("TestOS", "us-east-1")
	assert.True(t, ok)
	assert.Equal(t, "ami-111", id)
}

func TestLoadAMIs_NonexistentCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "amis.yaml")

	cfg, err := LoadAMIs(path)
	require.NoError(t, err)
	assert.Equal(t, 36, len(cfg.AMIs), "should create default AMIs when file doesn't exist")
}
