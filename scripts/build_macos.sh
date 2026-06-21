#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_CMD="${GO_CMD:-go}"
VERSION="${VERSION:-dev}"
ARCH="${GOARCH:-$("$GO_CMD" env GOARCH)}"
HOST_GOOS="$("$GO_CMD" env GOHOSTOS)"
HOST_GOARCH="$("$GO_CMD" env GOHOSTARCH)"
MACOS_MIN_VERSION="${MACOS_MIN_VERSION:-12.0}"
MACOS_COMPAT_LABEL="${MACOS_COMPAT_LABEL:-}"
DIST_DIR="$ROOT_DIR/dist"
APP_DIR="$DIST_DIR/DeguDesktop.app"
ZIP_BASENAME="DeguDesktop-macos-$ARCH.zip"
if [[ -n "$MACOS_COMPAT_LABEL" ]]; then
  ZIP_BASENAME="DeguDesktop-macos-$MACOS_COMPAT_LABEL-$ARCH.zip"
fi
ZIP_PATH="$DIST_DIR/$ZIP_BASENAME"

case "$ARCH" in
  arm64|amd64) ;;
  *)
    echo "unsupported macOS arch: $ARCH" >&2
    exit 1
    ;;
esac

case "$MACOS_MIN_VERSION" in
  11.0|12.0|13.0|14.0|15.0|16.0|17.0|18.0|19.0|20.0|21.0|22.0|23.0|24.0|25.0|26.0) ;;
  *)
    echo "unsupported MACOS_MIN_VERSION: $MACOS_MIN_VERSION" >&2
    exit 1
    ;;
esac

export MACOSX_DEPLOYMENT_TARGET="$MACOS_MIN_VERSION"
export CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=$MACOS_MIN_VERSION"
export CGO_CXXFLAGS="${CGO_CXXFLAGS:-} -mmacosx-version-min=$MACOS_MIN_VERSION"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=$MACOS_MIN_VERSION"

rm -rf "$APP_DIR"
mkdir -p "$APP_DIR/Contents/MacOS" "$APP_DIR/Contents/Resources"

python3 - "$ROOT_DIR/packaging/macos/Info.plist" "$APP_DIR/Contents/Info.plist" "$VERSION" "$MACOS_MIN_VERSION" <<'PY'
from pathlib import Path
import re
import sys

src, dst, version, min_macos = sys.argv[1:]
safe = re.sub(r"[^0-9A-Za-z.+-]", "-", version).strip("-") or "dev"
text = (
    Path(src)
    .read_text(encoding="utf-8")
    .replace("__VERSION__", safe)
    .replace("__MIN_MACOS_VERSION__", min_macos)
)
Path(dst).write_text(text, encoding="utf-8")
PY

ICON_TMP="$(mktemp -d)"
trap 'rm -rf "$ICON_TMP"' EXIT
ICONSET="$ICON_TMP/DeguDesktop.iconset"
mkdir -p "$ICONSET"
cat > "$ICON_TMP/make_icon.go" <<'GO'
package main

import (
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

const (
	frameW     = 96
	frameH     = 64
	iconFrames = 56
)

var iconFiles = map[string]int{
	"icon_16x16.png":     16,
	"icon_16x16@2x.png":  32,
	"icon_32x32.png":     32,
	"icon_32x32@2x.png":  64,
	"icon_128x128.png":   128,
	"icon_128x128@2x.png": 256,
	"icon_256x256.png":   256,
	"icon_256x256@2x.png": 512,
	"icon_512x512.png":   512,
	"icon_512x512@2x.png": 1024,
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: make_icon <sprite-sheet> <iconset-dir>")
	}
	src := readPNG(os.Args[1])
	if src.Bounds().Dx() < frameW*iconFrames || src.Bounds().Dy() < frameH {
		log.Fatalf("unexpected sprite sheet size: %v", src.Bounds())
	}
	frame := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	draw.Draw(frame, frame.Bounds(), src, image.Point{}, draw.Src)
	content := cropVisible(frame)
	for name, size := range iconFiles {
		writePNG(filepath.Join(os.Args[2], name), fitIcon(content, size))
	}
}

func readPNG(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func cropVisible(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.RGBAAt(x, y).A <= 8 {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+1 > maxX {
				maxX = x + 1
			}
			if y+1 > maxY {
				maxY = y + 1
			}
		}
	}
	if minX >= maxX || minY >= maxY {
		log.Fatal("sprite frame has no visible content")
	}
	r := image.Rect(max(b.Min.X, minX-2), max(b.Min.Y, minY-2), min(b.Max.X, maxX+2), min(b.Max.Y, maxY+2))
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
	return dst
}

func fitIcon(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	padding := int(math.Round(float64(size) * 0.14))
	limit := size - padding*2
	if limit < 1 {
		limit = size
	}
	sb := src.Bounds()
	scale := math.Min(float64(limit)/float64(sb.Dx()), float64(limit)/float64(sb.Dy()))
	w := max(1, int(math.Round(float64(sb.Dx())*scale)))
	h := max(1, int(math.Round(float64(sb.Dy())*scale)))
	scaled := scaleNearest(src, w, h)
	x := (size - w) / 2
	y := (size - h) / 2
	draw.Draw(dst, image.Rect(x, y, x+w, y+h), scaled, image.Point{}, draw.Over)
	return dst
}

func scaleNearest(src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sb := src.Bounds()
	for y := 0; y < height; y++ {
		sy := sb.Min.Y + y*sb.Dy()/height
		for x := 0; x < width; x++ {
			sx := sb.Min.X + x*sb.Dx()/width
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
GO
GOOS="$HOST_GOOS" GOARCH="$HOST_GOARCH" CGO_ENABLED=0 "$GO_CMD" run "$ICON_TMP/make_icon.go" "$ROOT_DIR/assets/sprites/degu_wild_agouti_set00.png" "$ICONSET"
iconutil -c icns "$ICONSET" -o "$APP_DIR/Contents/Resources/DeguDesktop.icns"

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
