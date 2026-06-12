# Agent Admin 技术设计计划

## 1. 设计目标

`agent-admin` 是 sub2api 的独立关联前端项目。它独立构建和部署，但复用主系统账号、认证、支付订单、订阅套餐和推荐关系数据。

技术设计目标：

- 保持账号体系与原系统一致。
- 在不破坏原推荐系统的前提下扩展代理业务表。
- 支持管理员和代理商登录 `agent-admin`。
- 支持 1/2/3 级代理层级、差额分成、周期结算和冲正。
- 数据隔离在后端服务层强制执行。
- 前端独立构建，后端 API 与主系统共库共服务。

## 2. 技术选型

### 前端

- Vue 3
- TypeScript
- Vite
- Axios
- 原生 CSS

沿用当前 `agent-admin` 项目骨架：

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

沿用 sub2api 主后端：

- Go
- Gin
- GORM
- 现有用户表、认证中间件、支付订单、订阅和推荐关系能力

新增内容优先放在后端代理业务模块中，不改动原推荐系统核心逻辑。

### 数据库

使用主系统同一数据库。新增代理业务表，避免直接改写原推荐系统数据。

金额字段统一使用最小货币单位存储，例如分或 cents。比例字段建议用 basis points 或 decimal：

- 推荐：`rate_bps`，例如 `2000` 表示 `20.00%`。
- 前端展示为百分比，保留两位小数。

## 3. 总体架构

```text
agent-admin Vue app
  |
  | /api/v1/admin/agents/*
  | /api/v1/agent/*
  v
sub2api Go backend
  |
  | existing auth / user / order / subscription / referral
  | new agent service / repositories / scheduler
  v
shared database
```

### 认证边界

- 管理员和代理商使用与原系统一致的账号密码。
- 管理员身份复用原系统管理员认证。
- 代理商身份通过新增代理档案表判断。
- JWT 可继续使用现有 token；后端 API 根据当前用户实时查询代理身份和代理状态。

### 权限边界

- 管理员可查看和管理所有代理数据。
- 代理商只能访问自己直接归属客户。
- 上级代理只能查看下级代理汇总数据和分成贡献，不可查看下级客户明细。
- 客户明细仅直接归属代理和管理员可见。
- 2/3 级代理可查看上级账号的用户名、邮箱、代理等级、联系信息；默认不展示手机号。

## 4. API 设计

### 管理员 API

```text
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
POST   /api/v1/admin/agent-customer-relations
GET    /api/v1/admin/agent-customer-relations/changes
GET    /api/v1/admin/agent-commissions
GET    /api/v1/admin/agent-settlements
POST   /api/v1/admin/agent-settlements/:id/mark-paid
GET    /api/v1/admin/agent-audit-logs
```

### 代理商 API

```text
GET    /api/v1/agent/profile
GET    /api/v1/agent/dashboard
GET    /api/v1/agent/customers
GET    /api/v1/agent/invites
GET    /api/v1/agent/children
POST   /api/v1/agent/children
PUT    /api/v1/agent/children/:id/commission-rate
GET    /api/v1/agent/upline
GET    /api/v1/agent/commissions
GET    /api/v1/agent/settlements
GET    /api/v1/agent/orders
```

### API 响应原则

- 保持主系统 `{ code, message, data }` 包装格式。
- 列表接口统一分页参数：`page`、`page_size`、`search`。
- 时间过滤使用 `start_at`、`end_at`。
- 金额字段后端返回最小货币单位和格式化字符串可二选一；建议返回最小货币单位，由前端格式化。

## 5. 数据结构设计

以下为草案字段，最终以现有数据库命名规范调整。

### agent_profiles

代理身份表。

```text
id                    bigint primary key
user_id               bigint not null unique
level                 int not null                 # 1, 2, 3
parent_agent_id       bigint null
status                varchar(20) not null         # active, disabled
name                  varchar(255) null
contact_info          json null
settlement_info       json null
created_by            bigint not null
created_at            timestamp not null
updated_at            timestamp not null
disabled_at           timestamp null
```

约束：

- `level=1` 时 `parent_agent_id` 必须为空。
- `level=2/3` 时 `parent_agent_id` 必须存在。
- 同一个 `user_id` 同时只能有一个代理身份。
- 禁用后保留历史数据。

### agent_hierarchy_edges

代理层级关系快照表，便于汇总查询和历史追踪。

```text
id                    bigint primary key
agent_id              bigint not null
parent_agent_id       bigint null
level                 int not null
path                  varchar(255) not null         # /1/5/9/
status                varchar(20) not null
effective_at          timestamp not null
expired_at            timestamp null
created_at            timestamp not null
```

### agent_customer_relations

代理客户归属表。

```text
id                    bigint primary key
agent_id              bigint not null
customer_user_id      bigint not null
source                varchar(20) not null          # referral, manual
source_referral_code  varchar(100) null
effective_at          timestamp not null
expired_at            timestamp null
status                varchar(20) not null          # active, expired
created_by            bigint not null
created_at            timestamp not null
```

约束：

- 一个客户同一时间只能有一个 active 代理归属。
- 管理员手动归属从下一个完整套餐周期生效。
- 代理之间不允许自行转移客户归属。

### agent_customer_relation_changes

客户来源调整记录。

```text
id                    bigint primary key
customer_user_id      bigint not null
from_agent_id         bigint null
to_agent_id           bigint not null
reason                text not null
operator_user_id      bigint not null
effective_at          timestamp not null
created_at            timestamp not null
```

`reason` 必填。

### agent_commission_rates

代理分成比例配置表。

```text
id                    bigint primary key
agent_id              bigint not null
rate_bps              int not null                  # 2000 = 20.00%
set_by_user_id        bigint not null
set_by_agent_id       bigint null
effective_at          timestamp not null
expired_at            timestamp null
created_at            timestamp not null
```

默认比例：

- 1 级：`2000`
- 2 级：`1500`
- 3 级：`1000`

规则：

- 管理员可调整 1 级代理比例。
- 1 级代理可调整直属 2 级代理比例。
- 2 级代理可调整直属 3 级代理比例。
- 比例允许 `0`。
- 比例精度到两位小数。
- 下级比例必须低于上级设置给该下级的比例。
- 上级下调比例时，如果下级比例不合法，阻止保存并提示先调整下级比例。

### agent_commission_periods

周期分成记录。

```text
id                    bigint primary key
customer_user_id      bigint not null
agent_id              bigint not null
order_id              bigint not null
subscription_id       bigint null
period_start_at       timestamp not null
period_end_at         timestamp not null
order_paid_amount     bigint not null               # cents/fen
confirmed_revenue     bigint not null               # cents/fen
rate_bps              int not null
commission_amount     bigint not null               # cents/fen, can be negative for reverse
reverse_amount        bigint not null default 0
reverse_reason_type   varchar(50) null              # refund, order_cancel, dispute, admin_adjustment
status                varchar(20) not null          # pending, frozen, payable, paid, reversed
generated_at          timestamp not null
frozen_until          timestamp null
settlement_id         bigint null
created_at            timestamp not null
updated_at            timestamp not null
```

规则：

- 周期确认收入以套餐周期结束时订单实付金额为准。
- 已确认且未退款收入进入分成。
- 结算前退款扣减本周期。
- 结算后退款、订单撤销、支付争议、管理员账务调整生成负数冲正记录。
- 负数冲正可结转抵扣后续分成。

### agent_settlements

结算记录。

```text
id                    bigint primary key
agent_id              bigint not null
period_month          varchar(7) not null            # YYYY-MM
amount                bigint not null
reverse_amount        bigint not null default 0
net_amount            bigint not null
status                varchar(20) not null           # pending, frozen, payable, paid, reversed
min_amount_met        boolean not null
frozen_until          timestamp null
paid_at               timestamp null
paid_by_user_id       bigint null
payment_reference     varchar(255) null
created_at            timestamp not null
updated_at            timestamp not null
```

结算参数：

- 最低结算金额：100 元。
- 冻结期：5 天。
- 结算周期：自然月，到月底结算。
- P0/P1 先由系统生成待结算记录，提现申请后置。

### agent_audit_logs

代理业务审计表。

```text
id                    bigint primary key
operator_user_id      bigint not null
operator_role         varchar(20) not null           # admin, agent
action                varchar(100) not null
target_type           varchar(100) not null
target_id             bigint not null
before_data           json null
after_data            json null
reason                text null
created_at            timestamp not null
```

必须记录：

- 指定代理。
- 修改层级。
- 修改分成比例。
- 手动归属客户。
- 禁用/恢复代理。
- 管理员账务调整。

## 6. 核心业务流程

### 指定代理

1. 管理员选择原系统用户。
2. 管理员选择代理等级。
3. 如果是 2/3 级，必须选择上级代理。
4. 系统校验层级合法性和用户唯一代理身份。
5. 系统创建 `agent_profiles` 和 `agent_hierarchy_edges`。
6. 系统写入默认分成比例。
7. 系统写入审计日志。

### 创建下级代理

1. 代理选择自己推荐码引入的推荐用户。
2. 系统校验该用户未拥有代理身份。
3. 系统校验创建层级不超过 3 级。
4. 系统使用默认比例或代理设置的比例创建下级代理。
5. 不需要管理员审核。
6. 管理员可在后台查看并调整。

### 手动归属客户

1. 管理员选择客户和目标代理。
2. 管理员填写调整原因。
3. 系统计算下一个完整套餐周期作为 `effective_at`。
4. 系统关闭旧 active 关系。
5. 系统创建新 active 关系。
6. 系统写入 `agent_customer_relation_changes` 和审计日志。

### 分成生成

1. 每日定时任务扫描已到期套餐周期。
2. 系统读取订单实付金额作为周期确认收入。
3. 系统读取该周期生效的客户归属和分成比例快照。
4. 系统按差额分成生成每级代理分成记录。
5. 系统设置冻结期 5 天。
6. 系统生成 `pending` 或 `frozen` 状态记录。
7. 如果已有结算后退款，生成负数冲正记录。

### 月底结算

1. 自然月结束后聚合代理可结算分成。
2. 系统扣除负数冲正。
3. 系统检查净额是否达到 100 元。
4. 达标则生成待结算记录。
5. 未达标则保留到后续周期。
6. 管理员审核和标记已支付。

### 禁用代理

1. 管理员禁用代理身份。
2. 该用户不能以代理身份登录 `agent-admin`。
3. 该代理不再产生新分成。
4. 历史归属、分成、结算保留。
5. 下级代理继续有效。
6. 被禁用代理原本应获得的差额分成归平台。
7. 未重新归属客户显示为“无有效代理分成”，支持管理员批量重新归属。

## 7. 前端页面设计

### 管理员页面

- 代理列表。
- 新增/编辑代理。
- 代理详情。
- 代理客户。
- 下级代理。
- 分成比例配置。
- 客户手动归属。
- 分成记录。
- 结算记录。
- 冲正记录。
- 审计日志。

### 代理商页面

- 工作台。
- 我的客户。
- 推荐用户。
- 下级代理汇总。
- 下级代理创建。
- 下级比例设置。
- 上级代理账号。
- 分成记录。
- 结算记录。
- 冲正记录。
- 订单与套餐。

### 数据脱敏

代理端：

- 不展示完整 API Key。
- 不展示上游账号。
- 不展示管理员内部备注。
- 不展示上级代理手机号。
- 不展示下级客户明细。

## 8. 编译打包流程

### 本地开发

```bash
cd agent-admin
pnpm install
pnpm dev
```

默认开发地址：

```text
http://localhost:3100/
```

默认代理：

```text
/api -> http://localhost:8080
```

可通过环境变量覆盖：

```bash
VITE_DEV_PROXY_TARGET=http://localhost:8080
VITE_DEV_PORT=3100
```

### 类型检查

```bash
cd agent-admin
pnpm typecheck
```

### 生产构建

```bash
cd agent-admin
pnpm build
```

输出目录：

```text
agent-admin/dist
```

### Docker 镜像构建

`agent-admin` 可以打包为独立容器镜像。推荐使用多阶段构建：第一阶段使用 Node/pnpm 构建静态资源，第二阶段使用 Nginx 或 Caddy 提供静态文件并反向代理 `/api` 到 sub2api 后端。

示例构建命令：

```bash
docker build -f agent-admin/Dockerfile -t sub2api-agent-admin:latest agent-admin
```

### 预览

```bash
cd agent-admin
pnpm preview
```

## 9. 部署方案

### 独立静态部署

推荐将 `agent-admin/dist` 部署到独立域名：

```text
https://agent.example.com
```

反向代理：

```text
https://agent.example.com/api -> sub2api backend /api
```

优点：

- 前后端边界清晰。
- 不影响主前端构建。
- 后续可独立迭代和灰度发布。

### 后端内嵌部署

可选方案：将 `agent-admin/dist` 嵌入 Go 后端或由同一个 Nginx 服务提供。

适用场景：

- 运维希望单服务部署。
- 不需要独立域名。

注意：

- 需要避免与现有 `frontend` 构建输出冲突。
- 建议使用独立路径，例如 `/agent-admin/`。
- Vite `base` 需要按部署路径配置。

### Docker 镜像部署

`agent-admin` 支持作为独立容器部署：

```bash
docker run -d \
  --name sub2api-agent-admin \
  -p 3100:80 \
  -e API_PROXY_TARGET=http://sub2api-backend:8080 \
  sub2api-agent-admin:latest
```

容器职责：

- 提供 `agent-admin/dist` 静态资源。
- 将 `/api` 请求反向代理到 sub2api 后端。
- 支持独立滚动升级，不影响主前端。

推荐镜像结构：

```text
agent-admin image
  /usr/share/nginx/html        # Vite build output
  /etc/nginx/conf.d/default.conf
```

示例 Dockerfile：

```dockerfile
FROM node:20-alpine AS build
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

示例 Nginx 配置：

```nginx
server {
  listen 80;
  server_name _;

  root /usr/share/nginx/html;
  index index.html;

  location / {
    try_files $uri $uri/ /index.html;
  }

  location /api/ {
    proxy_pass http://sub2api-backend:8080/api/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
```

Docker Compose 示例：

```yaml
services:
  agent-admin:
    image: sub2api-agent-admin:latest
    container_name: sub2api-agent-admin
    restart: unless-stopped
    ports:
      - "3100:80"
    depends_on:
      - backend
    networks:
      - sub2api

  backend:
    image: sub2api-backend:latest
    container_name: sub2api-backend
    restart: unless-stopped
    networks:
      - sub2api

networks:
  sub2api:
    external: true
```

如果部署在主系统同一个 Compose 栈中，`proxy_pass` 可指向后端服务名，例如 `http://backend:8080/api/`。如果部署在独立机器或独立网络中，应改为公网或内网后端地址。

## 10. 运行任务与调度

### 每日分成生成任务

- 频率：每日一次。
- 输入：已到期套餐周期、已支付订单、有效客户归属。
- 输出：周期分成记录。
- 幂等键：`customer_user_id + order_id + period_start_at + period_end_at + agent_id`。

### 月底结算聚合任务

- 频率：自然月月底或次日。
- 输入：可结算分成记录、冲正记录。
- 输出：结算记录。
- 未达到 100 元的净额保留到后续周期。

### 冲正生成任务

- 触发：退款、订单撤销、支付争议、管理员账务调整。
- 输出：负数周期分成记录或冲正记录。
- 抵扣：后续结算自动抵扣。

## 11. 测试计划

### 后端测试

- 代理层级创建校验。
- 下级比例上限校验。
- 差额分成计算。
- 客户单代理归属约束。
- 手动归属下周期生效。
- 每日定时任务幂等。
- 退款冲正。
- 代理禁用后的分成归平台。
- 权限隔离。

### 前端测试

- 管理员代理列表。
- 指定代理表单。
- 分成比例表单校验。
- 客户手动归属表单。
- 代理商工作台。
- 下级汇总数据展示。
- 冲正记录展示。

### 安全测试

- 代理不能访问其他代理客户。
- 上级代理不能访问下级客户明细。
- 禁用代理不能登录代理端。
- 非代理普通用户不能登录 `agent-admin`。

## 12. 未决事项

目前剩余需要后续确认或依赖代码梳理的事项：

- 周期确认收入对应的具体后端表和字段名。
- 原系统推荐模块兼容记录的启用时机和写入方式。
- 后端现有订单、订阅、推荐模型与新增代理表的字段映射。
- `agent-admin` 是否最终使用独立域名或后端内嵌路径。
