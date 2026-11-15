#!/usr/bin/env bash
# update.bash — refresh bundled barnacle binaries from fordaj/barnacle releases.
#
# Downloads the latest release's per-platform executables via gh, verifies
# SHA256 checksums, extracts each binary into tools/barnacle/<tag>/<platform>/
# (platform is the bundled <os>-<arch> layout: macos/linux/windows +
# x86_64/arm64, so darwin-amd64 -> macos-x86_64, etc.), and deletes old
# version folders. Only the binary is kept; the upstream LICENSE (MIT) is
# retained as <tag>/LICENSE (the archives all carry the same file — the
# script verifies they are identical, or fails); README.md is discarded.
# Old v*/ folders are deleted only after the new version is fully staged,
# so a failed download leaves the previous bundle intact.
#
# Usage:
#   tools/barnacle/update.bash              # download latest, replace old versions
#   tools/barnacle/update.bash --dry-run    # resolve tag + list assets, change nothing
#   tools/barnacle/update.bash --help       # usage
#
# Prerequisites: gh (https://cli.github.com/), tar, and unzip or 7z/7zz
# (p7zip) for the Windows .zip assets, and (for the real update) shasum or
# sha256sum for checksum verification. The script resolves its own
# directory, so it can be run from anywhere. Logging uses
# log <level> <message> from scripts/entry.sh (logamuffin, with printf
# fallback until tools/logamuffin is populated).
#
# Layout after a successful run (example for v0.0.1.0.5):
#   tools/barnacle/
#     update.bash
#     v0.0.1.0.5/
#       macos-arm64/barnacle
#       macos-x86_64/barnacle
#       linux-arm64/barnacle
#       linux-x86_64/barnacle
#       windows-arm64/barnacle.exe
#       windows-x86_64/barnacle.exe
#
# Env:
#   BARNACLE_REPO  Repository slug (default: fordaj/barnacle)
#   BARNACLE_TAG   Pinned tag (default: latest release)

set -euo pipefail

# Logging via logamuffin (log <level> <message> from scripts/entry.sh,
# which pulls in the env cascade and defines log(), with a printf fallback
# until tools/logamuffin is populated). Source the entry point first so
# every later message goes through log().
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT_FOR_ENV=$(cd -- "$SCRIPT_DIR/../.." &>/dev/null && pwd)
if [[ ! -f "$REPO_ROOT_FOR_ENV/scripts/entry.sh" ]]; then
    printf 'update.bash: error: script entry point not found: %s/scripts/entry.sh\n' "$REPO_ROOT_FOR_ENV" >&2
    exit 1
fi
# shellcheck disable=SC1091
. "$REPO_ROOT_FOR_ENV/scripts/entry.sh"

DRY_RUN=0
TAG="${BARNACLE_TAG:-}"
REPO="${BARNACLE_REPO:-fordaj/barnacle}"

usage() {
    cat <<'EOF'
Usage: tools/barnacle/update.bash [--dry-run] [--tag TAG] [--repo REPO]

Download the latest Barnacle release from GitHub via gh, extract each
platform binary into tools/barnacle/<tag>/<platform>/ (platform is the
bundled <os>-<arch> layout, e.g. darwin-arm64 -> macos-arm64), and delete
old version folders. Only the binary is kept.

Flags:
  --dry-run, -n   Resolve tag + list planned assets, change nothing.
  --tag TAG       Pin a specific release tag (default: latest release).
  --repo REPO     Repository slug (default: fordaj/barnacle).
  --help, -h      Show this help.

Env:
  BARNACLE_REPO  Repository slug (overridden by --repo).
  BARNACLE_TAG   Pinned tag (overridden by --tag).
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
    --dry-run | -n)
        DRY_RUN=1
        shift
        ;;
    --help | -h)
        usage
        exit 0
        ;;
    --tag)
        [[ $# -ge 2 ]] || {
            log ERROR "--tag requires a value"
            usage >&2
            exit 2
        }
        TAG="$2"
        shift 2
        ;;
    --tag=*)
        TAG="${1#--tag=}"
        shift
        ;;
    --repo)
        [[ $# -ge 2 ]] || {
            log ERROR "--repo requires a value"
            usage >&2
            exit 2
        }
        REPO="$2"
        shift 2
        ;;
    --repo=*)
        REPO="${1#--repo=}"
        shift
        ;;
    *)
        log ERROR "unknown argument: $1"
        usage >&2
        exit 2
        ;;
    esac
done

[[ -n "$REPO" ]] || {
    log ERROR "repository must not be empty"
    exit 2
}

die() {
    log ERROR "$*" || true
    exit 1
}
need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1 ($2)"
}

need_cmd gh "https://cli.github.com/"
need_cmd tar "https://www.gnu.org/software/tar/"

if [[ "$DRY_RUN" -ne 1 ]]; then
    if command -v unzip >/dev/null 2>&1; then
        UNZIP=unzip
    elif command -v 7zz >/dev/null 2>&1; then
        UNZIP=7zz
    elif command -v 7z >/dev/null 2>&1; then
        UNZIP=7z
    else
        die "required command not found: unzip or 7z/7zz (brew install unzip p7zip / apt-get install unzip p7zip-full)"
    fi
fi

BARNACLE_ROOT="$SCRIPT_DIR"
[[ -d "$BARNACLE_ROOT" ]] || die "barnacle bundle directory not found: $BARNACLE_ROOT"
log INFO "barnacle bundle directory: $BARNACLE_ROOT"
log INFO "repository: $REPO"

if [[ -z "$TAG" ]]; then
    TAG="$(gh release view --repo "$REPO" --json tagName -q .tagName)" ||
        die "could not determine latest release from $REPO (is gh authenticated and the repo visible?)"
fi
[[ -n "$TAG" ]] || die "could not determine release tag from $REPO"
if ! [[ "$TAG" =~ ^v[0-9]+(\.[0-9]+)+$ ]]; then
    die "invalid tag '$TAG' (expected vX.Y.Z...)"
fi
log INFO "release tag: $TAG"

# parse_asset_to_platform <asset> sets PLATFORM (e.g.
# barnacle-macos-arm64.tar.gz -> macos-arm64) or returns 1 for unknown
# values. Sets globals to avoid subshell-die pitfalls. Asset names append
# the platform after "barnacle-" and arch may itself contain "_" (x86_64),
# so the suffix is matched as a whole instead of splitting on the last "-".
parse_asset_to_platform() {
    local asset="$1" stem
    case "$asset" in
    barnacle-*.tar.gz) stem="${asset%.tar.gz}" ;;
    barnacle-*.zip) stem="${asset%.zip}" ;;
    *) return 1 ;;
    esac
    case "${stem#barnacle-}" in
    macos-x86_64 | darwin-amd64) PLATFORM="macos-x86_64" ;;
    macos-arm64 | darwin-arm64) PLATFORM="macos-arm64" ;;
    linux-x86_64 | linux-amd64) PLATFORM="linux-x86_64" ;;
    linux-arm64) PLATFORM="linux-arm64" ;;
    windows-x86_64 | windows-amd64) PLATFORM="windows-x86_64" ;;
    windows-arm64) PLATFORM="windows-arm64" ;;
    *) return 1 ;;
    esac
}

if [[ "$DRY_RUN" -eq 1 ]]; then
    log INFO "DRY RUN: no download, extraction, or deletion will happen"
    log INFO "release $TAG assets:"
    assets_list=$(gh release view "$TAG" --repo "$REPO" --json assets -q '.assets[].name' ||
        die "could not list assets for $TAG in $REPO")
    [[ -n "$assets_list" ]] || die "no assets found for $TAG in $REPO"
    asset_count=0
    while IFS= read -r asset; do
        [[ -n "$asset" ]] || continue
        case "$asset" in
        barnacle-*.tar.gz | barnacle-*.zip)
            asset_count=$((asset_count + 1))
            if parse_asset_to_platform "$asset"; then
                log INFO "  $asset -> $TAG/$PLATFORM"
            else
                log INFO "  $asset -> unknown platform (would fail)"
            fi
            ;;
        SHA256SUMS.txt | checksums.txt | *.sha256)
            log INFO "  $asset (checksums)"
            ;;
        *)
            log INFO "  $asset (ignored)"
            ;;
        esac
    done <<<"$assets_list"
    ((asset_count > 0)) || die "no barnacle archives found for $TAG in $REPO"
    shopt -s nullglob
    old_found=0
    for dir in "$BARNACLE_ROOT"/v*/; do
        [[ -d "$dir" ]] || continue
        dir_name="$(basename "$dir")"
        if [[ "$dir_name" != "$TAG" ]]; then
            log INFO "would remove old $dir_name"
            old_found=1
        fi
    done
    if [[ "$old_found" -eq 0 ]]; then
        log INFO "no old version folders to remove"
    fi
    log INFO "dry run complete: would update $BARNACLE_ROOT/$TAG"
    exit 0
fi

# --- Real update: download to a temp dir, stage the bundle, then swap. ---
TMP_DIR=""
STAGE_DIR=""
cleanup() {
    if [[ -n "${TMP_DIR:-}" && -d "$TMP_DIR" ]]; then
        rm -rf -- "$TMP_DIR"
    fi
    if [[ -n "${STAGE_DIR:-}" && -d "$STAGE_DIR" ]]; then
        rm -rf -- "$STAGE_DIR"
    fi
}
trap cleanup EXIT

TMP_DIR="$(mktemp -d)"
STAGE_DIR="$(mktemp -d "$BARNACLE_ROOT/.tmp.XXXXXX")"

log INFO "downloading $TAG assets from $REPO..."
gh release download "$TAG" --repo "$REPO" --dir "$TMP_DIR" --pattern 'barnacle-*.tar.gz' --pattern 'barnacle-*.zip' --pattern 'SHA256SUMS.txt' --pattern 'checksums.txt' --clobber

shopt -s nullglob
archives=("$TMP_DIR"/barnacle-*.tar.gz "$TMP_DIR"/barnacle-*.zip)
((${#archives[@]} > 0)) || die "no barnacle archives downloaded from $REPO $TAG"
log INFO "downloaded ${#archives[@]} archives"

checksums_file=""
for cand in "$TMP_DIR/SHA256SUMS.txt" "$TMP_DIR/checksums.txt"; do
    if [[ -f "$cand" ]]; then
        checksums_file="$cand"
        break
    fi
done
if [[ -n "$checksums_file" ]]; then
    log INFO "verifying checksums ($(basename "$checksums_file"))..."
    if command -v shasum >/dev/null 2>&1; then
        (cd "$TMP_DIR" && shasum -a 256 -c "$(basename "$checksums_file")") || die "checksum verification failed for $TAG"
    elif command -v sha256sum >/dev/null 2>&1; then
        (cd "$TMP_DIR" && sha256sum -c "$(basename "$checksums_file")") || die "checksum verification failed for $TAG"
    else
        die "$(basename "$checksums_file") present but neither shasum nor sha256sum found"
    fi
    log INFO "checksums verified"
else
    log WARNING "no SHA256SUMS.txt in $TAG release; skipping verification"
fi

for archive in "${archives[@]}"; do
    base="$(basename "$archive")"
    if ! parse_asset_to_platform "$base"; then
        die "unknown platform in asset '$base' (expected barnacle-<os>-<arch>, such as barnacle-darwin-arm64.tar.gz)"
    fi
    if [[ -e "$STAGE_DIR/$PLATFORM" ]]; then
        die "duplicate asset for platform $PLATFORM (from $base)"
    fi
    log INFO "extracting $base -> $TAG/$PLATFORM"
    extract_dir="$TMP_DIR/extract-$PLATFORM"
    mkdir -p "$extract_dir"
    case "$base" in
    *.tar.gz)
        tar -xzf "$archive" -C "$extract_dir"
        ;;
    *.zip)
        case "$UNZIP" in
        unzip) unzip -q -o "$archive" -d "$extract_dir" ;;
        7zz | 7z) "$UNZIP" x -y "$archive" -o"$extract_dir" >/dev/null ;;
        esac
        ;;
    esac
    bin_src=""
    for cand in \
        "$extract_dir/barnacle-$PLATFORM/barnacle" \
        "$extract_dir/barnacle-$PLATFORM/barnacle.exe" \
        "$extract_dir/barnacle" \
        "$extract_dir/barnacle.exe"; do
        if [[ -f "$cand" ]]; then
            bin_src="$cand"
        fi
    done
    [[ -n "$bin_src" ]] || die "no barnacle binary in $base (expected barnacle-<platform>/barnacle[.exe] or barnacle[.exe])"
    bin_name="$(basename "$bin_src")"
    target_dir="$STAGE_DIR/$PLATFORM"
    mkdir -p "$target_dir"
    cp -p "$bin_src" "$target_dir/$bin_name"
    chmod +x "$target_dir/$bin_name"
    [[ -s "$target_dir/$bin_name" ]] || die "installed binary is empty: $target_dir/$bin_name"
    license_src=""
    for cand in \
        "$extract_dir/barnacle-$PLATFORM/LICENSE" \
        "$extract_dir/LICENSE"; do
        if [[ -f "$cand" ]]; then
            license_src="$cand"
            break
        fi
    done
    [[ -z "$license_src" ]] || cp "$license_src" "$TMP_DIR/LICENSE.$PLATFORM"
    log INFO "installed $PLATFORM/$bin_name (from $base)"
    rm -rf -- "$extract_dir"
done

# Every archive ships the same LICENSE (MIT). Keep one copy as
# <tag>/LICENSE; refuse to guess when they differ.
shopt -s nullglob
license_files=("$TMP_DIR"/LICENSE.*)
if ((${#license_files[@]} > 0)); then
    for lic in "${license_files[@]:1}"; do
        cmp -s "${license_files[0]}" "$lic" ||
            die "upstream LICENSE differs between platforms (${license_files[0]##*/} vs ${lic##*/}); refusing to keep one"
    done
    cp "${license_files[0]}" "$STAGE_DIR/LICENSE"
    log INFO "retained upstream LICENSE as $TAG/LICENSE"
else
    log WARNING "no LICENSE found in any $TAG asset; bundle carries no license text"
fi

TAG_DIR="$BARNACLE_ROOT/$TAG"
rm -rf -- "$TAG_DIR"
mkdir -p "$BARNACLE_ROOT"
mv "$STAGE_DIR" "$TAG_DIR"
STAGE_DIR=""

for dir in "$BARNACLE_ROOT"/v*/; do
    [[ -d "$dir" ]] || continue
    dir_name="$(basename "$dir")"
    if [[ "$dir_name" != "$TAG" ]]; then
        log INFO "removing old $dir_name"
        rm -rf -- "$dir"
    fi
done

# Best-effort host verification: run the bundled binary for this machine.
host_os_raw="$(uname -s | tr '[:upper:]' '[:lower:]')"
host_arch_raw="$(uname -m | tr '[:upper:]' '[:lower:]')"
case "$host_os_raw" in
darwin*) host_os="macos" ;;
linux*) host_os="linux" ;;
msys* | mingw* | cygwin* | windows* | win*) host_os="windows" ;;
*) host_os="$host_os_raw" ;;
esac
case "$host_arch_raw" in
x86_64 | amd64) host_arch="x86_64" ;;
arm64 | aarch64) host_arch="arm64" ;;
*) host_arch="$host_arch_raw" ;;
esac
host_platform="$host_os-$host_arch"
host_bin=""
for cand in "$TAG_DIR/$host_platform/barnacle" "$TAG_DIR/$host_platform/barnacle.exe"; do
    if [[ -x "$cand" ]]; then
        host_bin="$cand"
    fi
done
tag_without_v="${TAG#v}"
if [[ -n "$host_bin" ]]; then
    host_ver="$("$host_bin" --version 2>&1 || true)"
    log INFO "host binary $host_platform reports: $host_ver"
    case "$host_ver" in
    *"$tag_without_v"*) ;;
    *) log WARNING "host binary version '$host_ver' does not contain $tag_without_v" ;;
    esac
else
    log WARNING "host platform $host_platform not in bundle; skipped version check"
fi

log INFO "done: $TAG_DIR now contains:"
for dir in "$TAG_DIR"/*/; do
    [[ -d "$dir" ]] || continue
    log INFO "  $(basename "$dir")"
done
