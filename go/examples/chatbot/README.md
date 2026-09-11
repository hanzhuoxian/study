# WebSocket 聊天示例

启动服务：

```sh
go run ./chatbot
```

用浏览器直接打开 `chatbot/index.html`，点击“连接”即可聊天。可以同时打开两个页面验证广播；页面支持修改服务器地址、断开、重新连接和清空本地消息记录。

连接地址为 `ws://localhost:8080/ws`。聊天消息使用 JSON 文本帧，客户端发送：

```json
{"type":"chat","content":"你好"}
```

服务端向所有在线连接（包括发送者）广播：

```json
{
  "type": "chat",
  "sender": "连接的随机 ID",
  "content": "你好",
  "timestamp": "2026-09-11T08:00:00Z"
}
```

- `type`：目前只支持 `chat`。
- `sender`：服务端为每条连接生成的 ID，重连后改变，不代表已认证的用户。
- `content`：聊天内容，不能是空字符串或全空白。
- `timestamp`：服务端收到消息后的 UTC 时间，使用 RFC 3339 格式（可含小数秒）。

客户端提供的 `sender`、`timestamp` 不会被采用。非法 JSON、未知类型、空内容会被忽略并记录日志；二进制消息会被忽略，连接继续保留。原来的纯文本发送方式需要改为 JSON。

浏览器控制台示例（可以在两个页面分别运行）：

```js
const ws = new WebSocket("ws://localhost:8080/ws");
ws.onopen = () => {
  ws.send(JSON.stringify({ type: "chat", content: "你好" }));
};
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(message.sender, message.content, message.timestamp);
};
```

心跳仍使用 WebSocket Ping/Pong 控制帧，不属于上述 JSON 消息。服务端每 30 秒发送 Ping，收到 Pong 后将读截止时间延后 60 秒。浏览器自动回复 Pong；Gorilla 客户端需要持续读取连接来处理 Ping。
