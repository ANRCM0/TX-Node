# TX-Node Runtime Boundary

Status: **active architecture baseline**

TX-Node is a dedicated Agent / Data Plane runtime for TXBoard and compatible control planes. It is not a second control plane and it is not a general-purpose host management agent.

The simplification goal is:

> Keep TX-Node small enough that its core responsibility can be stated as: receive bounded control-plane intent, run proxy nodes, enforce user/runtime policy, and report bounded runtime state.

## 1. Core runtime

The following responsibilities belong in TX-Node Core:

- ControlPlane adapter boundary and protocol translation;
- Machine orchestration and per-node runtime ownership;
- Node configuration/user synchronization;
- kernel lifecycle through the kernel interface;
- user, traffic, device and speed-limit enforcement;
- runtime status/traffic/device reporting;
- bounded typed Node Ops;
- Machine runtime update **delegation** to TX-Node-Installer;
- runtime-safe configuration validation needed before applying data-plane state.

These responsibilities are data-plane facts or actions. They must not depend on TXBoard database internals or host deployment implementation.

## 2. Authoritative runtimes outside TX-Node

TX-Node must delegate rather than duplicate specialized runtimes.

### TXBoard

TXBoard owns:

- authentication and authorization;
- User / Plan / Subscription / Order;
- Node / Machine / Group / Route facts;
- approval and audit policy;
- Agent Ops orchestration;
- module/plugin/theme lifecycle.

TX-Node never becomes a second source of truth for those domains.

### TX-Node-Installer

TX-Node-Installer owns:

- install;
- upgrade;
- rollback;
- uninstall;
- Docker Compose generation;
- host service units;
- multi-instance deployment layout;
- host backup/restore.

TX-Node may request a bounded deployment action through a typed bridge, but must not gain Docker socket, SSH or arbitrary shell access.

### Proxy kernels

sing-box / Xray adapters own protocol-specific packet/runtime implementation. TX-Node orchestrates them through the kernel interface instead of reimplementing protocol engines.

## 3. Supported adapters and backends

The following remain supported, but they do not expand the Core domain:

- Xboard-compatible ControlPlane adapter;
- sing-box backend;
- Xray backend.

Protocol-specific behavior should stay behind adapters. New panel-specific branches should not leak into Service or kernel-independent runtime code.

## 4. Compatibility surfaces

The following exist for compatibility and are **frozen for feature expansion**:

- Local / standalone ControlPlane;
- legacy single-node Xboard deployment behavior;
- `xboard-node` binary compatibility alias;
- `xbctl`;
- legacy native/systemd layout;
- legacy `/etc/xboard-node` paths.

Frozen means:

- bug fixes and security fixes are allowed;
- compatibility regressions are fixed;
- new product features should not target these surfaces first;
- removal requires an explicit breaking release and migration plan.

The canonical product path is Machine mode managed by TXBoard, deployed through TX-Node-Installer.

## 5. Optional capabilities

Optional capabilities must not quietly grow into core orchestration.

Current optional capabilities include:

- Access Audit reporter/client;
- certificate automation;
- DNS-provider integrations used by ACME;
- custom geo/routing assets.

Rules:

1. optional capability failure must not redefine core node truth;
2. optional capability code should remain isolated from Service orchestration;
3. adding a new third-party provider requires an explicit reason rather than default inclusion;
4. where TXBoard Plugin/Integration or host tooling can own the lifecycle, prefer delegation;
5. compatibility is preserved before any optional capability is removed.

Certificate consumption by the kernel remains a runtime need. Owning an ever-growing external DNS-provider catalog is not automatically a TX-Node Core responsibility.

## 6. Typed Ops boundary

Typed Ops are a core capability, but the Service orchestrator must not implement every operation directly.

The target shape is:

```text
ControlPlane event
       |
       v
nodeops.Executor
       |
       v
narrow Runtime adapter
       |
       v
Service / Kernel
```

The executor owns:

- operation allow-list dispatch;
- bounded argument validation;
- replay protection;
- bounded log/network diagnostics;
- typed result construction.

The Service adapter exposes only the minimum runtime actions needed by the executor.

Typed Ops must never turn into:

- arbitrary shell;
- arbitrary filesystem access;
- Docker API;
- SSH;
- arbitrary HTTP fetch;
- package management.

## 7. New feature gate

Before adding a new TX-Node feature, answer these questions in order:

1. Is this Control Plane orchestration or a business fact?  
   If yes, it belongs in TXBoard.

2. Is this install/upgrade/rollback/host lifecycle?  
   If yes, it belongs in TX-Node-Installer.

3. Is this proxy protocol/runtime behavior?  
   If yes, prefer the kernel adapter/runtime.

4. Is this an optional external integration?  
   If yes, prefer TXBoard Plugin/Integration or an isolated optional adapter.

5. Does TX-Node need this to safely execute or observe the Data Plane?  
   Only then should it enter TX-Node Core.

A feature being useful on a server is not sufficient reason to add it to TX-Node.

## 8. Simplification roadmap

### S1 — boundary + Node Ops isolation — complete

Completed in the first simplification PR:

- defined this runtime boundary;
- moved typed Node Ops dispatch/replay/log/network logic out of the Service god-object;
- preserved all Node Ops contracts and behavior;
- froze feature expansion of standalone/legacy management surfaces.

### S2 — Service decomposition — complete

S2 decomposes Service orchestration by responsibility without changing contracts.

Completed S2 slices:

1. REST snapshot polling extracted into `internal/nodesync.Controller`:

```text
Service
  -> supplies current config hash + cert renewal fact
  -> nodesync.Controller
       -> overlap prevention
       -> retry/backoff
       -> ControlPlane Poll
       -> config/user hashing
       -> immutable Result
  -> Service sync adapter validates/applies result
```

2. Push/WebSocket connection lifecycle extracted into `internal/pushsync.Controller`:

```text
ControlPlane Initial / Discover
        -> pushsync.Controller
             -> event/status channels
             -> PushClient start/stop
             -> connected/disconnected lifecycle
             -> delayed discovery eligibility
        -> Service push adapter
             -> logs
             -> REST reconcile request
             -> device-state clearing
             -> event application
```

The controllers own transport/synchronization mechanics only. Service remains authoritative for validating and applying data-plane state.

3. Desired user runtime state extracted into `internal/userstate.Controller`:

```text
ControlPlane users
      -> Service user adapter
      -> userstate.Controller
           -> desired user snapshot + hash
           -> limiter index
           -> speed-limiter index
      -> Service kernel mutation
           -> success keeps desired state
           -> failure restores previous snapshot
```

The controller owns only desired user state and derived limiter indexes. The proxy kernel remains the authoritative executor of applied users, tracked separately by `Service.appliedState`.

4. Report delivery lifecycle extracted into `internal/reporting.Controller`:

```text
Service report adapter
      -> prepares runtime payload
      -> reporting.Controller
           -> overlap prevention
           -> retry/backoff
           -> async/sync Sink.Report
      -> Service callback
           -> restore flushed traffic/devices on async failure
           -> success logging
```

The reporting controller owns delivery mechanics only. Runtime metric collection and tracker flush/restore stay behind the Service adapter so reporting cannot mutate unrelated data-plane state.

5. Disruptive kernel lifecycle extracted into `internal/kernellifecycle.Controller`:

```text
Service kernel adapter
      -> kernellifecycle.Controller
           -> Start
           -> Reload
           -> Stop / no-user stop
           -> applied runtime bookkeeping
      -> sing-box / Xray Kernel
```

The controller owns only disruptive lifecycle transitions and the last successfully applied full config/user snapshot. Atomic user add/remove/update remains in the existing kernel user API and is not duplicated.

6. Certificate coordination isolated behind `internal/certcoord.Coordinator`:

```text
Service / ControlPlane cert intent
        -> certcoord.Coordinator
             -> lifecycle + TLS material facts
             -> panel cert_config normalization
             -> legacy compatibility mapping
        -> existing cert.Manager
             -> self / file / content
             -> HTTP ACME / DNS ACME
             -> persistence / renewal
```

The coordinator is an adapter over the existing certificate runtime. It does not reimplement ACME, DNS providers, persistence, PEM validation or renewal.

S2 is complete. The top-level Service is now orchestration over dedicated synchronization, push, user-state, reporting, kernel-lifecycle and certificate adapters/controllers.

The next architecture stage is S3 optional-capability slimming. S3 must review each optional capability independently and preserve compatibility before any removal.

The top-level Service remains orchestration only.

### S3 — optional capability slimming — in progress

S3 reviews optional capabilities independently. The objective is dependency
direction and replaceability first; feature removal requires a separate
compatibility decision.

First S3 slice: Access Audit attachment isolation.

```text
Service
   -> auditcoord.Coordinator
        -> ControlPlane AuditTarget capability
        -> optional runtime audit hook
        -> existing audit.Reporter
```

The coordinator owns only configuration/identity mapping and optional runtime
attachment. The existing `audit.Reporter` remains authoritative for rule
refresh, matching, buffering and report transport. sing-box remains the only
current runtime that exposes the audit hook; Xray/standalone behavior is
unchanged.

This slice intentionally does not:

- change Access Audit HTTP endpoints or authentication;
- move audit rules/reporting into TXBoard Core;
- add audit support to Xray;
- stop or redesign the reporter goroutine;
- remove the embedded audit capability.

Second S3 slice: ACME / DNS provider catalog boundary.

The existing built-in DNS providers remain supported, but the catalog is now a
frozen compatibility surface rather than an open-ended Core growth point.

```text
Service runtime validation
        -> certcoord.ValidateNodeConfig
        -> built-in dnsproviders catalog
        -> existing cert.Manager DNS-01 runtime
```

Core Service no longer imports or queries the DNS-provider registry directly.
Provider-name validation stays with certificate coordination. The current
provider set and aliases remain unchanged and are regression-tested as a
compatibility set.

This slice intentionally does not:

- remove any built-in DNS provider;
- change DNS credentials or `dns_env`;
- change ACME behavior;
- create a new plugin/runtime loader inside TX-Node.

New provider expansion should require an explicit optional-integration design
instead of adding more provider code to Core by default.

Remaining S3 review:

- geo/routing assets.

Prefer adapters/delegation and remove nothing without a compatibility plan.

### S4 — legacy retirement proposal

Only after measured usage and a versioned migration plan:

- evaluate `xbctl`;
- evaluate legacy systemd layout;
- evaluate standalone feature growth;
- define a breaking release if removal is justified.

S4 is not authorized by this document alone.

## 9. Non-goals

This simplification effort does not:

- remove Xboard protocol compatibility;
- remove Xray;
- remove standalone mode immediately;
- remove certificate modes immediately;
- remove Access Audit immediately;
- change Node Protocol contracts;
- change Machine Runtime Update v1;
- change TXBoard data models;
- introduce a new plugin system inside TX-Node.

The first objective is dependency direction and responsibility clarity, not deletion for its own sake.
