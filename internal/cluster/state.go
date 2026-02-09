package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Felipalds/go-kubernetes-helper/internal/model"
)

// ClusterStatus represents the current state of a cluster
type ClusterStatus string

const (
	StatusCreating ClusterStatus = "creating"
	StatusRunning  ClusterStatus = "running"
	StatusFailed   ClusterStatus = "failed"
	StatusDeleting ClusterStatus = "deleting"
)

// ClusterState represents a deployed cluster
type ClusterState struct {
	Name          string          `json:"name"`
	Status        ClusterStatus   `json:"status"`
	Config        *model.Config   `json:"config"`
	BuildDir      string          `json:"build_dir"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	InstanceIPs   []string        `json:"instance_ips,omitempty"`
	InstanceDNS   []string        `json:"instance_dns,omitempty"`
	RancherURL    string          `json:"rancher_url,omitempty"`
}

// Store manages cluster state persistence
type Store struct {
	storePath string
	clusters  map[string]*ClusterState
}

// NewStore creates a new cluster state store
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storeDir := filepath.Join(homeDir, ".go-kubernetes-helper")
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	storePath := filepath.Join(storeDir, "clusters.json")
	store := &Store{
		storePath: storePath,
		clusters:  make(map[string]*ClusterState),
	}

	// Load existing clusters
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load cluster state: %w", err)
	}

	return store, nil
}

// List returns all clusters
func (s *Store) List() []*ClusterState {
	clusters := make([]*ClusterState, 0, len(s.clusters))
	for _, cluster := range s.clusters {
		clusters = append(clusters, cluster)
	}
	return clusters
}

// Get retrieves a cluster by name
func (s *Store) Get(name string) (*ClusterState, error) {
	cluster, exists := s.clusters[name]
	if !exists {
		return nil, fmt.Errorf("cluster '%s' not found", name)
	}
	return cluster, nil
}

// Add adds a new cluster to the store
func (s *Store) Add(cluster *ClusterState) error {
	if _, exists := s.clusters[cluster.Name]; exists {
		return fmt.Errorf("cluster '%s' already exists", cluster.Name)
	}

	cluster.CreatedAt = time.Now()
	cluster.UpdatedAt = time.Now()
	s.clusters[cluster.Name] = cluster

	return s.save()
}

// Update updates an existing cluster in the store
func (s *Store) Update(cluster *ClusterState) error {
	if _, exists := s.clusters[cluster.Name]; !exists {
		return fmt.Errorf("cluster '%s' not found", cluster.Name)
	}

	cluster.UpdatedAt = time.Now()
	s.clusters[cluster.Name] = cluster

	return s.save()
}

// Delete removes a cluster from the store
func (s *Store) Delete(name string) error {
	if _, exists := s.clusters[name]; !exists {
		return fmt.Errorf("cluster '%s' not found", name)
	}

	delete(s.clusters, name)
	return s.save()
}

// load reads the cluster state from disk
func (s *Store) load() error {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.clusters)
}

// save writes the cluster state to disk
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.clusters, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.storePath, data, 0600)
}
