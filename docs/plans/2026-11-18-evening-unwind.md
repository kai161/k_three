# 7 天收工晚安计划实现计划

> **致 Claude：** 必须使用子技能 dev-executing-plans 逐任务执行此计划。

**目标：** 交付一个可本地运行、可测试的移动端 H5 产品，完成匿名体验、两天免费、激活解锁、七天仪式与回顾，并在真实产品走查后修正体验问题。

**架构：** Go 标准库 HTTP 服务提供 JSON API 并托管无构建步骤的 HTML/CSS/JavaScript 前端。PostgreSQL 使用 pgx 驱动持久化会话、激活码和每日记录；领域服务通过接口隔离数据库与模型，模型使用 OpenAI 兼容 HTTP 适配器，测试使用可控假实现。原始倾诉只在请求内使用，不写入数据库或应用日志。

**技术栈：** Go 1.26、pgx/v5、PostgreSQL 17、原生 ES Modules、HTML/CSS、Node 内置测试与 Playwright（若本机可用则用于浏览器验收）。

---

### 任务 1：项目骨架与七日领域规则

**文件：**
- 创建：`go.mod`
- 创建：`internal/plan/model.go`
- 创建：`internal/plan/service.go`
- 测试：`internal/plan/service_test.go`

**步骤：**
1. 编写失败测试，覆盖第 1～2 天免费、第 3 天需激活、同一天重复完成幂等、不能跳天、第 7 天完成后生成回顾。
2. 运行 `go test ./internal/plan`，预期因类型和函数不存在而失败。
3. 定义 `Entry`、`Progress`、`Store`、`Generator` 及 `Service`，以显式当前日期和领域错误实现最小规则。
4. 运行 `go test ./internal/plan`，预期全部通过。
5. 当前目录不是 git 仓库，不执行提交。

关键接口：

```go
type Store interface {
    Progress(ctx context.Context, deviceID string) (Progress, error)
    Complete(ctx context.Context, deviceID string, day int, mood string, ritual string) (Entry, error)
}
type Generator interface {
    Generate(ctx context.Context, day int, text string) (Reply, error)
}
```

### 任务 2：PostgreSQL 持久化与匿名会话

**文件：**
- 创建：`migrations/001_init.sql`
- 创建：`internal/store/postgres.go`
- 创建：`internal/store/postgres_test.go`
- 创建：`internal/httpapi/session.go`
- 测试：`internal/httpapi/session_test.go`

**步骤：**
1. 编写会话测试，验证首次访问发放高熵匿名令牌、后续读取 HttpOnly Cookie、非法令牌返回新会话。
2. 编写存储测试，验证唯一设备、激活码单次绑定、每日唯一记录和事务内幂等；无数据库环境时明确跳过集成测试，不用假结果替代。
3. 运行对应 `go test`，确认先失败。
4. 建立 `devices`、`activations`、`entries` 表；只存 `day/mood/ritual/timestamps`，不建原始倾诉字段。
5. 用 pgx 实现存储，并用加密随机会话 ID 的 SHA-256 摘要作为数据库键。
6. 运行 `go test ./internal/store ./internal/httpapi`，预期通过。

### 任务 3：模型适配、安全分流与人工兜底

**文件：**
- 创建：`internal/reply/generator.go`
- 创建：`internal/reply/openai.go`
- 创建：`internal/reply/safety.go`
- 测试：`internal/reply/generator_test.go`
- 测试：`internal/reply/openai_test.go`
- 测试：`internal/reply/safety_test.go`

**步骤：**
1. 编写失败测试：模型超时返回可识别兜底；坏 JSON 不进入仪式；成功响应包含回应、小行动和来源；高风险输入只返回求助信息。
2. 运行 `go test ./internal/reply`，确认失败。
3. 实现严格的回复长度与 JSON 校验、上下文超时、有限重试、人工文案池、来源标识及高风险分流。
4. 用 `httptest.Server` 验证 OpenAI 兼容请求不会泄露到其他地址，密钥只放请求头。
5. 运行 `go test ./internal/reply`，预期通过。

### 任务 4：完整 HTTP API 与限流

**文件：**
- 创建：`internal/httpapi/router.go`
- 创建：`internal/httpapi/handler.go`
- 创建：`cmd/server/main.go`
- 测试：`internal/httpapi/router_test.go`

**步骤：**
1. 编写端到端 API 失败测试：建立会话、生成第 1 天、完成第 1 天、重复完成、免费边界、激活、完成 7 天和回顾。
2. 增加隐私测试：生成接口返回后，存储假实现中不存在原始文字；请求日志测试不记录请求体。
3. 运行 `go test ./internal/httpapi`，确认失败。
4. 实现 `POST /api/session`、`GET /api/plan`、`POST /api/replies`、`POST /api/entries`、`POST /api/activate`、`GET /api/review` 和 `GET /healthz`。
5. 使用统一 JSON 错误、内容长度限制、每设备生成频率限制和幂等响应；激活失败不泄露激活码。
6. 运行 `go test ./internal/httpapi`，预期通过。

### 任务 5：移动端产品界面

**文件：**
- 创建：`web/index.html`
- 创建：`web/styles.css`
- 创建：`web/app.js`
- 创建：`web/api.js`
- 创建：`web/assets/unwind.svg`
- 测试：`web/app.test.mjs`

**步骤：**
1. 用 Node 测试驱动天数状态、字数限制、免费锁定和回顾渲染逻辑，运行 `node --test web/*.test.mjs` 确认失败。
2. 实现欢迎页、今日记录、回应卡片、揉掉/封存仪式、情绪选择、七天路径与回顾；所有动态文本使用 `textContent`。
3. 加入加载、失败重试、草稿恢复、减少动态效果偏好、键盘焦点和触屏大按钮。
4. 用清晰来源标签区分 AI 回应与人工晚安卡片，不做医疗承诺。
5. 运行 `node --test web/*.test.mjs`，预期通过。

### 任务 6：本地启动与自动验收

**文件：**
- 创建：`Makefile`
- 创建：`README.md`
- 创建：`scripts/smoke.sh`
- 创建：`e2e/product-flow.spec.mjs`（仅当项目现有 Node 测试工具支持浏览器时保留）

**步骤：**
1. 提供 PostgreSQL 初始化、迁移、服务启动、健康检查和测试命令，模型未配置时明确使用人工兜底模式。
2. 启动本地 PostgreSQL 与服务，通过真实 HTTP 请求走通 7 天流程，检查 Cookie、激活和回顾。
3. 在可用浏览器中走移动端视口完整流程，截取欢迎、回应、仪式、付费锁定和回顾页面。
4. 运行 `go test ./...`、`node --test web/*.test.mjs` 与冒烟脚本，预期全部通过。

### 任务 7：产品走查、建议与修复回归

**文件：**
- 按走查结果修改：`web/*`、`internal/*`
- 创建：`docs/plans/2026-11-18-product-review.md`

**步骤：**
1. 以首次用户视角记录理解成本、情绪负担、文案、动效、付费时机、错误恢复和隐私感受，每项附实际页面或请求证据。
2. 只选择影响首版核心体验的高价值问题，写明建议与取舍。
3. 先为每个确认问题添加失败回归测试，再修复。
4. 重新执行浏览器完整流程和全部测试，确认修复没有破坏七天规则。
5. 汇总已实现、已体验、已修复和仍需真实外部服务验证的部分。
