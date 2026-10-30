# Feature: Local Harvester Clusters via libvirt (ISO auto-install)

**Status:** Partially implemented — backend done and unit-tested, but NOT verified
end-to-end (needs a real KVM/libvirt host). Advanced create-form fields are deferred.

## What it does now

Deploys a local [Harvester](https://harvesterhci.io/) HCI cluster (1–3 nodes) on a
KVM/libvirt host, driven directly via `virsh` + `qemu-img` (no PXE/DHCP, no Vagrant).

- Provider id is **`libvirt`** (`core.ProviderLibvirt`). Older configs created before the
  rename may carry the legacy id `"vagrant"`; the delete flow still recognizes it.
- Orchestrator is **`harvester`** (`core.OrchestratorHarvester`).

**Flow (`libvirt.Provider`):**
1. Download the Harvester ISO (cached under `harvester-isos/`, gitignored) and repack it
   once with `xorriso` to patch GRUB for an unattended install
   (`harvester.install.automatic=true` + `harvester.install.config_url=http://<gw>:8181/config`).
2. Each node is a libvirt domain: a blank qcow2 disk + the ISO as CD-ROM, boot order
   `hd,cdrom` (blank disk falls through to the installer, then boots the installed disk —
   no reinstall loop).
3. A built-in HTTP config server serves each node its install config by source IP:
   node 0 gets `mode: create`, the rest get `mode: join` (`server_url` + `token`). Node 0
   boots first; the provider waits for the VIP, then boots the joiners.
4. Node topology (names/MACs/IPs/roles + gateway) is shared between provider and
   orchestrator via `topology.json` in the build dir.

Teardown IS wired into the delete flow: `libvirt.Teardown` runs `virsh destroy`/`undefine`
+ `net-undefine` and removes disks (best effort), invoked from `deletemodal.go` for
`libvirt`/`vagrant` clusters.

## Where it lives

- `internal/providers/libvirt/` — `provider.go`, `config.go`, `validator.go` (checks
  `virsh`/`qemu-img`/`xorriso`), `iso_cache.go` (`EnsureISO`, `RepackAutoInstallISO`),
  `config_server.go`, `topology.go`, `templates/{network,domain}.xml.tmpl`. Has unit tests.
- `internal/orchestrators/harvester/` — `orchestrator.go`, `config.go`,
  `templates/config.yaml.tmpl` (role-driven create/join install config).
- Registered in `main.go`; selectable in the create form.

## What remains

- **End-to-end deploy is unverified** — only generation + unit tests run without a libvirt
  host. Needs real-host checks: GRUB patch for the specific ISO version, boot-order
  fallthrough, q35/virtio NIC naming (`management_interface.name`, default `enp1s0`),
  and libvirt/qemu file permissions for the ISO and disks.
- Advanced create-form fields (per-node CPU/mem/disk, VIP, cluster token) use
  provider/orchestrator defaults for now.
- Unit-test coverage plan tracked separately in `feats/harvester-unit-tests.md`.
