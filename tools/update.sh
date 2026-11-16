#!/usr/bin/env bash
# update.sh — refresh a bundled helper tool from its GitHub releases.
#
# Shared implementation for the tools/<name>/update.sh wrappers (run those
# instead of this file; they pass the tool name as the first argument and
# keep each tool's CLI, env vars, and pinned repository defaults). The
# wrappers are thin by design: this script holds the whole refresh logic
# so it can be audited and fixed once.
#
# Downloads the latest release's per-platform executables via gh, verifies
# SHA256 checksums when the release ships them, extracts each binary into
# tools/<tool>/<tag>/<platform>/ (platform is the bundled <os>-<arch> layout:
# macos/linux/windows + x86_64/arm64, so darwin-amd64 -> macos-x86_64, etc.),
# and deletes old version folders. Old v*/ folders are deleted only after the
# new version is fully staged, so a failed download leaves the previous
# bundle intact.
#
# Archives are consumed in their platform-native formats: .tar.gz on
# macOS/Linux and .zip on Windows. Legacy .7z assets are still accepted as
# a fallback (requires 7z/7zz) so this script keeps working against releases
# cut before the format change; new releases do not need p7zip at all.
#
# Usage:
#   tools/update.sh <tool>                 # download latest, replace old versions
#   tools/update.sh <tool> --dry-run       # resolve tag + list assets, change nothing
#   tools/update.sh <tool> --tag TAG       # pin a specific release tag
#   tools/update.sh <tool> --repo REPO     # override the repository slug
#   tools/update.sh --help                 # usage
#
# Supported tools: barnacle, formattington-lite, logamuffin, runner-spawner,
# versioner. Each tool ships an update.sh wrapper under tools/<tool>/ that
# passes <tool> to this script. The script resolves its own directory, so
# it can be run from anywhere. Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback).
#
# Layout after a successful run (example for v0.1.2.3, logamuffin):
#   tools/logamuffin/
#     update.sh
#     v0.1.2.3/
#       macos-arm64/logamuffin
#       macos-x86_64/logamuffin
#       linux-arm64/logamuffin
#       linux-x86_64/logamuffin
#       windows-x86_64/logamuffin
#       windows-arm64/logamuffin
#
# Binary name inside the archive defaults to the tool name (formattington-lite
# ships "formattington"); formattington-lite also keeps the release's
# NOTICE.md and LICENSES/ next to the binary, and barnacle keeps the
# upstream MIT LICENSE. runner-spawner refuses to install when the release
# carries no checksums file; the other tools warn and proceed.

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT_FOR_ENV=$(cd -- "$SCRIPT_DIR/.." &>/dev/null && pwd)
if [[ ! -f "$REPO_ROOT_FOR_ENV/scripts/entry.sh" ]]; then
    printf 'update.sh: error: entry point not found: %s/scripts/entry.sh\n' "$REPO_ROOT_FOR_ENV" >&2
    exit 1
fi
# shellcheck disable=SC1091
. "$REPO_ROOT_FOR_ENV/scripts/entry.sh"

TOOL="${1:-}"
if [[ "$TOOL" == "--help" || "$TOOL" == "-h" ]]; then
    TOOL=""
fi
if [[ -z "$TOOL" ]]; then
    cat <<'EOF'
Usage: tools/update.sh <tool> [--dry-run] [--tag TAG] [--repo REPO]

Refresh a bundled helper tool from its GitHub releases. Prefer the
per-tool wrappers (tools/<tool>/update.sh), which pass <tool> for you.

  barnacle           fordaj/barnacle
  formattington-lite fordaj/formattington (assets filtered to *-lite)
  logamuffin         fordaj/logamuffin
  runner-spawner     fordaj/runner-spawner (refuses unverified binaries)
  versioner          fordaj/Versioner

Flags:
  --dry-run, -n   Resolve tag + list planned assets, change nothing.
  --tag TAG       Pin a specific release tag (default: latest release).
  --repo REPO     Repository slug (default: the tool's upstream repo).
  --help, -h      Show this help.

Per-tool environment overrides (overridden by --repo / --tag):
  BARNACLE_REPO, BARNACLE_TAG
  FORMATTINGTON_REPO, FORMATTINGTON_TAG
  LOGAMUFFIN_REPO, LOGAMUFFIN_TAG
  RUNNER_SPAWNER_TOOL_REPO, RUNNER_SPAWNER_TAG
  VERSIONER_REPO, VERSIONER_TAG
EOF
    exit 0
fi
shift

# Per-tool configuration: default repository, binary name inside the
# archives, asset-name suffix to keep ("" keeps everything), environment
# variable names for the overrides, whether a missing checksums file is
# fatal, and which extra files to keep next to the binary.
case "$TOOL" in
barnacle)
    REPO_DEFAULT="fordaj/barnacle"
    BIN="barnacle"
    ASSET_SUFFIX=""
    ENV_REPO="BARNACLE_REPO"
    ENV_TAG="BARNACLE_TAG"
    REQUIRE_CHECKSUMS=0
    KEEP_EXTRAS="license"
    KEEP_EXTRAS_DESC=" (plus the upstream MIT LICENSE)"
    ;;
formattington-lite)
    REPO_DEFAULT="fordaj/formattington"
    BIN="formattington"
    ASSET_SUFFIX="-lite"
    ENV_REPO="FORMATTINGTON_REPO"
    ENV_TAG="FORMATTINGTON_TAG"
    REQUIRE_CHECKSUMS=0
    KEEP_EXTRAS="notice"
    KEEP_EXTRAS_DESC=", plus NOTICE.md and LICENSES/"
    ;;
logamuffin)
    REPO_DEFAULT="fordaj/logamuffin"
    BIN="logamuffin"
    ASSET_SUFFIX=""
    ENV_REPO="LOGAMUFFIN_REPO"
    ENV_TAG="LOGAMUFFIN_TAG"
    REQUIRE_CHECKSUMS=0
    KEEP_EXTRAS=""
    KEEP_EXTRAS_DESC=""
    ;;
runner-spawner)
    REPO_DEFAULT="fordaj/runner-spawner"
    BIN="runner-spawner"
    ASSET_SUFFIX=""
    ENV_REPO="RUNNER_SPAWNER_TOOL_REPO"
    ENV_TAG="RUNNER_SPAWNER_TAG"
    REQUIRE_CHECKSUMS=1
    KEEP_EXTRAS=""
    KEEP_EXTRAS_DESC=""
    ;;
versioner)
    REPO_DEFAULT="fordaj/Versioner"
    BIN="versioner"
    ASSET_SUFFIX=""
    ENV_REPO="VERSIONER_REPO"
    ENV_TAG="VERSIONER_TAG"
    REQUIRE_CHECKSUMS=0
    KEEP_EXTRAS=""
    KEEP_EXTRAS_DESC=""
    ;;
*)
    printf 'update.sh: error: unknown tool %q (expected barnacle, formattington-lite, logamuffin, runner-spawner, or versioner)\n' "$TOOL" >&2
    exit 2
    ;;
esac

DRY_RUN=0
TAG="${!ENV_TAG:-}"
REPO="${!ENV_REPO:-$REPO_DEFAULT}"
TOOL_ROOT="$REPO_ROOT_FOR_ENV/tools/$TOOL"

usage() {
    cat <<EOF
Usage: tools/$TOOL/update.sh [--dry-run] [--tag TAG] [--repo REPO]

Download the latest $TOOL release from GitHub via gh, extract each
platform binary into tools/$TOOL/<tag>/<platform>/ (platform is the
bundled <os>-<arch> layout, e.g. darwin-amd64 -> macos-x86_64), and delete old
version folders. Only the binary is kept${KEEP_EXTRAS_DESC}.

Flags:
  --dry-run, -n   Resolve tag + list planned assets, change nothing.
  --tag TAG       Pin a specific release tag (default: latest release).
  --repo REPO     Repository slug (default: $REPO_DEFAULT).
  --help, -h      Show this help.

Env:
  $ENV_REPO  Repository slug (overridden by --repo).
  $ENV_TAG   Pinned tag (overridden by --tag).
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
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need_cmd gh
need_cmd tar

[[ -d "$TOOL_ROOT" ]] || die "$TOOL bundle directory not found: $TOOL_ROOT"
log INFO "$TOOL bundle directory: $TOOL_ROOT"
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

# Matches an os-arch suffix (optionally followed by -lite/-full) at the end
# of an asset stem and maps it onto the bundled layout.
parse_stem_to_platform() {
    local stem="$1"
    if [[ "$stem" =~ [-_](darwin-amd64|darwin-arm64|darwin-x86_64|macos-x86_64|macos-arm64|macos-x64|macos-amd64|linux-x86_64|linux-amd64|linux-x64|linux-arm64|windows-x86_64|windows-amd64|windows-x64|windows-arm64)(-lite|-full)?$ ]]; then
        case "${BASH_REMATCH[1]}" in
        darwin-amd64 | darwin-x86_64 | macos-x86_64 | macos-x64 | macos-amd64) PLATFORM="macos-x86_64" ;;
        darwin-arm64 | macos-arm64) PLATFORM="macos-arm64" ;;
        linux-x86_64 | linux-amd64 | linux-x64) PLATFORM="linux-x86_64" ;;
        linux-arm64) PLATFORM="linux-arm64" ;;
        windows-x86_64 | windows-amd64 | windows-x64) PLATFORM="windows-x86_64" ;;
        windows-arm64) PLATFORM="windows-arm64" ;;
        *) return 1 ;;
        esac
        return 0
    fi
    return 1
}

strip_archive_ext() {
    local name="$1"
    case "$name" in
    *.tar.gz) printf '%s' "${name%.tar.gz}" ;;
    *.zip) printf '%s' "${name%.zip}" ;;
    *.7z) printf '%s' "${name%.7z}" ;;
    *) printf '%s' "$name" ;;
    esac
}

# keep_asset reports whether an archive's stem passes the tool's asset
# suffix filter. logamuffin/versioner/barnacle/runner-spawner keep every
# platform archive; formattington-lite keeps only the *-lite ones.
keep_asset() {
    local stem="$1"
    if [[ -n "$ASSET_SUFFIX" && "$stem" != *"$ASSET_SUFFIX" ]]; then
        return 1
    fi
    return 0
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
        *.tar.gz | *.zip | *.7z)
            stem="$(strip_archive_ext "$asset")"
            if ! keep_asset "$stem"; then
                log INFO "  $asset (ignored: not a $ASSET_SUFFIX archive)"
                continue
            fi
            asset_count=$((asset_count + 1))
            if parse_stem_to_platform "$stem"; then
                log INFO "  $asset -> $TAG/$PLATFORM"
            else
                log INFO "  $asset -> unknown platform (would fail)"
            fi
            ;;
        checksums.txt | SHA256SUMS.txt)
            log INFO "  $asset (checksums)"
            ;;
        *)
            log INFO "  $asset (ignored)"
            ;;
        esac
    done <<<"$assets_list"
    ((asset_count > 0)) || die "no platform archives found for $TAG in $REPO"
    shopt -s nullglob
    old_found=0
    for dir in "$TOOL_ROOT"/v*/; do
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
    log INFO "dry run complete: would update $TOOL_ROOT/$TAG"
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
STAGE_DIR="$(mktemp -d "$TOOL_ROOT/.tmp.XXXXXX")"

log INFO "downloading $TAG assets from $REPO..."
gh release download "$TAG" --repo "$REPO" --dir "$TMP_DIR" --pattern '*.tar.gz' --pattern '*.zip' --pattern '*.7z' --pattern 'checksums.txt' --pattern 'SHA256SUMS.txt' --clobber

shopt -s nullglob
archives=()
for archive in "$TMP_DIR"/*.tar.gz "$TMP_DIR"/*.zip "$TMP_DIR"/*.7z; do
    [[ -e "$archive" ]] || continue
    stem="$(strip_archive_ext "$(basename "$archive")")"
    if ! keep_asset "$stem"; then
        continue
    fi
    archives+=("$archive")
done
((${#archives[@]} > 0)) || die "no platform archives downloaded from $REPO $TAG"
log INFO "downloaded ${#archives[@]} platform archives"

checksums_file=""
for candidate in checksums.txt SHA256SUMS.txt; do
    if [[ -f "$TMP_DIR/$candidate" ]]; then
        checksums_file="$TMP_DIR/$candidate"
        break
    fi
done
if [[ -n "$checksums_file" ]]; then
    log INFO "verifying checksums ($(basename "$checksums_file"))..."
    # The checksums file may be cumulative across releases (or list every
    # uploaded asset), so verify only the archives downloaded for this run.
    checksum_tool=""
    if command -v shasum >/dev/null 2>&1; then
        checksum_tool="shasum -a 256"
    elif command -v sha256sum >/dev/null 2>&1; then
        checksum_tool="sha256sum"
    fi
    [[ -n "$checksum_tool" ]] || die "checksums file present but neither shasum nor sha256sum found"
    verify_list="$TMP_DIR/.verify-list"
    : >"$verify_list"
    for archive in "${archives[@]}"; do
        base="$(basename "$archive")"
        grep -F " $base" "$checksums_file" >>"$verify_list" ||
            die "no checksum entry for downloaded asset $base in $(basename "$checksums_file")"
    done
    # Entries are "<sha>  <name>" (or with a leading *), relative to $TMP_DIR.
    (cd "$TMP_DIR" && $checksum_tool -c "$verify_list") || die "checksum verification failed for $TAG"
    log INFO "checksums verified"
else
    if [[ "$REQUIRE_CHECKSUMS" -eq 1 ]]; then
        die "no checksums.txt/SHA256SUMS.txt in $TAG release; refusing to install unverified binaries"
    fi
    log WARNING "no checksums in $TAG release; skipping verification"
fi

# Extract one archive into a destination directory, format-agnostic.
extract_archive() {
    local archive="$1" dest="$2"
    case "$archive" in
    *.tar.gz)
        tar -xzf "$archive" -C "$dest"
        ;;
    *.zip)
        need_cmd unzip
        unzip -q "$archive" -d "$dest"
        ;;
    *.7z)
        if command -v 7zz >/dev/null 2>&1; then
            7zz x -y "$archive" -o"$dest" >/dev/null
        elif command -v 7z >/dev/null 2>&1; then
            7z x -y "$archive" -o"$dest" >/dev/null
        else
            die "legacy .7z asset $(basename "$archive") requires 7z/7zz (this release predates platform-native archives)"
        fi
        ;;
    *)
        die "unsupported archive format: $archive"
        ;;
    esac
}

copy_extras() {
    local extract_dir="$1" target_dir="$2"
    case "$KEEP_EXTRAS" in
    notice)
        for extra in NOTICE.md LICENSES; do
            if [[ -e "$extract_dir/$extra" ]]; then
                cp -pr "$extract_dir/$extra" "$target_dir/"
            fi
        done
        ;;
    license)
        # The archives nest everything one level deep, so check the archive
        # root and one level down, exactly like the binary search below.
        for candidate in "$extract_dir/LICENSE" "$extract_dir"/*/LICENSE; do
            if [[ -f "$candidate" ]]; then
                cp -p "$candidate" "$target_dir/LICENSE"
                break
            fi
        done
        ;;
    esac
}

for archive in "${archives[@]}"; do
    base="$(basename "$archive")"
    stem="$(strip_archive_ext "$base")"
    if ! parse_stem_to_platform "$stem"; then
        die "unknown platform in asset '$base' (expected <os>-<arch> such as darwin-arm64)"
    fi
    if [[ -e "$STAGE_DIR/$PLATFORM" ]]; then
        die "duplicate asset for platform $PLATFORM (from $base)"
    fi
    log INFO "extracting $base -> $TAG/$PLATFORM"
    extract_dir="$TMP_DIR/extract-$PLATFORM"
    mkdir -p "$extract_dir"
    extract_archive "$archive" "$extract_dir"
    bin_src=""
    for cand in "$extract_dir/$BIN" "$extract_dir/$BIN.exe"; do
        if [[ -f "$cand" ]]; then
            bin_src="$cand"
            break
        fi
    done
    if [[ -z "$bin_src" ]]; then
        # Some archives nest everything under a single top-level directory.
        for cand in "$extract_dir"/*/"$BIN" "$extract_dir"/*/"$BIN.exe"; do
            if [[ -f "$cand" ]]; then
                bin_src="$cand"
                break
            fi
        done
    fi
    [[ -n "$bin_src" ]] || die "no $BIN binary in $base (expected $BIN)"
    bin_name="$(basename "$bin_src")"
    target_dir="$STAGE_DIR/$PLATFORM"
    mkdir -p "$target_dir"
    cp -p "$bin_src" "$target_dir/$bin_name"
    chmod +x "$target_dir/$bin_name"
    [[ -s "$target_dir/$bin_name" ]] || die "installed binary is empty: $target_dir/$bin_name"
    copy_extras "$extract_dir" "$target_dir"

    log INFO "installed $PLATFORM/$bin_name (from $base)"
    rm -rf -- "$extract_dir"
done

TAG_DIR="$TOOL_ROOT/$TAG"
rm -rf -- "$TAG_DIR"
mkdir -p "$TOOL_ROOT"
mv "$STAGE_DIR" "$TAG_DIR"
STAGE_DIR=""

for dir in "$TOOL_ROOT"/v*/; do
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
for cand in "$TAG_DIR/$host_platform/$BIN" "$TAG_DIR/$host_platform/$BIN.exe"; do
    if [[ -x "$cand" ]]; then
        host_bin="$cand"
    fi
done
if [[ -n "$host_bin" ]]; then
    host_ver="$("$host_bin" --version 2>&1 || true)"
    log INFO "host binary $host_platform reports: $host_ver"
    case "$host_ver" in
    *"$TAG"* | *"${TAG#v}"*) ;;
    *) log WARNING "host binary version '$host_ver' does not contain $TAG" ;;
    esac
else
    log WARNING "host platform $host_platform not in bundle; skipped version check"
fi

log INFO "done: $TAG_DIR now contains:"
for dir in "$TAG_DIR"/*/; do
    [[ -d "$dir" ]] || continue
    log INFO "  $(basename "$dir")"
done
