#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
# The version number is passed as the first argument to the script
VERSION="$1"
# The path to your Go project's main package, relative to the repository root
SOURCE_DIR="btxz"
# The directory where build artifacts will be stored
OUTPUT_DIR="artifacts"
# List of platforms to build for (format: "os/arch")
PLATFORMS=("windows/amd64" "linux/amd64" "darwin/amd64" "darwin/arm64")

# --- Validation ---
if [ -z "$VERSION" ]; then
  echo "Error: No version number provided."
  echo "Usage: ./scripts/build.sh <version>"
  exit 1
fi

# --- Build Process ---
echo "Starting build process for version v${VERSION}..."

# Clean up previous builds and create the output directory
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Set the linker flags to embed the version number in the binary.
# The quotes are crucial to ensure this is treated as a single argument.
LDFLAGS="-ldflags=-X 'main.version=${VERSION}'"

# Loop through each platform and build the binary
for platform in "${PLATFORMS[@]}"; do
  # Split the platform string into OS and architecture
  GOOS=$(echo "$platform" | cut -d'/' -f1)
  GOARCH=$(echo "$platform" | cut -d'/' -f2)

  # Construct the output file name
  output_name="btxz-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    output_name+='.exe'
  fi

  echo "--> Building for ${GOOS}/${GOARCH}..."
  
  # Execute the build command. Note that we are building the SOURCE_DIR from the root.
  # The -v flag provides verbose output for better logging.
  env GOOS="$GOOS" GOARCH="$GOARCH" go build -v ${LDFLAGS} -o "${OUTPUT_DIR}/${output_name}" "./${SOURCE_DIR}"
done

echo ""
echo "✅ Build process completed successfully."
echo "Artifacts are located in the '${OUTPUT_DIR}' directory."
