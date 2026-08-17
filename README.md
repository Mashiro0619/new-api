<div align="center">

![New API](./web/public/logo.png)

# New API

新一代大模型网关与 AI 资产管理系统

[官方文档](https://docs.newapi.pro/zh/docs) · [自有镜像部署](./deploy/README.md) · [许可证](./LICENSE)

</div>

## 项目说明

本仓库是 [Mashiro0619/new-api](https://github.com/Mashiro0619/new-api) 的个人二次开发分支，基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 持续同步和开发。

New API 用于统一接入和管理多个 AI 服务，提供 API 转发、用户管理、额度与计费、渠道管理及管理后台。详细功能、支持的模型和接口以[官方文档](https://docs.newapi.pro/zh/docs)为准。

> [!IMPORTANT]
> 本项目仅用于合法授权的 AI API 网关、组织内部鉴权、用量统计、成本核算和私有部署。使用者必须合法取得上游账号、API Key 和模型服务权限，并遵守上游服务条款及适用法律法规。

## 文档

| 内容 | 链接 |
| --- | --- |
| 自有 GHCR 镜像、首次接管、更新与回滚 | [deploy/README.md](./deploy/README.md) |
| 官方安装与配置 | [docs.newapi.pro/zh/docs/installation](https://docs.newapi.pro/zh/docs/installation) |
| API 文档 | [docs.newapi.pro/zh/docs/api](https://docs.newapi.pro/zh/docs/api) |
| 环境变量 | [环境变量文档](https://docs.newapi.pro/zh/docs/installation/config-maintenance/environment-variables) |
| 用户鉴权与登录会话 | [docs/authentication.md](./docs/authentication.md) |

## 快速开始

运行环境需要 Docker 和 Docker Compose v2。本仓库通过根目录的 `docker-compose.override.yml` 将应用镜像替换为：

```text
ghcr.io/mashiro0619/new-api:<标签>
```

如果 GHCR Package 保持私有，先使用具备 `read:packages` 权限的 GitHub Token 登录。命令会交互式要求输入 Token，不要把 Token 写入仓库或命令历史：

```sh
docker login ghcr.io -u Mashiro0619
```

然后克隆仓库并使用不可变 SHA 标签启动：

```sh
git clone https://github.com/Mashiro0619/new-api.git
cd new-api

export FORK_IMAGE_TAG=sha-<完整提交哈希>
docker compose pull new-api
docker compose up -d
docker compose ps
curl -fsS http://127.0.0.1:3000/api/status
```

`<完整提交哈希>` 是 Git 提交的 40 位标识，可通过 `git rev-parse HEAD` 查看。发布工作流会生成对应的 `sha-<完整提交哈希>` 镜像标签。生产环境建议固定使用该标签，不直接依赖会移动的 `latest`。

已有官方 New API 实例需要保留数据库和 Redis 时，不要直接重新创建整套服务。请按照[自有镜像部署与更新](./deploy/README.md)先备份数据库，再使用 `deploy/update.sh` 仅替换 `new-api` 服务。

## 本地开发

后端测试：

```sh
make test
```

前端开发与检查：

```sh
cd web
bun install
bun run typecheck
bun run test
bun run build
```

涉及 `relaykit/` 的改动还需要验证其独立模块：

```sh
cd relaykit
GOWORK=off go build ./...
GOWORK=off go test ./...
```

## 同步上游

`origin` 指向个人 fork，`upstream` 指向官方仓库且仅用于拉取：

```sh
git fetch upstream
git switch main
git merge --no-edit upstream/main

# 完成后端、relaykit 和前端验证后
git push origin main
```

不对已经推送的 `main` 历史执行强制推送。合并上游后，推送 `origin/main` 会触发自有 GHCR 镜像发布工作流。

## 许可证与归属

Copyright (c) QuantumNous and contributors.

本项目基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 修改；本分支由 Mashiro0619 修改（修改日期：2026 年，具体变更见 Git 提交历史）。本项目采用 [GNU Affero 通用公共许可证 v3.0](./LICENSE)（AGPLv3）授权，并受 [NOTICE](./NOTICE) 中依据 AGPLv3 第 7 条列出的附加条款约束。修改版本不得歪曲软件来源。

修改版本必须在适当的法律声明，以及界面中显著的关于、法律、页脚或归属位置保留以下作者归属声明：

```text
Frontend design and development by New API contributors.
```

提供用户界面的修改版本还必须保留指向原始项目的可见链接：

<https://github.com/QuantumNous/new-api>

README 中的声明不能替代用户界面内要求保留的可见归属和原项目链接。

本项目基于 [One API](https://github.com/songquanpeng/one-api)（MIT License）二次开发。完整版权、归属和第三方许可信息见 [LICENSE](./LICENSE)、[NOTICE](./NOTICE) 和 [THIRD-PARTY-LICENSES.md](./THIRD-PARTY-LICENSES.md)。
