#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.sh [version] [target...]
#
# Examples:
#   ./update.sh
#   ./update.sh 0.21.0.0
#   ./update.sh latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq, unzip
#
# Fourmolu ships Haskell binaries for four of six targets: macOS arm64,
# macOS x86_64, Linux arm64, Linux x86_64. There is no Windows asset;
# Windows stays PATH-only (ghcup install fourmolu / cabal install
# fourmolu). Linux builds are glibc-dynamic — musl/Alpine users must
# install via PATH. The darwin-arm64 asset is suspiciously large
# (~23 MB vs ~7 MB x86_64); it is redistributed unmodified. Upstream
# publishes no checksums, so every asset is hash-pinned in this script.

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Default download targets. Windows is intentionally absent (no asset).
if [[ $# -eq 0 ]]; then
    TARGETS=(
        linux-arm64
        linux-x86_64
        macos-arm64
        macos-x86_64
    )
else
    TARGETS=("$@")
fi

# Pinned SHA-256 of every asset this script fetches (of the zip archives
# themselves). Release bumps must re-pin: audit the new upstream release,
# update PINNED_VERSION, and replace the hashes below with `shasum -a
# 256 <asset>`. The script refuses any version other than PINNED_VERSION
# and hard-fails on a hash mismatch, so an unverifiable binary is never
# fetched into a release.
PINNED_VERSION="0.21.0.0"
PINNED_SHA256_linux_arm64="8ef665cab33041a9ef702a4f6d7879b0e5988a71eb790fd0841c5c8d07789988"
PINNED_SHA256_linux_x86_64="aadf6931653df7b96f52014cf8eb4285a3e8778cd0b531499fc3ee013e1e00c7"
PINNED_SHA256_macos_arm64="e62da50e4fafc9b0a2ec437192b13b690f0d6ea663d18d97cc14a40c30f7a04c"
PINNED_SHA256_macos_x86_64="4799cf43ced9b9b63b01879df110ec67c89c8d3a7cee0188427c17fe388e2f10"

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
    linux-arm64) echo "fourmolu-$VERSION_NUMBER-linux-arm64.zip" ;;
    linux-x86_64) echo "fourmolu-$VERSION_NUMBER-linux-x86_64.zip" ;;
    macos-arm64) echo "fourmolu-$VERSION_NUMBER-darwin-arm64.zip" ;;
    macos-x86_64) echo "fourmolu-$VERSION_NUMBER-darwin-x86_64.zip" ;;
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

REPO="fourmolu/fourmolu"

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
FOURMOLU_VERSION="${TAG_NAME#v}"

if [[ "$FOURMOLU_VERSION" != "$PINNED_VERSION" ]]; then
    echo "Error: this script pins fourmolu $PINNED_VERSION, but the release resolved to $FOURMOLU_VERSION." >&2
    echo "Re-pin PINNED_VERSION and the hashes in this script after auditing the new release." >&2
    exit 1
fi
VERSION_NUMBER="$FOURMOLU_VERSION"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$FOURMOLU_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading fourmolu $FOURMOLU_VERSION..."

# Create an isolated temporary directory.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/fourmolu"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    ARCHIVE="$WORK_DIR/$ASSET_NAME"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$ARCHIVE"

    local_key="${TARGET//-/_}"
    local_var="PINNED_SHA256_${local_key}"
    sha256_check "$ARCHIVE" "${!local_var}" || exit 1

    # Extract the top-level fourmolu executable.
    EXTRACT_DIR="$WORK_DIR/extract-$TARGET"
    mkdir -p "$EXTRACT_DIR"

    unzip -q "$ARCHIVE" -d "$EXTRACT_DIR"

    FOUND="$(find "$EXTRACT_DIR" -type f -name 'fourmolu' -print | head -n 1)"

    if [[ -z "$FOUND" ]]; then
        echo "Error: fourmolu executable not found in $ASSET_NAME" >&2
        rm -rf "$TARGET_DIR"
        exit 1
    fi

    cp "$FOUND" "$OUTPUT"

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Created: $OUTPUT"
done

echo
echo "Download complete!"
echo "fourmolu version: $FOURMOLU_VERSION (BSD-3-Clause)"
echo "Output directory: $VERSION_DIR"
