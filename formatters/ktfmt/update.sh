#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 0.65
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq, tar, unzip, sha256sum or shasum
#
# ktfmt publishes GraalVM native-image binaries for three of six targets:
# Linux x86_64, macOS arm64, and Windows x86_64. There is no Linux arm64,
# macOS x86_64, or Windows arm64 asset; those platforms keep the PATH
# fallback (a user-managed ktfmt on PATH, or the with-dependencies jar,
# which needs a JDK >= 11). Upstream publishes a .sha256 sidecar per
# asset; it is verified during download.

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default download targets. Linux arm64, macOS x86_64 and Windows arm64
# are intentionally absent (no upstream asset).
if [[ $# -eq 0 ]]; then
    TARGETS=(
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
    linux-x86_64) echo "ktfmt-linux-x86_64-$VERSION_NUMBER.tar.gz" ;;
    macos-arm64) echo "ktfmt-macos-aarch64-$VERSION_NUMBER.tar.gz" ;;
    windows-x86_64) echo "ktfmt-windows-x86_64-$VERSION_NUMBER.zip" ;;
    *)
        echo "Error: Unsupported target: $1 (no upstream asset)" >&2
        return 1
        ;;
    esac
}

# Verify dependencies.
for command in curl jq tar unzip; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Error: $command is required." >&2
        exit 1
    fi
done

if command -v sha256sum >/dev/null 2>&1; then
    SHA256_COMMAND="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    SHA256_COMMAND="shasum"
else
    echo "Error: sha256sum or shasum is required." >&2
    exit 1
fi

REPO="Kotlin/ktfmt"

# Prefer the authenticated gh CLI (unauthenticated api.github.com
# requests may be refused); curl is the fallback.
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    if [[ "$VERSION" == "latest" ]]; then
        RELEASE_JSON="$(gh api "repos/$REPO/releases/latest")"
    else
        RELEASE_JSON="$(gh api "repos/$REPO/releases/tags/$VERSION")"
    fi
else
    if [[ "$VERSION" == "latest" ]]; then
        RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
    else
        RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
    fi
    RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")"
fi
TAG_NAME="$(jq -er '.tag_name' <<<"$RELEASE_JSON")"
KTFMT_VERSION="${TAG_NAME#v}"
VERSION_NUMBER="$KTFMT_VERSION"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$KTFMT_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading ktfmt $KTFMT_VERSION..."

# Create an isolated temporary directory.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/ktfmt"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    ARCHIVE="$WORK_DIR/$ASSET_NAME"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$ARCHIVE"
    curl -fL --retry 3 "${DOWNLOAD_URL}.sha256" -o "${ARCHIVE}.sha256"

    EXPECTED_SHA256="$(cat "${ARCHIVE}.sha256" | awk '{print $1}')"

    if [[ "$SHA256_COMMAND" == "sha256sum" ]]; then
        ACTUAL_SHA256="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
    else
        ACTUAL_SHA256="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
    fi

    if [[ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]]; then
        echo "Error: SHA-256 verification failed for $TARGET" >&2
        rm -rf "$TARGET_DIR"
        exit 1
    fi

    # Extract the single ktfmt executable from the archive's top directory.
    EXTRACT_DIR="$WORK_DIR/extract-$TARGET"
    mkdir -p "$EXTRACT_DIR"

    case "$ASSET_NAME" in
    *.tar.gz)
        tar -xzf "$ARCHIVE" -C "$EXTRACT_DIR"
        ;;
    *.zip)
        unzip -q "$ARCHIVE" -d "$EXTRACT_DIR"
        ;;
    *)
        echo "Error: Unknown archive format: $ASSET_NAME" >&2
        rm -rf "$TARGET_DIR"
        exit 1
        ;;
    esac

    FOUND="$(find "$EXTRACT_DIR" -type f -name 'ktfmt' -not -path '*/node_modules/*' -print | head -n 1)"
    if [[ "$TARGET" == windows-* ]]; then
        FOUND="$(find "$EXTRACT_DIR" -type f -name 'ktfmt.exe' -print | head -n 1)"
    fi

    if [[ -z "$FOUND" ]]; then
        echo "Error: ktfmt executable not found in $ASSET_NAME" >&2
        rm -rf "$TARGET_DIR"
        exit 1
    fi

    cp "$FOUND" "$OUTPUT"

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Verified: $OUTPUT"
done

echo
echo "Download complete!"
echo "ktfmt version: $KTFMT_VERSION"
echo "Output directory: $VERSION_DIR"
