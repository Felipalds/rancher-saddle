package resource

import (
	"fmt"
	"strings"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
)

const APIVersion = "saddle/v1"

// Metadata holds the identifying fields shared by all resource kinds.
type Metadata struct {
	Name string `yaml:"name"`
}

// envelope is the minimal structure read from every YAML document to
// determine its kind and name before full decoding.
type envelope struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
}

// ── Cluster ───────────────────────────────────────────────────────────────────

type ClusterResource struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Spec       ClusterSpec `yaml:"spec"`
}

type ClusterSpec struct {
	Method          string      `yaml:"method"`
	Provider        string      `yaml:"provider"`
	Credential      string      `yaml:"credential"`    // name of a saved Credential
	AccessKey       string      `yaml:"accessKey"`     // inline — overrides Credential
	SecretKey       string      `yaml:"secretKey"`     // inline — overrides Credential
	Region          string      `yaml:"region"`
	SubnetID        string      `yaml:"subnetId"`
	SecurityGroupID string      `yaml:"securityGroupId"`
	AMI             string      `yaml:"ami"`
	InstanceType    string      `yaml:"instanceType"`
	InstanceCount   int         `yaml:"instanceCount"`
	NodePrefix      string      `yaml:"nodePrefix"`
	Kubernetes      K8sSpec     `yaml:"kubernetes"`
	SSH             SSHSpec     `yaml:"ssh"`
	Rancher         RancherSpec `yaml:"rancher"`
}

type K8sSpec struct {
	Distribution string `yaml:"distribution"`
	Version      string `yaml:"version"`
}

type SSHSpec struct {
	KeyName        string `yaml:"keyName"`
	PrivateKeyPath string `yaml:"privateKeyPath"`
	User           string `yaml:"user"`
}

type RancherSpec struct {
	Enabled           bool   `yaml:"enabled"`
	Version           string `yaml:"version"`
	Prime             bool   `yaml:"prime"`
	BootstrapPassword string `yaml:"bootstrapPassword"`
	ImageTag          string `yaml:"imageTag"`
	Debug             bool   `yaml:"debug"`
}

// ToConfig converts the ClusterSpec into the workflow-ready Config.
// If the spec has a named Credential, its keys are loaded from credentialsFile.
// Inline accessKey/secretKey always override the named credential.
func (s ClusterSpec) ToConfig(name, credentialsFile string) (*config.Config, error) {
	accessKey := s.AccessKey
	secretKey := s.SecretKey

	if s.Credential != "" {
		creds, err := credentials.LoadCredentials(credentialsFile)
		if err != nil {
			return nil, err
		}
		cred, err := creds.GetAWSCredential(s.Credential)
		if err != nil {
			return nil, err
		}
		if accessKey == "" {
			accessKey = cred.AccessKey
		}
		if secretKey == "" {
			secretKey = cred.SecretKey
		}
	}

	distro := strings.ToLower(s.Kubernetes.Distribution)
	instanceCount := s.InstanceCount
	nodePrefix := s.NodePrefix
	provider := s.Provider

	if strings.ToLower(s.Method) == "docker" {
		distro = "docker"
		instanceCount = 1
		if nodePrefix == "" {
			nodePrefix = "rancher-docker"
		}
	}
	if distro == "" {
		distro = "rke2"
	}
	if nodePrefix == "" {
		nodePrefix = "k8s-node"
	}
	if instanceCount == 0 {
		instanceCount = 3
	}
	if provider == "" {
		provider = "aws"
	}

	k8sVersion := s.Kubernetes.Version
	if k8sVersion == "" && distro != "docker" {
		if distro == "k3s" {
			k8sVersion = "v1.33.7+k3s1"
		} else {
			k8sVersion = "v1.33.7+rke2r1"
		}
	}

	rancherVersion := s.Rancher.Version
	if rancherVersion == "" {
		rancherVersion = "2.11.7"
	}
	bootstrapPwd := s.Rancher.BootstrapPassword
	if bootstrapPwd == "" {
		bootstrapPwd = "admin"
	}
	sshUser := s.SSH.User
	if sshUser == "" {
		sshUser = "ubuntu"
	}

	return &config.Config{
		Provider:          strings.ToLower(provider),
		Orchestrator:      distro,
		ClusterName:       name,
		NodePrefix:        nodePrefix,
		InstanceCount:     instanceCount,
		SSHKeyName:        s.SSH.KeyName,
		SSHPrivateKeyPath: s.SSH.PrivateKeyPath,
		SSHUser:           sshUser,
		ProviderConfig: map[string]interface{}{
			"region":            s.Region,
			"subnet_id":         s.SubnetID,
			"security_group_id": s.SecurityGroupID,
			"ami":               s.AMI,
			"instance_type":     s.InstanceType,
			"access_key":        accessKey,
			"secret_key":        secretKey,
		},
		OrchestratorConfig: map[string]interface{}{
			"version":                    k8sVersion,
			"rancher_version":            rancherVersion,
			"deploy_rancher":             s.Rancher.Enabled,
			"rancher_prime":              s.Rancher.Prime,
			"rancher_bootstrap_password": bootstrapPwd,
			"rancher_image_tag":          s.Rancher.ImageTag,
			"rancher_debug":              s.Rancher.Debug,
		},
		Addons: []string{},
	}, nil
}

// ── Credential ────────────────────────────────────────────────────────────────

type CredentialResource struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   Metadata       `yaml:"metadata"`
	Spec       CredentialSpec `yaml:"spec"`
}

type CredentialSpec struct {
	Provider  string `yaml:"provider"`
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Region    string `yaml:"region"`
}

func (s CredentialSpec) ToAWSCredential(name string) credentials.AWSCredential {
	return credentials.AWSCredential{
		Name:          name,
		AccessKey:     s.AccessKey,
		SecretKey:     s.SecretKey,
		DefaultRegion: s.Region,
	}
}

// ── Profile ───────────────────────────────────────────────────────────────────

type ProfileResource struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Spec       ProfileSpec `yaml:"spec"`
}

type ProfileSpec struct {
	Region          string  `yaml:"region"`
	SubnetID        string  `yaml:"subnetId"`
	SecurityGroupID string  `yaml:"securityGroupId"`
	AMI             string  `yaml:"ami"`
	InstanceType    string  `yaml:"instanceType"`
	SSH             SSHSpec `yaml:"ssh"`
}

func (s ProfileSpec) ToProfile(name string) *config.Profile {
	return &config.Profile{
		Name:              name,
		Region:            s.Region,
		SubnetID:          s.SubnetID,
		SecurityGroupID:   s.SecurityGroupID,
		AMI:               s.AMI,
		InstanceType:      s.InstanceType,
		SSHKeyName:        s.SSH.KeyName,
		SSHPrivateKeyPath: s.SSH.PrivateKeyPath,
		SSHUser:           s.SSH.User,
	}
}

// ── AMI ───────────────────────────────────────────────────────────────────────

type AMIResource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       AMISpec  `yaml:"spec"`
}

type AMISpec struct {
	Distro string `yaml:"distro"`
	Region string `yaml:"region"`
	AMIID  string `yaml:"amiId"`
}

func (s AMISpec) ToEntry(name string) config.AMIEntry {
	return config.AMIEntry{
		Name:   name,
		Distro: s.Distro,
		Region: s.Region,
		AMIID:  s.AMIID,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// normalizeKind accepts singular or plural, case-insensitive, and returns the
// canonical lowercase singular form, or an error for unknown kinds.
func normalizeKind(kind string) (string, error) {
	switch strings.ToLower(kind) {
	case "cluster", "clusters":
		return "cluster", nil
	case "credential", "credentials":
		return "credential", nil
	case "profile", "profiles":
		return "profile", nil
	case "ami", "amis":
		return "ami", nil
	}
	return "", fmt.Errorf("unknown resource type %q — valid types: cluster, credential, profile, ami", kind)
}
