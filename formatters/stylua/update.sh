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
# publish checksums, so every asset is hash-pinned in this script instead (musl variants for linux).

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

# Pinned SHA-256 of every asset this script fetches (of the zip archives
# themselves; the linux targets pin the musl variants this script
# prefers). Release bumps must re-pin: audit the new upstream release,
# update PINNED_VERSION, and replace the hashes below with `shasum -a
# 256 <asset>`. The script refuses any version other than PINNED_VERSION
# and hard-fails on a hash mismatch, so an unverifiable binary is never
# fetched into a release.
PINNED_VERSION="2.6.0"
PINNED_SHA256_linux_arm64="278495597e3827417f3af7b9ad38011ad4a5ec725503553160ab69b7301059bb"
PINNED_SHA256_linux_x86_64="48ee7db28346f96f4bd545e190ab080a88a7b5a664e97649cf5d5c60426f402d"
PINNED_SHA256_macos_arm64="dab9b66d340ca5ffc8225211f1dcf89b84783e09b8cbec8c518497eba81f3c42"
PINNED_SHA256_macos_x86_64="90779aa873837b2dff8eb3361bd27950303914c24265a59cae0a0d0629d71585"
PINNED_SHA256_windows_x86_64="fd13b4014e3b779c244bbc03f2c282e357f15bc526de6a6fa5ecbc447538313c"

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

if [[ "$STYLUA_VERSION" != "$PINNED_VERSION" ]]; then
    echo "Error: this script pins stylua $PINNED_VERSION, but the release resolved to $STYLUA_VERSION." >&2
    echo "Re-pin PINNED_VERSION and the hashes in this script after auditing the new release." >&2
    exit 1
fi

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

    local_key="${TARGET//-/_}"
    local_var="PINNED_SHA256_${local_key}"
    sha256_check "$ARCHIVE" "${!local_var}" || exit 1

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
