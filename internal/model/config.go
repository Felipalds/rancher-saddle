package model

import (
	"encoding/json"
	"os"
)

// Config holds the application configuration for deploying Rancher on AWS.
type Config struct {
	AWSAccessKey      string `json:"aws_access_key"`
	AWSSecretKey      string `json:"aws_secret_key"`
	AWSRegion         string `json:"aws_region"`
	SubnetID          string `json:"subnet_id"`
	SecurityGroupID   string `json:"security_group_id"`
	SSHKeyName        string `json:"ssh_key_name"`
	SSHPrivateKeyPath string `json:"ssh_private_key_path"`
	NodePrefix        string `json:"node_prefix"`
	AMI               string `json:"ami"`
	InstanceCount     int    `json:"instance_count"`
	RootVolumeSize    int    `json:"root_volume_size"`
	RKE2Version       string `json:"rke2_version"`
	RancherVersion    string `json:"rancher_version"`
}

// LoadConfig reads the configuration from the specified JSON file.
// If the file does not exist, it returns an empty config and no error.
func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{
			AWSRegion:      "us-east-1",
			InstanceCount:  1,
			RKE2Version:    "v1.33.7+rke2r1",
			RancherVersion: "2.10.2",
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
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
	if cfg.RKE2Version == "" {
		cfg.RKE2Version = "v1.33.7+rke2r1"
	}
	// Auto-fix invalid version from previous default
	if cfg.RKE2Version == "v1.32.9" {
		cfg.RKE2Version = "v1.33.7+rke2r1"
	}

	if cfg.RancherVersion == "" {
		cfg.RancherVersion = "2.10.2"
	}

	if cfg.RootVolumeSize == 0 {
		cfg.RootVolumeSize = 20
	}

	return &cfg, nil
}

// SaveConfig writes the configuration to the specified JSON file.
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// File permission 0600 because it contains secrets
	return os.WriteFile(path, data, 0600)
}
