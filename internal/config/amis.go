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
// exist yet. Only us-west-2 entries are seeded; the previous multi-region
// table was hand-curated and several entries pointed at the wrong OS (e.g.
// the Ubuntu 22.04 us-west-2 AMI actually launched Amazon Linux). Users
// can extend this list via the AMIs management screen or by editing
// amis.yaml directly. See feats/4-dynamic-ami-lookup.md for the planned
// fix that replaces this static table with Terraform data lookups.
func DefaultAMIs() *AMIsConfig {
	entries := []AMIEntry{
		{Distro: "Ubuntu 22.04 LTS", Region: "us-west-2", AMIID: "ami-0640ac12c85f21746"},
		{Distro: "RHEL 9", Region: "us-west-2", AMIID: "ami-0692f64f10c04c66b"},
		{Distro: "SLES 15 SP5", Region: "us-west-2", AMIID: "ami-026b6f1e7a0e1ba76"},
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
