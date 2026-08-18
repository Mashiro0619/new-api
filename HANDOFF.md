# new-api 二开项目交接

更新时间：2026-08-19

## 1. 项目定位

- 个人仓库：`https://github.com/Mashiro0619/new-api.git`
- 官方上游：`https://github.com/QuantumNous/new-api.git`
- 开发主线：`main`
- 自有镜像：`ghcr.io/mashiro0619/new-api`
- 用户要求仓库和 GHCR Package 保持私有；本机无法核验远端可见性，接手后需在 GitHub 的仓库设置和 Package 设置中分别复核。不得提交 GitHub Token、服务器凭据、数据库备份或生产 `.env`。
- 必须保留 `LICENSE`、`NOTICE`、第三方许可证、QuantumNous/New API 归属及依法要求的原项目链接。

Git remote 已配置为：

```text
origin    https://github.com/Mashiro0619/new-api.git
upstream  https://github.com/QuantumNous/new-api.git
upstream push = DISABLED
```

同步上游时在 `main` 合并，不重写已推送历史：

```sh
git fetch upstream
git switch main
git merge --no-edit upstream/main
# 重新检查 .github/workflows，防止上游发布工作流被带回 fork
# 验证后
git push origin main
```

## 2. 交接时 Git 状态

功能代码基线（提交本交接文件之前）：`7067a30f93c82dc39c4a1c5d651c01166dd5e260`

交接前本地 `main` 比 `origin/main` 超前以下 3 个功能提交，尚未推送：

```text
7852ab83 feat: add per-channel forced outbound protocol
99c5742a feat: show consumed balance in token trends
7067a30f feat: simplify public home and embed sign in
```

`HANDOFF.md` 自身会形成后续文档提交，因此接手时以以下命令重新确认实际 HEAD 和是否已推送：

```sh
git status -sb
git log -5 --oneline --decorate
```

不要自行推送。用户明确授权推送后，再将这些提交推到 `origin/main`，并检查 `Publish fork image to GHCR` 工作流：

```sh
git push origin main
```

## 3. 已完成的发布与部署链路

当前 `.github/workflows/` 只保留：

- `ci.yml`
- `publish-ghcr.yml`

旧 Docker Hub、官方 Release 和 Electron 发布工作流已经移除。GHCR 工作流支持：

- 推送 `main`、推送 `v*` 标签和手动触发
- `linux/amd64`、`linux/arm64`
- `main` 发布 `latest` 和完整 `sha-<40位提交哈希>`
- `v*` 发布同名版本标签
- Go、RelayKit、前端检查
- SBOM、provenance 和 keyless cosign 签名
- 使用 GitHub Actions 自带的 `GITHUB_TOKEN` 写入 GHCR

根目录的 `docker-compose.yml` 以及 `docker-compose.override.yml` 的默认变量组合均解析为：

```text
ghcr.io/mashiro0619/new-api:latest
```

部署覆盖接口为 `FORK_IMAGE` 和 `FORK_IMAGE_TAG`。仓库还保留 `deploy/update.sh <image-tag>`，但该脚本固定使用根目录的 `docker-compose.yml` 和 `docker-compose.override.yml`，不能直接用于当前生产服务器的 `compose.yaml` 和 `.deploy/new-api.fork.override.yml`。生产环境继续使用第 4 节的双 `-f` 命令，除非后续先改造脚本支持自定义 Compose 文件。

### 私有 GHCR 约束

- 服务器使用 PAT classic 登录 GHCR 时，只授予 `read:packages`，无需 `write:packages` 或 `delete:packages`；Token 所属账号本身必须有该私有 Package 的读取权限。
- 在服务器执行一次 `docker login ghcr.io -u Mashiro0619` 后，Docker 会保存登录凭据。
- Token 被 GitHub 删除或撤销后，已保存的登录信息不能继续拉取私有镜像；服务器上已经存在的本地镜像仍可继续运行。
- 用户此前明确删除过旧 Token，当前没有证据证明服务器已配置新的有效 GHCR 凭据；下次执行 `pull` 前必须先复核或重新登录，不要把 Token 写入仓库、Compose 或交接文档。
- 用户明确偏好用 `latest` 简化日常更新。可以说明不可变 SHA 更容易审计和回滚，但不要在没有新要求时强制把日常流程改回 SHA。

## 4. 生产服务器最后一次已知状态

以下内容来自用户在 2026-08-18 提供的服务器执行报告，本地没有直接登录生产服务器复核。继续操作前必须重新检查。

- 项目目录：`/www/dk_project/wwwroot/newapi.mashiro.tech`
- 原 Compose：`compose.yaml`
- fork 覆盖文件：`.deploy/new-api.fork.override.yml`
- 当时应用镜像：`ghcr.io/mashiro0619/new-api:sha-6ab4d8d67658226550f2328a46c762b0dd43f4c6`
- 当时 `/api/status.data.version`：`main-6ab4d8d67658`
- PostgreSQL、Redis 容器未因镜像切换重建
- Hermes 直接运行在宿主机，不是 Docker 容器，更新 New API 时不要触碰
- 当时数据库备份：`/root/backup/newapi-pg-20260818-055134.dump`
- 当时业务数据：用户 2、令牌 1、渠道 29；普通 API 请求未用有效令牌完成最终验收

用户希望以后只用 `latest`。接手后先检查覆盖文件是否已经改成：

```yaml
services:
  new-api:
    image: ghcr.io/mashiro0619/new-api:latest
```

若仍固定在旧 SHA，先备份数据库并与用户确认服务器窗口，再修改覆盖文件。日常更新应保持 PostgreSQL、Redis 和 Hermes 不动：

```sh
cd /www/dk_project/wwwroot/newapi.mashiro.tech
docker compose -f compose.yaml -f .deploy/new-api.fork.override.yml pull new-api
docker compose -f compose.yaml -f .deploy/new-api.fork.override.yml up -d --no-deps new-api
```

生产环境执行 `pull`、`up`、`ps` 等 Compose 命令时必须同时带上这两个 `-f` 参数。原 `compose.yaml` 仍可能写着官方镜像；只运行 `docker compose pull` 或只传原文件会绕过 fork 覆盖配置。

更新后至少验证：

```sh
docker compose -f compose.yaml -f .deploy/new-api.fork.override.yml ps
curl -fsS http://127.0.0.1:3000/api/status
```

同时检查登录、实际 API 调用、应用日志，以及 PostgreSQL/Redis 容器 ID 未变化。不要执行 `docker compose down`、`prune` 或删除数据卷。

## 5. 已完成的管理员用户分析与 Token 趋势

主要功能已在 `6ab4d8d6` 及后续提交中完成：

- 管理员用户分析首页：`/dashboard/users`
- 用户详情：`/dashboard/users/$userId`
- 管理员趋势接口：`GET /api/data/token-trend`
- 当前用户趋势接口：`GET /api/data/token-trend/self`
- 管理员指定用户接口：`GET /api/data/token-trend/users/:id`
- 使用记录页接入共享 Token 趋势面板
- `quota_data` 聚合输入、输出、缓存创建、缓存读取和追踪请求数
- 旧数据不伪装成 0；依赖 `token_metrics_count` 区分新旧统计
- 不查询 ClickHouse，不回填旧日志

最新提交 `99c5742a` 增加 `consumed_quota` 聚合，并在趋势摘要中用“消耗余额”替换“缓存创建”。注意：

- 这是趋势接口和界面展示调整。
- 缓存创建 Token 仍在后端统计数据中保留。
- 该提交没有重写缓存创建的计费公式。
- 余额通过 `quota_data.quota` 聚合，前端用 `formatQuota()` 展示。

关键文件：

```text
model/usedata_token_trend.go
controller/usedata_token_trend.go
router/api-router.go
web/src/features/token-trend/
web/src/features/dashboard/components/users/
web/src/routes/_authenticated/dashboard/users/
```

## 6. 已完成的首页改造

提交：`7067a30f`

默认首页现在为：

- 左侧/移动端主标题：`Mashiro AI 中转站`
- 右侧直接显示登录表单
- 注册开放且未启用自用模式时显示注册链接
- 默认首页隐藏左上角品牌和右上角重复登录按钮
- 自定义首页内容或 URL 已配置时，仍优先显示原有自定义内容

关键文件：

```text
web/src/features/home/index.tsx
web/src/features/home/__tests__/registration-entry.test.tsx
web/src/components/layout/components/public-header.tsx
```

## 7. 已完成的渠道强制出站协议

提交：`7852ab83`

渠道高级设置新增 `forced_outbound_format`，保存在现有 `Channel.Setting` JSON，无数据库迁移。

支持渠道类型：

- OpenAI，类型 1
- New API，类型 60

支持目标协议：

```text
openai
openai_responses
claude
gemini
```

实现行为：

- 模型映射后转换到目标协议
- 使用目标协议标准路径
- 响应转换回客户端原入站协议
- 同时覆盖流式和非流式
- 强制协议优先于全局/渠道请求体透传和 Chat 自动转 Responses
- 参数覆盖作用于最终出站 JSON
- 系统提示词在目标协议转换完成后按目标协议合并
- 保留 Base URL、Bearer、组织、代理、HTTP transport 和 Header Override
- Claude 只补非凭据 Header，不额外添加 `x-api-key`
- Gemini 不额外添加 `x-goog-api-key`
- 非文本、`/v1/completions`、Responses Compact 和 Alpha Search 保持原逻辑
- 每次跨渠道重试清空转换、usage、工具计费、拒绝原因和流状态
- 渠道测试与正式请求共用包装层
- 日志记录入站格式、强制目标、最终格式和转换链，不记录密钥

前端行为：

- `auto` 代表跟随渠道类型，保存时省略字段
- 仅 OpenAI/New API 可编辑
- 切换到不支持的渠道类型会重置为 `auto`
- 启用强制协议时自动关闭并禁用请求体透传
- Advanced Custom 提示继续使用逐路由转换器
- 七种语言已同步

关键文件：

```text
relay/forced_outbound.go
common/forced_outbound.go
relaykit/dto/channel_settings.go
model/channel.go
model/pricing.go
service/log_info_generate.go
web/src/features/channels/components/drawers/channel-mutate-drawer.tsx
web/src/features/channels/lib/channel-form.ts
docs/channel/other_setting.md
```

尚未用真实外部上游完成四协议全组合验收。目前 HTTP 集成矩阵使用本地 `httptest` 上游，覆盖 OpenAI Chat 非流式入站到 4 个目标协议、2 种渠道类型；其他入站协议和流式行为主要由转换层及定向单元测试覆盖，尚无完整的 4x4 流式/非流式 HTTP 集成矩阵。后续修改转换链时重点保护 usage、工具附加计费、流开始后的错误处理和跨渠道重试隔离。

## 8. 本地开发环境

交接时本地服务正在运行：

- 前端热更新：`http://localhost:5173`
- 后端：`http://localhost:3000`
- Docker 服务：`new-api-dev`、`new-api-dev-pg`、`new-api-dev-redis`
- 应用数据和 PostgreSQL 使用 Docker volume 持久化；开发 Redis 没有持久卷，重建 Redis 会清空其状态

启动或重建后端：

```powershell
docker compose -f docker-compose.dev.yml up -d --build new-api
```

启动前端：

```powershell
cd web
bun run dev -- --host 0.0.0.0 --port 5173
```

不要使用 `docker compose -f docker-compose.dev.yml down -v`，除非用户明确要求删除本地开发数据。

本机没有可直接调用的 Go，Go 格式化和测试此前通过 `golang:1.26.1-alpine` Docker 镜像执行。Bun 已安装。

## 9. 最近一次验证结果

针对上述 3 个功能提交，已经在同一工作区代码上执行：

- 根 Go 模块全包测试：通过
- RelayKit `GOWORK=off go build ./...`：通过
- RelayKit `GOWORK=off go test ./...`：通过
- 前端 Vitest：41 个测试文件、198 个用例通过
- `bun run typecheck`：通过
- `bun run build`：通过
- `bun run i18n:sync`：七种语言均为 0 missing / 0 extras / 0 untranslated
- 本次渠道文件定向 oxlint 和 oxfmt：通过
- `git diff --check`：通过
- 本地 `/api/status`：成功

2026-08-19 重新生成本文件时还确认了前端 `http://localhost:5173` 返回 HTTP 200、后端 `/api/status` 返回 `success: true`，三个开发容器均处于运行状态。

完整 `bun run lint` 当前会因仓库其他模块已有错误失败，例如 channel affinity、models preferences、setup 等文件。本次改动涉及的渠道文件定向 lint 已通过。不要声称全仓 lint 已通过，也不要为了本功能顺手重构所有 lint 历史问题。

## 10. 明确待办和已知风险

### 10.1 站内“检查更新”仍指向官方仓库

文件：

```text
web/src/features/system-settings/maintenance/update-checker-section.tsx
```

其中仍请求：

```text
https://api.github.com/repos/Calcium-Ion/new-api/releases/latest
```

所以管理后台会显示官方版本。当前 fork 只发布 GHCR，并没有建立自己的 GitHub Release 发布链路。后续需要与用户确定一种方案：

1. 删除或隐藏站内版本检查；或
2. 增加 fork Release 发布后改为查询 `Mashiro0619/new-api`；或
3. 改成检查 GHCR/构建 revision，并为私有仓库设计不会向浏览器暴露 Token 的后端接口。

不要简单让浏览器携带 GitHub Token 请求私有仓库。

### 10.2 `latest` 的回滚能力有限

用户接受 `latest` 的简化更新方式，但同一个标签会移动。更新前仍应记录当前镜像 digest/revision并备份数据库。若未来需要一键回滚，可在服务器本地保留旧镜像 digest 或重新启用 SHA 标签流程。

### 10.3 强制协议前端交互测试仍可加强

现有测试覆盖默认值、序列化、非法类型、回显和透传冲突。仍可补充抽屉级 React Testing Library 测试，覆盖真实 Select 交互、敏感权限锁定、渠道类型切换和错误定位。这是测试完善项，不是当前已知阻断问题。

### 10.4 部署文档尚未完全同步 `latest` 偏好

根 `README.md` 和 `deploy/README.md` 仍以不可变 SHA 作为推荐日常流程，并且存在让既有实例直接使用 `deploy/update.sh` 的说明。这些内容尚未按当前生产文件布局和用户的 `latest` 偏好同步，不代表当前操作选择。后续应统一文档；在此之前，生产环境以第 4 节的双 `-f` 命令为准。

### 10.5 GHCR 制品验收尚未从本机完成

工作流已经配置双架构 manifest、SBOM、provenance 和 keyless cosign 签名，但本机无法读取私有 GHCR，因此没有对实际发布制品完成 manifest、attestation 和签名复核。首次推送新版本后应在有权限的环境中逐项验证。

### 10.6 首次新版生产验收

推送本地提交并生成新 `latest` 后，需要在生产环境验证：

- 首页布局与登录/注册状态
- 管理员和普通用户 Token 趋势
- “消耗余额”显示是否符合实际币种/额度设置
- 至少一个真实有效令牌的计费请求
- OpenAI/New API 渠道强制协议的真实上游请求
- GHCR 双架构 manifest、SBOM、provenance 和签名
- PostgreSQL、Redis 容器和 Hermes 未受影响

## 11. 接手规则

- 先完整阅读根 `AGENTS.md`；前端改动还要阅读 `web/AGENTS.md`。
- 保持 Router -> Controller -> Service -> Model 分层。
- `relaykit/` 必须能独立构建，不得导入根模块。
- JSON 使用项目封装，不直接调用 `encoding/json` 的 marshal/unmarshal。
- 数据库改动必须兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。
- 前端使用 Bun，不混用 npm、Yarn 或 pnpm。
- 当前仓库规则要求用户可见文案维护所有支持语言；除非用户再次明确改变项目约定，否则继续执行七语言同步。
- 不要删除许可证、作者归属或原项目链接。
- 不要使用 `git reset --hard`、`git checkout --` 清理用户改动。
- 不要自行执行 `git push`；只有用户明确要求时才推送。
- 不要接触生产凭据、Hermes、数据库卷或备份，除非用户明确授权并提供操作范围。
