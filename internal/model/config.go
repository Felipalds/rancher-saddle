package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration for deploying Rancher on AWS.
type Config struct {
	AWSAccessKey      string `json:"aws_access_key" yaml:"aws_access_key"`
	AWSSecretKey      string `json:"aws_secret_key" yaml:"aws_secret_key"`
	AWSRegion         string `json:"aws_region" yaml:"aws_region"`
	SubnetID          string `json:"subnet_id" yaml:"subnet_id"`
	SecurityGroupID   string `json:"security_group_id" yaml:"security_group_id"`
	SSHKeyName        string `json:"ssh_key_name" yaml:"ssh_key_name"`
	SSHPrivateKeyPath string `json:"ssh_private_key_path" yaml:"ssh_private_key_path"`
	NodePrefix        string `json:"node_prefix" yaml:"node_prefix"`
	AMI               string `json:"ami" yaml:"ami"`
	InstanceCount     int    `json:"instance_count" yaml:"instance_count"`
	RootVolumeSize    int    `json:"root_volume_size" yaml:"root_volume_size"`

	// NEW: Kubernetes distribution selection
	KubernetesDistribution string `json:"kubernetes_distribution" yaml:"kubernetes_distribution"` // "rke2" or "k3s"
	KubernetesVersion      string `json:"kubernetes_version" yaml:"kubernetes_version"`           // Version for selected distro

	// Backward compatibility
	RKE2Version    string `json:"rke2_version,omitempty" yaml:"rke2_version,omitempty"`
	RancherVersion string `json:"rancher_version" yaml:"rancher_version"`
}

// LoadConfig reads the configuration from YAML or JSON file.
// Prefers .yaml, falls back to .json for backward compatibility.
// If neither exists, returns default config.
func LoadConfig(path string) (*Config, error) {
	// Try to auto-detect format or use provided path
	yamlPath := path
	jsonPath := path

	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".json") {
		// No extension, try both
		yamlPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".yaml"
		jsonPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".json"
	} else if strings.HasSuffix(path, ".json") {
		yamlPath = strings.TrimSuffix(path, ".json") + ".yaml"
	} else if strings.HasSuffix(path, ".yaml") {
		jsonPath = strings.TrimSuffix(path, ".yaml") + ".json"
	}

	var cfg Config
	var data []byte

	// Try YAML first (preferred format)
	if _, err := os.Stat(yamlPath); err == nil {
		data, err = os.ReadFile(yamlPath)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	} else if _, err := os.Stat(jsonPath); err == nil {
		// Fall back to JSON for backward compatibility
		data, err = os.ReadFile(jsonPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		// Auto-migrate: save as YAML and delete JSON
		cfg.Save(yamlPath)
		os.Remove(jsonPath)
	} else {
		// Neither file exists, return default config
		return &Config{
			AWSRegion:              "us-east-1",
			InstanceCount:          1,
			KubernetesDistribution: "rke2",
			KubernetesVersion:      "v1.33.7+rke2r1",
			RancherVersion:         "2.10.2",
		}, nil
	}

	// Set defaults if loaded values are empty (optional, but good for stability)
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = "us-east-1"
	}
	if cfg.InstanceCount == 0 {
		cfg.InstanceCount = 1
	}
	if cfg.NodePrefix == "" {
		cfg.NodePrefix = "rancher-node"
	}
	if cfg.AMI == "" {
		// Default to Ubuntu 22.04 LTS in us-west-2 as requested, or keep generic if region variable.
		// The user asked for "save the ubuntu for west2 on the config json file now"
		// Ubuntu 22.04 LTS amd64 in us-west-2
		cfg.AMI = "ami-0c58b2975bef51185"
	}
	// Backward compatibility: migrate old RKE2Version to new format
	if cfg.RKE2Version != "" && cfg.KubernetesVersion == "" {
		cfg.KubernetesDistribution = "rke2"
		cfg.KubernetesVersion = cfg.RKE2Version
	}

	// Set defaults for new fields
	if cfg.KubernetesDistribution == "" {
		cfg.KubernetesDistribution = "rke2"
	}
	if cfg.KubernetesVersion == "" {
		if cfg.KubernetesDistribution == "rke2" {
			cfg.KubernetesVersion = "v1.33.7+rke2r1"
		} else if cfg.KubernetesDistribution == "k3s" {
			cfg.KubernetesVersion = "v1.30.3+k3s1"
		}
	}

	// Auto-fix invalid version from previous default
	if cfg.KubernetesVersion == "v1.32.9" {
		cfg.KubernetesVersion = "v1.33.7+rke2r1"
	}

	if cfg.RancherVersion == "" {
		cfg.RancherVersion = "2.10.2"
	}

	if cfg.RootVolumeSize == 0 {
		cfg.RootVolumeSize = 20
	}

	return &cfg, nil
}

// Save writes the configuration to the specified YAML file.
func (c *Config) Save(path string) error {
	// Ensure path has .yaml extension
	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		path = strings.TrimSuffix(path, filepath.Ext(path)) + ".yaml"
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// File permission 0600 because it contains secrets
	return os.WriteFile(path, data, 0600)
}
