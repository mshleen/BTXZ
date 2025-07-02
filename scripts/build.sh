#!/usr/bin/env bash
set -euo pipefail

VERSION="$1"
SRC="btxz"
OUT="artifacts"

if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "▶ Building BTXZ v${VERSION}"
rm -rf "$OUT" && mkdir -p "$OUT"

platforms=( windows/amd64 linux/amd64 darwin/amd64 darwin/arm64 )
for p in "${platforms[@]}"; do
  GOOS=${p%%/*} GOARCH=${p#*/}

  bin="btxz-${GOOS}-${GOARCH}"
  [[ $GOOS = windows ]] && bin+=".exe"

  echo "  • $GOOS/$GOARCH → $bin"
  env GOOS=$GOOS GOARCH=$GOARCH \
    go build -v -o "$OUT/$bin" "./$SRC"
done

echo "✅ Build complete; artifacts in $OUT/"
