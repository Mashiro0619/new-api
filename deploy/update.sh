#!/bin/sh

set -eu

DEFAULT_IMAGE="ghcr.io/mashiro0619/new-api"
SERVICE_NAME="new-api"
HEALTH_TIMEOUT_SECONDS=180
POLL_INTERVAL_SECONDS=5

usage() {
  printf 'Usage: %s <image-tag>\n' "$0" >&2
  printf 'Example: %s sha-0123456789abcdef\n' "$0" >&2
}

die() {
  printf 'Error: %s\n' "$1" >&2
  exit 1
}

shell_quote() {
  printf "'"
  printf '%s' "$1" | sed "s/'/'\"'\"'/g"
  printf "'"
}

compose() {
  docker compose \
    -f "$BASE_COMPOSE_FILE" \
    -f "$OVERRIDE_COMPOSE_FILE" \
    "$@"
}

print_rollback_command() {
  printf '\nThe previous image is still available locally as %s.\n' "$PREVIOUS_IMAGE_ID" >&2
  printf 'Run this exact command to recreate only %s with that image:\n\n' "$SERVICE_NAME" >&2
  printf 'docker image tag ' >&2
  shell_quote "$PREVIOUS_IMAGE_ID" >&2
  printf ' ' >&2
  shell_quote "$ROLLBACK_IMAGE" >&2
  printf ' && FORK_IMAGE=' >&2
  shell_quote "$FORK_IMAGE" >&2
  printf ' FORK_IMAGE_TAG=' >&2
  shell_quote "$ROLLBACK_TAG" >&2
  printf ' docker compose -f ' >&2
  shell_quote "$BASE_COMPOSE_FILE" >&2
  printf ' -f ' >&2
  shell_quote "$OVERRIDE_COMPOSE_FILE" >&2
  printf ' up -d --no-deps --force-recreate ' >&2
  shell_quote "$SERVICE_NAME" >&2
  printf '\n\nThis command does not restore or modify any database.\n' >&2
}

fail_update() {
  printf 'Update failed: %s\n' "$1" >&2
  print_rollback_command
  exit 1
}

if [ "$#" -ne 1 ]; then
  usage
  exit 64
fi

TARGET_TAG=$1
case "$TARGET_TAG" in
  '' | [!A-Za-z0-9_]* | *[!A-Za-z0-9_.-]*)
    die "invalid image tag: $TARGET_TAG"
    ;;
esac

if [ "${#TARGET_TAG}" -gt 128 ]; then
  die "image tags must be no longer than 128 characters"
fi

FORK_IMAGE=${FORK_IMAGE:-$DEFAULT_IMAGE}
case "$FORK_IMAGE" in
  '' | [!a-z0-9]* | *[!a-z0-9._:/-]* | *@* | */ | *:)
    die "FORK_IMAGE must be a lowercase image repository without a tag or digest"
    ;;
esac

IMAGE_NAME=${FORK_IMAGE##*/}
case "$IMAGE_NAME" in
  *:*)
    die "FORK_IMAGE must not include an image tag; pass the tag as the only argument"
    ;;
esac

command -v docker >/dev/null 2>&1 || die "docker is not installed or is not in PATH"
command -v sed >/dev/null 2>&1 || die "sed is not installed or is not in PATH"
command -v jq >/dev/null 2>&1 || die "jq is required to inspect the rendered Compose service image"

if ! docker compose version >/dev/null 2>&1; then
  die "Docker Compose v2 is required (the 'docker compose' command)"
fi

if ! docker info >/dev/null 2>&1; then
  die "the Docker daemon is not reachable"
fi

SCRIPT_DIR=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
PROJECT_DIR=$(CDPATH='' cd -P "$SCRIPT_DIR/.." && pwd)
BASE_COMPOSE_FILE="$PROJECT_DIR/docker-compose.yml"
OVERRIDE_COMPOSE_FILE="$PROJECT_DIR/docker-compose.override.yml"

[ -r "$BASE_COMPOSE_FILE" ] || die "base Compose file is not readable: $BASE_COMPOSE_FILE"
[ -r "$OVERRIDE_COMPOSE_FILE" ] || die "override Compose file is not readable: $OVERRIDE_COMPOSE_FILE"

FORK_IMAGE_TAG=$TARGET_TAG
export FORK_IMAGE FORK_IMAGE_TAG
TARGET_IMAGE="$FORK_IMAGE:$FORK_IMAGE_TAG"

if ! compose config >/dev/null; then
  die "the merged Compose configuration is invalid"
fi

if ! RESOLVED_SERVICE_IMAGE=$(compose config --format json "$SERVICE_NAME" | jq -r --arg service "$SERVICE_NAME" '.services[$service].image // empty'); then
  die "Docker Compose could not resolve the $SERVICE_NAME service image"
fi
if [ "$RESOLVED_SERVICE_IMAGE" != "$TARGET_IMAGE" ]; then
  die "the merged Compose configuration resolved $SERVICE_NAME to $RESOLVED_SERVICE_IMAGE, expected $TARGET_IMAGE"
fi

if ! PREVIOUS_CONTAINER=$(compose ps --all --quiet "$SERVICE_NAME"); then
  die "could not inspect the current $SERVICE_NAME service"
fi
PREVIOUS_CONTAINER=$(printf '%s\n' "$PREVIOUS_CONTAINER" | sed -n '1p')
[ -n "$PREVIOUS_CONTAINER" ] || die "no existing $SERVICE_NAME container was found; this updater requires an existing deployment"

if ! PREVIOUS_IMAGE_REF=$(docker inspect --format '{{.Config.Image}}' "$PREVIOUS_CONTAINER"); then
  die "could not inspect the current $SERVICE_NAME image reference"
fi
if ! PREVIOUS_IMAGE_ID=$(docker inspect --format '{{.Image}}' "$PREVIOUS_CONTAINER"); then
  die "could not inspect the current $SERVICE_NAME image ID"
fi
if ! PREVIOUS_REVISION=$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$PREVIOUS_IMAGE_ID" 2>/dev/null); then
  PREVIOUS_REVISION="unknown"
fi
case "$PREVIOUS_REVISION" in
  '' | '<no value>') PREVIOUS_REVISION="unknown" ;;
esac

ROLLBACK_TAG="rollback-$(date +%Y%m%d%H%M%S)"
ROLLBACK_IMAGE="$FORK_IMAGE:$ROLLBACK_TAG"

printf 'Current container: %s\n' "$PREVIOUS_CONTAINER"
printf 'Current image ref: %s\n' "$PREVIOUS_IMAGE_REF"
printf 'Current image ID: %s\n' "$PREVIOUS_IMAGE_ID"
printf 'Current revision: %s\n' "$PREVIOUS_REVISION"
printf 'Target image: %s\n' "$TARGET_IMAGE"

if ! compose pull "$SERVICE_NAME"; then
  fail_update "could not pull $TARGET_IMAGE"
fi

if ! compose up -d --no-deps --force-recreate "$SERVICE_NAME"; then
  fail_update "Docker Compose could not recreate $SERVICE_NAME"
fi

if ! UPDATED_CONTAINER=$(compose ps --all --quiet "$SERVICE_NAME"); then
  fail_update "could not inspect the recreated $SERVICE_NAME service"
fi
UPDATED_CONTAINER=$(printf '%s\n' "$UPDATED_CONTAINER" | sed -n '1p')
[ -n "$UPDATED_CONTAINER" ] || fail_update "the recreated $SERVICE_NAME container was not found"

printf 'Waiting up to %s seconds for the existing container healthcheck...\n' "$HEALTH_TIMEOUT_SECONDS"
ELAPSED_SECONDS=0
while [ "$ELAPSED_SECONDS" -lt "$HEALTH_TIMEOUT_SECONDS" ]; do
  if ! CONTAINER_STATE=$(docker inspect --format '{{.State.Running}} {{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$UPDATED_CONTAINER" 2>/dev/null); then
    fail_update "the recreated container disappeared during its healthcheck"
  fi

  IS_RUNNING=${CONTAINER_STATE%% *}
  HEALTH_STATUS=${CONTAINER_STATE#* }

  if [ "$IS_RUNNING" != "true" ]; then
    fail_update "the recreated container stopped before becoming healthy"
  fi

  case "$HEALTH_STATUS" in
    healthy)
      break
      ;;
    unhealthy)
      fail_update "the recreated container reported an unhealthy state"
      ;;
    missing)
      fail_update "the merged service does not provide the expected container healthcheck"
      ;;
  esac

  sleep "$POLL_INTERVAL_SECONDS"
  ELAPSED_SECONDS=$((ELAPSED_SECONDS + POLL_INTERVAL_SECONDS))
done

if [ "$HEALTH_STATUS" != "healthy" ]; then
  fail_update "the recreated container did not become healthy within $HEALTH_TIMEOUT_SECONDS seconds"
fi

if ! STATUS_RESPONSE=$(docker exec "$UPDATED_CONTAINER" wget -q -O - http://127.0.0.1:3000/api/status 2>/dev/null); then
  fail_update "the /api/status request failed after the container became healthy"
fi

if ! printf '%s\n' "$STATUS_RESPONSE" | jq -e 'type == "object" and .success == true' >/dev/null; then
  fail_update "the /api/status response did not report a top-level success=true"
fi

if ! UPDATED_IMAGE_ID=$(docker inspect --format '{{.Image}}' "$UPDATED_CONTAINER"); then
  fail_update "could not inspect the updated image ID"
fi
if ! UPDATED_REVISION=$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$UPDATED_IMAGE_ID" 2>/dev/null); then
  UPDATED_REVISION="unknown"
fi
case "$UPDATED_REVISION" in
  '' | '<no value>') UPDATED_REVISION="unknown" ;;
esac

printf 'Update succeeded.\n'
printf 'Updated container: %s\n' "$UPDATED_CONTAINER"
printf 'Updated image ID: %s\n' "$UPDATED_IMAGE_ID"
printf 'Updated revision: %s\n' "$UPDATED_REVISION"
printf '/api/status reported success=true. PostgreSQL and Redis were not recreated.\n'
