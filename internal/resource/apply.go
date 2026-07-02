package resource

import (
	"fmt"
	"strings"

	"github.com/Felipalds/rancher-saddle/internal/cluster"
	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
)

// ApplyFile reads every YAML document in filePath and applies each resource.
// Multi-document files (---) are fully supported.
func ApplyFile(filePath, configPath, credentialsFile string, registry *core.Registry) error {
	docs, err := parseFile(filePath)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return fmt.Errorf("no resources found in %q", filePath)
	}

	var errs []string
	for _, doc := range docs {
		if err := applyDoc(doc, configPath, credentialsFile, registry); err != nil {
			errs = append(errs, fmt.Sprintf("[%s/%s] %v", doc.kind, doc.name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors during apply:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func applyDoc(doc document, configPath, credentialsFile string, registry *core.Registry) error {
	switch strings.ToLower(doc.kind) {
	case "cluster":
		return applyCluster(doc, configPath, credentialsFile, registry)
	case "credential":
		return applyCredential(doc, credentialsFile)
	case "profile":
		return applyProfile(doc, "profiles.yaml")
	case "ami":
		return applyAMI(doc, "amis.yaml")
	default:
		return fmt.Errorf("unknown kind %q — valid kinds: Cluster, Credential, Profile, AMI", doc.kind)
	}
}

func applyCluster(doc document, configPath, credentialsFile string, registry *core.Registry) error {
	var cr ClusterResource
	if err := doc.node.Decode(&cr); err != nil {
		return fmt.Errorf("failed to decode Cluster spec: %w", err)
	}

	cfg, err := cr.Spec.ToConfig(doc.name, credentialsFile)
	if err != nil {
		return err
	}

	return cluster.CreateClusterNew(doc.name, cfg, registry, configPath)
}

func applyCredential(doc document, credentialsFile string) error {
	var cr CredentialResource
	if err := doc.node.Decode(&cr); err != nil {
		return fmt.Errorf("failed to decode Credential spec: %w", err)
	}

	cred := cr.Spec.ToAWSCredential(doc.name)
	if err := cred.Validate(); err != nil {
		return err
	}

	creds, err := credentials.LoadCredentials(credentialsFile)
	if err != nil {
		return err
	}
	if err := creds.AddAWSCredential(cred); err != nil {
		return err
	}
	if err := creds.Save(credentialsFile); err != nil {
		return err
	}

	fmt.Printf("credential/%s applied\n", doc.name)
	return nil
}

func applyProfile(doc document, profilesFile string) error {
	var pr ProfileResource
	if err := doc.node.Decode(&pr); err != nil {
		return fmt.Errorf("failed to decode Profile spec: %w", err)
	}

	profile := pr.Spec.ToProfile(doc.name)

	profiles, err := config.LoadProfiles(profilesFile)
	if err != nil {
		return err
	}
	profiles.AddProfile(doc.name, profile)
	if err := profiles.Save(profilesFile); err != nil {
		return err
	}

	fmt.Printf("profile/%s applied\n", doc.name)
	return nil
}

func applyAMI(doc document, amisFile string) error {
	var ar AMIResource
	if err := doc.node.Decode(&ar); err != nil {
		return fmt.Errorf("failed to decode AMI spec: %w", err)
	}

	if ar.Spec.Distro == "" {
		return fmt.Errorf("AMI spec.distro is required")
	}
	if ar.Spec.Region == "" {
		return fmt.Errorf("AMI spec.region is required")
	}
	if ar.Spec.AMIID == "" {
		return fmt.Errorf("AMI spec.amiId is required")
	}

	amis, err := config.LoadAMIs(amisFile)
	if err != nil {
		return err
	}
	amis.AddEntry(ar.Spec.ToEntry(doc.name))
	if err := amis.Save(amisFile); err != nil {
		return err
	}

	fmt.Printf("ami/%s applied\n", doc.name)
	return nil
}
