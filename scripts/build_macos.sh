#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_CMD="${GO_CMD:-go}"
VERSION="${VERSION:-dev}"
ARCH="${GOARCH:-$("$GO_CMD" env GOARCH)}"
DIST_DIR="$ROOT_DIR/dist"
APP_DIR="$DIST_DIR/DeguDesktop.app"
ZIP_PATH="$DIST_DIR/DeguDesktop-macos-$ARCH.zip"

case "$ARCH" in
  arm64|amd64) ;;
  *)
    echo "unsupported macOS arch: $ARCH" >&2
    exit 1
    ;;
esac

rm -rf "$APP_DIR"
mkdir -p "$APP_DIR/Contents/MacOS" "$APP_DIR/Contents/Resources"

python3 - "$ROOT_DIR/packaging/macos/Info.plist" "$APP_DIR/Contents/Info.plist" "$VERSION" <<'PY'
from pathlib import Path
import re
import sys

src, dst, version = sys.argv[1:]
safe = re.sub(r"[^0-9A-Za-z.+-]", "-", version).strip("-") or "dev"
text = Path(src).read_text(encoding="utf-8").replace("__VERSION__", safe)
Path(dst).write_text(text, encoding="utf-8")
PY

CGO_ENABLED=1 GOOS=darwin GOARCH="$ARCH" "$GO_CMD" build \
  -buildvcs=false \
  -ldflags="-s -w -X main.appVersion=$VERSION" \
  -o "$APP_DIR/Contents/MacOS/DeguDesktop" \
  ./cmd/degu

find "$APP_DIR" -name '._*' -delete
codesign --force --deep --sign - "$APP_DIR"
find "$APP_DIR" -name '._*' -delete
codesign --verify --deep --strict "$APP_DIR"
rm -f "$ZIP_PATH"
(cd "$DIST_DIR" && COPYFILE_DISABLE=1 zip -qry "$(basename "$ZIP_PATH")" DeguDesktop.app -x '*/._*' '*/.DS_Store')
echo "$ZIP_PATH"
