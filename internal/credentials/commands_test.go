package credentials

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func credPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "cloud-credentials.yaml")
}

func seedCredentials(t *testing.T, path string, creds ...AWSCredential) {
	t.Helper()
	c := &CloudCredentials{AWS: creds}
	require.NoError(t, c.Save(path))
}

// ── MaskKey ──────────────────────────────────────────────────────────────────

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Typical AWS access key (20 chars)
		{"AKIAIOSFODNN7EXAMPLE", "AKIA****MPLE"},
		// Very short key — fully masked
		{"short", "****"},
		// Exactly 8 chars — fully masked (boundary)
		{"12345678", "****"},
		// 9 chars — first 4 + **** + last 4
		{"123456789", "1234****6789"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, MaskKey(tt.input))
		})
	}
}

// ── ListCredentials ──────────────────────────────────────────────────────────

func TestListCredentials_Empty(t *testing.T) {
	path := credPath(t)
	err := ListCredentials(path)
	assert.NoError(t, err)
}

func TestListCredentials_ShowsEntries(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path,
		AWSCredential{Name: "prod", AccessKey: "AKIAIOSFODNN7EXAMPLE", SecretKey: "secret", DefaultRegion: "us-east-1"},
		AWSCredential{Name: "dev", AccessKey: "AKIAIOSFODNN7DEVEXAM", SecretKey: "devsec", DefaultRegion: "eu-west-1"},
	)
	err := ListCredentials(path)
	assert.NoError(t, err)
}

// ── AddCredential ────────────────────────────────────────────────────────────

func TestAddCredential_CreatesEntry(t *testing.T) {
	path := credPath(t)

	err := AddCredential(path, "myaccount", "AKIAIOSFODNN7EXAMPLE", "supersecret", "us-west-2")
	require.NoError(t, err)

	creds, err := LoadCredentials(path)
	require.NoError(t, err)
	cred, err := creds.GetAWSCredential("myaccount")
	require.NoError(t, err)
	assert.Equal(t, "myaccount", cred.Name)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cred.AccessKey)
	assert.Equal(t, "us-west-2", cred.DefaultRegion)
}

func TestAddCredential_UpdatesExisting(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path, AWSCredential{Name: "prod", AccessKey: "OLD", SecretKey: "oldsec", DefaultRegion: "us-east-1"})

	err := AddCredential(path, "prod", "NEWKEY", "newsecret", "eu-west-1")
	require.NoError(t, err)

	creds, err := LoadCredentials(path)
	require.NoError(t, err)
	cred, err := creds.GetAWSCredential("prod")
	require.NoError(t, err)
	assert.Equal(t, "NEWKEY", cred.AccessKey)
	assert.Equal(t, "eu-west-1", cred.DefaultRegion)
}

func TestAddCredential_ValidationError(t *testing.T) {
	path := credPath(t)

	tests := []struct {
		name      string
		credName  string
		accessKey string
		secretKey string
	}{
		{"empty name", "", "KEY", "SEC"},
		{"empty access key", "prod", "", "SEC"},
		{"empty secret key", "prod", "KEY", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddCredential(path, tt.credName, tt.accessKey, tt.secretKey, "us-east-1")
			assert.Error(t, err)
		})
	}
}

// ── DeleteCredential ─────────────────────────────────────────────────────────

func TestDeleteCredential_RemovesEntry(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path, AWSCredential{Name: "prod", AccessKey: "KEY", SecretKey: "SEC", DefaultRegion: "us-east-1"})

	err := DeleteCredential(path, "prod", true)
	require.NoError(t, err)

	creds, err := LoadCredentials(path)
	require.NoError(t, err)
	assert.Empty(t, creds.ListAWSCredentials())
}

func TestDeleteCredential_LeavesOtherEntries(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path,
		AWSCredential{Name: "prod", AccessKey: "KEY1", SecretKey: "SEC1", DefaultRegion: "us-east-1"},
		AWSCredential{Name: "dev", AccessKey: "KEY2", SecretKey: "SEC2", DefaultRegion: "us-west-2"},
	)

	err := DeleteCredential(path, "prod", true)
	require.NoError(t, err)

	creds, err := LoadCredentials(path)
	require.NoError(t, err)
	names := creds.ListAWSCredentials()
	assert.Equal(t, []string{"dev"}, names)
}

func TestDeleteCredential_NotFound(t *testing.T) {
	path := credPath(t)
	err := DeleteCredential(path, "nonexistent", true)
	assert.Error(t, err)
}

// ── EditCredential ───────────────────────────────────────────────────────────

func TestEditCredential_UpdatesOnlyChangedFields(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path, AWSCredential{
		Name:          "prod",
		AccessKey:     "ORIGKEY",
		SecretKey:     "ORIGSEC",
		DefaultRegion: "us-east-1",
	})

	// Update only region — access key and secret must be unchanged.
	err := EditCredential(path, "prod", "", "", "ap-southeast-1")
	require.NoError(t, err)

	creds, err := LoadCredentials(path)
	require.NoError(t, err)
	cred, err := creds.GetAWSCredential("prod")
	require.NoError(t, err)
	assert.Equal(t, "ORIGKEY", cred.AccessKey)
	assert.Equal(t, "ORIGSEC", cred.SecretKey)
	assert.Equal(t, "ap-southeast-1", cred.DefaultRegion)
}

func TestEditCredential_AllFields(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path, AWSCredential{Name: "dev", AccessKey: "OLD", SecretKey: "OLDSEC", DefaultRegion: "us-east-1"})

	err := EditCredential(path, "dev", "NEWKEY", "NEWSEC", "eu-central-1")
	require.NoError(t, err)

	creds, err := LoadCredentials(path)
	require.NoError(t, err)
	cred, err := creds.GetAWSCredential("dev")
	require.NoError(t, err)
	assert.Equal(t, "NEWKEY", cred.AccessKey)
	assert.Equal(t, "NEWSEC", cred.SecretKey)
	assert.Equal(t, "eu-central-1", cred.DefaultRegion)
}

func TestEditCredential_NotFound(t *testing.T) {
	path := credPath(t)
	err := EditCredential(path, "ghost", "KEY", "SEC", "us-east-1")
	assert.Error(t, err)
}

func TestEditCredential_NothingToUpdate(t *testing.T) {
	path := credPath(t)
	seedCredentials(t, path, AWSCredential{Name: "prod", AccessKey: "KEY", SecretKey: "SEC", DefaultRegion: "us-east-1"})

	// All three update params empty — must return an error.
	err := EditCredential(path, "prod", "", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}
