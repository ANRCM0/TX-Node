# TX-Node S4 Compatibility Inventory

Status: **S4-D retirement implementation in progress**

This document inventories TX-Node compatibility surfaces after Runtime Simplification S1–S3.

S4 follows the project rule:

> compatibility first, migration second, retirement last.

The operator has authorized retirement of the historical host-management/release aliases. Protocol compatibility and standalone behavior remain separate decisions.

## 1. Canonical product path

The current managed product path is:

```text
TXBoard
   -> Machine mode
   -> TX-Node
   -> sing-box / Xray

TX-Node deployment / upgrade / rollback
   -> TX-Node-Installer
   -> Docker Compose / official GHCR image
   -> txnode management command
```

The public Installer is authoritative for host deployment lifecycle.

Current canonical deployment locations include:

- config root: `/etc/txnode`;
- management command: `txnode`;
- runtime image: `ghcr.io/paimoncai/tx-node:latest`.

Machine Runtime Update v1 also delegates back to this Installer runtime.

## 2. Classification

Compatibility surfaces are classified as:

- **canonical** — receives product development;
- **supported adapter** — stable interoperability surface, not itself legacy;
- **frozen compatibility** — bug/security/compatibility fixes only;
- **migration source** — retained specifically so existing installs can move to the canonical path;
- **retirement candidate** — may be proposed for a future breaking release only after the S4 gates are satisfied.

## 3. Inventory

| Surface | Classification | Current owner | Canonical replacement / target | Current blocker to removal |
| --- | --- | --- | --- | --- |
| Machine mode | canonical | TX-Node Machine orchestrator + TXBoard | none | not a retirement target |
| Xboard-compatible ControlPlane protocol | supported adapter | `internal/controlplane` / panel adapter | none; interoperability remains supported | external panel compatibility is intentional |
| Installer Docker deployment | canonical | TX-Node-Installer | none | not a retirement target |
| `tx-node` binary identity | canonical runtime identity | `cmd/tx-node` | none | not a retirement target |
| panel-connected single-node mode | frozen compatibility | TX-Node Service + Xboard adapter | TXBoard Machine mode where available | existing non-Machine panel deployments; no forced migration window defined |
| Local / standalone ControlPlane | frozen compatibility | TX-Node local ControlPlane/config/model | no exact replacement | offline/local use is semantically different from TXBoard Machine mode |
| `xboard-node` binary alias/artifact | retired from v2 mainline | historical release workflow | `tx-node` | Installer migration remains for old hosts |
| `xbctl` | retired from v2 mainline | historical TX-Node CLI | Installer-owned `txnode` command | Installer migration/cleanup remains for old hosts |
| legacy host `/etc/xboard-node` layout | migration source; retirement candidate | historical native install compatibility | Installer `/etc/txnode` layout | migration/import support must remain available for existing hosts |
| legacy container `/etc/xboard-node/config.yml` | bounded compatibility fallback | TX-Node startup resolver | `/etc/txnode/config.yml` | existing generated Compose files may still mount the old target |
| `xboard-node.service` native systemd layout | migration source; retirement candidate | historical native deployment | Installer Docker deployment | existing services may still be running; safe import/rollback must be proven |
| `/usr/local/bin/xboard-node` native binary path | migration source; retirement candidate | historical native deployment | official container / `tx-node` identity | old service definitions and operator scripts may reference the path |
| `/usr/local/bin/xbctl` legacy management path | migration source; retirement candidate | historical native deployment | `/usr/local/bin/txnode` | old operational runbooks may still use it |

## 4. Canonical source entrypoint

The source-tree command entrypoint is canonicalized as:

```text
cmd/tx-node
   -> tx-node
```

The v2 retirement step now also stops building/publishing the historical
`xboard-node` artifact and removes `xbctl` from TX-Node. Existing native
installs are handled by Installer-owned migrate/legacy-cleanup paths rather
than by shipping the old host manager forever.

## 4. Important distinction: protocol compatibility is not deployment legacy

The Xboard-compatible ControlPlane adapter is not being proposed for removal merely because historical deployments used the `xboard-node` name.

These are separate concerns:

```text
Xboard protocol compatibility
        = supported ControlPlane adapter

xboard-node binary name / xbctl / old systemd layout
        = deployment compatibility surfaces
```

S4 must not accidentally turn a deployment cleanup into a ControlPlane protocol break.

## 5. Standalone requires a product decision, not mechanical deletion

Standalone mode has no exact canonical replacement.

Machine mode requires a Control Plane. Standalone exists specifically to run a locally managed node without handshake, polling, reporting or WebSocket synchronization.

Therefore standalone is **frozen**, but it is not automatically removable together with `xbctl` or the old systemd layout.

A future standalone-retirement proposal must first answer:

1. whether offline/local operation remains a supported product use case;
2. whether an Installer-managed local mode replaces it;
3. how existing standalone configuration maps without losing routing, TLS, user or limiter semantics;
4. what release/window communicates the change.

Until then, standalone receives compatibility and security fixes but no feature-first development.

## 6. Existing migration path for native/systemd installs

TX-Node-Installer already recognizes the historical native layout only as a migration source:

```text
/etc/xboard-node
/usr/local/bin/xboard-node
/usr/local/bin/xbctl
xboard-node.service
        |
        v
TX-Node-Installer detect/import
        |
        v
/etc/txnode
Docker Compose
txnode
```

The Installer documentation explicitly defines the Docker-based `deploy.sh` path as public/canonical and supports importing historical native installs.

This is the correct migration direction. TX-Node Core should not grow a second deployment manager to retire the first one.

## 7. Retirement gates

No compatibility surface may be removed until all applicable gates are satisfied.

### Gate A — replacement exists

There must be a concrete replacement for every supported operation the surface performs.

Examples:

- `xbctl` operation -> `txnode` equivalent;
- native deployment -> Installer Docker deployment;
- `xboard-node` executable reference -> `tx-node` / official image.

If there is no exact replacement, removal is blocked.

### Gate B — migration is tested

Migration must have regression coverage for:

- config import;
- credentials transfer without logging secrets;
- machine/node identity preservation;
- kernel selection;
- health port behavior;
- multi-instance constraints;
- failed migration rollback.

### Gate C — release compatibility window

A breaking release must define:

- first deprecated release;
- minimum supported migration version;
- last release publishing compatibility artifacts;
- removal release;
- operator-facing migration instructions.

S4 does not choose those version numbers yet.

### Gate D — usage evidence

Retirement should use available operational evidence without adding default phone-home telemetry.

Acceptable evidence includes:

- Installer migration/support cases;
- release artifact/download observations where available;
- issue/support reports;
- explicit operator feedback;
- opt-in diagnostics if ever introduced under a separate privacy review.

Absence of telemetry is not evidence of zero usage.

### Gate E — rollback path

For host deployment migrations, operators must be able to recover from a failed migration without losing configuration or credentials.

## 8. Freeze policy

The following remaining surfaces are frozen for feature expansion:

- native/systemd deployment behavior as migration input;
- legacy `/etc/xboard-node` paths;
- Local/standalone management surfaces;
- panel-connected legacy single-node-first workflows.

Allowed changes:

- security fixes;
- correctness fixes;
- migration fixes;
- compatibility fixes;
- tests/documentation required to preserve the migration path.

Not allowed by default:

- new product capabilities implemented only for legacy surfaces;
- new deployment lifecycle logic in TX-Node Core;
- new `xbctl` feature development;
- a second host-management runtime competing with TX-Node-Installer.

## 9. S4 work plan

S4 is intentionally split into independent work:

### S4-A — compatibility inventory

This document.

No runtime behavior change.

### S4-B — post-S3 runtime stabilization

Harden the extracted runtime controllers with race/lifecycle regressions, especially:

- consecutive poll/result handoff;
- resync overlap;
- push start/stop/discovery;
- report overlap/backoff;
- user desired-state rollback;
- kernel applied-state transitions;
- shutdown behavior.

The normal test suite already runs with Go's race detector; S4-B should add focused lifecycle/stress coverage rather than another runtime implementation.

### S4-C — freeze enforcement

Add lightweight regression checks that make accidental feature growth on legacy deployment surfaces visible during review/CI.

This must not prevent security or migration fixes.

### S4-D — versioned retirement implementation

Authorized retirement scope:

- remove `xbctl` source/build/release output from TX-Node;
- stop producing the `xboard-node` binary/release alias;
- keep legacy native/systemd detection, import and cleanup in TX-Node-Installer;
- canonicalize the container config path as `/etc/txnode/config.yml`;
- retain a bounded TX-Node startup fallback to `/etc/xboard-node/config.yml`
  only for already-generated Compose files;
- update TX-Node-Installer to generate only the canonical container config
  target while preserving remote-update repair for old Compose files.

Not authorized by this step:

- Xboard ControlPlane protocol removal;
- standalone removal;
- single-node protocol removal.

## 10. Non-goals

S4-A does not:

- delete `xbctl`;
- stop publishing `xboard-node`;
- remove standalone;
- remove single-node panel mode;
- change Xboard protocol compatibility;
- change TXBoard ↔ TX-Node contracts;
- change Installer behavior;
- add telemetry;
- add a database or persistent retirement-state model.
