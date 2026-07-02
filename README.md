# Saddle

Saddle deploys RKE2, K3s, or single-node Docker Rancher clusters on AWS EC2. It provisions infrastructure with OpenTofu and configures nodes with Ansible, then hands you a running Rancher URL.

Built at SUSE by [Luiz Felipe Rosa](mailto:luiz.rosa@suse.com).

---

## Why use it

- **Zero boilerplate** — declare your cluster in YAML and apply it, just like kubectl.
- **Full TUI or full CLI** — every operation works interactively or headlessly in scripts and CI.
- **LLM-friendly** — generate a YAML blueprint and pipe it straight to `saddle apply -f`.
- **Rancher-native** — deploys standard Rancher or Rancher Prime, supports upgrades and hotfix image tags.

---

## Prerequisites

- Go 1.24+
- [`tofu`](https://opentofu.org/) in `PATH`
- [`ansible-playbook`](https://www.ansible.com/) in `PATH`
- An AWS account with EC2 permissions, an existing VPC subnet, a security group, and an EC2 key pair

---

## Install

```bash
git clone https://github.com/Felipalds/rancher-saddle.git
cd rancher-saddle
make build          # produces ./saddle
```

---

## Quick start

### TUI (interactive)

```bash
./saddle
```

The fullscreen TUI opens. Keybindings: `n` create cluster, `ctrl+x` credentials, `ctrl+p` profiles, `?` help.

### CLI

**1. Save credentials**

```bash
saddle apply -f - <<EOF
apiVersion: saddle/v1
kind: Credential
metadata:
  name: prod
spec:
  provider: aws
  accessKey: AKIAIOSFODNN7EXAMPLE
  secretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  region: us-west-2
EOF
```

**2. Apply a cluster**

```bash
saddle apply -f cluster.yaml
```

**3. Apply an LLM-generated blueprint**

```bash
saddle apply -f blueprint.yaml   # inline keys or --credentials-file to override
```

**4. Common operations**

```bash
saddle get clusters                          # list all clusters
saddle get cluster my-cluster                # describe one
saddle upgrade my-cluster --version 2.13.5  # upgrade Rancher
saddle delete cluster my-cluster            # destroy + remove
```

---

## Resource kinds

| Kind | Description |
|---|---|
| `Cluster` | Full RKE2 / K3s / Docker Rancher cluster on AWS EC2 |
| `Credential` | AWS access key / secret key pair |
| `Profile` | Reusable infrastructure settings (subnet, security group, SSH, etc.) |
| `AMI` | AMI catalog entry (distro × region → AMI ID) |

---

## Configuration files

| File | Purpose |
|---|---|
| `config.yaml` | Cluster state (auto-managed) |
| `cloud-credentials.yaml` | Saved credentials |
| `profiles.yaml` | Reusable profiles |
| `amis.yaml` | AMI catalog |

All files containing secrets use `0600` permissions. Never commit them.

---

## Further reading

- **[docs/cli.md](docs/cli.md)** — full CLI reference: every command, YAML schema, and example
- **[docs/architecture.md](docs/architecture.md)** — package layout, deploy/delete flows, interface contracts
- **[docs/product.md](docs/product.md)** — feature history and roadmap
