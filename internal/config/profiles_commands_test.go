package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func profPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "profiles.yaml")
}

func seedProfiles(t *testing.T, path string, profiles ...*Profile) {
	t.Helper()
	cfg := &ProfilesConfig{Profiles: make(map[string]*Profile)}
	for _, p := range profiles {
		cfg.AddProfile(p.Name, p)
	}
	require.NoError(t, cfg.Save(path))
}

// ── ListProfilesCLI ──────────────────────────────────────────────────────────

func TestListProfilesCLI_Empty(t *testing.T) {
	path := profPath(t)
	err := ListProfilesCLI(path)
	assert.NoError(t, err)
}

func TestListProfilesCLI_ShowsEntries(t *testing.T) {
	path := profPath(t)
	seedProfiles(t, path,
		&Profile{Name: "default", Region: "us-east-1", InstanceType: "t3.xlarge"},
		&Profile{Name: "eu", Region: "eu-west-1", InstanceType: "m5.large"},
	)
	err := ListProfilesCLI(path)
	assert.NoError(t, err)
}

// ── AddProfileCLI ────────────────────────────────────────────────────────────

func TestAddProfileCLI_CreatesProfile(t *testing.T) {
	path := profPath(t)
	p := &Profile{
		Name:         "myprod",
		Region:       "us-west-2",
		InstanceType: "m5.xlarge",
		SSHUser:      "ec2-user",
	}
	err := AddProfileCLI(path, p)
	require.NoError(t, err)

	profiles, err := LoadProfiles(path)
	require.NoError(t, err)
	got, err := profiles.GetProfile("myprod")
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", got.Region)
	assert.Equal(t, "m5.xlarge", got.InstanceType)
}

func TestAddProfileCLI_ReplacesExisting(t *testing.T) {
	path := profPath(t)
	seedProfiles(t, path, &Profile{Name: "default", Region: "us-east-1", InstanceType: "t3.small"})

	err := AddProfileCLI(path, &Profile{Name: "default", Region: "eu-west-1", InstanceType: "t3.xlarge"})
	require.NoError(t, err)

	profiles, err := LoadProfiles(path)
	require.NoError(t, err)
	got, _ := profiles.GetProfile("default")
	assert.Equal(t, "eu-west-1", got.Region)
}

func TestAddProfileCLI_RequiresName(t *testing.T) {
	path := profPath(t)
	err := AddProfileCLI(path, &Profile{Region: "us-east-1"})
	assert.Error(t, err)
}

// ── DeleteProfileCLI ─────────────────────────────────────────────────────────

func TestDeleteProfileCLI_RemovesProfile(t *testing.T) {
	path := profPath(t)
	seedProfiles(t, path, &Profile{Name: "tobedeleted", Region: "us-east-1"})

	err := DeleteProfileCLI(path, "tobedeleted", true)
	require.NoError(t, err)

	profiles, err := LoadProfiles(path)
	require.NoError(t, err)
	assert.Empty(t, profiles.ListProfiles())
}

func TestDeleteProfileCLI_NotFound(t *testing.T) {
	path := profPath(t)
	err := DeleteProfileCLI(path, "ghost", true)
	assert.Error(t, err)
}

// ── EditProfileCLI ───────────────────────────────────────────────────────────

func TestEditProfileCLI_UpdatesOnlyChangedFields(t *testing.T) {
	path := profPath(t)
	seedProfiles(t, path, &Profile{
		Name:         "base",
		Region:       "us-east-1",
		InstanceType: "t3.small",
		SSHUser:      "ubuntu",
	})

	err := EditProfileCLI(path, "base", &Profile{InstanceType: "m5.xlarge"})
	require.NoError(t, err)

	profiles, err := LoadProfiles(path)
	require.NoError(t, err)
	got, _ := profiles.GetProfile("base")
	assert.Equal(t, "us-east-1", got.Region, "region should be unchanged")
	assert.Equal(t, "m5.xlarge", got.InstanceType)
	assert.Equal(t, "ubuntu", got.SSHUser, "ssh user should be unchanged")
}

func TestEditProfileCLI_NotFound(t *testing.T) {
	path := profPath(t)
	err := EditProfileCLI(path, "ghost", &Profile{Region: "us-east-1"})
	assert.Error(t, err)
}

func TestEditProfileCLI_AllFields(t *testing.T) {
	path := profPath(t)
	seedProfiles(t, path, &Profile{Name: "full", Region: "us-east-1", SubnetID: "subnet-old", SecurityGroupID: "sg-old", AMI: "ami-old", InstanceType: "t3.small", SSHKeyName: "oldkey", SSHPrivateKeyPath: "/old/path", SSHUser: "root"})

	updates := &Profile{
		Region:            "ap-southeast-1",
		SubnetID:          "subnet-new",
		SecurityGroupID:   "sg-new",
		AMI:               "ami-new",
		InstanceType:      "m5.large",
		SSHKeyName:        "newkey",
		SSHPrivateKeyPath: "/new/path",
		SSHUser:           "ubuntu",
	}
	err := EditProfileCLI(path, "full", updates)
	require.NoError(t, err)

	profiles, err := LoadProfiles(path)
	require.NoError(t, err)
	got, _ := profiles.GetProfile("full")
	assert.Equal(t, "ap-southeast-1", got.Region)
	assert.Equal(t, "subnet-new", got.SubnetID)
	assert.Equal(t, "sg-new", got.SecurityGroupID)
	assert.Equal(t, "ami-new", got.AMI)
	assert.Equal(t, "m5.large", got.InstanceType)
	assert.Equal(t, "newkey", got.SSHKeyName)
	assert.Equal(t, "/new/path", got.SSHPrivateKeyPath)
	assert.Equal(t, "ubuntu", got.SSHUser)
}
