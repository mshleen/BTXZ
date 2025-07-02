#!/usr/bin/env bash
set -euo pipefail

VERSION="$1"
SOURCE_DIR="btxz"
OUTPUT_DIR="artifacts"
PLATFORMS=("windows/amd64" "linux/amd64" "darwin/amd64" "darwin/arm64")

if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "▶ Building BTXZ v${VERSION} for: ${PLATFORMS[*]}"
rm -rf "${OUTPUT_DIR}" && mkdir -p "${OUTPUT_DIR}"

# Put -ldflags and its argument in a Bash array to avoid quoting hell
LDFLAGS=( "-ldflags=-X" "main.version=${VERSION}" )

for platform in "${PLATFORMS[@]}"; do
  GOOS=${platform%%/*}
  GOARCH=${platform#*/}
  output="btxz-${GOOS}-${GOARCH}"
  [ "$GOOS" = "windows" ] && output+=".exe"

  echo "  • ${GOOS}/${GOARCH} → ${output}"
  env GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -v \
      "${LDFLAGS[@]}" \
      -o "${OUTPUT_DIR}/${output}" \
      "./${SOURCE_DIR}"
done

echo "✅ Build succeeded; artifacts in ${OUTPUT_DIR}/"
