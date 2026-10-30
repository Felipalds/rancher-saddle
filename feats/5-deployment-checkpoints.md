# Feature: Deployment Checkpoints and Resumable Retries

**Status:** Proposed (not implemented)

## Problem

`internal/workflow/runner_new.go` (`RunWithBuildDir`) runs the deploy as one sequential
pipeline (generate tofu → init → apply → outputs → playbook → inventory → wait SSH →
ansible). Any failure marks the cluster `failed` and the only recovery is delete + recreate,
re-provisioning EC2 instances that were already up. A trivial typo (e.g. wrong SSH key path)
costs minutes.

## Proposal

Persist which stage of a deploy completed so a failure can **resume from the last successful
stage** instead of starting over.

- Three user-facing stages — **Infrastructure → SSH → Provisioning** — backed by finer
  internal substages.
- New `DeploymentSection` on `ClusterConfig` (stage, status, hashes, timestamps, attempt
  count), persisted in `config.yaml`.
- Config-drift detection via per-stage hashes: changing the AMI invalidates from
  Infrastructure down; changing only the Rancher version invalidates only Provisioning.
- Runner refactor: `RunFrom(stage)` + `NextStage(...)` + per-stage methods; completion also
  gated on on-disk artifacts (tfstate, an ansible-done marker).
- TUI: new partial/failed statuses, an `R` (retry) keybinding, and a retry modal offering
  resume-from-stage.

## Not done

Nothing exists yet: no `DeploymentSection`, no `RunFrom`/`NextStage`, no
`internal/workflow/checkpoints.go`, no retry modal/keybinding. The runner is still a single
linear pipeline.
