# Go Kubernetes Helper - Context Documentation

**Version**: 2.0
**Last Updated**: 2026-02-10

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
- Interactive TUI for configuration
- Multiple Kubernetes distributions (RKE2, K3s)
- Multi-cluster management
- Infrastructure as Code (OpenTofu/Terraform)
- Configuration management (Ansible)

### Tech Stack
- **Language**: Go 1.24.9
- **TUI**: Bubbletea + Bubbles + Lipgloss
- **CLI**: Cobra
- **IaC**: OpenTofu (Terraform-compatible)
- **Config Management**: Ansible
- **Cloud**: AWS EC2
- **Logging**: Zap

---

## Architecture

### Modular Design Pattern

The project follows a **Provider/Orchestrator** pattern for extensibility:

```
┌─────────────────────────────────────┐
│         User Interface              │
│    (TUI / CLI Commands)             │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│      Configuration Layer            │
│  • config.yaml (user settings)      │
│  • clusters.yaml (cluster tracking) │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│         Core Registry               │
│  • Providers (AWS, Azure, GCP)      │
│  • Orchestrators (RKE2, K3s)        │
└────────────┬────────────────────────┘
             │
        ┌────┴────┐
        ▼         ▼
┌──────────┐  ┌──────────┐
│ Provider │  │Orchestr. │
│  (AWS)   │  │(RKE2/K3s)│
└────┬─────┘  └────┬─────┘
     │             │
     ▼             ▼
┌─────────┐   ┌─────────┐
│OpenTofu │   │ Ansible │
│Templates│   │Playbooks│
└─────────┘   └─────────┘
```

### Package Structure

```
github.com/Felipalds/go-kubernetes-helper/
├── cmd/                           # Command implementations
│   └── tui.go                    # TUI command
├── internal/                      # Private application code
│   ├── cluster/                  # Cluster management
│   │   ├── commands.go          # CREATE, LIST, DELETE
│   │   └── state.go             # Cluster state persistence
│   ├── config/                   # Configuration handling
│   │   ├── bridge.go            # Legacy ↔ New config bridge
│   │   ├── config.go            # New config format
│   │   └── validation.go        # Config validation with path expansion
│   ├── core/                     # Core abstractions
│   │   ├── interfaces.go        # Provider/Orchestrator interfaces
│   │   ├── registry.go          # Component registry
│   │   └── types.go             # Common types
│   ├── generator/                # Template rendering
│   │   └── template.go          # Go template renderer
│   ├── model/                    # Legacy data model
│   │   └── config.go            # Config struct (YAML format)
│   ├── orchestrators/            # Kubernetes distributions
│   │   ├── k3s/                 # K3s implementation
│   │   │   ├── config.go
│   │   │   ├── orchestrator.go
│   │   │   └── templates/       # Ansible templates
│   │   └── rke2/                # RKE2 implementation
│   │       ├── config.go
│   │       ├── orchestrator.go
│   │       └── templates/
│   ├── providers/                # Cloud providers
│   │   └── aws/                 # AWS implementation
│   │       ├── config.go
│   │       ├── provider.go
│   │       └── templates/       # Terraform templates
│   ├── tui/                      # Terminal UI
│   │   ├── form.go              # Configuration form
│   │   └── menu.go              # Main menu
│   ├── utils/                    # Utilities
│   │   └── logger.go            # Logging setup
│   └── workflow/                 # Deployment orchestration
│       └── runner.go            # Workflow execution
├── config.yaml                   # User configuration (YAML)
├── main.go                       # Application entry point
└── CONTEXT.md                    # This file
```

### Data Flow

#### Configuration Flow
```
config.yaml (disk)
    ↓
LoadConfig()
    ↓
Config struct (memory)
    ↓
TUI Form (edit)
    ↓
Save() → config.yaml (updated)
```

#### Cluster Tracking Flow
```
~/.go-kubernetes-helper/clusters.yaml
    ↓
ClusterStore.List()
    ↓
Display clusters with status
    ↓
CREATE/DELETE operations
    ↓
Update clusters.yaml
```

#### Deployment Flow
```
TUI Submit
    ↓
config.yaml saved
    ↓
Provider.GenerateInfrastructure() → main.tf
    ↓
tofu init && tofu apply → EC2 instances
    ↓
Provider.GetOutputs() → IPs, DNS names
    ↓
Orchestrator.GenerateInventory() → hosts.ini
    ↓
Orchestrator.GeneratePlaybook() → site.yml
    ↓
ansible-playbook → Cluster deployed
```

---

## How to Use - TUI

### Launching the TUI

```bash
# Interactive menu
./go-kubernetes-helper

# Direct cluster creation
./go-kubernetes-helper create my-cluster
```

### TUI Navigation

#### Main Menu
- **↑/↓ or j/k**: Navigate options
- **Enter**: Select option
- **Esc/Ctrl+C**: Exit

Menu options:
1. **Create New Cluster**: Launch configuration form
2. **List Clusters**: Show all managed clusters
3. **Delete Cluster**: Remove a cluster
4. **Exit**: Quit application

#### Configuration Form (14 fields)

**Navigation**:
- **Tab** / **Enter**: Next field
- **Shift+Tab**: Previous field
- **↑/↓**: Navigate fields
- **Ctrl+C** / **Esc**: Cancel

**Fields** (in order):
1. AWS Access Key
2. AWS Secret Key (password masked)
3. AWS Region (e.g., `us-west-2`)
4. Subnet ID (e.g., `subnet-xxxxx`)
5. Security Group ID (e.g., `sg-xxxxx`)
6. SSH Key Name (e.g., `my-key-pair`)
7. SSH Private Key Path (e.g., `~/.ssh/id_rsa`)
8. Node Prefix (e.g., `rancher-node`)
9. AMI ID (e.g., `ami-xxxxx`)
10. Instance Count (integer)
11. Root Volume Size (GB)
12. **Kubernetes Distribution** (↑/↓ or j/k to select)
    - **RKE2** (production-grade)
    - **K3s** (lightweight)
13. **Distribution Version** (auto-labeled based on selection)
    - Examples: `v1.33.7+rke2r1` or `v1.30.3+k3s1`
14. Rancher Version (e.g., `2.10.2`)

**Special Field: Kubernetes Distribution**
- Use **j** (down) or **k** (up) to toggle between RKE2 and K3s
- Press **Tab** or **Enter** to move to version field
- Version field label updates dynamically

**Submit**: Navigate to submit button and press **Enter**

### TUI Features

- **Auto-save**: Configuration persists to `config.yaml`
- **Smart defaults**: Pre-filled values from previous runs
- **Validation**: Real-time input validation
- **Password masking**: AWS secret key hidden
- **Path expansion**: `~` expands to home directory

---

## How to Use - CLI

### Commands Overview

```bash
./go-kubernetes-helper <command> [flags]
```

### Available Commands

#### 1. Create Cluster

```bash
# With TUI configuration
./go-kubernetes-helper create [cluster-name]

# With custom config file
./go-kubernetes-helper create my-cluster --config custom.yaml

# With name flag
./go-kubernetes-helper create -n production
```

**Workflow**:
1. Launches TUI for configuration
2. Prompts for cluster name (if not provided)
3. Saves configuration to `config.yaml`
4. Creates cluster entry (status: creating)
5. Provisions AWS infrastructure (EC2 instances)
6. Deploys Kubernetes (RKE2/K3s)
7. Installs Rancher
8. Updates cluster status (running/failed)

**Output**:
```
Launching configuration form...
Enter cluster name: production

Creating cluster 'production'...
Generating infrastructure code...
Running OpenTofu...
✓ Created 3 EC2 instances

Waiting for SSH availability...
✓ All instances ready

Deploying Kubernetes (RKE2)...
✓ Cluster initialized
✓ Nodes joined
✓ Rancher deployed

✓ Cluster 'production' created successfully!
Rancher URL: https://ec2-xx-xx-xx-xx.compute.amazonaws.com/dashboard
```

#### 2. List Clusters

```bash
./go-kubernetes-helper list
```

**Output**:
```
Registered Clusters:
NAME          STATUS     NODES   REGION      CREATED    RANCHER URL
production    running    3       us-west-2   2h ago     https://ec2-...
staging       running    1       us-east-1   5d ago     https://ec2-...
dev           failed     -       us-west-1   1d ago     -
```

**Statuses**:
- `creating`: Deployment in progress
- `running`: Cluster operational
- `failed`: Deployment failed
- `deleting`: Cleanup in progress

#### 3. Delete Cluster

```bash
# With confirmation prompt
./go-kubernetes-helper delete <cluster-name>

# Skip confirmation
./go-kubernetes-helper delete production --force
```

**Workflow**:
1. Verifies cluster exists
2. Prompts for confirmation (unless `--force`)
3. Updates status to `deleting`
4. Runs `tofu destroy` (removes AWS resources)
5. Removes build directory
6. Removes cluster from registry

**Output**:
```
Are you sure you want to delete cluster 'staging'? (yes/no): yes
Deleting cluster 'staging'...
Destroying AWS infrastructure...
✓ Infrastructure destroyed
✓ Build directory removed
✓ Cluster 'staging' deleted successfully!
```

#### 4. List Providers

```bash
./go-kubernetes-helper list-providers
```

**Output**:
```
Registered Providers:
  - aws
```

#### 5. List Orchestrators

```bash
./go-kubernetes-helper list-orchestrators
```

**Output**:
```
Registered Orchestrators:
  - rke2
  - k3s
```

### Global Flags

- `--config <path>`: Custom config file path (default: `config.yaml`)

---

## Main Features

### 1. Multi-Distribution Support

**Kubernetes Distributions**:
- **RKE2**: Production-grade, FIPS-compliant, optimized for security
- **K3s**: Lightweight, fast deployment, IoT/Edge-friendly

**Selection**: Choose distribution in TUI (field 12) using j/k keys

**Version Management**: Each distribution has independent versioning
- RKE2 example: `v1.33.7+rke2r1`
- K3s example: `v1.30.3+k3s1`

### 2. Multi-Cluster Management

**Cluster Tracking**: All clusters stored in `~/.go-kubernetes-helper/clusters.yaml`

**Isolation**: Each cluster has its own:
- Build directory (`clusters/<name>/`)
- Infrastructure state (`terraform.tfstate`)
- Ansible inventory and playbooks

**Operations**:
- Create multiple clusters with different configurations
- List all clusters with status
- Delete clusters independently

### 3. Modular Architecture

**Provider System**:
- Interface-based design
- Currently supports: AWS
- Extensible to: Azure, GCP, vSphere

**Orchestrator System**:
- Interface-based design
- Currently supports: RKE2, K3s
- Extensible to: Kubeadm, Minikube

**Registry Pattern**:
```go
core.GlobalRegistry.RegisterProvider(aws.NewProvider())
core.GlobalRegistry.RegisterOrchestrator(rke2.NewOrchestrator())
core.GlobalRegistry.RegisterOrchestrator(k3s.NewOrchestrator())
```

### 4. Configuration Management

**Format**: YAML (was JSON, migrated to YAML in v2.0)

**Files**:
- `config.yaml`: User configuration template (AWS creds, defaults)
- `~/.go-kubernetes-helper/clusters.yaml`: Cluster registry

**Features**:
- Tilde expansion (`~/path` → `/home/user/path`)
- Smart defaults for missing values
- Secure file permissions (0600)
- Backward compatibility with old JSON format

**Example `config.yaml`**:
```yaml
aws_access_key: AKIA...
aws_secret_key: Ca8k...
aws_region: us-west-2
subnet_id: subnet-xxxxx
security_group_id: sg-xxxxx
ssh_key_name: my-key-pair
ssh_private_key_path: ~/.ssh/id_rsa
node_prefix: rancher-node
ami: ami-xxxxx
instance_count: 3
root_volume_size: 20
kubernetes_distribution: rke2
kubernetes_version: v1.33.7+rke2r1
rancher_version: 2.10.2
```

### 5. Infrastructure as Code

**Tool**: OpenTofu (Terraform-compatible)

**Generated Files**:
- `clusters/<name>/main.tf`: AWS EC2 instance definitions
- `clusters/<name>/terraform.tfstate`: Infrastructure state

**Resources Created**:
- EC2 instances (t3.xlarge)
- Public IPs
- Instance tags
- SSH access configuration

### 6. Configuration Management

**Tool**: Ansible

**Generated Files**:
- `clusters/<name>/site.yml`: Kubernetes deployment playbook
- `clusters/<name>/hosts.ini`: Dynamic inventory

**Playbook Stages**:
1. **Initialize**: First node becomes control plane
2. **Join**: Additional nodes join cluster
3. **Deploy Rancher**: Helm chart installation

**Inventory Groups**:
- `[init]`: First node (Rancher host)
- `[join]`: Additional nodes

### 7. SSH Readiness Checks

**Problem**: EC2 instances need 30-90 seconds to boot

**Solution**: Automatic SSH availability checking
- Tests SSH connection to each instance
- Retries up to 30 times (5 minutes)
- 10-second delay between attempts
- Progress feedback per instance

**Output**:
```
Waiting for SSH to be available on all instances...
  [1/3] Waiting for SSH on 44.251.186.75...
  [1/3] ✓ SSH ready on 44.251.186.75
  [2/3] ✓ SSH ready on 35.95.104.65
  [3/3] ✓ SSH ready on 44.243.144.213
✓ All instances ready for provisioning
```

### 8. Rancher DNS Support

**Why**: Rancher Ingress requires DNS name (not IP)

**Solution**: Uses EC2 auto-generated DNS names
- Format: `ec2-X-X-X-X.compute-1.amazonaws.com`
- Passed to Ansible as `rancher_hostname` variable
- Enables SSL certificates and ingress

**Access**: `https://ec2-X-X-X-X.compute.amazonaws.com/dashboard`

### 9. Enhanced Error Reporting

**Features**:
- Detailed error messages with command output
- Boxed formatting for visibility
- Dedicated error log files per command
- Context-aware error messages

**Error Display**:
```
╔══════════════════════════════════════════╗
║ COMMAND FAILED: tofu apply -auto-approve
╠══════════════════════════════════════════╣
║ Error: exit status 1
╠══════════════════════════════════════════╣
║ OUTPUT:
╚══════════════════════════════════════════╝

  [actual command output]

╔══════════════════════════════════════════╗
║ Full logs: logs/deployment.log
╚══════════════════════════════════════════╝
```

### 10. Structured Logging

**Tool**: Zap logger

**Log Files**:
- `logs/deployment.log`: Main deployment logs
- `logs/tofu_error.log`: OpenTofu failures
- `logs/ansible-playbook_error.log`: Ansible failures

**Log Content**:
- Command execution details
- Full stdout/stderr output
- Error stack traces
- Timestamp and context

---

## Configuration

### AWS Prerequisites

**Required**:
1. AWS account with programmatic access
2. IAM user with EC2 permissions
3. Existing VPC with:
   - Subnet (public or private)
   - Security group with required ports:
     - 22 (SSH)
     - 80 (HTTP)
     - 443 (HTTPS)
     - 6443 (Kubernetes API)
     - 9345 (RKE2 supervisor) or 10250 (K3s)
4. EC2 key pair (`.pem` file)

**Security Group Ports**:
```
Inbound Rules:
- 22/tcp    (SSH)
- 80/tcp    (HTTP)
- 443/tcp   (HTTPS)
- 6443/tcp  (Kubernetes API)
- 9345/tcp  (RKE2 supervisor)
- 10250/tcp (Kubelet API)
```

### Local Prerequisites

**Required Tools**:
- OpenTofu (or Terraform) - `tofu` command in PATH
- Ansible - `ansible-playbook` command in PATH
- SSH client
- Go 1.24+ (for building from source)

**Installation**:
```bash
# OpenTofu
brew install opentofu  # macOS
# or download from https://opentofu.org/

# Ansible
pip install ansible

# Verify
tofu --version
ansible-playbook --version
```

### File Permissions

**Security**:
- `config.yaml`: 0600 (owner read/write only)
- `~/.go-kubernetes-helper/clusters.yaml`: 0600
- SSH private keys: 0600
- Generated files: 0644 (no secrets)

---

## Version History

### v2.0 (2026-02-10)
- ✅ Added K3s orchestrator support
- ✅ TUI distribution selector (RKE2/K3s)
- ✅ Migrated from JSON to YAML configuration
- ✅ Fixed Jinja2 template escaping in Ansible playbooks
- ✅ Added path expansion for SSH keys (~/ support)
- ✅ Comprehensive unit tests for validation

### v1.1 (2026-02-09)
- ✅ Multi-cluster management (create, list, delete)
- ✅ Cluster state tracking in ~/.go-kubernetes-helper/
- ✅ Isolated build directories per cluster
- ✅ Enhanced error reporting with command output
- ✅ SSH readiness checks before Ansible
- ✅ Rancher DNS name support

### v1.0 (Initial Release)
- ✅ Interactive TUI configuration
- ✅ RKE2 cluster deployment
- ✅ AWS EC2 provisioning with OpenTofu
- ✅ Ansible playbook generation
- ✅ Rancher installation
- ✅ Structured logging

---

## Troubleshooting

### Common Issues

**1. SSH Connection Failures**
- Check security group allows port 22
- Verify SSH key path is correct and has 0600 permissions
- Ensure key pair matches AWS key name

**2. Tofu Apply Failures**
- Verify AWS credentials are valid
- Check IAM permissions for EC2
- Ensure subnet and security group exist
- Check AWS quota limits

**3. Ansible Playbook Failures**
- Wait for SSH readiness checks to complete
- Verify all instances are running in AWS console
- Check Ansible logs in `logs/ansible-playbook_error.log`

**4. Rancher Access Issues**
- Use DNS name, not IP address
- Wait 5-10 minutes for Rancher to initialize
- Check security group allows port 443
- Verify cert-manager deployed successfully

---

**End of Context Documentation**
