#!/usr/bin/env bash
set -Eeuo pipefail

# This script is called by the private Mac mini pull controller. The Docker
# daemon is Docker Desktop on the host, so bind mounts must use the host path
# (MACMINI_DEPLOY_ROOT), not the controller container's /opt/deploy path.

APP_NAME="${APP_NAME:-websitecore-demo-app}"
DB_NAME="${DB_NAME:-websitecore-demo-db}"
REDIS_NAME="${REDIS_NAME:-websitecore-demo-redis}"
MEILI_NAME="${MEILI_NAME:-websitecore-demo-meili}"
APP_NETWORK="${APP_NETWORK:-websitecore-demo-net}"
TUNNEL_NETWORK="${TUNNEL_NETWORK:-gsc-net}"
APP_IMAGE_REPO="${APP_IMAGE_REPO:-websitecore-demo-app}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:18.6}"
REDIS_IMAGE="${REDIS_IMAGE:-redis:7.4.11}"
MEILI_IMAGE="${MEILI_IMAGE:-getmeili/meilisearch:v1.54.0}"
HOST_ROOT="${MACMINI_DEPLOY_ROOT:?MACMINI_DEPLOY_ROOT must point to the host deployment directory}"
RUNTIME_DIR="${RUNTIME_DIR:-$HOST_ROOT/runtime}"
CONFIG_FILE="${RUNTIME_DIR}/custom/config.yaml"
POSTGRES_ENV_FILE="${RUNTIME_DIR}/postgres.env"
MEILI_ENV_FILE="${RUNTIME_DIR}/meili.env"
SOURCE_DIR="${GITHUB_WORKSPACE:-$(git rev-parse --show-toplevel)}"
TARGET_SHA="${GITHUB_SHA:-$(git -C "$SOURCE_DIR" rev-parse HEAD)}"
BUILD_HTTP_PROXY="${BUILD_HTTP_PROXY:-http://host.docker.internal:7890}"
BUILD_HTTPS_PROXY="${BUILD_HTTPS_PROXY:-$BUILD_HTTP_PROXY}"

log() { printf '[websitecore-deploy] %s\n' "$*"; }
fail() { log "ERROR: $*" >&2; exit 1; }

[[ -d "$SOURCE_DIR" ]] || fail "source directory does not exist: $SOURCE_DIR"
[[ -f "$SOURCE_DIR/Dockerfile" ]] || fail "Dockerfile is missing from $SOURCE_DIR"
[[ -f "$CONFIG_FILE" ]] || fail "runtime config is missing: $CONFIG_FILE"
[[ -f "$POSTGRES_ENV_FILE" ]] || fail "database env file is missing: $POSTGRES_ENV_FILE"
[[ -f "$MEILI_ENV_FILE" ]] || fail "Meilisearch env file is missing: $MEILI_ENV_FILE"
[[ "$TARGET_SHA" =~ ^[0-9a-f]{40}$ ]] || fail "GITHUB_SHA is not a full commit SHA"
[[ "$(git -C "$SOURCE_DIR" rev-parse HEAD)" == "$TARGET_SHA" ]] || fail "source HEAD does not match GITHUB_SHA"
git -C "$SOURCE_DIR" diff --quiet || fail "source checkout is dirty"

mkdir -p "$RUNTIME_DIR" "$RUNTIME_DIR/postgres" "$RUNTIME_DIR/redis" "$RUNTIME_DIR/meili" "$RUNTIME_DIR/custom/data" "$RUNTIME_DIR/backups"
chmod 700 "$RUNTIME_DIR" "$RUNTIME_DIR/backups"
chmod 600 "$CONFIG_FILE" "$POSTGRES_ENV_FILE" "$MEILI_ENV_FILE"

command -v docker >/dev/null 2>&1 || fail "docker is not installed in the controller"
docker info >/dev/null 2>&1 || fail "the controller cannot reach Docker Desktop"
docker network inspect "$APP_NETWORK" >/dev/null 2>&1 || docker network create "$APP_NETWORK" >/dev/null
docker network inspect "$TUNNEL_NETWORK" >/dev/null 2>&1 || fail "required Cloudflare tunnel network is missing: $TUNNEL_NETWORK"

container_exists() { docker container inspect "$1" >/dev/null 2>&1; }
container_running() { [[ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null || true)" == "true" ]]; }

wait_healthy() {
  local name="$1" timeout="${2:-180}" status="" i
  for ((i=0; i<timeout; i++)); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$name" 2>/dev/null || true)"
    case "$status" in
      healthy|running) return 0 ;;
      unhealthy|exited|dead)
        docker logs --tail 80 "$name" >&2 || true
        return 1
        ;;
    esac
    sleep 1
  done
  docker logs --tail 80 "$name" >&2 || true
  return 1
}

ensure_postgres() {
  if ! container_exists "$DB_NAME"; then
    log "creating PostgreSQL container"
    docker run -d --name "$DB_NAME" --restart unless-stopped \
      --network "$APP_NETWORK" \
      --env-file "$POSTGRES_ENV_FILE" \
      -e HTTP_PROXY= -e HTTPS_PROXY= -e ALL_PROXY= \
      -e http_proxy= -e https_proxy= -e all_proxy= \
      -v "$HOST_ROOT/runtime/postgres:/var/lib/postgresql" \
      --health-cmd 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
      --health-interval 5s --health-timeout 5s --health-retries 24 --health-start-period 20s \
      "$POSTGRES_IMAGE" >/dev/null
  elif ! container_running "$DB_NAME"; then
    log "starting existing PostgreSQL container"
    docker start "$DB_NAME" >/dev/null
  fi
  wait_healthy "$DB_NAME" 240 || fail "PostgreSQL did not become healthy"
}

ensure_redis() {
  if ! container_exists "$REDIS_NAME"; then
    log "creating Redis container"
    docker run -d --name "$REDIS_NAME" --restart unless-stopped \
      --network "$APP_NETWORK" \
      -e HTTP_PROXY= -e HTTPS_PROXY= -e ALL_PROXY= \
      -e http_proxy= -e https_proxy= -e all_proxy= \
      -v "$HOST_ROOT/runtime/redis:/data" \
      --health-cmd 'redis-cli ping | grep -q PONG' \
      --health-interval 5s --health-timeout 5s --health-retries 24 \
      "$REDIS_IMAGE" redis-server --appendonly yes >/dev/null
  elif ! container_running "$REDIS_NAME"; then
    log "starting existing Redis container"
    docker start "$REDIS_NAME" >/dev/null
  fi
  wait_healthy "$REDIS_NAME" 120 || fail "Redis did not become healthy"
}

ensure_meili() {
  if ! container_exists "$MEILI_NAME"; then
    log "creating Meilisearch container"
    docker run -d --name "$MEILI_NAME" --restart unless-stopped \
      --network "$APP_NETWORK" \
      --env-file "$MEILI_ENV_FILE" \
      -e MEILI_ENV=production \
      -e MEILI_NO_ANALYTICS=true \
      -e HTTP_PROXY= -e HTTPS_PROXY= -e ALL_PROXY= \
      -e http_proxy= -e https_proxy= -e all_proxy= \
      -v "$HOST_ROOT/runtime/meili:/meili_data" \
      --health-cmd 'curl -fsS http://127.0.0.1:7700/health >/dev/null' \
      --health-interval 5s --health-timeout 5s --health-retries 24 --health-start-period 15s \
      "$MEILI_IMAGE" >/dev/null
  elif ! container_running "$MEILI_NAME"; then
    log "starting existing Meilisearch container"
    docker start "$MEILI_NAME" >/dev/null
  fi
  wait_healthy "$MEILI_NAME" 180 || fail "Meilisearch did not become healthy"
}

backup_database() {
  local backup="$RUNTIME_DIR/backups/$(date -u +%Y%m%dT%H%M%SZ)-${TARGET_SHA}.sql.gz"
  log "backing up PostgreSQL before app cutover: $backup"
  docker exec "$DB_NAME" pg_dump -U websitecore -d websitecore --no-owner --no-privileges | gzip -c >"$backup"
  chmod 600 "$backup"
  [[ -s "$backup" ]] || fail "PostgreSQL backup is empty: $backup"
}

run_app() {
  local image="$1" revision="$2"
  docker run -d --name "$APP_NAME" --restart unless-stopped \
    --network "$APP_NETWORK" \
    --label "org.opencontainers.image.source=https://github.com/BZYA-Community/WebsiteCore" \
    --label "org.opencontainers.image.revision=$revision" \
    --publish 127.0.0.1:18008:8008 \
    --env HTTP_PROXY= -e HTTPS_PROXY= -e ALL_PROXY= \
    --env http_proxy= -e https_proxy= -e all_proxy= \
    --volume "$HOST_ROOT/runtime/custom:/app/custom" \
    "$image" >/dev/null
  if ! docker network connect "$TUNNEL_NETWORK" "$APP_NAME" >/dev/null; then
    docker rm -f "$APP_NAME" >/dev/null 2>&1 || true
    return 1
  fi
}

app_api_ready() {
  docker exec "$APP_NAME" wget -q -O /dev/null --timeout=5 \
    'http://127.0.0.1:8008/v1/posts?style=newest'
}

wait_app_ready() {
  local timeout="${1:-240}" status="" i
  for ((i=0; i<timeout; i++)); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$APP_NAME" 2>/dev/null || true)"
    if [[ "$status" == "healthy" ]] && app_api_ready; then
      return 0
    fi
    if [[ "$status" == "unhealthy" || "$status" == "exited" || "$status" == "dead" ]]; then
      docker logs --tail 120 "$APP_NAME" >&2 || true
      return 1
    fi
    sleep 1
  done
  docker logs --tail 120 "$APP_NAME" >&2 || true
  return 1
}

ensure_postgres
ensure_redis
ensure_meili

IMAGE="${APP_IMAGE_REPO}:${TARGET_SHA}"
log "building immutable app image $IMAGE"
docker build --pull \
  --build-arg HTTP_PROXY="$BUILD_HTTP_PROXY" \
  --build-arg HTTPS_PROXY="$BUILD_HTTPS_PROXY" \
  -t "$IMAGE" "$SOURCE_DIR"

previous_image_id=""
previous_sha=""
if container_exists "$APP_NAME"; then
  previous_image_id="$(docker inspect -f '{{.Image}}' "$APP_NAME")"
  previous_sha="$(docker inspect -f '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$APP_NAME" 2>/dev/null || true)"
  backup_database
  log "removing previous app container (image=$previous_image_id sha=${previous_sha:-unknown})"
  docker rm -f "$APP_NAME" >/dev/null
else
  backup_database
fi

log "starting app image $IMAGE"
if ! run_app "$IMAGE" "$TARGET_SHA" || ! wait_app_ready 240; then
  log "new app failed health/API check; attempting immutable image rollback" >&2
  docker logs --tail 120 "$APP_NAME" >&2 || true
  docker rm -f "$APP_NAME" >/dev/null 2>&1 || true
  if [[ -n "$previous_image_id" ]]; then
    if run_app "$previous_image_id" "${previous_sha:-unknown}" && wait_app_ready 180; then
      log "previous app image restored; database schema was not rolled back automatically" >&2
    else
      fail "new app failed and previous app image could not be restored"
    fi
  fi
  fail "new app failed health/API check"
fi

docker tag "$IMAGE" "${APP_IMAGE_REPO}:current"
printf '%s\n' "$TARGET_SHA" >"$RUNTIME_DIR/deployed_sha"
chmod 600 "$RUNTIME_DIR/deployed_sha"
log "deployment succeeded: $TARGET_SHA"
