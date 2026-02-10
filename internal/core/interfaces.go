package core

import (
	"context"
)

// Provider interface for cloud providers
// Implementations handle infrastructure provisioning (VMs, networking, etc.)
type Provider interface {
	// Name returns the provider type
	Name() ProviderType

	// Validate validates the provider-specific configuration
	Validate(config map[string]interface{}) error

	// GenerateInfrastructure generates infrastructure code (e.g., Terraform) and outputs it to outputDir
	GenerateInfrastructure(ctx context.Context, config map[string]interface{}, outputDir string) error

	// GetOutputs retrieves infrastructure outputs after provisioning (IPs, DNS names, etc.)
	GetOutputs(ctx context.Context, buildDir string) (*InfrastructureOutputs, error)

	// GetRequiredFields returns the configuration fields required by this provider
	GetRequiredFields() []FormField

	// GetDefaultConfig returns default configuration values for this provider
	GetDefaultConfig() map[string]interface{}
}

// Orchestrator interface for Kubernetes distributions
// Implementations handle K8s cluster deployment and configuration
type Orchestrator interface {
	// Name returns the orchestrator type
	Name() OrchestratorType

	// Validate validates the orchestrator-specific configuration
	Validate(config map[string]interface{}) error

	// GeneratePlaybook generates the deployment playbook (e.g., Ansible) and outputs it to outputDir
	GeneratePlaybook(ctx context.Context, config map[string]interface{}, outputDir string) error

	// GenerateInventory generates the inventory file based on infrastructure outputs
	GenerateInventory(ctx context.Context, outputs *InfrastructureOutputs, config map[string]interface{}, outputDir string) error

	// GetRequiredFields returns the configuration fields required by this orchestrator
	GetRequiredFields() []FormField

	// GetDefaultConfig returns default configuration values for this orchestrator
	GetDefaultConfig() map[string]interface{}

	// GetModules returns the logical deployment modules for this orchestrator
	GetModules() []Module
}

// Generator interface for code generation (Terraform, Ansible, etc.)
type Generator interface {
	// Generate generates code from templates
	Generate(ctx context.Context, templatePath string, data interface{}, outputPath string) error
}
