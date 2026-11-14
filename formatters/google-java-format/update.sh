#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 2.0.0
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq
#
# google-java-format publishes GraalVM native-image binaries (unsigned,
# single file) for four of six targets since v2.0.0. There is no asset
# for macOS x86_64 or Windows arm64; those platforms keep the PATH
# fallback (a user-managed JDK >= 21 with the six --add-exports flags, or
# the all-deps jar). Upstream does not publish checksums for the native
# binaries, so the version is pinned by this script instead.

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default download targets. macOS x86_64 and Windows arm64 are intentionally
# absent (no upstream asset).
if [[ $# -eq 0 ]]; then
    TARGETS=(
        linux-arm64
        linux-x86_64
        macos-arm64
        windows-x86_64
    )
else
    TARGETS=("$@")
fi

# Map platform/architecture names to GitHub release assets.
get_asset_name() {
    case "$1" in
    linux-arm64) echo "google-java-format_linux-arm64" ;;
    linux-x86_64) echo "google-java-format_linux-x86-64" ;;
    macos-arm64) echo "google-java-format_darwin-arm64" ;;
    windows-x86_64) echo "google-java-format_windows-x86-64.exe" ;;
    *)
        echo "Error: Unsupported target: $1 (no upstream asset)" >&2
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

REPO="google/google-java-format"

if [[ "$VERSION" == "latest" ]]; then
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
else
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
fi

echo "Resolving google-java-format release..."

RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")"

TAG_NAME="$(jq -er '.tag_name' <<<"$RELEASE_JSON")"
GJF_VERSION="${TAG_NAME#v}"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$GJF_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading google-java-format $GJF_VERSION..."

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/google-java-format"

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
echo "google-java-format version: $GJF_VERSION"
echo "Output directory: $VERSION_DIR"
