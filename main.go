package main

import (
	"fmt"
	"os"

	"github.com/Felipalds/go-kubernetes-helper/cmd"
	"github.com/Felipalds/go-kubernetes-helper/internal/cluster"
	"github.com/Felipalds/go-kubernetes-helper/internal/model"
	"github.com/Felipalds/go-kubernetes-helper/internal/tui"
	"github.com/spf13/cobra"
)

func main() {
	var configPath string
	var clusterName string
	var force bool

	var rootCmd = &cobra.Command{
		Use:   "go-kubernetes-helper",
		Short: "Automate Rancher deployment on AWS",
		Long:  "A tool to automate the deployment and management of Rancher clusters on AWS using RKE2",
		Run: func(c *cobra.Command, args []string) {
			// Load Config
			cfg, err := model.LoadConfig(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}

			// Menu loop
			for {
				// Show main menu
				action, err := cmd.RunMenuTUI(cfg)
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
					submitted, err := cmd.RunTUI(cfg)
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

					// Save Config
					if err := cfg.Save(configPath); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Error saving config: %v\n", err)
					}

					// Create Cluster
					if err := cluster.CreateCluster(name, cfg); err != nil {
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

			// Load Config
			cfg, err := model.LoadConfig(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}

			// Launch TUI
			submitted, err := cmd.RunTUI(cfg)
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

			// Save Config
			if err := cfg.Save(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Error saving config: %v\n", err)
			}

			// Create Cluster
			if err := cluster.CreateCluster(name, cfg); err != nil {
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

	// Add commands to root
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)

	// Flags for root command (shared with create)
	rootCmd.Flags().StringVarP(&clusterName, "name", "n", "", "Cluster name")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.json", "Path to configuration file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
