package cluster

import (
	"fmt"
	"strings"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/core"
	"github.com/Felipalds/rancher-saddle/internal/credentials"
)

// ApplyFile reads a ClustersConfig YAML blueprint and provisions each cluster
// defined in it. If clusterNames is non-empty only those clusters are applied;
// otherwise every cluster in the file is applied.
//
// If credentialName is non-empty, the named credential from credentialsFile
// overrides the access_key / secret_key embedded in the blueprint.
func ApplyFile(
	filePath string,
	clusterNames []string,
	credentialName string,
	credentialsFile string,
	configPath string,
	registry *core.Registry,
) error {
	blueprint, err := config.LoadClustersConfig(filePath)
	if err != nil {
		return fmt.Errorf("failed to load %q: %w", filePath, err)
	}

	if len(blueprint.Clusters) == 0 {
		return fmt.Errorf("no clusters found in %q", filePath)
	}

	// Determine target set.
	targets := clusterNames
	if len(targets) == 0 {
		targets = blueprint.ListClusters()
	}

	// Resolve credential override once, before iterating.
	var overrideAccessKey, overrideSecretKey string
	if credentialName != "" {
		creds, err := credentials.LoadCredentials(credentialsFile)
		if err != nil {
			return fmt.Errorf("failed to load credentials file %q: %w", credentialsFile, err)
		}
		cred, err := creds.GetAWSCredential(credentialName)
		if err != nil {
			return fmt.Errorf("credential %q not found: %w", credentialName, err)
		}
		overrideAccessKey = cred.AccessKey
		overrideSecretKey = cred.SecretKey
	}

	var errs []string
	for _, name := range targets {
		cc, exists := blueprint.GetCluster(name)
		if !exists {
			errs = append(errs, fmt.Sprintf("cluster %q not found in %s", name, filePath))
			continue
		}

		cfg := buildConfigFromCluster(name, cc, overrideAccessKey, overrideSecretKey)

		fmt.Printf("Applying cluster %q...\n", name)
		if err := CreateClusterNew(name, cfg, registry, configPath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during apply:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// buildConfigFromCluster converts a ClusterConfig (YAML format) into the
// workflow-ready Config. If accessKey / secretKey are non-empty they override
// whatever credentials are embedded in cc.Provider.Config.
func buildConfigFromCluster(name string, cc *config.ClusterConfig, accessKey, secretKey string) *config.Config {
	cfg := cc.ToModernConfig()
	cfg.ClusterName = name

	if accessKey != "" {
		cfg.ProviderConfig["access_key"] = accessKey
		cfg.ProviderConfig["secret_key"] = secretKey
	}

	return cfg
}
