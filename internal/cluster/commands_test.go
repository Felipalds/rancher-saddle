package cluster

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func writeClustersConfig(t *testing.T, dir string, clusters map[string]*config.ClusterConfig) string {
	t.Helper()
	cfg := &config.ClustersConfig{Clusters: clusters}
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, cfg.Save(path))
	return path
}

func runningCluster() *config.ClusterConfig {
	return &config.ClusterConfig{
		Status:    "running",
		CreatedAt: time.Now(),
		Provider: config.ProviderSection{
			Type:   "aws",
			Config: map[string]interface{}{"region": "us-east-1"},
		},
		Cluster: config.ClusterSection{InstanceCount: 3},
	}
}

// ── ListClusters ─────────────────────────────────────────────────────────────

func TestListClusters_EmptyConfig(t *testing.T) {
	path := writeClustersConfig(t, t.TempDir(), map[string]*config.ClusterConfig{})
	err := ListClusters(path)
	assert.NoError(t, err)
}

func TestListClusters_NonExistentFile(t *testing.T) {
	// A missing file is treated as an empty config — no error.
	err := ListClusters(filepath.Join(t.TempDir(), "missing.yaml"))
	assert.NoError(t, err)
}

func TestListClusters_WithClusters(t *testing.T) {
	dir := t.TempDir()
	path := writeClustersConfig(t, dir, map[string]*config.ClusterConfig{
		"prod": runningCluster(),
		"dev":  {Status: "creating", Cluster: config.ClusterSection{InstanceCount: 1}},
	})
	err := ListClusters(path)
	assert.NoError(t, err)
}

func TestListClusters_RespectsConfigPath(t *testing.T) {
	dir := t.TempDir()

	// Write cluster to a non-default path.
	customPath := writeClustersConfig(t, dir, map[string]*config.ClusterConfig{
		"custom-cluster": runningCluster(),
	})

	// A different (empty) path should show no clusters.
	emptyPath := writeClustersConfig(t, dir, map[string]*config.ClusterConfig{})

	// Neither call should error.
	assert.NoError(t, ListClusters(customPath))
	assert.NoError(t, ListClusters(emptyPath))
}

// ── DeleteCluster ─────────────────────────────────────────────────────────────

func TestDeleteCluster_NotFound(t *testing.T) {
	path := writeClustersConfig(t, t.TempDir(), map[string]*config.ClusterConfig{})
	err := DeleteCluster("ghost", true, path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}

func TestDeleteCluster_RemovesFromConfig(t *testing.T) {
	dir := t.TempDir()
	// BuildDir is empty so os.Stat returns "not found" → tofu destroy is skipped.
	path := writeClustersConfig(t, dir, map[string]*config.ClusterConfig{
		"mycluster": {Status: "running"},
	})

	err := DeleteCluster("mycluster", true, path)
	require.NoError(t, err)

	cfg, err := config.LoadClustersConfig(path)
	require.NoError(t, err)
	_, exists := cfg.GetCluster("mycluster")
	assert.False(t, exists)
}

func TestDeleteCluster_LeavesOtherClusters(t *testing.T) {
	dir := t.TempDir()
	path := writeClustersConfig(t, dir, map[string]*config.ClusterConfig{
		"prod": {Status: "running"},
		"dev":  {Status: "running"},
	})

	err := DeleteCluster("prod", true, path)
	require.NoError(t, err)

	cfg, err := config.LoadClustersConfig(path)
	require.NoError(t, err)
	_, devExists := cfg.GetCluster("dev")
	assert.True(t, devExists, "dev cluster should survive deletion of prod")
	_, prodExists := cfg.GetCluster("prod")
	assert.False(t, prodExists)
}

func TestDeleteCluster_RespectsConfigPath(t *testing.T) {
	dir := t.TempDir()
	customPath := writeClustersConfig(t, dir, map[string]*config.ClusterConfig{
		"target": {Status: "running"},
	})

	err := DeleteCluster("target", true, customPath)
	require.NoError(t, err)

	cfg, err := config.LoadClustersConfig(customPath)
	require.NoError(t, err)
	_, exists := cfg.GetCluster("target")
	assert.False(t, exists)
}
