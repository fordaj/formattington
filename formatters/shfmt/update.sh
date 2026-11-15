#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.sh [version] [target...]
#
# Examples:
#   ./update.sh
#   ./update.sh 3.14.1
#   ./update.sh latest windows-x86_64 macos-arm64
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

# Pinned SHA-256 of every asset this script fetches. Release bumps must
# re-pin (audit the new upstream release, update PINNED_VERSION, then the
# hashes below with `shasum -a 256 <asset>`, and re-run this script for all
# targets). The script refuses any version other than PINNED_VERSION and
# hard-fails on a hash mismatch, so an unverifiable binary is never
# fetched into a release.
PINNED_VERSION="3.14.1"
PINNED_SHA256_linux_arm64="5f2db09dae91fca848f7adbdd014632e921a383863a2ad7e0450ad3aba0c6489"
PINNED_SHA256_linux_x86_64="76e77641faa025814b77f153b29796b8e6fa2fca03e0c76a691608b86c7ea7bf"
PINNED_SHA256_macos_arm64="b7c872db63553ccffc7253aba3ed7d4885a27d83f1ba567b1138c6315a5847e5"
PINNED_SHA256_macos_x86_64="d33eee0da0f92835b3562e9767a05cee7e4eaeef47daa03bfd09da17b4b590a6"
PINNED_SHA256_windows_x86_64="13629ce28442ca80b6b5a819f7574ab39e1c28c6e26734ca816c9714e04851df"

sha256_check() { # <file> <expected>
    local actual
    actual="$(shasum -a 256 "$1" | awk '{print $1}')"
    if [[ "$actual" != "$2" ]]; then
        echo "Error: SHA-256 verification failed for $1" >&2
        echo "  expected: $2" >&2
        echo "  actual:   $actual" >&2
        return 1
    fi
}

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

if [[ "$SHFMT_VERSION" != "$PINNED_VERSION" ]]; then
    echo "Error: this script pins shfmt $PINNED_VERSION, but the release resolved to $SHFMT_VERSION." >&2
    echo "Re-pin PINNED_VERSION and the hashes in this script after auditing the new release." >&2
    exit 1
fi

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

    local_key="${TARGET//-/_}"
    local_var="PINNED_SHA256_${local_key}"
    sha256_check "$OUTPUT" "${!local_var}" || { rm -f "$OUTPUT"; exit 1; }

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Created: $OUTPUT"
done

echo
echo "Download complete!"
echo "shfmt version: $SHFMT_VERSION"
echo "Output directory: $VERSION_DIR"
