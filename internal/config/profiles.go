package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProfilesConfig represents all saved profiles
type ProfilesConfig struct {
	Profiles map[string]*Profile `yaml:"profiles"`
}

// Profile represents a saved configuration profile
type Profile struct {
	Name              string `yaml:"name"`
	Region            string `yaml:"region"`
	SubnetID          string `yaml:"subnet_id"`
	SecurityGroupID   string `yaml:"security_group_id"`
	AMI               string `yaml:"ami"`
	InstanceType      string `yaml:"instance_type"`
	SSHKeyName        string `yaml:"ssh_key_name"`
	SSHPrivateKeyPath string `yaml:"ssh_private_key_path"`
	SSHUser           string `yaml:"ssh_user"`
}

// LoadProfiles loads profiles from file
func LoadProfiles(path string) (*ProfilesConfig, error) {
	// If file doesn't exist, return empty config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ProfilesConfig{
			Profiles: make(map[string]*Profile),
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles file: %w", err)
	}

	var cfg ProfilesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse profiles file: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}

	return &cfg, nil
}

// Save writes profiles to file
func (p *ProfilesConfig) Save(path string) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write profiles file: %w", err)
	}

	return nil
}

// AddProfile adds or updates a profile
func (p *ProfilesConfig) AddProfile(name string, profile *Profile) {
	if p.Profiles == nil {
		p.Profiles = make(map[string]*Profile)
	}
	profile.Name = name
	p.Profiles[name] = profile
}

// GetProfile retrieves a profile by name
func (p *ProfilesConfig) GetProfile(name string) (*Profile, error) {
	profile, ok := p.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}
	return profile, nil
}

// DeleteProfile removes a profile
func (p *ProfilesConfig) DeleteProfile(name string) error {
	if _, ok := p.Profiles[name]; !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}
	delete(p.Profiles, name)
	return nil
}

// ListProfiles returns all profile names
func (p *ProfilesConfig) ListProfiles() []string {
	names := make([]string, 0, len(p.Profiles))
	for name := range p.Profiles {
		names = append(names, name)
	}
	return names
}

// HasProfiles returns true if any profiles exist
func (p *ProfilesConfig) HasProfiles() bool {
	return len(p.Profiles) > 0
}
