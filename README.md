# TX-Node

TX-Node 是 TXBoard 体系中的独立 **Agent / Data Plane Runtime**，同时保留 Xboard 协议兼容。它负责 Control Plane 通信、Machine / Node 编排、Kernel 生命周期、用户与流量策略执行、运行状态上报和有限 typed ops；它不是第二个 Control Plane，也不是通用主机管理 Agent。

TX-Node 支持 `sing-box` / `xray-core` 双内核。安装、升级、回滚与主机部署生命周期继续由公开的 TX-Node-Installer 负责。

运行时职责收口规则见 [`docs/runtime-boundary.md`](docs/runtime-boundary.md)。

项目起源于 [cedar2025/Xboard-Node](https://github.com/cedar2025/Xboard-Node)。TX-Node 保留 Xboard 面板 API、认证字段和现有节点配置的协议兼容，但不再以周期性同步上游作为开发模式；上游后续修复会按需审查并选择性移植。独立维护策略见 [`docs/standalone.md`](docs/standalone.md)。

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

开发者需要直接运行二进制时，可使用：

```bash
make build
./tx-node -c ./config.yml.example
```

常用开发/发布命令：

```bash
make test
make build
make build-all
make docker
```

## 审计说明

| 项 | 行为 |
|---|---|
| 启用方式 | config.yml 加 `audit: enabled: true`（**必须挂配置文件**，纯环境变量模式无法开启审计） |
| 认证 | 复用 `panel.url` / `token` / `node_id`（与原版节点上报同一套 ServerV2 认证，零额外密钥） |
| 数据流 | 连接路由时提取（user_id, 目标域名/IP, 来源 IP）→ 节点本地按面板下发的规则预过滤 → 只上报命中项 |
| 批量 | 攒批 50 条 / 15 秒上报一次；一次 tick 内连续发送直到队列清空（上限 10 批），面板不可达时本地排队（上限 5000 条），恢复后自动补报 |
| 内核范围 | **仅 sing-box**。xray 内核请用 AccessAudit 插件自带的旁路 `audit-agent.py`（tail access log） |
| 目标提取 | sniff 域名 > 代理协议自带域名 > 目标 IP（代理协议自带域名，绝大多数场景不依赖 sniff） |

可调参数（都有默认值）：

```yaml
audit:
  enabled: true
  report_all: false    # true = 上报全部连接（含未命中），面板留存全量访问日志
  batch_max: 50        # 每次上报最多事件数（report_all 默认 200；面板单批上限 500）
  flush_interval: 15   # 上报间隔（秒）
  rules_refresh: 5     # 规则拉取间隔（分钟）
  queue_cap: 5000      # 面板不可达时的本地队列上限（report_all 默认 50000）
```

> `batch_max` 上限为 **500**（面板上报接口 `MAX_EVENTS` 限制），超过会被 422 拒绝整批。

### ⚠️ 最常见的坑：开了 `enabled` 却一条数据都没有

**`report_all: false` 时，唯一能上报的只有「命中规则」的连接。如果面板上一条启用规则都没有，节点会丢弃每一个连接，一条数据都不上报** —— 而配置和启动日志看起来一切正常。

原因：节点在本地按面板下发的规则预过滤（省面板流量），规则集为空 → `match()` 恒为 false → 全部丢弃。

所以只有两种有效组合：

| 目标 | 配置 |
|---|---|
| 只要**违规命中**记录（量小） | `report_all: false` **且面板上至少配一条启用规则** |
| 要**全量访问日志**（面板能看到所有连接） | `report_all: true`（无需配置规则，规则仅用于标记哪条算命中） |

节点启动时会打印一条 WARN 提醒这个状态；规则拉取到空集且 `report_all=false` 时也会再告警一次（每分钟最多一条）：

```
WARN [core] audit: report_all=false — only rule-matched targets are reported;
           with no enabled rules NOTHING will be sent. ...
```

看到这条日志就说明当前配置不会产生任何上报。

### 负载与容量边界（重要）

**单个节点对面板的压力上界是确定的**，可以按下面的公式估算后再决定是否开启 `report_all`：

```
节点发送速率上限 = batch_max × 10 批 / flush_interval(秒)   [条/秒]
```

默认值下：`50 × 10 / 15 ≈ 33 条/秒`；`report_all` 默认值下：`200 × 10 / 15 ≈ 133 条/秒`。

面板侧每条上报的 SQL 次数（无论批内多少条）为固定开销 + `ceil(N/200)` 次批量插入，**不再随事件数线性放大**。但事件**产生**速率是随在线用户数线性增长的：

| 规模 | 事件产生速率（估算） | report_all 默认配置是否跟得上 |
|---|---|---|
| 1000 在线用户，人均 3 并发，连接均值 300s | ≈ 10 条/秒 | 跟得上 |
| 5000 在线用户，同假设 | ≈ 50 条/秒 | 跟得上（接近上限） |
| 20000 在线用户，同假设 | ≈ 200 条/秒 | **跟不上**，队列会持续堆积并最终丢弃 |

队列满或补报失败溢出时，**事件会被丢弃并记录 `dropped` 计数与限流告警**（每分钟最多一条 WARN），不会静默丢失。

**建议**：
- 只要「命中项上报」（默认模式）时，事件量极小，任何规模都无需调整。
- 需要**全量访问日志**（`report_all`）且在线用户数超过 ~5000 时，请同时调大 `batch_max`（如 500）与 `flush_interval`，或在面板侧接受日志采样；单节点无法保证不丢时，日志仅适合做抽样审计，不适合做计费依据。

## 配套面板插件（AccessAudit）

AccessAudit 是**可选的面板插件**，其唯一源码位于 TXBoard：

- 插件源码：[ANRCM0/TXBoard → integrations/AccessAudit](https://github.com/ANRCM0/TXBoard/tree/main/integrations/AccessAudit)
- TX-Node 只保留可选的审计 reporter/client，用于拉取规则和上报事件。
- TX-Node 不再 vendor、构建或随 Release 打包面板插件。
- 面板未安装/启用 AccessAudit 时，请保持 `audit.enabled: false`；核心节点、流量与状态上报不受影响。
- xray 兼容所需的旁路 `audit-agent.py` 作为插件资产由 TXBoard 的 AccessAudit 插件维护。

这样插件生命周期属于控制平面，TX-Node 的发布生命周期只负责 Agent/runtime。

## Xboard 兼容与项目来源

TX-Node 保留 Xboard 面板协议及兼容配置字段。`xbctl` 和 `xboard-node` 二进制兼容产物已从 TX-Node v2 主线移除。历史 native/systemd 安装仍可由 TX-Node-Installer 识别、迁移和清理；新的 Docker 运维入口统一为 `deploy.sh` / `txnode`。

项目历史来源于 [cedar2025/Xboard-Node](https://github.com/cedar2025/Xboard-Node)。后续 TX-Node 版本独立维护和发布；上游修复仅按需审查、移植，不再整分支同步。详见 [`docs/standalone.md`](docs/standalone.md)。

## License

MPL-2.0。项目保留其历史来源及适用的上游版权与许可证声明。

> **Disclaimer**: This project is for educational and learning purposes only.
