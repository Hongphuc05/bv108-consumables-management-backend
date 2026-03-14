#!/usr/bin/env bash

set -euo pipefail

require_bin() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

resolve_az_bin() {
  local candidate

  if command -v az >/dev/null 2>&1; then
    command -v az
    return 0
  fi

  if command -v az.cmd >/dev/null 2>&1; then
    command -v az.cmd
    return 0
  fi

  for candidate in \
    "/c/Program Files/Microsoft SDKs/Azure/CLI2/wbin/az" \
    "/c/Program Files (x86)/Microsoft SDKs/Azure/CLI2/wbin/az" \
    "/mnt/c/Program Files/Microsoft SDKs/Azure/CLI2/wbin/az" \
    "/mnt/c/Program Files (x86)/Microsoft SDKs/Azure/CLI2/wbin/az" \
    "/c/Program Files/Microsoft SDKs/Azure/CLI2/wbin/az.cmd" \
    "/c/Program Files (x86)/Microsoft SDKs/Azure/CLI2/wbin/az.cmd" \
    "/mnt/c/Program Files/Microsoft SDKs/Azure/CLI2/wbin/az.cmd" \
    "/mnt/c/Program Files (x86)/Microsoft SDKs/Azure/CLI2/wbin/az.cmd"; do
    if [[ -f "$candidate" ]]; then
      echo "$candidate"
      return 0
    fi
  done

  return 1
}

AZ_BIN="$(resolve_az_bin || true)"
if [[ -z "${AZ_BIN:-}" ]]; then
  echo "Missing Azure CLI. Install Azure CLI, then re-run this script." >&2
  exit 1
fi

az_cli() {
  if [[ "$AZ_BIN" == *.cmd ]]; then
    local az_win_path="$AZ_BIN"
    if command -v cygpath >/dev/null 2>&1; then
      az_win_path="$(cygpath -w "$AZ_BIN")"
    elif command -v wslpath >/dev/null 2>&1; then
      az_win_path="$(wslpath -w "$AZ_BIN")"
    fi
    cmd.exe /c "$az_win_path" "$@"
    return $?
  fi

  "$AZ_BIN" "$@"
}

to_az_path() {
  local p="$1"
  if [[ "$AZ_BIN" != *.cmd ]]; then
    echo "$p"
    return 0
  fi

  if command -v cygpath >/dev/null 2>&1; then
    cygpath -w "$p"
    return 0
  fi

  if command -v wslpath >/dev/null 2>&1; then
    wslpath -w "$p"
    return 0
  fi

  echo "$p"
}

if ! az_cli account show >/dev/null 2>&1; then
  echo "Please run 'az login' first." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

load_env_file() {
  local file="$1"
  [[ -f "$file" ]] || return 0
  while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
    local line="${raw_line%$'\r'}"
    line="${line#"${line%%[![:space:]]*}"}"
    [[ -z "$line" || "${line:0:1}" == "#" ]] && continue
    export "$line"
  done < "$file"
}

ENV_FILE="${ENV_FILE:-$BACKEND_DIR/.env}"
load_env_file "$ENV_FILE"

if [[ -z "${FRONTEND_DIR:-}" ]]; then
  FRONTEND_DIR="$(cd "$BACKEND_DIR/../bv108-consumables-management" 2>/dev/null && pwd || true)"
fi

if [[ -z "$FRONTEND_DIR" || ! -f "$FRONTEND_DIR/package.json" ]]; then
  echo "Frontend directory not found. Set FRONTEND_DIR=<path-to-bv108-consumables-management>." >&2
  exit 1
fi

FRONTEND_ENV_FILE="${FRONTEND_ENV_FILE:-$FRONTEND_DIR/.env}"
if [[ -f "$FRONTEND_ENV_FILE" ]]; then
  while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
    line="${raw_line%$'\r'}"
    line="${line#"${line%%[![:space:]]*}"}"
    [[ -z "$line" || "${line:0:1}" == "#" ]] && continue

    key="${line%%=*}"
    value="${line#*=}"
    case "$key" in
      VITE_GEMINI_API_KEY|VITE_GEMINI_MODEL|VITE_GEMINI_WEB_SEARCH)
        if [[ -z "${!key:-}" ]]; then
          export "${key}=${value}"
        fi
        ;;
    esac
  done < "$FRONTEND_ENV_FILE"
fi

: "${DB_HOST:?DB_HOST is required}"
: "${DB_USER:?DB_USER is required}"
: "${DB_PASSWORD:?DB_PASSWORD is required}"
: "${DB_NAME:?DB_NAME is required}"

RG="${RG:-bv108-rg}"
LOCATION="${LOCATION:-southeastasia}"
ACA_ENV_NAME="${ACA_ENV_NAME:-bv108-aca-env}"
BACKEND_APP_NAME="${BACKEND_APP_NAME:-bv108-backend}"
FRONTEND_APP_NAME="${FRONTEND_APP_NAME:-bv108-frontend}"
DB_PORT="${DB_PORT:-3306}"
DB_TLS="${DB_TLS:-true}"
GIN_MODE="${GIN_MODE:-release}"
JWT_SECRET="${JWT_SECRET:-change-this-secret-in-production}"

if [[ -z "${ACR_NAME:-}" ]]; then
  ACR_NAME="bv108acr$(date +%s | cut -c 5-10)"
fi

STAMP="$(date +%Y%m%d%H%M%S)"
BACKEND_IMAGE="${ACR_NAME}.azurecr.io/${BACKEND_APP_NAME}:${STAMP}"
FRONTEND_IMAGE="${ACR_NAME}.azurecr.io/${FRONTEND_APP_NAME}:${STAMP}"
BACKEND_DIR_FOR_AZ="$(to_az_path "$BACKEND_DIR")"
FRONTEND_DIR_FOR_AZ="$(to_az_path "$FRONTEND_DIR")"
BUILD_MODE="${BUILD_MODE:-docker}" # docker | acr_tasks

if [[ "$BUILD_MODE" == "docker" ]]; then
  require_bin docker
  if ! docker version >/dev/null 2>&1; then
    echo "Docker engine is not running. Start Docker Desktop, then run this script again." >&2
    exit 1
  fi
fi

echo "==> Ensure required Azure extension"
az_cli extension add --name containerapp --upgrade --yes >/dev/null

echo "==> Create resource group"
az_cli group create --name "$RG" --location "$LOCATION" >/dev/null

echo "==> Create ACR (if not exists)"
if ! az_cli acr show --name "$ACR_NAME" --resource-group "$RG" >/dev/null 2>&1; then
  az_cli acr create --name "$ACR_NAME" --resource-group "$RG" --sku Basic --admin-enabled true >/dev/null
fi

echo "==> Create Container Apps environment (if not exists)"
if ! az_cli containerapp env show --name "$ACA_ENV_NAME" --resource-group "$RG" >/dev/null 2>&1; then
  az_cli containerapp env create --name "$ACA_ENV_NAME" --resource-group "$RG" --location "$LOCATION" >/dev/null
fi

if az_cli containerapp show --name "$BACKEND_APP_NAME" --resource-group "$RG" >/dev/null 2>&1; then
  echo "Backend app '$BACKEND_APP_NAME' already exists. Use a new BACKEND_APP_NAME or remove existing app first." >&2
  exit 1
fi

if az_cli containerapp show --name "$FRONTEND_APP_NAME" --resource-group "$RG" >/dev/null 2>&1; then
  echo "Frontend app '$FRONTEND_APP_NAME' already exists. Use a new FRONTEND_APP_NAME or remove existing app first." >&2
  exit 1
fi

echo "==> Get ACR credentials"
ACR_USERNAME="$(az_cli acr credential show --name "$ACR_NAME" --query username --output tsv | tr -d '\r')"
ACR_PASSWORD="$(az_cli acr credential show --name "$ACR_NAME" --query passwords[0].value --output tsv | tr -d '\r')"

echo "==> Build and push backend image"
if [[ "$BUILD_MODE" == "docker" ]]; then
  docker login "${ACR_NAME}.azurecr.io" --username "$ACR_USERNAME" --password "$ACR_PASSWORD" >/dev/null
  docker build -t "$BACKEND_IMAGE" "$BACKEND_DIR"
  docker push "$BACKEND_IMAGE"
else
  az_cli acr build \
    --registry "$ACR_NAME" \
    --image "${BACKEND_APP_NAME}:${STAMP}" \
    "$BACKEND_DIR_FOR_AZ"
fi

echo "==> Deploy backend container app"
az_cli containerapp create \
  --name "$BACKEND_APP_NAME" \
  --resource-group "$RG" \
  --environment "$ACA_ENV_NAME" \
  --image "$BACKEND_IMAGE" \
  --ingress external \
  --target-port 8080 \
  --min-replicas 0 \
  --max-replicas 1 \
  --registry-server "${ACR_NAME}.azurecr.io" \
  --registry-username "$ACR_USERNAME" \
  --registry-password "$ACR_PASSWORD" \
  --secrets db-password="$DB_PASSWORD" \
  --env-vars \
    DB_HOST="$DB_HOST" \
    DB_PORT="$DB_PORT" \
    DB_USER="$DB_USER" \
    DB_PASSWORD=secretref:db-password \
    DB_NAME="$DB_NAME" \
    DB_TLS="$DB_TLS" \
    GIN_MODE="$GIN_MODE" \
    JWT_SECRET="$JWT_SECRET" \
    FRONTEND_URL="http://localhost" \
  >/dev/null

BACKEND_FQDN="$(az_cli containerapp show --name "$BACKEND_APP_NAME" --resource-group "$RG" --query properties.configuration.ingress.fqdn --output tsv | tr -d '\r')"
BACKEND_URL="https://${BACKEND_FQDN}"

if [[ -z "${VITE_API_URL:-}" ]]; then
  VITE_API_URL="${BACKEND_URL}/api"
fi

echo "==> Build and push frontend image"
FRONTEND_BUILD_ARGS=(--build-arg "VITE_API_URL=${VITE_API_URL}")
if [[ -n "${VITE_GEMINI_API_KEY:-}" ]]; then
  FRONTEND_BUILD_ARGS+=(--build-arg "VITE_GEMINI_API_KEY=${VITE_GEMINI_API_KEY}")
fi
if [[ -n "${VITE_GEMINI_MODEL:-}" ]]; then
  FRONTEND_BUILD_ARGS+=(--build-arg "VITE_GEMINI_MODEL=${VITE_GEMINI_MODEL}")
fi
if [[ -n "${VITE_GEMINI_WEB_SEARCH:-}" ]]; then
  FRONTEND_BUILD_ARGS+=(--build-arg "VITE_GEMINI_WEB_SEARCH=${VITE_GEMINI_WEB_SEARCH}")
fi

if [[ "$BUILD_MODE" == "docker" ]]; then
  docker build -t "$FRONTEND_IMAGE" "${FRONTEND_BUILD_ARGS[@]}" "$FRONTEND_DIR"
  docker push "$FRONTEND_IMAGE"
else
  az_cli acr build \
    --registry "$ACR_NAME" \
    --image "${FRONTEND_APP_NAME}:${STAMP}" \
    "${FRONTEND_BUILD_ARGS[@]}" \
    "$FRONTEND_DIR_FOR_AZ"
fi

echo "==> Deploy frontend container app"
az_cli containerapp create \
  --name "$FRONTEND_APP_NAME" \
  --resource-group "$RG" \
  --environment "$ACA_ENV_NAME" \
  --image "$FRONTEND_IMAGE" \
  --ingress external \
  --target-port 8080 \
  --min-replicas 0 \
  --max-replicas 1 \
  --registry-server "${ACR_NAME}.azurecr.io" \
  --registry-username "$ACR_USERNAME" \
  --registry-password "$ACR_PASSWORD" \
  >/dev/null

FRONTEND_FQDN="$(az_cli containerapp show --name "$FRONTEND_APP_NAME" --resource-group "$RG" --query properties.configuration.ingress.fqdn --output tsv | tr -d '\r')"
FRONTEND_URL="https://${FRONTEND_FQDN}"

echo "==> Update backend CORS origin with frontend URL"
az_cli containerapp update \
  --name "$BACKEND_APP_NAME" \
  --resource-group "$RG" \
  --set-env-vars FRONTEND_URL="$FRONTEND_URL" \
  >/dev/null

echo
echo "Deployment completed."
echo "Backend URL : $BACKEND_URL"
echo "Frontend URL: $FRONTEND_URL"
echo "API URL     : ${VITE_API_URL}"
