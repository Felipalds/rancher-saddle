# Feature: Custom Rancher Image Tag and Debug Mode

**Status:** Implemented

## What it does now

Two optional Rancher settings, wired end-to-end (config, templates, create form, upgrade form, upgrade runner):

- **Image Tag** (`rancher.image_tag`, YAML key `image_tag`) — when non-empty, adds
  `--set rancherImageTag=<value>` to the Helm install/upgrade so hotfix builds
  (e.g. `v2.13.3-hotfix-751b.1`) can be deployed. Empty by default (Helm then uses the
  version tag — unchanged behavior).
- **Debug** (`rancher.debug`, YAML key `debug`) — when true, adds `--debug=true` to the
  Helm command. Off by default.

Both apply to initial deploy and to upgrades, for RKE2 and K3s, Prime and non-Prime.

## Where it lives

- `internal/config/clusters.go` — `ImageTag` / `Debug` on `RancherSection`; passed through
  `ToModernConfig`/`FromModernConfig` as `rancher_image_tag` / `rancher_debug`.
- `internal/orchestrators/rke2/templates/addons.yml.tmpl`,
  `internal/orchestrators/k3s/templates/addons.yml.tmpl` — Helm flag injection.
- `internal/upgrade/templates/upgrade-rancher.yml.tmpl` — same flags on upgrade.
- `internal/tui/views/createform.go` (Image Tag field + Debug select),
  `internal/tui/views/upgradeform.go` (pre-filled from the cluster's current values).

## Example config

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
