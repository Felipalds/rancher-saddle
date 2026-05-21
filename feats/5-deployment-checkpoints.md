# Feature: Deployment Checkpoints and Resumable Retries

## Summary

Track which stages of a cluster deployment have completed so a failure midway through (SSH key typo, transient Ansible error, lost network) can be **resumed from the last successful stage** instead of redoing the 5–10 minute Terraform apply and SSH wait. The TUI offers a `R` (retry) action that picks up where the last attempt left off, with an option to force re-run from any earlier stage.

---

## Motivation

`internal/workflow/runner_new.go::RunWithBuildDir` runs nine sequential steps. They have very different costs and failure modes:

| # | Step | Typical cost | Common failure |
|---|---|---|---|
| 1 | Generate `main.tf` | <1 s | Template bug |
| 2 | `tofu init` | ~10 s | Network, wrong binary |
| 3 | `tofu apply` | 2–4 min | AWS quota, wrong AMI, key not registered |
| 4 | Fetch outputs (IPs/DNS) | ~2 s | tofu state corruption |
| 5 | Generate playbook | <1 s | Template bug |
| 6 | Generate inventory | <1 s | — |
| 7 | Wait for SSH | up to 5 min | Wrong user, missing key file, SG blocks 22 |
| 8 | `ansible-playbook site.yml` | 8–15 min | Mirror outage, helm chart fetch, RKE2 race |
| 9 | Display success | <1 s | — |

Today, **any failure throws everything away.** The cluster ends up in `status: failed`, and the only path forward is to delete + recreate. That deletes EC2 instances that took minutes to provision, only to provision them again seconds later. Every retry is a minutes-to-hours setback for a trivial typo.

Observed examples in this repo's logs:
- `logs/elemental4.log` — failed at step 7 after 5 minutes because the user typed the SSH key path without `.pem`. Steps 1–6 were perfect; everything has to be re-done.
- `logs/elemental.log` — failed at step 2 because `tofu` was the wrong binary. No EC2 instances were created, but the cluster was marked failed; retrying the same cluster row isn't possible today.

---

## Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Stage granularity | **Three user-facing stages** (Infrastructure, SSH, Provisioning) backed by finer internal substages | Matches the user's mental model ("redo ansible without redoing tofu"); finer detail helps debugging without cluttering the UI |
| State storage | New `Deployment` section on `ClusterConfig`, persisted in `config.yaml` | Survives across saddle restarts; reuses the existing config file (no new state DB) |
| What proves completion | Both an in-config marker AND an on-disk artifact check | Belt-and-suspenders — if the user nukes `clusters/<name>/`, we re-run; if they edit config.yaml, we re-run |
| Retry behavior | Default to "resume from last successful stage" with manual override | Saves time by default; user can force a clean run when they suspect drift |
| Config drift handling | Hash the relevant config sections per stage; invalidate downstream stages on mismatch | Editing the AMI invalidates Infrastructure (forces re-apply); editing only Rancher version invalidates only Provisioning |
| Idempotency assumption | `tofu apply` is naturally idempotent and is **always** re-run when we resume into Infrastructure; the checkpoint only skips it if Infrastructure is fully done | OpenTofu's "No changes" path is fast (~5 s); the safety is worth the small cost |
| UI surface | Add `R` keybinding to cluster list → modal showing last stage + retry options | One key, clear options; reuses the existing footer key pattern |
| Status column | New statuses: `failed-infrastructure`, `failed-ssh`, `failed-provisioning`, `partial` | Tells the user at a glance which stage failed without opening logs |

---

## Stage Model

### User-facing stages

```
┌────────────────────────┐    ┌────────────────────┐    ┌────────────────────────┐
│ 1. Infrastructure      │ →  │ 2. SSH Ready       │ →  │ 3. Provisioning        │
│  • tofu generate       │    │  • preflight key   │    │  • ansible-playbook    │
│  • tofu init           │    │  • wait for sshd   │    │                        │
│  • tofu apply          │    │                    │    │                        │
│  • cache outputs       │    │                    │    │                        │
│  • generate playbook   │    │                    │    │                        │
│  • generate inventory  │    │                    │    │                        │
└────────────────────────┘    └────────────────────┘    └────────────────────────┘
```

### Internal substages (for logs and forced re-runs)

| Stage | Substage IDs | Maps to current runner step |
|---|---|---|
| Infrastructure | `tf-generate`, `tf-init`, `tf-apply`, `tf-outputs`, `playbook-generate`, `inventory-generate` | 1–6 |
| SSH | `ssh-preflight`, `ssh-wait` | 7 |
| Provisioning | `ansible-playbook` | 8 |

`tf-init` is skipped on resume if `.terraform/` exists (since `tofu init` is the slow part). `tf-apply` is always re-run when entering the Infrastructure stage — it's the safety net that reconciles drift, and it's fast when there's nothing to do.

---

## Data Model

### New section on `ClusterConfig`

```go
type ClusterConfig struct {
    // ... existing fields ...
    Deployment DeploymentSection `yaml:"deployment,omitempty"`
}

type DeploymentSection struct {
    Stage         string    `yaml:"stage"`            // "infrastructure" | "ssh" | "provisioning" | "complete"
    StageStatus   string    `yaml:"stage_status"`     // "pending" | "running" | "completed" | "failed"
    LastSubstage  string    `yaml:"last_substage,omitempty"`
    StartedAt     time.Time `yaml:"started_at,omitempty"`
    CompletedAt   time.Time `yaml:"completed_at,omitempty"`
    AttemptCount  int       `yaml:"attempt_count,omitempty"`
    InfraHash     string    `yaml:"infra_hash,omitempty"`     // hash of provider+ssh+cluster config
    PlaybookHash  string    `yaml:"playbook_hash,omitempty"`  // hash of orchestrator+rancher config
}
```

### Example YAML after a half-completed deploy

```yaml
clusters:
  my-cluster:
    # ... existing fields ...
    deployment:
      stage: ssh
      stage_status: failed
      last_substage: ssh-wait
      started_at: 2026-05-20T19:37:41-03:00
      attempt_count: 1
      infra_hash: "a1b2c3..."     # infrastructure is done; resume from SSH
      playbook_hash: ""           # not yet attempted
```

### Drift invalidation

Before resuming, compute hashes from the **current** config and compare against the persisted ones:

| Hash | Inputs | Effect on mismatch |
|---|---|---|
| `infra_hash` | `Provider.Config`, `SSH.KeyName`, `Cluster.InstanceCount`, `Cluster.NodePrefix` | Reset to Infrastructure (forces fresh `tofu apply`) |
| `playbook_hash` | `Kubernetes.*`, `Rancher.*`, `SSH.User` | Reset Provisioning only (Infra/SSH still valid) |

This means: changing the AMI invalidates everything from Infra down; changing the Rancher version invalidates only Provisioning. The user never has to think about it.

---

## TUI Changes

### Cluster list

Add status icons for partial states (see `internal/tui/views/color_helper.go`):

| Status | Display |
|---|---|
| `running` (today) | `● running` (green) — unchanged |
| `creating` (today) | `⟳ creating` (blue) — unchanged |
| `failed-infrastructure` (new) | `✗ failed (infra)` (red) |
| `failed-ssh` (new) | `✗ failed (ssh)` (red) |
| `failed-provisioning` (new) | `✗ failed (ansible)` (red) |
| `partial` (new) | `◐ partial` (yellow) — infra up but provisioning never ran (e.g., saddle was killed) |

### New keybinding: `R` (retry)

Add `R` to `clusterListKeys` in `internal/tui/footer.go`. Pressing `R` on a failed/partial cluster opens a modal:

```
┌──────────────────────────────────────────────────────────────┐
│ Retry: my-cluster                                            │
│                                                              │
│ Last attempt failed at: SSH (ssh-wait, attempt #1)           │
│ Reason: ssh: ... Identity file ... not accessible            │
│                                                              │
│ Resume from:                                                 │
│  ▸ [1] SSH stage      ← default, picks up where we left off  │
│    [2] Provisioning stage (assume SSH is now OK)             │
│    [3] Infrastructure stage (full re-apply)                  │
│                                                              │
│ [enter] resume    [esc] cancel                               │
└──────────────────────────────────────────────────────────────┘
```

### Logs panel

The existing log panel (Enter on a cluster) already shows `logs/<name>.log` — that already contains stage markers like `=== Starting deployment for cluster: X ===`. Add finer markers:

```
[19:37:41] === Stage: Infrastructure (attempt 1) ===
[19:39:12] === Stage: Infrastructure ✓ (1m 31s) ===
[19:39:12] === Stage: SSH (attempt 1) ===
[19:44:12] === Stage: SSH ✗ failed after 5m 0s: ssh key not found at "..." ===
[19:50:18] === Stage: SSH (attempt 2) ===
[19:50:24] === Stage: SSH ✓ (6s) ===
[19:50:24] === Stage: Provisioning (attempt 1) ===
```

This makes the log self-documenting for resumed runs.

---

## Runner Changes

`RunWithBuildDir` becomes a thin shell that calls per-stage methods, each respecting a `resumeFrom` argument:

```go
type Stage string

const (
    StageInfrastructure Stage = "infrastructure"
    StageSSH            Stage = "ssh"
    StageProvisioning   Stage = "provisioning"
    StageComplete       Stage = "complete"
)

// RunFrom executes stages starting at `from` (inclusive). Reports progress
// via the optional onStageStart/onStageEnd callbacks so the TUI can update
// status as each stage transitions.
func (r *ModularRunner) RunFrom(buildDir string, from Stage) error {
    stages := []Stage{StageInfrastructure, StageSSH, StageProvisioning}
    skip := true
    for _, s := range stages {
        if s == from {
            skip = false
        }
        if skip {
            continue
        }
        if err := r.runStage(s, buildDir); err != nil {
            return &StageError{Stage: s, Err: err}
        }
    }
    return nil
}
```

`runStage` dispatches to the existing logic, but the SSH preflight from the previous fix becomes a substage with its own progress line.

### Resume helper

```go
// NextStage decides which stage to start from given the persisted state and
// the current cluster config. Returns StageInfrastructure if anything is
// inconsistent or if config drift invalidates previous progress.
func NextStage(cluster *config.ClusterConfig, current *config.Config) Stage {
    if cluster.Deployment.Stage == "" {
        return StageInfrastructure
    }
    if cluster.Deployment.InfraHash != hashInfra(current) {
        return StageInfrastructure
    }
    if cluster.Deployment.PlaybookHash != hashPlaybook(current) {
        return StageProvisioning // infra is fine, redo ansible
    }
    // Resume at the failed stage, or the next pending stage.
    switch cluster.Deployment.Stage {
    case "infrastructure":
        if cluster.Deployment.StageStatus == "completed" {
            return StageSSH
        }
        return StageInfrastructure
    case "ssh":
        if cluster.Deployment.StageStatus == "completed" {
            return StageProvisioning
        }
        return StageSSH
    case "provisioning":
        return StageProvisioning
    }
    return StageInfrastructure
}
```

### Stage completion artifact checks (belt + suspenders)

Before treating a stage as "previously completed", also verify the on-disk artifact still exists. If the user blew away `clusters/<name>/terraform.tfstate`, the Infrastructure stage gets re-run regardless of what `config.yaml` says.

| Stage | Artifact |
|---|---|
| Infrastructure | `clusters/<name>/terraform.tfstate` and non-empty `instance_ips` output |
| SSH | (no persistent artifact — always re-runs; preflight makes it cheap when already OK) |
| Provisioning | `clusters/<name>/.saddle-ansible-done` marker file written on success |

---

## Files to Modify

| File | Action |
|---|---|
| `internal/config/clusters.go` | **MODIFY** — add `DeploymentSection` + field |
| `internal/config/clusters_test.go` | **MODIFY** — extend round-trip test |
| `internal/workflow/runner_new.go` | **MODIFY** — split `RunWithBuildDir` into stages; add `RunFrom`, `NextStage`, `runStage`, hashing helpers |
| `internal/workflow/checkpoints.go` | **NEW** — hash functions, stage marker file helpers, drift detection |
| `internal/workflow/checkpoints_test.go` | **NEW** — table-driven tests for `NextStage` covering: fresh cluster, mid-stage failure, completed cluster, infra config drift, playbook config drift, missing tfstate |
| `internal/tui/views/color_helper.go` | **MODIFY** — add `failed-infrastructure`, `failed-ssh`, `failed-provisioning`, `partial` |
| `internal/tui/views/clusterlist.go` | **MODIFY** — handle `R` key, route to retry modal |
| `internal/tui/views/retrymodal.go` | **NEW** — modal showing last-failed-stage and resume options |
| `internal/tui/views/messages.go` | **MODIFY** — add `StateRetryModal`, `ClusterRetryMsg` |
| `internal/tui/footer.go` | **MODIFY** — add Retry to `clusterListKeys`; add `retryModalKeys` |
| `internal/tui/root.go` | **MODIFY** — wire `StateRetryModal` into the router |
| `internal/tui/views/createform.go::deployCluster` | **MODIFY** — call `runner.RunFrom(...)` instead of `RunWithBuildDir`; persist stage transitions to `config.yaml` |

---

## Step-by-Step Implementation Order (TDD)

1. **`internal/workflow/checkpoints.go` + `_test.go`** — pure logic for `NextStage`, `hashInfra`, `hashPlaybook`, artifact existence checks. No I/O dependency on the runner, easy to test exhaustively.
2. **`internal/config/clusters.go`** — add `DeploymentSection`. Round-trip test.
3. **`internal/workflow/runner_new.go`** — refactor `RunWithBuildDir` into `runInfraStage`, `runSSHStage`, `runProvisioningStage`. Add `RunFrom`. Each stage method updates `cluster.Deployment` and saves config on entry/exit.
4. **`internal/tui/views/color_helper.go`** — add the new statuses (visible feedback before the modal exists).
5. **`internal/tui/views/createform.go::deployCluster`** — use `RunFrom`; thread the cluster pointer in so stage transitions are persisted.
6. **`internal/tui/views/retrymodal.go` + state machine plumbing** — the user-facing retry flow.
7. **Manual test** end-to-end: create a cluster with a bad SSH key path (force SSH stage failure) → fix the path → press `R` → resume from SSH → cluster reaches running without re-applying tofu.

---

## Edge Cases & Open Questions

1. **What if the user changes the credential between attempts?** Credential isn't included in `infra_hash` today (would require reading the credentials file), so a credential swap won't trigger re-apply. Probably fine — tofu apply with new creds would either work (same account) or fail (different account, in which case the user wants a fresh start anyway). Acceptable for v1.
2. **Parallel retries on the same cluster** — what if the user spams `R`? Need a per-cluster lock. Easiest: refuse to start a new attempt while `stage_status == "running"`. Add a stale-attempt timeout (e.g., if `started_at` is older than 30 min and status is still `running`, treat as failed).
3. **What happens to `attempt_count` on success?** Reset to 0, or keep accumulating across deploys? Recommend: reset on full success so a future failure starts at attempt 1.
4. **`failed-provisioning` retry** — should it re-run from the start of ansible, or use ansible's own `--start-at-task`? Recommend: always full ansible re-run for v1. Tasks are mostly idempotent. `--start-at-task` is a v2 nice-to-have.
5. **Should `partial` clusters auto-resume on saddle startup?** No — that violates "actions taken visibly". The user has to press `R`.
6. **Migration of existing clusters in `config.yaml`** — they have no `deployment` block. Treat as `stage: complete` if `status == running`, otherwise as a fresh deploy on retry. No auto-rewrite needed.

---

## Pros

- **Trivial-typo failures cost seconds instead of minutes** — fix the SSH path, press `R`, watch SSH go green in 6 seconds, then ansible runs.
- **Discoverable failure stage** — the new status column tells you whether to look at AWS, your network, or your playbook.
- **Honest cost accounting** — `attempt_count` and per-stage timings in the log let the user see when they're stuck in a retry loop.
- **Cleanly extensible** — adding a "deploy add-on X" stage later is just another entry in the stages slice.

## Cons / Caveats

- **Increased state surface area.** Three new statuses, a new section in `config.yaml`, and drift hashes. Mitigated by keeping the model small (3 user-facing stages, two hashes).
- **Hash collisions could let stale infra pass through.** Using SHA-256 over a canonical yaml dump makes this functionally impossible for human-edited configs.
- **The legacy `Runner` in `internal/workflow/runner.go` is dead code but mirrors `ModularRunner`.** This feature targets only `ModularRunner`. The legacy file should be deleted in a separate cleanup PR; otherwise the divergence grows.
- **`tofu apply` on resume still talks to AWS** for plan reconciliation. Fast (~5 s for no-op) but not free.
- **Doesn't help if AWS itself rejects the instance launch** (e.g., quota, missing key pair). That fails in Infrastructure with a clear tofu error, and retry will hit the same wall until the underlying issue is fixed. The feature only helps when the *next* attempt would succeed.
