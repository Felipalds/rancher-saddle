# Feature: Dynamic AMI Lookup via Terraform Data Sources

**Status:** Proposed (not implemented)

## Problem

The AWS template launches instances from a hardcoded AMI ID (`ami = "{{.AMI}}"` in
`internal/providers/aws/templates/main.tf.tmpl`), resolved from a static per-region table
(`internal/config/amis.go`). Stale or mismatched IDs silently install the wrong OS (observed:
"Ubuntu 22.04 / us-west-2" launched an Amazon Linux instance). The template already contains
a dead, unreferenced `data "aws_ami" "ubuntu"` block — leftover scaffolding.

## Proposal

Resolve the AMI at apply time with a `data "aws_ami"` block keyed by the selected distro's
owner ID + name pattern, instead of a hardcoded ID. AMI IDs become an optional override.

- Add `internal/providers/aws/distros.go` — a `Distro` map (display name → owner ID, name
  pattern, SSH user) for Ubuntu 22.04/24.04, RHEL 9, Debian 12, Amazon Linux 2023, SLES 15 SP6.
- Rewrite `main.tf.tmpl`: active `data "aws_ami" "selected"` block when no override AMI is
  set; `lifecycle { ignore_changes = [ami] }` so `most_recent` doesn't recreate running nodes.
- `AWSConfig` gains `Distro`/`AMIOwner`/`AMINamePattern`; legacy `ami` still wins when set so
  existing clusters keep deploying unchanged.
- Create/profile forms store `distro` (not `ami`) for known distros and auto-fill SSH user
  from the distro map. `amis.yaml` is repurposed as an override table.

## Not done

None of the above exists yet: no `distros.go`, the template still hardcodes the AMI, and the
dead `data "aws_ami" "ubuntu"` block is still present and unused.
