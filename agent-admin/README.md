# Agent Admin

Agent Admin is a standalone but related management platform for sub2api代理经销商. It lives outside the main `frontend` app so the reseller workflow can evolve independently while still talking to the same sub2api backend.

## Scope

- 独立代理经销商工作台
- 客户、邀请、返利、结算和套餐分发管理
- 对接当前后端的 `/api/v1/admin/affiliates/*` 能力
- 后续可扩展为独立域名、独立权限、独立部署流水线

## Development

```bash
cd agent-admin
pnpm install
pnpm dev
```

By default the dev server runs on `http://localhost:3100` and proxies `/api` to `http://localhost:8080`.

Override the backend with:

```bash
VITE_DEV_PROXY_TARGET=http://localhost:8080 pnpm dev
```

## Project Boundary

This project intentionally duplicates only the small API/types surface it needs from the main frontend. Avoid importing from `../frontend/src` directly, because that would make this app depend on the main admin UI build structure.
