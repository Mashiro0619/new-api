# AGENTS.md - new-api 开发约定

## 项目约定

- 后端使用 Go、Gin 和 GORM，按 `Router -> Controller -> Service -> Model` 分层。
- 前端位于 `web/`，使用 React、TypeScript、Rsbuild 和 Bun；前端改动同时遵循 `web/AGENTS.md`。
- 优先沿用现有模块、工具函数和代码风格，保持改动范围小且直接，避免无关重构。
- 不在代码、日志、文档或配置中提交密钥、令牌和生产凭据。
- 遵守 `LICENSE`、`NOTICE` 和第三方许可证要求，不删除依法需要保留的版权及许可声明。

## 后端

### relaykit 独立性

- `relaykit/` 必须能脱离根模块独立构建，不得导入根模块包或依赖根模块配置、生成文件及 workspace。
- 修改 `relaykit/` 或其公开 API 后，运行：

```sh
cd relaykit && GOWORK=off go build ./... && GOWORK=off go test ./...
```

### JSON

- 业务代码通过 `common/json.go` 中的 `Marshal`、`Unmarshal`、`UnmarshalJsonStr`、`DecodeJson` 等封装处理 JSON。
- 可以使用 `encoding/json` 的类型，但不要直接调用其 marshal/unmarshal 函数。

### 数据库

- 所有数据库改动必须同时兼容 SQLite、MySQL >= 5.7.8 和 PostgreSQL >= 9.6。
- 优先使用 GORM；让 GORM 生成主键。必须使用原生 SQL 时，为三种数据库提供有效实现或回退。
- 标准行锁统一使用 `lockForUpdate(tx)`，不要使用已失效的 `gorm:query_option` 或在调用处重复实现锁逻辑。
- 原生 SQL 使用 `commonGroupCol`、`commonKeyCol`、`commonTrueVal`、`commonFalseVal` 以及 `common.UsingMainDatabase` / `common.UsingLogDatabase` 处理方言差异。
- 迁移必须跨数据库可执行；不要引入缺少兼容方案的数据库专属语法、函数或 JSON 类型。
- 业务默认值优先在规范化、Hook、构造或服务逻辑中设置，避免仅为业务规则添加可能导致重复迁移的 GORM 布尔默认标签。

### Relay DTO

- 客户端请求会重新序列化给上游时，可选标量使用指针类型和 `omitempty`，确保缺省值被省略、显式 `0` 和 `false` 被保留。
- 新增渠道时确认是否支持 `StreamOptions`；支持时同步更新 `streamSupportedChannels`。

### 计费安全

- 修改动态计费前先阅读 `pkg/billingexpr/expr.md`。
- 所有来自用户、上游、媒体元数据或透传字段的计费乘数都必须校验上界；复用 `dto.MaxImageN`、`relaycommon.MaxTaskDurationSeconds`、`maxTokensLimit` 等现有常量。
- 配额和 token 数转换使用 `common/quota_math.go`，计费路径优先使用 `*Checked` 版本并把饱和信息写入 `relayInfo.QuotaClamp` 或任务结算上下文，最终通过 `attachQuotaSaturation` 记录审计信息。
- 比率只能通过 `types.PriceData.AddOtherRatio` 写入，不要直接修改 `OtherRatios`。
- 检查完整链路：请求验证、费用估算、配额转换、预扣费、结算或退款；任何溢出或异常输入都不得产生负数费用。

### 后端测试

- 测试应保护真实行为、API 契约、数据兼容性、计费不变量或明确的回归路径，避免仅追求覆盖率或断言实现细节。
- 使用确定性输入并显式初始化数据库、缓存、设置和请求上下文；不要依赖随机数、固定延时或测试执行顺序。
- 新增或大幅改写的 Go 测试使用 `testify/require` 做前置和致命断言，使用 `testify/assert` 做非致命断言。
- 根据影响范围运行相关测试；涉及共享后端行为时运行 `make test`。

## 前端

- 使用 Bun 安装依赖和运行脚本，不混用 npm、Yarn 或 pnpm。
- 所有用户可见文案必须接入 i18n；新增或修改翻译时同步维护全部支持的语言。
- 修改 TypeScript 或 TSX 后至少运行相关测试、`bun run typecheck` 和 `bun run lint`；发布相关改动还需运行 `bun run build`。
- 组件、样式、无障碍、测试和文件组织的详细规则见 `web/AGENTS.md`。

## 验证原则

- 验证范围与改动风险相匹配，并在交付时说明实际运行过的检查。
- 不声称未运行的测试已经通过；若环境限制导致无法验证，明确说明。
