package cluster

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/Felipalds/rancher-saddle/internal/config"
	"github.com/Felipalds/rancher-saddle/internal/upgrade"
)

// UpgradeOptions holds parameters for upgrading Rancher on a cluster.
// Zero values fall back to the current cluster configuration.
type UpgradeOptions struct {
	Version       string
	Replicas      int
	AuditLog      bool
	AuditLogLevel int
	ImageTag      string
	Debug         bool
}

// UpgradeCluster upgrades Rancher on the named cluster. It blocks until the
// operation completes and returns any error. Progress is written to out; pass
// nil to write only to the log file at logs/<name>-upgrade.log.
// configPath is the path to config.yaml (the clusters config file).
func UpgradeCluster(clusterName string, opts UpgradeOptions, configPath string, out io.Writer) error {
	cfg, err := config.LoadClustersConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cluster, exists := cfg.GetCluster(clusterName)
	if !exists {
		return fmt.Errorf("cluster %q not found", clusterName)
	}

	cluster.Status = "upgrading"
	cfg.AddCluster(clusterName, cluster)
	cfg.Save(configPath)

	targetVersion := opts.Version
	if targetVersion == "" {
		targetVersion = cluster.Rancher.Version
	}
	replicas := opts.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	auditLogLevel := opts.AuditLogLevel
	if auditLogLevel <= 0 {
		auditLogLevel = 1
	}

	os.MkdirAll("logs", 0755)
	logPath := fmt.Sprintf("logs/%s-upgrade.log", clusterName)
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	var w io.Writer = logFile
	if out != nil && logFile != nil {
		w = io.MultiWriter(logFile, out)
	} else if out != nil {
		w = out
	}

	writeLog := func(msg string) {
		if w != nil {
			fmt.Fprintf(w, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
		}
	}

	hostname := ""
	if len(cluster.InstanceDNS) > 0 {
		hostname = cluster.InstanceDNS[0]
	} else if len(cluster.InstanceIPs) > 0 {
		hostname = cluster.InstanceIPs[0]
	}
	initIP := ""
	if len(cluster.InstanceIPs) > 0 {
		initIP = cluster.InstanceIPs[0]
	}

	writeLog(fmt.Sprintf("=== Starting Rancher upgrade for cluster: %s ===", clusterName))
	writeLog(fmt.Sprintf("Target version: %s, Prime: %v", targetVersion, cluster.Rancher.Prime))

	var upgradeErr error
	if cluster.Kubernetes.Distribution == "docker" {
		upgradeErr = execDockerUpgrade(
			initIP, cluster.SSH.PrivateKeyPath, cluster.SSH.User,
			targetVersion, cluster.Rancher.Prime, cluster.Rancher.BootstrapPassword,
			opts.ImageTag, opts.Debug, w,
		)
	} else {
		runner := upgrade.NewRunner(upgrade.UpgradeConfig{
			ClusterName:       clusterName,
			Distribution:      cluster.Kubernetes.Distribution,
			InitIP:            initIP,
			SSHPrivateKeyPath: cluster.SSH.PrivateKeyPath,
			SSHUser:           cluster.SSH.User,
			Hostname:          hostname,
			RancherVersion:    targetVersion,
			BootstrapPassword: cluster.Rancher.BootstrapPassword,
			Prime:             cluster.Rancher.Prime,
			Replicas:          replicas,
			AuditLog:          opts.AuditLog,
			AuditLogLevel:     auditLogLevel,
			ImageTag:          opts.ImageTag,
			Debug:             opts.Debug,
		})
		upgradeErr = runner.Run(w)
	}

	if upgradeErr != nil {
		writeLog(fmt.Sprintf("ERROR: Upgrade failed: %v", upgradeErr))
		cluster.Status = "upgrade-failed"
		cfg.AddCluster(clusterName, cluster)
		cfg.Save(configPath)
		return upgradeErr
	}

	writeLog("Upgrade completed successfully!")
	cluster.Status = "running"
	cluster.Rancher.Version = targetVersion
	cluster.Rancher.AuditLog = opts.AuditLog
	cluster.Rancher.AuditLogLevel = auditLogLevel
	cluster.Rancher.ImageTag = opts.ImageTag
	cluster.Rancher.Debug = opts.Debug
	cfg.AddCluster(clusterName, cluster)
	cfg.Save(configPath)

	return nil
}

// execDockerUpgrade SSHs into the remote host and runs docker stop/rm/run to
// replace the running Rancher container with a new image.
func execDockerUpgrade(initIP, sshKeyPath, sshUser, targetVersion string, prime bool, bootstrapPassword, imageTag string, debug bool, w io.Writer) error {
	image := "rancher/rancher"
	if prime {
		image = "registry.suse.com/rancher/rancher"
	}
	tag := "v" + targetVersion
	if imageTag != "" {
		tag = imageTag
	}
	fullImage := image + ":" + tag

	dockerCmds := fmt.Sprintf(
		"sudo docker stop rancher && sudo docker rm rancher && sudo docker run -d --name rancher --restart=unless-stopped --privileged -p 80:80 -p 443:443 -v rancher-data:/var/lib/rancher -e CATTLE_BOOTSTRAP_PASSWORD=%s",
		bootstrapPassword,
	)
	if debug {
		dockerCmds += " -e CATTLE_DEBUG=true"
	}
	if prime {
		dockerCmds += " -e RANCHER_VERSION_TYPE=prime -e CATTLE_BASE_UI_BRAND=suse"
	}
	dockerCmds += " " + fullImage

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-i", sshKeyPath,
		fmt.Sprintf("%s@%s", sshUser, initIP),
		dockerCmds,
	}

	cmd := exec.Command("ssh", sshArgs...)
	if w != nil {
		cmd.Stdout = w
		cmd.Stderr = w
	}

	return cmd.Run()
}
