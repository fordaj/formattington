#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 0.16.6
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq, tar, unzip, sha256sum or shasum
#
# cljfmt ships native GraalVM binaries for four of six targets:
# macOS arm64, Linux arm64, Linux x86_64 (static/musl), and Windows
# x86_64. There is no macOS x86_64 or Windows arm64 asset; those
# platforms keep the PATH fallback (brew install cljfmt, or the
# standalone jar on a JRE). cljfmt is distributed under EPL-1.0; the
# redistributed binaries are unmodified upstream build artifacts. A
# .sha256 sidecar is published per asset and is verified on download.

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default download targets. macOS x86_64 and Windows arm64 are
# intentionally absent (no upstream asset).
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
    linux-arm64) echo "cljfmt-$VERSION_NUMBER-linux-aarch64.tar.gz" ;;
    linux-x86_64) echo "cljfmt-$VERSION_NUMBER-linux-amd64-static.tar.gz" ;;
    macos-arm64) echo "cljfmt-$VERSION_NUMBER-darwin-aarch64.tar.gz" ;;
    windows-x86_64) echo "cljfmt-$VERSION_NUMBER-win-amd64.zip" ;;
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

REPO="weavejester/cljfmt"

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
CLJFMT_VERSION="${TAG_NAME#v}"
VERSION_NUMBER="$CLJFMT_VERSION"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$CLJFMT_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading cljfmt $CLJFMT_VERSION..."

# Create an isolated temporary directory.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/cljfmt"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    ARCHIVE="$WORK_DIR/$ASSET_NAME"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$ARCHIVE"
    curl -fL --retry 3 "${DOWNLOAD_URL}.sha256" -o "${ARCHIVE}.sha256"

    EXPECTED_SHA256="$(awk '{print $1}' "${ARCHIVE}.sha256")"

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

    # Extract the bare cljfmt executable.
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

    if [[ "$TARGET" == windows-* ]]; then
        FOUND="$(find "$EXTRACT_DIR" -type f -name 'cljfmt.exe' -print | head -n 1)"
    else
        FOUND="$(find "$EXTRACT_DIR" -type f -name 'cljfmt' -print | head -n 1)"
    fi

    if [[ -z "$FOUND" ]]; then
        echo "Error: cljfmt executable not found in $ASSET_NAME" >&2
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
echo "cljfmt version: $CLJFMT_VERSION (EPL-1.0; unmodified upstream binaries)"
echo "Output directory: $VERSION_DIR"
