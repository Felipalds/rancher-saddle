# Go Kubernetes Helper - Context Documentation

**Version**: 0.5
**Last Updated**: 2026-02-12

This is the **supreme source of truth** for the go-kubernetes-helper project. All major changes and features are documented here.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [How to Use - TUI](#how-to-use---tui)
4. [How to Use - CLI](#how-to-use---cli)
5. [Main Features](#main-features)
6. [Configuration](#configuration)

---

## Project Overview

### Purpose
Automated deployment tool for Kubernetes clusters (RKE2/K3s) with Rancher on AWS EC2. Features include:
- Fullscreen interactive TUI with live log panel
- Multiple Kubernetes distributions (RKE2, K3s)
- Rancher deployment (standard and Prime)
- Multi-cluster management with real infrastructure delete
- Credentials and profiles management
- Infrastructure as Code (OpenTofu/Terraform)
- Configuration management (Ansible)

### Tech Stack
- **Language**: Go 1.24.9
- **TUI**: Bubbletea + Bubbles + Lipgloss
- **CLI**: Cobra
- **IaC**: OpenTofu (Terraform-compatible)
- **Config Management**: Ansible
- **Cloud**: AWS EC2

---

## Architecture

### Modular Design Pattern

The project follows a **Provider/Orchestrator** pattern for extensibility:

```
+-------------------------------------+
|         User Interface              |
|    (Fullscreen TUI / CLI)           |
+-----------------+-------------------+
                  |
                  v
+-------------------------------------+
|      Configuration Layer            |
|  config.yaml      (clusters)        |
|  cloud-credentials.yaml (secrets)   |
|  profiles.yaml    (infra presets)   |
+-----------------+-------------------+
                  |
                  v
+-------------------------------------+
|         Core Registry               |
|  Providers (AWS)                    |
|  Orchestrators (RKE2, K3s)         |
+-----------------+-------------------+
                  |
            +-----+-----+
            v           v
      +---------+  +-----------+
      |Provider |  |Orchestr.  |
      | (AWS)   |  |(RKE2/K3s) |
      +----+----+  +-----+-----+
           |              |
           v              v
      +---------+  +-----------+
      |OpenTofu |  |  Ansible  |
      |Templates|  | Playbooks |
      +---------+  +-----------+
```

### Package Structure

```
github.com/Felipalds/go-kubernetes-helper/
├── main.go                           # Entry point & CLI (Cobra)
├── internal/
│   ├── cluster/                      # CLI cluster commands
│   │   ├── commands.go              # CREATE, LIST, DELETE
│   │   └── state.go                 # Cluster state persistence
│   ├── config/                       # Configuration handling
│   │   ├── clusters.go             # ClusterConfig with RancherSection
│   │   ├── config.go               # Modern Config format
│   │   ├── profiles.go            # Infrastructure profiles
│   │   ├── bridge.go              # Legacy config bridge
│   │   └── validation.go          # Config validation
│   ├── core/                         # Core abstractions
│   │   ├── interfaces.go          # Provider/Orchestrator interfaces
│   │   ├── registry.go            # Component registry
│   │   └── types.go               # Common types
│   ├── credentials/                  # Cloud credential management
│   ├── generator/                    # Template rendering
│   │   └── template.go            # Go template renderer
│   ├── orchestrators/
│   │   ├── k3s/                    # K3s implementation
│   │   │   ├── config.go          # K3sConfig (with Prime/BootstrapPassword)
│   │   │   ├── orchestrator.go    # Playbook & inventory generation
│   │   │   └── templates/         # Ansible templates
│   │   │       ├── init.yml.tmpl
│   │   │       ├── join.yml.tmpl
│   │   │       ├── addons.yml.tmpl  # Rancher install (Prime-aware)
│   │   │       └── playbook.yml.tmpl
│   │   └── rke2/                   # RKE2 implementation (same structure)
│   ├── providers/
│   │   └── aws/
│   │       ├── config.go          # AWSConfig (handles int/float64)
│   │       ├── provider.go        # Terraform generation & outputs
│   │       └── templates/
│   │           └── main.tf.tmpl   # EC2 instances with count
│   ├── tui/                          # Terminal UI
│   │   ├── root.go                # State machine, layout (logs=33%)
│   │   ├── footer.go             # Footer with live log panel
│   │   ├── header.go             # Header bar
│   │   └── views/
│   │       ├── clusterlist.go    # Cluster table, auto-refresh (1s)
│   │       ├── createform.go     # 19-field form + background deploy
│   │       ├── deletemodal.go    # Confirm + tofu destroy + cleanup
│   │       ├── color_helper.go   # statusColor, formatStatus, formatAge
│   │       ├── messages.go       # State/message types
│   │       ├── credentialsform.go
│   │       ├── credentialslist.go
│   │       ├── profilesform.go
│   │       └── profileslist.go
│   └── workflow/                     # Deployment orchestration
│       └── runner.go              # ModularRunner
├── clusters/                         # Per-cluster build dirs (gitignored)
├── logs/                             # Per-cluster log files (gitignored)
├── CONTEXT.md                        # This file
└── CONTEXT/README.md                 # Future planning documents
```

### Data Flow

#### Deployment Flow
```
TUI Create Form (19 fields)
    |
    v
config.yaml saved (ClusterConfig with RancherSection)
    |
    v
ToModernConfig() -> Config struct
    |
    v
Provider.GenerateInfrastructure() -> clusters/<name>/main.tf
    |
    v
tofu init && tofu apply -> EC2 instances (count = InstanceCount)
    |
    v
Provider.GetOutputs() -> IPs, DNS names
    |
    v
Orchestrator.GenerateInventory() -> clusters/<name>/hosts.ini
    [init] = first node (rancher_hostname set)
    [join] = remaining nodes
    |
    v
Orchestrator.GeneratePlaybook() -> clusters/<name>/site.yml
    Play 1: Init node (install RKE2/K3s, create cluster)
    Play 2: Join nodes (join existing cluster)
    Play 3: Addons (cert-manager + Rancher, if DeployRancher=true)
    |
    v
ansible-playbook site.yml -i hosts.ini
    |
    v
Status -> "running", Rancher URL saved
```

#### Delete Flow
```
TUI: press 'd' -> DeleteModalModel confirms
    |
    v
Status -> "deleting" (saved to config.yaml)
    |
    v
go destroyCluster(name) [background goroutine]
    |
    v
tofu destroy -auto-approve (in clusters/<name>/)
    |
    v
os.RemoveAll(clusters/<name>/)
    |
    v
cfg.DeleteCluster(name) -> config.yaml updated
    |
    v
TUI auto-refreshes (cluster disappears from list)
```

---

## How to Use - TUI

### Launching the TUI

```bash
# Fullscreen interactive mode (recommended)
./go-kubernetes-helper
```

### TUI Views

#### Cluster List (Main Screen)

Table columns: Name | Status | Nodes | Provider | Region | Rancher URL | Age

- Rows are colored by status (green=running, blue=creating, red=failed, gray=deleting)
- Auto-refreshes every 1 second
- Rancher URL shown when cluster is running
- Sorted alphabetically by name (stable order)

**Keybindings**:
- `n`/`c`: Create new cluster
- `d`: Delete selected cluster
- `r`: Manual refresh
- `Enter`: Toggle live log viewer (33% of screen)
- `x`: Manage AWS credentials
- `Ctrl+P`: Manage infrastructure profiles
- `?`: Help overlay
- `q`: Quit

#### Create Form (19 fields)

**Navigation**:
- `Tab`/`Down`: Next field
- `Shift+Tab`/`Up`: Previous field
- `Left`/`Right`: Toggle select fields
- `Enter`: Submit form (from any text field)
- `Ctrl+P`: Load saved profile into form
- `Esc`: Cancel

**Fields** (in order):
1. Provider (select: AWS)
2. Credentials (select: saved credential sets)
3. Kubernetes Distribution (select: RKE2, K3s)
4. Cluster Name (text)
5. Node Prefix (text)
6. Region (text)
7. Subnet ID (text)
8. Security Group ID (text)
9. AMI ID (text)
10. Instance Type (text)
11. Instance Count (text)
12. SSH Key Name (text)
13. SSH Private Key Path (text)
14. SSH User (text)
15. K8s Version (text)
16. Deploy Rancher (select: No, Yes)
17. Rancher Prime (select: No, Yes)
18. Rancher Version (text)
19. Bootstrap Password (text)

#### Live Log Panel

- Press `Enter` on a cluster to show its logs
- Occupies bottom 33% of the terminal
- Cluster list shrinks to fill the remaining 67%
- Updates in real-time (1-second refresh)
- Shows tail of `logs/<cluster-name>.log`
- Press `Enter` again to hide

#### Delete Modal

- Press `d` on a cluster to open confirmation
- `y`/`Enter`: Confirm (runs `tofu destroy` in background)
- `n`/`Esc`: Cancel
- Status changes to "deleting" immediately
- Cluster removed from list after infrastructure is destroyed

---

## How to Use - CLI

### Commands

```bash
# Create cluster (launches TUI form)
./go-kubernetes-helper create [cluster-name]

# List all clusters
./go-kubernetes-helper list

# Delete cluster
./go-kubernetes-helper delete <cluster-name> [--force]

# List providers
./go-kubernetes-helper list-providers

# List orchestrators
./go-kubernetes-helper list-orchestrators
```

---

## Main Features

### 1. Rancher Prime Support (v0.5)

**Standard Rancher** (default):
- Helm repo: `rancher-latest` from `https://releases.rancher.com/server-charts/latest`
- Default container images

**Rancher Prime**:
- Helm repo: `rancher-prime` from `https://charts.rancher.com/server-charts/prime`
- Image: `registry.suse.com/rancher/rancher`
- System default registry: `registry.suse.com`

Both use configurable bootstrap password (default: `admin`).

### 2. Multi-Node HA Clusters

- Instance count is correctly passed through the entire pipeline
- First node is `[init]` group (control plane + Rancher host)
- Remaining nodes are `[join]` group (join existing cluster via token)
- Fixed type mismatch bug where `instance_count` was always 1

### 3. Live Log Panel

- 33% of terminal height when active
- Cluster list dynamically shrinks to 67%
- Reads from `logs/<cluster-name>.log`
- Auto-refreshes every 1 second
- Shows both deployment and deletion logs

### 4. Real Infrastructure Delete

- TUI delete runs `tofu destroy -auto-approve`
- Removes build directory and config entry
- Runs in background goroutine (non-blocking TUI)
- Logs to `logs/<cluster-name>.log`
- Falls back to "failed" status if destroy fails

### 5. Credentials & Profiles

- **Credentials**: Save multiple AWS access key/secret key pairs
- **Profiles**: Save region, subnet, SG, AMI, instance type, SSH settings
- Load profiles into create form with `Ctrl+P`
- Stored in `cloud-credentials.yaml` and `profiles.yaml`

### 6. Auto-Refresh

- Cluster list reloads from `config.yaml` every 1 second
- Detects status changes (creating -> running, deleting -> removed)
- Log panel updates with new log lines
- Cluster names sorted alphabetically for stable row order

### 7. Multi-Distribution Support

- **RKE2**: Production-grade, FIPS-compliant
- **K3s**: Lightweight, fast deployment
- Each has independent config, templates, and version defaults
- Cert-manager v1.17.2 for both

---

## Configuration

### Config Files

| File | Format | Purpose |
|------|--------|---------|
| `config.yaml` | YAML | All cluster configurations (status, provider, k8s, rancher, ssh, etc.) |
| `cloud-credentials.yaml` | YAML | Saved AWS credential sets |
| `profiles.yaml` | YAML | Saved infrastructure profiles |

### RancherSection Schema

```yaml
rancher:
  version: "2.11.7"
  deploy: true
  prime: false
  bootstrap_password: "admin"
```

### ClusterConfig Schema

```yaml
clusters:
  my-cluster:
    provider:
      type: aws
      config:
        region: us-east-1
        instance_type: t3.xlarge
        subnet_id: subnet-xxxxx
        security_group_id: sg-xxxxx
        ami: ami-xxxxx
        access_key: AKIA...
        secret_key: ...
    kubernetes:
      distribution: rke2
      config:
        version: v1.33.7+rke2r1
    rancher:
      version: "2.11.7"
      deploy: true
      prime: false
      bootstrap_password: admin
    ssh:
      key_name: my-key
      private_key_path: ~/.ssh/my-key.pem
      user: ubuntu
    cluster:
      node_prefix: k8s-node
      instance_count: 3
    status: running
    build_dir: clusters/my-cluster
    instance_ips: [54.x.x.x, 54.x.x.x, 54.x.x.x]
    instance_dns: [ec2-xx.compute.amazonaws.com, ...]
    rancher_url: https://ec2-xx.compute.amazonaws.com
    created_at: 2026-02-12T10:00:00Z
    updated_at: 2026-02-12T10:15:00Z
```

### AWS Prerequisites

**Required**:
1. AWS account with programmatic access
2. IAM user with EC2 permissions
3. Existing VPC with:
   - Public subnet
   - Security group (ports: 22, 80, 443, 6443, 9345, 10250)
4. EC2 key pair (`.pem` file)

### Local Prerequisites

**Required Tools**:
- `tofu` (OpenTofu) or `terraform` in PATH
- `ansible-playbook` in PATH
- SSH client
- Go 1.24+ (for building)

---

## Version History

### v0.5 (2026-02-12)
- Added Rancher Prime support (SUSE registry, prime helm chart)
- Added Deploy Rancher toggle, Rancher Version, Bootstrap Password form fields
- Fixed multi-node cluster creation (instance_count type mismatch bug)
- Implemented real infrastructure delete (tofu destroy + cleanup)
- Added live log panel (33% of screen, 1-second refresh)
- Added Rancher URL column to cluster list
- Auto-refresh cluster list every 1 second
- Fullscreen TUI layout (uses entire terminal height)
- Dynamic column sizing for cluster table
- Sorted cluster list (stable row order)
- Bumped cert-manager to v1.17.2
- Credentials and profiles management views
- Extracted color/status helpers to color_helper.go

### v0.4 (2026-02-10)
- Added K3s orchestrator support
- TUI distribution selector (RKE2/K3s)
- Migrated from JSON to YAML configuration
- Fixed Jinja2 template escaping in Ansible playbooks
- Added path expansion for SSH keys
- Fullscreen TUI refactor with state machine

### v0.3 (2026-02-09)
- Multi-cluster management (create, list, delete)
- Cluster state tracking
- Isolated build directories per cluster
- Enhanced error reporting
- SSH readiness checks
- Rancher DNS name support

### v0.1 (Initial Release)
- Interactive TUI configuration
- RKE2 cluster deployment
- AWS EC2 provisioning with OpenTofu
- Ansible playbook generation
- Rancher installation

---

**End of Context Documentation**
