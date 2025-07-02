#!/bin/sh

# BTXZ™ Installer Script
#
# This script downloads and installs the latest version of BTXZ for your system.
# It is designed to be run directly from the web, e.g.:
# curl -fsSL https://raw.githubusercontent.com/BlackTechX011/BTXZ/main/scripts/install.sh | sh
#
# Copyright (c) 2025-present, BlackTechX011
# Licensed under the BTXZ EULA. See https://github.com/BlackTechX011/BTXZ/blob/main/LICENSE.md

set -e # Exit immediately if a command exits with a non-zero status.

# --- Configuration ---
REPO="BlackTechX011/BTXZ"
INSTALL_DIR="$HOME/.btxz"
PROFILE_FILES="$HOME/.zshrc $HOME/.bashrc $HOME/.profile"
EXE_NAME="btxz"

# --- Helper Functions ---
print_info() {
    printf "\033[1;34m%s\033[0m\n" "$1"
}

print_success() {
    printf "\033[1;32m%s\033[0m\n" "$1"
}

print_error() {
    printf "\033[1;31mError: %s\033[0m\n" "$1" >&2
    exit 1
}

# --- Main Logic ---
main() {
    print_info "Starting BTXZ™ installation..."

    # 1. Detect OS and Architecture
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$arch" in
        x86_64 | amd64)
            arch="amd64"
            ;;
        aarch64 | arm64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch. BTXZ currently supports amd64 and arm64."
            ;;
    esac

    case "$os" in
        linux | darwin)
            # OS is supported, continue
            ;;
        *)
            print_error "Unsupported operating system: $os. BTXZ currently supports Linux and macOS."
            ;;
    esac
    
    binary_name="${EXE_NAME}-${os}-${arch}"
    print_info "Detected System: ${os}-${arch}. Target binary: ${binary_name}"

    # 2. Get the download URL for the latest release
    api_url="https://api.github.com/repos/${REPO}/releases/latest"
    print_info "Fetching latest release information from GitHub..."
    
    # Use curl to fetch and grep/cut to parse JSON without needing jq
    download_url=$(curl -fsSL "$api_url" | grep "browser_download_url" | grep "$binary_name" | cut -d '"' -f 4)

    if [ -z "$download_url" ]; then
        print_error "Could not find a download URL for '$binary_name'. The release may be missing or the API is unavailable."
    fi

    print_info "Download URL: $download_url"

    # 3. Download the binary
    temp_file=$(mktemp)
    print_info "Downloading to a temporary file..."
    curl -fLo "$temp_file" "$download_url"

    # 4. Install the binary
    install_path="$INSTALL_DIR/$EXE_NAME"
    print_info "Installing to $install_path..."
    mkdir -p "$INSTALL_DIR"
    mv "$temp_file" "$install_path"
    chmod +x "$install_path"

    # 5. Add to PATH
    print_info "Adding installation directory to your shell's PATH..."
    path_export="export PATH=\"\$HOME/.btxz:\$PATH\""
    
    # Find the correct profile file to update
    profile_to_update=""
    for profile in $PROFILE_FILES; do
        if [ -f "$profile" ]; then
            profile_to_update="$profile"
            break
        fi
    done

    if [ -z "$profile_to_update" ]; then
        # If no common profile file is found, create a .profile
        profile_to_update="$HOME/.profile"
        touch "$profile_to_update"
        print_info "Created $profile_to_update as no standard profile was found."
    fi

    if ! grep -q "export PATH=\"\$HOME/.btxz:\$PATH\"" "$profile_to_update"; then
        echo "\n# Added by BTXZ installer" >> "$profile_to_update"
        echo "$path_export" >> "$profile_to_update"
        print_info "Updated '$profile_to_update'. Please restart your terminal or run:"
        print_info "  source $profile_to_update"
    else
        print_info "PATH already configured in '$profile_to_update'."
    fi

    print_success "\nBTXZ™ was installed successfully!"
    print_info "You can now run 'btxz' from your terminal."
}

# Run the main function
main
