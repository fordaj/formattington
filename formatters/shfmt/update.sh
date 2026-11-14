#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 3.14.1
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default download targets.
if [[ $# -eq 0 ]]; then
    TARGETS=(
        linux-arm64
        linux-x86_64
        macos-arm64
        macos-x86_64
        # windows-arm64
        windows-x86_64
    )
else
    TARGETS=("$@")
fi

# Map platform/architecture names to GitHub release assets.
get_asset_name() {
    case "$1" in
    linux-arm64) echo "shfmt_${TAG_NAME}_linux_arm64" ;;
    linux-x86_64) echo "shfmt_${TAG_NAME}_linux_amd64" ;;
    macos-arm64) echo "shfmt_${TAG_NAME}_darwin_arm64" ;;
    macos-x86_64) echo "shfmt_${TAG_NAME}_darwin_amd64" ;;
    windows-arm64)
        echo "Error: shfmt does not publish Windows ARM64 binaries." >&2
        return 1
        ;;
    windows-x86_64) echo "shfmt_${TAG_NAME}_windows_amd64.exe" ;;
    *)
        echo "Error: Unsupported target: $1" >&2
        return 1
        ;;
    esac
}

# Verify dependencies.
for command in curl jq; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Error: $command is required." >&2
        exit 1
    fi
done

# Resolve release information.
REPO="mvdan/sh"

if [[ "$VERSION" == "latest" ]]; then
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
else
    TAG_NAME="${VERSION#v}"
    TAG_NAME="v$TAG_NAME"
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$TAG_NAME"
fi

echo "Resolving shfmt release..."

RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")"

TAG_NAME="$(jq -er '.tag_name' <<<"$RELEASE_JSON")"
SHFMT_VERSION="${TAG_NAME#v}"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$SHFMT_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading shfmt $SHFMT_VERSION..."

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="$(
        jq -er --arg name "$ASSET_NAME" \
            '.assets[] | select(.name == $name) | .browser_download_url' \
            <<<"$RELEASE_JSON"
    )" || {
        echo "Error: Release asset not found: $ASSET_NAME" >&2
        exit 1
    }

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/shfmt"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$OUTPUT"

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Created: $OUTPUT"
done

echo
echo "Download complete!"
echo "shfmt version: $SHFMT_VERSION"
echo "Output directory: $VERSION_DIR"
