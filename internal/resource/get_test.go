package resource

import (
	"path/filepath"
	"testing"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── clusters ──────────────────────────────────────────────────────────────────

func TestGet_Clusters_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, (&config.ClustersConfig{Clusters: make(map[string]*config.ClusterConfig)}).Save(path))

	err := Get("clusters", "", path, "creds.yaml")
	assert.NoError(t, err)
}

func TestGet_Cluster_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, (&config.ClustersConfig{Clusters: make(map[string]*config.ClusterConfig)}).Save(path))

	err := Get("cluster", "ghost", path, "creds.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}

func TestGet_Clusters_WithData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &config.ClustersConfig{Clusters: map[string]*config.ClusterConfig{
		"prod": {Status: "running", Cluster: config.ClusterSection{InstanceCount: 3}},
	}}
	require.NoError(t, cfg.Save(path))

	err := Get("cluster", "", path, "creds.yaml")
	assert.NoError(t, err)
}

// ── credentials ───────────────────────────────────────────────────────────────

func TestGet_Credentials_Empty(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	require.NoError(t, (&credentials.CloudCredentials{}).Save(credsFile))

	err := Get("credentials", "", "config.yaml", credsFile)
	assert.NoError(t, err)
}

func TestGet_Credential_ByName(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	creds := &credentials.CloudCredentials{
		AWS: []credentials.AWSCredential{
			{Name: "prod", AccessKey: "AKIAIOSFODNN7EXAMPLE", SecretKey: "secret", DefaultRegion: "us-east-1"},
		},
	}
	require.NoError(t, creds.Save(credsFile))

	err := Get("credential", "prod", "config.yaml", credsFile)
	assert.NoError(t, err)
}

func TestGet_Credential_NotFound(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	require.NoError(t, (&credentials.CloudCredentials{}).Save(credsFile))

	err := Get("credential", "ghost", "config.yaml", credsFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}

// ── profiles ──────────────────────────────────────────────────────────────────

func TestGet_Profiles_Empty(t *testing.T) {
	dir := t.TempDir()
	profilesFile := filepath.Join(dir, "profiles.yaml")
	require.NoError(t, (&config.ProfilesConfig{Profiles: make(map[string]*config.Profile)}).Save(profilesFile))

	err := getProfile("", profilesFile)
	assert.NoError(t, err)
}

func TestGet_Profile_ByName(t *testing.T) {
	dir := t.TempDir()
	profilesFile := filepath.Join(dir, "profiles.yaml")
	profiles := &config.ProfilesConfig{Profiles: map[string]*config.Profile{
		"us-east": {Name: "us-east", Region: "us-east-1", InstanceType: "t3.xlarge"},
	}}
	require.NoError(t, profiles.Save(profilesFile))

	err := getProfile("us-east", profilesFile)
	assert.NoError(t, err)
}

func TestGet_Profile_NotFound(t *testing.T) {
	dir := t.TempDir()
	profilesFile := filepath.Join(dir, "profiles.yaml")
	require.NoError(t, (&config.ProfilesConfig{Profiles: make(map[string]*config.Profile)}).Save(profilesFile))

	err := getProfile("ghost", profilesFile)
	assert.Error(t, err)
}

// ── AMIs ──────────────────────────────────────────────────────────────────────

func TestGet_AMIs_WithNamedEntry(t *testing.T) {
	dir := t.TempDir()
	amisFile := filepath.Join(dir, "amis.yaml")
	amis := &config.AMIsConfig{AMIs: []config.AMIEntry{
		{Name: "ubuntu-22-us-east-1", Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0abc"},
	}}
	require.NoError(t, amis.Save(amisFile))

	err := getAMI("", amisFile)
	assert.NoError(t, err)
}

func TestGet_AMI_ByName(t *testing.T) {
	dir := t.TempDir()
	amisFile := filepath.Join(dir, "amis.yaml")
	amis := &config.AMIsConfig{AMIs: []config.AMIEntry{
		{Name: "ubuntu-22-us-east-1", Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0abc"},
	}}
	require.NoError(t, amis.Save(amisFile))

	err := getAMI("ubuntu-22-us-east-1", amisFile)
	assert.NoError(t, err)
}

func TestGet_AMI_NotFound(t *testing.T) {
	dir := t.TempDir()
	amisFile := filepath.Join(dir, "amis.yaml")
	require.NoError(t, (&config.AMIsConfig{AMIs: []config.AMIEntry{}}).Save(amisFile))

	err := getAMI("ghost", amisFile)
	assert.Error(t, err)
}

// ── unknown kind ──────────────────────────────────────────────────────────────

func TestGet_UnknownKind(t *testing.T) {
	err := Get("pod", "", "config.yaml", "creds.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}
