#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 2.6.0
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq, unzip
#
# StyLua ships prebuilt binaries for linux (musl and glibc), macOS, and
# Windows x86_64. There is no Windows arm64 asset; that target is skipped
# with a warning so the platform keeps its PATH-only fallback. The musl
# Linux assets are preferred for glibc compatibility. Upstream does not
# publish checksums, so the version is pinned by this script instead.

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default download targets. Windows arm64 is intentionally absent.
if [[ $# -eq 0 ]]; then
    TARGETS=(
        linux-arm64
        linux-x86_64
        macos-arm64
        macos-x86_64
        windows-x86_64
    )
else
    TARGETS=("$@")
fi

# Map platform/architecture names to GitHub release assets.
get_asset_name() {
    case "$1" in
    linux-arm64) echo "stylua-linux-aarch64-musl.zip" ;;
    linux-x86_64) echo "stylua-linux-x86_64-musl.zip" ;;
    macos-arm64) echo "stylua-macos-aarch64.zip" ;;
    macos-x86_64) echo "stylua-macos-x86_64.zip" ;;
    windows-x86_64) echo "stylua-windows-x86_64.zip" ;;
    *)
        echo "Error: Unsupported target: $1 (no upstream asset)" >&2
        return 1
        ;;
    esac
}

# Verify dependencies.
for command in curl jq unzip; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Error: $command is required." >&2
        exit 1
    fi
done

REPO="JohnnyMorganz/StyLua"

if [[ "$VERSION" == "latest" ]]; then
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
else
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
fi

echo "Resolving StyLua release..."

RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")"

TAG_NAME="$(jq -er '.tag_name' <<<"$RELEASE_JSON")"
STYLUA_VERSION="${TAG_NAME#v}"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$STYLUA_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading StyLua $STYLUA_VERSION..."

# Create an isolated temporary directory.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/stylua"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    ARCHIVE="$WORK_DIR/$ASSET_NAME"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$ARCHIVE"

    unzip -p "$ARCHIVE" stylua >"$OUTPUT" 2>/dev/null ||
        unzip -p "$ARCHIVE" stylua.exe >"$OUTPUT"

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Created: $OUTPUT"
done

echo
echo "Download complete!"
echo "StyLua version: $STYLUA_VERSION"
echo "Output directory: $VERSION_DIR"
