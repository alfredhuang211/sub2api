FROM node:20-alpine AS web-build
WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@9.15.9 --activate && pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

FROM golang:1.23-alpine AS api-build
WORKDIR /src

COPY api/go.mod api/go.sum ./api/
WORKDIR /src/api
RUN go mod download

COPY api/ ./
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/agent-admin-api ./cmd/server

FROM nginx:1.27-alpine

COPY --from=web-build /app/dist /usr/share/nginx/html
COPY --from=api-build /out/agent-admin-api /usr/local/bin/agent-admin-api
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV AGENT_ADMIN_API_ADDR=127.0.0.1:3101
EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
