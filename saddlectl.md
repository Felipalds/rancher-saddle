# saddlectl — kubectl-style redesign proposal

This document describes a proposed CLI redesign. Nothing is implemented yet. Read through it and tell me what to change, cut, or keep before I write any code.

---

## Core idea

Replace the current flag-heavy subcommands with three universal verbs — `apply`, `get`, `delete` — that operate on typed YAML resources, exactly like kubectl. Every resource is declared in a file, applied declaratively, and listed/deleted by kind and name.

```
saddle apply -f resource.yaml     # create or update
saddle get   <kind> [name]        # list or describe
saddle delete <kind> <name>       # remove
```

The TUI is **unchanged**.

---

## New YAML schemas

Every resource shares the same envelope:

```yaml
apiVersion: saddle/v1
kind: <Kind>
metadata:
  name: <name>
spec:
  ...
```

Multiple resources can live in one file, separated by `---`.

---

### Kind: Cluster

```yaml
apiVersion: saddle/v1
kind: Cluster
metadata:
  name: repro-sure-11610
spec:
  method: local               # local | docker
  provider: aws
  credential: prod            # name of a saved Credential (optional — embed below if needed)
  region: us-west-2
  subnetId: subnet-0abc12345
  securityGroupId: sg-0abc12345
  ami: ami-0a3e3ef8596692376
  instanceType: t3.xlarge
  instanceCount: 3
  nodePrefix: sure-11610
  kubernetes:
    distribution: rke2        # rke2 | k3s | docker
    version: v1.33.7+rke2r1
  ssh:
    keyName: my-key
    privateKeyPath: ~/.ssh/my-key.pem
    user: ubuntu
  rancher:
    enabled: true
    version: 2.13.5
    prime: false
    bootstrapPassword: admin
    imageTag: ""
    debug: false
```

> When `credential` references a saved Credential by name, `saddle apply` loads the keys automatically. If you prefer self-contained files (e.g. for LLM-generated blueprints), you can embed the keys directly under `spec.accessKey` / `spec.secretKey` instead.

---

### Kind: Credential

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

```yaml
apiVersion: saddle/v1
kind: AMI
metadata:
  name: ubuntu-22-us-east-1    # slug used as the CLI identifier
spec:
  distro: "Ubuntu 22.04 LTS"
  region: us-east-1
  amiId: ami-0c7217cdde317cfec
```

---

## Multi-resource file (like kubectl)

A single file can define multiple resources. `saddle apply -f all.yaml` processes them in order.

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

## New command surface

### `saddle apply -f <file>`

Create or update any resource. Dispatches by `kind`. Works with multi-doc files.

```bash
saddle apply -f cluster.yaml
saddle apply -f credentials.yaml
saddle apply -f all-resources.yaml
```

`apply` on an existing Cluster is **not** a re-deploy — it updates the stored config (like `kubectl apply` patches an existing object). To re-provision, delete and re-apply.

---

### `saddle get <kind> [name]`

List all resources of a kind, or describe one by name. Kind is case-insensitive and can be singular or plural.

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

Replaces: `saddle list`, `saddle credentials list`, `saddle profiles list`, `saddle amis list`.

---

### `saddle delete <kind> <name>`

Remove a resource. For clusters, this also runs `tofu destroy`. Prompts for confirmation unless `--force` is passed.

```bash
saddle delete cluster repro-sure-11610
saddle delete credential prod --force
saddle delete profile us-east
saddle delete ami ubuntu-22-us-east-1
```

Replaces: `saddle delete <name>`, `saddle credentials delete`, `saddle profiles delete`, `saddle amis delete`.

---

### `saddle upgrade <name>` (unchanged)

Upgrade Rancher on a running cluster. Flags stay the same. This is kept as a dedicated verb because it is an action, not a resource state change.

```bash
saddle upgrade repro-sure-11610 --version 2.13.5
```

---

## Commands removed

| Removed | Replaced by |
|---|---|
| `saddle list` | `saddle get clusters` |
| `saddle create` | `saddle apply -f cluster.yaml` |
| `saddle credentials list/add/delete/edit` | `saddle get/apply/delete credential` |
| `saddle profiles list/add/delete/edit` | `saddle get/apply/delete profile` |
| `saddle amis list/add/delete/edit` | `saddle get/apply/delete ami` |
| `saddle list-providers` | keep or remove — low value |
| `saddle list-orchestrators` | keep or remove — low value |

---

## Open questions for you to decide

1. **Backward compatibility** — Should the old commands (`saddle credentials add`, `saddle profiles list`, etc.) still work as hidden aliases, or is a clean break acceptable? No, you can delete all old ones and keep only the current.

2. **`saddle apply` on an existing Cluster** — Patch the config only, or re-provision? (I propose: patch only, delete+apply to re-deploy.) do not need to care about this.

3. **Embedded vs referenced credentials** — Should a Cluster spec be allowed to contain raw `accessKey`/`secretKey` inline (for LLM-generated files), or must it always reference a Credential by name? can allow, but remember of updating the docs.

4. **`saddle upgrade` as YAML** — Should upgrade also become `kind: ClusterUpgrade` applied with `saddle apply -f`, or stay as a dedicated verb? can you explain me better and give me pros and cons?

5. **Old `ClustersConfig` YAML** — The current `apply -f` reads the raw `ClustersConfig` format (no `apiVersion`/`kind`). Should we keep that as a fallback, or require the new format going forward? only new format.

6. **Output format** — Should `saddle get clusters` support `-o json` or `-o yaml` output like kubectl, or is a table enough for now?

---

## What needs to change in code

| Area | Change |
|---|---|
| `internal/cluster/apply.go` | New YAML parser for `apiVersion/kind/metadata/spec` envelope; dispatch by kind |
| New `internal/resource/` package | Types for `ClusterResource`, `CredentialResource`, `ProfileResource`, `AMIResource` |
| `main.go` | Replace subcommand groups with `apply`, `get`, `delete` verbs |
| `docs/cli.md` | Rewrite with new command surface |
| `README.md` | Update quick-start examples |

The underlying service packages (`internal/credentials`, `internal/config`, `internal/cluster`) are **unchanged** — only the CLI wiring and YAML parsing layer changes.\
`
One question: where will we save this? when we do an apply, we need to save it somewhere. All these settings. SQLITE?, ETCD? JSON? YAML? how?
