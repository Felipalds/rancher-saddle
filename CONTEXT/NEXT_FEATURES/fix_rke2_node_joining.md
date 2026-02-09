# Feature: Fix RKE2 Node Joining for High Availability

**Status**: 🚨 Critical Bug - Needs Immediate Fix
**Priority**: Critical
**Complexity**: Low (30 minutes implementation)
**Discovered**: 2026-02-09

## Problem Statement

**Critical Bug**: Additional nodes are NOT joining the RKE2 cluster, preventing High Availability.

### Current Broken Behavior
- First node initializes RKE2 successfully ✅
- Additional nodes install RKE2 but DO NOT join the cluster ❌
- Each node runs as an independent single-node cluster ❌
- No HA, no distributed etcd, no cluster redundancy ❌

### Impact
- Users think they have a 3-node HA cluster, but actually have 3 separate single-node clusters
- No etcd redundancy - single point of failure
- No control plane failover
- Defeats the entire purpose of multi-node deployments

## Root Cause Analysis

### File: `internal/generator/ansible.go` (Lines 48-73)

**Current Implementation**:
```yaml
- name: Join RKE2 Nodes
  hosts: join
  tasks:
    - name: Install RKE2 server (Join)
      shell: curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION="{{ rke2_version }}" sh -
      environment:
        RKE2_URL: "{{ server_url }}"      # ❌ WRONG: Install script ignores this
        RKE2_TOKEN: "{{ token }}"         # ❌ WRONG: Install script ignores this

    - name: Enable and start RKE2 Server
      systemd:
        name: rke2-server
        state: started
```

### Why This Fails

1. **The RKE2 install script (`get.rke2.io`) ONLY downloads binaries**
   - It installs files to `/usr/local/bin/`
   - It does NOT configure the cluster join settings

2. **Environment variables are IGNORED**
   - `RKE2_URL` and `RKE2_TOKEN` are NOT read by the install script
   - These variables have no effect whatsoever

3. **No config file is created**
   - RKE2 needs `/etc/rancher/rke2/config.yaml` to know how to join
   - Without this file, rke2-server starts a NEW cluster instead of joining

4. **Result: Multiple independent clusters**
   - Each node thinks it's the first node
   - Each node creates its own etcd database
   - No cluster communication happens

## Correct RKE2 HA Join Process

According to [RKE2 HA Documentation](https://docs.rke2.io/install/ha):

### For HA Server Nodes (Control Plane)

1. **Install RKE2 binaries**
   ```bash
   curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=v1.33.7+rke2r1 sh -
   ```

2. **Create config file** `/etc/rancher/rke2/config.yaml`:
   ```yaml
   server: https://<first-node-ip>:9345
   token: <node-token-from-first-node>
   ```

3. **Start rke2-server service**:
   ```bash
   systemctl enable rke2-server
   systemctl start rke2-server
   ```

**The config file MUST exist before starting the service!**

## Proposed Solution

### Updated Ansible Playbook

**Replace lines 48-73 in `internal/generator/ansible.go`**:

```yaml
- name: Join RKE2 Nodes
  hosts: join
  become: yes
  vars:
    rke2_version: "{{.RKE2Version}}"
    token: "{{ "{{" }} hostvars[groups['init'][0]]['rke2_token'] {{ "}}" }}"
    server_url: "{{ "{{" }} hostvars[groups['init'][0]]['rke2_url'] {{ "}}" }}"

  tasks:
    - name: Wait for cloud-init
      command: cloud-init status --wait
      changed_when: false

    - name: Install RKE2 binaries (Join)
      shell: curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION="{{ "{{" }} rke2_version {{ "}}" }}" sh -
      args:
        creates: /usr/local/bin/rke2

    - name: Create RKE2 config directory
      file:
        path: /etc/rancher/rke2
        state: directory
        mode: '0755'

    - name: Create RKE2 config file for joining
      copy:
        dest: /etc/rancher/rke2/config.yaml
        content: |
          server: {{ "{{" }} server_url {{ "}}" }}
          token: {{ "{{" }} token {{ "}}" }}
        mode: '0600'

    - name: Enable and start RKE2 Server
      systemd:
        name: rke2-server
        state: started
        enabled: yes

    - name: Wait for node to join cluster
      command: /var/lib/rancher/rke2/bin/kubectl --kubeconfig /etc/rancher/rke2/rke2.yaml get nodes
      register: nodes_check
      until: nodes_check.rc == 0
      retries: 30
      delay: 10
      changed_when: false
```

### Key Changes

| Change | Why |
|--------|-----|
| **Create config directory** | Ensures `/etc/rancher/rke2/` exists |
| **Create config.yaml** | Contains `server` and `token` - tells RKE2 to join |
| **Remove env variables** | They don't work, remove confusion |
| **Wait for join** | Verifies node actually joined successfully |

## Files to Modify

### `internal/generator/ansible.go`

**Location**: Lines 48-73 (Join RKE2 Nodes play)

**Changes**:
1. Add task: Create RKE2 config directory
2. Add task: Write config.yaml with server URL and token
3. Remove: Environment variables from install task
4. Add task: Verify node joined cluster

## Testing & Verification

### After Implementation

1. **Deploy a 3-node cluster**:
   ```bash
   ./go-kubernetes-helper
   # In TUI, set Instance Count: 3
   ```

2. **SSH to first node and check**:
   ```bash
   ssh ubuntu@<first-node-ip> -i <ssh-key>
   sudo /var/lib/rancher/rke2/bin/kubectl get nodes
   ```

3. **Expected output** (ALL 3 nodes should appear):
   ```
   NAME              STATUS   ROLES                       AGE   VERSION
   rancher-node-0    Ready    control-plane,etcd,master   10m   v1.33.7+rke2r1
   rancher-node-1    Ready    control-plane,etcd,master   8m    v1.33.7+rke2r1
   rancher-node-2    Ready    control-plane,etcd,master   6m    v1.33.7+rke2r1
   ```

4. **Check etcd members** (should have 3):
   ```bash
   sudo /var/lib/rancher/rke2/bin/kubectl -n kube-system get pods -l component=etcd
   ```

   **Expected**:
   ```
   NAME                         READY   STATUS    RESTARTS   AGE
   etcd-rancher-node-0          1/1     Running   0          10m
   etcd-rancher-node-1          1/1     Running   0          8m
   etcd-rancher-node-2          1/1     Running   0          6m
   ```

5. **Verify in Rancher UI**:
   - Navigate to Rancher dashboard
   - Go to Cluster Explorer → local cluster
   - Should see all 3 nodes listed

### Test HA Failover

1. Deploy 3-node cluster
2. Terminate one EC2 instance
3. Verify cluster still functional (2/3 etcd quorum)
4. Verify Rancher still accessible

## Benefits

After this fix:
- ✅ True High Availability with distributed etcd
- ✅ Control plane redundancy (failover capability)
- ✅ All nodes visible in cluster
- ✅ Proper etcd quorum (majority voting)
- ✅ Users get the HA cluster they expect

## Risk Assessment

**Risk Level**: Low

- Changes isolated to Ansible playbook
- Fix follows official RKE2 documentation
- Doesn't affect first node initialization
- Doesn't affect Rancher deployment

**Rollback**: Easy
- Keep backup of current ansible.go
- Revert if issues occur
- Manual fix possible on running nodes

## Timeline

| Phase | Duration |
|-------|----------|
| Planning | 30 minutes ✅ (complete) |
| Implementation | 30 minutes |
| Testing | 1 hour (full deployment) |
| **Total** | **2 hours** |

## Additional Improvements (Optional)

While fixing this, consider:

1. **Add config file to first node too**
   - For consistency
   - Makes future scaling easier
   - Simpler playbook logic

2. **Add node labels**
   - Label nodes with role (init/join)
   - Helps with troubleshooting
   - Good for scheduling workloads

3. **Verify etcd quorum**
   - After all nodes join
   - Ensure cluster is truly HA
   - Log etcd member count

## Related Documentation

- [RKE2 High Availability](https://docs.rke2.io/install/ha)
- [RKE2 Server Configuration](https://docs.rke2.io/install/configuration)
- [RKE2 Server Options Reference](https://docs.rke2.io/reference/server_config)

---

## Notes for Implementation

**CRITICAL**: The config file MUST be created BEFORE starting rke2-server service!

**Order matters**:
1. Install binaries ✅
2. Create config directory ✅
3. Write config.yaml with server + token ✅
4. Start service ✅
5. Verify join ✅

**Common mistakes to avoid**:
- ❌ Don't rely on environment variables
- ❌ Don't start service before config file exists
- ❌ Don't forget to verify the join was successful

---

**Status**: Ready for implementation
**Next Step**: Review this document, approve changes, and proceed with implementation
