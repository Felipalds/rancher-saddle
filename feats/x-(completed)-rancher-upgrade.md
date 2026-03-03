# Feature: Rancher Upgrade

## Summary

Add the ability to upgrade Rancher on any running cluster that has Rancher deployed, directly from the TUI. The upgrade re-runs only the Helm layer — no infrastructure reprovisioning and no full Ansible playbook. It is triggered with `u` from the cluster list, is blocked for clusters without Rancher deployed, and persists the new version back to `config.yaml`.

---

## Motivation

`helm upgrade --install` is idempotent, so a targeted Ansible playbook against the first node is the minimal and safest upgrade path. The full deployment workflow (`ModularRunner`) is not reused — it would recreate infrastructure and re-run the entire Kubernetes bootstrap, which is not what we want.

---

## Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Where to run | SSH → first node (init) | All Helm/kubectl tools already exist there |
| Execution method | Ansible playbook (same as addons) | Consistent with existing pattern; idempotent |
| Template location | `internal/upgrade/templates/` | Cross-distro — not owned by rke2 or k3s packages |
| Inventory | Minimal single-host (init only) | Upgrade only touches the control plane node |
| Prime detection | Read from `config.yaml` `rancher.prime` | Source of truth is what was deployed |
| K8s distro detection | Read from `config.yaml` `kubernetes.distribution` | Determines kubectl/kubeconfig paths |
| Config persistence | Update `RancherSection` in `config.yaml` after upgrade | Keep state consistent |

---

## Files to Create / Modify

| File | Action |
|---|---|
| `internal/upgrade/runner.go` | **CREATE** — upgrade runner |
| `internal/upgrade/templates/upgrade-rancher.yml.tmpl` | **CREATE** — Ansible upgrade playbook |
| `internal/upgrade/templates/upgrade-inventory.ini.tmpl` | **CREATE** — single-node inventory |
| `internal/config/clusters.go` | **MODIFY** — extend `RancherSection` |
| `internal/tui/views/upgradeform.go` | **CREATE** — TUI upgrade form |
| `internal/tui/views/messages.go` | **MODIFY** — add `StateUpgradeForm` |
| `internal/tui/root.go` | **MODIFY** — wire new state |
| `internal/tui/views/clusterlist.go` | **MODIFY** — add `u` key |
| `internal/tui/footer.go` | **MODIFY** — add upgrade keybinding |

---

## Step 1 — Extend `RancherSection` in `internal/config/clusters.go`

Add two new fields to persist audit log configuration alongside the existing Rancher fields:

```go
type RancherSection struct {
    Version           string `yaml:"version"`
    Deploy            bool   `yaml:"deploy"`
    Prime             bool   `yaml:"prime"`
    BootstrapPassword string `yaml:"bootstrap_password"`
    AuditLog          bool   `yaml:"audit_log,omitempty"`
    AuditLogLevel     int    `yaml:"audit_log_level,omitempty"`
}
```

These fields are written back to `config.yaml` after every successful upgrade so the TUI can pre-fill them on the next run.

---

## Step 2 — Upgrade Runner (`internal/upgrade/runner.go`)

```go
package upgrade

type UpgradeConfig struct {
    ClusterName       string
    Distribution      string   // "rke2" or "k3s"
    InstanceIPs       []string // first IP = init node
    SSHPrivateKeyPath string
    SSHUser           string
    RancherVersion    string
    RancherHostname   string
    BootstrapPassword string
    Prime             bool
    Replicas          int
    AuditLog          bool
    AuditLogLevel     int
    BuildDir          string
}

type RancherUpgradeRunner struct {
    Config UpgradeConfig
}

func NewRancherUpgradeRunner(cfg UpgradeConfig) *RancherUpgradeRunner

// Run orchestrates: generate files → run ansible → return error
func (r *RancherUpgradeRunner) Run() error
func (r *RancherUpgradeRunner) generateInventory(dir string) error
func (r *RancherUpgradeRunner) generatePlaybook(dir string) error
func (r *RancherUpgradeRunner) runAnsible(dir string) error
```

`Run()` flow:
1. Create a temp work directory under `clusters/<name>/upgrade/`
2. Render `upgrade-inventory.ini.tmpl` → `hosts.ini` (single init node)
3. Render `upgrade-rancher.yml.tmpl` → `upgrade.yml`
4. Execute `ansible-playbook -i hosts.ini upgrade.yml`
5. Return error on failure

---

## Step 3 — Ansible Inventory Template (`upgrade-inventory.ini.tmpl`)

Minimal inventory with only the first node:

```ini
[init]
{{ index .InstanceIPs 0 }} ansible_user={{ .SSHUser }} ansible_ssh_private_key_file={{ .SSHPrivateKeyPath }} ansible_ssh_common_args='-o StrictHostKeyChecking=no'

[all:vars]
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

---

## Step 4 — Ansible Upgrade Playbook Template (`upgrade-rancher.yml.tmpl`)

Key design points:
- Targets only `init` hosts
- Two conditional branches: Prime and non-Prime
- K8s distro determines `kubectl` and `kubeconfig` paths
- Adds audit log flags only when `audit_log` is true
- Waits for the rollout to complete before returning

```yaml
---
- name: Upgrade Rancher
  hosts: init
  become: true
  vars:
    rancher_version: "{{ .RancherVersion }}"
    rancher_hostname: "{{ .RancherHostname }}"
    bootstrap_password: "{{ .BootstrapPassword }}"
    replicas: {{ .Replicas }}
    rancher_prime: {{ .Prime }}
    audit_log: {{ .AuditLog }}
    audit_log_level: {{ .AuditLogLevel }}
    {{- if eq .Distribution "rke2" }}
    kubectl: /var/lib/rancher/rke2/bin/kubectl
    kubeconfig: /etc/rancher/rke2/rke2.yaml
    {{- else }}
    kubectl: /usr/local/bin/kubectl
    kubeconfig: /etc/rancher/k3s/k3s.yaml
    {{- end }}

  tasks:
    - name: Add Rancher Prime Helm repo
      command: /usr/local/bin/helm repo add rancher-prime https://charts.rancher.com/server-charts/prime
      when: rancher_prime | bool

    - name: Add Rancher Latest Helm repo
      command: /usr/local/bin/helm repo add rancher-latest https://releases.rancher.com/server-charts/latest
      when: not (rancher_prime | bool)

    - name: Update Helm repos
      command: /usr/local/bin/helm repo update

    - name: Upgrade Rancher (Prime)
      command: >
        /usr/local/bin/helm upgrade --install rancher rancher-prime/rancher
        --namespace cattle-system
        --set hostname={{ rancher_hostname }}
        --set replicas={{ replicas }}
        --set bootstrapPassword={{ bootstrap_password }}
        --set auditLog.enabled={{ audit_log }}
        --set auditLog.level={{ audit_log_level }}
        --set rancherImage=registry.suse.com/rancher/rancher
        --set "extraEnv[0].name=CATTLE_DEBUG"
        --set "extraEnv[0].value=true"
        --set "extraEnv[1].name=RANCHER_VERSION_TYPE"
        --set "extraEnv[1].value=prime"
        --set "extraEnv[2].name=CATTLE_BASE_UI_BRAND"
        --set "extraEnv[2].value=suse"
        --version {{ rancher_version }}
        --kubeconfig {{ kubeconfig }}
        --create-namespace
      when: rancher_prime | bool

    - name: Upgrade Rancher (Latest)
      command: >
        /usr/local/bin/helm upgrade --install rancher rancher-latest/rancher
        --namespace cattle-system
        --set hostname={{ rancher_hostname }}
        --set replicas={{ replicas }}
        --set bootstrapPassword={{ bootstrap_password }}
        --set auditLog.enabled={{ audit_log }}
        --set auditLog.level={{ audit_log_level }}
        --version {{ rancher_version }}
        --kubeconfig {{ kubeconfig }}
        --create-namespace
      when: not (rancher_prime | bool)

    - name: Wait for Rancher rollout
      command: >
        {{ kubectl }} rollout status deploy/rancher
        -n cattle-system --timeout=600s
        --kubeconfig {{ kubeconfig }}
```

---

## Step 5 — TUI: `StateUpgradeForm`

### `internal/tui/views/messages.go`
Add `StateUpgradeForm` to the state enum (before `StateHelp`).

### `internal/tui/views/upgradeform.go`

Form fields (pre-filled from `ClusterConfig`):

| # | Field | Type | Pre-filled from |
|---|---|---|---|
| 0 | Target Version | FieldText | `rancher.version` |
| 1 | Replicas | FieldText | `"1"` |
| 2 | Audit Log | FieldSelect (No / Yes) | `rancher.audit_log` |
| 3 | Audit Log Level | FieldText | `rancher.audit_log_level` |

The form receives the cluster name via `Data` on `StateChangeMsg`. On submit it:
1. Emits a `rancherUpgradeStartedMsg` that carries the `UpgradeConfig`
2. Spawns a goroutine running `RancherUpgradeRunner.Run()`
3. On completion writes a `rancherUpgradeDoneMsg` (success) or `rancherUpgradeErrorMsg`
4. Updates `config.yaml` with the new version and audit log settings
5. Transitions back to `StateClusterList`

### `internal/tui/root.go`
- Add `upgradeForm views.UpgradeFormModel` field to `RootModel`
- In `StateChangeMsg` handler, when `NewState == StateUpgradeForm`, populate the form with the cluster config from `Data.(string)` (cluster name)
- Add `StateUpgradeForm` cases to `routeUpdate` and `View`

### `internal/tui/views/clusterlist.go`
```go
case "u":
    if len(m.clusterNames) > 0 {
        row := m.table.Cursor()
        if row < len(m.clusterNames) {
            clusterName := m.clusterNames[row]
            cluster := m.clusters[clusterName]
            if cluster.Rancher.Deploy {
                return m, func() tea.Msg {
                    return StateChangeMsg{
                        NewState: StateUpgradeForm,
                        Data:     clusterName,
                    }
                }
            }
        }
    }
```

### `internal/tui/footer.go`
Add `upgradeFormKeys` keymap and wire it in `ViewForState`. Add `u upgrade` to `clusterListKeys` (e.g., extend the `New` or `Refresh` binding, or add a dedicated field to `keyMap`).

---

## Data Flow

```
Cluster List (u key, rancher.deploy == true)
    │
    ▼
StateUpgradeForm  ←── pre-filled with cluster's current rancher config
    │  (submit)
    ▼
goroutine: RancherUpgradeRunner.Run()
    ├─ Render upgrade-inventory.ini.tmpl  →  clusters/<name>/upgrade/hosts.ini
    ├─ Render upgrade-rancher.yml.tmpl    →  clusters/<name>/upgrade/upgrade.yml
    └─ ansible-playbook -i hosts.ini upgrade.yml
           │
           ├─ helm repo add (prime or latest)
           ├─ helm upgrade --install rancher ...
           └─ kubectl rollout status deploy/rancher
    │
    ▼
Update config.yaml:  rancher.version, rancher.audit_log, rancher.audit_log_level
    │
    ▼
StateClusterList
```

---

## Prime vs. Non-Prime Differences

| Setting | Rancher Prime | Rancher (Latest) |
|---|---|---|
| Helm repo name | `rancher-prime` | `rancher-latest` |
| Helm repo URL | `https://charts.rancher.com/server-charts/prime` | `https://releases.rancher.com/server-charts/latest` |
| Chart reference | `rancher-prime/rancher` | `rancher-latest/rancher` |
| `rancherImage` | `registry.suse.com/rancher/rancher` | *(not set)* |
| `extraEnv` | `CATTLE_DEBUG=true`, `RANCHER_VERSION_TYPE=prime`, `CATTLE_BASE_UI_BRAND=suse` | *(not set)* |

---

## Constraints and Guards

- The `u` key is **silently ignored** when the selected cluster has `rancher.deploy = false`
- The `u` key is **silently ignored** when `status != "running"` (avoid upgrading a cluster that is still being created or is deleting)
- Upgrade progress is written to `logs/<cluster-name>-upgrade.log` (same log mechanism as deployment)
- The cluster status is set to `"upgrading"` for the duration, then back to `"running"` on success or `"upgrade-failed"` on error
