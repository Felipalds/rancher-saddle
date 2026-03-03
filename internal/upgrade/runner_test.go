package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunner_templateData_RKE2(t *testing.T) {
	runner := NewRunner(UpgradeConfig{
		ClusterName:       "test-cluster",
		Distribution:      "rke2",
		InitIP:            "10.0.0.1",
		SSHPrivateKeyPath: "/tmp/key.pem",
		SSHUser:           "ubuntu",
		Hostname:          "rancher.example.com",
		RancherVersion:    "2.11.7",
		BootstrapPassword: "admin",
		Prime:             true,
		Replicas:          3,
		AuditLog:          true,
		AuditLogLevel:     2,
	})

	data := runner.templateData()

	assert.Equal(t, "10.0.0.1", data.InitIP)
	assert.Equal(t, "ubuntu", data.SSHUser)
	assert.Equal(t, "/tmp/key.pem", data.SSHPrivateKeyPath)
	assert.Equal(t, "rancher.example.com", data.Hostname)
	assert.Equal(t, "2.11.7", data.RancherVersion)
	assert.Equal(t, "admin", data.BootstrapPassword)
	assert.Equal(t, true, data.Prime)
	assert.Equal(t, 3, data.Replicas)
	assert.Equal(t, true, data.AuditLog)
	assert.Equal(t, 2, data.AuditLogLevel)
	assert.Equal(t, "/etc/rancher/rke2/rke2.yaml", data.Kubeconfig)
	assert.Equal(t, "/var/lib/rancher/rke2/bin/kubectl", data.Kubectl)
}

func TestRunner_templateData_K3s(t *testing.T) {
	runner := NewRunner(UpgradeConfig{
		ClusterName:       "k3s-cluster",
		Distribution:      "k3s",
		InitIP:            "10.0.0.2",
		SSHPrivateKeyPath: "/tmp/k3s-key.pem",
		SSHUser:           "ec2-user",
		Hostname:          "rancher.k3s.local",
		RancherVersion:    "2.10.0",
		BootstrapPassword: "secret",
		Prime:             false,
		Replicas:          1,
		AuditLog:          false,
		AuditLogLevel:     0,
	})

	data := runner.templateData()

	assert.Equal(t, "10.0.0.2", data.InitIP)
	assert.Equal(t, "ec2-user", data.SSHUser)
	assert.Equal(t, "rancher.k3s.local", data.Hostname)
	assert.Equal(t, "2.10.0", data.RancherVersion)
	assert.Equal(t, false, data.Prime)
	assert.Equal(t, 1, data.Replicas)
	assert.Equal(t, "/etc/rancher/k3s/k3s.yaml", data.Kubeconfig)
	assert.Equal(t, "/usr/local/bin/k3s kubectl", data.Kubectl)
}

func TestRunner_templateData_DefaultDistributionIsRKE2(t *testing.T) {
	// Non-k3s distribution should use RKE2 paths
	runner := NewRunner(UpgradeConfig{
		Distribution: "rke2",
	})

	data := runner.templateData()

	assert.Equal(t, "/etc/rancher/rke2/rke2.yaml", data.Kubeconfig)
	assert.Equal(t, "/var/lib/rancher/rke2/bin/kubectl", data.Kubectl)
}

func TestRunner_templateData_AllFieldsMapped(t *testing.T) {
	cfg := UpgradeConfig{
		ClusterName:       "full-test",
		Distribution:      "rke2",
		InitIP:            "192.168.1.1",
		SSHPrivateKeyPath: "/keys/id_rsa",
		SSHUser:           "admin",
		Hostname:          "rancher.test",
		RancherVersion:    "2.12.0",
		BootstrapPassword: "p@ss",
		Prime:             true,
		Replicas:          5,
		AuditLog:          true,
		AuditLogLevel:     3,
	}

	runner := NewRunner(cfg)
	data := runner.templateData()

	// Verify every UpgradeConfig field maps to templateData
	assert.Equal(t, cfg.InitIP, data.InitIP)
	assert.Equal(t, cfg.SSHUser, data.SSHUser)
	assert.Equal(t, cfg.SSHPrivateKeyPath, data.SSHPrivateKeyPath)
	assert.Equal(t, cfg.Hostname, data.Hostname)
	assert.Equal(t, cfg.RancherVersion, data.RancherVersion)
	assert.Equal(t, cfg.BootstrapPassword, data.BootstrapPassword)
	assert.Equal(t, cfg.Prime, data.Prime)
	assert.Equal(t, cfg.Replicas, data.Replicas)
	assert.Equal(t, cfg.AuditLog, data.AuditLog)
	assert.Equal(t, cfg.AuditLogLevel, data.AuditLogLevel)
}
