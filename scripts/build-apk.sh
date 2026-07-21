#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../app"

VERSION=$(grep "^version:" pubspec.yaml | awk '{print $2}' | cut -d+ -f1)
echo "==> Building DeskControl v$VERSION APK for Android..."

flutter build apk --release

mkdir -p ../build/android
cp build/app/outputs/flutter-apk/app-release.apk ../build/android/

echo "==> Done! APK: build/android/app-release.apk  (v$VERSION)"
ls -lh ../build/android/app-release.apk
