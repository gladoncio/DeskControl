#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

echo "╔══════════════════════════════════════════════════╗"
echo "║   DeskControl — Build all targets               ║"
echo "╚══════════════════════════════════════════════════╝"

echo ""
echo "▸ Linux daemon…"
./scripts/build-linux.sh

echo ""
echo "▸ Windows daemon (cross)…"
./scripts/build-windows.sh

echo ""
echo "▸ Android APK…"
./scripts/build-apk.sh

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║   All builds complete!                           ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""
ls -lh build/*/
