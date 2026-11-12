#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"

# 可选：仓库自带的 Go 工具链；没有就用系统 go
if [ -x "$ROOT/toolchain/go/bin/go" ]; then
    export PATH="$ROOT/toolchain/go/bin:$PATH"
    export GOPATH="$ROOT/toolchain/gopath"
fi

export CGO_ENABLED=0
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

cd "$ROOT/src"
gofmt -l internal/app
go build -ldflags="-s -w" -trimpath -o gemini-web2api .
docker build -f Dockerfile.local -t gemini-web2api:local .

cd "$ROOT"
docker compose up -d --build
