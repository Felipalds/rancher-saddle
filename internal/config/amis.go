package config

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// AMIEntry is a single distro→region→AMI-ID mapping row.
type AMIEntry struct {
	Distro string `yaml:"distro"`
	Region string `yaml:"region"`
	AMIID  string `yaml:"ami_id"`
}

// AMIsConfig holds all AMI entries persisted in amis.yaml.
type AMIsConfig struct {
	AMIs []AMIEntry `yaml:"amis"`
}

// DefaultAMIs returns the built-in seed table used when amis.yaml does not
// exist yet. Users can freely edit or extend it afterwards.
func DefaultAMIs() *AMIsConfig {
	entries := []AMIEntry{
		// Ubuntu 22.04 LTS
		{Distro: "Ubuntu 22.04 LTS", Region: "us-east-1", AMIID: "ami-0c7217cdde317cfec"},
		{Distro: "Ubuntu 22.04 LTS", Region: "us-east-2", AMIID: "ami-05fb0b8c1424f266b"},
		{Distro: "Ubuntu 22.04 LTS", Region: "us-west-1", AMIID: "ami-0ce2cb35386fc22e9"},
		{Distro: "Ubuntu 22.04 LTS", Region: "us-west-2", AMIID: "ami-0dc8f589abe99f538"},
		{Distro: "Ubuntu 22.04 LTS", Region: "eu-west-1", AMIID: "ami-0694d931cee176e7d"},
		{Distro: "Ubuntu 22.04 LTS", Region: "eu-west-2", AMIID: "ami-09744628bed84e434"},
		{Distro: "Ubuntu 22.04 LTS", Region: "eu-central-1", AMIID: "ami-0faab6bdbac9486fb"},
		{Distro: "Ubuntu 22.04 LTS", Region: "ap-southeast-1", AMIID: "ami-0df7a207adb9748c7"},
		{Distro: "Ubuntu 22.04 LTS", Region: "ap-southeast-2", AMIID: "ami-0310483fb2b488153"},
		{Distro: "Ubuntu 22.04 LTS", Region: "ap-northeast-1", AMIID: "ami-0d52744d6551d851e"},
		{Distro: "Ubuntu 22.04 LTS", Region: "ap-south-1", AMIID: "ami-0f5ee92e2d63afc18"},
		{Distro: "Ubuntu 22.04 LTS", Region: "sa-east-1", AMIID: "ami-0fb4cf3a99aa89f08"},
		// RHEL 9
		{Distro: "RHEL 9", Region: "us-east-1", AMIID: "ami-0fe630eb857a6ec83"},
		{Distro: "RHEL 9", Region: "us-east-2", AMIID: "ami-05693d5fe8c807e59"},
		{Distro: "RHEL 9", Region: "us-west-1", AMIID: "ami-0ff0b5043ca8428f0"},
		{Distro: "RHEL 9", Region: "us-west-2", AMIID: "ami-0692f64f10c04c66b"},
		{Distro: "RHEL 9", Region: "eu-west-1", AMIID: "ami-0b14a9f8c2e72b6f6"},
		{Distro: "RHEL 9", Region: "eu-west-2", AMIID: "ami-0aab355e1bfa1e72e"},
		{Distro: "RHEL 9", Region: "eu-central-1", AMIID: "ami-0a4a7676b07ba5162"},
		{Distro: "RHEL 9", Region: "ap-southeast-1", AMIID: "ami-0bd66fc77d6c8a5e4"},
		{Distro: "RHEL 9", Region: "ap-southeast-2", AMIID: "ami-0dd6e21a56d8c30f7"},
		{Distro: "RHEL 9", Region: "ap-northeast-1", AMIID: "ami-0f4539b28a18e3d0b"},
		{Distro: "RHEL 9", Region: "ap-south-1", AMIID: "ami-02e3b29e59e1c50a9"},
		{Distro: "RHEL 9", Region: "sa-east-1", AMIID: "ami-0b05ba07e2b99af9e"},
		// SLES 15 SP5
		{Distro: "SLES 15 SP5", Region: "us-east-1", AMIID: "ami-01cf5b14e09028ea5"},
		{Distro: "SLES 15 SP5", Region: "us-east-2", AMIID: "ami-052dc7f41b4c42879"},
		{Distro: "SLES 15 SP5", Region: "us-west-1", AMIID: "ami-06a98ba76c8aa33d7"},
		{Distro: "SLES 15 SP5", Region: "us-west-2", AMIID: "ami-026b6f1e7a0e1ba76"},
		{Distro: "SLES 15 SP5", Region: "eu-west-1", AMIID: "ami-09fd1ed3b26c5d6de"},
		{Distro: "SLES 15 SP5", Region: "eu-west-2", AMIID: "ami-065cd38073f06e90f"},
		{Distro: "SLES 15 SP5", Region: "eu-central-1", AMIID: "ami-065e49f3bb57ef0de"},
		{Distro: "SLES 15 SP5", Region: "ap-southeast-1", AMIID: "ami-06c9e8c5c7c74a32c"},
		{Distro: "SLES 15 SP5", Region: "ap-southeast-2", AMIID: "ami-01b7b39ef58a4da52"},
		{Distro: "SLES 15 SP5", Region: "ap-northeast-1", AMIID: "ami-0ee89af7b8a6d1a79"},
		{Distro: "SLES 15 SP5", Region: "ap-south-1", AMIID: "ami-0e3c41e3be4ce4e00"},
		{Distro: "SLES 15 SP5", Region: "sa-east-1", AMIID: "ami-087f2af2a2a9be9e8"},
	}
	return &AMIsConfig{AMIs: entries}
}

// LoadAMIs loads amis.yaml from path. If the file does not exist it creates it
// with the default seed table and returns that.
func LoadAMIs(path string) (*AMIsConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := DefaultAMIs()
		if saveErr := cfg.Save(path); saveErr != nil {
			return cfg, nil // return defaults even if we can't write
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg AMIsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.AMIs == nil {
		cfg.AMIs = []AMIEntry{}
	}
	return &cfg, nil
}

// Save writes the config to path (file mode 0644).
func (c *AMIsConfig) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// HasAMIs returns true when there is at least one entry.
func (c *AMIsConfig) HasAMIs() bool {
	return len(c.AMIs) > 0
}

// ListDistros returns a deduplicated, sorted list of all distro names.
func (c *AMIsConfig) ListDistros() []string {
	seen := map[string]struct{}{}
	for _, e := range c.AMIs {
		seen[e.Distro] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// GetAMI returns the AMI ID for the given distro and region.
func (c *AMIsConfig) GetAMI(distro, region string) (string, bool) {
	for _, e := range c.AMIs {
		if e.Distro == distro && e.Region == region {
			return e.AMIID, true
		}
	}
	return "", false
}

// FindDistro reverse-looks up the distro name for a given AMI ID + region.
func (c *AMIsConfig) FindDistro(amiID, region string) (string, bool) {
	for _, e := range c.AMIs {
		if e.AMIID == amiID && e.Region == region {
			return e.Distro, true
		}
	}
	return "", false
}

// AddEntry inserts or replaces the entry matching (distro, region).
func (c *AMIsConfig) AddEntry(entry AMIEntry) {
	for i, e := range c.AMIs {
		if e.Distro == entry.Distro && e.Region == entry.Region {
			c.AMIs[i] = entry
			return
		}
	}
	c.AMIs = append(c.AMIs, entry)
}

// DeleteEntry removes the entry matching (distro, region).
// Returns an error when no such entry exists.
func (c *AMIsConfig) DeleteEntry(distro, region string) error {
	for i, e := range c.AMIs {
		if e.Distro == distro && e.Region == region {
			c.AMIs = append(c.AMIs[:i], c.AMIs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("AMI entry %q / %q not found", distro, region)
}
