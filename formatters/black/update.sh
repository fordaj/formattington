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
#   ./update.bash latest macos-x86_64   (builds from source; needs a native Intel Mac)
#
# Requirements: curl, jq (plus git, uv, xcrun, and lipo on a native
# Intel Mac, for the locally built macOS x86_64 binary)

VERSION="${1:-latest}"
shift || true

# Resolve paths.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPT_DIR}"

# Black publishes no standalone macOS x86_64 binary, so on a native Intel
# Mac that target is built from source with build_macos-x64.sh instead of
# downloaded.
on_macos_x86_64_host() {
    [[ "$(uname -s)" == "Darwin" && "$(uname -m)" == "x86_64" ]]
}

# Track whether the target list came from the command line or from the
# default; only an explicit request must fail hard on a target this host
# cannot provide.
EXPLICIT=0

# Default download targets.
if [[ $# -eq 0 ]]; then
    TARGETS=(
        linux-arm64
        linux-x86_64
        macos-arm64
        windows-arm64
        windows-x86_64
    )
    if on_macos_x86_64_host; then
        TARGETS+=(macos-x86_64)
    fi
else
    EXPLICIT=1
    TARGETS=("$@")
fi

# Pinned SHA-256 of every asset this script downloads. Release bumps must
# re-pin: audit the new upstream release, update PINNED_VERSION, and
# replace the hashes below with `shasum -a 256 <asset>`. The script
# refuses any version other than PINNED_VERSION and hard-fails on a hash
# mismatch, so an unverifiable binary is never fetched into a release.
# (The macOS x86_64 target is built from verified source instead — see
# build_macos-x64.sh, which pins the tag's commit hash.)
PINNED_VERSION="26.10.1"
PINNED_SHA256_linux_arm64="a470befedf349ea38391b00f0a10b84bdca645cc9df0cd5db70d4cf7a4efa2b5"
PINNED_SHA256_linux_x86_64="94324773c2d112e7a4d62052ac9d7ed811ff1c826599f6d394e7b65d16ab1278"
PINNED_SHA256_macos_arm64="1336c90c027f854e9806ca8476c4f4ce84d3a3eb4bc5495797f0446ee253d2c2"
PINNED_SHA256_windows_arm64="8b2233f85d92cef5f916c44b6b9b51530dd3502c6a396ecf8ddda807544b3d0c"
PINNED_SHA256_windows_x86_64="9901f869bcfe0df3304ee7a42913dc3a8f6d85ad4feff6f6c30c26bfd7063dd4"

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

# macos-x86_64 is intentionally absent from this map; it is built from
# source by build_macos-x64.sh (see below).
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

if [[ "$RELEASE_VERSION" != "$PINNED_VERSION" ]]; then
    echo "Error: this script pins black $PINNED_VERSION, but the release resolved to $RELEASE_VERSION." >&2
    echo "Re-pin PINNED_VERSION and the hashes in this script after auditing the new release." >&2
    exit 1
fi

VERSION_DIR="$OUTPUT_DIR/v$RELEASE_VERSION"
mkdir -p "$VERSION_DIR"

echo "Black version: $RELEASE_VERSION"
echo

# Build the standalone macOS x86_64 executable from source on a native
# Intel Mac, into $VERSION_DIR/macos-x86_64/black.
build_macos_x86_64() {
    local platform="macos-x86_64"

    if ! on_macos_x86_64_host; then
        if [[ "$EXPLICIT" -eq 1 ]]; then
            echo "Error: black does not publish a standalone macOS x86_64 binary; build_macos-x64.sh needs a native Intel Mac." >&2
            exit 1
        fi
        echo "  -> macos-x86_64"
        echo "     Skipping: black does not publish a macOS x86_64 binary; the build script needs a native Intel Mac."
        return 0
    fi

    echo "  -> macos-x86_64 (building from source)"
    OUTPUT_DIR="$OUTPUT_DIR" "$SCRIPT_DIR/build_macos-x64.sh" "$RELEASE_VERSION"

    if [[ ! -x "$VERSION_DIR/$platform/black" ]]; then
        echo "Error: the macos-x86_64 build did not produce $VERSION_DIR/$platform/black" >&2
        exit 1
    fi
    echo "     Created: $VERSION_DIR/$platform/black"
}

# Download (or build, for macos-x86_64) each requested platform.
for TARGET in "${TARGETS[@]}"; do

    if [[ "$TARGET" == "macos-x86_64" ]]; then
        build_macos_x86_64
        continue
    fi

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
echo "Black version: $RELEASE_VERSION"
echo "Output directory: $VERSION_DIR"
