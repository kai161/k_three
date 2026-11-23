# 晚安计划免费公测版实现计划

> **致 Claude：** 必须使用子技能 dev-executing-plans 逐任务执行此计划。

**目标：** 交付可配置七天全免费、具备成本保护和轻量反馈能力的公测版本，同时保持旧付费模式兼容。

**架构：** 在计划服务和 HTTP 路由增加显式公测配置，状态向前端暴露 `publicBeta`。模型调用通过每日预算装饰器保护；反馈使用 PostgreSQL 持久化。原有匿名会话、七天顺序、原始倾诉不落库和高风险分流保持不变。

**技术栈：** Go 1.26、pgx/v5、PostgreSQL、原生 JavaScript、Node 测试、Playwright。

---

### 任务 1：公测权益与状态契约

**文件：** `internal/plan/model.go`、`internal/plan/service.go`、`internal/plan/service_test.go`、`internal/httpapi/router.go`

1. 保留现有付费测试，新增失败测试：公测第 3～7 天无需激活、状态含 `publicBeta=true`、旧模式仍返回 `ErrLocked`。
2. 运行 `go test ./internal/plan ./internal/httpapi`，确认因公测配置或字段缺失而失败。
3. 为服务增加只读公测选项和状态字段；旧构造函数保留原行为，新构造函数接收公测配置。
4. 为路由增加配置构造函数，旧 `NewRouter` 行为不变。
5. 运行测试，预期新旧模式全部通过。

### 任务 2：公测频率与模型预算

**文件：** `internal/httpapi/router.go`、`internal/httpapi/router_test.go`、`internal/reply/budget.go`、`internal/reply/budget_test.go`、`cmd/server/main.go`

1. 新增失败测试：公测限流窗口为 24 小时且可配置；预算内调用模型，额度耗尽返回人工文案，高风险输入不消耗模型预算。
2. 运行 `go test ./internal/httpapi ./internal/reply`，确认失败。
3. 实现线程安全的单日计数装饰器和路由限流配置；模型错误继续由现有人工兜底处理。
4. 从 `PUBLIC_BETA`、`PUBLIC_DAILY_REPLY_LIMIT`、`AI_DAILY_LIMIT` 读取生产配置。
5. 运行测试，预期通过。

### 任务 3：反馈持久化与 API

**文件：** `migrations/002_feedback.sql`、`migrations/migrations.go`、`internal/store/postgres.go`、`internal/store/postgres_test.go`、`internal/httpapi/handler.go`、`internal/httpapi/router.go`、`internal/httpapi/router_test.go`

1. 新增失败测试：第 2/7 天可提交 1～500 字反馈，空内容和其他天数返回 400，同设备同天重复提交更新而非新增。
2. 运行 Go 测试，确认因反馈接口缺失而失败。
3. 建立反馈表和存储方法，设备外键与现有匿名设备一致，唯一约束为 `(device_id, day)`。
4. 实现 `POST /api/feedback`，反馈失败不影响既有完成接口。
5. 使用真实 PostgreSQL 运行存储和 API 测试，预期通过。

### 任务 4：公测前端与反馈体验

**文件：** `web/state.js`、`web/state.test.mjs`、`web/api.js`、`web/app.js`、`web/index.html`、`web/styles.css`

1. 新增失败前端测试：`publicBeta=true` 时第 3 天进入书写页；旧模式仍进入锁定页。
2. 实现公测状态判断，隐藏激活流程，增加第 2 天完成页和第 7 天回顾页的可选反馈表单。
3. 反馈明确“反馈会保存，原文倾诉不会保存”，提供跳过和提交状态；动态内容使用 `textContent`。
4. 运行 `npm test`，预期通过。

### 任务 5：公测端到端与发布准备

**文件：** `e2e/product-flow.spec.mjs`、`e2e/global-setup.mjs`、`scripts/smoke.sh`、`README.md`、`Makefile`、`docs/plans/2026-11-20-public-beta-design.md`

1. 将主浏览器流程改为七天无激活，增加第 2/7 天反馈断言；保留旧模式单元测试作为付费兼容证据。
2. 冒烟测试不再激活，验证七天和反馈；公测测试环境提高频率上限，生产默认仍为 3。
3. 更新启动与线上环境变量、HTTPS、单实例预算边界和发布检查清单。
4. 启动最新服务，运行 `go vet ./...`、`go build ./...`、Go/前端/PostgreSQL 测试、冒烟和 Playwright。
5. 检查覆盖率不低于 70%，清理测试产物，不启动长期后台服务，由用户自行发布。
