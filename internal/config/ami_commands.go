package config

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// ListAMIsCLI prints all AMI catalog entries as a table to stdout.
func ListAMIsCLI(path string) error {
	amis, err := LoadAMIs(path)
	if err != nil {
		return fmt.Errorf("failed to load AMIs: %w", err)
	}

	if len(amis.AMIs) == 0 {
		fmt.Println("No AMI entries found.")
		fmt.Println("\nUse 'saddle amis add' to add an entry.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "DISTRO\tREGION\tAMI ID")
	fmt.Fprintln(w, strings.Repeat("-", 70))
	for _, e := range amis.AMIs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Distro, e.Region, e.AMIID)
	}
	w.Flush()
	return nil
}

// AddAMICLI inserts or replaces the AMI entry identified by (Distro, Region).
func AddAMICLI(path string, entry AMIEntry) error {
	if entry.Distro == "" {
		return fmt.Errorf("distro name is required")
	}
	if entry.Region == "" {
		return fmt.Errorf("region is required")
	}
	if entry.AMIID == "" {
		return fmt.Errorf("AMI ID is required")
	}

	amis, err := LoadAMIs(path)
	if err != nil {
		return fmt.Errorf("failed to load AMIs: %w", err)
	}

	amis.AddEntry(entry)

	if err := amis.Save(path); err != nil {
		return fmt.Errorf("failed to save AMIs: %w", err)
	}

	fmt.Printf("✓ AMI entry '%s' / '%s' saved.\n", entry.Distro, entry.Region)
	return nil
}

// DeleteAMICLI removes the AMI entry identified by (distro, region).
func DeleteAMICLI(path, distro, region string, force bool) error {
	if distro == "" {
		return fmt.Errorf("--distro is required")
	}
	if region == "" {
		return fmt.Errorf("--region is required")
	}

	if !force {
		fmt.Printf("Delete AMI entry '%s' / '%s'? (yes/no): ", distro, region)
		var resp string
		fmt.Scanln(&resp)
		if resp != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	amis, err := LoadAMIs(path)
	if err != nil {
		return fmt.Errorf("failed to load AMIs: %w", err)
	}
	if err := amis.DeleteEntry(distro, region); err != nil {
		return err
	}
	if err := amis.Save(path); err != nil {
		return fmt.Errorf("failed to save AMIs: %w", err)
	}

	fmt.Printf("✓ AMI entry '%s' / '%s' deleted.\n", distro, region)
	return nil
}

// EditAMICLI updates the AMI ID for an existing (distro, region) entry.
func EditAMICLI(path, distro, region, newAMIID string) error {
	if distro == "" {
		return fmt.Errorf("--distro is required")
	}
	if region == "" {
		return fmt.Errorf("--region is required")
	}
	if newAMIID == "" {
		return fmt.Errorf("--ami is required")
	}

	amis, err := LoadAMIs(path)
	if err != nil {
		return fmt.Errorf("failed to load AMIs: %w", err)
	}

	if _, ok := amis.GetAMI(distro, region); !ok {
		return fmt.Errorf("AMI entry %q / %q not found", distro, region)
	}

	amis.AddEntry(AMIEntry{Distro: distro, Region: region, AMIID: newAMIID})

	if err := amis.Save(path); err != nil {
		return fmt.Errorf("failed to save AMIs: %w", err)
	}

	fmt.Printf("✓ AMI entry '%s' / '%s' updated to '%s'.\n", distro, region, newAMIID)
	return nil
}
