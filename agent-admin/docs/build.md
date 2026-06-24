# Agent Admin 打包说明

本文档说明 `agent-admin` 的本地构建、Docker 镜像打包和 GitHub Actions 自动打包流程。

## 前置条件

- Node.js 20
- pnpm 9 或通过 Corepack 启用 pnpm
- Docker 24+
- 如需推送 GHCR 镜像，需要 GitHub Packages 写入权限

## 本地前端构建

```bash
cd agent-admin
corepack enable
pnpm install --frozen-lockfile
pnpm run build
```

构建产物输出到：

```text
agent-admin/dist
```

`pnpm run build` 会先执行 `vue-tsc -b` 类型检查，再执行 `vite build`。

## 本地 Docker 镜像打包

在仓库根目录执行：

```bash
docker build \
  -f agent-admin/Dockerfile \
  -t sub2api-agent-admin:latest \
  agent-admin
```

只构建 amd64 镜像：

```bash
docker buildx build \
  --platform linux/amd64 \
  -f agent-admin/Dockerfile \
  -t sub2api-agent-admin:latest \
  agent-admin \
  --load
```

如果需要推送到私有仓库：

```bash
docker tag sub2api-agent-admin:latest <registry>/<namespace>/sub2api-agent-admin:<tag>
docker push <registry>/<namespace>/sub2api-agent-admin:<tag>
```

## GitHub Actions 自动打包

Workflow 文件：

```text
.github/workflows/agent-admin-image.yml
```

触发方式：

- push 到 `main` 或 `master`，且改动包含 `agent-admin/**` 或 workflow 文件
- pull request 改动包含 `agent-admin/**` 或 workflow 文件
- 手动触发 `workflow_dispatch`

PR 行为：

- 只构建镜像
- 不登录 GHCR
- 不推送镜像

push / 手动触发行为：

- 构建 `linux/amd64` 镜像
- 推送到 GitHub Container Registry

默认镜像地址：

```text
ghcr.io/<owner>/sub2api-agent-admin
```

自动生成标签：

- `latest`：默认分支
- 分支名标签：例如 `main`
- commit sha 标签：例如 `sha-abcdef1`

## Dockerfile 结构

当前镜像使用两阶段构建：

1. `node:20-alpine` 安装依赖并执行 `pnpm build`
2. `nginx:1.27-alpine` 托管静态文件，并通过 Nginx 反向代理 `/api/`

运行时镜像只包含：

- `dist` 静态文件
- `agent-admin/nginx.conf`

## 校验命令

本地打包前建议执行：

```bash
cd agent-admin
pnpm run typecheck
pnpm run build
```

镜像构建后可检查：

```bash
docker image inspect sub2api-agent-admin:latest
```
