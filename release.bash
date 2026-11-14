#!/usr/bin/env bash
# release.bash — compute the next version with versioner (dogfooding),
# cross-compile the binaries, tag HEAD, push the tag, and create a GitHub
# release with the binaries attached.
#
# The release carries one archive per platform (dist/<platform>/ holds the
# raw cross-compiled binary; the uploaded asset is
# formattington-<platform>.tar.gz, or .zip on Windows). Formattington is a
# Go program, so the binaries are produced with CGO disabled and are
# statically linked and portable.
#
# The program version is dynamic via versioner
# (internal/formattington/version.go + .versioner.json); the version is
# stamped into every cross-compiled binary with -ldflags -X, so the tag and
# `formattington --version` agree by construction.
#
# Usage:
#   ./release.bash                # version, build, tag, push, release + upload
#   ./release.bash --build-only   # cross-compile into dist/ only
#   ./release.bash --dry-run      # compute + verify + build, touch nothing remote
#
# Prerequisites: git, a Go toolchain (1.24+), gh authenticated via
# `gh auth login` (real release), and bundled versioner/logamuffin binaries
# (refresh with tools/versioner/update.bash and tools/logamuffin/update.bash;
# select with scripts/entry.sh). Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback until
# tools/logamuffin is populated).
# Working tree must be clean for a real release; HEAD should be pushed.
#
# The script is idempotent: an existing tag is reused, and an existing
# GitHub release receives the binaries through "gh release upload", so
# re-running it attaches the assets instead of failing.

set -euo pipefail

# Logging via logamuffin (log <level> <message> from scripts/entry.sh,
# with printf fallback until tools/logamuffin is populated). Source the
# entry point first: it pulls in the env cascade and defines log().
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
if [[ ! -f "$SCRIPT_DIR/scripts/entry.sh" ]]; then
    printf 'release.bash: error: script entry point not found: %s/scripts/entry.sh\n' "$SCRIPT_DIR" >&2
    exit 1
fi
# shellcheck disable=SC1091
. "$SCRIPT_DIR/scripts/entry.sh"

DRY_RUN=0
BUILD_ONLY=0

# The cross-compile matrix: "<platform> <goos> <goarch> <binary name>".
# The platform label names the dist/ subdirectory and the release asset.
PLATFORMS=(
    "linux-x64 linux amd64 formattington"
    "linux-arm64 linux arm64 formattington"
    "macos-x64 darwin amd64 formattington"
    "macos-arm64 darwin arm64 formattington"
    "windows-x64 windows amd64 formattington.exe"
    "windows-arm64 windows arm64 formattington.exe"
)

# The Go package path of the variable that carries the stamped version.
VERSION_PACKAGE="github.com/fordaj/formattington/internal/formattington"
VERSION_VARIABLE="buildVersion"

usage() {
    cat <<'EOF'
Usage: ./release.bash [--build-only | --dry-run]

Compute the next VERSION/TAG with versioner itself (dogfooding, bundled
binary selected via scripts/entry.sh), cross-compile the binaries for
every supported platform into dist/, verify the program's version
matches, create an annotated tag, push it, and create a GitHub release
with the per-platform archives attached.

Flags:
  --build-only    Cross-compile into dist/ and stop (no tag/push/release).
  --dry-run, -n   Compute + verify + build, but do not tag/push/release.
  --help, -h      Show this help.

Env:
  FORMATTINGTON_BIN  Path to the formattington binary (default: ./bin/formattington, then PATH).
  VERSIONER_BIN   Path to versioner (default: via scripts/entry.sh, then PATH).
  LOGAMUFFIN_BIN  Path to logamuffin for log() (default: via scripts/entry.sh, then PATH).
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
    --dry-run | -n)
        DRY_RUN=1
        shift
        ;;
    --build-only)
        BUILD_ONLY=1
        shift
        ;;
    --help | -h)
        usage
        exit 0
        ;;
    *)
        log ERROR "unknown argument: $1"
        usage >&2
        exit 2
        ;;
    esac
done

die() {
    log ERROR "$*" || true
    exit 1
}
need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need_cmd git
need_cmd go

if [[ "$DRY_RUN" -ne 1 && "$BUILD_ONLY" -ne 1 ]]; then
    need_cmd gh
    if ! gh auth status >/dev/null 2>&1; then
        die "gh is not authenticated (run: gh auth login)"
    fi
fi

if ! REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    die "not inside a git repository"
fi
cd "$REPO_ROOT"
log INFO "repo: $REPO_ROOT"

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
HEAD_SHORT="$(git rev-parse --short HEAD)"
log INFO "branch: $BRANCH  HEAD: $HEAD_SHORT"
if [[ "$DRY_RUN" -eq 1 ]]; then
    log INFO "DRY RUN: no tag/push/release will happen"
fi

# --- Precondition: clean working tree ---------------------------------
# Only a real release needs a pristine tree; --build-only and --dry-run
# tolerate a dirty one.
if [[ -n "$(git status --porcelain)" ]]; then
    if [[ "$DRY_RUN" -eq 1 || "$BUILD_ONLY" -eq 1 ]]; then
        log WARNING "working tree is not clean (allowed here)"
    else
        die "working tree is not clean (commit or stash first):"$'\n'"$(git status --short)"
    fi
fi

# Best-effort fetch so remote tag-existence checks are current.
if ! git fetch origin --tags 2>/dev/null; then
    if [[ "$DRY_RUN" -eq 1 || "$BUILD_ONLY" -eq 1 ]]; then
        log WARNING "git fetch origin --tags failed; remote tag check may be stale"
    else
        die "git fetch origin --tags failed"
    fi
fi

# --- Dogfooding: versioner/logamuffin via scripts/entry.sh (sourced above)
# shellcheck disable=SC2154
if [[ -z "${VERSIONER_BIN:-}" ]]; then
    die "no versioner binary found (run tools/versioner/update.bash, or set VERSIONER_BIN)"
fi
[[ -x "$VERSIONER_BIN" ]] || die "VERSIONER_BIN=$VERSIONER_BIN is not executable"
log INFO "using versioner: $VERSIONER_BIN"
if [[ -n "${LOGAMUFFIN_BIN:-}" && -x "${LOGAMUFFIN_BIN}" ]]; then
    log INFO "using logamuffin: $LOGAMUFFIN_BIN"
else
    log WARNING "logamuffin not found; falling back to printf logging (run tools/logamuffin/update.bash)"
fi
"$VERSIONER_BIN" --version | sed 's/^/release.bash:   /'

# --- Compute VERSION/TAG with versioner (dogfooding) ------------------
log INFO "computing version with versioner (dogfooding)..."
VERSION_OUT="$("$VERSIONER_BIN" --repo-dir "$REPO_ROOT" --quiet)"
log INFO "versioner output:"
printf '%s\n' "$VERSION_OUT" | sed 's/^/release.bash:   /'

VERSION="$(printf '%s\n' "$VERSION_OUT" | sed -n 's/^VERSION=//p' | tail -1)"
TAG="$(printf '%s\n' "$VERSION_OUT" | sed -n 's/^TAG=//p' | tail -1)"
[[ -n "$VERSION" ]] || die "could not parse VERSION from versioner output"
[[ -n "$TAG" ]] || die "could not parse TAG from versioner output"
if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    die "computed tag looks invalid: '$TAG' (expected vX.Y.Z.W.V)"
fi
log INFO "VERSION=$VERSION  TAG=$TAG"

# --- The program's own version must agree with the tag ----------------
# The binary computes its version from the same versioner binary, so the
# two must match; a mismatch means the binary is stale or was built from
# a different checkout.
resolve_formattington() {
    if [[ -n "${FORMATTINGTON_BIN:-}" ]]; then
        if [[ -x "${FORMATTINGTON_BIN}" ]]; then
            printf '%s' "$FORMATTINGTON_BIN"
            return 0
        fi
        return 1
    fi
    local candidate
    for candidate in "$REPO_ROOT/bin/formattington" "$REPO_ROOT/formattington"; do
        if [[ -x "$candidate" ]]; then
            printf '%s' "$candidate"
            return 0
        fi
    done
    if command -v formattington >/dev/null 2>&1; then
        command -v formattington
        return 0
    fi
    return 1
}

log INFO "verifying program version..."
if ! FMT_BIN="$(resolve_formattington)"; then
    log INFO "no formattington binary found; building it with go build..."
    mkdir -p "$REPO_ROOT/bin"
    (cd "$REPO_ROOT" && go build -o "$REPO_ROOT/bin/formattington" ./cmd/formattington) ||
        die "could not build formattington (install Go 1.24+ or set FORMATTINGTON_BIN)"
    FMT_BIN="$REPO_ROOT/bin/formattington"
fi
log INFO "using formattington: $FMT_BIN"

PKG_VERSION="$("$FMT_BIN" --version)"
log INFO "formattington --version reports: $PKG_VERSION"
if [[ "$PKG_VERSION" != "$VERSION" ]]; then
    die "program version $PKG_VERSION does not match versioner VERSION=$VERSION"
fi

# --- Cross-compile the release binaries -------------------------------
# dist/<platform>/formattington[.exe] holds the raw binary; the release
# asset for a platform is formattington-<platform>.tar.gz (or .zip on
# Windows) so every asset name is unique.
DIST_DIR="$REPO_ROOT/dist"
RELEASE_ASSETS=()

build_binaries() {
    local version="$1"
    local ldflags="-s -w -X $VERSION_PACKAGE.$VERSION_VARIABLE=$version"

    log INFO "cross-compiling ${#PLATFORMS[@]} platforms into dist/..."
    rm -rf "$DIST_DIR"
    mkdir -p "$DIST_DIR"

    local entry platform goos goarch binary outdir
    for entry in "${PLATFORMS[@]}"; do
        read -r platform goos goarch binary <<<"$entry"
        outdir="$DIST_DIR/$platform"
        mkdir -p "$outdir"
        log INFO "  $platform ($goos/$goarch) -> dist/$platform/$binary"
        (
            cd "$REPO_ROOT" &&
                CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
                    go build -trimpath -ldflags "$ldflags" \
                    -o "$outdir/$binary" ./cmd/formattington
        ) || die "cross-compile failed for $platform ($goos/$goarch)"
    done
}

package_binaries() {
    local entry platform goos goarch binary archive
    for entry in "${PLATFORMS[@]}"; do
        read -r platform goos goarch binary <<<"$entry"
        if [[ "$goos" == "windows" ]] && command -v zip >/dev/null 2>&1; then
            archive="$DIST_DIR/formattington-$platform.zip"
            (
                cd "$DIST_DIR" && zip -q -r "formattington-$platform.zip" "$platform"
            ) || die "could not package $platform"
        else
            if [[ "$goos" == "windows" ]]; then
                log WARNING "zip is not available; packaging the Windows binary as a tar.gz"
            fi
            archive="$DIST_DIR/formattington-$platform.tar.gz"
            tar -czf "$archive" -C "$DIST_DIR" "$platform" ||
                die "could not package $platform"
        fi
        log INFO "  packaged $(basename "$archive")"
        RELEASE_ASSETS+=("$archive")
    done
}

build_binaries "$VERSION"
package_binaries
log INFO "built ${#RELEASE_ASSETS[@]} release assets in dist/"

if [[ "$BUILD_ONLY" -eq 1 ]]; then
    log INFO "build only: stopping before tag/push/release"
    exit 0
fi

# --- Tag ----------------------------------------------------------------
# The script is idempotent: a tag that already exists is reused instead of
# being recreated, so a release can be re-run to (re)attach the binaries.
TAG_EXISTS_LOCALLY=0
TAG_EXISTS_REMOTELY=0
if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null 2>&1; then
    TAG_EXISTS_LOCALLY=1
fi
if git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
    TAG_EXISTS_REMOTELY=1
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
    log INFO "DRY RUN: would run:"
    if [[ "$TAG_EXISTS_LOCALLY" -eq 1 ]]; then
        log INFO "  git tag -a $TAG -m \"formattington $TAG\"   (skipped: the tag already exists)"
    else
        log INFO "  git tag -a $TAG -m \"formattington $TAG\""
    fi
    if [[ "$TAG_EXISTS_REMOTELY" -eq 1 ]]; then
        log INFO "  git push origin $TAG   (skipped: the tag is already on origin)"
    else
        log INFO "  git push origin $TAG"
    fi
    log INFO "  gh release create $TAG --title \"formattington $TAG\" --generate-notes ${RELEASE_ASSETS[*]}"
    log INFO "  gh release upload $TAG ${RELEASE_ASSETS[*]} --clobber   (only if the release already exists)"
    log INFO "dry run complete: VERSION=$VERSION TAG=$TAG"
    exit 0
fi

if [[ "$TAG_EXISTS_LOCALLY" -eq 0 ]]; then
    log INFO "creating annotated tag $TAG..."
    git tag -a "$TAG" -m "formattington $TAG"
else
    log WARNING "tag $TAG already exists locally; reusing it"
fi

if [[ "$TAG_EXISTS_REMOTELY" -eq 0 ]]; then
    log INFO "pushing tag $TAG..."
    git push origin "$TAG"
else
    log WARNING "tag $TAG is already on origin; not pushing it again"
fi

# --- GitHub release (binaries attached) --------------------------------
if gh release view "$TAG" >/dev/null 2>&1; then
    log INFO "GitHub release $TAG already exists; attaching ${#RELEASE_ASSETS[@]} assets..."
    gh release upload "$TAG" "${RELEASE_ASSETS[@]}" --clobber
    log INFO "attached ${#RELEASE_ASSETS[@]} assets to release $TAG"
else
    log INFO "creating GitHub release $TAG with ${#RELEASE_ASSETS[@]} assets..."
    gh release create "$TAG" \
        --title "formattington $TAG" \
        --generate-notes \
        "${RELEASE_ASSETS[@]}"
fi

if REPO_SLUG="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" && [[ -n "$REPO_SLUG" ]]; then
    log INFO "release: https://github.com/$REPO_SLUG/releases/tag/$TAG"
else
    log INFO "release $TAG created (see GitHub Releases)"
fi
