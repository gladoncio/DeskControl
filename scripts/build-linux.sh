#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../daemon"

VERSION=$(grep -E "^const Version" cmd/deskcontrol-ui/version.go | cut -d'"' -f2)
echo "==> Building DeskControl v$VERSION for Linux..."

go build -ldflags="-s -w" -o deskcontrol-daemon ./cmd/deskcontrol-ui

echo "==> Done! Binary: daemon/deskcontrol-daemon  (v$VERSION)"
ls -lh deskcontrol-daemon
