// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Felipalds/go-kubernetes-helper/internal/config"
	"github.com/Felipalds/go-kubernetes-helper/internal/core"
	"github.com/Felipalds/go-kubernetes-helper/internal/orchestrators/rke2"
	"github.com/Felipalds/go-kubernetes-helper/internal/providers/aws"
)

func main() {
	fmt.Println("=== Testing Modular Architecture ===\n")

	// Initialize registry
	registry := core.NewRegistry()
	registry.RegisterProvider(aws.NewProvider())
	registry.RegisterOrchestrator(rke2.NewOrchestrator())

	fmt.Println("✓ Registry initialized")
	fmt.Printf("  Providers: %v\n", registry.ListProviders())
	fmt.Printf("  Orchestrators: %v\n\n", registry.ListOrchestrators())

	// Create test configuration
	cfg := &config.Config{
		Provider:          string(core.ProviderAWS),
		Orchestrator:      string(core.OrchestratorRKE2),
		ClusterName:       "test-cluster",
		NodePrefix:        "test-node",
		InstanceCount:     1,
		SSHKeyName:        "test-key",
		SSHPrivateKeyPath: "/tmp/test-key.pem",
		SSHUser:           "ubuntu",
		ProviderConfig: map[string]interface{}{
			"access_key":        "AKIAIOSFODNN7EXAMPLE",
			"secret_key":        "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":            "us-east-1",
			"subnet_id":         "subnet-12345",
			"security_group_id": "sg-12345",
			"ami":               "ami-0c58b2975bef51185",
			"instance_type":     "t3.xlarge",
			"root_volume_size":  20,
		},
		OrchestratorConfig: map[string]interface{}{
			"rke2_version":    "v1.33.7+rke2r1",
			"rancher_version": "2.10.2",
			"deploy_rancher":  true,
		},
	}

	fmt.Println("✓ Configuration created")

	// Get provider and test generation
	provider, err := registry.GetProvider(cfg.GetProviderType())
	if err != nil {
		fmt.Printf("✗ Failed to get provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Provider retrieved: %s\n", provider.Name())

	// Validate provider config
	if err := provider.Validate(cfg.ProviderConfig); err != nil {
		fmt.Printf("✗ Provider validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Provider config validated")

	// Test infrastructure generation
	testDir := "/tmp/test-k8s-helper"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	ctx := context.Background()
	if err := provider.GenerateInfrastructure(ctx, cfg.ProviderConfig, testDir); err != nil {
		fmt.Printf("✗ Failed to generate infrastructure: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Infrastructure code generated")

	// Check if main.tf was created
	mainTfPath := filepath.Join(testDir, "main.tf")
	if _, err := os.Stat(mainTfPath); os.IsNotExist(err) {
		fmt.Printf("✗ main.tf not found at %s\n", mainTfPath)
		os.Exit(1)
	}

	fmt.Printf("✓ main.tf created at %s\n", mainTfPath)

	// Get orchestrator and test generation
	orchestrator, err := registry.GetOrchestrator(cfg.GetOrchestratorType())
	if err != nil {
		fmt.Printf("✗ Failed to get orchestrator: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Orchestrator retrieved: %s\n", orchestrator.Name())

	// Validate orchestrator config
	if err := orchestrator.Validate(cfg.OrchestratorConfig); err != nil {
		fmt.Printf("✗ Orchestrator validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Orchestrator config validated")

	// Test playbook generation
	if err := orchestrator.GeneratePlaybook(ctx, cfg.OrchestratorConfig, testDir); err != nil {
		fmt.Printf("✗ Failed to generate playbook: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Playbook generated")

	// Check if site.yml was created
	siteYmlPath := filepath.Join(testDir, "site.yml")
	if _, err := os.Stat(siteYmlPath); os.IsNotExist(err) {
		fmt.Printf("✗ site.yml not found at %s\n", siteYmlPath)
		os.Exit(1)
	}

	fmt.Printf("✓ site.yml created at %s\n", siteYmlPath)

	// Test inventory generation
	mockOutputs := &core.InfrastructureOutputs{
		InstanceIPs:      []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
		InstanceDNSNames: []string{"node1.example.com", "node2.example.com", "node3.example.com"},
	}

	orchestratorConfig := map[string]interface{}{
		"ssh_private_key_path": "/tmp/test-key.pem",
		"ssh_user":             "ubuntu",
	}

	if err := orchestrator.GenerateInventory(ctx, mockOutputs, orchestratorConfig, testDir); err != nil {
		fmt.Printf("✗ Failed to generate inventory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Inventory generated")

	// Check if hosts.ini was created
	hostsIniPath := filepath.Join(testDir, "hosts.ini")
	if _, err := os.Stat(hostsIniPath); os.IsNotExist(err) {
		fmt.Printf("✗ hosts.ini not found at %s\n", hostsIniPath)
		os.Exit(1)
	}

	fmt.Printf("✓ hosts.ini created at %s\n", hostsIniPath)

	// Read and display generated files (truncated)
	fmt.Println("\n=== Generated Files ===")

	fmt.Println("\n--- main.tf (first 500 chars) ---")
	mainTfContent, _ := os.ReadFile(mainTfPath)
	if len(mainTfContent) > 500 {
		fmt.Println(string(mainTfContent[:500]) + "...")
	} else {
		fmt.Println(string(mainTfContent))
	}

	fmt.Println("\n--- site.yml (first 500 chars) ---")
	siteYmlContent, _ := os.ReadFile(siteYmlPath)
	if len(siteYmlContent) > 500 {
		fmt.Println(string(siteYmlContent[:500]) + "...")
	} else {
		fmt.Println(string(siteYmlContent))
	}

	fmt.Println("\n--- hosts.ini ---")
	hostsIniContent, _ := os.ReadFile(hostsIniPath)
	fmt.Println(string(hostsIniContent))

	fmt.Println("\n=== All Tests Passed! ===")
}
