# CLI Reference

`saddle` is the compiled binary. Running it with no arguments opens the interactive TUI. Every operation available in the TUI is also reachable from the CLI using three verbs: **apply**, **get**, **delete**.

## Global flags

| Flag | Default | Description |
|---|---|---|
| `--config` | `config.yaml` | Clusters state file |
| `--credentials-file` | `cloud-credentials.yaml` | Credentials file |

---

## Resource schema

Every YAML file uses the same envelope:

```yaml
apiVersion: saddle/v1
kind: <Kind>          # Cluster | Credential | Profile | AMI
metadata:
  name: <name>
spec:
  ...
```

Multiple resources can live in one file, separated by `---`. `saddle apply -f` processes them in order.

---

## `saddle apply -f <file.yaml>`

Create or update any resource. Dispatches by `kind`.

```bash
saddle apply -f cluster.yaml
saddle apply -f credentials.yaml
saddle apply -f everything.yaml    # multi-doc file
```

| Flag | Description |
|---|---|
| `-f`, `--file` | Path to YAML resource file **(required)** |

---

### Kind: Cluster

Provisions a full RKE2 / K3s / Docker Rancher cluster on AWS EC2.

Credentials can be referenced by name (from the credentials file) or embedded inline. Inline `accessKey`/`secretKey` always override a named credential.

```yaml
apiVersion: saddle/v1
kind: Cluster
metadata:
  name: repro-sure-11610
spec:
  method: local               # local | docker  (default: local)
  provider: aws
  credential: prod            # name from credentials file (optional)
  accessKey: ""               # inline — overrides credential
  secretKey: ""               # inline — overrides credential
  region: us-west-2
  subnetId: subnet-0abc12345
  securityGroupId: sg-0abc12345
  ami: ami-0a3e3ef8596692376
  instanceType: t3.xlarge     # default: t3.xlarge
  instanceCount: 3            # default: 3; forced to 1 in docker mode
  nodePrefix: sure-11610      # default: k8s-node
  kubernetes:
    distribution: rke2        # rke2 | k3s | docker  (default: rke2)
    version: v1.33.7+rke2r1   # default: latest stable
  ssh:
    keyName: my-key
    privateKeyPath: ~/.ssh/my-key.pem
    user: ubuntu              # default: ubuntu
  rancher:
    enabled: true
    version: 2.13.5           # default: 2.11.7
    prime: false
    bootstrapPassword: admin  # default: admin
    imageTag: ""              # hotfix override, e.g. v0.0.0-hotfix-abc.1
    debug: false
```

**LLM-generated blueprint** — `credential` can be `""` with inline keys for self-contained files:

```yaml
apiVersion: saddle/v1
kind: Cluster
metadata:
  name: repro-sure-11610
spec:
  accessKey: PLACEHOLDER_ACCESS_KEY
  secretKey: PLACEHOLDER_SECRET_KEY
  region: us-west-2
  subnetId: subnet-0abc12345
  securityGroupId: sg-0abc12345
  ami: ami-0a3e3ef8596692376
  instanceType: t3.xlarge
  instanceCount: 3
  nodePrefix: sure-11610
  kubernetes:
    distribution: rke2
    version: v1.33.7+rke2r1
  ssh:
    keyName: my-key
    privateKeyPath: ~/.ssh/my-key.pem
    user: ubuntu
  rancher:
    enabled: true
    version: 2.13.5
    bootstrapPassword: admin
```

---

### Kind: Credential

Saves AWS credentials to `cloud-credentials.yaml` (or `--credentials-file`). Applying an existing name replaces it.

```yaml
apiVersion: saddle/v1
kind: Credential
metadata:
  name: prod
spec:
  provider: aws
  accessKey: AKIAIOSFODNN7EXAMPLE
  secretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  region: us-east-1
```

---

### Kind: Profile

Saves a reusable infrastructure profile to `profiles.yaml`. Profiles can be loaded in the TUI create form with `ctrl+p`.

```yaml
apiVersion: saddle/v1
kind: Profile
metadata:
  name: us-east
spec:
  region: us-east-1
  subnetId: subnet-0abc12345
  securityGroupId: sg-0abc12345
  ami: ami-0c7217cdde317cfec
  instanceType: t3.xlarge
  ssh:
    keyName: my-key
    privateKeyPath: ~/.ssh/my-key.pem
    user: ubuntu
```

---

### Kind: AMI

Adds an entry to the AMI catalog (`amis.yaml`). The `metadata.name` is the identifier used in `saddle get ami` and `saddle delete ami`.

```yaml
apiVersion: saddle/v1
kind: AMI
metadata:
  name: ubuntu-22-us-east-1
spec:
  distro: "Ubuntu 22.04 LTS"
  region: us-east-1
  amiId: ami-0c7217cdde317cfec
```

---

### Multi-resource file

```yaml
apiVersion: saddle/v1
kind: Credential
metadata:
  name: prod
spec:
  provider: aws
  accessKey: AKIAIOSFODNN7EXAMPLE
  secretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  region: us-west-2
---
apiVersion: saddle/v1
kind: Cluster
metadata:
  name: repro-sure-11610
spec:
  credential: prod
  region: us-west-2
  subnetId: subnet-0abc12345
  securityGroupId: sg-0abc12345
  ami: ami-0a3e3ef8596692376
  instanceType: t3.xlarge
  instanceCount: 3
  nodePrefix: sure-11610
  kubernetes:
    distribution: rke2
    version: v1.33.7+rke2r1
  ssh:
    keyName: my-key
    privateKeyPath: ~/.ssh/my-key.pem
    user: ubuntu
  rancher:
    enabled: true
    version: 2.13.5
    bootstrapPassword: admin
```

---

## `saddle get <kind> [name]`

List all resources of a kind, or describe one by name. Kind is case-insensitive and accepts singular or plural.

```bash
saddle get clusters
saddle get cluster repro-sure-11610

saddle get credentials
saddle get credential prod

saddle get profiles
saddle get profile us-east

saddle get amis
saddle get ami ubuntu-22-us-east-1
```

**Cluster status values:** `creating`, `running`, `upgrading`, `deleting`, `failed`, `upgrade-failed`

---

## `saddle delete <kind> <name>`

Delete a resource by kind and name. For clusters, also destroys AWS infrastructure via `tofu destroy`.

```bash
saddle delete cluster repro-sure-11610
saddle delete credential prod --force
saddle delete profile us-east
saddle delete ami ubuntu-22-us-east-1
```

| Flag | Default | Description |
|---|---|---|
| `--force`, `-f` | `false` | Skip the `(yes/no)` confirmation |

> **Note:** AMI entries can only be deleted by the `metadata.name` set when they were applied. Seeded defaults (no name) are not directly deletable — apply them first with a name to manage them.

---

## `saddle upgrade <cluster-name>`

Upgrade Rancher on a running cluster. Defaults for unset flags come from the stored cluster config.

```bash
saddle upgrade repro-sure-11610 --version 2.13.5
saddle upgrade repro-sure-11610 --version 2.13.5 --replicas 2 --audit-log
```

- **K8s clusters** (rke2 / k3s): runs the Ansible upgrade playbook.
- **Docker clusters**: SSHs in and runs `docker stop && docker rm && docker run` with the new image.
- Progress streams to stdout and is written to `logs/<name>-upgrade.log`.

| Flag | Default | Description |
|---|---|---|
| `--version` | current | Target Rancher version |
| `--replicas` | `1` | Kubernetes deployment replicas |
| `--audit-log` | `false` | Enable Rancher audit log |
| `--audit-log-level` | `1` | Audit log verbosity (0–3) |
| `--image-tag` | current | Hotfix image tag override |
| `--debug` | `false` | Enable Rancher debug mode |

---

## Configuration files

| File | Purpose |
|---|---|
| `config.yaml` | Cluster state (auto-managed) |
| `cloud-credentials.yaml` | AWS credentials |
| `profiles.yaml` | Reusable infrastructure profiles |
| `amis.yaml` | AMI catalog (distro × region → AMI ID) |
| `logs/<name>.log` | Deployment / deletion logs |
| `logs/<name>-upgrade.log` | Upgrade logs |

All files containing secrets are created with `0600` permissions.
