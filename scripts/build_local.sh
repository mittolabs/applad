#!/usr/bin/env bash
set -e

# ==============================================================================
# Configuration
# ==============================================================================
INSTALL_DIR="$HOME/.applad/bin"
EXE_NAME="applad"
CLI_DIR="$(pwd)/packages/applad_cli"

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

# ==============================================================================
# Build Script
# ==============================================================================
echo ""
echo "${BOLD}Applad Local CLI Builder${RESET}"
echo ""

# 1. Verify in root
if [ ! -d "$CLI_DIR" ]; then
    error "Could not find $CLI_DIR. Please run this script from the project root."
    exit 1
fi

# 2. Compile
info "Compiling Applad CLI from source..."
cd "$CLI_DIR"
dart pub get > /dev/null
dart compile exe bin/applad.dart -o applad_bin

# 3. Install
mkdir -p "$INSTALL_DIR"
mv applad_bin "$INSTALL_DIR/$EXE_NAME"
chmod +x "$INSTALL_DIR/$EXE_NAME"

echo ""
success "${BOLD}Applad CLI built and installed successfully!${RESET}"
echo "    Location: ${DIM}${INSTALL_DIR}/${EXE_NAME}${RESET}"
echo ""

# 4. PATH Instructions
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "${BOLD}Next steps:${RESET}"
    echo "  Please add the following line to your shell profile (~/.zshrc, ~/.bashrc, etc.):"
    echo ""
    echo "    ${BLUE}export PATH=\"\$PATH:${INSTALL_DIR}\"${RESET}"
    echo ""
    echo "  Then restart your terminal."
else
    info "Applad is already in your PATH."
fi
echo ""
