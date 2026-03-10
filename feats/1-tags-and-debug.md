# Feature: Custom Image Tag and Debug Mode

## Summary

Add two optional Rancher deployment settings: a custom **image tag** (`rancherImageTag`) and a **debug mode** (`--debug=true`). The image tag enables deploying hotfix builds (e.g., `v2.13.3-hotfix-751b.1`) without changing the standard deployment flow. Debug mode toggles `--debug=true` on the Helm install for troubleshooting. Both fields are optional, rarely used, and default to off/empty.

---

## Motivation

Hotfix builds of Rancher are published with custom image tags (e.g., `v2.13.3-hotfix-751b.1`). To deploy them:

- **Prime**: `--set rancherImage=registry.suse.com/rancher/rancher --set rancherImageTag=v2.13.3-hotfix-751b.1 --create-namespace`
- **Standard (Latest)**: `--set rancherImageTag=v2.13.3-hotfix-751b.1 --create-namespace`

Additionally, both Prime and Standard deployments benefit from a `--debug=true` flag for troubleshooting Helm/Rancher issues.

Currently neither option is supported — the tool always deploys the tag matching `--version`. This feature adds both without complicating the common case (both are optional and hidden behind empty/false defaults).

---

## Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Image tag field name | `rancher_image_tag` | Matches Helm's `rancherImageTag`, clear purpose |
| Debug field name | `debug` | Simple boolean, maps directly to `--debug=true` |
| Default image tag | `""` (empty) | When empty, Helm uses the version tag — no change to current behavior |
| Default debug | `false` | Debug is opt-in |
| Scope | Deploy + Upgrade | Both initial install and upgrades should support hotfix tags and debug |
| TUI placement | After existing Rancher fields in create form and upgrade form | Low-priority fields go at the end |

---

## Files to Modify

| File | Action |
|---|---|
| `internal/config/clusters.go` | **MODIFY** — add `ImageTag string` and `Debug bool` to `RancherSection` |
| `internal/config/clusters.go` | **MODIFY** — pass new fields through `ToModernConfig` / `FromModernConfig` |
| `internal/orchestrators/rke2/templates/addons.yml.tmpl` | **MODIFY** — add `rancherImageTag` and `--debug` flags to Helm commands |
| `internal/orchestrators/k3s/templates/addons.yml.tmpl` | **MODIFY** — same changes for K3s |
| `internal/orchestrators/rke2/config.go` | **MODIFY** — pass new fields to template data |
| `internal/orchestrators/k3s/config.go` | **MODIFY** — pass new fields to template data |
| `internal/tui/views/createform.go` | **MODIFY** — add Image Tag and Debug fields to creation form |
| `internal/tui/views/upgradeform.go` | **MODIFY** — add Image Tag and Debug fields to upgrade form |
| `internal/upgrade/runner.go` | **MODIFY** — add fields to `UpgradeConfig`, pass to template |
| `internal/upgrade/templates/upgrade-rancher.yml.tmpl` | **MODIFY** — add `rancherImageTag` and `--debug` to upgrade playbook |

---

## Step 1 — Extend `RancherSection`

```go
type RancherSection struct {
    Version           string `yaml:"version"`
    Deploy            bool   `yaml:"deploy"`
    Prime             bool   `yaml:"prime"`
    BootstrapPassword string `yaml:"bootstrap_password"`
    AuditLog          bool   `yaml:"audit_log,omitempty"`
    AuditLogLevel     int    `yaml:"audit_log_level,omitempty"`
    ImageTag          string `yaml:"image_tag,omitempty"`
    Debug             bool   `yaml:"debug,omitempty"`
}
```

- `ImageTag`: when non-empty, adds `--set rancherImageTag=<value>` to Helm install/upgrade. For Prime, also ensures `--set rancherImage=registry.suse.com/rancher/rancher` is present (it already is in the Prime path).
- `Debug`: when true, adds `--debug=true` to Helm install/upgrade.

---

## Step 2 — Update Ansible Templates (RKE2 + K3s)

### Prime path (existing)

Add after existing `--set` flags:

```yaml
{% if rancher_image_tag %}    --set rancherImageTag={{ rancher_image_tag }}{% endif %}
{% if rancher_debug | bool %}    --debug=true{% endif %}
```

### Standard (Latest) path (existing)

Add after existing `--set` flags:

```yaml
{% if rancher_image_tag %}    --set rancherImageTag={{ rancher_image_tag }}{% endif %}
{% if rancher_debug | bool %}    --debug=true{% endif %}
```

Both RKE2 and K3s addons templates get the same additions.

---

## Step 3 — Pass Fields Through Orchestrator Configs

In `rke2/config.go` and `k3s/config.go`, add to template vars:

```go
"rancher_image_tag": cfg["rancher_image_tag"],
"rancher_debug":     cfg["rancher_debug"],
```

---

## Step 4 — Update `ToModernConfig` / `FromModernConfig`

```go
// ToModernConfig
cfg.OrchestratorConfig["rancher_image_tag"] = cc.Rancher.ImageTag
cfg.OrchestratorConfig["rancher_debug"] = cc.Rancher.Debug

// FromModernConfig
if v, ok := cfg.OrchestratorConfig["rancher_image_tag"].(string); ok {
    cc.Rancher.ImageTag = v
}
if v, ok := cfg.OrchestratorConfig["rancher_debug"].(bool); ok {
    cc.Rancher.Debug = v
}
```

---

## Step 5 — TUI Changes

### Create Form (`createform.go`)

Add two fields after the existing Rancher fields:

| # | Field | Type | Default |
|---|---|---|---|
| N | Image Tag (hotfix) | FieldText | `""` (empty) |
| N+1 | Debug Mode | FieldSelect (No / Yes) | `No` |

### Upgrade Form (`upgradeform.go`)

Add the same two fields, pre-filled from `cluster.Rancher.ImageTag` and `cluster.Rancher.Debug`.

---

## Step 6 — Upgrade Runner

Add to `UpgradeConfig`:

```go
ImageTag string
Debug    bool
```

Pass these to the upgrade template data so the upgrade playbook can conditionally add the flags.

---

## Helm Command Examples

### Standard deploy (no hotfix, no debug) — unchanged

```
helm upgrade --install rancher rancher-latest/rancher \
  --namespace cattle-system \
  --set hostname=... \
  --set bootstrapPassword=... \
  --set replicas=1 \
  --version 2.11.7 \
  --create-namespace
```

### Standard with hotfix tag

```
helm upgrade --install rancher rancher-latest/rancher \
  --namespace cattle-system \
  --set hostname=... \
  --set bootstrapPassword=... \
  --set rancherImageTag=v2.13.3-hotfix-751b.1 \
  --set replicas=1 \
  --version 2.11.7 \
  --create-namespace
```

### Prime with hotfix tag and debug

```
helm upgrade --install rancher rancher-prime/rancher \
  --namespace cattle-system \
  --set hostname=... \
  --set bootstrapPassword=... \
  --set rancherImage=registry.suse.com/rancher/rancher \
  --set rancherImageTag=v2.13.3-hotfix-751b.1 \
  --set replicas=1 \
  --version 2.11.7 \
  --create-namespace \
  --debug
```

---

## Config Example

```yaml
clusters:
  my-hotfix-cluster:
    rancher:
      version: "2.13.3"
      deploy: true
      prime: true
      bootstrap_password: admin
      image_tag: "v2.13.3-hotfix-751b.1"
      debug: true
```

---
Questions:
1. Why do we have different installs for k3s and rke2? Why we need to change both files (on the ansible)? Isnt there a way of having only one file for that?
2. We need to chang one little thing, the helm must be: --debug=true. IDK if only --debug works. Can you explain me that?
