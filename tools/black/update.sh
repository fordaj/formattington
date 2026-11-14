#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash latest
#   ./update.bash 26.10.1
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
        windows-arm64
        windows-x86_64
    )
else
    TARGETS=("$@")
fi

# Map target names to GitHub asset matching patterns.
get_asset_name() {
    case "$1" in
    linux-arm64) echo "black_linux-arm" ;;
    linux-x86_64) echo "black_linux" ;;
    macos-arm64) echo "black_macos" ;;
    windows-arm64) echo "black_windows-arm.exe" ;;
    windows-x86_64) echo "black_windows.exe" ;;
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

# Retrieve release metadata.
if [[ "$VERSION" == "latest" ]]; then
    RELEASE_URL="https://api.github.com/repos/psf/black/releases/latest"
else
    VERSION="${VERSION#v}"
    RELEASE_URL="https://api.github.com/repos/psf/black/releases/tags/$VERSION"
fi

echo "Retrieving Black release information..."

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

curl -fsSL "$RELEASE_URL" -o "$WORK_DIR/release.json"

RELEASE_VERSION="$(jq -er '.tag_name' "$WORK_DIR/release.json")"
RELEASE_VERSION="${RELEASE_VERSION#v}"

VERSION_DIR="$OUTPUT_DIR/v$RELEASE_VERSION"
mkdir -p "$VERSION_DIR"

echo "Black version: $RELEASE_VERSION"
echo

# Download each requested platform.
for TARGET in "${TARGETS[@]}"; do

    # Find exactly one matching standalone executable.
    ASSET_NAME="$(get_asset_name "$TARGET")"

    ASSET_URL="$(
        jq -er --arg name "$ASSET_NAME" '
            .assets[]
            | select(.name == $name)
            | .browser_download_url
        ' "$WORK_DIR/release.json"
    )" || {
        echo "Error: Release asset not found: $ASSET_NAME" >&2
        exit 1
    }

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/black"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"
    curl -fSL "$ASSET_URL" -o "$OUTPUT"

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Created: $OUTPUT"
done

echo
echo "Download complete!"
echo "Black version: $RELEASE_VERSION"
echo "Output directory: $VERSION_DIR"
