# Project Overview: Go Kubernetes Helper

## Purpose
This is an automated deployment tool for Rancher/RKE2 clusters on AWS EC2 using OpenTofu (Terraform) and Ansible. It features a Bubbletea TUI for user input collection.

## Architecture

### Main Components

1. **Entry Point** (`main.go`)
   - Loads configuration from `config.json`
   - Launches the TUI for user input
   - Saves updated configuration
   - Executes the deployment workflow

2. **TUI Interface** (`cmd/tui.go` + `internal/tui/form.go`)
   - Built with Bubbletea/Bubbles
   - Collects 12 configuration parameters from user
   - Supports keyboard navigation (tab/shift+tab/up/down/enter)
   - Password masking for AWS secret key
   - Submit button for deployment initiation

3. **Configuration Model** (`internal/model/config.go`)
   - Defines the Config struct with all deployment parameters
   - Loads/saves JSON configuration files
   - Provides sensible defaults for missing values

4. **Infrastructure Generators**
   - **OpenTofu Generator** (`internal/generator/tofu.go`)
     - Generates `main.tf` from configuration
     - Creates AWS EC2 instances
     - Uses t3.xlarge instance type
     - Outputs public IPs for provisioning

   - **Ansible Generator** (`internal/generator/ansible.go`)
     - Generates `site.yml` playbook
     - Handles RKE2 cluster initialization
     - Configures additional nodes to join cluster
     - Deploys Rancher management server

5. **Workflow Runner** (`internal/workflow/runner.go`)
   - Orchestrates the deployment process
   - Executes OpenTofu commands
   - Generates Ansible inventory from EC2 IPs
   - Runs Ansible playbook
   - Logs all operations with Zap logger

6. **Utilities** (`internal/utils/logger.go`)
   - Zap logger initialization
   - Outputs to `deployment.log`

## Workflow Sequence

1. User runs the application
2. TUI displays form with configuration fields
3. User fills/edits values and submits
4. Configuration is saved to `config.json`
5. OpenTofu configuration is generated in `build/` directory
6. `tofu init` initializes provider
7. `tofu apply` creates EC2 instances
8. Instance IPs are fetched from Tofu output
9. Ansible inventory (`hosts.ini`) is generated with IP addresses
10. Ansible playbook (`site.yml`) is generated
11. `ansible-playbook` runs to:
    - Install RKE2 on first node (init group)
    - Join additional nodes to cluster (join group)
    - Deploy Rancher on first node
12. Deployment complete

## Configuration Parameters

| Field | Description | Default |
|-------|-------------|---------|
| aws_access_key | AWS IAM access key | - |
| aws_secret_key | AWS IAM secret key | - |
| aws_region | AWS region for deployment | us-east-1 |
| subnet_id | VPC subnet ID | - |
| security_group_id | Security group ID | - |
| ssh_key_name | EC2 key pair name | - |
| ssh_private_key_path | Local path to private key | - |
| node_prefix | EC2 instance name prefix | rancher-node |
| ami | Ubuntu AMI ID | ami-0c58b2975bef51185 |
| instance_count | Number of EC2 instances | 1 |
| rke2_version | RKE2 version to install | v1.33.7+rke2r1 |
| rancher_version | Rancher version to deploy | 2.10.2 |

## Tech Stack

- **Language**: Go 1.24.9
- **TUI Framework**: Bubbletea + Bubbles + Lipgloss
- **CLI Framework**: Cobra
- **Logging**: Zap
- **IaC**: OpenTofu (Terraform-compatible)
- **Configuration Management**: Ansible
- **Cloud Provider**: AWS

## Dependencies

```go
- github.com/charmbracelet/bubbles v0.21.1
- github.com/charmbracelet/bubbletea v1.3.10
- github.com/charmbracelet/lipgloss v1.1.0
- github.com/spf13/cobra v1.10.2
- go.uber.org/zap v1.27.1
```

## Directory Structure

```
.
├── cmd/
│   └── tui.go              # TUI entry point
├── internal/
│   ├── generator/
│   │   ├── ansible.go      # Ansible playbook generator
│   │   └── tofu.go         # OpenTofu config generator
│   ├── model/
│   │   └── config.go       # Configuration struct
│   ├── tui/
│   │   └── form.go         # TUI form implementation
│   ├── utils/
│   │   └── logger.go       # Logger setup
│   └── workflow/
│       └── runner.go       # Deployment orchestration
├── build/                  # Generated IaC files (gitignored)
│   ├── main.tf
│   ├── hosts.ini
│   └── site.yml
├── CONTEXT/                # Documentation for AI context
│   └── PROJECT_OVERVIEW.md
├── logs/                   # Log files (gitignored)
│   ├── deployment.log      # Main deployment logs
│   └── *_error.log         # Command error logs
├── config.json             # User configuration (contains secrets!)
├── .gitignore              # Git ignore patterns
├── main.go                 # Application entry point
├── go.mod
└── go.sum
```

## Build Artifacts

The `build/` directory contains:
- `main.tf` - OpenTofu/Terraform configuration
- `hosts.ini` - Ansible inventory with EC2 IPs
- `site.yml` - Ansible playbook
- `.terraform/` - OpenTofu provider cache
- `terraform.tfstate` - Infrastructure state

## Security Considerations

⚠️ **IMPORTANT**:
- `config.json` contains AWS credentials and should NEVER be committed to version control
- File permissions on `config.json` are set to 0600 (owner read/write only)
- Consider using AWS credential chain or environment variables instead
- SSH private keys should be protected separately

## Future Enhancement Ideas

- Add destroy/cleanup workflow
- Support for multiple cloud providers (Azure, GCP)
- Environment validation before deployment
- Pre-flight checks for AWS credentials
- Progress indicators during long-running operations
- Rollback capabilities
- Multi-region deployment
- Support for custom Ansible playbooks
- Integration with CI/CD pipelines
