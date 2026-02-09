# Go Kubernetes Helper (Rancher Deployment Tool)

Automated deployment tool for Rancher/RKE2 clusters on AWS EC2 using OpenTofu and Ansible with an interactive TUI.

## Features

- 🖥️ **Interactive TUI** - Bubbletea-based terminal interface for configuration
- ☁️ **AWS EC2 Deployment** - Automated instance provisioning with OpenTofu
- 🎯 **RKE2 Cluster** - Multi-node Kubernetes cluster setup
- 🐮 **Rancher Management** - Automatic Rancher server deployment
- 📝 **Configuration Persistence** - Save and reuse deployment configurations
- 📊 **Structured Logging** - Detailed deployment logs with Zap

## Prerequisites

### Required Tools
- [OpenTofu](https://opentofu.org/) or Terraform
- [Ansible](https://www.ansible.com/)
- Go 1.24.9+
- AWS Account with appropriate permissions

### AWS Requirements
- Valid AWS access key and secret key
- Existing VPC with subnet
- Security group with required ports:
  - SSH (22)
  - Kubernetes API (6443)
  - RKE2 supervisor (9345)
  - Rancher UI (80, 443)
- EC2 key pair for SSH access

## Installation

```bash
# Clone the repository
git clone https://github.com/Felipalds/go-kubernetes-helper.git
cd go-kubernetes-helper

# Build the application
go build -o go-kubernetes-helper

# Or run directly
go run main.go
```

## Usage

### Quick Start

```bash
# Create a new cluster with interactive TUI
./go-kubernetes-helper create my-cluster

# List all clusters
./go-kubernetes-helper list

# Delete a cluster
./go-kubernetes-helper delete my-cluster
```

### Available Commands

#### `create [cluster-name]`
Creates a new Rancher cluster with interactive configuration.

```bash
# Create with inline name
./go-kubernetes-helper create production

# Create with --name flag
./go-kubernetes-helper create --name staging

# Create with custom config file
./go-kubernetes-helper create dev --config dev-config.json
```

#### `list`
Lists all managed clusters with their status.

```bash
./go-kubernetes-helper list
```

Output example:
```
NAME          STATUS     NODES   REGION      CREATED   RANCHER URL
production    running    3       us-west-2   2h        https://ec2-xx-xx-xx-xx.compute.amazonaws.com/dashboard
staging       running    1       us-east-1   5d        https://ec2-yy-yy-yy-yy.compute.amazonaws.com/dashboard
```

#### `delete <cluster-name>`
Deletes a cluster and all its AWS resources.

```bash
# Delete with confirmation prompt
./go-kubernetes-helper delete staging

# Delete without confirmation
./go-kubernetes-helper delete staging --force
```

### TUI Navigation

- **Tab / Shift+Tab** - Move between fields
- **Arrow Keys** - Navigate up/down
- **Enter** - Submit form or move to next field
- **Ctrl+C / ESC** - Cancel deployment

### Configuration Fields

The TUI will prompt for:
- AWS Access Key
- AWS Secret Key (masked)
- AWS Region (default: us-east-1)
- Subnet ID
- Security Group ID
- SSH Key Name
- SSH Private Key Path
- Node Prefix (default: rancher-node)
- AMI ID (default: Ubuntu 22.04 LTS)
- Instance Count (default: 1)
- RKE2 Version (default: v1.33.7+rke2r1)
- Rancher Version (default: 2.10.2)

### Deployment Process

The tool automatically:
1. Generates OpenTofu configuration
2. Provisions EC2 instances
3. Creates Ansible inventory
4. Installs RKE2 on all nodes
5. Deploys Rancher on the first node

Deployment typically takes 6-17 minutes.

## Configuration File

Configuration is stored in JSON format:

```json
{
  "aws_access_key": "AKIA...",
  "aws_secret_key": "...",
  "aws_region": "us-west-2",
  "subnet_id": "subnet-...",
  "security_group_id": "sg-...",
  "ssh_key_name": "my-key",
  "ssh_private_key_path": "~/Downloads/my-key.pem",
  "node_prefix": "rancher-lrosa",
  "ami": "ami-...",
  "instance_count": 3,
  "rke2_version": "v1.33.7+rke2r1",
  "rancher_version": "2.12.0"
}
```

⚠️ **Security Warning**: The config file contains AWS credentials. Never commit it to version control!

## Cluster State Management

Cluster state is persisted in `~/.go-kubernetes-helper/clusters.json`. This file tracks:
- Cluster name and status (creating, running, failed, deleting)
- Configuration used for deployment
- Build directory location
- Instance IPs and DNS names
- Rancher dashboard URL
- Creation and update timestamps

Example cluster state:
```json
{
  "production": {
    "name": "production",
    "status": "running",
    "config": { /* deployment config */ },
    "build_dir": "clusters/production",
    "created_at": "2026-02-09T10:30:00Z",
    "updated_at": "2026-02-09T10:45:00Z",
    "instance_ips": ["54.x.x.x", "54.x.x.x", "54.x.x.x"],
    "instance_dns": ["ec2-xx-xx-xx-xx.compute.amazonaws.com", ...],
    "rancher_url": "https://ec2-xx-xx-xx-xx.compute.amazonaws.com/dashboard"
  }
}
```

## Project Structure

```
.
├── cmd/                       # Command implementations
│   └── tui.go                # TUI entry point
├── internal/                 # Internal packages
│   ├── cluster/              # Cluster state management
│   │   ├── state.go         # Cluster state store
│   │   └── commands.go      # List/create/delete commands
│   ├── generator/            # Tofu/Ansible generators
│   ├── model/                # Data models
│   ├── tui/                  # TUI components
│   ├── utils/                # Utilities
│   └── workflow/             # Deployment orchestration
├── CONTEXT/                  # AI context documentation
├── clusters/                 # Cluster-specific build dirs (gitignored)
│   ├── production/           # Per-cluster infrastructure
│   │   ├── main.tf          # OpenTofu configuration
│   │   ├── site.yml         # Ansible playbook
│   │   ├── hosts.ini        # Ansible inventory
│   │   └── *.tfstate        # Infrastructure state
│   └── staging/
│       └── ...
├── logs/                     # Log files (gitignored)
│   ├── deployment.log        # Main deployment logs
│   └── *_error.log           # Command-specific error logs
├── config.json               # User configuration
├── main.go                   # Application entry point
├── .gitignore                # Git ignore patterns
└── README.md                 # This file
```

Cluster state is stored in: `~/.go-kubernetes-helper/clusters.json`

## Logs

Deployment logs are written to the `logs/` directory:
- `logs/deployment.log` - Structured Zap logs with full command output
- `logs/tofu_error.log` - OpenTofu error output (if any)
- `logs/ansible-playbook_error.log` - Ansible error output (if any)

## Accessing Rancher

After successful deployment:

1. Get the first node's DNS name from the output
2. Navigate to `https://<first-node-dns>` in your browser (e.g., `https://ec2-X-X-X-X.compute-1.amazonaws.com`)
3. Login with:
   - Username: `admin`
   - Password: `admin` (set during deployment)

## Development

### Tech Stack

- **Language**: Go 1.24.9
- **TUI**: Bubbletea, Bubbles, Lipgloss
- **CLI**: Cobra
- **Logging**: Zap
- **IaC**: OpenTofu
- **Config Mgmt**: Ansible

### Building from Source

```bash
# Install dependencies
go mod download

# Build
go build -o go-kubernetes-helper

# Run tests (if available)
go test ./...
```

## Documentation

Detailed documentation is available in `CONTEXT/`:
- [Project Overview](CONTEXT/PROJECT_OVERVIEW.md) - Architecture and workflow
- [Features](CONTEXT/FEATURES.md) - Feature documentation
- [Architecture](CONTEXT/ARCHITECTURE.md) - Technical design
- [Quick Reference](CONTEXT/QUICK_REFERENCE.md) - Common tasks

## Troubleshooting

### OpenTofu Errors
- Check AWS credentials are valid
- Verify subnet and security group exist
- Ensure IAM permissions are sufficient

### Ansible Errors
- Verify SSH key path is correct
- Check security group allows SSH (port 22)
- Wait for EC2 instances to fully initialize

### Connection Issues
- Security group must allow ingress on required ports
- Instances need public IPs for external access
- Check VPC routing and internet gateway

## Security Considerations

- Store AWS credentials securely (use AWS credential chain when possible)
- Rotate access keys regularly
- Use restrictive security groups
- Keep private keys secure (0600 permissions)
- Never commit `config.json` to version control

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

[Add your license here]

## Author

Felipalds @ SUSE (luiz.rosa@suse.com)

## Acknowledgments

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [OpenTofu](https://opentofu.org/) - Infrastructure as Code
- [Ansible](https://www.ansible.com/) - Configuration management
- [Rancher](https://rancher.com/) - Kubernetes management
