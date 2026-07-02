# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Automated deployment tool for Kubernetes clusters (RKE2/K3s) or a single-node Docker Rancher on AWS EC2.
Tech stack: Go 1.24, Bubbletea TUI, Cobra CLI, OpenTofu (IaC), Ansible (config management).

The compiled binary is `saddle`. Running it with no args launches the fullscreen TUI; subcommands provide scriptable access.

## Build & Test

```bash
make build          # go build -o saddle
make test           # go test ./...
make test-cover     # go test -coverprofile=coverage.out + go tool cover -func
make test-verbose   # go test -v ./...
make lint           # go vet ./...
make clean          # remove binary + coverage.out

# Run a single test or package
go test ./internal/config -run TestValidate -v
go test ./internal/upgrade -run TestRunner -v
```

## CLI Surface (cmd/, main.go)

| Command | Purpose |
|---|---|
| `saddle` | Launch fullscreen TUI |
| `saddle list` | List clusters from `config.yaml` |
| `saddle create [name]` | TUI create form, then provision |
| `saddle delete <name> [-f]` | Destroy infra + remove from config |
| `saddle list-providers` | Dump registered providers |
| `saddle list-orchestrators` | Dump registered orchestrators |

`--config` flag (persistent) overrides default `config.yaml` path.

## Architecture

The system is built around a **Provider × Orchestrator registry** initialized in `main.go`:

- **Providers** generate cloud infrastructure (Terraform/OpenTofu). Only `aws` is implemented.
- **Orchestrators** install Kubernetes (or Docker Rancher) on top. `rke2`, `k3s`, and `docker` are registered.

Adding a new provider or orchestrator means: implement the interface in `internal/core/interfaces.go`, register in `main.go` `init()`, supply form fields + templates. The registry (`internal/core/registry.go`) is thread-safe and consulted at validate/deploy/upgrade time.

### Deployment pipeline (`internal/workflow/ModularRunner`)

```
TUI form → ClusterConfig (config.yaml)
   → Provider.GenerateInfrastructure() → clusters/<name>/main.tf
   → tofu init && tofu apply
   → Provider.GetOutputs() → IPs, DNS names
   → Orchestrator.GenerateInventory() → clusters/<name>/hosts.ini
   → Orchestrator.GeneratePlaybook() → clusters/<name>/site.yml
   → ansible-playbook site.yml -i hosts.ini
   → status="running", Rancher URL saved
```

For the `docker` orchestrator, the playbook installs Docker on a single node and runs Rancher as a container (no K8s).

### Upgrade pipeline (`internal/upgrade`)

K8s clusters: `upgrade.Runner` renders templates from `internal/upgrade/templates/` and runs `ansible-playbook` against the init node. Distribution determines the kubeconfig/kubectl paths inside the playbook.

Docker clusters: bypasses the Runner — `internal/tui/views/upgradeform.go` `runDockerUpgrade()` SSHs directly and runs `docker stop && docker rm && docker run` with the new image tag.

### TUI (`internal/tui`)

Bubbletea state machine in `root.go`; states defined in `views/messages.go`. The cluster list auto-refreshes every 1s. Long operations (deploy/delete/upgrade) run in background goroutines that write to `logs/<cluster>-<op>.log`; the footer streams the current log live.

## Package map

| Package | Purpose |
|---|---|
| `cmd/` | Cobra wiring + TUI launcher |
| `internal/cluster/` | CLI cluster commands (CREATE/LIST/DELETE) |
| `internal/config/` | YAML persistence: clusters, AMIs, profiles, validation, legacy bridge |
| `internal/core/` | `Provider` / `Orchestrator` interfaces, registry, shared types |
| `internal/credentials/` | AWS credential file management |
| `internal/docker/` | Docker Rancher helpers (deploy/delete/upgrade local containers) |
| `internal/generator/` | Go `text/template` renderer for Terraform + Ansible |
| `internal/model/` | TUI-side config model (loaded by `cmd.RunTUI`) |
| `internal/orchestrators/{rke2,k3s,docker}/` | Per-orchestrator playbook + inventory generation |
| `internal/orchestrators/shared/` | Shared Ansible task templates |
| `internal/providers/aws/` | AWS provider: Terraform generation, EC2 outputs |
| `internal/tui/` + `views/` | Bubbletea root + view components (lists, forms, modals) |
| `internal/upgrade/` | Rancher upgrade runner (Ansible-based, K8s only) |
| `internal/utils/` | Zap logger initialization |
| `internal/workflow/` | `ModularRunner` (deploy pipeline) |

## Config files (all mode 0600)

| File | Content |
|---|---|
| `config.yaml` | Cluster definitions with status, IPs, timestamps |
| `cloud-credentials.yaml` | AWS access/secret key pairs |
| `profiles.yaml` | Saved infrastructure profiles |
| `amis.yaml` | AMI catalog (distro × region → AMI ID) |

## TDD conventions

Write the failing test first, then implement. Style guide is `internal/config/validation_test.go`:

- Table-driven, `t.Run()` subtests
- `testify/assert` for assertions
- `t.TempDir()` for any file I/O
- No mocks for pure logic — only for `exec`/network boundaries
- Every new feature or bug fix ships with tests

## Feature workflow

1. Drop a proposal in `feats/<name>.md`
2. Implement with tests
3. Rename to `feats/x-(completed)-<name>.md` on completion

## Further reading

- `docs/architecture.md` — interface contracts, full deploy/delete flows
- `docs/product.md` — product decisions, features, version history
