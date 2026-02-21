#!/usr/bin/env bash
set -e

REPO="mittolabs/applad"
INSTALL_DIR="$HOME/.applad/bin"
EXE_NAME="applad"

echo "================================================="
echo " Installing Applad CLI"
echo "================================================="

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     PLATFORM="linux";;
    Darwin*)    PLATFORM="macos";;
    MINGW*|CYGWIN*|MSYS*) PLATFORM="windows";;
    *)          echo "Error: Unsupported OS: ${OS}"; exit 1;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) ARCH="x64";;
    arm64|aarch64) ARCH="arm64";;
    *)             echo "Error: Unsupported architecture: ${ARCH}"; exit 1;;
esac

echo "=> Fetching latest release version from ${REPO}..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "Error: Could not fetch latest release. Are you rate-limited?"
    exit 1
fi

echo "=> Latest version is ${LATEST_VERSION}"

# Construct download URL (matching the GitHub Action artifact names)
BINARY_NAME="applad-${PLATFORM}-${ARCH}"
if [ "${PLATFORM}" = "windows" ]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${BINARY_NAME}"

echo "=> Downloading ${BINARY_NAME}..."
mkdir -p "${INSTALL_DIR}"

TMP_FILE="$(mktemp)"
if ! curl -sL --fail "${DOWNLOAD_URL}" -o "${TMP_FILE}"; then
    echo "Error: Failed to download the binary from ${DOWNLOAD_URL}"
    echo "Ensure that the release assets are successfully published."
    rm -f "${TMP_FILE}"
    exit 1
fi

mv "${TMP_FILE}" "${INSTALL_DIR}/${EXE_NAME}"
chmod +x "${INSTALL_DIR}/${EXE_NAME}"

echo ""
echo "=> Applad CLI installed successfully to:"
echo "   ${INSTALL_DIR}/${EXE_NAME}"
echo ""

# Provide PATH instructions
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "================================================="
    echo " ACTION REQUIRED: Add Applad to your PATH "
    echo "================================================="
    echo "Please add the following line to your shell profile"
    echo "(e.g., ~/.bashrc, ~/.zshrc, or ~/.profile):"
    echo ""
    echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
    echo "Then restart your terminal or run source on your profile."
    echo "================================================="
else
    echo "=> Applad is already in your PATH! Run 'applad' to get started."
fi
