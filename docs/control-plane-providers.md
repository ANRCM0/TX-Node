# Control-plane provider selection

TX-Node uses `panel.provider` for remote protocol selection in both node mode and machine mode.

- Omitted or `xboard`: existing Xboard-compatible protocol (default).
- `txboard`: TXBoard native node/v1 adapter (Bearer + identity headers, versioned HTTP/WSS).
- Standalone mode: always selects the local control plane.

```yaml
panel:
  provider: xboard
  url: https://panel.example.com
  token: your-token
  node_id: 1
```

TXBoard native transport uses /txapi/node/v1 and does not send Xboard-style query/body credentials. Node/machine DTOs are normalized before reaching Service.

## S7 implementation and S8 handoff

Node mode and machine mode both select their remote provider from `panel.provider`. Unsupported values fail closed; both `xboard` and `txboard` are implemented. The checked constructors (`controlplane.NewForConfigChecked`, `service.NewChecked`, and `machine.NewChecked`) return errors to the startup path instead of panicking. Existing convenience constructors remain for legacy callers; new code should prefer checked constructors.

Multi-instance configuration inherits `panel.provider` from the parent when omitted and preserves explicit child values. Standalone selects the local control plane independently of remote provider selection, subject to configuration validation.

Native TXBoard v1 requires server-side `/txapi/node/v1` endpoints, the corresponding Bearer/ID identity, and optional Workerman WSS. HTTP 202 on usage report acknowledges queue acceptance only; SQL settlement must be verified in TXBoard. Test real kernel connectivity, queue failures, WSS upgrades and token rotation before a production rollout.
