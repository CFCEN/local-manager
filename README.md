# Local Dev Workspace Backend

Go + SQLite 本地开发工作台后端，默认监听 `127.0.0.1:9527`。除配置、日志、应用和 Git 管理外，后端还作为 Claude Automation 的中心服务：接收 Adapter 的出站 WebSocket、持久化会话与事件，并向浏览器提供 REST 和 WebSocket。

## Claude Automation 架构

```text
Claude Code hooks / transcripts
  → local-proxy claude-adapter
  → GET /api/claude-automation/adapters/ws
  → AutomationHub + SQLite
  → REST / browser WebSocket
  → React Claude Automation 页面
```

Adapter 主动连接管理后端，后端无需连接用户终端，也不会直接修改 Claude Code transcript。

## 配置

启动时默认读取 backend 目录下的 `config.json`，可从 `config.example.json` 复制一份后修改。`config.json` 已加入 gitignore，不会提交本地 token。

```json
{
  "port": "9527",
  "dbPath": "data/workspace.db",
  "workspaceRoot": "",
  "automationToken": "replace-with-a-dedicated-token",
  "host": "127.0.0.1",
  "allowedOrigins": ["http://localhost:3000"]
}
```

也可以通过 `BACKEND_CONFIG` 指定配置文件路径。环境变量 `PORT`、`DB_PATH`、`WORKSPACE_ROOT`、`HOST`、`AUTOMATION_TOKEN`、`ALLOWED_ORIGINS` 会覆盖配置文件值。`AUTOMATION_TOKEN` 必须非空才能访问 Claude Automation API；Adapter 的 `remote.token` 和前端的 `VITE_AUTOMATION_TOKEN` 必须使用同一值；不要复用上游模型 API key。

## Claude Automation API

所有 REST 请求使用：

```text
Authorization: Bearer <AUTOMATION_TOKEN>
```

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/claude-automation/sessions` | 获取会话列表，可使用 `after`、`sessionId` 查询参数 |
| `GET` | `/api/claude-automation/sessions/{sessionId}` | 获取会话和待处理操作 |
| `GET` | `/api/claude-automation/sessions/{sessionId}/events` | 获取已持久化的会话事件，可使用 `after` 游标 |
| `POST` | `/api/claude-automation/sessions/{sessionId}/messages` | 接受消息并异步派发 `session.message` |
| `POST` | `/api/claude-automation/sessions/{sessionId}/actions/{actionId}/respond` | 批准、拒绝或继续待处理操作 |
| `GET` | `/api/claude-automation/adapters/ws` | Adapter WebSocket |
| `GET` | `/api/claude-automation/ws?sessionId=...&token=...` | 浏览器会话或 `*` 全局订阅 WebSocket |

### 发送消息

请求：

```json
{
  "content": "继续处理这个问题"
}
```

`content` 会 trim，长度必须为 1–32000 个字符。成功立即返回 `202`，不等待 Claude 完成本轮：

```json
{
  "data": {
    "requestId": "message-...",
    "accepted": true
  }
}
```

找不到 session 返回 `404`；Adapter 离线、会话仍在运行或同一 session 已有 continuation 时返回 `409`。

### 处理待办

请求：

```json
{
  "decision": "approve"
}
```

`decision` 仅允许 `approve`、`reject`、`continue`。操作采用一次性状态迁移，重复或迟到响应返回冲突；Adapter 确认 Hook 已恢复后才写入最终结果。

## WebSocket 协议摘要

Adapter 首帧为 `hello`，随后可发送：

- `snapshot`、`session.status`：会话快照和生命周期状态。
- `event`：实时会话事件。
- `action.pending`：权限或计划确认；后端先持久化再广播。
- `command.result`：`sessions.read`、`session.message`、`action.respond` 等结果。
- `ping` / `pong`：心跳。

后端向 Adapter 发送 allowlist `command`。人工审批不占用普通命令的 10 秒同步等待；断线或超时由 Adapter fail-closed。

## 历史与状态

- 首次读取尚未入库的历史时，后端调用 `sessions.read`，过滤主链 user/assistant 记录并写入 `automation_events`。
- 后续读取直接来自 SQLite，Adapter 暂时离线时仍可查看已有历史。
- pending action 会将会话展示状态覆盖为 `waiting`。
- Adapter 的 lifecycle Hook 和浏览器 WebSocket 用于增量状态更新；REST 用于初始快照和重连补拉。

## 验证

```bash
gofmt -w internal/api/automation_handler.go internal/service/automation_service.go internal/repository/automation_repo.go
go test ./...
go build -o server ./cmd/server
```

服务启动、停止和重启由操作者手动执行。
