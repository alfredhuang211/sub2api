# Agent Admin 启动运行说明

本文档说明如何以开发模式、Docker 单镜像和 Docker Compose 方式运行 `agent-admin`。

## 运行前提

`agent-admin` 是独立管理平台，默认以一个容器镜像运行：

- Nginx 托管前端静态页面。
- `agent-admin-api` 在同容器内监听 `127.0.0.1:3101`。
- Nginx 将 `/api/` 代理到容器内 `agent-admin-api`。
- `agent-admin-api` 通过 `SUB2API_BASE_URL` 对接同服务器或同网络内的 `sub2api`。

登录账号密码与原系统一致。认证由 `agent-admin-api` 代理登录到 `sub2api`，JWT 按 `sub2api` 规则本地验签。

## 开发模式运行

```bash
cd agent-admin
corepack enable
pnpm install --frozen-lockfile
pnpm dev
```

默认访问地址：

```text
http://localhost:3100
```

默认开发代理目标：

```text
http://localhost:3101
```

可通过环境变量调整：

```bash
VITE_DEV_PORT=3100 \
VITE_DEV_PROXY_TARGET=http://localhost:3101 \
pnpm dev
```

开发模式下可单独启动 API：

```bash
cd agent-admin/api
go run ./cmd/server
```

## Docker 单镜像运行

先构建镜像：

```bash
docker build \
  -f agent-admin/Dockerfile \
  -t sub2api-agent-admin:latest \
  agent-admin
```

确保 `sub2api` 和 agent-admin 在同一个 Docker 网络中。示例网络名：

```bash
docker network create sub2api
```

启动：

```bash
docker run -d \
  --name sub2api-agent-admin \
  --restart unless-stopped \
  --network sub2api \
  -p 3100:80 \
  -e SUB2API_BASE_URL=http://sub2api:8080 \
  -e DATABASE_URL="${DATABASE_URL}" \
  -e JWT_SECRET="${JWT_SECRET}" \
  -e JWT_SIGNING_METHOD=HS256 \
  sub2api-agent-admin:latest
```

访问：

```text
http://localhost:3100
```

## Docker Compose 运行

示例文件：

```text
agent-admin/docker-compose.example.yml
```

复制为本地 compose 文件后启动：

```bash
cd agent-admin
docker compose -f docker-compose.example.yml up -d
```

查看状态：

```bash
docker compose -f docker-compose.example.yml ps
```

查看日志：

```bash
docker compose -f docker-compose.example.yml logs -f agent-admin
```

停止：

```bash
docker compose -f docker-compose.example.yml down
```

## 使用 GHCR 镜像运行

GitHub Actions 推送后的镜像地址格式：

```text
ghcr.io/<owner>/sub2api-agent-admin:<tag>
```

拉取示例：

```bash
docker pull ghcr.io/<owner>/sub2api-agent-admin:latest
```

运行示例：

```bash
docker run -d \
  --name sub2api-agent-admin \
  --restart unless-stopped \
  --network sub2api \
  -p 3100:80 \
  -e SUB2API_BASE_URL=http://sub2api:8080 \
  -e DATABASE_URL="${DATABASE_URL}" \
  -e JWT_SECRET="${JWT_SECRET}" \
  -e JWT_SIGNING_METHOD=HS256 \
  ghcr.io/<owner>/sub2api-agent-admin:latest
```

如果 GHCR package 是私有的，需要先登录：

```bash
echo "<github_token>" | docker login ghcr.io -u <github_username> --password-stdin
```

## 后端连接配置

容器镜像内置 Nginx 配置：

```nginx
location /api/ {
  proxy_pass http://127.0.0.1:3101/api/;
}
```

`agent-admin-api` 再通过环境变量连接 `sub2api`：

```text
SUB2API_BASE_URL=http://sub2api:8080
```

如果 `sub2api` 容器名或端口不同，调整 `SUB2API_BASE_URL` 即可，通常不需要修改 `nginx.conf`。

如果使用独立域名部署，需要确保浏览器能访问：

```text
https://<agent-admin-domain>/api/v1/...
```

## 更新版本

拉取新镜像：

```bash
docker pull ghcr.io/<owner>/sub2api-agent-admin:latest
```

重启容器：

```bash
docker stop sub2api-agent-admin
docker rm sub2api-agent-admin
docker run -d \
  --name sub2api-agent-admin \
  --restart unless-stopped \
  --network sub2api \
  -p 3100:80 \
  ghcr.io/<owner>/sub2api-agent-admin:latest
```

使用 Compose 时：

```bash
docker compose -f docker-compose.example.yml pull
docker compose -f docker-compose.example.yml up -d
```

## 常见检查

检查容器：

```bash
docker ps --filter name=sub2api-agent-admin
```

检查日志：

```bash
docker logs -f sub2api-agent-admin
```

检查静态页面：

```bash
curl -I http://localhost:3100
```

检查 API 代理：

```bash
curl -I http://localhost:3100/api/v1/health
```

如果页面可以打开但接口失败，优先检查：

- 后端服务是否运行
- agent-admin 和 sub2api 是否在同一个 Docker 网络
- `SUB2API_BASE_URL` 是否能从 agent-admin 容器内访问
- 外层网关是否正确转发 `/api/`
