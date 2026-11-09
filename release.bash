#!/usr/bin/env bash
# release.bash — compute the next version with versioner (dogfooding),
# tag HEAD, push the tag, and create a GitHub release for version tracking.
#
# Unlike the Versioner repo release, this packs no binaries: GitHub Releases
# are for version tracking only (no .exe uploads). Python wheels/sdists are
# built separately with `uv build` from the tagged commit; they are not
# attached here. The package version itself is dynamic via versioner
# (src/formattington/_version.py + .versioner.json + pyproject.toml
# dynamic = ["version"]), so the tag and `uv build` agree by construction.
#
# Usage:
#   ./release.bash           # compute version, tag, push, gh release create
#   ./release.bash --dry-run # compute + verify only, touch nothing remote
#
# Prerequisites: git, gh authenticated via `gh auth login` (real release),
# python3 for _version verification, and a bundled versioner binary
# (refresh with tools/versioner/update.bash; select with scripts/env.sh).
# Working tree must be clean; HEAD should be pushed.

set -euo pipefail

DRY_RUN=0

usage() {
    cat <<'EOF'
Usage: ./release.bash [--dry-run]

Compute the next VERSION/TAG with versioner itself (dogfooding, bundled
binary selected via scripts/env.sh), verify the dynamic package version
matches, create an annotated tag, push it, and create a GitHub release
with generated notes (version tracking only, no assets).

Flags:
  --dry-run, -n   Compute + verify version, but do not tag/push/release.
  --help, -h      Show this help.

Env:
  VERSIONER_BIN   Path to versioner (default: via scripts/env.sh, then PATH).
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
    *)
        echo "release.bash: unknown argument: $1" >&2
        usage >&2
        exit 2
        ;;
    esac
done

log() { printf 'release.bash: %s\n' "$*"; }
die() {
    printf 'release.bash: error: %s\n' "$*" >&2
    exit 1
}
need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need_cmd git

if command -v python3.14 >/dev/null 2>&1; then
    PYTHON_BIN="python3.14"
elif command -v python3 >/dev/null 2>&1; then
    PYTHON_BIN="python3"
else
    die "required command not found: python3.14 (project requires Python >=3.14)"
fi

if [[ "$DRY_RUN" -ne 1 ]]; then
    need_cmd gh
    if ! gh auth status >/dev/null 2>&1; then
        die "gh is not authenticated (run: gh auth login)"
    fi
fi

if ! REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    die "not inside a git repository"
fi
cd "$REPO_ROOT"
log "repo: $REPO_ROOT"

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
HEAD_SHORT="$(git rev-parse --short HEAD)"
log "branch: $BRANCH  HEAD: $HEAD_SHORT"
if [[ "$DRY_RUN" -eq 1 ]]; then
    log "DRY RUN: no tag/push/release will happen"
fi

# --- Precondition: clean working tree ---------------------------------
if [[ -n "$(git status --porcelain)" ]]; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
        log "warning: working tree is not clean (allowed in --dry-run)"
        git status --short | sed 's/^/release.bash:   /' >&2
    else
        die "working tree is not clean (commit or stash first):"$'\n'"$(git status --short)"
    fi
fi

# Best-effort fetch so remote tag-existence checks are current.
if ! git fetch origin --tags 2>/dev/null; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
        log "warning: git fetch origin --tags failed; remote tag check may be stale"
    else
        die "git fetch origin --tags failed"
    fi
fi

# --- Dogfooding: select versioner via scripts/env.sh ------------------
if [[ -z "${VERSIONER_BIN:-}" ]]; then
    # shellcheck disable=SC1091
    . "$REPO_ROOT/scripts/env.sh"
fi
# shellcheck disable=SC2154
if [[ -z "${VERSIONER_BIN:-}" ]]; then
    die "no versioner binary found (run tools/versioner/update.bash, or set VERSIONER_BIN)"
fi
[[ -x "$VERSIONER_BIN" ]] || die "VERSIONER_BIN=$VERSIONER_BIN is not executable"
log "using versioner: $VERSIONER_BIN"
"$VERSIONER_BIN" --version | sed 's/^/release.bash:   /'

# --- Compute VERSION/TAG with versioner (dogfooding) ------------------
log "computing version with versioner (dogfooding)..."
VERSION_OUT="$("$VERSIONER_BIN" --repo-dir "$REPO_ROOT" --quiet)"
log "versioner output:"
printf '%s\n' "$VERSION_OUT" | sed 's/^/release.bash:   /'

VERSION="$(printf '%s\n' "$VERSION_OUT" | sed -n 's/^VERSION=//p' | tail -1)"
TAG="$(printf '%s\n' "$VERSION_OUT" | sed -n 's/^TAG=//p' | tail -1)"
[[ -n "$VERSION" ]] || die "could not parse VERSION from versioner output"
[[ -n "$TAG" ]] || die "could not parse TAG from versioner output"
if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    die "computed tag looks invalid: '$TAG' (expected vX.Y.Z.W.V)"
fi
log "VERSION=$VERSION  TAG=$TAG"

# --- Dynamic package version must agree with the tag ------------------
log "verifying dynamic package version..."
# shellcheck disable=SC2154
PKG_VERSION="$(PYTHONPATH="$REPO_ROOT/src" "$PYTHON_BIN" -c 'import formattington._version as v; print(v.__version__)')"
log "formattington._version reports: $PKG_VERSION"
if [[ "$PKG_VERSION" != "$VERSION" ]]; then
    die "dynamic version $PKG_VERSION does not match versioner VERSION=$VERSION"
fi

# --- Tag must not already exist ---------------------------------------
if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null 2>&1; then
    die "tag $TAG already exists locally (make a new feat:/fix: commit to bump the version, then re-run)"
fi
if git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
    die "tag $TAG already exists on origin (make a new feat:/fix: commit to bump the version, then re-run)"
fi

# --- Tag + GitHub release (version tracking only, no assets) -----------
if [[ "$DRY_RUN" -eq 1 ]]; then
    log "DRY RUN: would run:"
    log "  git tag -a $TAG -m \"formattington $TAG\""
    log "  git push origin $TAG"
    log "  gh release create $TAG --title \"formattington $TAG\" --generate-notes"
    log "dry run complete: VERSION=$VERSION TAG=$TAG"
    exit 0
fi

log "creating annotated tag $TAG..."
git tag -a "$TAG" -m "formattington $TAG"

log "pushing tag $TAG..."
git push origin "$TAG"

if gh release view "$TAG" >/dev/null 2>&1; then
    die "GitHub release $TAG already exists (tag was pushed; add it manually with: gh release create $TAG --title \"formattington $TAG\" --generate-notes)"
fi

log "creating GitHub release $TAG (version tracking only, no assets)..."
gh release create "$TAG" \
    --title "formattington $TAG" \
    --generate-notes

if REPO_SLUG="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" && [[ -n "$REPO_SLUG" ]]; then
    log "release: https://github.com/$REPO_SLUG/releases/tag/$TAG"
else
    log "release $TAG created (see GitHub Releases)"
fi
