package resource

import (
	"fmt"
	"strings"

	"github.com/Felipalds/rancher-saddle/internal/cluster"
	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
)

// Delete removes the named resource of the given kind.
// For clusters, this also destroys the AWS infrastructure via tofu.
// Prompts for confirmation unless force is true.
func Delete(kind, name string, force bool, configPath, credentialsFile string) error {
	k, err := normalizeKind(kind)
	if err != nil {
		return err
	}

	switch k {
	case "cluster":
		return cluster.DeleteCluster(name, force, configPath)
	case "credential":
		return deleteCredential(name, force, credentialsFile)
	case "profile":
		return deleteProfile(name, force, "profiles.yaml")
	case "ami":
		return deleteAMI(name, force, "amis.yaml")
	}
	return nil
}

func deleteCredential(name string, force bool, credentialsFile string) error {
	if !force {
		fmt.Printf("Delete credential %q? (yes/no): ", name)
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(resp) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	creds, err := credentials.LoadCredentials(credentialsFile)
	if err != nil {
		return err
	}
	if err := creds.DeleteAWSCredential(name); err != nil {
		return err
	}
	if err := creds.Save(credentialsFile); err != nil {
		return err
	}
	fmt.Printf("credential/%s deleted\n", name)
	return nil
}

func deleteProfile(name string, force bool, profilesFile string) error {
	if !force {
		fmt.Printf("Delete profile %q? (yes/no): ", name)
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(resp) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	profiles, err := config.LoadProfiles(profilesFile)
	if err != nil {
		return err
	}
	if err := profiles.DeleteProfile(name); err != nil {
		return err
	}
	if err := profiles.Save(profilesFile); err != nil {
		return err
	}
	fmt.Printf("profile/%s deleted\n", name)
	return nil
}

func deleteAMI(name string, force bool, amisFile string) error {
	if !force {
		fmt.Printf("Delete ami %q? (yes/no): ", name)
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(resp) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	amis, err := config.LoadAMIs(amisFile)
	if err != nil {
		return err
	}
	if err := amis.DeleteByName(name); err != nil {
		return err
	}
	if err := amis.Save(amisFile); err != nil {
		return err
	}
	fmt.Printf("ami/%s deleted\n", name)
	return nil
}
