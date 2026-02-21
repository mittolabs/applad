#!/usr/bin/env bash
set -e

# ==============================================================================
# Configuration
# ==============================================================================
REPO="mittolabs/applad"
INSTALL_DIR="$HOME/.applad/bin"
EXE_NAME="applad"

# ==============================================================================
# Colors & Formatting
# ==============================================================================
BOLD="$(tput bold 2>/dev/null || printf '')"
DIM="$(tput dim 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
BLUE="$(tput setaf 4 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
RESET="$(tput sgr0 2>/dev/null || printf '')"

info() { echo "  ${BLUE}•${RESET} $*"; }
success() { echo "  ${GREEN}✓${RESET} $*"; }
error() { echo "  ${RED}✗${RESET} $*" >&2; }
bold() { echo "${BOLD}$*${RESET}"; }

# ==============================================================================
# Installation Script
# ==============================================================================
echo ""
echo "${BOLD}Applad CLI Installer${RESET}"
echo ""

# 1. Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     TARGET_OS="unknown-linux-gnu";;
    Darwin*)    TARGET_OS="apple-darwin";;
    MINGW*|CYGWIN*|MSYS*) TARGET_OS="pc-windows-msvc";;
    *)          error "Unsupported OS: ${OS}"; exit 1;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) TARGET_ARCH="x86_64";;
    arm64|aarch64) TARGET_ARCH="aarch64";;
    *)             error "Unsupported architecture: ${ARCH}"; exit 1;;
esac

TARGET="${TARGET_ARCH}-${TARGET_OS}"
info "Detected platform: ${DIM}${TARGET}${RESET}"

# 3. Fetch Latest Version
LATEST_VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    error "Could not fetch latest release. Are you rate-limited?"
    exit 1
fi
info "Latest version: ${BOLD}${LATEST_VERSION}${RESET}"

# 4. Construct download URL
ZIP_NAME="applad-${TARGET}.zip"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${ZIP_NAME}"

info "Downloading ${DIM}${ZIP_NAME}${RESET}..."
mkdir -p "${INSTALL_DIR}"

TMP_DIR="$(mktemp -d)"
TMP_ZIP="${TMP_DIR}/${ZIP_NAME}"

if ! curl -sL --fail "${DOWNLOAD_URL}" -o "${TMP_ZIP}"; then
    error "Failed to download the binary from ${DOWNLOAD_URL}"
    echo "    Ensure that the release assets are successfully published."
    rm -rf "${TMP_DIR}"
    exit 1
fi

info "Extracting archive..."
unzip -q "${TMP_ZIP}" -d "${TMP_DIR}"

# 5. Move and permission binary
if [ -f "${TMP_DIR}/applad.exe" ]; then
    mv "${TMP_DIR}/applad.exe" "${INSTALL_DIR}/${EXE_NAME}.exe"
    chmod +x "${INSTALL_DIR}/${EXE_NAME}.exe"
else
    mv "${TMP_DIR}/applad" "${INSTALL_DIR}/${EXE_NAME}"
    chmod +x "${INSTALL_DIR}/${EXE_NAME}"
fi

rm -rf "${TMP_DIR}"

echo ""
success "${BOLD}Applad CLI installed successfully!${RESET}"
echo "    Location: ${DIM}${INSTALL_DIR}/${EXE_NAME}${RESET}"
echo ""

# 6. Provide PATH instructions
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "${BOLD}Next steps:${RESET}"
    echo "  It looks like ${INSTALL_DIR} is not in your PATH."
    echo "  Please add the following line to your shell profile (~/.zshrc, ~/.bashrc, etc.):"
    echo ""
    echo "    ${BLUE}export PATH=\"\$PATH:${INSTALL_DIR}\"${RESET}"
    echo ""
    echo "  Then restart your terminal."
else
    echo "${BOLD}Next steps:${RESET}"
    echo "  Applad is already in your PATH. Run ${BLUE}applad --help${RESET} to get started."
fi
echo ""
