package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Felipalds/go-kubernetes-helper/internal/model"
	"gopkg.in/yaml.v3"
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
	Name          string          `json:"name" yaml:"name"`
	Status        ClusterStatus   `json:"status" yaml:"status"`
	Config        *model.Config   `json:"config" yaml:"config"`
	BuildDir      string          `json:"build_dir" yaml:"build_dir"`
	CreatedAt     time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" yaml:"updated_at"`
	InstanceIPs   []string        `json:"instance_ips,omitempty" yaml:"instance_ips,omitempty"`
	InstanceDNS   []string        `json:"instance_dns,omitempty" yaml:"instance_dns,omitempty"`
	RancherURL    string          `json:"rancher_url,omitempty" yaml:"rancher_url,omitempty"`
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

	yamlPath := filepath.Join(storeDir, "clusters.yaml")
	jsonPath := filepath.Join(storeDir, "clusters.json")

	store := &Store{
		storePath: yamlPath,
		clusters:  make(map[string]*ClusterState),
	}

	// Try to load from YAML first
	if _, err := os.Stat(yamlPath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load cluster state from YAML: %w", err)
		}
	} else if _, err := os.Stat(jsonPath); err == nil {
		// Migrate from JSON to YAML
		if err := store.loadJSON(jsonPath); err != nil {
			return nil, fmt.Errorf("failed to load cluster state from JSON: %w", err)
		}
		// Save as YAML
		if err := store.save(); err != nil {
			return nil, fmt.Errorf("failed to migrate clusters to YAML: %w", err)
		}
		// Remove old JSON file
		os.Remove(jsonPath)
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

// load reads the cluster state from disk (YAML format)
func (s *Store) load() error {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, &s.clusters)
}

// loadJSON reads the cluster state from JSON (for migration)
func (s *Store) loadJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.clusters)
}

// save writes the cluster state to disk (YAML format)
func (s *Store) save() error {
	data, err := yaml.Marshal(s.clusters)
	if err != nil {
		return err
	}

	return os.WriteFile(s.storePath, data, 0600)
}
