#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../daemon"

VERSION=$(grep -E "^const Version" cmd/deskcontrol-ui/version.go | cut -d'"' -f2)
echo "==> Building DeskControl v$VERSION for Windows (cross-compile via fyne-cross)..."

# Ensure fyne-cross is installed
PATH="$HOME/go/bin:$PATH"
if ! command -v fyne-cross &>/dev/null; then
    echo "fyne-cross not found. Installing..."
    go install github.com/fyne-io/fyne-cross@latest
fi

fyne-cross windows \
    -app-id com.deskcontrol.daemon \
    -name "DeskControl $VERSION" \
    -icon cmd/deskcontrol-ui/assets/tray.png \
    -output DeskControl.exe \
    -env GOTOOLCHAIN=auto \
    ./cmd/deskcontrol-ui

mkdir -p ../build/windows
cp fyne-cross/dist/windows-amd64/DeskControl.exe.zip ../build/windows/
unzip -o ../build/windows/DeskControl.exe.zip -d ../build/windows/ 2>&1 | tail -2
rm ../build/windows/DeskControl.exe.zip

echo "==> Done! Binary: build/windows/DeskControl.exe  (v$VERSION)"
ls -lh ../build/windows/DeskControl.exe
