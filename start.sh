#!/usr/bin/env sh
set -eu

BINARY="${BINARY:-./build/mcp-database}"
CONFIG_PATH="${CONFIG_PATH:-config.yaml}"

case "${1:-run}" in
  build)
    mkdir -p ./build
    go build -o "$BINARY" ./cmd
    ;;
  test)
    go test ./...
    ;;
  run)
    if [ ! -x "$BINARY" ]; then
      mkdir -p ./build
      go build -o "$BINARY" ./cmd
    fi
    exec "$BINARY" -config "$CONFIG_PATH"
    ;;
  *)
    echo "usage: $0 [build|test|run]" >&2
    echo "Set CONFIG_PATH to the YAML configuration path (default: config.yaml)." >&2
    exit 2
    ;;
esac
