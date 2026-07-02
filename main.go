package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Felipalds/rancher-saddle/cmd"
	"github.com/Felipalds/rancher-saddle/internal/cluster"
	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
	"github.com/Felipalds/rancher-saddle/internal/model"
	dockerorch "github.com/Felipalds/rancher-saddle/internal/orchestrators/docker"
	"github.com/Felipalds/rancher-saddle/internal/orchestrators/k3s"
	"github.com/Felipalds/rancher-saddle/internal/orchestrators/rke2"
	"github.com/Felipalds/rancher-saddle/internal/providers/aws"
	"github.com/spf13/cobra"
)

func init() {
	core.GlobalRegistry.RegisterProvider(aws.NewProvider())
	core.GlobalRegistry.RegisterOrchestrator(rke2.NewOrchestrator())
	core.GlobalRegistry.RegisterOrchestrator(k3s.NewOrchestrator())
	core.GlobalRegistry.RegisterOrchestrator(dockerorch.NewOrchestrator())
}

func main() {
	var configPath string

	rootCmd := &cobra.Command{
		Use:   "saddle",
		Short: "Automate Kubernetes cluster deployment on multiple cloud providers",
		Long:  "A modular tool to automate the deployment and management of Kubernetes clusters across AWS using RKE2, K3s, or Docker Rancher.",
		Run: func(c *cobra.Command, args []string) {
			cmd.LaunchTUI()
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "Path to configuration file")

	// ── list ────────────────────────────────────────────────────────────────
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all clusters",
		Run: func(c *cobra.Command, args []string) {
			if err := cluster.ListClusters(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// ── create ──────────────────────────────────────────────────────────────
	var (
		clusterName       string
		method            string
		provider          string
		credentialName    string
		distro            string
		k8sVersion        string
		nodePrefix        string
		region            string
		subnetID          string
		securityGroup     string
		osImage           string
		amiID             string
		instanceType      string
		instanceCount     int
		sshKeyName        string
		sshKeyPath        string
		sshUser           string
		deployRancher     bool
		rancherPrime      bool
		rancherVersion    string
		bootstrapPassword string
		imageTag          string
		debugMode         bool
		hostPort          string
		profileName       string
	)

	createCmd := &cobra.Command{
		Use:   "create [cluster-name]",
		Short: "Create a new cluster",
		Args:  cobra.MaximumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			name := clusterName
			if len(args) > 0 {
				name = args[0]
			}

			// Non-interactive when any infrastructure flag is provided.
			if c.Flags().Changed("subnet") || c.Flags().Changed("credential") {
				if err := runCreateNonInteractive(c, name, configPath, method, provider,
					credentialName, distro, k8sVersion, nodePrefix, region, subnetID,
					securityGroup, osImage, amiID, instanceType, instanceCount,
					sshKeyName, sshKeyPath, sshUser, deployRancher, rancherPrime,
					rancherVersion, bootstrapPassword, imageTag, debugMode, hostPort,
					profileName); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				return
			}

			// Fall back to TUI.
			tuiCfg, err := model.LoadConfig(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}
			submitted, err := cmd.RunTUI(tuiCfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
				os.Exit(1)
			}
			if !submitted {
				fmt.Println("Deployment cancelled.")
				return
			}
			if name == "" {
				fmt.Print("Enter cluster name: ")
				fmt.Scanln(&name)
				if name == "" {
					fmt.Fprintf(os.Stderr, "Error: cluster name is required\n")
					os.Exit(1)
				}
			}
			cfg := config.FromLegacyConfig(tuiCfg)
			if err := cfg.ValidateWithRegistry(core.GlobalRegistry); err != nil {
				fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
				os.Exit(1)
			}
			if err := cluster.CreateClusterNew(name, cfg, core.GlobalRegistry, configPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	createCmd.Flags().StringVarP(&clusterName, "name", "n", "", "Cluster name")
	createCmd.Flags().StringVar(&method, "method", "local", "Installation method: local or docker")
	createCmd.Flags().StringVar(&provider, "provider", "aws", "Cloud provider (aws)")
	createCmd.Flags().StringVar(&credentialName, "credential", "", "Credential name from cloud-credentials.yaml")
	createCmd.Flags().StringVar(&distro, "distro", "rke2", "Kubernetes distribution: rke2 or k3s (ignored in docker mode)")
	createCmd.Flags().StringVar(&k8sVersion, "k8s-version", "", "Kubernetes version (e.g. v1.33.7+rke2r1)")
	createCmd.Flags().StringVar(&nodePrefix, "node-prefix", "", "EC2 instance name prefix")
	createCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")
	createCmd.Flags().StringVar(&subnetID, "subnet", "", "AWS subnet ID (required for non-interactive mode)")
	createCmd.Flags().StringVar(&securityGroup, "security-group", "", "AWS security group ID")
	createCmd.Flags().StringVar(&osImage, "os-image", "Ubuntu 22.04 LTS", "OS distro name resolved via amis.yaml")
	createCmd.Flags().StringVar(&amiID, "ami", "", "AMI ID (overrides --os-image)")
	createCmd.Flags().StringVar(&instanceType, "instance-type", "t3.xlarge", "EC2 instance type")
	createCmd.Flags().IntVar(&instanceCount, "instance-count", 3, "Number of nodes (forced to 1 in docker mode)")
	createCmd.Flags().StringVar(&sshKeyName, "ssh-key-name", "", "AWS KeyPair name")
	createCmd.Flags().StringVar(&sshKeyPath, "ssh-key-path", "", "Local path to SSH private key file")
	createCmd.Flags().StringVar(&sshUser, "ssh-user", "ubuntu", "SSH login user")
	createCmd.Flags().BoolVar(&deployRancher, "deploy-rancher", false, "Deploy Rancher on the cluster")
	createCmd.Flags().BoolVar(&rancherPrime, "rancher-prime", false, "Use Rancher Prime (registry.suse.com)")
	createCmd.Flags().StringVar(&rancherVersion, "rancher-version", "2.11.7", "Rancher version")
	createCmd.Flags().StringVar(&bootstrapPassword, "bootstrap-password", "admin", "Rancher bootstrap password")
	createCmd.Flags().StringVar(&imageTag, "image-tag", "", "Hotfix image tag override")
	createCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable Rancher debug mode")
	createCmd.Flags().StringVar(&hostPort, "host-port", "443", "HTTPS port for Docker mode")
	createCmd.Flags().StringVar(&profileName, "profile", "", "Load a saved profile to pre-fill infra settings")

	// ── apply ───────────────────────────────────────────────────────────────
	var (
		applyFile       string
		applyCredential string
		applyCredsFile  string
	)

	applyCmd := &cobra.Command{
		Use:   "apply [cluster-name...]",
		Short: "Provision clusters from a YAML blueprint file",
		Long: `Reads a clusters YAML file (-f) and provisions every cluster defined in it.

Pass one or more cluster names as arguments to apply only those clusters.
If no names are given, every cluster in the file is applied.

The YAML schema must match the clusters configuration format (same as config.yaml).`,
		Run: func(c *cobra.Command, args []string) {
			if err := cluster.ApplyFile(applyFile, args, applyCredential, applyCredsFile, configPath, core.GlobalRegistry); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "Path to clusters YAML blueprint (required)")
	applyCmd.Flags().StringVar(&applyCredential, "credential", "", "Credential name from credentials file to override access/secret keys in the blueprint")
	applyCmd.Flags().StringVar(&applyCredsFile, "credentials-file", "cloud-credentials.yaml", "Path to credentials file (used with --credential)")
	applyCmd.MarkFlagRequired("file")

	// ── delete ──────────────────────────────────────────────────────────────
	var force bool

	deleteCmd := &cobra.Command{
		Use:   "delete <cluster-name>",
		Short: "Delete a cluster",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			if err := cluster.DeleteCluster(args[0], force, configPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	// ── upgrade ─────────────────────────────────────────────────────────────
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
		Short: "Upgrade Rancher on an existing cluster",
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

	// ── credentials ─────────────────────────────────────────────────────────
	var credentialsFile string

	credentialsCmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manage cloud provider credentials",
	}
	credentialsCmd.PersistentFlags().StringVar(&credentialsFile, "credentials-file", "cloud-credentials.yaml", "Path to credentials file")

	credListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all credentials",
		Run: func(c *cobra.Command, args []string) {
			if err := credentials.ListCredentials(credentialsFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var (
		credName      string
		credAccessKey string
		credSecretKey string
		credRegion    string
	)

	credAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add or update an AWS credential",
		Run: func(c *cobra.Command, args []string) {
			if err := credentials.AddCredential(credentialsFile, credName, credAccessKey, credSecretKey, credRegion); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	credAddCmd.Flags().StringVar(&credName, "name", "", "Credential name (required)")
	credAddCmd.Flags().StringVar(&credAccessKey, "access-key", "", "AWS access key ID (required)")
	credAddCmd.Flags().StringVar(&credSecretKey, "secret-key", "", "AWS secret access key (required)")
	credAddCmd.Flags().StringVar(&credRegion, "region", "", "Default AWS region")
	credAddCmd.MarkFlagRequired("name")
	credAddCmd.MarkFlagRequired("access-key")
	credAddCmd.MarkFlagRequired("secret-key")

	var credForce bool

	credDeleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a credential",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			if err := credentials.DeleteCredential(credentialsFile, args[0], credForce); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	credDeleteCmd.Flags().BoolVarP(&credForce, "force", "f", false, "Skip confirmation")

	var (
		credEditAccessKey string
		credEditSecretKey string
		credEditRegion    string
	)

	credEditCmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Update one or more fields of an existing credential",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			if err := credentials.EditCredential(credentialsFile, args[0], credEditAccessKey, credEditSecretKey, credEditRegion); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	credEditCmd.Flags().StringVar(&credEditAccessKey, "access-key", "", "New access key ID")
	credEditCmd.Flags().StringVar(&credEditSecretKey, "secret-key", "", "New secret access key")
	credEditCmd.Flags().StringVar(&credEditRegion, "region", "", "New default region")

	credentialsCmd.AddCommand(credListCmd, credAddCmd, credDeleteCmd, credEditCmd)

	// ── profiles ────────────────────────────────────────────────────────────
	var profilesFile string

	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage reusable infrastructure profiles",
	}
	profilesCmd.PersistentFlags().StringVar(&profilesFile, "profiles-file", "profiles.yaml", "Path to profiles file")

	profileListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Run: func(c *cobra.Command, args []string) {
			if err := config.ListProfilesCLI(profilesFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var (
		profName            string
		profRegion          string
		profSubnet          string
		profSecurityGroup   string
		profAMI             string
		profInstanceType    string
		profSSHKeyName      string
		profSSHKeyPath      string
		profSSHUser         string
	)

	profileAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Create or replace a profile",
		Run: func(c *cobra.Command, args []string) {
			p := &config.Profile{
				Name:              profName,
				Region:            profRegion,
				SubnetID:          profSubnet,
				SecurityGroupID:   profSecurityGroup,
				AMI:               profAMI,
				InstanceType:      profInstanceType,
				SSHKeyName:        profSSHKeyName,
				SSHPrivateKeyPath: profSSHKeyPath,
				SSHUser:           profSSHUser,
			}
			if err := config.AddProfileCLI(profilesFile, p); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	profileAddCmd.Flags().StringVar(&profName, "name", "", "Profile name (required)")
	profileAddCmd.Flags().StringVar(&profRegion, "region", "", "AWS region")
	profileAddCmd.Flags().StringVar(&profSubnet, "subnet", "", "Subnet ID")
	profileAddCmd.Flags().StringVar(&profSecurityGroup, "security-group", "", "Security group ID")
	profileAddCmd.Flags().StringVar(&profAMI, "ami", "", "AMI ID")
	profileAddCmd.Flags().StringVar(&profInstanceType, "instance-type", "", "EC2 instance type")
	profileAddCmd.Flags().StringVar(&profSSHKeyName, "ssh-key-name", "", "AWS KeyPair name")
	profileAddCmd.Flags().StringVar(&profSSHKeyPath, "ssh-key-path", "", "Path to SSH private key")
	profileAddCmd.Flags().StringVar(&profSSHUser, "ssh-user", "", "SSH login user")
	profileAddCmd.MarkFlagRequired("name")

	var profForce bool

	profileDeleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			if err := config.DeleteProfileCLI(profilesFile, args[0], profForce); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	profileDeleteCmd.Flags().BoolVarP(&profForce, "force", "f", false, "Skip confirmation")

	var (
		profEditRegion        string
		profEditSubnet        string
		profEditSecurityGroup string
		profEditAMI           string
		profEditInstanceType  string
		profEditSSHKeyName    string
		profEditSSHKeyPath    string
		profEditSSHUser       string
	)

	profileEditCmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Update one or more fields of an existing profile",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			updates := &config.Profile{
				Region:            profEditRegion,
				SubnetID:          profEditSubnet,
				SecurityGroupID:   profEditSecurityGroup,
				AMI:               profEditAMI,
				InstanceType:      profEditInstanceType,
				SSHKeyName:        profEditSSHKeyName,
				SSHPrivateKeyPath: profEditSSHKeyPath,
				SSHUser:           profEditSSHUser,
			}
			if err := config.EditProfileCLI(profilesFile, args[0], updates); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	profileEditCmd.Flags().StringVar(&profEditRegion, "region", "", "New AWS region")
	profileEditCmd.Flags().StringVar(&profEditSubnet, "subnet", "", "New subnet ID")
	profileEditCmd.Flags().StringVar(&profEditSecurityGroup, "security-group", "", "New security group ID")
	profileEditCmd.Flags().StringVar(&profEditAMI, "ami", "", "New AMI ID")
	profileEditCmd.Flags().StringVar(&profEditInstanceType, "instance-type", "", "New EC2 instance type")
	profileEditCmd.Flags().StringVar(&profEditSSHKeyName, "ssh-key-name", "", "New AWS KeyPair name")
	profileEditCmd.Flags().StringVar(&profEditSSHKeyPath, "ssh-key-path", "", "New SSH private key path")
	profileEditCmd.Flags().StringVar(&profEditSSHUser, "ssh-user", "", "New SSH login user")

	profilesCmd.AddCommand(profileListCmd, profileAddCmd, profileDeleteCmd, profileEditCmd)

	// ── amis ────────────────────────────────────────────────────────────────
	var amisFile string

	amisCmd := &cobra.Command{
		Use:   "amis",
		Short: "Manage the AMI catalog (amis.yaml)",
	}
	amisCmd.PersistentFlags().StringVar(&amisFile, "amis-file", "amis.yaml", "Path to AMI catalog file")

	amisListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all AMI catalog entries",
		Run: func(c *cobra.Command, args []string) {
			if err := config.ListAMIsCLI(amisFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var (
		amiDistro string
		amiRegion string
		amiAMIID  string
	)

	amisAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add or replace an AMI entry",
		Run: func(c *cobra.Command, args []string) {
			entry := config.AMIEntry{Distro: amiDistro, Region: amiRegion, AMIID: amiAMIID}
			if err := config.AddAMICLI(amisFile, entry); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	amisAddCmd.Flags().StringVar(&amiDistro, "distro", "", "Distro name, e.g. 'Ubuntu 22.04 LTS' (required)")
	amisAddCmd.Flags().StringVar(&amiRegion, "region", "", "AWS region, e.g. us-east-1 (required)")
	amisAddCmd.Flags().StringVar(&amiAMIID, "ami", "", "AMI ID, e.g. ami-0abc12345 (required)")
	amisAddCmd.MarkFlagRequired("distro")
	amisAddCmd.MarkFlagRequired("region")
	amisAddCmd.MarkFlagRequired("ami")

	var (
		amiDeleteDistro string
		amiDeleteRegion string
		amiForce        bool
	)

	amisDeleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an AMI entry by distro and region",
		Run: func(c *cobra.Command, args []string) {
			if err := config.DeleteAMICLI(amisFile, amiDeleteDistro, amiDeleteRegion, amiForce); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	amisDeleteCmd.Flags().StringVar(&amiDeleteDistro, "distro", "", "Distro name (required)")
	amisDeleteCmd.Flags().StringVar(&amiDeleteRegion, "region", "", "AWS region (required)")
	amisDeleteCmd.Flags().BoolVarP(&amiForce, "force", "f", false, "Skip confirmation")
	amisDeleteCmd.MarkFlagRequired("distro")
	amisDeleteCmd.MarkFlagRequired("region")

	var (
		amiEditDistro string
		amiEditRegion string
		amiEditAMIID  string
	)

	amisEditCmd := &cobra.Command{
		Use:   "edit",
		Short: "Update the AMI ID for an existing distro/region entry",
		Run: func(c *cobra.Command, args []string) {
			if err := config.EditAMICLI(amisFile, amiEditDistro, amiEditRegion, amiEditAMIID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	amisEditCmd.Flags().StringVar(&amiEditDistro, "distro", "", "Distro name (required)")
	amisEditCmd.Flags().StringVar(&amiEditRegion, "region", "", "AWS region (required)")
	amisEditCmd.Flags().StringVar(&amiEditAMIID, "ami", "", "New AMI ID (required)")
	amisEditCmd.MarkFlagRequired("distro")
	amisEditCmd.MarkFlagRequired("region")
	amisEditCmd.MarkFlagRequired("ami")

	amisCmd.AddCommand(amisListCmd, amisAddCmd, amisDeleteCmd, amisEditCmd)

	// ── informational ────────────────────────────────────────────────────────
	listProvidersCmd := &cobra.Command{
		Use:   "list-providers",
		Short: "List all registered cloud providers",
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("Registered Providers:")
			for _, p := range core.GlobalRegistry.ListProviders() {
				fmt.Printf("  - %s\n", p)
			}
		},
	}

	listOrchestratorsCmd := &cobra.Command{
		Use:   "list-orchestrators",
		Short: "List all registered Kubernetes orchestrators",
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("Registered Orchestrators:")
			for _, o := range core.GlobalRegistry.ListOrchestrators() {
				fmt.Printf("  - %s\n", o)
			}
		},
	}

	// ── wire up ─────────────────────────────────────────────────────────────
	rootCmd.AddCommand(
		listCmd,
		createCmd,
		applyCmd,
		deleteCmd,
		upgradeCmd,
		credentialsCmd,
		profilesCmd,
		amisCmd,
		listProvidersCmd,
		listOrchestratorsCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runCreateNonInteractive builds a config.Config from CLI flags and runs the
// deployment workflow without opening the TUI.
func runCreateNonInteractive(
	c *cobra.Command, name, configPath, method, provider,
	credentialName, distro, k8sVersion, nodePrefix, region, subnetID,
	securityGroup, osImage, amiID, instanceType string, instanceCount int,
	sshKeyName, sshKeyPath, sshUser string, deployRancher, rancherPrime bool,
	rancherVersion, bootstrapPassword, imageTag string, debugMode bool,
	hostPort, profileName string,
) error {
	if name == "" {
		return fmt.Errorf("cluster name is required (pass as argument or --name)")
	}

	// Apply profile overrides first, then explicit flags win.
	if profileName != "" {
		prof, err := config.LoadProfiles("profiles.yaml")
		if err == nil {
			if p, err := prof.GetProfile(profileName); err == nil {
				if !c.Flags().Changed("region") && p.Region != "" {
					region = p.Region
				}
				if !c.Flags().Changed("subnet") && p.SubnetID != "" {
					subnetID = p.SubnetID
				}
				if !c.Flags().Changed("security-group") && p.SecurityGroupID != "" {
					securityGroup = p.SecurityGroupID
				}
				if !c.Flags().Changed("ami") && !c.Flags().Changed("os-image") && p.AMI != "" {
					amiID = p.AMI
				}
				if !c.Flags().Changed("instance-type") && p.InstanceType != "" {
					instanceType = p.InstanceType
				}
				if !c.Flags().Changed("ssh-key-name") && p.SSHKeyName != "" {
					sshKeyName = p.SSHKeyName
				}
				if !c.Flags().Changed("ssh-key-path") && p.SSHPrivateKeyPath != "" {
					sshKeyPath = p.SSHPrivateKeyPath
				}
				if !c.Flags().Changed("ssh-user") && p.SSHUser != "" {
					sshUser = p.SSHUser
				}
			}
		}
	}

	// Validate required fields.
	missing := []string{}
	if subnetID == "" {
		missing = append(missing, "--subnet")
	}
	if securityGroup == "" {
		missing = append(missing, "--security-group")
	}
	if sshKeyName == "" {
		missing = append(missing, "--ssh-key-name")
	}
	if sshKeyPath == "" {
		missing = append(missing, "--ssh-key-path")
	}
	if credentialName == "" {
		missing = append(missing, "--credential")
	}
	if len(missing) > 0 {
		return fmt.Errorf("required flags missing: %s", strings.Join(missing, ", "))
	}

	// Load AWS credentials.
	creds, err := credentials.LoadCredentials("cloud-credentials.yaml")
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	awsCred, err := creds.GetAWSCredential(credentialName)
	if err != nil {
		return fmt.Errorf("credential %q not found: %w", credentialName, err)
	}

	// Resolve AMI.
	resolvedAMI := amiID
	if resolvedAMI == "" {
		amis, err := config.LoadAMIs("amis.yaml")
		if err == nil {
			if id, ok := amis.GetAMI(osImage, region); ok {
				resolvedAMI = id
			}
		}
	}
	if resolvedAMI == "" {
		return fmt.Errorf("could not resolve AMI for %q in %q — use --ami to provide one directly", osImage, region)
	}

	// Docker mode adjustments.
	isDocker := strings.ToLower(method) == "docker"
	if isDocker {
		distro = "docker"
		instanceCount = 1
		deployRancher = true
		if nodePrefix == "" {
			nodePrefix = "rancher-docker"
		}
	} else {
		if nodePrefix == "" {
			nodePrefix = "k8s-node"
		}
		if k8sVersion == "" {
			if strings.ToLower(distro) == "k3s" {
				k8sVersion = "v1.33.7+k3s1"
			} else {
				k8sVersion = "v1.33.7+rke2r1"
			}
		}
	}

	cfg := &config.Config{
		Provider:          strings.ToLower(provider),
		Orchestrator:      strings.ToLower(distro),
		ClusterName:       name,
		NodePrefix:        nodePrefix,
		InstanceCount:     instanceCount,
		SSHKeyName:        sshKeyName,
		SSHPrivateKeyPath: sshKeyPath,
		SSHUser:           sshUser,
		ProviderConfig: map[string]interface{}{
			"region":            region,
			"subnet_id":         subnetID,
			"security_group_id": securityGroup,
			"ami":               resolvedAMI,
			"instance_type":     instanceType,
			"access_key":        awsCred.AccessKey,
			"secret_key":        awsCred.SecretKey,
		},
		OrchestratorConfig: map[string]interface{}{
			"version":                      k8sVersion,
			"rancher_version":              rancherVersion,
			"deploy_rancher":               deployRancher,
			"rancher_prime":                rancherPrime,
			"rancher_bootstrap_password":   bootstrapPassword,
			"rancher_image_tag":            imageTag,
			"rancher_debug":                debugMode,
			"host_port":                    hostPort,
		},
		Addons: []string{},
	}

	if err := cfg.ValidateWithRegistry(core.GlobalRegistry); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return cluster.CreateClusterNew(name, cfg, core.GlobalRegistry, configPath)
}
