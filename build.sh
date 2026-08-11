#!/bin/sh
# Build the universal (arm64 + amd64) koreansearch binary, ad-hoc sign it, and
# place it in workflow/ ready for packaging.
#
# Ad-hoc signing (codesign -s -) is REQUIRED: unsigned Mach-O binaries are
# killed on Apple Silicon. Notarization is intentionally not used; the workflow's
# `run` shim strips the download quarantine at first launch instead.
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
PKG="./cmd/koreansearch"
OUT="$ROOT/workflow/koreansearch"
DIST="$ROOT/dist"
LDFLAGS="-s -w"

mkdir -p "$DIST"

echo "==> building arm64"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/koreansearch-arm64" "$PKG"

echo "==> building amd64"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/koreansearch-amd64" "$PKG"

echo "==> lipo -> universal"
lipo -create -output "$OUT" "$DIST/koreansearch-arm64" "$DIST/koreansearch-amd64"

echo "==> codesign (ad-hoc)"
codesign --force --sign - "$OUT"
codesign --verify --verbose "$OUT"

lipo -info "$OUT"
echo "==> done: $OUT"
