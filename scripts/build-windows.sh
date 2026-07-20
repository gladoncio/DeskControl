#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../daemon"

VERSION=$(grep -E "^const Version" cmd/deskcontrol-ui/version.go | cut -d'"' -f2)
echo "==> Building DeskControl v$VERSION for Windows (cross-compile via fyne-cross)..."

# Ensure fyne-cross is installed
if ! command -v fyne-cross &>/dev/null; then
    echo "fyne-cross not found. Installing..."
    go install github.com/fyne-io/fyne-cross@latest
fi

fyne-cross windows \
    -app-id com.deskcontrol.daemon \
    -name "DeskControl $VERSION" \
    -icon cmd/deskcontrol-ui/assets/tray.png \
    -output DeskControl.exe \
    ./cmd/deskcontrol-ui

echo "==> Done! Package: daemon/fyne-cross/dist/windows-amd64/DeskControl.exe.zip  (v$VERSION)"
ls -lh fyne-cross/dist/windows-amd64/DeskControl.exe.zip
