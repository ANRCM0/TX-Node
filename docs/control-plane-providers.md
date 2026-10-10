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
