# Features Documentation

## Current Features

### 1. Interactive TUI Configuration Form
**Status**: ✅ Implemented
**Location**: `internal/tui/form.go`, `cmd/tui.go`

**Description**:
- Bubbletea-based interactive form for collecting deployment parameters
- 12 configurable fields with validation
- Keyboard navigation (Tab, Shift+Tab, Arrow keys, Enter)
- Password masking for sensitive fields (AWS Secret Key)
- Submit button with focus highlighting
- ESC/Ctrl+C to abort deployment

**User Experience**:
- Pink highlighting for focused fields
- Gray highlighting for blurred fields
- Real-time input validation (e.g., instance count must be integer)
- Character limit of 64 per field
- Cursor mode indicator (ctrl+r to change style)

---

### 2. Configuration Persistence
**Status**: ✅ Implemented
**Location**: `internal/model/config.go`

**Description**:
- JSON-based configuration storage in `config.json`
- Automatic loading of previous configurations
- Smart defaults for missing values
- Secure file permissions (0600) for credential protection
- Auto-migration of invalid values (e.g., old RKE2 versions)

**Default Values**:
- AWS Region: us-east-1
- Instance Count: 1
- Node Prefix: rancher-node
- AMI: ami-0c58b2975bef51185 (Ubuntu 22.04 LTS)
- RKE2 Version: v1.33.7+rke2r1
- Rancher Version: 2.10.2

---

### 3. OpenTofu Infrastructure Generation
**Status**: ✅ Implemented
**Location**: `internal/generator/tofu.go`

**Description**:
- Template-based generation of `main.tf`
- AWS provider configuration with credentials
- EC2 instance resource definition
- Uses t3.xlarge instance type
- Public IP association
- VPC subnet and security group assignment
- Output of instance public IPs

**Generated Resources**:
- AWS provider configuration
- Data source for Ubuntu AMI lookup
- EC2 instances (count configurable)
- Output block for IP addresses

---

### 4. Ansible Playbook Generation
**Status**: ✅ Implemented
**Location**: `internal/generator/ansible.go`

**Description**:
- Template-based generation of `site.yml`
- Multi-stage playbook structure:
  1. **Initialize RKE2 on First Node**: Sets up initial control plane
  2. **Join Additional Nodes**: Expands cluster with more control plane nodes
  3. **Deploy Rancher**: Installs Rancher management server

**Playbook Features**:
- Cloud-init wait handling
- RKE2 version pinning
- Node token fetching and distribution
- Helm and kubectl installation
- Cert-Manager deployment
- Rancher Helm chart installation
- Kubeconfig setup

---

### 5. Automated Deployment Workflow
**Status**: ✅ Implemented
**Location**: `internal/workflow/runner.go`

**Description**:
- End-to-end orchestration of deployment process
- Sequential execution of deployment stages
- Comprehensive logging with Zap
- Error handling and propagation
- Automatic inventory generation from Tofu outputs

**Workflow Steps**:
1. Create build directory
2. Generate OpenTofu configuration
3. Run `tofu init`
4. Run `tofu apply -auto-approve`
5. Extract instance IPs from Tofu output
6. Generate Ansible inventory (hosts.ini)
7. Generate Ansible playbook (site.yml)
8. Run `ansible-playbook -i hosts.ini site.yml`

**Inventory Structure**:
- `[init]` group: First node for cluster initialization
- `[join]` group: Additional nodes to join cluster
- SSH parameters: ubuntu user, StrictHostKeyChecking disabled
- Private key path from configuration

---

### 6. Structured Logging
**Status**: ✅ Implemented
**Location**: `internal/utils/logger.go`, `internal/workflow/runner.go`

**Description**:
- Zap logger integration
- File-based logging to `logs/deployment.log`
- Structured log format with context
- Command execution logging with full output
- Error tracking and debugging support
- Organized log directory structure

**Log Information**:
- Command name and arguments
- Command output (stdout + stderr)
- Error messages with stack traces
- Workflow stage information

**Log Files**:
- `logs/deployment.log` - Main deployment logs
- `logs/tofu_error.log` - OpenTofu command errors
- `logs/ansible-playbook_error.log` - Ansible playbook errors

---

### 7. CLI Interface
**Status**: ✅ Implemented
**Location**: `main.go`

**Description**:
- Cobra-based command-line interface
- Configuration file path flag
- Clean error handling and exit codes
- User-friendly console messages

**Usage**:
```bash
go-kubernetes-helper --config path/to/config.json
```

---

### 8. Enhanced Error Reporting
**Status**: ✅ Implemented
**Location**: `internal/workflow/runner.go`, `main.go`
**Added**: 2026-02-09

**Description**:
- Detailed error messages when OpenTofu or Ansible commands fail
- Formatted error output displayed directly in the terminal
- Command output captured and shown to users
- Dedicated error log files for each failed command

**Implementation Details**:
- Custom `CommandError` type that includes command, args, and full output
- Boxed error display format for better visibility
- Automatic creation of `<command>_error.log` files (e.g., `tofu_error.log`, `ansible-playbook_error.log`)
- Full command output included in error messages

**Error Display Format**:
```
╔══════════════════════════════════════════════════════════════════╗
║ COMMAND FAILED: tofu apply -auto-approve
╠══════════════════════════════════════════════════════════════════╣
║ Error: exit status 1
╠══════════════════════════════════════════════════════════════════╣
║ OUTPUT:
╚══════════════════════════════════════════════════════════════════╝

  [actual command output shown here line by line]

╔══════════════════════════════════════════════════════════════════╗
║ Full logs available in: logs/deployment.log
╚══════════════════════════════════════════════════════════════════╝
```

**Error Log Files Created**:
- `logs/tofu_error.log` - When OpenTofu commands fail
- `logs/ansible-playbook_error.log` - When Ansible playbook fails

**User Benefits**:
- Immediate visibility of what went wrong
- No need to search through deployment.log for errors
- Clear indication of which command failed
- Full output available for debugging

---

### 9. SSH Availability Check
**Status**: ✅ Implemented
**Location**: `internal/workflow/runner.go`
**Added**: 2026-02-09

**Description**:
- Automatically waits for SSH to be available on all EC2 instances before running Ansible
- Prevents "Connection refused" errors when instances are still booting
- Provides clear progress feedback for each instance
- Configurable retry mechanism with timeout

**Implementation Details**:
- Checks each instance sequentially for SSH availability
- Maximum 30 retry attempts per instance (5 minutes total)
- 10-second delay between retry attempts
- Uses SSH connection test with timeout
- Displays progress: `[1/3] Waiting for SSH on 44.234.116.251...`

**SSH Connection Parameters**:
- ConnectTimeout: 5 seconds
- StrictHostKeyChecking: disabled (for automation)
- BatchMode: enabled (non-interactive)
- Uses configured SSH private key from config

**User Experience**:
```
Waiting for SSH to be available on all instances...
  [1/3] Waiting for SSH on 44.234.116.251...
  [1/3] SSH not ready yet, waiting... (attempt 1/30)
  [1/3] ✓ SSH ready on 44.234.116.251
  [2/3] Waiting for SSH on 54.123.45.67...
  [2/3] ✓ SSH ready on 54.123.45.67
  [3/3] Waiting for SSH on 34.56.78.90...
  [3/3] ✓ SSH ready on 34.56.78.90
✓ All instances are ready for provisioning
Running Ansible Playbook...
```

**Error Handling**:
- Fails gracefully after maximum retries
- Returns detailed error message with IP address
- Full error reporting through CommandError system

**Why This Matters**:
- EC2 instances take 30-90 seconds to boot after creation
- SSH service starts after cloud-init completes
- Running Ansible too early results in connection failures
- This feature eliminates timing-related deployment failures

---

### 10. Public DNS Name Support for Rancher
**Status**: ✅ Implemented
**Location**: `internal/generator/tofu.go`, `internal/generator/ansible.go`, `internal/workflow/runner.go`
**Added**: 2026-02-09

**Description**:
- Fetches EC2 public DNS names in addition to IPs
- Uses DNS name for Rancher hostname (required by Rancher ingress)
- Prevents "must be a DNS name, not an IP address" error
- Enhanced cert-manager webhook readiness checks

**Why This Matters**:
- Rancher Ingress requires a valid DNS name, not an IP address
- AWS EC2 instances have auto-generated public DNS names (e.g., `ec2-X-X-X-X.compute-1.amazonaws.com`)
- Using DNS names ensures SSL certificates and ingress work correctly

**Implementation Details**:

**OpenTofu Changes**:
- Added `instance_dns_names` output to capture `public_dns` from EC2 instances
- Both IPs and DNS names are now fetched after instance creation

**Ansible Changes**:
- Changed Rancher hostname from `{{ ansible_host }}` (IP) to `{{ rancher_hostname }}` (DNS)
- Added `rancher_hostname` variable to init host inventory
- Enhanced cert-manager readiness checks:
  - Wait for all deployments to be Available
  - Wait for webhook pod to be Ready
  - Additional 30-second pause for webhook registration

**Workflow Changes**:
- Fetches both `instance_ips` and `instance_dns_names` outputs
- Displays instance information: `IP: 44.234.116.251, DNS: ec2-44-234-116-251.compute-1.amazonaws.com`
- Passes DNS name as `rancher_hostname` variable in inventory for init node

**Inventory Example**:
```ini
[init]
X.X.X.1 ansible_user=ubuntu ansible_ssh_private_key_file=~/.ssh/key.pem rancher_hostname=ec2-X-X-X-1.compute-1.amazonaws.com

[join]
X.X.X.2 ansible_user=ubuntu ansible_ssh_private_key_file=~/.ssh/key.pem
```

**User Experience**:
```
Fetching instance information...
Created 3 instance(s):
  [1] IP: X.X.X.1, DNS: ec2-X-X-X-1.compute-1.amazonaws.com
  [2] IP: X.X.X.2, DNS: ec2-X-X-X-2.compute-1.amazonaws.com
  [3] IP: X.X.X.3, DNS: ec2-X-X-X-3.compute-1.amazonaws.com
```

**Accessing Rancher**:
After deployment, access Rancher using the DNS name:
```
https://ec2-X-X-X-X.compute-1.amazonaws.com
```

---

## Planned Features

### Template for New Features

Use this template when documenting new features in future contexts:

```markdown
### [Feature Number]. [Feature Name]
**Status**: 🚧 In Progress / 📋 Planned / ✅ Implemented
**Location**: [File paths]
**Added**: [Date]
**Requested By**: [Context/Issue reference]

**Description**:
[What the feature does]

**Implementation Details**:
- [Key technical points]
- [Dependencies]
- [Configuration changes]

**Usage**:
[How users interact with this feature]

**Testing**:
[How to verify the feature works]

**Related Issues**:
[Links to related features or bugs]
```

---

## Feature Request Tracking

Add new feature requests here with date and description:

<!-- Example:
### 2026-02-09: Multi-Cloud Support
Add support for Azure and GCP in addition to AWS
- Requires provider abstraction layer
- New configuration parameters
- Provider-specific generators
-->

