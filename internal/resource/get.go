package resource

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
)

// Get lists all resources of kind (when name is empty) or describes one by name.
func Get(kind, name, configPath, credentialsFile string) error {
	k, err := normalizeKind(kind)
	if err != nil {
		return err
	}

	switch k {
	case "cluster":
		return getCluster(name, configPath)
	case "credential":
		return getCredential(name, credentialsFile)
	case "profile":
		return getProfile(name, "profiles.yaml")
	case "ami":
		return getAMI(name, "amis.yaml")
	}
	return nil
}

// ── cluster ───────────────────────────────────────────────────────────────────

func getCluster(name, configPath string) error {
	cfg, err := config.LoadClustersConfig(configPath)
	if err != nil {
		return err
	}

	if name != "" {
		cluster, exists := cfg.GetCluster(name)
		if !exists {
			return fmt.Errorf("cluster %q not found", name)
		}
		printClusterDetail(name, cluster)
		return nil
	}

	if len(cfg.Clusters) == 0 {
		fmt.Println("No clusters found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tNODES\tREGION\tAGE\tRANCHER URL")
	for _, n := range cfg.ListClusters() {
		cl := cfg.Clusters[n]
		region := "-"
		if r, ok := cl.Provider.Config["region"].(string); ok {
			region = r
		}
		url := cl.RancherURL
		if url == "" {
			url = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			n, cl.Status, cl.Cluster.InstanceCount, region, formatAge(cl.CreatedAt), url)
	}
	w.Flush()
	return nil
}

func printClusterDetail(name string, cl *config.ClusterConfig) {
	region := "-"
	if r, ok := cl.Provider.Config["region"].(string); ok {
		region = r
	}
	url := cl.RancherURL
	if url == "" {
		url = "-"
	}
	fmt.Printf("name:        %s\n", name)
	fmt.Printf("status:      %s\n", cl.Status)
	fmt.Printf("nodes:       %d\n", cl.Cluster.InstanceCount)
	fmt.Printf("provider:    %s\n", cl.Provider.Type)
	fmt.Printf("region:      %s\n", region)
	fmt.Printf("age:         %s\n", formatAge(cl.CreatedAt))
	fmt.Printf("rancher url: %s\n", url)
}

// ── credential ────────────────────────────────────────────────────────────────

func getCredential(name, credentialsFile string) error {
	creds, err := credentials.LoadCredentials(credentialsFile)
	if err != nil {
		return err
	}

	if name != "" {
		cred, err := creds.GetAWSCredential(name)
		if err != nil {
			return err
		}
		printCredentialDetail(cred)
		return nil
	}

	if !creds.HasAWSCredentials() {
		fmt.Println("No credentials found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROVIDER\tREGION\tACCESS KEY")
	for _, n := range creds.ListAWSCredentials() {
		c, _ := creds.GetAWSCredential(n)
		fmt.Fprintf(w, "%s\tAWS\t%s\t%s\n", c.Name, c.DefaultRegion, credentials.MaskKey(c.AccessKey))
	}
	w.Flush()
	return nil
}

func printCredentialDetail(c *credentials.AWSCredential) {
	fmt.Printf("name:       %s\n", c.Name)
	fmt.Printf("provider:   AWS\n")
	fmt.Printf("region:     %s\n", c.DefaultRegion)
	fmt.Printf("access key: %s\n", credentials.MaskKey(c.AccessKey))
}

// ── profile ───────────────────────────────────────────────────────────────────

func getProfile(name, profilesFile string) error {
	profiles, err := config.LoadProfiles(profilesFile)
	if err != nil {
		return err
	}

	if name != "" {
		p, err := profiles.GetProfile(name)
		if err != nil {
			return err
		}
		printProfileDetail(p)
		return nil
	}

	if !profiles.HasProfiles() {
		fmt.Println("No profiles found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tREGION\tINSTANCE TYPE\tAMI")
	for _, n := range profiles.ListProfiles() {
		p, _ := profiles.GetProfile(n)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n, p.Region, p.InstanceType, p.AMI)
	}
	w.Flush()
	return nil
}

func printProfileDetail(p *config.Profile) {
	fmt.Printf("name:           %s\n", p.Name)
	fmt.Printf("region:         %s\n", p.Region)
	fmt.Printf("subnet:         %s\n", p.SubnetID)
	fmt.Printf("security group: %s\n", p.SecurityGroupID)
	fmt.Printf("ami:            %s\n", p.AMI)
	fmt.Printf("instance type:  %s\n", p.InstanceType)
	fmt.Printf("ssh key name:   %s\n", p.SSHKeyName)
	fmt.Printf("ssh key path:   %s\n", p.SSHPrivateKeyPath)
	fmt.Printf("ssh user:       %s\n", p.SSHUser)
}

// ── ami ───────────────────────────────────────────────────────────────────────

func getAMI(name, amisFile string) error {
	amis, err := config.LoadAMIs(amisFile)
	if err != nil {
		return err
	}

	if name != "" {
		entry, ok := amis.FindByName(name)
		if !ok {
			return fmt.Errorf("AMI %q not found", name)
		}
		printAMIDetail(entry)
		return nil
	}

	if !amis.HasAMIs() {
		fmt.Println("No AMI entries found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tDISTRO\tREGION\tAMI ID")
	for _, e := range amis.AMIs {
		n := e.Name
		if n == "" {
			n = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n, e.Distro, e.Region, e.AMIID)
	}
	w.Flush()
	return nil
}

func printAMIDetail(e config.AMIEntry) {
	fmt.Printf("name:   %s\n", e.Name)
	fmt.Printf("distro: %s\n", e.Distro)
	fmt.Printf("region: %s\n", e.Region)
	fmt.Printf("ami id: %s\n", e.AMIID)
}

// ── shared ────────────────────────────────────────────────────────────────────

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	h := int(time.Since(t).Hours())
	switch {
	case h < 1:
		return fmt.Sprintf("%dm", int(time.Since(t).Minutes()))
	case h < 24:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dd", h/24)
	}
}

