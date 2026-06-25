# Agent Admin

Agent Admin is a standalone but related management platform for sub2api 代理经销商. It lives outside the main `frontend` app so the reseller workflow can evolve independently while sharing the same account system and database.

## Scope

- 独立代理经销商工作台
- 客户、邀请、返利、结算和套餐分发管理
- 独立 `agent-admin-api` 承接代理业务接口
- 单镜像部署：Nginx + 前端静态资源 + `agent-admin-api`
- 后续可扩展为独立域名和独立发布流水线

## Development

```bash
cd agent-admin
pnpm install
pnpm dev
```

By default the dev server runs on `http://localhost:3100` and proxies `/api` to `http://localhost:3101`.

Override the backend with:

```bash
VITE_DEV_PROXY_TARGET=http://localhost:3101 pnpm dev
```

Start the API skeleton separately during development:

```bash
cd agent-admin/api
go run ./cmd/server
```

## Project Boundary

This project intentionally duplicates only the small API/types surface it needs from the main frontend. Avoid importing from `../frontend/src` directly, because that would make this app depend on the main admin UI build structure.

## Docs

- [打包说明](docs/build.md)
- [启动运行说明](docs/run.md)
