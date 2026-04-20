package docker

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// CheckPrerequisites verifies that Docker is available and running.
func CheckPrerequisites() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH: please install Docker first")
	}

	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker daemon is not running: %s", string(out))
	}

	return nil
}

// BuildRunArgs constructs the argument list for `docker run` from the config.
func BuildRunArgs(cfg DeployConfig) []string {
	hostPort := cfg.HostPort
	if hostPort == "" {
		hostPort = "443"
	}

	// Compute the HTTP port: if HTTPS is 443 use 80, otherwise use hostPort-1
	httpPort := "80"
	if hostPort != "443" {
		httpPort = hostPort // For custom ports, only expose HTTPS
	}

	args := []string{
		"run", "-d",
		"--name", cfg.ContainerName(),
		"--restart=unless-stopped",
		"--privileged",
		"-p", httpPort + ":80",
		"-p", hostPort + ":443",
		"-v", cfg.VolumeName() + ":/var/lib/rancher",
		"-e", "CATTLE_BOOTSTRAP_PASSWORD=" + cfg.BootstrapPassword,
	}

	if cfg.Debug {
		args = append(args, "-e", "CATTLE_DEBUG=true")
	}

	if cfg.Prime {
		args = append(args, "-e", "RANCHER_VERSION_TYPE=prime")
		args = append(args, "-e", "CATTLE_BASE_UI_BRAND=suse")
	}

	args = append(args, cfg.Image())
	return args
}

// DeployRancher runs a Rancher container using Docker.
func DeployRancher(ctx context.Context, cfg DeployConfig, logWriter io.Writer) error {
	args := BuildRunArgs(cfg)

	fmt.Fprintf(logWriter, "Running: docker %v\n", args)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %w", err)
	}

	fmt.Fprintf(logWriter, "Rancher container '%s' started successfully\n", cfg.ContainerName())
	return nil
}

// DeleteRancher stops and removes the Rancher container and its volume.
func DeleteRancher(ctx context.Context, clusterName string, logWriter io.Writer) error {
	containerName := "rancher-" + clusterName
	volumeName := "rancher-data-" + clusterName

	// Stop container (ignore error — may already be stopped)
	fmt.Fprintf(logWriter, "Stopping container %s...\n", containerName)
	stop := exec.CommandContext(ctx, "docker", "stop", containerName)
	stop.Stdout = logWriter
	stop.Stderr = logWriter
	stop.Run() //nolint:errcheck

	// Remove container
	fmt.Fprintf(logWriter, "Removing container %s...\n", containerName)
	rm := exec.CommandContext(ctx, "docker", "rm", containerName)
	rm.Stdout = logWriter
	rm.Stderr = logWriter
	rm.Run() //nolint:errcheck

	// Remove volume
	fmt.Fprintf(logWriter, "Removing volume %s...\n", volumeName)
	rmv := exec.CommandContext(ctx, "docker", "volume", "rm", volumeName)
	rmv.Stdout = logWriter
	rmv.Stderr = logWriter
	rmv.Run() //nolint:errcheck

	fmt.Fprintf(logWriter, "Docker cleanup completed for '%s'\n", clusterName)
	return nil
}

// UpgradeRancher stops the old container, removes it, and runs a new one with the same volume.
func UpgradeRancher(ctx context.Context, cfg DeployConfig, logWriter io.Writer) error {
	containerName := cfg.ContainerName()

	// Stop old container
	fmt.Fprintf(logWriter, "Stopping container %s...\n", containerName)
	stop := exec.CommandContext(ctx, "docker", "stop", containerName)
	stop.Stdout = logWriter
	stop.Stderr = logWriter
	if err := stop.Run(); err != nil {
		fmt.Fprintf(logWriter, "Warning: stop failed (container may not be running): %v\n", err)
	}

	// Remove old container
	fmt.Fprintf(logWriter, "Removing container %s...\n", containerName)
	rm := exec.CommandContext(ctx, "docker", "rm", containerName)
	rm.Stdout = logWriter
	rm.Stderr = logWriter
	if err := rm.Run(); err != nil {
		fmt.Fprintf(logWriter, "Warning: rm failed: %v\n", err)
	}

	// Run new container with same volume
	fmt.Fprintf(logWriter, "Starting new container with image %s...\n", cfg.Image())
	return DeployRancher(ctx, cfg, logWriter)
}
