package cluster

import (
	"path/filepath"
	"testing"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func writeBlueprintFile(t *testing.T, dir string, clusters map[string]*config.ClusterConfig) string {
	t.Helper()
	path := filepath.Join(dir, "blueprint.yaml")
	cfg := &config.ClustersConfig{Clusters: clusters}
	require.NoError(t, cfg.Save(path))
	return path
}

func minimalCluster() *config.ClusterConfig {
	return &config.ClusterConfig{
		Provider: config.ProviderSection{
			Type: "aws",
			Config: map[string]interface{}{
				"region":            "us-west-2",
				"access_key":        "PLACEHOLDER_ACCESS_KEY",
				"secret_key":        "PLACEHOLDER_SECRET_KEY",
				"subnet_id":         "subnet-0abc123",
				"security_group_id": "sg-0abc123",
				"ami":               "ami-0a3e3ef8596692376",
				"instance_type":     "t3.xlarge",
			},
		},
		Kubernetes: config.KubernetesSection{
			Distribution: "rke2",
			Config: map[string]interface{}{
				"version": "v1.33.7+rke2r1",
			},
		},
		SSH: config.SSHSection{
			KeyName:        "my-key",
			PrivateKeyPath: "/nonexistent/key.pem",
			User:           "ubuntu",
		},
		Cluster: config.ClusterSection{
			NodePrefix:    "repro-sure-11610",
			InstanceCount: 3,
		},
		Rancher: config.RancherSection{
			Version:           "2.13.5",
			Deploy:            true,
			BootstrapPassword: "admin",
		},
	}
}

// ── buildConfigFromCluster ───────────────────────────────────────────────────

func TestBuildConfigFromCluster_SetsClusterName(t *testing.T) {
	cfg := buildConfigFromCluster("repro-sure-11610", minimalCluster(), "", "")
	assert.Equal(t, "repro-sure-11610", cfg.ClusterName)
}

func TestBuildConfigFromCluster_MapsProviderAndOrchestrator(t *testing.T) {
	cfg := buildConfigFromCluster("repro-sure-11610", minimalCluster(), "", "")
	assert.Equal(t, "aws", cfg.Provider)
	assert.Equal(t, "rke2", cfg.Orchestrator)
}

func TestBuildConfigFromCluster_PreservesEmbeddedCredentials(t *testing.T) {
	cfg := buildConfigFromCluster("repro-sure-11610", minimalCluster(), "", "")
	assert.Equal(t, "PLACEHOLDER_ACCESS_KEY", cfg.ProviderConfig["access_key"])
	assert.Equal(t, "PLACEHOLDER_SECRET_KEY", cfg.ProviderConfig["secret_key"])
}

func TestBuildConfigFromCluster_OverridesCredentials(t *testing.T) {
	cfg := buildConfigFromCluster("repro-sure-11610", minimalCluster(), "REAL_KEY", "REAL_SECRET")
	assert.Equal(t, "REAL_KEY", cfg.ProviderConfig["access_key"])
	assert.Equal(t, "REAL_SECRET", cfg.ProviderConfig["secret_key"])
}

func TestBuildConfigFromCluster_EmptyOverrideKeepsEmbedded(t *testing.T) {
	// accessKey="" → no override → placeholder stays
	cfg := buildConfigFromCluster("repro-sure-11610", minimalCluster(), "", "")
	assert.Equal(t, "PLACEHOLDER_ACCESS_KEY", cfg.ProviderConfig["access_key"])
}

func TestBuildConfigFromCluster_MapsSSHFields(t *testing.T) {
	cfg := buildConfigFromCluster("x", minimalCluster(), "", "")
	assert.Equal(t, "my-key", cfg.SSHKeyName)
	assert.Equal(t, "/nonexistent/key.pem", cfg.SSHPrivateKeyPath)
	assert.Equal(t, "ubuntu", cfg.SSHUser)
}

func TestBuildConfigFromCluster_MapsClusterSection(t *testing.T) {
	cfg := buildConfigFromCluster("x", minimalCluster(), "", "")
	assert.Equal(t, 3, cfg.InstanceCount)
	assert.Equal(t, "repro-sure-11610", cfg.NodePrefix)
}

func TestBuildConfigFromCluster_MapsRancherIntoOrchestratorConfig(t *testing.T) {
	cfg := buildConfigFromCluster("x", minimalCluster(), "", "")
	assert.Equal(t, "2.13.5", cfg.OrchestratorConfig["rancher_version"])
	assert.Equal(t, true, cfg.OrchestratorConfig["deploy_rancher"])
	assert.Equal(t, "admin", cfg.OrchestratorConfig["rancher_bootstrap_password"])
}

// ── ApplyFile ────────────────────────────────────────────────────────────────

func TestApplyFile_FileNotFound(t *testing.T) {
	err := ApplyFile("/nonexistent/blueprint.yaml", nil, "", "cloud-credentials.yaml", "config.yaml", nil)
	assert.Error(t, err)
}

func TestApplyFile_EmptyClusters(t *testing.T) {
	path := writeBlueprintFile(t, t.TempDir(), map[string]*config.ClusterConfig{})
	err := ApplyFile(path, nil, "", "cloud-credentials.yaml", "config.yaml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no clusters found")
}

func TestApplyFile_ClusterNameNotInFile(t *testing.T) {
	dir := t.TempDir()
	path := writeBlueprintFile(t, dir, map[string]*config.ClusterConfig{
		"repro-sure-11610": minimalCluster(),
	})
	configPath := filepath.Join(dir, "config.yaml")

	// Ask for a cluster that doesn't exist in the file — credential/registry
	// are never reached, so nil registry is fine here.
	err := ApplyFile(path, []string{"repro-ghost-0"}, "", "cloud-credentials.yaml", configPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repro-ghost-0")
}

func TestApplyFile_CredentialNotFound(t *testing.T) {
	dir := t.TempDir()
	blueprint := writeBlueprintFile(t, dir, map[string]*config.ClusterConfig{
		"repro-sure-11610": minimalCluster(),
	})
	credsPath := filepath.Join(dir, "creds.yaml")
	// Write an empty credentials file.
	require.NoError(t, (&credentials.CloudCredentials{}).Save(credsPath))

	// Credential lookup fails before reaching the registry, so nil is safe.
	err := ApplyFile(blueprint, nil, "nonexistent-cred", credsPath, filepath.Join(dir, "config.yaml"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-cred")
}

func TestApplyFile_CredentialOverrideApplied(t *testing.T) {
	dir := t.TempDir()
	blueprint := writeBlueprintFile(t, dir, map[string]*config.ClusterConfig{
		"repro-sure-11610": minimalCluster(),
	})
	credsPath := filepath.Join(dir, "creds.yaml")
	creds := &credentials.CloudCredentials{
		AWS: []credentials.AWSCredential{
			{Name: "prod", AccessKey: "REALKEY", SecretKey: "REALSECRET", DefaultRegion: "us-west-2"},
		},
	}
	require.NoError(t, creds.Save(credsPath))

	// Use an empty (but non-nil) registry. The deployment will fail with
	// "invalid provider" — that's expected. What must NOT appear is a
	// "not found" credential error, which would mean the lookup failed.
	registry := core.NewRegistry()
	err := ApplyFile(blueprint, nil, "prod", credsPath, filepath.Join(dir, "config.yaml"), registry)
	require.Error(t, err, "deployment should fail without registered providers")
	// A credential error would say "credential … not found".
	// A provider/registry error means the credential lookup succeeded.
	assert.NotContains(t, err.Error(), "credential", "error must not be a credential error")
}
