#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../daemon"

VERSION=$(grep -E "^const Version" cmd/deskcontrol-ui/version.go | cut -d'"' -f2)
echo "==> Building DeskControl v$VERSION for Linux..."

mkdir -p ../build/linux
go build -ldflags="-s -w" -o ../build/linux/deskcontrol-daemon ./cmd/deskcontrol-ui

echo "==> Done! Binary: build/linux/deskcontrol-daemon  (v$VERSION)"
ls -lh ../build/linux/deskcontrol-daemon
