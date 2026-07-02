package cluster

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/workflow"
)

// ANSI color codes for status indicators
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

func getStatusDisplay(status string) string {
	switch status {
	case "running":
		return colorGreen + "● running" + colorReset
	case "pending":
		return colorYellow + "⚠ pending" + colorReset
	case "failed":
		return colorRed + "✗ failed" + colorReset
	case "creating":
		return colorCyan + "⟳ creating" + colorReset
	case "deleting":
		return colorGray + "◐ deleting" + colorReset
	default:
		return colorGray + "○ " + status + colorReset
	}
}

// ListClusters displays all clusters from configPath in a table.
func ListClusters(configPath string) error {
	cfg, err := config.LoadClustersConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Clusters) == 0 {
		fmt.Println("No clusters found.")
		fmt.Println("\nUse 'saddle create' to create a new cluster.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tNODES\tREGION\tCREATED\tRANCHER URL")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for name, cluster := range cfg.Clusters {
		age := formatAge(cluster.CreatedAt)
		nodeCount := cluster.Cluster.InstanceCount

		region := "-"
		if r, ok := cluster.Provider.Config["region"].(string); ok {
			region = r
		}

		rancherURL := cluster.RancherURL
		if rancherURL == "" {
			rancherURL = "-"
		}

		status := cluster.Status
		if status == "" {
			status = "unknown"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			name,
			getStatusDisplay(status),
			nodeCount,
			region,
			age,
			rancherURL,
		)
	}

	w.Flush()
	return nil
}

// CreateClusterNew creates a new cluster using the modular architecture.
// configPath is the path to the clusters config file (config.yaml by default).
func CreateClusterNew(name string, cfg *config.Config, registry *core.Registry, configPath string) error {
	clustersCfg, err := config.LoadClustersConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := clustersCfg.GetCluster(name); exists {
		return fmt.Errorf("cluster '%s' already exists", name)
	}

	clusterCfg := config.FromModernConfig(cfg)
	clusterCfg.Status = "creating"
	clusterCfg.BuildDir = filepath.Join("clusters", name)

	clustersCfg.AddCluster(name, clusterCfg)
	if err := clustersCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Creating cluster '%s'...\n", name)

	buildDir := filepath.Join("clusters", name)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	runner, err := workflow.NewModularRunner(cfg, registry)
	if err != nil {
		return fmt.Errorf("failed to create workflow runner: %w", err)
	}
	if err := runner.RunWithBuildDir(buildDir); err != nil {
		clusterCfg.Status = "failed"
		clustersCfg.AddCluster(name, clusterCfg)
		clustersCfg.Save(configPath)
		return fmt.Errorf("deployment failed: %w", err)
	}

	provider, err := registry.GetProvider(cfg.GetProviderType())
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	outputs, err := provider.GetOutputs(nil, buildDir)
	if err != nil {
		fmt.Printf("Warning: failed to get infrastructure outputs: %v\n", err)
	} else {
		clusterCfg.InstanceIPs = outputs.InstanceIPs
		clusterCfg.InstanceDNS = outputs.InstanceDNSNames

		if len(outputs.InstanceDNSNames) > 0 {
			clusterCfg.RancherURL = fmt.Sprintf("https://%s/dashboard", outputs.InstanceDNSNames[0])
		} else if len(outputs.InstanceIPs) > 0 {
			clusterCfg.RancherURL = fmt.Sprintf("https://%s/dashboard", outputs.InstanceIPs[0])
		}
	}

	clusterCfg.Status = "running"
	clustersCfg.AddCluster(name, clusterCfg)
	if err := clustersCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save final config: %w", err)
	}

	fmt.Printf("\n✓ Cluster '%s' created successfully!\n", name)
	if clusterCfg.RancherURL != "" {
		fmt.Printf("Rancher URL: %s\n", clusterCfg.RancherURL)
	}

	return nil
}

// DeleteCluster destroys a cluster's infrastructure and removes it from configPath.
func DeleteCluster(name string, force bool, configPath string) error {
	cfg, err := config.LoadClustersConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cluster, exists := cfg.GetCluster(name)
	if !exists {
		return fmt.Errorf("cluster '%s' not found", name)
	}

	if !force {
		fmt.Printf("Are you sure you want to delete cluster '%s'? (yes/no): ", name)
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	fmt.Printf("Deleting cluster '%s'...\n", name)

	cluster.Status = "deleting"
	cfg.AddCluster(name, cluster)
	cfg.Save(configPath)

	buildDir := cluster.BuildDir
	if buildDir == "" {
		buildDir = filepath.Join("clusters", name)
	}

	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		fmt.Println("Destroying infrastructure...")
		cmd := exec.Command("tofu", "destroy", "-auto-approve")
		cmd.Dir = buildDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: failed to destroy infrastructure: %v\n", err)
			fmt.Println("You may need to manually clean up AWS resources.")
		}

		fmt.Println("Removing build directory...")
		if err := os.RemoveAll(buildDir); err != nil {
			fmt.Printf("Warning: failed to remove build directory: %v\n", err)
		}
	}

	cfg.DeleteCluster(name)
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Cluster '%s' deleted successfully!\n", name)
	return nil
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)
	hours := int(duration.Hours())

	if hours < 1 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", hours/24)
}
