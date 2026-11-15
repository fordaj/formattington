#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.sh [version] [target...]
#
# Examples:
#   ./update.sh
#   ./update.sh 2.0.0
#   ./update.sh latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq
#
# google-java-format publishes GraalVM native-image binaries (unsigned,
# single file) for four of six targets since v2.0.0. There is no asset
# for macOS x86_64 or Windows arm64; those platforms keep the PATH
# fallback (a user-managed JDK >= 21 with the six --add-exports flags, or
# the all-deps jar). Upstream does not publish checksums for the native
# binaries, so every asset is hash-pinned in this script instead.

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

# Pinned SHA-256 of every asset this script fetches (the unsigned native
# binaries themselves). Release bumps must re-pin: audit the new upstream
# release, update PINNED_VERSION, and replace the hashes below with
# `shasum -a 256 <asset>`. The script refuses any version other than
# PINNED_VERSION and hard-fails on a hash mismatch, so an unverifiable
# binary is never fetched into a release.
PINNED_VERSION="2.0.0"
PINNED_SHA256_linux_arm64="3193ae2e3a32634a45effc9f575005e7adf202699ab4d9d6488ff1a715588d10"
PINNED_SHA256_linux_x86_64="c351f587e6dd1f11836be7296894ea2ab3bd91847dbd14101f4f04d390380fe9"
PINNED_SHA256_macos_arm64="04b13ad486bcc29ef64f7168c2d3eb330e73ae458610ba5bcf44f6018b655dc4"
PINNED_SHA256_windows_x86_64="6e757021d3b5a8133ca3b5b1915c4cf38dc46ad0886bc9059fcc2ccb7b6036e5"

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

if [[ "$GJF_VERSION" != "$PINNED_VERSION" ]]; then
    echo "Error: this script pins google-java-format $PINNED_VERSION, but the release resolved to $GJF_VERSION." >&2
    echo "Re-pin PINNED_VERSION and the hashes in this script after auditing the new release." >&2
    exit 1
fi

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
echo "google-java-format version: $GJF_VERSION"
echo "Output directory: $VERSION_DIR"
