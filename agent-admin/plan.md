# Agent Admin 技术设计计划

## 1. 设计目标

`agent-admin` 调整为独立全栈项目：

- 独立前端：`agent-admin-web`。
- 独立后端：`agent-admin-api`。
- 独立迁移：`agent-admin` 自己维护 `agent_*` 表。
- 与 `sub2api` 共用同一个数据库。
- 认证继续复用 `sub2api`。
- 不修改 `sub2api` 原项目代码。
- 与 `sub2api` 部署在同一服务器或同一内网。
- `sub2api` 和 `agent-admin` 后续分别独立更新。

## 2. 总体架构

```text
Browser
  |
  | agent-admin frontend
  v
agent-admin-web
  |
  | same container /api/*
  v
agent-admin-api
  |
  | read/write agent-admin owned tables
  | read sub2api existing tables
  v
shared database
  ^
  |
sub2api backend
```

`sub2api` 继续负责：

- 用户登录。
- JWT 签发。
- 原系统管理员身份。
- 原推荐系统。
- 原订单、套餐、用量、支付、退款数据。

`agent-admin-api` 负责：

- JWT 校验和权限判断。
- 代理业务 API。
- 代理业务迁移。
- 代理业务定时任务。
- 代理业务审计。
- 对原系统表的只读查询。

部署上默认采用单镜像：一个 `sub2api-agent-admin` 容器同时运行 `agent-admin-web` 静态服务和 `agent-admin-api`。Nginx 对外监听 80 端口，`/api/` 反向代理到容器内 `127.0.0.1:3101`。

## 3. 技术选型

### 前端

继续使用当前实现：

- Vue 3
- TypeScript
- Vite
- Axios
- 原生 CSS

目录：

```text
agent-admin/
  src/
    api/
    components/
    views/
    main.ts
    App.vue
    style.css
```

### 后端

建议优先使用 Go：

- Gin 或 Chi 作为 HTTP 框架。
- pgx 或 database/sql + lib/pq 访问 PostgreSQL。
- golang-migrate 或自研轻量 SQL migration runner。
- robfig/cron 或内部 ticker 执行定时任务。

选择 Go 的理由：

- 当前 `sub2api` 后端是 Go，便于复用 JWT、数据库、部署和运维经验。
- 适合长驻服务和定时任务。
- 容器镜像较小，部署简单。

可选 Node/Fastify/NestJS，但初期不推荐引入不同后端技术栈。

### 数据库

- 使用与 `sub2api` 相同的 PostgreSQL 数据库。
- `agent-admin` 自有表以 `agent_` 前缀命名。
- 自有迁移记录表使用 `agent_admin_schema_migrations`，避免与 `sub2api.schema_migrations` 冲突。
- 金额统一使用最小货币单位，例如分。
- 比例使用 `rate_bps`，例如 `2000 = 20.00%`。

## 4. 认证设计

### 登录流程

```text
agent-admin-web
  -> 调用 agent-admin-api 登录代理接口
  -> agent-admin-api 转发到 sub2api 登录接口
  -> 获得 sub2api JWT
  -> 存储 token
  -> 请求 agent-admin-api 时携带 Authorization: Bearer <token>
```

### agent-admin-api 认证策略

已确认方案：本地 JWT 验签。

`agent-admin-api` 需要配置：

- JWT secret 或 public key。
- signing method。
- issuer / audience 策略。
- token 过期校验。
- token version / 改密失效校验。

验签后：

1. 从 token 读取 `user_id`。
2. 查询共享数据库 `users`。
3. 确认用户状态 active。
4. 如果访问管理员 API，要求 `users.role = admin`。
5. 如果访问代理 API，要求存在 `agent_profiles.status = active`。

JWT 细节必须与 `sub2api` 保持一致，包括 secret、公钥、签名算法、claims、过期规则和改密失效机制。

远程校验仅作为应急 fallback。

```text
agent-admin-api
  -> 调用 sub2api /api/v1/user/profile 或等价接口
  -> 获得当前用户身份
```

远程校验可作为后续安全增强或 JWT 规则不易同步时的 fallback。

## 5. API 设计

建议 `agent-admin-api` 对外统一暴露：

```text
/api/v1/admin/agents/*
/api/v1/admin/agent-*
/api/v1/agent/*
```

前端仍可使用 `/api/v1` 作为 base URL，由 agent-admin-web 的 Nginx 代理到 `agent-admin-api`。

### 管理员 API

```text
GET    /api/v1/admin/agents/summary
GET    /api/v1/admin/agents
POST   /api/v1/admin/agents
GET    /api/v1/admin/agents/:id
PUT    /api/v1/admin/agents/:id
POST   /api/v1/admin/agents/:id/disable
POST   /api/v1/admin/agents/:id/restore
GET    /api/v1/admin/agents/:id/customers
GET    /api/v1/admin/agents/:id/children
GET    /api/v1/admin/agents/:id/summary
PUT    /api/v1/admin/agents/:id/commission-rate
POST   /api/v1/admin/agents/:id/force-adjust
GET    /api/v1/admin/admin-users
POST   /api/v1/admin/admin-users
GET    /api/v1/admin/admin-users/candidates
POST   /api/v1/admin/admin-users/:id/revoke
POST   /api/v1/admin/agent-customer-relations
GET    /api/v1/admin/agent-customer-relations/changes
GET    /api/v1/admin/agent-commissions
GET    /api/v1/admin/agent-settlements
POST   /api/v1/admin/agent-settlements/:id/register-payment
POST   /api/v1/admin/agent-settlements/:id/adjust
GET    /api/v1/admin/agent-audit-logs
```

### 代理商 API

```text
GET    /api/v1/agent/profile
GET    /api/v1/agent/dashboard
GET    /api/v1/agent/customers
GET    /api/v1/agent/developable-users
GET    /api/v1/agent/invites
GET    /api/v1/agent/children
POST   /api/v1/agent/children
PUT    /api/v1/agent/children/:id/commission-rate
GET    /api/v1/agent/upline
GET    /api/v1/agent/commissions
GET    /api/v1/agent/settlements
GET    /api/v1/agent/orders
```

### 响应格式

为降低前端改造成本，沿用 `sub2api` 风格：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

分页：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "page_size": 20,
  "pages": 1
}
```

## 6. 数据结构设计

### agent_admin_schema_migrations

agent-admin 自有迁移记录表。

```text
filename              text primary key
checksum              text not null
applied_at            timestamptz not null default now()
```

### agent_admin_users

Agent Admin 授权管理员表。原系统 `users.role = admin` 的用户是基础管理员，不需要写入该表；该表只保存基础管理员额外授权的后台管理员。

```text
id                    bigint primary key
user_id               bigint not null unique       # references users(id)
status                varchar(20) not null          # active, disabled
created_by            bigint null                   # users(id)
revoked_by            bigint null                   # users(id)
revoked_at            timestamptz null
created_at            timestamptz not null
updated_at            timestamptz not null
```

### agent_profiles

代理身份表。

```text
id                    bigint primary key
user_id               bigint not null unique       # references users(id)
level                 smallint not null             # 1, 2, 3
parent_agent_id       bigint null                   # references agent_profiles(id)
status                varchar(20) not null          # active, disabled
contact_info          jsonb not null default '{}'
settlement_info       jsonb not null default '{}'
created_by            bigint null                   # users(id)
disabled_at           timestamptz null
created_at            timestamptz not null
updated_at            timestamptz not null
```

### agent_commission_rates

分成比例历史表。

```text
id                    bigint primary key
agent_id              bigint not null
rate_bps              int not null
set_by_user_id        bigint null
set_by_agent_id       bigint null
effective_at          timestamptz not null
expired_at            timestamptz null
created_at            timestamptz not null
updated_at            timestamptz not null
```

### agent_customer_relations

客户归属表。

```text
id                    bigint primary key
agent_id              bigint not null
customer_user_id      bigint not null
source                varchar(20) not null          # referral, manual
source_referral_code  varchar(32) null
effective_at          timestamptz not null
expired_at            timestamptz null
status                varchar(20) not null          # active, expired
created_by            bigint null
created_at            timestamptz not null
updated_at            timestamptz not null
```

约束：

- 同一客户同一时间只能有一个 active relation。
- 手动归属从下一完整套餐周期生效。

### agent_customer_relation_changes

客户归属变更记录。

```text
id                    bigint primary key
customer_user_id      bigint not null
from_agent_id         bigint null
to_agent_id           bigint null
reason                text not null
operator_user_id      bigint null
effective_at          timestamptz not null
created_at            timestamptz not null
```

### agent_commission_periods

周期佣金记录。

```text
id                    bigint primary key
customer_user_id      bigint not null
agent_id              bigint not null
order_id              bigint null
subscription_id       bigint null
period_start_at       timestamptz not null
period_end_at         timestamptz not null
order_paid_amount     bigint not null
confirmed_revenue     bigint not null
rate_bps              int not null
commission_amount     bigint not null
reverse_amount        bigint not null default 0
reverse_reason_type   varchar(64) null
status                varchar(20) not null
generated_at          timestamptz not null
frozen_until          timestamptz null
settlement_id         bigint null
created_at            timestamptz not null
updated_at            timestamptz not null
```

幂等键建议：

```text
customer_user_id + order_id + period_start_at + period_end_at + agent_id
```

### agent_settlements

结算记录。

```text
id                    bigint primary key
agent_id              bigint not null
period_month          date not null
amount                bigint not null
reverse_amount        bigint not null
net_amount            bigint not null
status                varchar(20) not null
min_amount_met        boolean not null
frozen_until          timestamptz null
paid_at               timestamptz null
paid_by_user_id       bigint null
payment_reference     varchar(128) null
created_at            timestamptz not null
updated_at            timestamptz not null
```

### agent_settlement_payments

管理员线下结算登记明细。当前一条结算只允许登记一次，后续如需拆分付款可调整唯一约束和业务状态。

```text
id                    bigint primary key
settlement_id         bigint not null unique
agent_id              bigint not null
amount                bigint not null
payment_method        varchar(32) null          # bank_transfer, alipay, wechat_pay, cash, other
payment_reference     varchar(128) null
paid_at               timestamptz null
remark                text null
created_by            bigint null
created_at            timestamptz not null
updated_at            timestamptz not null
```

### agent_audit_logs

审计日志。

```text
id                    bigint primary key
operator_user_id      bigint null
operator_role         varchar(20) not null          # admin, agent
action                varchar(64) not null
target_type           varchar(64) not null
target_id             bigint not null
before_data           jsonb null
after_data            jsonb null
reason                text null
created_at            timestamptz not null
```

## 7. 原 sub2api 表读取映射

需要读取但不改写：

```text
users
user_affiliates
payment_orders
user_subscriptions
subscription_plans
usage_logs / usage aggregation tables
payment_audit_logs
```

具体字段映射仍需按当前数据库确认。

读取原则：

- `users`：认证主体、管理员角色、代理关联用户信息。
- `user_affiliates`：自然推荐关系、推荐码、推荐上下级。
- `payment_orders`：订单实付金额、支付状态、退款状态。
- `user_subscriptions`：套餐周期开始和结束。
- `subscription_plans`：套餐名称。
- `usage_logs`：客户使用数据和报表。

## 8. 核心业务流程

### 服务启动

1. 读取配置。
2. 连接共享数据库。
3. 获取 agent-admin migration lock。
4. 执行 `agent-admin-api/migrations/*.sql` 中未执行的迁移。
5. 启动 HTTP 服务。
6. 启动定时任务。

当前代码实现：

- `agent-admin-api` 启动时执行自有表迁移。
- `SCHEDULER_ENABLED=true` 时启动内置 scheduler。
- scheduler 启动后立即执行一轮，之后每天执行一次。
- 执行顺序为：同步自然推荐客户归属、生成周期佣金、生成自然月结算。

### 指定代理

1. 管理员通过 `sub2api` JWT 登录 agent-admin。
2. `agent-admin-api` 校验管理员身份。
3. 管理员选择原系统用户。
4. 系统校验用户存在、状态正常、未成为代理。
5. 系统创建 `agent_profiles`。
6. 系统创建默认 `agent_commission_rates`。
7. 系统记录审计日志。

### 管理员强制调整

管理员可处理代理商问题和历史配置错误：

1. 管理员选择代理、客户或结算记录。
2. 系统要求填写调整原因。
3. 系统读取调整前快照。
4. 系统执行强制调整，可覆盖普通代理商权限限制。
5. 系统记录调整后快照、原因、生效时间和操作人。
6. 关键调整包括代理等级、上级代理、比例、客户归属、生效时间、结算金额、结算状态和账务冲正。

限制：

- 管理员可以覆盖业务权限限制，但不能绕过数据库约束和审计。
- 调整后若造成下级比例不合法，系统应优先阻止；确需特殊处理时应通过账务调整而不是破坏比例链路。

### 创建下级代理

1. 代理选择直接归属于自己的可发展用户。
2. 系统读取 `agent_customer_relations` 校验客户归属；若自然推荐关系尚未同步为归属记录，则兼容读取 `user_affiliates` 校验推荐关系。
3. 系统校验层级不超过 3 级。
4. 系统校验分成比例低于上级比例。
5. 系统创建代理身份和比例。
6. 系统记录审计日志。

### 手动归属客户

1. 管理员选择客户和目标代理。
2. 管理员填写调整原因。
3. 系统计算下一完整套餐周期的 `effective_at`。
4. 系统过期该客户旧的 scheduled relation。
5. 系统创建新 scheduled relation。
6. 系统记录 relation change 和审计日志。
7. 调度任务在 `effective_at <= now()` 后切换为 active，并过期旧 active relation。

### 分成生成

1. 定时任务扫描已到期套餐周期。
2. 读取订单实付金额作为周期确认收入。
3. 读取周期生效的客户归属。
4. 读取周期生效的比例快照。
5. 按差额分成生成每级记录。
6. 设置冻结期 5 天。
7. 保证幂等。

P0 SQL 策略：

- 读取 `payment_orders`、`agent_customer_relations`、`agent_profiles`、`agent_commission_rates`；`user_subscriptions` 仅作为可选关联记录展示，不作为订单周期历史来源。
- 仅处理订阅订单：`payment_orders.order_type = 'subscription'`。
- 仅处理已完成订单：`payment_orders.status = 'COMPLETED'`。
- 周期开始：`payment_orders.completed_at`。
- 周期结束：`payment_orders.completed_at + payment_orders.subscription_days`。
- 周期确认收入：`ROUND(GREATEST(pay_amount - refund_amount, 0) * 100)`。
- 周期归属：选择 `effective_at <= period_start_at` 的 active 归属。
- 代理链路：从直属代理向上递归到 1 级代理。
- 差额分成：当前代理比例减去直接下级比例，负数按 0。
- 幂等约束：`agent_commission_periods(order_id, agent_id)` 唯一。

当前限制：

- 已结算后的退款自动冲正入口暂未完成。
- 支付争议自动同步入口暂未完成。
- 由于原 `user_subscriptions` 不是周期历史表，多次续费、提前续费或特殊补单场景如需完全精确，需要 agent-admin 后续增加订单周期快照表或结算快照。

### 月底结算

1. 聚合自然月可结算佣金。
2. 扣除负数冲正。
3. 检查是否达到 100 元。
4. 生成结算记录。
5. 管理员线下付款后登记结算。

P0 SQL 策略：

- 按 `period_end_at` 所属自然月汇总 `agent_commission_periods`。
- 当前自然月未结束时不生成该月结算。
- `net_amount = amount - reverse_amount`。
- `net_amount >= 10000` 且冻结期结束后状态为 `payable`，否则为 `pending`。
- 已标记 `paid` 的结算记录不会被后续汇总覆盖状态。

线下结算登记：

- 管理员对 `payable` 状态结算调用登记接口。
- 请求中 `amount` 必填并以分存储；支付方式、支付时间、流水号和备注可选。
- 登记成功后写入 `agent_settlement_payments`，并更新 `agent_settlements.status = paid`、`paid_at`、`paid_by_user_id` 和可选 `payment_reference`。
- 代理商查询自己的结算列表时可看到付款金额、支付方式、支付时间、流水号、备注和登记人。
- 登记操作写入 `agent_audit_logs`，用于管理员追溯。

## 9. 配置设计

`agent-admin-api` 配置项：

```text
HTTP_ADDR=:3101
DATABASE_URL=postgres://...
JWT_SECRET=...
JWT_SIGNING_METHOD=HS256
JWT_ISSUER=
JWT_AUDIENCE=
SUB2API_BASE_URL=http://sub2api-backend:8080
AUTH_MODE=local_jwt | remote_sub2api
LOGIN_PROXY_ENABLED=true
MIGRATION_ENABLED=true
SCHEDULER_ENABLED=true
COMMISSION_FREEZE_DAYS=5
MIN_SETTLEMENT_AMOUNT=10000
SETTLEMENT_TIMEZONE=Asia/Shanghai
```

`agent-admin-web` 运行时配置：

```text
API_PROXY_TARGET=http://127.0.0.1:3101
```

前端统一请求 `agent-admin-api`。登录由 `agent-admin-api` 代理到 `sub2api`，因此后续使用独立域名时前端仍只需要一个 API base。

如果后续要改成外层 Nginx 直接分流，可以再增加：

```text
/api/agent-admin/* -> agent-admin-api
/api/sub2api-auth/* -> sub2api
```

## 10. 部署方案

### 推荐 Compose 拓扑

```yaml
services:
  sub2api:
    image: sub2api:latest
    networks:
      - app

  agent-admin:
    image: sub2api-agent-admin:latest
    ports:
      - "3100:80"
    environment:
      DATABASE_URL: ${DATABASE_URL}
      JWT_SECRET: ${JWT_SECRET}
      JWT_SIGNING_METHOD: HS256
      SUB2API_BASE_URL: http://sub2api:8080
      AUTH_MODE: local_jwt
      MIGRATION_ENABLED: "true"
      SCHEDULER_ENABLED: "true"
    depends_on:
      - sub2api
    networks:
      - app

networks:
  app:
```

### 镜像

默认使用一个镜像：

- `sub2api-agent-admin`

镜像内包含：

- Nginx：托管 Vue/Vite 前端静态资源。
- `agent-admin-api`：监听容器内 `127.0.0.1:3101`。
- Nginx `/api/` 反向代理到 `agent-admin-api`。

后续如果代理业务规模变大，可以再拆分为 web/api 两个镜像，但当前以简化部署为优先。

## 11. 编译打包流程

### 前端

```bash
cd agent-admin
pnpm install --frozen-lockfile
pnpm build
```

### 后端

建议目录：

```text
agent-admin/
  api/
    cmd/server/
    internal/
    migrations/
    go.mod
```

构建：

```bash
cd agent-admin/api
go test ./...
go build -o bin/agent-admin-api ./cmd/server
```

### Docker

单镜像：

```bash
docker build -f agent-admin/Dockerfile -t sub2api-agent-admin:latest agent-admin
```

## 12. GitHub Actions

现有：

- `Agent Admin Image` 构建单个 `sub2api-agent-admin` 镜像。

该镜像包含前端静态资源和 `agent-admin-api` 二进制。

后续可新增：

- `Agent Admin CI` 运行前端 typecheck、后端测试、迁移校验。

建议触发：

```text
agent-admin/**
.github/workflows/agent-admin-*.yml
```

非 agent-admin 改动不触发 agent-admin workflow。

agent-admin 改动不触发 sub2api CI 和 Security Scan。

## 13. 测试计划

### 后端

- JWT 本地验签。
- 管理员识别。
- 代理身份识别。
- 代理层级创建校验。
- 下级比例上限校验。
- 差额分成计算。
- 客户单代理归属约束。
- 手动归属下周期生效。
- 分成生成幂等。
- 退款冲正。
- 月底结算。
- 代理禁用。
- 权限隔离。
- 迁移幂等。

### 前端

- 登录态处理。
- 管理员代理列表。
- 指定代理表单。
- 分成比例表单校验。
- 客户手动归属表单。
- 代理商工作台。
- 下级汇总数据展示。
- 冲正记录展示。

### 安全

- 普通用户不能访问 agent-admin。
- 禁用代理不能访问代理端。
- 代理不能访问其他代理客户。
- 上级代理不能访问下级客户明细。
- agent-admin 数据库用户不能写非 agent 表。

## 14. 当前落地状态

已完成：

- agent-admin 独立前端。
- agent-admin 独立 Go 后端。
- 登录代理到 `sub2api`。
- JWT 本地验签。
- agent-admin 自有迁移表和业务表。
- 管理员代理管理、客户归属、佣金、结算、审计 API。
- 基础管理员授权和撤销 agent-admin 管理员 API。
- 管理员代理强制调整和结算调整 API。
- 代理端资料、客户、可发展用户、下级、上级、佣金、结算、订单 API。
- 代理经营前端页面。
- 客户归属变更记录 API 和前端列表。
- 自然推荐关系同步。
- 周期佣金生成。
- 自然月结算生成。
- 单镜像 Docker 构建与部署。
- GitHub Action amd64 镜像构建。
- `sub2api` 后端 agent-admin 代码清理。

仍需增强：

- 退款自动冲正和支付争议自动同步入口。
- 多实例 scheduler 抢锁。
- 更复杂订单和订阅周期匹配规则。
- 前端补充更完整的筛选、分页和导出能力。
- 单独数据库账号和最小权限授权脚本。
