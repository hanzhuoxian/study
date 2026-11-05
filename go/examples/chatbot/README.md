# 角色扮演 Chatbot

Go 示例项目中的一个聊天系统骨架，支持 WebSocket 长连接接入、gRPC 流式大模型协议，以及 Redis Pub/Sub 多网关房间事件广播。

## 架构

```text
Browser ── WebSocket ──> Chatbot Gateway ── gRPC stream ──> Model Service
                              │
                              └──── Redis Pub/Sub ────> Other Gateways / Room Clients
```

- `main.go`：WebSocket 网关、房间管理、事件广播、Redis Pub/Sub 和模型流适配接口。
- `proto/chatbot.proto`：大模型服务的 gRPC 服务定义，模型端按 token/delta 返回内容。
- 当前 gateway 使用内置 `MockModel` 按字符流式回显，用于在模型服务接好前联调。替换它的 `ModelClient` 实现即可连接真实 gRPC 服务。
- 设置 `REDIS_ADDR` 后启用 Redis 房间广播；未设置时使用单实例内存广播。

## 启动

```sh
# 单实例联调（默认监听 :8080，无需 Redis）
go run ./chatbot

# 多网关模式（Redis 6+）
REDIS_ADDR=127.0.0.1:6379 CHATBOT_ADDR=:8080 go run ./chatbot
```

也可以用 `CHATBOT_ADDR` 指定监听地址，并通过 `REDIS_PASSWORD` 配置 Redis 密码。当前 Redis DB 固定为 0，可按部署需要改造为配置项。

浏览器打开 `chatbot/index.html`，连接 `ws://localhost:8080/ws?user_id=u1`。连接后先发送 join，再发送 chat：

```json
{"type":"join","room_id":"room-1","character_id":"tsundere"}
{"type":"chat","content":"今天过得怎么样？"}
```

事件按 WebSocket JSON 文本帧返回：

```json
{"type":"joined","room_id":"room-1","character_id":"tsundere","sender":"...","timestamp":"..."}
{"type":"token","room_id":"room-1","character_id":"tsundere","sender":"assistant","content":"（","timestamp":"..."}
{"type":"done","room_id":"room-1","character_id":"tsundere","sender":"assistant","timestamp":"..."}
```

事件类型包括 `joined`、`token`、`done` 和 `error`。前端应按同一轮回复聚合 `token.content`，在 `done` 时完成渲染。`user_id` 当前从 URL 读取，仅供示例，线上应从已验证的登录态获取，并校验用户对房间的访问权限。

## 接入真实模型 gRPC 服务

用项目匹配版本的 `protoc` 与 Go 插件生成 `proto/chatbot.proto` 的 Go stub。实现 `ModelClient.Stream`：创建 gRPC `Generate` 流、发送 `ModelRequest` 字段、循环 `Recv()`，每个 `delta` 调用传入的回调；收到结束标志后正常返回。然后在 `main` 中将 `MockModel{}` 替换成 gRPC adapter。协议定义预留 `history`，由业务层读取对话历史后传入模型服务。

生产环境还应配置模型服务地址、调用凭证与超时；限制 WebSocket Origin 和输入长度；对身份、房间成员关系及角色 ID 做服务端校验；持久化对话历史。Redis Pub/Sub 是在线事件分发，不提供离线消息和历史回放，断线恢复需要额外的消息存储方案。
