# Control-plane provider selection

TX-Node uses `panel.provider` for remote protocol selection in both node mode and machine mode.

- Omitted or `xboard`: existing Xboard-compatible protocol (default).
- `txboard`: reserved for the future native adapter; config validation currently rejects it.
- Standalone mode: always selects the local control plane.

```yaml
panel:
  provider: xboard
  url: https://panel.example.com
  token: your-token
  node_id: 1
```

Provider selection does not imply TXBoard wire protocol support. Native TXBoard integration will require its own client, DTO mappings, and protocol contract tests.

## S7 implementation and S8 handoff

Node mode and machine mode both select their remote provider from `panel.provider`. Unsupported values fail closed; `txboard` is reserved, not implemented. The checked constructors (`controlplane.NewForConfigChecked`, `service.NewChecked`, and `machine.NewChecked`) return errors to the startup path instead of panicking. Existing convenience constructors remain for legacy callers; new code should prefer checked constructors.

Multi-instance configuration inherits `panel.provider` from the parent when omitted and preserves explicit child values. Standalone selects the local control plane independently of remote provider selection, subject to configuration validation.

Before enabling native TXBoard support, inspect the actual TXBoard node and machine models, REST authentication and endpoints, WebSocket event contract, state and traffic reporting, and the mapping to TX-Node internal DTOs. Add protocol contract tests for both node and machine modes. Do not assume TXBoard uses Xboard wire formats.
