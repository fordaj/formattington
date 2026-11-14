#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 0.50.0
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: pnpm, bun

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default compilation targets.
if [[ $# -eq 0 ]]; then
    TARGETS=(
        linux-arm64
        linux-x86_64
        macos-arm64
        macos-x86_64
        windows-arm64
        windows-x86_64
    )
else
    TARGETS=("$@")
fi

# Map platform/architecture names to Bun targets.
get_bun_target() {
    case "$1" in
    linux-arm64) echo "bun-linux-arm64" ;;
    linux-x86_64) echo "bun-linux-x64" ;;
    macos-arm64) echo "bun-darwin-arm64" ;;
    macos-x86_64) echo "bun-darwin-x64" ;;
    windows-arm64) echo "bun-windows-arm64" ;;
    windows-x86_64) echo "bun-windows-x64" ;;
    *)
        echo "Error: Unsupported target: $1" >&2
        return 1
        ;;
    esac
}

# Verify dependencies.
for command in pnpm bun; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Error: $command is required." >&2
        exit 1
    fi
done

# Create an isolated temporary installation.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Installing markdownlint-cli@$VERSION..."

cd "$WORK_DIR"
pnpm add --save-exact "markdownlint-cli@$VERSION"

MARKDOWNLINT_VERSION="$(pnpm exec markdownlint --version)"
MARKDOWNLINT_DIR="$WORK_DIR/node_modules/markdownlint-cli"
CLI_MODULE="$MARKDOWNLINT_DIR/markdownlint.js"

if [[ ! -f "$CLI_MODULE" ]]; then
    echo "Error: markdownlint CLI module not found: $CLI_MODULE" >&2
    exit 1
fi

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$MARKDOWNLINT_VERSION"
mkdir -p "$VERSION_DIR"

echo "Building markdownlint-cli $MARKDOWNLINT_VERSION..."

for TARGET in "${TARGETS[@]}"; do
    BUN_TARGET="$(get_bun_target "$TARGET")"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/markdownlint"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    bun build \
        --compile \
        --target="$BUN_TARGET" \
        "$CLI_MODULE" \
        --outfile "$OUTPUT"

    echo "     Created: $OUTPUT"
done

echo
echo "Build complete!"
echo "markdownlint-cli version: $MARKDOWNLINT_VERSION"
echo "Output directory: $VERSION_DIR"
