#!/usr/bin/env bash
# Build, publish and roll out the Summit game server and web client to Kubernetes.
#
# Usage:
#   scripts/deploy-summit.sh            # build + push + deploy both
#   scripts/deploy-summit.sh build      # images only
#   scripts/deploy-summit.sh push       # build + push
#   scripts/deploy-summit.sh deploy     # apply manifests, roll out the tag, wait
#
# Environment:
#   KUBE_CONTEXT  kube context (default: talos-vm3)
#   REGISTRY      image repository prefix (default: ghcr.io/pilab-dev)
#   TAG           image tag (default: git sha, "-dirty" when modified)
#   PLATFORM      target platform (default: linux/amd64)
#   ASSET_HOST    asset server the client is built against (default: https://assets-summit.dev.pilab.hu)
set -euo pipefail

cd "$(dirname "$0")/.."

KUBE_CONTEXT="${KUBE_CONTEXT:-talos-vm3}"
NAMESPACE="summit"
REGISTRY="${REGISTRY:-ghcr.io/pilab-dev}"
TAG="${TAG:-$(git describe --always --dirty | cut -c1-13)}"
PLATFORM="${PLATFORM:-linux/amd64}"
GOARCH="${PLATFORM#*/}"
ASSET_HOST="${ASSET_HOST:-https://assets-summit.dev.pilab.hu}"
MANIFESTS="deploy/k8s/summit"

SERVER_IMAGE="$REGISTRY/summit-server"
CLIENT_IMAGE="$REGISTRY/summit-client"

# Atlas URI for the summit-mongodb secret. Read from PROD_MONGO_URI (env or
# .env) and normalise the common export typos ("srv+mongodb://", "@>@").
if [[ -z "${PROD_MONGO_URI:-}" && -f .env ]]; then
  PROD_MONGO_URI="$(grep -E '^PROD_MONGO_URI=' .env | head -1 | cut -d= -f2-)"
fi
if [[ -n "${PROD_MONGO_URI:-}" ]]; then
  PROD_MONGO_URI="${PROD_MONGO_URI/srv+mongodb:\/\//mongodb+srv:\/\/}"
  PROD_MONGO_URI="${PROD_MONGO_URI/@>@/@}"
fi

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
kctl() { kubectl --context "$KUBE_CONTEXT" -n "$NAMESPACE" "$@"; }

build() {
  log "Compiling summit for $PLATFORM"
  mkdir -p bin
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
    go build -a -tags netgo -ldflags "-s -w" -o bin/summit ./cmd/summit

  log "Building $SERVER_IMAGE:$TAG"
  docker build --platform "$PLATFORM" -f build/package/Dockerfile.summit \
    -t "$SERVER_IMAGE:$TAG" -t "$SERVER_IMAGE:latest" .

  log "Building $CLIENT_IMAGE:$TAG (assets from $ASSET_HOST)"
  docker build --platform "$PLATFORM" -f build/package/Dockerfile.client \
    --build-arg "VITE_ASSET_HOST=$ASSET_HOST" \
    -t "$CLIENT_IMAGE:$TAG" -t "$CLIENT_IMAGE:latest" .
}

push() {
  for img in "$SERVER_IMAGE" "$CLIENT_IMAGE"; do
    log "Pushing $img:$TAG and :latest"
    docker push "$img:$TAG"
    docker push "$img:latest"
  done
}

deploy() {
  log "Deploying tag $TAG to context $KUBE_CONTEXT"
  if ! kctl get secret summit-management >/dev/null 2>&1; then
    echo "secret summit-management is missing; see $MANIFESTS/secret.example.yaml" >&2
    exit 1
  fi
  if ! kctl get secret summit-mongodb >/dev/null 2>&1; then
    if [[ -n "${PROD_MONGO_URI:-}" ]]; then
      log "Creating secret summit-mongodb from PROD_MONGO_URI"
      kctl create secret generic summit-mongodb --from-literal=uri="$PROD_MONGO_URI" >/dev/null
    else
      echo "secret summit-mongodb is missing; set PROD_MONGO_URI or create it (see $MANIFESTS/secret-mongodb.example.yaml)" >&2
      exit 1
    fi
  fi
  for f in "$MANIFESTS"/*.yaml; do
    [[ "$f" == *.example.yaml ]] && continue
    kubectl --context "$KUBE_CONTEXT" apply -f "$f"
  done
  # Pin the tag so the rollout is a real change even though the manifests say :latest
  kctl set image deployment/summit-server "summit=$SERVER_IMAGE:$TAG"
  kctl set image deployment/summit-client "nginx=$CLIENT_IMAGE:$TAG"
  kctl rollout status deployment/summit-server --timeout=300s
  kctl rollout status deployment/summit-client --timeout=300s
  kctl get pods -l app.kubernetes.io/part-of=summit
}

case "${1:-all}" in
  build)  build ;;
  push)   build; push ;;
  deploy) deploy ;;
  all)    build; push; deploy ;;
  *) echo "unknown command: $1" >&2; exit 2 ;;
esac
