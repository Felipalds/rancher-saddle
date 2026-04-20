# Feature: Deploy Rancher via Docker on Cloud

## Summary

Added a new **"Installation Method"** field as the first field in the create form. Two options:

- **Local** (default) — the current full flow: AWS provider → RKE2/K3s → optional Rancher via Helm
- **Docker** — AWS provider → single EC2 instance → Docker + `docker run rancher/rancher`. No K8s orchestrator, no cert-manager, no Helm.

When **Docker** is selected, the form hides K8s-specific fields (Distribution, K8s Version, Deploy Rancher) and forces Instance Count to 1. All cloud/SSH fields remain visible since the EC2 instance is still provisioned via OpenTofu.

---

## Motivation

The current flow requires a full RKE2 or K3s cluster (3 nodes recommended) just to run Rancher. For many use cases this is overkill:

1. **Quick demos** — spin up Rancher in ~5 minutes on a single t3.medium instead of 15+ minutes on 3× t3.xlarge
2. **Development** — test Rancher UI, APIs, or downstream cluster provisioning without a full K8s cluster
3. **Cost** — one small instance vs. three large ones (~$0.04/hr vs ~$0.50/hr)
4. **Simplicity** — no cert-manager, no Helm, no kubeconfig wrangling

The `docker run rancher/rancher` method is [officially supported by Rancher](https://ranchermanager.docs.rancher.com/getting-started/installation-and-requirements/other-installation-methods/rancher-on-a-single-node-with-docker) and widely used for dev/test.

---

## Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Form entry point | New "Installation Method" field at index 0 | Clear top-level fork before any other config |
| Infrastructure | Same AWS provider, OpenTofu, SSH | Docker still runs on a cloud instance, not locally |
| Orchestrator | New `docker` orchestrator | Follows Provider/Orchestrator pattern; generates Ansible playbook |
| Deployment | Ansible playbook installs Docker + runs Rancher | Same pipeline as RKE2/K3s, just different playbook |
| Instance count | Forced to 1 | Docker Rancher is single-node |
| TLS | Rancher's built-in self-signed certs | No cert-manager needed |
| Persistent volume | Named Docker volume `rancher-data` | Survives container restarts |
| Privileged mode | `--privileged` flag | Required for Rancher's embedded K3s |
| Rancher Prime | Yes — `registry.suse.com/rancher/rancher` | Same Prime/Community toggle |
| Custom port | Host Port field (default 443) | Users can map to non-standard HTTPS port |
| Upgrade | SSH into instance, docker stop/rm/run | Simple, no Helm needed |
| Delete | Normal `tofu destroy` | EC2 instance destruction handles everything |

---

## Architecture

### New Docker Orchestrator

```
internal/orchestrators/docker/
├── orchestrator.go          # DockerOrchestrator implements core.Orchestrator
├── config.go                # DockerRancherConfig + FromMap()
├── templates/
│   ├── playbook.yml.tmpl    # Single-play Ansible playbook
│   └── install.yml.tmpl     # Docker install + Rancher run tasks
```

The orchestrator follows the exact same pattern as RKE2/K3s:
- `GeneratePlaybook()` renders the install template and embeds it into the playbook
- `GenerateInventory()` creates `hosts.ini` with `[init]` group only (single node, no `[join]`)
- Registered in `main.go` alongside RKE2 and K3s

### Docker Utility Package

```
internal/docker/
├── config.go       # DeployConfig struct (used by local Docker operations)
├── deploy.go       # BuildRunArgs(), DeployRancher(), DeleteRancher(), UpgradeRancher()
├── deploy_test.go  # Table-driven tests for arg building, image selection
```

This package is used for building Docker command args and can be used for local Docker operations. The cloud deployment uses Ansible instead.

---

## TUI Form: Two Modes

### Mode 1: Local (default — unchanged)

```
 [0]  Installation Method:   < Local >                  ◀ ▶
 [1]  Provider:              < AWS >                    ◀ ▶
 [2]  Credentials:           < my-aws-creds >           ◀ ▶
 [3]  K8s Distribution:      < RKE2 >                   ◀ ▶
 [4]  Cluster Name:          [my-cluster          ]
 [5]  Node Prefix:           [k8s-node            ]
     ... all 22 fields as before ...
[22]  Debug Mode:            < No >                     ◀ ▶
      [ Apply ]
```

### Mode 2: Docker (3 fields hidden, 1 new field visible)

```
 [0]  Installation Method:   < Docker >                 ◀ ▶
 [1]  Provider:              < AWS >                    ◀ ▶
 [2]  Credentials:           < my-aws-creds >           ◀ ▶
 [3]  K8s Distribution:      ────────────────────────    HIDDEN
 [4]  Cluster Name:          [my-rancher          ]
 [5]  Node Prefix:           [rancher-docker      ]
 [6]  Region:                [us-east-1           ]
 [7]  Subnet ID:             [subnet-xxxxx        ]
 [8]  Security Group ID:     [sg-xxxxx            ]
 [9]  OS Image:              < Ubuntu 22.04 LTS >       ◀ ▶
[11]  Instance Type:         [t3.medium           ]
[12]  Instance Count:        [1                   ]     ← forced to 1
[13]  SSH Key Name:          [my-key              ]
[14]  SSH Private Key Path:  [~/.ssh/my-key.pem   ]
[15]  SSH User:              [ubuntu              ]
[16]  K8s Version:           ────────────────────────    HIDDEN
[17]  Deploy Rancher:        ────────────────────────    HIDDEN (always true)
[18]  Rancher Prime:         < No >                     ◀ ▶
[19]  Rancher Version:       [2.11.7              ]
[20]  Bootstrap Password:    [admin               ]
[21]  Image Tag (hotfix):    [                    ]
[22]  Debug Mode:            < No >                     ◀ ▶
[23]  Host Port (HTTPS):     [443                 ]     ← Docker-only

      ⚠ Docker install is for dev/test only.
      [ Apply ]
```

**Hidden in Docker mode:** K8s Distribution (3), K8s Version (16), Deploy Rancher (17)
**New in Docker mode:** Host Port (23)
**Forced in Docker mode:** Instance Count = 1, Deploy Rancher = true

---

## Deployment Flow

```
TUI Create Form (Docker mode)
    → AWS Provider: GenerateInfrastructure → main.tf (1 instance)
    → tofu init && tofu apply → 1 EC2 instance
    → Provider.GetOutputs → IP, DNS
    → Docker Orchestrator: GenerateInventory → hosts.ini [init] only
    → Docker Orchestrator: GeneratePlaybook → site.yml
        Play: "Deploy Rancher via Docker"
        Tasks:
          1. Wait for cloud-init
          2. Install Docker (get.docker.com)
          3. Start Docker systemd
          4. docker run rancher/rancher (Prime or Community)
          5. Health check (https://127.0.0.1/healthz)
    → ansible-playbook -i hosts.ini site.yml
    → Status → "running", Rancher URL saved
```

### Delete Flow

Same as RKE2/K3s: `tofu destroy` removes the EC2 instance. The Docker container dies with it.

### Upgrade Flow

SSH into the EC2 instance and run:
```
docker stop rancher && docker rm rancher && docker run -d ... <new-image>
```
The existing `rancher-data` volume preserves state across upgrades.

---

## Files Created

| File | Purpose |
|---|---|
| `internal/orchestrators/docker/orchestrator.go` | Docker orchestrator (core.Orchestrator) |
| `internal/orchestrators/docker/config.go` | DockerRancherConfig + FromMap() |
| `internal/orchestrators/docker/templates/playbook.yml.tmpl` | Ansible playbook |
| `internal/orchestrators/docker/templates/install.yml.tmpl` | Docker install + Rancher tasks |
| `internal/docker/config.go` | DeployConfig utility struct |
| `internal/docker/deploy.go` | Docker command builders + lifecycle functions |
| `internal/docker/deploy_test.go` | Tests for arg building, image selection |

## Files Modified

| File | Change |
|---|---|
| `internal/core/types.go` | Added `OrchestratorDocker` constant |
| `main.go` | Registered docker orchestrator |
| `internal/tui/views/createform.go` | Installation Method field, Host Port field, visibility logic, Docker submit path |
| `internal/tui/views/upgradeform.go` | Docker upgrade via SSH (docker stop/rm/run) |

---

## Comparison: Docker vs Local (RKE2/K3s)

| Aspect | Docker (new) | Local (existing) |
|---|---|---|
| Infrastructure | 1 EC2 instance | 3 EC2 instances (HA) |
| Instance type default | t3.medium | t3.xlarge |
| Deploy time | ~5 min | ~15-20 min |
| Est. cost (us-east-1) | ~$0.04/hr | ~$0.50/hr |
| K8s cluster | Embedded (inside Rancher) | Full external cluster |
| cert-manager | Not needed | Required |
| Helm | Not needed | Required |
| HA / production | No | Yes |
| Upgrade | SSH + docker stop/rm/run | Ansible + helm upgrade |
| Delete | tofu destroy | tofu destroy |
| Official support | Dev/test only | Production supported |
