package resource

import (
	"path/filepath"
	"testing"

	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── normalizeKind ─────────────────────────────────────────────────────────────

func TestNormalizeKind(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cluster", "cluster"},
		{"Cluster", "cluster"},
		{"clusters", "cluster"},
		{"CLUSTERS", "cluster"},
		{"credential", "credential"},
		{"credentials", "credential"},
		{"profile", "profile"},
		{"profiles", "profile"},
		{"ami", "ami"},
		{"amis", "ami"},
		{"AMI", "ami"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeKind(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeKind_Unknown(t *testing.T) {
	_, err := normalizeKind("pod")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

// ── ClusterSpec.ToConfig ──────────────────────────────────────────────────────

func TestClusterSpec_ToConfig_Defaults(t *testing.T) {
	spec := ClusterSpec{
		Region:          "us-west-2",
		SubnetID:        "subnet-0abc",
		SecurityGroupID: "sg-0abc",
		AMI:             "ami-0abc",
		AccessKey:       "KEY",
		SecretKey:       "SECRET",
		SSH: SSHSpec{
			KeyName:        "my-key",
			PrivateKeyPath: "/tmp/key.pem",
		},
	}
	cfg, err := spec.ToConfig("my-cluster", "creds.yaml")
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", cfg.ClusterName)
	assert.Equal(t, "aws", cfg.Provider)
	assert.Equal(t, "rke2", cfg.Orchestrator)
	assert.Equal(t, 3, cfg.InstanceCount)
	assert.Equal(t, "k8s-node", cfg.NodePrefix)
	assert.Equal(t, "ubuntu", cfg.SSHUser)
	assert.Equal(t, "KEY", cfg.ProviderConfig["access_key"])
}

func TestClusterSpec_ToConfig_DockerMode(t *testing.T) {
	spec := ClusterSpec{
		Method:          "docker",
		AccessKey:       "KEY",
		SecretKey:       "SECRET",
		Region:          "us-west-2",
		SubnetID:        "subnet-0abc",
		SecurityGroupID: "sg-0abc",
		AMI:             "ami-0abc",
		SSH:             SSHSpec{KeyName: "k", PrivateKeyPath: "/tmp/k"},
	}
	cfg, err := spec.ToConfig("docker-cluster", "creds.yaml")
	require.NoError(t, err)
	assert.Equal(t, "docker", cfg.Orchestrator)
	assert.Equal(t, 1, cfg.InstanceCount)
	assert.Equal(t, "rancher-docker", cfg.NodePrefix)
}

func TestClusterSpec_ToConfig_InlineKeys(t *testing.T) {
	spec := ClusterSpec{
		AccessKey: "INLINE_KEY",
		SecretKey: "INLINE_SECRET",
		SSH:       SSHSpec{KeyName: "k", PrivateKeyPath: "/tmp/k"},
	}
	cfg, err := spec.ToConfig("c", "creds.yaml")
	require.NoError(t, err)
	assert.Equal(t, "INLINE_KEY", cfg.ProviderConfig["access_key"])
	assert.Equal(t, "INLINE_SECRET", cfg.ProviderConfig["secret_key"])
}

func TestClusterSpec_ToConfig_NamedCredential(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	creds := &credentials.CloudCredentials{
		AWS: []credentials.AWSCredential{
			{Name: "prod", AccessKey: "PRODKEY", SecretKey: "PRODSECRET", DefaultRegion: "us-east-1"},
		},
	}
	require.NoError(t, creds.Save(credsFile))

	spec := ClusterSpec{
		Credential: "prod",
		SSH:        SSHSpec{KeyName: "k", PrivateKeyPath: "/tmp/k"},
	}
	cfg, err := spec.ToConfig("c", credsFile)
	require.NoError(t, err)
	assert.Equal(t, "PRODKEY", cfg.ProviderConfig["access_key"])
	assert.Equal(t, "PRODSECRET", cfg.ProviderConfig["secret_key"])
}

func TestClusterSpec_ToConfig_InlineOverridesNamedCredential(t *testing.T) {
	dir := t.TempDir()
	credsFile := filepath.Join(dir, "creds.yaml")
	creds := &credentials.CloudCredentials{
		AWS: []credentials.AWSCredential{
			{Name: "prod", AccessKey: "PRODKEY", SecretKey: "PRODSECRET"},
		},
	}
	require.NoError(t, creds.Save(credsFile))

	spec := ClusterSpec{
		Credential: "prod",
		AccessKey:  "OVERRIDE_KEY", // inline wins
		SSH:        SSHSpec{KeyName: "k", PrivateKeyPath: "/tmp/k"},
	}
	cfg, err := spec.ToConfig("c", credsFile)
	require.NoError(t, err)
	assert.Equal(t, "OVERRIDE_KEY", cfg.ProviderConfig["access_key"])
}

// ── CredentialSpec.ToAWSCredential ───────────────────────────────────────────

func TestCredentialSpec_ToAWSCredential(t *testing.T) {
	spec := CredentialSpec{
		Provider:  "aws",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "secret",
		Region:    "us-east-1",
	}
	cred := spec.ToAWSCredential("prod")
	assert.Equal(t, "prod", cred.Name)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cred.AccessKey)
	assert.Equal(t, "us-east-1", cred.DefaultRegion)
}

// ── ProfileSpec.ToProfile ─────────────────────────────────────────────────────

func TestProfileSpec_ToProfile(t *testing.T) {
	spec := ProfileSpec{
		Region:          "us-east-1",
		SubnetID:        "subnet-0abc",
		SecurityGroupID: "sg-0abc",
		AMI:             "ami-0abc",
		InstanceType:    "t3.xlarge",
		SSH: SSHSpec{
			KeyName:        "my-key",
			PrivateKeyPath: "~/.ssh/key.pem",
			User:           "ubuntu",
		},
	}
	p := spec.ToProfile("us-east")
	assert.Equal(t, "us-east", p.Name)
	assert.Equal(t, "us-east-1", p.Region)
	assert.Equal(t, "subnet-0abc", p.SubnetID)
	assert.Equal(t, "ubuntu", p.SSHUser)
}

// ── AMISpec.ToEntry ───────────────────────────────────────────────────────────

func TestAMISpec_ToEntry(t *testing.T) {
	spec := AMISpec{
		Distro: "Ubuntu 22.04 LTS",
		Region: "us-east-1",
		AMIID:  "ami-0c7217cdde317cfec",
	}
	e := spec.ToEntry("ubuntu-22-us-east-1")
	assert.Equal(t, "ubuntu-22-us-east-1", e.Name)
	assert.Equal(t, "Ubuntu 22.04 LTS", e.Distro)
	assert.Equal(t, "ami-0c7217cdde317cfec", e.AMIID)
}
