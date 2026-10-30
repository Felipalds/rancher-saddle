# Feature: Rancher Upgrade

**Status:** Implemented

## What it does now

Upgrades Rancher on any running cluster that has Rancher deployed, from the TUI. It re-runs
only the Helm layer — no infrastructure reprovisioning, no full Ansible playbook. Triggered
with `u` in the cluster list; ignored when the cluster has no Rancher or isn't `running`.
The new version is persisted back to `config.yaml`.

Mechanism: an Ansible playbook targeting the first (init) node over SSH runs
`helm upgrade --install rancher ...` and waits for the rollout. Prime vs non-Prime (repo,
chart, `rancherImage`, `extraEnv`) and the K8s distro (kubectl/kubeconfig paths for
RKE2 vs K3s) are read from `config.yaml`. Honors the audit-log and image-tag/debug settings.
Docker-based Rancher clusters upgrade via `docker stop/rm/run` instead (see
`feats/x-(completed)-rancher-docker.md`).

## Where it lives

- `internal/upgrade/runner.go` — `RancherUpgradeRunner` (generate inventory + playbook,
  run ansible).
- `internal/upgrade/templates/upgrade-rancher.yml.tmpl`, `upgrade-inventory.ini.tmpl`.
- `internal/tui/views/upgradeform.go` (`StateUpgradeForm`), `clusterlist.go` (`u` key),
  `footer.go`, `root.go` wiring.
- `internal/config/clusters.go` — `RancherSection` audit-log fields persisted after upgrade.

Progress is written to `logs/<cluster>-upgrade.log`; status shows `upgrading` →
`running` / `upgrade-failed`.
