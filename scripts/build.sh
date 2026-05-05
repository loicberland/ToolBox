#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/_build"

mkdir -p "$BUILD_DIR"

show_help() {
  echo "Usage: ./scripts/build.sh [api|web|web-server|module <name>|modules|all|help]"
}

build_api() {
  echo "Building api..."
  (cd "$ROOT_DIR" && CGO_ENABLED=1 go build -o "$BUILD_DIR/api-toolbox.exe" ./apps/api/cmd/api)
}

build_web() {
  echo "Building web..."
  (cd "$ROOT_DIR/apps/web" && npm run build)
}

sync_web_dist() {
  echo "Embedding web dist..."
  rm -rf "$ROOT_DIR/apps/web-server/cmd/dist"
  mkdir -p "$ROOT_DIR/apps/web-server/cmd/dist"
  cp -R "$ROOT_DIR/apps/web/dist/." "$ROOT_DIR/apps/web-server/cmd/dist/"
}

build_web_server() {
  echo "Building web-server..."
  build_web
  sync_web_dist
  (cd "$ROOT_DIR" && go build -o "$BUILD_DIR/web-server-toolbox.exe" ./apps/web-server/cmd/web-server)
}

build_module() {
  local name="$1"
  echo "Building module $name..."
  (cd "$ROOT_DIR" && CGO_ENABLED=1 go build -o "$BUILD_DIR/$name.exe" "./modules/$name/cmd/$name")
}

build_modules() {
  build_module "test-sheet"
  build_module "test-env"
}

case "${1:-help}" in
  api)
    build_api
    ;;
  web)
    build_web
    ;;
  web-server)
    build_web_server
    ;;
  module)
    if [[ $# -lt 2 ]]; then
      echo "Missing module name"
      show_help
      exit 1
    fi
    build_module "$2"
    ;;
  modules)
    build_modules
    ;;
  all)
    build_api
    build_web_server
    build_modules
    ;;
  help|*)
    show_help
    ;;
esac
