#!/usr/bin/env bash
# Deploy lexmora-api or lexmora-webui to Irancell-T3 (matches .armin/docker-scripts flow).
# VPN-safe: SSH via host :22 (not HAProxy :443), never compose down, never recreate
# windscribe/softether/haproxy stacks; only lexmora-* service names are started.
set -euo pipefail

COMPONENT="${1:?Usage: deploy-t3-component.sh api|webui}"

: "${T3_HOST:?}"
: "${T3_SSH_USER:?}"
: "${T3_SSH_PORT:=22}"
: "${GITHUB_WORKSPACE:?}"

DEPLOY_KEY="${HOME}/.ssh/deploy_key"
SSH_BASE=(ssh -i "$DEPLOY_KEY" -p "$T3_SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=yes)
SCP_BASE=(scp -i "$DEPLOY_KEY" -P "$T3_SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=yes)

REMOTE="${T3_SSH_USER}@${T3_HOST}"
STACK_NAME=lexmora
NETWORK=t3-net
VOLUME_DIR=/cloud-admin/docker/lexmora

remote() {
  "${SSH_BASE[@]}" "$REMOTE" "$@"
}

scp_to() {
  local src="$1"
  local dest="$2"
  "${SCP_BASE[@]}" "$src" "${REMOTE}:${dest}"
}

case "$COMPONENT" in
  api)
    SUBDIR=lexmora-api
    IMAGE_TAG=lexmora-api:latest
    DOCKERFILE=dockerfile
    INTERNAL_PORT=8080
    PUBLISH_PORT=
    COMPOSE_FILES=(docker-compose.yml)
    if [[ -f "${GITHUB_WORKSPACE}/${SUBDIR}/docker-compose.openrouter-proxy.yml" ]]; then
      COMPOSE_FILES+=(docker-compose.openrouter-proxy.yml)
    fi
    ;;
  webui)
    SUBDIR=lexmora-webui
    IMAGE_TAG=lexmora-webui:latest
    DOCKERFILE=Dockerfile
    INTERNAL_PORT=80
    PUBLISH_PORT=
    API_HOST=lexmora-api
    API_PORT=8080
    COMPOSE_FILES=(docker-compose.yml)
    ;;
  *)
    echo "Unknown component: $COMPONENT" >&2
    exit 1
    ;;
esac

ROOT="${GITHUB_WORKSPACE}/${SUBDIR}"
cd "$ROOT"

echo ">> Building ${IMAGE_TAG} (context=${ROOT})"
docker build -f "$DOCKERFILE" -t "$IMAGE_TAG" .

TAR_NAME="${IMAGE_TAG//[:/]/_}.tar"
TAR_PATH="${RUNNER_TEMP}/${TAR_NAME}"
REMOTE_TAR="/tmp/${TAR_NAME}"

echo ">> Saving image to ${TAR_PATH}"
docker save -o "$TAR_PATH" "$IMAGE_TAG"

remote "mkdir -p '${VOLUME_DIR}'"

echo ">> Uploading image"
scp_to "$TAR_PATH" "$REMOTE_TAR"
rm -f "$TAR_PATH"

for f in "${COMPOSE_FILES[@]}"; do
  echo ">> Sync ${f}"
  scp_to "${ROOT}/${f}" "${VOLUME_DIR}/${f}"
done

COMPOSE_FILES_REMOTE=""
for f in "${COMPOSE_FILES[@]}"; do
  COMPOSE_FILES_REMOTE+=" -f '${VOLUME_DIR}/${f}'"
done

echo ">> Loading image on remote"
remote "docker load -i '${REMOTE_TAR}' && rm -f '${REMOTE_TAR}'"

if ! remote "docker network inspect '${NETWORK}' >/dev/null 2>&1"; then
  echo "ERR external network '${NETWORK}' missing — refusing to create (would risk VPN/HAProxy wiring)" >&2
  exit 1
fi

ENV_PREFIX="PUBLISH_PORT='${PUBLISH_PORT}' IMAGE_TAG='${IMAGE_TAG}' DOCKER_NETWORK='${NETWORK}' INTERNAL_PORT='${INTERNAL_PORT}'"
if [[ "$COMPONENT" == webui ]]; then
  ENV_PREFIX+=" API_HOST='${API_HOST}' API_PORT='${API_PORT}'"
fi

COMPOSE_CMD="${ENV_PREFIX} docker compose -p '${STACK_NAME}' ${COMPOSE_FILES_REMOTE} --project-directory '${VOLUME_DIR}'"

echo ">> docker compose up (${COMPONENT}, lexmora-* only)"
case "$COMPONENT" in
  api)
    # Keep Postgres running; refresh API image only (no stack-wide --force-recreate).
    remote "${COMPOSE_CMD} up -d --no-recreate lexmora-pgsql"
    remote "${COMPOSE_CMD} up -d --no-deps lexmora-api"
    ;;
  webui)
    remote "${COMPOSE_CMD} up -d --no-deps lexmora-webui"
    ;;
esac

echo "OK  ${COMPONENT} deployed at ${VOLUME_DIR} on ${REMOTE}"
