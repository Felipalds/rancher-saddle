package config

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// ListProfilesCLI prints all saved profiles as a table to stdout.
func ListProfilesCLI(path string) error {
	profiles, err := LoadProfiles(path)
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(profiles.Profiles) == 0 {
		fmt.Println("No profiles configured.")
		fmt.Println("\nUse 'saddle profiles add' to create a profile.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tREGION\tINSTANCE TYPE\tAMI")
	fmt.Fprintln(w, strings.Repeat("-", 80))
	for _, name := range profiles.ListProfiles() {
		p, _ := profiles.GetProfile(name)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, p.Region, p.InstanceType, p.AMI)
	}
	w.Flush()
	return nil
}

// AddProfileCLI creates or replaces a profile. Name is required.
func AddProfileCLI(path string, p *Profile) error {
	if p.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	profiles, err := LoadProfiles(path)
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	profiles.AddProfile(p.Name, p)

	if err := profiles.Save(path); err != nil {
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	fmt.Printf("✓ Profile '%s' saved.\n", p.Name)
	return nil
}

// DeleteProfileCLI removes a profile by name.
func DeleteProfileCLI(path, name string, force bool) error {
	if !force {
		fmt.Printf("Delete profile '%s'? (yes/no): ", name)
		var resp string
		fmt.Scanln(&resp)
		if resp != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	profiles, err := LoadProfiles(path)
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}
	if err := profiles.DeleteProfile(name); err != nil {
		return err
	}
	if err := profiles.Save(path); err != nil {
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	fmt.Printf("✓ Profile '%s' deleted.\n", name)
	return nil
}

// EditProfileCLI updates one or more fields of an existing profile.
// Only non-empty values in updates are applied.
func EditProfileCLI(path, name string, updates *Profile) error {
	profiles, err := LoadProfiles(path)
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	existing, err := profiles.GetProfile(name)
	if err != nil {
		return err
	}

	if updates.Region != "" {
		existing.Region = updates.Region
	}
	if updates.SubnetID != "" {
		existing.SubnetID = updates.SubnetID
	}
	if updates.SecurityGroupID != "" {
		existing.SecurityGroupID = updates.SecurityGroupID
	}
	if updates.AMI != "" {
		existing.AMI = updates.AMI
	}
	if updates.InstanceType != "" {
		existing.InstanceType = updates.InstanceType
	}
	if updates.SSHKeyName != "" {
		existing.SSHKeyName = updates.SSHKeyName
	}
	if updates.SSHPrivateKeyPath != "" {
		existing.SSHPrivateKeyPath = updates.SSHPrivateKeyPath
	}
	if updates.SSHUser != "" {
		existing.SSHUser = updates.SSHUser
	}

	profiles.AddProfile(name, existing)
	if err := profiles.Save(path); err != nil {
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	fmt.Printf("✓ Profile '%s' updated.\n", name)
	return nil
}
