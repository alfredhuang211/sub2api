#!/bin/sh
set -eu

: "${AGENT_ADMIN_API_ADDR:=127.0.0.1:3101}"
export AGENT_ADMIN_API_ADDR

/usr/local/bin/agent-admin-api &
api_pid="$!"

nginx -g "daemon off;" &
nginx_pid="$!"

terminate() {
  kill "$nginx_pid" 2>/dev/null || true
  kill "$api_pid" 2>/dev/null || true
}

trap terminate INT TERM

while kill -0 "$nginx_pid" 2>/dev/null && kill -0 "$api_pid" 2>/dev/null; do
  sleep 2
done

terminate
wait "$nginx_pid" 2>/dev/null || true
wait "$api_pid" 2>/dev/null || true
