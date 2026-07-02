package main

import (
	"fmt"
	"os"

	"github.com/Felipalds/rancher-saddle/cmd"
	"github.com/Felipalds/rancher-saddle/internal/cluster"
	"github.com/Felipalds/rancher-saddle/internal/core"
	dockerorch "github.com/Felipalds/rancher-saddle/internal/orchestrators/docker"
	"github.com/Felipalds/rancher-saddle/internal/orchestrators/k3s"
	"github.com/Felipalds/rancher-saddle/internal/orchestrators/rke2"
	"github.com/Felipalds/rancher-saddle/internal/providers/aws"
	"github.com/Felipalds/rancher-saddle/internal/resource"
	"github.com/spf13/cobra"
)

func init() {
	core.GlobalRegistry.RegisterProvider(aws.NewProvider())
	core.GlobalRegistry.RegisterOrchestrator(rke2.NewOrchestrator())
	core.GlobalRegistry.RegisterOrchestrator(k3s.NewOrchestrator())
	core.GlobalRegistry.RegisterOrchestrator(dockerorch.NewOrchestrator())
}

func main() {
	var (
		configPath      string
		credentialsFile string
	)

	rootCmd := &cobra.Command{
		Use:   "saddle",
		Short: "Kubernetes cluster automation for AWS",
		Long:  "Deploy RKE2, K3s, or Docker Rancher clusters on AWS EC2. Run without arguments to open the interactive TUI.",
		Run: func(c *cobra.Command, args []string) {
			cmd.LaunchTUI()
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "Clusters state file")
	rootCmd.PersistentFlags().StringVar(&credentialsFile, "credentials-file", "cloud-credentials.yaml", "Credentials file")

	// ── apply ────────────────────────────────────────────────────────────────
	var applyFile string

	applyCmd := &cobra.Command{
		Use:   "apply -f <file.yaml>",
		Short: "Create or update resources from a YAML file",
		Long: `Reads a YAML file and applies every resource in it.

Supported kinds: Cluster, Credential, Profile, AMI.
Multi-document files (---) are supported.`,
		Run: func(c *cobra.Command, args []string) {
			if err := resource.ApplyFile(applyFile, configPath, credentialsFile, core.GlobalRegistry); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "Path to YAML resource file (required)")
	applyCmd.MarkFlagRequired("file")

	// ── get ──────────────────────────────────────────────────────────────────
	getCmd := &cobra.Command{
		Use:   "get <kind> [name]",
		Short: "List or describe resources",
		Long: `List all resources of a kind, or describe one by name.

Supported kinds (singular or plural): cluster, credential, profile, ami

Examples:
  saddle get clusters
  saddle get cluster my-cluster
  saddle get credentials
  saddle get credential prod`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(c *cobra.Command, args []string) {
			kind := args[0]
			name := ""
			if len(args) == 2 {
				name = args[1]
			}
			if err := resource.Get(kind, name, configPath, credentialsFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// ── delete ───────────────────────────────────────────────────────────────
	var deleteForce bool

	deleteCmd := &cobra.Command{
		Use:   "delete <kind> <name>",
		Short: "Delete a resource by kind and name",
		Long: `Delete a resource. For clusters, also destroys AWS infrastructure.

Supported kinds: cluster, credential, profile, ami

Examples:
  saddle delete cluster my-cluster
  saddle delete credential prod --force
  saddle delete profile us-east
  saddle delete ami ubuntu-22-us-east-1`,
		Args: cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			if err := resource.Delete(args[0], args[1], deleteForce, configPath, credentialsFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")

	// ── upgrade ──────────────────────────────────────────────────────────────
	var (
		upgradeVersion       string
		upgradeReplicas      int
		upgradeAuditLog      bool
		upgradeAuditLogLevel int
		upgradeImageTag      string
		upgradeDebug         bool
	)

	upgradeCmd := &cobra.Command{
		Use:   "upgrade <cluster-name>",
		Short: "Upgrade Rancher on a running cluster",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			opts := cluster.UpgradeOptions{
				Version:       upgradeVersion,
				Replicas:      upgradeReplicas,
				AuditLog:      upgradeAuditLog,
				AuditLogLevel: upgradeAuditLogLevel,
				ImageTag:      upgradeImageTag,
				Debug:         upgradeDebug,
			}
			if err := cluster.UpgradeCluster(args[0], opts, configPath, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	upgradeCmd.Flags().StringVar(&upgradeVersion, "version", "", "Target Rancher version (defaults to current)")
	upgradeCmd.Flags().IntVar(&upgradeReplicas, "replicas", 1, "Kubernetes deployment replicas")
	upgradeCmd.Flags().BoolVar(&upgradeAuditLog, "audit-log", false, "Enable Rancher audit log")
	upgradeCmd.Flags().IntVar(&upgradeAuditLogLevel, "audit-log-level", 1, "Audit log level (0-3)")
	upgradeCmd.Flags().StringVar(&upgradeImageTag, "image-tag", "", "Hotfix image tag override")
	upgradeCmd.Flags().BoolVar(&upgradeDebug, "debug", false, "Enable Rancher debug mode")

	// ── wire ─────────────────────────────────────────────────────────────────
	rootCmd.AddCommand(applyCmd, getCmd, deleteCmd, upgradeCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
