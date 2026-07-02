package resource

import (
	"path/filepath"
	"testing"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── credential ────────────────────────────────────────────────────────────────

func TestDelete_Credential_Removes(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	creds := &credentials.CloudCredentials{
		AWS: []credentials.AWSCredential{
			{Name: "prod", AccessKey: "KEY", SecretKey: "SEC", DefaultRegion: "us-east-1"},
			{Name: "dev", AccessKey: "KEY2", SecretKey: "SEC2", DefaultRegion: "us-west-2"},
		},
	}
	require.NoError(t, creds.Save(credsFile))

	err := deleteCredential("prod", true, credsFile)
	require.NoError(t, err)

	loaded, err := credentials.LoadCredentials(credsFile)
	require.NoError(t, err)
	assert.Equal(t, []string{"dev"}, loaded.ListAWSCredentials())
}

func TestDelete_Credential_NotFound(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	require.NoError(t, (&credentials.CloudCredentials{}).Save(credsFile))

	err := deleteCredential("ghost", true, credsFile)
	assert.Error(t, err)
}

// ── profile ───────────────────────────────────────────────────────────────────

func TestDelete_Profile_Removes(t *testing.T) {
	dir := t.TempDir()
	profilesFile := filepath.Join(dir, "profiles.yaml")
	profiles := &config.ProfilesConfig{Profiles: map[string]*config.Profile{
		"us-east": {Name: "us-east", Region: "us-east-1"},
		"eu-west": {Name: "eu-west", Region: "eu-west-1"},
	}}
	require.NoError(t, profiles.Save(profilesFile))

	err := deleteProfile("us-east", true, profilesFile)
	require.NoError(t, err)

	loaded, err := config.LoadProfiles(profilesFile)
	require.NoError(t, err)
	_, err = loaded.GetProfile("us-east")
	assert.Error(t, err, "deleted profile must not exist")
	_, err = loaded.GetProfile("eu-west")
	assert.NoError(t, err, "other profile must survive")
}

func TestDelete_Profile_NotFound(t *testing.T) {
	dir := t.TempDir()
	profilesFile := filepath.Join(dir, "profiles.yaml")
	require.NoError(t, (&config.ProfilesConfig{Profiles: make(map[string]*config.Profile)}).Save(profilesFile))

	err := deleteProfile("ghost", true, profilesFile)
	assert.Error(t, err)
}

// ── ami ───────────────────────────────────────────────────────────────────────

func TestDelete_AMI_RemovesByName(t *testing.T) {
	dir := t.TempDir()
	amisFile := filepath.Join(dir, "amis.yaml")
	amis := &config.AMIsConfig{AMIs: []config.AMIEntry{
		{Name: "ubuntu-22-us-east-1", Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0abc"},
		{Name: "rhel-9-us-east-1", Distro: "RHEL 9", Region: "us-east-1", AMIID: "ami-0xyz"},
	}}
	require.NoError(t, amis.Save(amisFile))

	err := deleteAMI("ubuntu-22-us-east-1", true, amisFile)
	require.NoError(t, err)

	loaded, err := config.LoadAMIs(amisFile)
	require.NoError(t, err)
	_, ok := loaded.FindByName("ubuntu-22-us-east-1")
	assert.False(t, ok, "deleted entry must not exist")
	_, ok = loaded.FindByName("rhel-9-us-east-1")
	assert.True(t, ok, "other entry must survive")
}

func TestDelete_AMI_NotFound(t *testing.T) {
	dir := t.TempDir()
	amisFile := filepath.Join(dir, "amis.yaml")
	require.NoError(t, (&config.AMIsConfig{AMIs: []config.AMIEntry{}}).Save(amisFile))

	err := deleteAMI("ghost", true, amisFile)
	assert.Error(t, err)
}

// ── unknown kind ──────────────────────────────────────────────────────────────

func TestDelete_UnknownKind(t *testing.T) {
	err := Delete("pod", "my-pod", true, "config.yaml", "creds.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

// ── cluster (via Delete public API) ──────────────────────────────────────────

func TestDelete_Cluster_NotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, (&config.ClustersConfig{Clusters: make(map[string]*config.ClusterConfig)}).Save(configPath))

	err := Delete("cluster", "ghost", true, configPath, "creds.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}
