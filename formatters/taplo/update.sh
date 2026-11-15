#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.sh [version] [target...]
#
# Examples:
#   ./update.sh
#   ./update.sh 0.10.0
#   ./update.sh latest windows-x86_64 macos-arm64
#
# Requirements: curl, gzip, jq, tar

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
        windows-arm64
        windows-x86_64
    )
else
    TARGETS=("$@")
fi

# Pinned SHA-256 of every asset this script fetches (of the gzipped
# release archives themselves). Release bumps must re-pin: audit the new
# upstream release, update PINNED_VERSION, and replace the hashes below
# with `shasum -a 256 <asset>`. The script refuses any version other than
# PINNED_VERSION and hard-fails on a hash mismatch, so an unverifiable
# binary is never fetched into a release.
PINNED_VERSION="0.10.0"
PINNED_SHA256_linux_arm64="033681d01eec8376c3fd38fa3703c79316f5e14bb013d859943b60a07bccdcc3"
PINNED_SHA256_linux_x86_64="8fe196b894ccf9072f98d4e1013a180306e17d244830b03986ee5e8eabeb6156"
PINNED_SHA256_macos_arm64="713734314c3e71894b9e77513c5349835eefbd52908445a0d73b0c7dc469347d"
PINNED_SHA256_macos_x86_64="898122cde3a0b1cd1cbc2d52d3624f23338218c91b5ddb71518236a4c2c10ef2"
PINNED_SHA256_windows_arm64="c16a9a1248bdd746fde657b0196bfc47c8bede1e50ae73b7d9d33088a6682180"
PINNED_SHA256_windows_x86_64="550fdc955343f8a196447a05346ecf8827e391726e4ded577bb00a0a36cf220c"

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
    linux-arm64) echo "taplo-linux-aarch64.gz" ;;
    linux-x86_64) echo "taplo-linux-x86_64.gz" ;;
    macos-arm64) echo "taplo-darwin-aarch64.gz" ;;
    macos-x86_64) echo "taplo-darwin-x86_64.gz" ;;
    windows-arm64) echo "taplo-windows-aarch64.gz" ;;
    windows-x86_64) echo "taplo-windows-x86_64.gz" ;;
    *)
        echo "Error: Unsupported target: $1" >&2
        return 1
        ;;
    esac
}

# Verify dependencies.
for command in curl gzip jq; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Error: $command is required." >&2
        exit 1
    fi
done

# Resolve release information.
REPO="tamasfe/taplo"

if [[ "$VERSION" == "latest" ]]; then
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
else
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
fi

echo "Resolving Taplo release..."

RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")"

TAG_NAME="$(jq -er '.tag_name' <<<"$RELEASE_JSON")"
TAPLO_VERSION="${TAG_NAME#v}"
TAPLO_VERSION="${TAPLO_VERSION#v}"

if [[ "$TAPLO_VERSION" != "$PINNED_VERSION" ]]; then
    echo "Error: this script pins taplo $PINNED_VERSION, but the release resolved to $TAPLO_VERSION." >&2
    echo "Re-pin PINNED_VERSION and the hashes in this script after auditing the new release." >&2
    exit 1
fi

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$TAPLO_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading Taplo $TAPLO_VERSION..."

# Create an isolated temporary directory.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/taplo"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    ARCHIVE="$WORK_DIR/$ASSET_NAME"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$ARCHIVE"

    local_key="${TARGET//-/_}"
    local_var="PINNED_SHA256_${local_key}"
    sha256_check "$ARCHIVE" "${!local_var}" || exit 1

    gzip -t "$ARCHIVE"
    gzip -dc "$ARCHIVE" >"$OUTPUT"

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Created: $OUTPUT"
done

echo
echo "Download complete!"
echo "Taplo version: $TAPLO_VERSION"
echo "Output directory: $VERSION_DIR"
