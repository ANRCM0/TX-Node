# TX-Node standalone maintenance policy

TX-Node is maintained as an independent project. It originated from Xboard-Node and keeps Xboard panel compatibility where that compatibility is part of the protocol surface, but TX-Node no longer treats upstream synchronization as its development model.

## Source of truth

- `ANRCM0/TX-Node` is the source of truth for development and releases.
- `main` is the product mainline.
- Releases are created from semantic `v*` tags only.
- Upstream repositories are references for fixes and ideas, not merge targets.

## Upstream changes

Do not periodically merge an upstream branch into TX-Node. For a useful upstream change: inspect the diff, confirm it applies, port the minimal change, adapt it to TX-Node, run tests, and release it under the TX-Node version line.

## Control-plane boundary

TX-Node core code depends on the `ControlPlane` interface and TX-native `NodeSpec` / `UserSpec` models rather than directly on a panel implementation.

- `LocalControlPlane` provides panel-free standalone operation.
- `XboardControlPlane` is the Xboard-compatible protocol adapter.
- machine mode uses the corresponding machine Xboard adapter with a shared websocket transport.
- future control planes such as TuneX should be added as new adapters instead of introducing panel-specific branches throughout service/kernel code.

See [`controlplane.md`](controlplane.md) for the adapter contract and extension rules.

## Optional panel plugins

TX-Node does not vendor or release panel-side plugins. AccessAudit is maintained by TXBoard under `integrations/AccessAudit/`; TX-Node only implements the optional audit reporter/client that interoperates with that plugin. Plugin absence must not affect the core node/control-plane protocol.

## Product-path freeze

The canonical product path is now:

```text
TXBoard
  -> Machine mode
  -> TX-Node
  -> sing-box / Xray
```

Local/standalone mode and legacy single-node/systemd management remain supported compatibility surfaces, but they are frozen for feature expansion. They may receive bug fixes, security fixes and compatibility repairs; new product features should target the canonical Machine path first.

This is a maintenance-policy change, not a removal. Any future removal requires an explicit breaking release and migration plan.

See [Runtime Boundary](./runtime-boundary.md).

## Compatibility boundary

The following are compatibility surfaces, not branding leftovers:

- Xboard panel API paths such as `/api/v1/server/UniProxy/*` and `/api/v2/server/*`;
- panel authentication fields such as `token`, `node_type`, and `machine_id`;
- existing Xboard node configuration fields consumed by deployed panels;
- historical `xboard-node` executable and `xbctl` CLI names, **removed from current TX-Node v2 artifacts**; existing native/systemd deployments can still be detected and migrated by TX-Node-Installer;
- the legacy native/systemd `/etc/xboard-node` layout on existing hosts, still recognized by the Installer's migration/cleanup flow;
- the in-container `/etc/xboard-node/config.yml` path until deployment compatibility is migrated explicitly.

Canonical TX-Node-facing identities are:

- node binary: `tx-node`;
- Docker image: `ghcr.io/anrcm0/tx-node`;
- runtime source/release repository: `ANRCM0/TX-Node`;
- installation and host-management entry point: public `ANRCM0/TX-Node-Installer`.

The runtime repository intentionally does not carry an installer copy. Host deployment layout such as `/etc/txnode`, the `txnode` management command, multi-panel instance management, backup/rollback, and Docker Compose generation are owned by TX-Node-Installer.

The old `xbctl` host-management program has been retired from TX-Node v2 builds and releases. The Installer now owns the `txnode` management command, Docker deployment, migration, and rollback. Do not reinstate the legacy CLI or host binaries as release artifacts.

## Go module identity

The Go module and TX-Node self-imports use:

```text
github.com/ANRCM0/TX-Node
```

This source-identity change does not alter Xboard panel protocol compatibility.

## Kernel fork dependencies

TX-Node uses the upstream Go **module APIs** `github.com/sagernet/sing-box` and `github.com/xtls/xray-core` for compatibility with their own import graphs. However, `go.mod` pins the concrete kernel source through `replace` to repositories **within the same GitHub account**:

- [`ANRCM0/sing-box`](https://github.com/ANRCM0/sing-box): contains the Mieru adapter and outbound-target-IP reporting used by TX-Node.
- [`ANRCM0/Xray-core`](https://github.com/ANRCM0/Xray-core): the TX-Node bandwidth-manager, per-user rate-limit writer, and tests are restored here.

Both fork revisions are pinned in `go.mod`, with checksum entries in `go.sum`; builds must not silently use upstream kernel releases instead of these patches. Run `go mod verify`, the node suites, cold-cache Docker builds, and test both kernel modes after changing a pin.

Original upstream names in `require` and source `import` are Go module identifiers, **not remote download targets**: the `replace` entries specify the actual ANRCM0 fork repositories. Historical `PaiMonCai` kernel fork fetch paths are no longer needed.

## License provenance

The historical `cedar2025/Xboard-Node` repository declares `MPL-2.0` in its README but does not currently expose a top-level LICENSE file through GitHub. TX-Node preserves project provenance and existing notices; before changing license terms or distributing under a different license, verify the licensing of inherited source and dependencies explicitly.

## Compatibility window

TX-Node v2 releases publish only the canonical `tx-node` binary. Existing `xboard-node` / `xbctl` installations remain **migration inputs** for TX-Node-Installer, not binaries or aliases published by TX-Node.
