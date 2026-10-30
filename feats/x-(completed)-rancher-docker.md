# Feature: Deploy Rancher via Docker on Cloud

**Status:** Implemented

## What it does now

Adds an **Installation Method** choice at the top of the create form:

- **Local** (default) — AWS provider → RKE2/K3s cluster → optional Rancher via Helm.
- **Docker** — AWS provider → a single EC2 instance → Docker + `docker run rancher/rancher`.
  No Kubernetes orchestrator, cert-manager, or Helm.

In Docker mode the form hides the K8s-specific fields (Distribution, K8s Version, Deploy
Rancher), forces instance count to 1, and shows a Docker-only **Host Port (HTTPS)** field.
All cloud/SSH fields stay visible (the EC2 instance is still provisioned via OpenTofu).

Deploy uses the same pipeline as RKE2/K3s but with the `docker` orchestrator: it provisions
one EC2 instance, generates a single-node `hosts.ini` (`[init]` only), and runs an Ansible
playbook that installs Docker and starts the Rancher container (Prime or Community, with a
persistent `rancher-data` volume and `--privileged` for the embedded K3s). Delete is a normal
`tofu destroy`. Upgrade SSHes into the instance and runs `docker stop/rm/run` with the new image.

## Where it lives

- `internal/orchestrators/docker/` — orchestrator, config, `playbook.yml.tmpl` /
  `install.yml.tmpl`. Registered in `main.go` (`dockerorch`); `core.OrchestratorDocker = "docker"`.
- `internal/docker/` — Docker command builders + lifecycle helpers (`deploy.go`) with tests.
- `internal/tui/views/createform.go` — Installation Method + Host Port fields, Docker
  visibility logic (`isDockerMode`), Docker submit path.
- `internal/tui/views/upgradeform.go` — `runDockerUpgrade` (SSH + docker stop/rm/run).

Docker install is intended for dev/test only (single node, self-signed certs).
