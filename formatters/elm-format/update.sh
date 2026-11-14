#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 0.8.8
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq, tar, unzip, sha256sum or shasum
#
# elm-format ships static Haskell binaries for five of six targets:
# Linux x86_64, Linux arm64, macOS x86_64, macOS arm64, and Windows
# x86_64. There is no Windows arm64 asset; that platform keeps the
# PATH fallback. The mac-arm64 binary is large (~42 MB) — possibly an
# unstripped build; that is upstream's artifact, redistributed
# unmodified. Upstream signs assets with GPG (.asc); no keyring is
# pinned here, so the version is pinned and SHA-256s are printed for
# the release notes instead.

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

# Map platform/architecture names to GitHub release assets.
get_asset_name() {
    case "$1" in
    linux-arm64) echo "elm-format-$VERSION_NUMBER-linux-aarch64.tgz" ;;
    linux-x86_64) echo "elm-format-$VERSION_NUMBER-linux-x64.tgz" ;;
    macos-arm64) echo "elm-format-$VERSION_NUMBER-mac-arm64.tgz" ;;
    macos-x86_64) echo "elm-format-$VERSION_NUMBER-mac-x64.tgz" ;;
    windows-x86_64) echo "elm-format-$VERSION_NUMBER-win-x64.zip" ;;
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

REPO="avh4/elm-format"

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
ELM_FORMAT_VERSION="${TAG_NAME#v}"
VERSION_NUMBER="$ELM_FORMAT_VERSION"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$ELM_FORMAT_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading elm-format $ELM_FORMAT_VERSION..."

# Create an isolated temporary directory.
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

for TARGET in "${TARGETS[@]}"; do
    ASSET_NAME="$(get_asset_name "$TARGET")"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG_NAME/$ASSET_NAME"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/elm-format"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    ARCHIVE="$WORK_DIR/$ASSET_NAME"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$ARCHIVE"

    if [[ "$SHA256_COMMAND" == "sha256sum" ]]; then
        ACTUAL_SHA256="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
    else
        ACTUAL_SHA256="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
    fi
    echo "     sha256: $ACTUAL_SHA256  $ASSET_NAME"

    # Extract the single elm-format executable.
    EXTRACT_DIR="$WORK_DIR/extract-$TARGET"
    mkdir -p "$EXTRACT_DIR"

    case "$ASSET_NAME" in
    *.tgz)
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
        FOUND="$(find "$EXTRACT_DIR" -type f -name 'elm-format.exe' -print | head -n 1)"
    else
        FOUND="$(find "$EXTRACT_DIR" -type f -name 'elm-format' -print | head -n 1)"
    fi

    if [[ -z "$FOUND" ]]; then
        echo "Error: elm-format executable not found in $ASSET_NAME" >&2
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
echo "elm-format version: $ELM_FORMAT_VERSION (BSD-3-Clause)"
echo "Output directory: $VERSION_DIR"
