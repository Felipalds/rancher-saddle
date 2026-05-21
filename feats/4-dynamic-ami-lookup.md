# Feature: Dynamic AMI Lookup via Terraform Data Sources

## Summary

Replace the static AMI ID table in `amis.yaml` / `DefaultAMIs()` with a `data "aws_ami"` block in the Terraform template that resolves the latest official image for the selected distro at deploy time. AMI IDs become an *override* feature instead of the primary mechanism, eliminating the class of bug where stale or incorrect hardcoded AMI IDs silently install the wrong OS.

---

## Motivation

A user selected **Ubuntu 22.04 LTS** in **us-west-2** and got an Amazon Linux instance instead. Trace:

1. Form passed `(distro="Ubuntu 22.04 LTS", region="us-west-2")` to `AMIsConfig.GetAMI` — correct.
2. `amis.yaml` returned `ami-0dc8f589abe99f538` — the value AWS associates with that ID is NOT Ubuntu in us-west-2.
3. Terraform launched the instance with that AMI ID — exactly as instructed.

The plumbing is correct end-to-end. The data is wrong. And it will keep going wrong because:

- AMI IDs in `DefaultAMIs()` (`internal/config/amis.go:25-67`) appear hand-picked at some past date; there is no source of truth or refresh mechanism.
- AWS publishes new Ubuntu/RHEL/SLES images frequently; a list curated today will diverge from "current" within weeks.
- An AMI ID is opaque — there is no way to tell "is `ami-0dc8f589abe99f538` actually Ubuntu?" without launching it or calling `DescribeImages`.
- Mistakes are silent: the wrong distro launches, SSH still works (sort of), Ansible later fails on package-manager mismatch with a confusing error.

The template at `internal/providers/aws/templates/main.tf.tmpl:16-29` already declares a `data "aws_ami" "ubuntu"` block, but it is never referenced — leftover scaffolding from an unfinished migration. This feature finishes that migration.

---

## Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Source of truth for AMI ID | AWS `DescribeImages` via `data "aws_ami"` at apply time | Always current, owner-verified, no per-region table |
| What flows through code | Distro name (e.g. `"Ubuntu 22.04 LTS"`) | A symbolic identifier the user actually picked, not a magic ID |
| Backward compat | If `cluster.Provider.Config["ami"]` is set (legacy clusters), use it directly and skip the data source | Existing clusters in `config.yaml` keep working without re-deploy |
| `amis.yaml` | Repurposed as **override table**, not the default path | Power users can still pin specific AMIs; the existing TUI catalog screen stays useful |
| Distro→filter mapping location | `internal/providers/aws/distros.go` (new file) | Small Go map; doesn't need to be edited at runtime |
| Image freshness pinning | `lifecycle { ignore_changes = [ami] }` on `aws_instance` | `most_recent = true` means each `tofu apply` could see a newer image; we don't want to recreate running nodes |
| SSH user | Auto-fill from distro map; user can still override | Currently the form always defaults to `ubuntu`, which is wrong for RHEL (`ec2-user`), SLES (`ec2-user`), etc. |

---

## Distros Supported Out of the Box

| Distro | Owner ID | Name pattern | SSH user |
|---|---|---|---|
| Ubuntu 22.04 LTS | `099720109477` (Canonical) | `ubuntu/images/hvm-ssd*/ubuntu-jammy-22.04-amd64-server-*` | `ubuntu` |
| Ubuntu 24.04 LTS | `099720109477` | `ubuntu/images/hvm-ssd*/ubuntu-noble-24.04-amd64-server-*` | `ubuntu` |
| RHEL 9 | `309956199498` (Red Hat) | `RHEL-9.*_HVM-*-x86_64-*-Hourly2-GP3` | `ec2-user` |
| Debian 12 | `136693071363` (Debian) | `debian-12-amd64-*` | `admin` |
| Amazon Linux 2023 | `137112412989` (Amazon) | `al2023-ami-2023.*-x86_64` | `ec2-user` |
| SLES 15 SP6 | (varies — see Open Questions) | `suse-sles-15-sp6-v*-hvm-ssd-x86_64` | `ec2-user` |

---

## Files to Modify

| File | Action |
|---|---|
| `internal/providers/aws/distros.go` | **NEW** — `Distro` struct + map keyed by display name |
| `internal/providers/aws/templates/main.tf.tmpl` | **MODIFY** — add active `data "aws_ami"` block, remove dead one, switch `aws_instance.ami` to data lookup with override branch |
| `internal/providers/aws/config.go` | **MODIFY** — `AWSConfig` gains `Distro`, `AMIOwner`, `AMINamePattern`; remove the hardcoded `ami-0c58b2975bef51185` fallback |
| `internal/providers/aws/config_test.go` | **MODIFY** — cover both legacy AMI path and new distro path |
| `internal/tui/views/createform.go` | **MODIFY** — store distro name in `cluster.Provider.Config["distro"]`; pre-fill SSH user from the distro map when user changes the OS picker |
| `internal/tui/views/profilesform.go` | **MODIFY** — same as createform for profiles |
| `internal/config/clusters.go` | No structural change — `Provider.Config` is `map[string]interface{}`, so `distro` rides along automatically |
| `internal/config/amis.go` | **MODIFY** — keep `AMIEntry`/`GetAMI` for override lookups; drop the 33-entry `DefaultAMIs()` seed (or shrink to a few well-tested IDs as a fallback) |

---

## Step 1 — Distro Catalog

`internal/providers/aws/distros.go`:

```go
package aws

type Distro struct {
    Name        string
    Owner       string
    NamePattern string
    SSHUser     string
}

var Distros = map[string]Distro{
    "Ubuntu 22.04 LTS": {
        Name:        "Ubuntu 22.04 LTS",
        Owner:       "099720109477",
        NamePattern: "ubuntu/images/hvm-ssd*/ubuntu-jammy-22.04-amd64-server-*",
        SSHUser:     "ubuntu",
    },
    "Ubuntu 24.04 LTS": { /* ... */ },
    "RHEL 9":           { /* ... */ },
    "Debian 12":        { /* ... */ },
    "Amazon Linux 2023":{ /* ... */ },
    "SLES 15 SP6":      { /* ... */ },
}

// ListDistros returns sorted display names for the form picker.
func ListDistros() []string { /* ... */ }
```

---

## Step 2 — Terraform Template

`internal/providers/aws/templates/main.tf.tmpl`:

```hcl
provider "aws" {
  region     = "{{.Region}}"
  access_key = "{{.AccessKey}}"
  secret_key = "{{.SecretKey}}"
}

{{- if eq .AMI "" }}
data "aws_ami" "selected" {
  most_recent = true
  owners      = ["{{.AMIOwner}}"]

  filter {
    name   = "name"
    values = ["{{.AMINamePattern}}"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}
{{- end }}

locals {
  resolved_ami = {{ if eq .AMI "" }}data.aws_ami.selected.id{{ else }}"{{.AMI}}"{{ end }}
}

resource "aws_instance" "k8s_node" {
  count         = {{.InstanceCount}}
  ami           = local.resolved_ami
  instance_type = "{{.InstanceType}}"

  subnet_id                   = "{{.SubnetID}}"
  vpc_security_group_ids      = ["{{.SecurityGroupID}}"]
  key_name                    = "{{.SSHKeyName}}"
  associate_public_ip_address = true

  root_block_device {
    volume_size           = {{.RootVolumeSize}}
    volume_type           = "gp3"
    delete_on_termination = true
  }

  lifecycle {
    # Don't replace running nodes when AWS publishes a newer image.
    ignore_changes = [ami]
  }

  tags = {
    Name = "{{.NodePrefix}}-${count.index}"
  }
}
```

- `{{.AMI}}` empty → use data source.
- `{{.AMI}}` set → legacy override, render literal ID.
- `ignore_changes = [ami]` pins the AMI after first apply.

---

## Step 3 — AWSConfig

`internal/providers/aws/config.go`:

```go
type AWSConfig struct {
    // ... existing fields ...
    Distro          string // e.g. "Ubuntu 22.04 LTS"
    AMIOwner        string // resolved from Distros[Distro]
    AMINamePattern  string // resolved from Distros[Distro]
    AMI             string // legacy / override
}

func (c *AWSConfig) FromMap(m map[string]interface{}) {
    // ... existing ...
    if v, ok := m["distro"].(string); ok {
        c.Distro = v
        if d, ok := Distros[v]; ok {
            c.AMIOwner = d.Owner
            c.AMINamePattern = d.NamePattern
        }
    }
    if v, ok := m["ami"].(string); ok {
        c.AMI = v // takes precedence if set
    }
    // REMOVE: the c.AMI = "ami-0c58b2975bef51185" fallback. Never silently substitute.
}
```

If neither `distro` nor `ami` is set, `GenerateInfrastructure` should return a clear validation error.

---

## Step 4 — TUI Form

In `createform.go`:

- The OS Image picker keeps its current behavior (options from `amis.yaml` distros + `"Custom"`), **plus** options from `Distros` (deduplicated).
- On submit:
  - If user picked a known distro from `Distros`: write `"distro": "..."` into `cluster.Provider.Config`; do **not** write `ami`.
  - If user picked "Custom": write `"ami": "<their input>"` only.
  - If user picked an entry that exists only in `amis.yaml` (override): resolve the AMI ID and write `"ami": "..."` only.
- On distro change: pre-fill the SSH user field from `Distros[picked].SSHUser` if the field is empty or matches a previous distro's default.

---

## Step 5 — amis.yaml Becomes Optional

- `LoadAMIs` no longer creates a default-seeded file when missing — return an empty config instead.
- The AMIs management screen (`amislist.go`, `amisform.go`) stays as-is and lets power users add override entries.
- Document in the screen header: "Entries here override the default dynamic AMI lookup for the given distro+region."

---

## Migration of Existing Clusters

`config.yaml` entries that already have `ami: ami-xxx`:
- Continue to deploy with the legacy AMI (the template's `{{ if eq .AMI "" }}` branch).
- Status quo for them — no auto-migration, no surprise replacements.

`config.yaml` entries created after this feature lands:
- Store `distro: "Ubuntu 22.04 LTS"` (no `ami:` key).
- Resolve AMI fresh on every deploy.

A small CLI helper could be added later (`saddle migrate-amis`) to rewrite legacy entries, but it's not in scope here.

---

## Testing Plan

- **`internal/providers/aws/distros_test.go`** — verify `Distros` map has the expected keys; `ListDistros` returns sorted unique names.
- **`internal/providers/aws/config_test.go`** — extend with cases:
  - `FromMap({"distro": "Ubuntu 22.04 LTS"})` populates `Owner`/`NamePattern`/no `AMI`.
  - `FromMap({"ami": "ami-xyz"})` populates `AMI` only.
  - `FromMap({"distro": "...", "ami": "ami-xyz"})` — override wins, AMI is set.
  - `FromMap({})` leaves `AMI` empty (no silent fallback).
- **Template render test** — render `main.tf.tmpl` with `AMI=""` and with `AMI="ami-xxx"`; assert the data block appears/doesn't and `local.resolved_ami` resolves correctly.
- **Manual** — deploy one Ubuntu cluster and one RHEL cluster end-to-end and SSH in to confirm `/etc/os-release`.

---

## Pros

- **Wrong OS becomes structurally impossible** for known distros — owner ID + name pattern is what AWS itself uses to identify Canonical / Red Hat / etc. images.
- **No more per-region table to maintain.** One distro row replaces ~12 region rows.
- **Always current.** Hotfix CVE-patched images are picked up automatically on new cluster creation.
- **Removes ~50 lines of fragile seed data** and the dead `data "aws_ami" "ubuntu"` block.
- **Auto-fill SSH user** as a free bonus — currently the form defaults to `ubuntu` even when the user picks RHEL, causing first-time SSH failures.

## Cons / Caveats

- **SLES via SUSE Marketplace is messy.** SUSE-published AMIs have owner IDs that vary by product/region. Either (a) limit to BYOS images, (b) keep SLES on the static `amis.yaml` path, or (c) accept that SLES needs a region-specific owner table. See open questions.
- **`most_recent = true` is non-deterministic.** Mitigated by `lifecycle { ignore_changes = [ami] }`, but the first apply of two clusters created an hour apart could pick different AMIs. For this tool's use case (one-off test/demo clusters), that's acceptable.
- **One extra `DescribeImages` API call** per deploy. Negligible (<200 ms, free).
- **Distros map needs occasional updates** when AWS deprecates name patterns (rare — Canonical hasn't changed Ubuntu's pattern in years).

---

## Open Questions

1. **SLES owner IDs** — keep SLES on the `amis.yaml` static path for now, or chase down per-region owner IDs? Recommend: keep static for SLES, dynamic for the rest. The `amis.yaml` override mechanism handles it cleanly.
2. **Should the OS Image picker show only `Distros` entries by default, with a "Show all from amis.yaml" toggle?** Or merge both lists with annotations? Current proposal merges and lets the override take precedence when both exist.
3. **Validation error when neither `distro` nor `ami` is provided** — surface in TUI via the new `LastError` field we just added, or block submission in the form? Recommend: block in the form (consistent with other required-field checks).
4. **Do we want Ubuntu 24.04 as the new default distro for the form?** Currently 22.04 is the de-facto default; 24.04 is LTS and supported by RKE2/K3s.
