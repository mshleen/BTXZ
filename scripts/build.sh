#!/bin/bash

# set -e: Exit immediately if a command fails.
# set -x: Print each command before it is executed, for excellent debugging.
set -e
set -x

# --- Parameters ---
VERSION="$1"
SOURCE_DIR="btxz"
OUTPUT_DIR="artifacts"
PLATFORMS=("windows/amd64" "linux/amd64" "darwin/amd64" "darwin/arm64")

# --- Validation ---
if [ -z "$VERSION" ]; then
  echo "Error: No version number provided."
  exit 1
fi

# --- Build Process ---
echo "Starting build process for version v${VERSION}..."
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Define build flags using a Bash Array. This is the most robust method.
LDFLAGS_ARRAY=(
  "-ldflags=-X" 
  "main.version=${VERSION}"
)

# --- Main Build Loop ---
for platform in "${PLATFORMS[@]}"; do
  GOOS=$(echo "$platform" | cut -d'/' -f1)
  GOARCH=$(echo "$platform" | cut -d'/' -f2)

  output_name="btxz-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    output_name+='.exe'
  fi

  echo "--> Building for ${GOOS}/${GOARCH}..."
  
  # Execute the build command using the array for flags.
  # This prevents all quoting and word-splitting issues.
  env GOOS="$GOOS" GOARCH="$GOARCH" go build \
    -v \
    "${LDFLAGS_ARRAY[@]}" \
    -o "${OUTPUT_DIR}/${output_name}" \
    "./${SOURCE_DIR}"
done

echo "✅ Build process completed successfully."
