package credentials

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// ListCredentials prints all saved credentials as a table to stdout.
func ListCredentials(path string) error {
	creds, err := LoadCredentials(path)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	if len(creds.AWS) == 0 {
		fmt.Println("No credentials configured.")
		fmt.Println("\nUse 'saddle credentials add' to add AWS credentials.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROVIDER\tREGION\tACCESS KEY")
	fmt.Fprintln(w, strings.Repeat("-", 70))
	for _, c := range creds.AWS {
		fmt.Fprintf(w, "%s\tAWS\t%s\t%s\n", c.Name, c.DefaultRegion, MaskKey(c.AccessKey))
	}
	w.Flush()
	return nil
}

// AddCredential adds or updates an AWS credential in the file at path.
func AddCredential(path, name, accessKey, secretKey, region string) error {
	cred := AWSCredential{
		Name:          name,
		AccessKey:     accessKey,
		SecretKey:     secretKey,
		DefaultRegion: region,
	}
	if err := cred.Validate(); err != nil {
		return err
	}

	creds, err := LoadCredentials(path)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if err := creds.AddAWSCredential(cred); err != nil {
		return err
	}
	if err := creds.Save(path); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("✓ Credential '%s' saved.\n", name)
	return nil
}

// DeleteCredential removes an AWS credential by name.
func DeleteCredential(path, name string, force bool) error {
	if !force {
		fmt.Printf("Delete credential '%s'? (yes/no): ", name)
		var resp string
		fmt.Scanln(&resp)
		if resp != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	creds, err := LoadCredentials(path)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if err := creds.DeleteAWSCredential(name); err != nil {
		return err
	}
	if err := creds.Save(path); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("✓ Credential '%s' deleted.\n", name)
	return nil
}

// EditCredential updates one or more fields of an existing AWS credential.
// Only non-empty values in accessKey, secretKey, and region are applied.
// Returns an error when no fields are provided (nothing to update).
func EditCredential(path, name, accessKey, secretKey, region string) error {
	if accessKey == "" && secretKey == "" && region == "" {
		return fmt.Errorf("nothing to update: provide at least one of --access-key, --secret-key, --region")
	}

	creds, err := LoadCredentials(path)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	existing, err := creds.GetAWSCredential(name)
	if err != nil {
		return err
	}

	updated := *existing
	if accessKey != "" {
		updated.AccessKey = accessKey
	}
	if secretKey != "" {
		updated.SecretKey = secretKey
	}
	if region != "" {
		updated.DefaultRegion = region
	}

	if err := creds.AddAWSCredential(updated); err != nil {
		return err
	}
	if err := creds.Save(path); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("✓ Credential '%s' updated.\n", name)
	return nil
}
