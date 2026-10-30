# feats/ — feature index

Each file documents the **current behavior** of a feature (not its build history).
Every file starts with a `**Status:**` line: *Implemented* / *Partially implemented* /
*Proposed (not implemented)*.

Naming convention: an `x-(completed)-` filename prefix means the feature is fully
implemented and verified. Numbered/un-prefixed files are in progress or proposed — trust
the Status line inside.

| File | Status — purpose |
|---|---|
| `x-(completed)-tags-and-debug.md` | Implemented — optional Rancher image tag (hotfix builds) + Helm `--debug`. |
| `x-(completed)-rancher-docker.md` | Implemented — "Docker" install method: single EC2 instance running `docker run rancher/rancher`. |
| `x-(completed)-rancher-upgrade.md` | Implemented — in-place Rancher Helm upgrade (`u` in the cluster list). |
| `2-harvester-libvirt.md` | Partially implemented — local Harvester via libvirt ISO auto-install; backend + unit tests done, not verified end-to-end. |
| `4-dynamic-ami-lookup.md` | Proposed — resolve AWS AMIs at apply time via `data "aws_ami"` instead of hardcoded IDs. |
| `5-deployment-checkpoints.md` | Proposed — resumable deploys that retry from the last successful stage. |
| `rancher-tui-transformation.md` | Phase 1 implemented — overall rancher-tui roadmap (rename, dashboard, nvim keymap, registry) + later phases. |
| `harvester-unit-tests.md` | Harvester unit-test coverage plan (maintained by a separate effort). |
