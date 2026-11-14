#!/usr/bin/env bash
set -euo pipefail

# Build Black's standalone macOS x86_64 executable.
#
# Usage:
#   ./build-black-macos-x86_64.bash
#   ./build-black-macos-x86_64.bash 26.10.1
#
# Requirements: git, uv, Xcode Command Line Tools

VERSION="${1:-26.10.1}"
VERSION="${VERSION#v}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Verify native Intel environment.
if [[ "$(uname -m)" != "x86_64" ]]; then
    echo "Error: Must run natively on an Intel Mac." >&2
    exit 1
fi

for COMMAND in git uv xcrun lipo; do
    if ! command -v "$COMMAND" >/dev/null 2>&1; then
        echo "Error: $COMMAND is required." >&2
        exit 1
    fi
done

if ! xcrun --find clang >/dev/null 2>&1; then
    echo "Error: Xcode Command Line Tools are required." >&2
    exit 1
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

REPO_DIR="$WORK_DIR/black"
VENV_DIR="$WORK_DIR/venv"

echo "Cloning Black..."
git clone --quiet https://github.com/psf/black.git "$REPO_DIR"

cd "$REPO_DIR"

echo "Checking out Black $VERSION..."
git checkout --quiet "$VERSION"

echo "Creating Intel Python environment..."
uv venv --python 3.13 "$VENV_DIR"

PYTHON="$VENV_DIR/bin/python"

if [[ "$("$PYTHON" -c 'import platform; print(platform.machine())')" != "x86_64" ]]; then
    echo "Error: Python interpreter is not x86_64." >&2
    exit 1
fi

echo "Installing Black and PyInstaller..."
uv pip install --python "$PYTHON" ".[colorama]" pyinstaller

# Black's release workflow explicitly generates this file.
echo "version = \"$VERSION\"" >src/_black_version.py

echo "Building standalone executable..."
"$PYTHON" -m PyInstaller \
    --add-data "$REPO_DIR/src/blib2to3:blib2to3" \
    --onefile \
    --name black_macos \
    --distpath "$WORK_DIR/dist" \
    --workpath "$WORK_DIR/build" \
    --specpath "$WORK_DIR" \
    src/black/__main__.py

BINARY="$WORK_DIR/dist/black_macos"

echo "Verifying architecture..."
ARCH="$(lipo -archs "$BINARY")"

if [[ "$ARCH" != "x86_64" ]]; then
    echo "Error: Expected x86_64, got: $ARCH" >&2
    exit 1
fi

echo "Testing executable..."
"$BINARY" --version
"$BINARY" src --check --verbose

# Publish output.
TARGET_DIR="$OUTPUT_DIR/v$VERSION/macos-x86_64"
mkdir -p "$TARGET_DIR"

install -m 755 "$BINARY" "$TARGET_DIR/black"

echo
echo "Build successful!"
echo "Version:     $VERSION"
echo "Architecture: $ARCH"
echo "Output:      $TARGET_DIR/black"
