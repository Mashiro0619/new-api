# 自有镜像部署与更新

仓库根目录的 `docker-compose.override.yml` 会被常规 `docker compose` 命令自动加载，并在原有 `docker-compose.yml` 之上只覆盖 `services.new-api.image`。因此，`docker compose up -d` 和 `docker compose pull new-api` 默认都会使用自有 GHCR 镜像。PostgreSQL、Redis、端口、环境变量、网络和数据卷仍由原 Compose 文件管理，不会被覆盖文件改写。

更新脚本需要 Docker Compose v2、`jq` 和 `sed`；服务器应先安装这些命令并确认 Docker daemon 可访问。

默认镜像为 `ghcr.io/mashiro0619/new-api:latest`。可通过 `FORK_IMAGE` 更换镜像仓库，通过 `FORK_IMAGE_TAG` 更换标签。生产环境应使用发布工作流生成的不可变 `sha-<完整提交哈希>` 标签，不建议直接部署会移动的 `latest`。

## GitHub 与 GHCR 首次配置

1. 在 fork 的 GitHub `Actions` 页面启用 `publish-ghcr.yml`。首次启用后先不要创建 Git 标签。
2. 手动运行一次 `publish-ghcr.yml`，或推送 `main` 触发它，确认 `ghcr.io/mashiro0619/new-api` 已生成 `linux/amd64` 和 `linux/arm64` 镜像。
3. GHCR Package 保持私有。为服务器准备只具备 `read:packages` 权限的 GitHub Personal Access Token，并在服务器上执行：

   ```sh
   docker login ghcr.io -u Mashiro0619
   ```

   在交互式密码提示中输入 Token。不要把 Token 直接写在命令行、脚本或仓库文件中。登录凭据由服务器上的 Docker 凭据存储管理。
4. 用以下命令确认 `latest`、完整 SHA 和版本标签存在，并检查多架构清单。把 `<版本标签>` 替换为实际创建的 `v*` 标签：

   ```sh
   docker buildx imagetools inspect ghcr.io/mashiro0619/new-api:latest
   docker buildx imagetools inspect ghcr.io/mashiro0619/new-api:sha-<完整提交哈希>
   docker buildx imagetools inspect ghcr.io/mashiro0619/new-api:<版本标签>
   ```

   每个清单都应列出 `linux/amd64` 和 `linux/arm64`。从清单的 digest 复制 `<DIGEST>`，再确认 BuildKit 的 provenance、SBOM 不为空，并验证无密钥签名（需要已安装 `cosign`）：

   ```sh
   IMAGE=ghcr.io/mashiro0619/new-api
   docker buildx imagetools inspect --format '{{json .SLSA}}' "$IMAGE@<DIGEST>"
   docker buildx imagetools inspect --format '{{json .SBOM}}' "$IMAGE@<DIGEST>"
   cosign verify \
     --certificate-oidc-issuer https://token.actions.githubusercontent.com \
     --certificate-identity-regexp 'https://github.com/Mashiro0619/new-api/.github/workflows/publish-ghcr.yml@.*' \
     "$IMAGE@<DIGEST>"
   ```

GHCR 发布使用工作流自带的 `GITHUB_TOKEN`；服务器拉取使用单独的只读 Token。不要把 GitHub Token、服务器密码、`.env`、数据库转储、签名私钥或其他凭据提交到 Git。

## 更新前备份

任何首次接管、日常更新或回滚都应先备份数据库，并记录当前应用、PostgreSQL 和 Redis 的容器 ID。更新脚本不会备份或恢复数据库，也不会自动回退数据库迁移。

以下 PostgreSQL 示例沿用仓库 Compose 示例中的用户和库名；生产环境如已修改配置，必须替换为实际值，并把备份文件保存到受控目录：

```sh
BACKUP_DIR="$HOME/new-api-backups"
install -d -m 700 "$BACKUP_DIR"
umask 077
docker compose exec -T postgres pg_dump -U root -d new-api > "$BACKUP_DIR/new-api-$(date +%Y%m%d-%H%M%S).sql"
docker compose ps --quiet new-api postgres redis
docker inspect --format '{{.Name}} {{.Id}} {{.Config.Image}} {{.Image}}' \
  "$(docker compose ps --quiet new-api)" \
  "$(docker compose ps --quiet postgres)" \
  "$(docker compose ps --quiet redis)"
docker compose config --volumes
docker inspect --format '{{.Name}} {{range .Mounts}}{{.Type}}={{.Source}}->{{.Destination}};{{end}}' \
  "$(docker compose ps --quiet new-api)" \
  "$(docker compose ps --quiet postgres)" \
  "$(docker compose ps --quiet redis)"
```

使用 MySQL 或 SQLite 时，应使用对应数据库的可靠备份流程。不要直接复制仍在写入的 SQLite 文件。开始更新前要实际验证备份可读、大小合理且具备恢复条件。

## 首次接管现有部署

脚本要求 Compose 项目中已经存在 `new-api` 容器，适用于把当前服务器从上游镜像切换到自有镜像。它会从脚本位置自动找到仓库根目录的 `docker-compose.yml`，因此可以从任意工作目录调用。

先查看合并后的配置。输出中 `new-api` 的镜像应改变，PostgreSQL、Redis、数据卷、网络、端口和环境变量应与原配置一致：

```sh
FORK_IMAGE=ghcr.io/mashiro0619/new-api \
FORK_IMAGE_TAG=sha-<完整提交哈希> \
docker compose \
  -f docker-compose.yml \
  -f docker-compose.override.yml \
  config
```

备份完成后，用明确的 SHA 标签接管：

```sh
sh deploy/update.sh sha-<完整提交哈希>
```

脚本会校验 Docker、Compose 和合并配置，记录旧容器的镜像引用、镜像 ID 及 OCI revision，只拉取并以 `--no-deps` 重建 `new-api`，等待原 Compose 中的健康检查，再从新容器内部检查 `/api/status`。它不会重启 PostgreSQL 或 Redis。

接管后再次比较 PostgreSQL 和 Redis 的容器 ID，确认二者未变化，并手工验证登录、管理后台和主要 API 请求。

## 日常更新与回滚

日常更新同样先备份，再传入发布工作流生成的完整 SHA 标签：

```sh
sh deploy/update.sh sha-<新版本完整提交哈希>
```

如需主动回滚到仍在 GHCR 中的旧版本，传入上一个不可变标签：

```sh
sh deploy/update.sh sha-<旧版本完整提交哈希>
```

更新失败时，脚本会打印一条可直接执行的精确回滚命令。该命令把更新前的本地镜像 ID 标记为临时 `rollback-<时间>` 标签，然后只用 `--no-deps` 重建 `new-api`。脚本不会自动执行回滚，以便先保留现场和检查日志：

```sh
docker compose \
  -f docker-compose.yml \
  -f docker-compose.override.yml \
  logs --no-color --tail=200 new-api
```

镜像回滚不会撤销数据库结构变化。如果新版本已经执行不兼容迁移，应停止应用写入，按已审核的恢复流程使用更新前备份恢复数据库，再启动旧镜像；不要让脚本猜测或自动执行数据库回滚。

首次上线验收时应在维护窗口内用上一版本的 SHA 标签完成一次回滚演练，并再次切回目标版本；每次切换都要重复健康检查、`/api/status`、登录验证，以及 PostgreSQL、Redis 容器 ID 和卷挂载状态核对：

```sh
sh deploy/update.sh sha-<上一版本完整提交哈希>
sh deploy/update.sh sha-<当前版本完整提交哈希>
```

## 同步官方上游

保持 fork 为 `origin`，官方仓库为只读 `upstream`。如果尚未配置，只需执行一次：

```sh
git remote add upstream https://github.com/QuantumNous/new-api.git
git remote set-url --push upstream DISABLED
```

日常同步在 `main` 上合并官方历史，不对已经推送的共享分支做 rebase 或强制推送：

```sh
git fetch upstream
git switch main
git merge --no-edit upstream/main
# 运行项目要求的后端、relaykit 和前端检查
git push origin main
```

同步和二次开发应继续遵守仓库中的 `LICENSE`、`NOTICE` 和第三方许可证要求，并在分发修改版本时保留依法要求的版权、许可、原项目链接和变更说明。
