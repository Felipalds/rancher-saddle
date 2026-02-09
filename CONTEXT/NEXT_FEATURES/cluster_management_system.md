# Feature: Cluster Management System

**Status**: 📋 Planned
**Priority**: High
**Complexity**: High (30 days estimated)
**Requested**: 2026-02-09

## Overview

Transform go-kubernetes-helper from a single-deployment tool into a comprehensive cluster lifecycle management system. Users will be able to track, view, edit, scale, and destroy multiple Rancher clusters through an enhanced TUI with menu navigation.

## User Requirements

1. **Track Clusters**: Maintain a registry of all created clusters
2. **View Clusters**: List and view details of existing clusters
3. **Edit Clusters**: Modify cluster configurations and reapply
4. **Scale Clusters**: Add or remove nodes from existing clusters
5. **Destroy Clusters**: Clean up infrastructure when no longer needed

## Current State

The application currently:
- Supports creating a single cluster per invocation
- Uses a form-based TUI for configuration input
- Saves config to `config.json` (single file, overwritten each time)
- Runs tofu/ansible in a single `build/` directory
- No persistent deployment tracking
- No way to manage previously created clusters

## Proposed Architecture

### 1. State Management

**Central Registry**: `deployments.json`
```json
{
  "deployments": [
    {
      "id": "cluster-1707513920",
      "name": "rancher-prod",
      "created_at": "2024-02-09T15:12:00Z",
      "updated_at": "2024-02-09T15:25:00Z",
      "status": "active",
      "region": "us-west-2",
      "instance_count": 3,
      "instance_ips": ["X.X.X.1", "X.X.X.2", "X.X.X.3"],
      "instance_dns": ["ec2-...", "ec2-...", "ec2-..."],
      "config": { /* full config snapshot */ },
      "build_dir": "build/cluster-1707513920",
      "rancher_url": "https://ec2-X-X-X-X.compute-1.amazonaws.com",
      "rancher_version": "2.12.0",
      "rke2_version": "v1.33.7+rke2r1"
    }
  ]
}
```

**Directory Structure**:
```
/
├── deployments.json              # Central registry
├── build/
│   ├── cluster-<timestamp1>/    # Per-deployment build dir
│   │   ├── main.tf
│   │   ├── terraform.tfstate
│   │   ├── site.yml
│   │   └── hosts.ini
│   └── cluster-<timestamp2>/
│       └── ...
└── logs/
    ├── cluster-<timestamp1>.log
    └── cluster-<timestamp2>.log
```

### 2. TUI Menu System

**Navigation Flow**:
```
Main Menu
├─ Create New Cluster → Form → Workflow → Complete
├─ List Clusters → Select Cluster → Cluster Detail
│                                    ├─ Edit Config
│                                    ├─ Scale Cluster
│                                    ├─ Destroy Cluster
│                                    ├─ View Logs
│                                    └─ Back
└─ Exit
```

**View Hierarchy**:
- MainModel (orchestrator with ViewState)
  - MenuModel (3 options)
  - FormModel (existing, renamed)
  - ClusterListModel (shows all active/failed clusters)
  - ClusterDetailModel (shows details + action menu)

### 3. Operations

**Create**:
- Generate unique ID (timestamp-based)
- Create deployment-specific build directory
- Save deployment record with status "creating"
- Run workflow
- Update deployment with IPs/DNS, status "active"

**List**:
- Load deployments.json
- Display active and creating clusters
- Show: name, status, node count, age
- Navigate with arrow keys

**View Details**:
- Display full deployment metadata
- Show all nodes (IP + DNS)
- List available actions
- Show Rancher URL for access

**Edit**:
- Load deployment's config snapshot
- Populate form
- Submit → regenerate tofu → apply changes
- Update deployment record

**Scale**:
- Prompt for new node count
- Update config, regenerate tofu
- Apply infrastructure changes
- Provision new nodes (if scaling up)
- Update deployment record

**Destroy**:
- Show confirmation dialog
- Run `tofu destroy` in deployment's build dir
- Update deployment status to "destroyed"
- Keep deployment record (audit trail)

## Technical Design

### New Files

1. **internal/model/deployment.go**
   - `Deployment` struct (ID, name, timestamps, status, IPs, DNS, config, etc.)
   - `DeploymentRegistry` with CRUD operations
   - JSON serialization

2. **internal/tui/main_model.go**
   - Composite model with ViewState enum
   - Delegates to sub-models
   - Manages shared state

3. **internal/tui/menu.go**
   - Simple menu with 3 options
   - Focus-based navigation

4. **internal/tui/cluster_list.go**
   - Lists deployments
   - Formatted display with status colors

5. **internal/tui/cluster_detail.go**
   - Shows deployment details
   - Action menu

6. **internal/tui/styles.go**
   - Shared lipgloss styles

7. **internal/workflow/destroyer.go**
   - Handles `tofu destroy` workflow
   - Updates deployment status

8. **internal/workflow/scaler.go**
   - Handles scaling operations
   - Updates instance count

### Modified Files

1. **internal/tui/form.go**
   - Rename `Model` to `FormModel`

2. **cmd/tui.go**
   - Launch `MainModel` instead of `FormModel`
   - Return `TUIResult` with action type

3. **main.go**
   - Add action handlers (create, edit, destroy, scale)
   - Generate deployment IDs
   - Update registry before/after operations
   - Add migration logic

4. **internal/workflow/runner.go**
   - Add `Deployment` field
   - Use deployment-specific build directories
   - Store IPs/DNS for retrieval

## Implementation Phases

### Phase 1: Foundation (Days 1-3)
- ✅ Create deployment model
- ✅ Implement registry with tests
- ✅ Refactor form.go
- ✅ Create shared styles

### Phase 2: Menu System (Days 4-6)
- ✅ Implement menu
- ✅ Create main model with routing
- ✅ Update TUI entry point
- ✅ Test navigation

### Phase 3: Deployment Tracking (Days 7-10)
- ✅ Modify runner for per-deployment builds
- ✅ Update create flow
- ✅ Add migration logic
- ✅ Test create → track

### Phase 4: Cluster List (Days 11-13)
- ✅ Implement list view
- ✅ Wire to main model
- ✅ Test with multiple clusters

### Phase 5: Cluster Detail (Days 14-16)
- ✅ Implement detail view
- ✅ Wire navigation
- ✅ Test display

### Phase 6: Destroy Workflow (Days 17-19)
- ✅ Implement destroyer
- ✅ Add confirmation
- ✅ Test destroy flow

### Phase 7: Edit and Scale (Days 20-25)
- ✅ Implement edit flow
- ✅ Implement scaler
- ✅ Test operations

### Phase 8: Polish (Days 26-30)
- ✅ Add view logs
- ✅ Error handling
- ✅ Documentation
- ✅ Integration testing

## Benefits

1. **Multi-Cluster Support**: Manage multiple clusters simultaneously
2. **Persistent Tracking**: Never lose track of created infrastructure
3. **Easy Cleanup**: Destroy clusters with a few keystrokes
4. **Audit Trail**: Keep history of all deployments
5. **Better UX**: Menu-driven navigation vs. re-running the whole app
6. **Scalability**: Grow/shrink clusters as needed
7. **State Isolation**: Each cluster has its own state files

## Migration Path

Users with existing deployments will be auto-migrated on first launch:

1. Detect legacy setup (config.json + build/ directory)
2. Show migration prompt
3. Create deployment record from existing state
4. Move `build/` to `build/<id>/`
5. Save to deployments.json
6. User can now manage legacy cluster through new system

No manual intervention required.

## Potential Issues and Mitigations

### Issue 1: Build Directory Conflicts
**Mitigation**: Use unique timestamp-based IDs for each deployment

### Issue 2: Concurrent Access
**Mitigation**: File locking or atomic writes (not a concern for single-user TUI)

### Issue 3: Orphaned Resources
**Mitigation**: Never delete deployment records; mark as destroyed

### Issue 4: Config Drift
**Mitigation**: Store config snapshots; edit creates new config

### Issue 5: Scaling Down Leader
**Mitigation**: Prevent removal of node 0 (Rancher server)

## Testing Plan

1. **Unit Tests**: Deployment model CRUD, registry save/load
2. **Integration Tests**: Full lifecycle (create → list → view → destroy)
3. **Manual Tests**: Multi-cluster scenarios, migration, error cases
4. **Performance Tests**: 50+ deployments in registry

## Success Criteria

- ✅ Can create and track multiple clusters
- ✅ Can view details of any cluster
- ✅ Can destroy clusters cleanly
- ✅ Can scale clusters up and down
- ✅ Can edit cluster configurations
- ✅ Legacy deployments migrate automatically
- ✅ All operations logged properly
- ✅ State never corrupted
- ✅ Intuitive navigation

## Future Enhancements (Out of Scope)

- Export kubeconfig for cluster access
- View cluster health metrics
- Backup/restore cluster state
- Multi-region view
- Cost estimation
- Tagging and filtering
- Batch operations (destroy all, scale all)
- Web UI for remote management

---

## Notes for Implementation

**READ THIS BEFORE STARTING**:

1. Follow the phased approach - each phase is independently testable
2. Write unit tests for deployment model first
3. Test menu navigation thoroughly before moving to list/detail
4. Use existing patterns (focusIndex, lipgloss styles, runner.runCommand)
5. Log everything to per-deployment log files
6. Never delete deployment records (mark as destroyed instead)
7. Test migration with a real legacy setup
8. Consider UX carefully - this will be used frequently

**Key Files to Reference**:
- `internal/tui/form.go` - Existing TUI patterns
- `internal/workflow/runner.go` - Workflow execution patterns
- `internal/model/config.go` - State management patterns

**Questions to Resolve Before Implementation**:
- [ ] Should we support renaming clusters after creation?
- [ ] Should we allow viewing destroyed clusters in the list?
- [ ] How long should we keep destroyed cluster records?
- [ ] Should edit trigger a full redeployment or just config update?
- [ ] Should scale down be allowed on single-node clusters?

---

**Status**: Awaiting approval to proceed with implementation.
**Next Step**: Review this document, answer questions, and approve plan.
