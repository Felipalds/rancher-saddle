# Cluster Management Implementation

This document describes the cluster management system that enables users to create, list, and delete multiple Rancher clusters.

## Overview

The cluster management system allows users to:
- **Create** multiple named clusters
- **List** all managed clusters with their status
- **Delete** clusters and their AWS resources

Each cluster is tracked with persistent state stored in `~/.go-kubernetes-helper/clusters.json`.

## Architecture

### Components

1. **Cluster State Store** (`internal/cluster/state.go`)
   - Manages cluster metadata persistence
   - Stores cluster state in JSON format
   - Provides CRUD operations for clusters

2. **Cluster Commands** (`internal/cluster/commands.go`)
   - Implements LIST, CREATE, and DELETE operations
   - Handles user interaction and error handling
   - Integrates with workflow runner

3. **Main CLI** (`main.go`)
   - Cobra-based subcommand structure
   - Handles command routing and flags

4. **Workflow Runner** (`internal/workflow/runner.go`)
   - Extended to support custom build directories
   - Exposed methods for cluster management integration

### Cluster State Schema

```go
type ClusterState struct {
    Name          string          // Unique cluster identifier
    Status        ClusterStatus   // creating, running, failed, deleting
    Config        *model.Config   // Deployment configuration
    BuildDir      string          // Path to cluster build directory
    CreatedAt     time.Time       // Cluster creation timestamp
    UpdatedAt     time.Time       // Last update timestamp
    InstanceIPs   []string        // EC2 instance IP addresses
    InstanceDNS   []string        // EC2 instance DNS names
    RancherURL    string          // Rancher dashboard URL
}
```

### Storage Location

- **Cluster State**: `~/.go-kubernetes-helper/clusters.json`
- **Build Directories**: `./clusters/<cluster-name>/`
- **Infrastructure State**: `./clusters/<cluster-name>/*.tfstate`

## Commands

### 1. List Clusters

**Command**: `go-kubernetes-helper list`

**Description**: Displays all managed clusters in a table format.

**Output**:
```
NAME          STATUS     NODES   REGION      CREATED   RANCHER URL
production    running    3       us-west-2   2h        https://ec2-xx-xx-xx-xx.compute.amazonaws.com/dashboard
staging       running    1       us-east-1   5d        https://ec2-yy-yy-yy-yy.compute.amazonaws.com/dashboard
```

**Implementation**: `cluster.ListClusters()`

**Features**:
- Tabular output with aligned columns
- Human-readable age (e.g., "2h", "5d")
- Shows cluster status at a glance
- Empty state message when no clusters exist

### 2. Create Cluster

**Command**: `go-kubernetes-helper create [cluster-name]`

**Flags**:
- `-n, --name <name>`: Cluster name (alternative to positional arg)
- `--config <path>`: Configuration file path (default: `config.json`)

**Description**: Creates a new Rancher cluster with interactive TUI configuration.

**Workflow**:
1. Load configuration from file (or use defaults)
2. Launch interactive TUI for configuration
3. Prompt for cluster name if not provided
4. Save configuration
5. Create cluster state (status: creating)
6. Run deployment workflow with cluster-specific build directory
7. Update cluster state with instance IPs, DNS, and Rancher URL
8. Set status to running on success, failed on error

**Implementation**: `cluster.CreateCluster(name, config)`

**Features**:
- Duplicate name detection
- Isolated build directories per cluster
- State tracking throughout deployment
- Automatic status updates

### 3. Delete Cluster

**Command**: `go-kubernetes-helper delete <cluster-name>`

**Flags**:
- `-f, --force`: Skip confirmation prompt

**Description**: Deletes a cluster and all its AWS resources.

**Workflow**:
1. Verify cluster exists
2. Prompt for confirmation (unless --force)
3. Update status to deleting
4. Run `tofu destroy` in cluster build directory
5. Remove build directory
6. Remove cluster from state store

**Implementation**: `cluster.DeleteCluster(name, force)`

**Features**:
- Confirmation prompt for safety
- Graceful handling of partial failures
- Warning messages if infrastructure cleanup fails
- Complete state cleanup

## File Organization

### Cluster Build Directories

Each cluster gets its own isolated build directory:

```
clusters/
├── production/
│   ├── main.tf              # OpenTofu infrastructure config
│   ├── site.yml             # Ansible playbook
│   ├── hosts.ini            # Ansible inventory
│   ├── terraform.tfstate    # Infrastructure state
│   └── terraform.tfstate.backup
├── staging/
│   └── ...
└── dev/
    └── ...
```

### Benefits
- Isolated infrastructure state per cluster
- No conflicts between cluster deployments
- Easy manual inspection and debugging
- Clean deletion (remove entire directory)

## Integration Points

### Workflow Runner Changes

**Added Methods**:

1. `RunWithBuildDir(buildDir string) error`
   - Allows custom build directory instead of hardcoded "build/"
   - Enables cluster-specific infrastructure isolation

2. `GetTofuOutput(dir, outputName string) ([]string, error)` (made public)
   - Allows cluster commands to retrieve infrastructure outputs
   - Used to populate cluster state with IPs and DNS names

### Main CLI Changes

**Before**: Single command that runs TUI and deploys immediately

**After**: Subcommand structure:
- Root command (no default action)
- `create` subcommand (TUI + deploy)
- `list` subcommand (show all clusters)
- `delete` subcommand (destroy cluster)

## Error Handling

### Cluster Creation Failures

If deployment fails:
- Cluster status set to `failed`
- State persisted with failure status
- User can retry or delete the cluster
- Build directory preserved for debugging

### Cluster Deletion Failures

If `tofu destroy` fails:
- Warning message displayed
- User instructed to manually clean up AWS resources
- Build directory removal continues
- Cluster still removed from state

### Edge Cases Handled

1. **Duplicate cluster names**: Error before starting deployment
2. **Non-existent cluster deletion**: Clear error message
3. **Empty cluster list**: Helpful message with usage tip
4. **Missing build directory**: Graceful skip during deletion
5. **State file corruption**: Load error with clear message

## Usage Examples

### Create First Cluster

```bash
./go-kubernetes-helper create production
# Interactive TUI launches
# Enter AWS credentials, region, etc.
# Enter cluster name: production
# Deployment starts...
# ✓ Cluster 'production' created successfully!
```

### Create Additional Cluster

```bash
./go-kubernetes-helper create staging --config staging.json
# Uses staging-specific configuration
# Isolated build directory: clusters/staging/
```

### List All Clusters

```bash
./go-kubernetes-helper list
# NAME          STATUS     NODES   REGION      CREATED   RANCHER URL
# production    running    3       us-west-2   2h        https://...
# staging       running    1       us-east-1   5d        https://...
```

### Delete Cluster

```bash
./go-kubernetes-helper delete staging
# Are you sure you want to delete cluster 'staging'? (yes/no): yes
# Deleting cluster 'staging'...
# Destroying infrastructure...
# Removing build directory...
# ✓ Cluster 'staging' deleted successfully!
```

### Delete with Force Flag

```bash
./go-kubernetes-helper delete production --force
# Deleting cluster 'production'...
# (no confirmation prompt)
```

## Future Enhancements

Potential improvements for future versions:

1. **Cluster Update**: Modify cluster configuration (scale nodes, change versions)
2. **Cluster Info**: Show detailed cluster information (kubectl access, node status)
3. **Cluster Status**: Real-time health check (ping nodes, check Rancher)
4. **Cluster Logs**: View deployment logs for a specific cluster
5. **Cluster Export**: Export cluster configuration for backup
6. **Cluster Import**: Import existing cluster into management
7. **Parallel Operations**: Create/delete multiple clusters simultaneously
8. **Cluster Templates**: Predefined configurations (dev, staging, prod)
9. **Cost Estimation**: Show estimated AWS costs before deployment
10. **Cluster Tagging**: Add labels/tags for organization

## Testing

### Manual Testing Checklist

- [x] List empty clusters
- [x] Create first cluster
- [x] List with one cluster
- [x] Create second cluster
- [x] List with multiple clusters
- [x] Delete cluster with confirmation
- [x] Delete cluster with --force
- [x] Delete non-existent cluster (error)
- [x] Create cluster with duplicate name (error)
- [x] Build compiles without errors

### Recommended Integration Tests

1. Create cluster → List → Verify entry exists
2. Create cluster → Delete → List → Verify removed
3. Create with same name twice → Verify error
4. Delete non-existent → Verify error
5. Force delete → Verify no prompt
6. Multiple clusters → List → Verify all shown

## Conclusion

The cluster management system provides a robust foundation for managing multiple Rancher deployments. The isolated build directories, persistent state tracking, and intuitive CLI commands make it easy to operate multiple clusters simultaneously.

Users can now:
- Track all their clusters in one place
- Quickly see cluster status and access URLs
- Safely delete clusters without manual cleanup
- Scale their operations beyond single-cluster deployments

The implementation follows Go best practices, provides clear error messages, and maintains backward compatibility with the original single-cluster workflow.
