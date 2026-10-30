# Feature: rancher-tui (overall transformation plan)

**Status:** Phase 1 implemented; later phases in progress / proposed.

Grows the project from an RKE2/K3s + Rancher-on-AWS deployer into **rancher-tui**: a
keyboard-first TUI for the Rancher product family (Rancher, Harvester, Longhorn,
observability, RKE2, K3s) across cloud and local environments. Ships in phases.

## Phase 1 — Foundation (implemented)

1. **Rename** to module `github.com/Felipalds/rancher-tui`, binary `rancher-tui`, cobra
   command `rancher-tui` (from the former rancher-saddle / corral names).
2. **Dashboard home screen** — `internal/tui/views/dashboard.go`, `StateDashboard`: ASCII
   banner, centered action menu, recent-cluster summary, nvim-hint footer. Initial state;
   `esc` returns here.
3. **nvim-style command engine** — `internal/tui/keymap/`: a pure-logic `Resolver` (no
   bubbletea dependency) for multi-key sequences, leader expansion (leader = space), timeout
   reset, and modes (normal / `:` / `/`), plus a which-key popup. Bindings include `j/k`,
   `gg`, `G`, `:`, `/`, `?`, `<leader>ff`, `<leader>q`. Additive — runs alongside existing
   per-view key handling.
4. **Provider/Orchestrator registry** — extensible registration in `main.go`.

All Phase 1 work is TDD-first, table-driven with testify.

## Later phases

- **Phase 2 — Harvester installs (in progress):** local Harvester via libvirt ISO
  auto-install is implemented but not yet end-to-end verified — see
  `feats/2-harvester-libvirt.md`. Cloud Harvester is not started.
- **Longhorn (proposed):** block-storage management (Helm/Ansible on an existing cluster).
- **Observability (proposed):** Rancher monitoring / Prometheus + Grafana.
- **Environments (proposed):** production vs. test presets.

## Deferred in Phase 1 (conscious)

- The legacy `internal/model.Config` vs `internal/config.Config` split, and the legacy
  `cmd/tui.go` model stacks, are left intact (the rename only touched imports).
- The on-disk directory stays `rancher-saddle/`; only the Go module path was renamed.

## Verification

```bash
make build   # produces ./rancher-tui
make test    # all packages green
./rancher-tui list-orchestrators   # rke2, k3s, docker, harvester
./rancher-tui list-providers       # aws, libvirt
```
