#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./update.bash [version] [target...]
#
# Examples:
#   ./update.bash
#   ./update.bash 23
#   ./update.bash 23.1.5
#   ./update.bash latest windows-x86_64 macos-arm64
#
# Requirements: curl, jq, sha512sum or shasum

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

# Map internal target names to index.json platform names.
get_platform() {
    case "$1" in
    linux-arm64) echo "linux-arm64" ;;
    linux-x86_64) echo "linux-amd64" ;;
    macos-arm64) echo "macos-arm64" ;;
    macos-x86_64) echo "macos-amd64" ;;
    windows-arm64) echo "windows-arm64" ;;
    windows-x86_64) echo "windows-amd64" ;;
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

if command -v sha512sum >/dev/null 2>&1; then
    SHA512_COMMAND="sha512sum"
elif command -v shasum >/dev/null 2>&1; then
    SHA512_COMMAND="shasum"
else
    echo "Error: sha512sum or shasum is required." >&2
    exit 1
fi

# Fetch upstream version and asset metadata.
INDEX_URL="https://cpp-linter.github.io/clang-tools-static-binaries/index.json"

echo "Resolving clang-format release..."

INDEX_JSON="$(curl -fsSL "$INDEX_URL")"

if [[ "$VERSION" == "latest" ]]; then
    REQUESTED_VERSION="$(jq -er '.latest_version' <<<"$INDEX_JSON")"
else
    REQUESTED_VERSION="${VERSION#v}"
fi

# Resolve the requested version.
VERSION_INFO="$(
    jq -cer --arg version "$REQUESTED_VERSION" '
        .versions
        | to_entries[]
        | select(
            .key == $version
            or .value.version == $version
        )
        | .value
    ' <<<"$INDEX_JSON"
)" || {
    echo "Error: clang-format version not found: $REQUESTED_VERSION" >&2
    exit 1
}

CLANG_FORMAT_VERSION="$(jq -er '.version' <<<"$VERSION_INFO")"

# Create version-specific output directory.
VERSION_DIR="$OUTPUT_DIR/v$CLANG_FORMAT_VERSION"
mkdir -p "$VERSION_DIR"

echo "Downloading clang-format $CLANG_FORMAT_VERSION..."

for TARGET in "${TARGETS[@]}"; do
    PLATFORM="$(get_platform "$TARGET")"

    ASSET_INFO="$(
        jq -cer --arg platform "$PLATFORM" '
            .tools["clang-format"][$platform]
        ' <<<"$VERSION_INFO"
    )" || {
        echo "Error: No clang-format binary available for $TARGET" >&2
        exit 1
    }

    DOWNLOAD_URL="$(jq -er '.url' <<<"$ASSET_INFO")"
    EXPECTED_SHA512="$(jq -er '.sha512' <<<"$ASSET_INFO")"

    TARGET_DIR="$VERSION_DIR/$TARGET"
    mkdir -p "$TARGET_DIR"

    OUTPUT="$TARGET_DIR/clang-format"

    if [[ "$TARGET" == windows-* ]]; then
        OUTPUT+=".exe"
    fi

    echo "  -> $TARGET"

    curl -fL --retry 3 "$DOWNLOAD_URL" -o "$OUTPUT"

    # Verify SHA-512 checksum.
    if [[ "$SHA512_COMMAND" == "sha512sum" ]]; then
        ACTUAL_SHA512="$(sha512sum "$OUTPUT" | awk '{print $1}')"
    else
        ACTUAL_SHA512="$(shasum -a 512 "$OUTPUT" | awk '{print $1}')"
    fi

    if [[ "$ACTUAL_SHA512" != "$EXPECTED_SHA512" ]]; then
        echo "Error: SHA-512 verification failed for $TARGET" >&2
        rm -f "$OUTPUT"
        exit 1
    fi

    if [[ "$TARGET" != windows-* ]]; then
        chmod +x "$OUTPUT"
    fi

    echo "     Verified: $OUTPUT"
done

echo
echo "Download complete!"
echo "clang-format version: $CLANG_FORMAT_VERSION"
echo "Output directory: $VERSION_DIR"
