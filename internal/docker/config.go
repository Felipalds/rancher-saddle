package docker

// DeployConfig holds configuration for a Docker-based Rancher deployment.
type DeployConfig struct {
	ClusterName       string
	RancherVersion    string
	Prime             bool
	BootstrapPassword string
	ImageTag          string
	Debug             bool
	HostPort          string // e.g. "8443" maps to -p <HostPort>:443
}

// DefaultDeployConfig returns a DeployConfig with sensible defaults.
func DefaultDeployConfig() DeployConfig {
	return DeployConfig{
		RancherVersion:    "2.11.7",
		BootstrapPassword: "admin",
		HostPort:          "443",
	}
}

// ContainerName returns the Docker container name for this cluster.
func (c DeployConfig) ContainerName() string {
	return "rancher-" + c.ClusterName
}

// VolumeName returns the Docker volume name for this cluster.
func (c DeployConfig) VolumeName() string {
	return "rancher-data-" + c.ClusterName
}

// Image returns the full image:tag string.
func (c DeployConfig) Image() string {
	base := "rancher/rancher"
	if c.Prime {
		base = "registry.suse.com/rancher/rancher"
	}

	tag := "v" + c.RancherVersion
	if c.ImageTag != "" {
		tag = c.ImageTag
	}

	return base + ":" + tag
}
