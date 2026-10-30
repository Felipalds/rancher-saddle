# Harvester (local libvirt) — Unit-Test Coverage Plan

**Status:** Planned coverage; a solid base of tests already exists. This lists the
remaining pure-logic gaps to close. Scope: `internal/providers/libvirt/` and
`internal/orchestrators/harvester/`. End-to-end (real KVM host) is out of scope.

Follow repo conventions: table-driven `t.Run()` subtests, `testify/assert`/`require`,
`t.TempDir()` for file I/O, stub external deps only (the existing `commandRunner`/
`lookPath`/`statFn` hooks and `releaseBaseURL` swap). No mocks for pure logic.

## Already covered (do not duplicate)

- `libvirt/config_test.go` — `FromMap` defaults/overrides, `instance_count` fallback,
  float64 node_count, `Validate` table, `MediaDir`, default config, required fields.
- `libvirt/provider_test.go` — `Name`, `Validate`, `CustomApply` interface,
  `GenerateInfrastructure` (+ UEFI/BIOS firmware + invalid config), `PlannedOutputs`,
  `GetOutputs`, `GenerateMAC`, `ComputeTopology` (+ bad subnet).
- `libvirt/config_server_test.go` — serves config by IP, serves ISO, 404 for unknown IP.
- `libvirt/iso_cache_test.go` — `CacheDir`, `EnsureISO` (skip/error/download),
  `ListCachedVersions` (+ empty), `AutoInstallISOPath` (determinism + URL sensitivity),
  `patchGrubCfg` (happy path).
- `libvirt/validator_test.go` — `CheckPrerequisites` all-present / all-missing / only-xorriso.
- `harvester/config_test.go` — `FromMap` defaults/overrides, `Validate` table, defaults, fields.
- `harvester/orchestrator_test.go` — `Name`, `GeneratePlaybook` (create vs join, iso_url,
  server_url), missing topology, `GenerateInventory` no-op, `Validate`, `GetModules`.

## Unit tests to add

### `libvirt/config.go` (config_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| FromMap firmware override | `firmware: "bios"` lands in `cfg.Firmware`; default is UEFI | map→struct, no deps |
| Validate bad firmware | `Firmware:"xen"` → error; UEFI/BIOS → no error | pure branch |

### `libvirt/topology.go` (topology_test.go — new)
| Test | Asserts | Why pure |
|---|---|---|
| subnetBase valid/invalid | `"192.168.0.0/24"`→`"192.168.0"`; octet>255, <4 octets, non-numeric → error | string/int logic |
| nodeIP | base + index → `.30`, `.31`… (offset 30) | arithmetic |
| ComputeTopology single node | 1 node, role `create`, Gateway `.1`, ISOURL has port+`/harvester.iso` | pure, no deps |

### `libvirt/provider.go` (provider_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| configURL | formats `http://<gw>:8181/config` | string fmt |
| topologyOutputs | maps node IPs → `InstanceIPs`, gateway → `AdditionalOutputs` | pure map |
| domainName | `cluster + "-" + node` | pure |
| hasActiveYes | parses `Active: yes`/`no`/absent `virsh net-info` output | string parse |
| GenerateInfrastructure network DHCP range | network.xml has `.100`/`.200` range | template render, TempDir |

### `libvirt/iso_cache.go` (iso_cache_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| ISOName / artifactNames | filename `harvester-<v>-amd64.iso`; single artifact | pure fmt |
| RepackAutoInstallISO existing dst = no-op | pre-create dst → returns nil, no xorriso needed | early `os.Stat` return, TempDir |
| patchGrubCfg no `set default=` | inject prepended when anchor absent; old timeouts dropped | pure string, TempDir |

### `libvirt/config_server.go` (config_server_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| routes two IPs distinctly | create-IP and join-IP each get their own file body | loopback HTTP, no external dep |
| config file unreadable → 500 | mapped path missing on disk → `500` | real handler, TempDir |

### `libvirt/validator.go` (validator_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| only libvirtd inactive | `is-active`→"inactive" only → error names libvirtd, not the tools | stubbed hooks |

### `harvester/config.go` (config_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| parseNTP variants | `[]string`, `[]interface{}` (drops non-strings/empties), comma/space string, nil/unknown→nil | pure switch |
| boolFrom variants | `bool`, `"Yes"`/`"yes"`/`"true"`→true, other string→false, fallback | pure switch |
| FromMap `version` alias | `version` key used when `harvester_version` absent | map precedence |
| FromMap node_count fallback | `instance_count` used when `node_count` unset; default 1 | pure |

### `harvester/orchestrator.go` + template (orchestrator_test.go)
| Test | Asserts | Why pure |
|---|---|---|
| GeneratePlaybook empty-nodes topology | topology with `Nodes: []` → "topology has no nodes" error | pure guard, TempDir |
| rendered NTP / token / password | config contains token, password, each NTP server, and the single-line `ntp-servers` JSON (`{"ntpServers":[...]}`) for multi-server input | template render, TempDir |

## Out of scope (needs real KVM host — integration, not unit)

- `Provider.CustomApply`, `bootNode`, `ensureNetwork` — drive `virsh`/`qemu-img`.
- `waitForVIP` — real HTTPS polling of a live cluster VIP.
- `RepackAutoInstallISO` full run — invokes `xorriso` against a real Harvester ISO
  (the dst-exists no-op path is unit-tested above).
- `downloadFile` against the real `releases.rancher.com` (already covered via httptest).
- `Teardown` — `virsh destroy/undefine/net-*` (wired into the delete flow via `internal/cluster/commands.go` and `deletemodal.go`; the virsh calls still need a real host).
- `CheckPrerequisites` against the actual host (already covered via stubbed hooks).
