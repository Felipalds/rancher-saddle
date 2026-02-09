package cluster

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Felipalds/go-kubernetes-helper/internal/model"
	"github.com/Felipalds/go-kubernetes-helper/internal/workflow"
)

// ListClusters displays all clusters in a table format
func ListClusters() error {
	store, err := NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize cluster store: %w", err)
	}

	clusters := store.List()
	if len(clusters) == 0 {
		fmt.Println("No clusters found.")
		fmt.Println("\nUse 'go-kubernetes-helper create' to create a new cluster.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tNODES\tREGION\tCREATED\tRANCHER URL")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for _, cluster := range clusters {
		age := formatAge(cluster.CreatedAt)
		nodeCount := cluster.Config.InstanceCount
		region := cluster.Config.AWSRegion
		rancherURL := cluster.RancherURL
		if rancherURL == "" {
			rancherURL = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			cluster.Name,
			cluster.Status,
			nodeCount,
			region,
			age,
			rancherURL,
		)
	}

	w.Flush()
	return nil
}

// CreateCluster creates a new cluster with the given configuration
func CreateCluster(name string, config *model.Config) error {
	store, err := NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize cluster store: %w", err)
	}

	// Check if cluster already exists
	if _, err := store.Get(name); err == nil {
		return fmt.Errorf("cluster '%s' already exists", name)
	}

	// Create cluster state
	buildDir := filepath.Join("clusters", name)
	cluster := &ClusterState{
		Name:     name,
		Status:   StatusCreating,
		Config:   config,
		BuildDir: buildDir,
	}

	// Add to store
	if err := store.Add(cluster); err != nil {
		return fmt.Errorf("failed to save cluster state: %w", err)
	}

	fmt.Printf("Creating cluster '%s'...\n", name)

	// Run the deployment workflow
	runner, err := workflow.NewRunner(config)
	if err != nil {
		cluster.Status = StatusFailed
		store.Update(cluster)
		return fmt.Errorf("failed to initialize workflow: %w", err)
	}

	if err := runner.RunWithBuildDir(buildDir); err != nil {
		cluster.Status = StatusFailed
		store.Update(cluster)
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Update cluster state with deployment info
	ips, _ := runner.GetTofuOutput(buildDir, "instance_ips")
	dnsNames, _ := runner.GetTofuOutput(buildDir, "instance_dns_names")

	cluster.Status = StatusRunning
	cluster.InstanceIPs = ips
	cluster.InstanceDNS = dnsNames
	if len(dnsNames) > 0 {
		cluster.RancherURL = fmt.Sprintf("https://%s/dashboard", dnsNames[0])
	}

	if err := store.Update(cluster); err != nil {
		return fmt.Errorf("failed to update cluster state: %w", err)
	}

	fmt.Printf("\n✓ Cluster '%s' created successfully!\n", name)
	return nil
}

// DeleteCluster deletes a cluster and all its resources
func DeleteCluster(name string, force bool) error {
	store, err := NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize cluster store: %w", err)
	}

	cluster, err := store.Get(name)
	if err != nil {
		return err
	}

	if !force {
		fmt.Printf("Are you sure you want to delete cluster '%s'? (yes/no): ", name)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Update status to deleting
	cluster.Status = StatusDeleting
	store.Update(cluster)

	fmt.Printf("Deleting cluster '%s'...\n", name)

	// Run tofu destroy
	buildDir := cluster.BuildDir
	if _, err := os.Stat(buildDir); err == nil {
		fmt.Println("Destroying infrastructure...")
		cmd := exec.Command("tofu", "destroy", "-auto-approve")
		cmd.Dir = buildDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to destroy infrastructure: %v\n", err)
			fmt.Println("You may need to manually clean up AWS resources.")
		}

		// Remove build directory
		fmt.Println("Removing build directory...")
		if err := os.RemoveAll(buildDir); err != nil {
			fmt.Printf("Warning: Failed to remove build directory: %v\n", err)
		}
	}

	// Remove from store
	if err := store.Delete(name); err != nil {
		return fmt.Errorf("failed to remove cluster from store: %w", err)
	}

	fmt.Printf("✓ Cluster '%s' deleted successfully!\n", name)
	return nil
}

// formatAge returns a human-readable time duration
func formatAge(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh", hours)
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}
