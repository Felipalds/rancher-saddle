package main

import (
	"fmt"
	"os"

	"github.com/Felipalds/go-kubernetes-helper/cmd"
	"github.com/Felipalds/go-kubernetes-helper/internal/cluster"
	"github.com/Felipalds/go-kubernetes-helper/internal/config"
	"github.com/Felipalds/go-kubernetes-helper/internal/core"
	"github.com/Felipalds/go-kubernetes-helper/internal/model"
	"github.com/Felipalds/go-kubernetes-helper/internal/orchestrators/k3s"
	"github.com/Felipalds/go-kubernetes-helper/internal/orchestrators/rke2"
	"github.com/Felipalds/go-kubernetes-helper/internal/providers/aws"
	"github.com/Felipalds/go-kubernetes-helper/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	// Register providers
	core.GlobalRegistry.RegisterProvider(aws.NewProvider())

	// Register orchestrators
	core.GlobalRegistry.RegisterOrchestrator(rke2.NewOrchestrator())
	core.GlobalRegistry.RegisterOrchestrator(k3s.NewOrchestrator())
}

func main() {
	var configPath string
	var clusterName string
	var force bool

	var rootCmd = &cobra.Command{
		Use:   "go-kubernetes-helper",
		Short: "Automate Kubernetes cluster deployment on multiple cloud providers",
		Long:  "A modular tool to automate the deployment and management of Kubernetes clusters across AWS, Azure, GCP, and vSphere using RKE2, K3s, Minikube, or Kubeadm",
		Run: func(c *cobra.Command, args []string) {
			// Load config for TUI
			tuiCfg, err := model.LoadConfig(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}

			// Menu loop
			for {
				// Show main menu
				action, err := cmd.RunMenuTUI(tuiCfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error running menu: %v\n", err)
					os.Exit(1)
				}

				switch action {
				case tui.MenuList:
					// List clusters
					fmt.Println()
					if err := cluster.ListClusters(); err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					}
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()

				case tui.MenuCreate:
					// Create cluster flow
					submitted, err := cmd.RunTUI(tuiCfg)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
						os.Exit(1)
					}

					if !submitted {
						fmt.Println("Cluster creation cancelled.")
						continue
					}

					// Prompt for cluster name
					name := clusterName
					if len(args) > 0 {
						name = args[0]
					}
					if name == "" {
						fmt.Print("\nEnter cluster name: ")
						fmt.Scanln(&name)
						if name == "" {
							fmt.Fprintf(os.Stderr, "Error: cluster name is required\n")
							continue
						}
					}

					// Convert to config format
					cfg := config.FromLegacyConfig(tuiCfg)

					// Validate with registry
					if err := cfg.ValidateWithRegistry(core.GlobalRegistry); err != nil {
						fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
						fmt.Println("\nPress Enter to continue...")
						fmt.Scanln()
						continue
					}

					// Create cluster
					if err := cluster.CreateClusterNew(name, cfg, core.GlobalRegistry); err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						fmt.Println("\nPress Enter to continue...")
						fmt.Scanln()
						continue
					}

					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()

				case tui.MenuDelete:
					// Delete cluster flow
					clusterName, canceled, err := cmd.RunDeleteMenuTUI()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						fmt.Println("\nPress Enter to continue...")
						fmt.Scanln()
						continue
					}

					if canceled || clusterName == "" {
						fmt.Println("Deletion cancelled.")
						continue
					}

					// Confirm deletion
					fmt.Printf("\nAre you sure you want to delete cluster '%s'? (yes/no): ", clusterName)
					var response string
					fmt.Scanln(&response)
					if response != "yes" {
						fmt.Println("Deletion cancelled.")
						continue
					}

					// Delete cluster
					if err := cluster.DeleteCluster(clusterName, true); err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					}

					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()

				case tui.MenuExit:
					fmt.Println("Goodbye!")
					return
				}
			}
		},
	}

	// LIST command
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all clusters",
		Run: func(c *cobra.Command, args []string) {
			if err := cluster.ListClusters(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// CREATE command
	var createCmd = &cobra.Command{
		Use:   "create [cluster-name]",
		Short: "Create a new cluster",
		Args:  cobra.MaximumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			// Determine cluster name
			name := clusterName
			if len(args) > 0 {
				name = args[0]
			}

			// Load config for TUI
			tuiCfg, err := model.LoadConfig(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}

			// Launch TUI
			submitted, err := cmd.RunTUI(tuiCfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
				os.Exit(1)
			}

			if !submitted {
				fmt.Println("Deployment cancelled.")
				return
			}

			// Prompt for cluster name if not provided
			if name == "" {
				fmt.Print("Enter cluster name: ")
				fmt.Scanln(&name)
				if name == "" {
					fmt.Fprintf(os.Stderr, "Error: cluster name is required\n")
					os.Exit(1)
				}
			}

			// Convert to config format
			cfg := config.FromLegacyConfig(tuiCfg)

			// Validate
			if err := cfg.ValidateWithRegistry(core.GlobalRegistry); err != nil {
				fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
				os.Exit(1)
			}

			// Create cluster
			if err := cluster.CreateClusterNew(name, cfg, core.GlobalRegistry); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	createCmd.Flags().StringVarP(&clusterName, "name", "n", "", "Cluster name")

	// DELETE command
	var deleteCmd = &cobra.Command{
		Use:   "delete <cluster-name>",
		Short: "Delete a cluster",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			name := args[0]
			if err := cluster.DeleteCluster(name, force); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	// LIST-PROVIDERS command
	var listProvidersCmd = &cobra.Command{
		Use:   "list-providers",
		Short: "List all registered cloud providers",
		Run: func(c *cobra.Command, args []string) {
			providers := core.GlobalRegistry.ListProviders()
			fmt.Println("Registered Providers:")
			for _, p := range providers {
				fmt.Printf("  - %s\n", p)
			}
		},
	}

	// LIST-ORCHESTRATORS command
	var listOrchestratorsCmd = &cobra.Command{
		Use:   "list-orchestrators",
		Short: "List all registered Kubernetes orchestrators",
		Run: func(c *cobra.Command, args []string) {
			orchestrators := core.GlobalRegistry.ListOrchestrators()
			fmt.Println("Registered Orchestrators:")
			for _, o := range orchestrators {
				fmt.Printf("  - %s\n", o)
			}
		},
	}

	// Add commands to root
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listProvidersCmd)
	rootCmd.AddCommand(listOrchestratorsCmd)

	// Flags for root command (shared)
	rootCmd.Flags().StringVarP(&clusterName, "name", "n", "", "Cluster name")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "Path to configuration file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
