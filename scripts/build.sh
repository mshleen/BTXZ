#!/usr/bin/env bash
set -euo pipefail

# Usage: scripts/build.sh <version>
VERSION="$1"
ARTIFACTS="artifacts"
PLATFORMS=(windows/amd64 linux/amd64 darwin/amd64 darwin/arm64)

if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "▶ Building BTXZ v${VERSION}"
rm -rf "${ARTIFACTS}"
mkdir -p "${ARTIFACTS}"

# Move into the btxz/ dir where go.mod lives
# "$(dirname "$0")" is the scripts/ folder, so ../btxz is the module root
cd "$(dirname "$0")/../btxz"

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"

  BIN="../${ARTIFACTS}/btxz-${GOOS}-${GOARCH}"
  [[ "$GOOS" == windows ]] && BIN+=".exe"

  echo "  • $GOOS/$GOARCH → $(basename "$BIN")"
  env GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -v -o "$BIN" .
done

echo "✅ Done — artifacts in ${ARTIFACTS}/"
