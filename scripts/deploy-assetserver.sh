#!/usr/bin/env bash
# Build, publish and roll out the Summit asset server, then evict its
# on-disk conversion cache so clients get assets produced by the new converters.
#
# Usage:
#   scripts/deploy-assetserver.sh              # build + push + deploy + evict cache
#   scripts/deploy-assetserver.sh build        # cross-compile + docker build only
#   scripts/deploy-assetserver.sh push         # build + push
#   scripts/deploy-assetserver.sh deploy       # apply manifests, roll out the tag, wait
#   scripts/deploy-assetserver.sh evict-cache  # wipe /data/cache in the running pod
#
# Environment:
#   KUBE_CONTEXT  kube context to deploy to        (default: talos-vm3)
#   IMAGE         image repository                 (default: ghcr.io/pilab-dev/summit-assetserver)
#   TAG           image tag                        (default: git sha, "-dirty" when the tree is modified)
#   PLATFORM      target platform                  (default: linux/amd64)
set -euo pipefail

cd "$(dirname "$0")/.."

KUBE_CONTEXT="${KUBE_CONTEXT:-talos-vm3}"
NAMESPACE="summit"
DEPLOYMENT="summit-assetserver"
CONTAINER="assetserver"
IMAGE="${IMAGE:-ghcr.io/pilab-dev/summit-assetserver}"
TAG="${TAG:-$(git describe --always --dirty | cut -c1-13)}"
PLATFORM="${PLATFORM:-linux/amd64}"
GOARCH="${PLATFORM#*/}"
MANIFESTS="deploy/k8s/assetserver"
CACHE_DIR="/data/cache"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

kctl() { kubectl --context "$KUBE_CONTEXT" -n "$NAMESPACE" "$@"; }

build() {
  log "Compiling assetserver for $PLATFORM"
  mkdir -p bin
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
    go build -a -tags netgo -ldflags "-s -w" -o bin/assetserver ./cmd/assetserver

  log "Building image $IMAGE:$TAG"
  docker build --platform "$PLATFORM" \
    -f build/package/Dockerfile.assetserver \
    -t "$IMAGE:$TAG" -t "$IMAGE:latest" .
}

push() {
  log "Pushing $IMAGE:$TAG and :latest"
  docker push "$IMAGE:$TAG"
  docker push "$IMAGE:latest"
}

pod_name() {
  kctl get pods -l "app.kubernetes.io/name=$DEPLOYMENT" \
    --field-selector=status.phase=Running \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

evict_cache() {
  local pod
  pod="$(pod_name)"
  if [[ -z "$pod" ]]; then
    log "No running $DEPLOYMENT pod; skipping cache eviction"
    return 0
  fi
  log "Evicting $CACHE_DIR in pod $pod"
  # Delete contents only: the directory itself is the mount's cache root
  kctl exec "$pod" -c "$CONTAINER" -- sh -c "rm -rf $CACHE_DIR/* $CACHE_DIR/.[!.]* 2>/dev/null; du -sh $CACHE_DIR"
}

deploy() {
  log "Deploying $IMAGE:$TAG to context $KUBE_CONTEXT"
  kubectl --context "$KUBE_CONTEXT" apply -f "$MANIFESTS"
  # Pin the tag so the rollout is a real change even though the manifest says :latest
  kctl set image "deployment/$DEPLOYMENT" "$CONTAINER=$IMAGE:$TAG"
  kctl rollout status "deployment/$DEPLOYMENT" --timeout=300s
  kctl get pods -l "app.kubernetes.io/name=$DEPLOYMENT"
}

case "${1:-all}" in
  build)       build ;;
  push)        build; push ;;
  deploy)      deploy ;;
  evict-cache) evict_cache ;;
  all)
    build
    push
    # Drop stale files before the switch so the old pod cannot hand out
    # old-format assets during the rollout, and again after so nothing the
    # old converter produced in between survives.
    evict_cache
    deploy
    evict_cache
    ;;
  *)
    echo "unknown command: $1" >&2
    exit 2
    ;;
esac
