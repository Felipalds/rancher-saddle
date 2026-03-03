package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilesConfig_AddAndGet(t *testing.T) {
	cfg := &ProfilesConfig{Profiles: make(map[string]*Profile)}

	profile := &Profile{
		Region:          "us-east-1",
		SubnetID:        "subnet-123",
		SecurityGroupID: "sg-456",
		AMI:             "ami-789",
		InstanceType:    "t3.xlarge",
		SSHKeyName:      "my-key",
	}

	cfg.AddProfile("prod", profile)

	got, err := cfg.GetProfile("prod")
	require.NoError(t, err)
	assert.Equal(t, "prod", got.Name, "AddProfile should set the Name field")
	assert.Equal(t, "us-east-1", got.Region)
	assert.Equal(t, "subnet-123", got.SubnetID)
}

func TestProfilesConfig_GetNotFound(t *testing.T) {
	cfg := &ProfilesConfig{Profiles: make(map[string]*Profile)}

	_, err := cfg.GetProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProfilesConfig_Delete(t *testing.T) {
	cfg := &ProfilesConfig{Profiles: make(map[string]*Profile)}
	cfg.AddProfile("to-delete", &Profile{Region: "us-east-1"})

	err := cfg.DeleteProfile("to-delete")
	assert.NoError(t, err)

	_, err = cfg.GetProfile("to-delete")
	assert.Error(t, err)
}

func TestProfilesConfig_DeleteNotFound(t *testing.T) {
	cfg := &ProfilesConfig{Profiles: make(map[string]*Profile)}

	err := cfg.DeleteProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProfilesConfig_List(t *testing.T) {
	cfg := &ProfilesConfig{Profiles: make(map[string]*Profile)}
	cfg.AddProfile("beta", &Profile{})
	cfg.AddProfile("alpha", &Profile{})

	names := cfg.ListProfiles()
	assert.Equal(t, 2, len(names))
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
}

func TestProfilesConfig_HasProfiles(t *testing.T) {
	empty := &ProfilesConfig{Profiles: make(map[string]*Profile)}
	assert.False(t, empty.HasProfiles())

	empty.AddProfile("test", &Profile{})
	assert.True(t, empty.HasProfiles())
}

func TestProfilesConfig_AddNilMap(t *testing.T) {
	cfg := &ProfilesConfig{}
	cfg.AddProfile("test", &Profile{Region: "us-east-1"})

	got, err := cfg.GetProfile("test")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", got.Region)
}

func TestProfilesConfig_LoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.yaml")

	original := &ProfilesConfig{Profiles: make(map[string]*Profile)}
	original.AddProfile("my-profile", &Profile{
		Region:            "eu-west-1",
		SubnetID:          "subnet-abc",
		SecurityGroupID:   "sg-def",
		AMI:               "ami-ghi",
		InstanceType:      "t3.large",
		SSHKeyName:        "key-pair",
		SSHPrivateKeyPath: "/home/user/.ssh/key.pem",
		SSHUser:           "ec2-user",
	})

	err := original.Save(path)
	require.NoError(t, err)

	loaded, err := LoadProfiles(path)
	require.NoError(t, err)

	got, err := loaded.GetProfile("my-profile")
	require.NoError(t, err)
	assert.Equal(t, "my-profile", got.Name)
	assert.Equal(t, "eu-west-1", got.Region)
	assert.Equal(t, "subnet-abc", got.SubnetID)
	assert.Equal(t, "ec2-user", got.SSHUser)
}

func TestLoadProfiles_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yaml")

	cfg, err := LoadProfiles(path)
	require.NoError(t, err)
	assert.NotNil(t, cfg.Profiles)
	assert.Empty(t, cfg.Profiles)
}
