# 晚安计划 · 免费公测版

一个七天晚间情绪收尾 H5：记录一件消耗自己的事，获得简短回应，通过揉掉或封存完成仪式，并在七天后查看非诊断性的回顾。公测期间七天全部免费，无需激活码。

## 本地启动

需要 Go 1.26、Node.js 和运行中的 PostgreSQL。

```bash
createdb unwind
createdb unwind_test
npm install
PUBLIC_BETA=true PUBLIC_DAILY_REPLY_LIMIT=20 make run
```

打开 `http://localhost:8080`。未配置模型时使用明确标识的人工晚安卡片。生产环境建议使用默认的每日 3 次设备限额，20 次仅用于本地完整流程测试。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DATABASE_URL` | `postgresql:///unwind?host=/tmp` | PostgreSQL 连接 |
| `PORT` | `8080` | HTTP 监听端口 |
| `PUBLIC_BETA` | `true` | 开启后七天免费；设为 `false` 恢复旧付费权益逻辑 |
| `PUBLIC_DAILY_REPLY_LIMIT` | `3` | 单设备每 24 小时回应次数 |
| `AI_DAILY_LIMIT` | `100` | 单实例每日模型调用上限，耗尽后使用人工文案 |
| `COOKIE_SECURE` | `false` | HTTPS 线上环境必须设为 `true` |
| `AI_BASE_URL` | `https://api.openai.com/v1` | OpenAI 兼容服务根地址 |
| `AI_API_KEY` | 空 | 模型密钥；未配置时使用人工文案 |
| `AI_MODEL` | `gpt-4o-mini` | 模型名称 |
| `WEB_DIR` | `web` | 静态资源目录，部署时不可遗漏 |
| `ACTIVATION_CODES` | `UNWIND-DEMO` | 仅旧付费模式使用，公测无需配置 |

## 线上发布检查

1. 执行 `go build -o unwind ./cmd/server`，部署二进制和 `web/` 目录，服务启动时自动执行幂等迁移。
2. 配置生产 `DATABASE_URL`、`PUBLIC_BETA=true`、`COOKIE_SECURE=true`、模型密钥及每日额度。
3. 使用 HTTPS 和同源反向代理，将 `/healthz` 作为健康检查；不要把密钥或数据库连接暴露给前端。
4. 初始建议 `PUBLIC_DAILY_REPLY_LIMIT=3`、`AI_DAILY_LIMIT=100`，根据真实调用成本调整。
5. 设备与模型额度都是单实例内存计数，服务重启会重置；多实例部署时模型总上限约等于单实例额度乘实例数。
6. 匿名设备限流不是强身份防刷，公开发布时保留全局模型上限，并观察异常流量和费用。

## 验证

```bash
make test
make coverage
npx playwright install chromium
npm run test:e2e
make smoke
```

`npm run test:e2e` 和 `make smoke` 需要服务以 `PUBLIC_BETA=true PUBLIC_DAILY_REPLY_LIMIT=20` 启动。覆盖率命令使用 `TEST_DATABASE_URL`，默认连接本机 `unwind_test`。

## 隐私与安全

原始倾诉只用于当次模型请求，不写入数据库。数据库保存匿名设备摘要、情绪标签、仪式、完成时间，以及用户主动提交的第 2/7 天反馈。检测到自伤等高风险内容时不调用模型，而返回现实求助建议。本产品不提供心理诊断或治疗。
