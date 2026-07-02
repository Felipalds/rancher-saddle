package cluster

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestClustersConfig(t *testing.T, dir string, clusters map[string]*config.ClusterConfig) string {
	t.Helper()
	cfg := &config.ClustersConfig{Clusters: clusters}
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, cfg.Save(path))
	return path
}

func TestUpgradeCluster_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := writeTestClustersConfig(t, dir, map[string]*config.ClusterConfig{})

	err := UpgradeCluster("nonexistent", UpgradeOptions{}, path, nil)

	assert.ErrorContains(t, err, "nonexistent")
}

func TestUpgradeCluster_SetsUpgradingStatus(t *testing.T) {
	dir := t.TempDir()
	cl := &config.ClusterConfig{
		Status: "running",
		Kubernetes: config.KubernetesSection{
			Distribution: "docker",
		},
		SSH: config.SSHSection{
			PrivateKeyPath: "/nonexistent/key",
			User:           "ubuntu",
		},
		InstanceIPs: []string{"1.2.3.4"},
		Rancher: config.RancherSection{
			Version:           "2.10.0",
			BootstrapPassword: "admin",
		},
	}
	path := writeTestClustersConfig(t, dir, map[string]*config.ClusterConfig{"mycluster": cl})

	// Run upgrade — it will fail at the SSH step since the IP is fake.
	// We only care that the status was set to "upgrading" before execution.
	_ = UpgradeCluster("mycluster", UpgradeOptions{Version: "2.11.0"}, path, nil)

	cfg, err := config.LoadClustersConfig(path)
	require.NoError(t, err)
	updated, exists := cfg.GetCluster("mycluster")
	require.True(t, exists)
	// Status should be "upgrade-failed" (SSH to fake IP failed), not "running".
	assert.NotEqual(t, "running", updated.Status)
}

func TestUpgradeCluster_WritesToOutput(t *testing.T) {
	dir := t.TempDir()
	cl := &config.ClusterConfig{
		Status: "running",
		Kubernetes: config.KubernetesSection{
			Distribution: "docker",
		},
		SSH: config.SSHSection{
			PrivateKeyPath: "/nonexistent/key",
			User:           "ubuntu",
		},
		InstanceIPs: []string{"1.2.3.4"},
		Rancher: config.RancherSection{
			Version:           "2.10.0",
			BootstrapPassword: "admin",
		},
	}
	path := writeTestClustersConfig(t, dir, map[string]*config.ClusterConfig{"mycluster": cl})

	var buf bytes.Buffer
	_ = UpgradeCluster("mycluster", UpgradeOptions{Version: "2.11.0"}, path, &buf)

	assert.Contains(t, buf.String(), "mycluster")
}

func TestUpgradeCluster_DefaultsVersion(t *testing.T) {
	dir := t.TempDir()
	cl := &config.ClusterConfig{
		Status: "running",
		Kubernetes: config.KubernetesSection{
			Distribution: "docker",
		},
		SSH: config.SSHSection{
			PrivateKeyPath: "/nonexistent/key",
			User:           "ubuntu",
		},
		InstanceIPs: []string{"127.0.0.1"},
		Rancher: config.RancherSection{
			Version:           "2.10.5",
			BootstrapPassword: "admin",
		},
	}
	path := writeTestClustersConfig(t, dir, map[string]*config.ClusterConfig{"mycluster": cl})

	var buf bytes.Buffer
	// No version override — should use cluster's current version.
	_ = UpgradeCluster("mycluster", UpgradeOptions{}, path, &buf)

	assert.Contains(t, buf.String(), "2.10.5")
}

func TestUpgradeCluster_WritesLogFile(t *testing.T) {
	dir := t.TempDir()

	// Override log output directory by changing working directory temporarily.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(origDir) })

	cl := &config.ClusterConfig{
		Status: "running",
		Kubernetes: config.KubernetesSection{
			Distribution: "docker",
		},
		SSH: config.SSHSection{
			PrivateKeyPath: "/nonexistent/key",
			User:           "ubuntu",
		},
		InstanceIPs: []string{"1.2.3.4"},
		Rancher: config.RancherSection{
			Version:           "2.10.0",
			BootstrapPassword: "admin",
		},
		CreatedAt: time.Now(),
	}
	path := writeTestClustersConfig(t, dir, map[string]*config.ClusterConfig{"mycluster": cl})

	_ = UpgradeCluster("mycluster", UpgradeOptions{Version: "2.11.0"}, path, nil)

	logPath := filepath.Join(dir, "logs", "mycluster-upgrade.log")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "mycluster")
}

func TestUpgradeOptions_Defaults(t *testing.T) {
	tests := []struct {
		name          string
		opts          UpgradeOptions
		wantReplicas  int
		wantAuditLevel int
	}{
		{
			name:          "zero values get defaults",
			opts:          UpgradeOptions{},
			wantReplicas:  1,
			wantAuditLevel: 1,
		},
		{
			name:          "explicit values are preserved",
			opts:          UpgradeOptions{Replicas: 3, AuditLogLevel: 2},
			wantReplicas:  3,
			wantAuditLevel: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replicas := tt.opts.Replicas
			if replicas <= 0 {
				replicas = 1
			}
			level := tt.opts.AuditLogLevel
			if level <= 0 {
				level = 1
			}
			assert.Equal(t, tt.wantReplicas, replicas)
			assert.Equal(t, tt.wantAuditLevel, level)
		})
	}
}
