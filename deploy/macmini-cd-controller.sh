#!/usr/bin/env bash
set -Eeuo pipefail

# Runs inside the private Mac mini pull-controller container. It only follows
# the public main branch; pull requests never enter this path.

REPOSITORY_URL="${REPOSITORY_URL:-https://github.com/BZYA-Community/WebsiteCore.git}"
SOURCE_DIR="${SOURCE_DIR:-/opt/source}"
RUNTIME_DIR="${RUNTIME_DIR:-/opt/deploy/runtime}"
HOST_ROOT="${MACMINI_DEPLOY_ROOT:?MACMINI_DEPLOY_ROOT must be the host deployment root}"
POLL_SECONDS="${POLL_SECONDS:-60}"

log() { printf '[websitecore-cd] %s\n' "$*"; }

main_sha() {
  git ls-remote "$REPOSITORY_URL" refs/heads/main | awk 'NR == 1 { print $1; exit }'
}

checkout_main() {
  local sha="$1"
  mkdir -p "$(dirname "$SOURCE_DIR")"
  if [[ ! -d "$SOURCE_DIR/.git" ]]; then
    rm -rf "$SOURCE_DIR"
    git clone --filter=blob:none --depth=1 --branch main "$REPOSITORY_URL" "$SOURCE_DIR"
  else
    git -C "$SOURCE_DIR" fetch --depth=1 origin main
    git -C "$SOURCE_DIR" reset --hard FETCH_HEAD
    git -C "$SOURCE_DIR" clean -fdx
  fi
  [[ "$(git -C "$SOURCE_DIR" rev-parse HEAD)" == "$sha" ]]
}

deploy_once() {
  local sha="$1"
  checkout_main "$sha"
  if [[ ! -f "$SOURCE_DIR/Dockerfile" || ! -f "$SOURCE_DIR/deploy/macmini-deploy.sh" ]]; then
    log "main $sha has no deploy contract yet; waiting for the CD merge"
    return 2
  fi
  MACMINI_DEPLOY_ROOT="$HOST_ROOT" \
  RUNTIME_DIR="$RUNTIME_DIR" \
  GITHUB_WORKSPACE="$SOURCE_DIR" \
  GITHUB_SHA="$sha" \
  bash "$SOURCE_DIR/deploy/macmini-deploy.sh"
}

mkdir -p "$RUNTIME_DIR"
log "watching $REPOSITORY_URL main every ${POLL_SECONDS}s"
while :; do
  if sha="$(main_sha 2>/dev/null)" && [[ "$sha" =~ ^[0-9a-f]{40}$ ]]; then
    deployed=""
    [[ -f "$RUNTIME_DIR/deployed_sha" ]] && deployed="$(tr -d '[:space:]' < "$RUNTIME_DIR/deployed_sha")"
    if [[ "$sha" != "$deployed" ]]; then
      log "new main commit $sha (deployed=${deployed:-none})"
      if deploy_once "$sha"; then
        log "main commit $sha deployed"
      else
        rc=$?
        [[ "$rc" == 2 ]] || log "deployment failed for $sha; will retry on the next poll" >&2
      fi
    fi
  else
    log "cannot read main SHA; will retry on the next poll" >&2
  fi
  sleep "$POLL_SECONDS"
done
