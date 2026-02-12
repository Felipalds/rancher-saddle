# Go Kubernetes Helper

Automated deployment tool for Kubernetes clusters (RKE2/K3s) with Rancher on AWS EC2 using OpenTofu and Ansible, featuring an interactive fullscreen TUI.

## Features

- **Interactive Fullscreen TUI** - Bubbletea-based terminal interface with live log panel
- **AWS EC2 Deployment** - Automated multi-node instance provisioning with OpenTofu
- **RKE2 & K3s Support** - Choose between production-grade RKE2 or lightweight K3s
- **Rancher Management** - Deploy Rancher (standard or Prime) with configurable bootstrap password
- **Multi-Cluster Management** - Create, monitor, and delete clusters independently
- **Live Deployment Logs** - Real-time log viewer occupying 33% of the screen
- **Credentials & Profiles** - Save and reuse AWS credentials and infrastructure profiles
- **Infrastructure Cleanup** - Delete clusters with full `tofu destroy` and config removal

## Prerequisites

### Required Tools
- [OpenTofu](https://opentofu.org/) (or Terraform) - `tofu` command in PATH
- [Ansible](https://www.ansible.com/) - `ansible-playbook` command in PATH
- Go 1.24.9+
- AWS Account with appropriate permissions

### AWS Requirements
- Valid AWS access key and secret key
- Existing VPC with subnet
- Security group with required ports:
  - SSH (22)
  - HTTP/HTTPS (80, 443)
  - Kubernetes API (6443)
  - RKE2 supervisor (9345) / K3s (10250)
- EC2 key pair for SSH access

## Installation

```bash
git clone https://github.com/Felipalds/go-kubernetes-helper.git
cd go-kubernetes-helper
go build -o go-kubernetes-helper
```

## Usage

### Fullscreen TUI (Recommended)

```bash
./go-kubernetes-helper
```

The TUI provides a fullscreen experience with:

- **Cluster List** - Table showing all clusters with name, status, nodes, provider, region, Rancher URL, and age
- **Create Form** - 19-field form for full cluster configuration
- **Live Logs** - Press Enter on a cluster to show real-time deployment logs (33% of screen)
- **Delete** - Press `d` to destroy infrastructure and remove cluster
- **Auto-refresh** - Cluster status and logs update every second

#### TUI Keybindings

| Key | Action |
|-----|--------|
| `n` / `c` | Create new cluster |
| `d` | Delete selected cluster |
| `r` | Manual refresh |
| `Enter` | Toggle log viewer |
| `x` | Manage credentials |
| `Ctrl+P` | Manage profiles |
| `?` | Help |
| `q` / `Ctrl+C` | Quit |

#### Create Form Fields

| Field | Default | Description |
|-------|---------|-------------|
| Provider | AWS | Cloud provider |
| Credentials | - | Saved AWS credential set |
| K8s Distribution | RKE2 | RKE2 or K3s |
| Cluster Name | my-cluster | Unique cluster identifier |
| Node Prefix | k8s-node | EC2 instance name prefix |
| Region | us-east-1 | AWS region |
| Subnet ID | - | VPC subnet |
| Security Group ID | - | Security group |
| AMI ID | - | Ubuntu 22.04 recommended |
| Instance Type | t3.xlarge | EC2 instance type |
| Instance Count | 3 | Number of nodes (HA) |
| SSH Key Name | - | AWS key pair name |
| SSH Private Key Path | - | Path to .pem file |
| SSH User | ubuntu | SSH username |
| K8s Version | v1.33.7+rke2r1 | Distribution version |
| Deploy Rancher | No | Enable Rancher deployment |
| Rancher Prime | No | Use Rancher Prime (SUSE registry) |
| Rancher Version | 2.11.7 | Rancher chart version |
| Bootstrap Password | admin | Rancher initial admin password |

### CLI Commands

```bash
# Create cluster (opens TUI form)
./go-kubernetes-helper create my-cluster

# List all clusters
./go-kubernetes-helper list

# Delete cluster (with confirmation)
./go-kubernetes-helper delete my-cluster

# Delete without confirmation
./go-kubernetes-helper delete my-cluster --force

# List available providers
./go-kubernetes-helper list-providers

# List available orchestrators
./go-kubernetes-helper list-orchestrators
```

## How It Works

### Deployment Flow

1. User configures cluster via TUI form
2. Configuration saved to `config.yaml`
3. OpenTofu provisions EC2 instances (`clusters/<name>/main.tf`)
4. SSH readiness check on all instances
5. Ansible inventory generated (`clusters/<name>/hosts.ini`)
   - First node in `[init]` group (control plane + Rancher host)
   - Remaining nodes in `[join]` group (HA members)
6. Ansible playbook generated and executed (`clusters/<name>/site.yml`)
   - Installs RKE2/K3s on init node
   - Joins remaining nodes to cluster
   - Deploys cert-manager (v1.17.2) and Rancher (if enabled)
7. Cluster status updates to `running`, Rancher URL shown in table

### Delete Flow

1. User presses `d` and confirms
2. Status set to `deleting` (visible in TUI immediately)
3. Background goroutine runs `tofu destroy -auto-approve`
4. Build directory removed (`clusters/<name>/`)
5. Cluster entry removed from `config.yaml`
6. TUI auto-refreshes to reflect changes

### Rancher Prime vs Standard

When **Rancher Prime** is enabled:
- Helm repo: `https://charts.rancher.com/server-charts/prime`
- Container image: `registry.suse.com/rancher/rancher`
- System default registry: `registry.suse.com`

When **standard Rancher** (default):
- Helm repo: `https://releases.rancher.com/server-charts/latest`
- Default upstream container images

## Configuration Files

| File | Purpose |
|------|---------|
| `config.yaml` | Cluster configurations (auto-generated, contains secrets - 0600) |
| `cloud-credentials.yaml` | Saved AWS credentials (0600) |
| `profiles.yaml` | Saved infrastructure profiles |
| `clusters/<name>/` | Per-cluster build directory (Terraform state, playbooks) |
| `logs/<name>.log` | Per-cluster deployment/deletion logs |

## Project Structure

```
go-kubernetes-helper/
├── main.go                           # Entry point & CLI commands
├── internal/
│   ├── config/                       # Configuration management
│   │   ├── clusters.go              # ClusterConfig, RancherSection, load/save
│   │   ├── config.go                # Modern Config format
│   │   └── profiles.go             # Infrastructure profiles
│   ├── core/                         # Interfaces & registry
│   │   ├── interfaces.go           # Provider/Orchestrator interfaces
│   │   └── registry.go             # Component registry
│   ├── credentials/                  # Cloud credential management
│   ├── generator/                    # Template rendering
│   ├── orchestrators/
│   │   ├── rke2/                    # RKE2 orchestrator + Ansible templates
│   │   └── k3s/                     # K3s orchestrator + Ansible templates
│   ├── providers/
│   │   └── aws/                     # AWS provider + Terraform templates
│   ├── tui/                          # Terminal UI
│   │   ├── root.go                  # State machine & layout
│   │   ├── footer.go               # Footer with log panel
│   │   └── views/
│   │       ├── clusterlist.go      # Cluster table with auto-refresh
│   │       ├── createform.go       # 19-field creation form
│   │       ├── deletemodal.go      # Delete confirmation + tofu destroy
│   │       ├── color_helper.go     # Status colors & formatting
│   │       ├── credentialsform.go  # Credentials CRUD
│   │       ├── credentialslist.go  # Credentials list
│   │       ├── profilesform.go    # Profiles CRUD
│   │       └── profileslist.go    # Profiles list
│   └── workflow/                     # Deployment orchestration
├── clusters/                         # Per-cluster build dirs (gitignored)
├── logs/                             # Deployment logs (gitignored)
├── CONTEXT.md                        # Detailed project documentation
└── CONTEXT/README.md                 # Future planning documents
```

## Troubleshooting

### OpenTofu Errors
- Verify AWS credentials are valid
- Check subnet and security group exist in the specified region
- Ensure IAM permissions include EC2 full access

### Ansible Errors
- Verify SSH key path is correct and has 0600 permissions
- Check security group allows SSH (port 22)
- Check logs: `logs/<cluster-name>.log`

### Only 1 Node Created
- Ensure Instance Count field is set (default: 3)
- This was a known bug (type mismatch in config passing) - fixed in v0.5

### Rancher Access
- Use the DNS name shown in the Rancher URL column
- Wait 5-10 minutes after deployment for Rancher to initialize
- Default login: admin / (your bootstrap password)

## Security Considerations

- `config.yaml` and `cloud-credentials.yaml` contain secrets - never commit to git
- All credential files use 0600 permissions
- SSH private keys should have 0600 permissions
- Use restrictive security groups in production

## Author

Felipalds @ SUSE (luiz.rosa@suse.com)

## Acknowledgments

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [OpenTofu](https://opentofu.org/) - Infrastructure as Code
- [Ansible](https://www.ansible.com/) - Configuration management
- [Rancher](https://rancher.com/) - Kubernetes management
