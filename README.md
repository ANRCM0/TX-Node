# TX-Node

TX-Node 是 TXBoard 体系中的独立 **Agent / Data Plane Runtime**，同时保留 Xboard 协议兼容。它负责 Control Plane 通信、Machine / Node 编排、Kernel 生命周期、用户与流量策略执行、运行状态上报和有限 typed ops；它不是第二个 Control Plane，也不是通用主机管理 Agent。

TX-Node 支持 `sing-box` / `xray-core` 双内核。安装、升级、回滚与主机部署生命周期继续由公开的 TX-Node-Installer 负责。

运行时职责收口规则见 [`docs/runtime-boundary.md`](docs/runtime-boundary.md)。

项目起源于 [cedar2025/Xboard-Node](https://github.com/cedar2025/Xboard-Node)。TX-Node 保留 Xboard 面板 API、认证字段和现有节点配置的协议兼容，但不再以周期性同步上游作为开发模式；上游后续修复会按需审查并选择性移植。独立维护策略见 [`docs/standalone.md`](docs/standalone.md)。

## Control Plane Provider 支持状态

普通节点与 Machine Mode 均通过 `panel.provider` 选择远程控制平面；省略该字段或设置为 `xboard` 时使用现有 Xboard 兼容协议。`txboard` 启用 TXBoard 原生 `/txapi/node/v1` HTTP/WSS 协议（Node 与 Machine Mode）；原生 WebSocket 默认由 TXBoard 服务端关闭，不影响 HTTP 轮询。上线前仍需跨仓库真实环境联调。Standalone 模式使用本地控制平面。

Provider 的构造错误处理、配置继承及原生协议验收边界见 [`docs/control-plane-providers.md`](docs/control-plane-providers.md)。

## 安装与部署

TX-Node 当前为**公开的运行时源码仓库**，负责源码、构建、测试与发布。安装、升级、多面板实例、隔离实例和主机运维脚本则统一由独立的公开安装器仓库维护：

**TX-Node Installer**  
https://github.com/ANRCM0/TX-Node-Installer

推荐安装入口：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/ANRCM0/TX-Node-Installer/main/deploy.sh)
```

TXBoard 生成的非交互安装命令同样使用公开 Installer。运行时镜像仍由本仓库 CI 构建并发布到：

```text
ghcr.io/anrcm0/tx-node:latest
```

TX-Node 运行时仓库与 TX-Node-Installer（**两者均公开**）的职责边界：

- **TX-Node**：Go 运行时、ControlPlane、Machine、内核适配、审计客户端、配置模型、测试、Docker 镜像与二进制发布。
- **TX-Node-Installer**：安装/升级/卸载、Docker Compose、`txnode` 运维命令、多面板 `instances:`、隔离实例、配置备份与回滚。
- 运行时仓库不再维护 `deploy.sh` / `install.sh` 的副本，避免安装逻辑双轨演进。

开发者需要直接运行二进制时，可以参照 [config.yml.example](config.yml.example) 复制并填写凭据：

```bash
make build
cp config.yml.example config.yml
# 编辑 config.yml 中的 panel.url、token 和 node_id
./tx-node -c ./config.yml
```

常用开发/发布命令：

```bash
make test
make build
make build-all
make docker
```

## 镜像发布策略

TX-Node 的 CI 分离了验证、测试镜像和正式发布，普通合并不会直接覆盖生产使用的 `latest`：

- **合并/推送 `main`**：运行 Go 测试与稳定性测试；**不推送 Docker 镜像**。
- **手动构建测试镜像**：到 [CI Actions](https://github.com/ANRCM0/TX-Node/actions/workflows/ci.yml) 选择 `main`，点击 **Run workflow**。成功后推送 `ghcr.io/anrcm0/tx-node:test` 和精确提交 SHA 标签，**不会修改 `latest`**。
- **正式发布**：将经过验证的源码打上符合语义化版本的 `vX.Y.Z` 标签。tag CI 完成测试、双架构构建和镜像发布，推送 `ghcr.io/anrcm0/tx-node:vX.Y.Z` 与 `:latest`，并生成 GitHub Release。预发布标签（例如 `v2.3.0-rc.1`）不更新 `latest`。
- **可选的版本发布向导**：[Publish semantic release](https://github.com/ANRCM0/TX-Node/actions/workflows/publish-release.yml) 只接受人工从 `main` 触发。使用前必须更新 `.github/release/VERSION` 与 `.github/release/NOTES.md` 到相同的新版本并清除过时资产说明。仅编辑这些文件不会自动发布。
- 已安装的节点不会因为 GHCR 标签更新而自行升级；实际升级仍由 TX-Node-Installer 管理。

## 访问审计（可选、兼容性功能）

TX-Node 保留 `sing-box` 的嵌入式访问审计 reporter，但它不是原生 TXBoard 节点协议的一部分；默认关闭，且不参与正常用户、配置、流量计费与节点状态同步。

| 部署模式 | 访问审计支持情况 |
| --- | --- |
| Xboard 兼容 Provider (`panel.provider: xboard`) | 保留旧版可选 reporter，要求面板**实际提供并启用**兼容 `/api/v1/plugin/access-audit/rules` 与 `/report` 的插件接口 |
| TXBoard 原生 Provider (`panel.provider: txboard`) | **当前不支持旧审计上报**；保持 `audit.enabled: false` |
| Xray / Standalone | 无内置的同等审计能力；不要把嵌入式 sing-box reporter 视为通用功能 |

当前 TXBoard `main` **没有** `integrations/AccessAudit` 目录，且原生 `/txapi/node/v1` 未定义旧插件审计接口。因此不再提供原先的失效源码链接，也不将该插件作为 TXBoard 的内置能力宣传。历史兼容插件的来源、安装方式和接口可用性须由实际部署独立验证。

仅在已验证的 Xboard 兼容部署中，可显式选择：

```yaml
panel:
  provider: xboard
  url: "https://xboard.example.com"
  token: "REPLACE_WITH_SERVER_TOKEN"
  node_id: 1
audit:
  enabled: true
  report_all: false
  batch_max: 50
  flush_interval: 15
  rules_refresh: 5
  queue_cap: 5000
```

`report_all: false` 只上报命中规则的连接；如果面板没有下发启用规则，则不会产生审计上报。`report_all: true` 会显著增加事件量；本地队列受容量限制，拥塞时可能丢弃审计事件。**审计日志不作为流量计费依据**。

相关实现：[审计 reporter](internal/audit/reporter.go)、[ControlPlane 审计能力接口](internal/controlplane/capabilities.go)。原生 TXBoard 访问审计如需支持，必须先单独定义其 API、鉴权和行为契约，不应直接复用旧 Query Token 机制。

## Xboard 兼容与项目来源

TX-Node 保留 Xboard 面板协议及兼容配置字段。`xbctl` 和 `xboard-node` 二进制兼容产物已从 TX-Node v2 主线移除。历史 native/systemd 安装仍可由 TX-Node-Installer 识别、迁移和清理；新的 Docker 运维入口统一为 `deploy.sh` / `txnode`。

项目历史来源于 [cedar2025/Xboard-Node](https://github.com/cedar2025/Xboard-Node)。后续 TX-Node 版本独立维护和发布；上游修复仅按需审查、移植，不再整分支同步。详见 [`docs/standalone.md`](docs/standalone.md)。

## License

MPL-2.0。项目保留其历史来源及适用的上游版权与许可证声明。

> **Disclaimer**: This project is for educational and learning purposes only.
