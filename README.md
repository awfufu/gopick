# gopick

## 项目结构

- 标准入口：`cmd/gopick/main.go`
- 应用装配：`internal/app`
- 配置加载：`internal/config`
- HTTP 服务：`internal/httpserver`
- 领域模型：`internal/domain`
- 麦芽田客户端边界：`internal/maiyatian`
- 业务服务层：`internal/service`

## 启动

```bash
gopick -f config.yml
```

默认监听：`http://127.0.0.1:22800`

配置文件示例：[config-example.yml](./config-example.yml)

## 已提供接口

- `GET /health`
- `GET /status`
- `GET /order-context`
- `GET /list-orders`
- `GET /list-orders/{status}`
- `GET /all-orders`
- `GET /all-orders/{date}`

说明：

- `GET /list-orders` 已接到麦芽田 `order/list` 请求，并会继续按 `id` 拉取 `order/detail` 补全商品和配送信息
- `GET /order-context` 会请求固定的麦芽田 `order/` HTML 页面，并提取 `merchantId` 与 `accountId`
- `GET /all-orders` 已接到麦芽田 `query/list` 请求，默认返回当天订单
- `GET /all-orders/{date}` 支持按日期查询，日期格式为 `YYYY-MM-DD`，同样会补全商品和配送信息
- 服务启动后会自动连接 `wss://msg.maiyatian.com/acc`，使用 `merchantId/accountId` 登录，并每 30 秒发送一次心跳；断线后会自动重连，连接状态可通过 `GET /status` 查看
- 收到 ws 新单事件 `type=confirm` 后，服务会按 `id` 拉取订单详情，并主动上报到配置中的 `upload.base_url` `/api/v1/api-key/listened-orders`；新单上报时不会携带配送信息
