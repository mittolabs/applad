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
    Linux*)     TARGET_OS="unknown-linux-gnu";;
    Darwin*)    TARGET_OS="apple-darwin";;
    MINGW*|CYGWIN*|MSYS*) TARGET_OS="pc-windows-msvc";;
    *)          echo "Error: Unsupported OS: ${OS}"; exit 1;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) TARGET_ARCH="x86_64";;
    arm64|aarch64) TARGET_ARCH="aarch64";;
    *)             echo "Error: Unsupported architecture: ${ARCH}"; exit 1;;
esac

TARGET="${TARGET_ARCH}-${TARGET_OS}"

echo "=> Fetching latest release version from ${REPO}..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "Error: Could not fetch latest release. Are you rate-limited?"
    exit 1
fi

echo "=> Latest version is ${LATEST_VERSION}"

# Construct download URL
ZIP_NAME="applad-${TARGET}.zip"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${ZIP_NAME}"

echo "=> Downloading ${ZIP_NAME}..."
mkdir -p "${INSTALL_DIR}"

TMP_DIR="$(mktemp -d)"
TMP_ZIP="${TMP_DIR}/${ZIP_NAME}"

if ! curl -sL --fail "${DOWNLOAD_URL}" -o "${TMP_ZIP}"; then
    echo "Error: Failed to download the binary from ${DOWNLOAD_URL}"
    echo "Ensure that the release assets are successfully published."
    rm -rf "${TMP_DIR}"
    exit 1
fi

echo "=> Extracting archive..."
unzip -q "${TMP_ZIP}" -d "${TMP_DIR}"

# Check for windows .exe
if [ -f "${TMP_DIR}/applad.exe" ]; then
    mv "${TMP_DIR}/applad.exe" "${INSTALL_DIR}/${EXE_NAME}.exe"
    chmod +x "${INSTALL_DIR}/${EXE_NAME}.exe"
else
    mv "${TMP_DIR}/applad" "${INSTALL_DIR}/${EXE_NAME}"
    chmod +x "${INSTALL_DIR}/${EXE_NAME}"
fi

rm -rf "${TMP_DIR}"

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
