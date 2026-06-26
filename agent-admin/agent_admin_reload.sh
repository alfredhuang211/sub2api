#!/usr/bin/env bash
set -Eeuo pipefail

CONTAINER_NAME="${CONTAINER_NAME:-sub2api-agent-admin}"
IMAGE="${IMAGE:-ghcr.io/alfredhuang211/sub2api-agent-admin:latest}"
NETWORK="${NETWORK:-sub2api_sub2api-network}"
HOST_PORT="${HOST_PORT:-3100}"
SUB2API_BASE_URL="${SUB2API_BASE_URL:-http://sub2api:8080}"
JWT_SIGNING_METHOD="${JWT_SIGNING_METHOD:-HS256}"
MIGRATION_ENABLED="${MIGRATION_ENABLED:-true}"
SCHEDULER_ENABLED="${SCHEDULER_ENABLED:-true}"
ENV_FILE="${ENV_FILE:-.env}"

log() {
  printf '[agent-admin] %s\n' "$*"
}

die() {
  printf '[agent-admin] ERROR: %s\n' "$*" >&2
  exit 1
}

command -v docker >/dev/null 2>&1 || die "docker command not found"

if [ -f "$ENV_FILE" ]; then
  log "loading env file: $ENV_FILE"
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
else
  log "env file not found, using current shell environment: $ENV_FILE"
fi

: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${JWT_SECRET:?JWT_SECRET is required}"

POSTGRES_USER="${POSTGRES_USER:-sub2api}"
POSTGRES_DB="${POSTGRES_DB:-sub2api}"
DATABASE_URL="${DATABASE_URL:-postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable}"

if ! docker network inspect "$NETWORK" >/dev/null 2>&1; then
  die "docker network not found: $NETWORK"
fi

existing_container_id="$(docker ps -aq --filter "name=^/${CONTAINER_NAME}$" | head -n 1 || true)"
if [ -n "$existing_container_id" ]; then
  running_container_id="$(docker ps -q --filter "name=^/${CONTAINER_NAME}$" | head -n 1 || true)"
  if [ -n "$running_container_id" ]; then
    log "stopping running container: $CONTAINER_NAME"
    docker stop "$CONTAINER_NAME" >/dev/null
  else
    log "container exists but is not running: $CONTAINER_NAME"
  fi

  log "removing old container: $CONTAINER_NAME"
  docker rm "$CONTAINER_NAME" >/dev/null
else
  log "no existing container found: $CONTAINER_NAME"
fi

log "pulling latest image: $IMAGE"
docker pull "$IMAGE"

log "starting container: $CONTAINER_NAME"
docker run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  --network "$NETWORK" \
  -p "${HOST_PORT}:80" \
  -e SUB2API_BASE_URL="$SUB2API_BASE_URL" \
  -e DATABASE_URL="$DATABASE_URL" \
  -e JWT_SECRET="$JWT_SECRET" \
  -e JWT_SIGNING_METHOD="$JWT_SIGNING_METHOD" \
  -e MIGRATION_ENABLED="$MIGRATION_ENABLED" \
  -e SCHEDULER_ENABLED="$SCHEDULER_ENABLED" \
  "$IMAGE"

log "started successfully"
log "url: http://localhost:${HOST_PORT}"
