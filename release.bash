#!/usr/bin/env bash
# release.bash — compute the next version with versioner (dogfooding),
# fetch the bundled formatters, cross-compile the binaries, tag HEAD,
# push the tag, and create a GitHub release with the archives attached.
#
# Two archives are produced per platform:
#
#   formattington-lite-<platform>  the program alone
#   formattington-full-<platform>  the program with the bundled formatter
#                                  executables for that platform embedded
#                                  inside it, so the binary is a single,
#                                  self-contained executable (the archive
#                                  holds just that one binary)
#
# The full build embeds a zip of the platform's formatter executables in
# the binary (Go "full" build tag + go:embed). On first use the archive is
# unpacked into the per-user cache directory (formattington/formatters
# inside os.UserCacheDir()); the FORMATTINGTON_FORMATTERS_DIR environment
# variable still overrides formatter resolution, and lite builds resolve
# from a checkout or PATH.
#
# The bundled formatter executables are fetched with the per-tool
# formatters/<tool>/update.sh scripts and land in a staging directory as
# formatters/<tool>/v<version>/<platform>/, which is exactly the layout
# the program resolves at runtime.
#
# Formattington is a Go program, so the binaries are produced with CGO
# disabled and are statically linked and portable.
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
# tools/logamuffin is populated). Fetching the formatters additionally
# needs curl, jq, and zip (and pnpm plus bun for the tools that are built
# from source).
# Working tree must be clean for a real release; HEAD should be pushed.
#
# The script is idempotent: an existing tag is reused, and an existing
# GitHub release receives the archives through "gh release upload", so
# re-running it attaches the assets instead of failing.
#
# Env:
#   FORMATTER_VERSION  Version of the bundled formatters to fetch (default: latest).

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

# The cross-compile matrix:
# "<release platform> <goos> <goarch> <bundle platform> <binary name>".
#
# The release platform names the dist/ directory and the release asset
# (for example "windows-x64"). The bundle platform is the name the
# bundled formatter executables use in their directory layout (for
# example "windows-x86_64"); it is the name the program resolves at
# runtime, so the two differ on purpose.
PLATFORMS=(
	"linux-x64 linux amd64 linux-x86_64 formattington"
	"linux-arm64 linux arm64 linux-arm64 formattington"
	"macos-x64 darwin amd64 macos-x86_64 formattington"
	"macos-arm64 darwin arm64 macos-arm64 formattington"
	"windows-x64 windows amd64 windows-x86_64 formattington.exe"
	"windows-arm64 windows arm64 windows-arm64 formattington.exe"
)

# The version of the bundled formatters to fetch. "latest" resolves the
# newest upstream release of each tool.
FORMATTER_VERSION="${FORMATTER_VERSION:-latest}"

# The Go package path of the variable that carries the stamped version.
VERSION_PACKAGE="github.com/fordaj/formattington/internal/formattington"
VERSION_VARIABLE="buildVersion"

usage() {
	cat <<'EOF'
Usage: ./release.bash [--build-only | --dry-run]

Compute the next VERSION/TAG with versioner itself (dogfooding, bundled
binary selected via scripts/entry.sh), fetch the bundled formatters with
formatters/<tool>/update.sh, cross-compile the program for every
supported platform into dist/, verify the program's version matches,
create an annotated tag, push it, and create a GitHub release with the
per-platform archives attached.

Two archives are produced per platform: formattington-lite-<platform>
(the program alone) and formattington-full-<platform> (the program with
the bundled formatter executables for that platform embedded inside the
binary, so it is a single, self-contained executable).

Flags:
  --build-only    Cross-compile into dist/ and stop (no tag/push/release).
  --dry-run, -n   Compute + verify + build, but do not tag/push/release.
  --help, -h      Show this help.

Env:
  FORMATTINGTON_BIN  Path to the formattington binary (default: ./bin/formattington, then PATH).
  VERSIONER_BIN   Path to versioner (default: via scripts/entry.sh, then PATH).
  LOGAMUFFIN_BIN  Path to logamuffin for log() (default: via scripts/entry.sh, then PATH).
  FORMATTER_VERSION  Version of the bundled formatters to fetch (default: latest).
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
need_cmd curl
need_cmd jq
need_cmd zip
need_cmd tar

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
# Two variants are produced per platform:
#
#   formattington-lite-<platform>  the program alone (dist/lite/)
#   formattington-full-<platform>  the program with the bundled formatter
#                                  executables for that platform embedded
#                                  inside the binary (dist/full/), so the
#                                  archive is a single self-contained
#                                  executable
#
# dist/lite/<platform>/ and dist/full/<platform>/ hold the cross-compiled
# programs; the archives are built from a staging tree and land in dist/.
DIST_DIR="$REPO_ROOT/dist"
# The full build embeds the payload from here (see embedded_full.go);
# release.bash stages it per platform and removes it again on exit.
ASSETS_DIR="$REPO_ROOT/internal/formattington/assets"
RELEASE_ASSETS=()
STAGE_DIR=""

cleanup_stage() {
	rm -rf -- "$ASSETS_DIR"
	if [[ -n "${STAGE_DIR:-}" && -d "$STAGE_DIR" ]]; then
		rm -rf -- "$STAGE_DIR"
	fi
}
trap cleanup_stage EXIT

build_binaries() {
	local version="$1"
	local ldflags="-s -w -X $VERSION_PACKAGE.$VERSION_VARIABLE=$version"

	log INFO "cross-compiling ${#PLATFORMS[@]} platforms into dist/..."
	rm -rf "$DIST_DIR"
	rm -rf "$ASSETS_DIR"
	mkdir -p "$DIST_DIR/lite" "$DIST_DIR/full"

	local entry platform goos goarch bundle binary lite_out full_out
	for entry in "${PLATFORMS[@]}"; do
		read -r platform goos goarch bundle binary <<<"$entry"

		# The lite build never touches the embed staging area.
		lite_out="$DIST_DIR/lite/$platform/$binary"
		mkdir -p "$(dirname "$lite_out")"
		log INFO "  $platform ($goos/$goarch) lite -> dist/lite/$platform/$binary"
		(
			cd "$REPO_ROOT" &&
				CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
					go build -trimpath -ldflags "$ldflags" \
					-o "$lite_out" ./cmd/formattington
		) || die "cross-compile failed for $platform ($goos/$goarch) lite"

		# Stage this platform's formatter payload, then the full build.
		stage_formatter_payload "$bundle"

		full_out="$DIST_DIR/full/$platform/$binary"
		mkdir -p "$(dirname "$full_out")"
		log INFO "  $platform ($goos/$goarch) full -> dist/full/$platform/$binary"
		(
			cd "$REPO_ROOT" &&
				CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
					go build -trimpath -tags full -ldflags "$ldflags" \
					-o "$full_out" ./cmd/formattington
		) || die "cross-compile failed for $platform ($goos/$goarch) full"

		rm -rf "$ASSETS_DIR"
	done
}

# stage_formatter_payload <bundle platform>
# Fill the embed staging area with a zip of the fetched formatter
# executables for one bundle platform, ready for the "-tags full" build.
# A valid empty archive is written when no formatter was fetched for the
# platform, so the embed pattern always matches; an empty archive is
# treated as "no embedded formatters" at runtime.
stage_formatter_payload() {
	local platform="$1"
	local staging="$STAGE_DIR/payload-$platform"
	local zip_file="$ASSETS_DIR/formatters.zip"

	rm -rf "$staging" "$ASSETS_DIR"
	mkdir -p "$staging" "$ASSETS_DIR"
	copy_formatters "$platform" "$staging"

	local count
	count="$(find "$staging" -type f | wc -l | tr -d ' ')"
	if [[ "$count" -gt 0 ]]; then
		# Archive the payload with clean relative names at the root:
		# <tool>/v<version>/<platform>/<executable>.
		(
			cd "$staging" &&
				find . -type f -print | sed 's|^\./||' | LC_ALL=C sort |
				zip -q -X "$zip_file" -@
		) || die "could not stage the embedded formatters for $platform"
		log INFO "  embedded $count formatter executable(s) in the $platform full binary"
	else
		log WARNING "no bundled formatters for $platform; the full binary will match lite"
		printf 'PK\005\006' >"$zip_file"
		# An empty zip archive is exactly the 22-byte end-of-central-directory record.
		head -c 18 /dev/zero >>"$zip_file"
	fi
}

# fetch_formatters
# Run every formatters/<tool>/update.sh once per tool. The scripts
# download the upstream executables into
# <stage>/formatters/<tool>/v<version>/<platform>/, which is exactly the
# layout the program resolves at runtime. The Black script additionally
# builds the macOS x86_64 executable from source on a native Intel Mac,
# because upstream publishes no standalone asset for it.
#
# A script stops at the first target it cannot provide, so the platforms
# that are still missing afterwards are retried one by one.
fetch_formatters() {
	local tool_dir tool update_script platform
	local platforms=()
	local entry release_platform goos goarch bundle binary

	for entry in "${PLATFORMS[@]}"; do
		# shellcheck disable=SC2034 # only the bundle platform is needed here
		read -r release_platform goos goarch bundle binary <<<"$entry"
		platforms+=("$bundle")
	done

	for tool_dir in "$REPO_ROOT"/formatters/*/; do
		[[ -d "$tool_dir" ]] || continue
		tool="$(basename "$tool_dir")"
		update_script="$tool_dir/update.sh"
		[[ -f "$update_script" ]] || continue

		run_update "$tool" "$update_script" "${platforms[@]}" || true

		for platform in "${platforms[@]}"; do
			if ! formatter_present "$tool" "$platform"; then
				run_update "$tool" "$update_script" "$platform" || true
			fi
		done

		local missing=""
		for platform in "${platforms[@]}"; do
			if ! formatter_present "$tool" "$platform"; then
				missing="$missing $platform"
			fi
		done
		if [[ -z "$missing" ]]; then
			log INFO "  fetched $tool for ${#platforms[@]} platforms"
		else
			log WARNING "$tool is not available for:$missing (it will be missing from those full bundles)"
		fi
	done
}

# run_update <tool> <update-script> <platform>...
run_update() {
	local tool="$1" update_script="$2"
	shift 2
	if OUTPUT_DIR="$STAGE_DIR/formatters/$tool" \
		"$update_script" "$FORMATTER_VERSION" "$@" \
		>"$STAGE_DIR/$tool.log" 2>&1; then
		return 0
	fi
	log WARNING "$tool: formatters/$tool/update.sh $FORMATTER_VERSION $* failed; last log lines:"
	tail -n 10 "$STAGE_DIR/$tool.log" 2>/dev/null | sed 's/^/release.bash:   /' || true
	return 1
}

# formatter_present <tool> <platform>
# Whether the tool was downloaded for a platform, that is whether
# <stage>/formatters/<tool>/v<version>/<platform>/ exists.
formatter_present() {
	local tool="$1" platform="$2"
	local dir
	for dir in "$STAGE_DIR/formatters/$tool"/v*/"$platform"; do
		if [[ -d "$dir" ]]; then
			return 0
		fi
	done
	return 1
}

# copy_formatters <bundle-platform> <destination>
# Copy the formatter executables for one platform into a staging tree.
# The tree layout (<tool>/v<version>/<platform>/<executable>) becomes the
# archive members and, after extraction, the runtime layout.
copy_formatters() {
	local platform="$1"
	local destination="$2"
	local source="$STAGE_DIR/formatters"
	local dir tool version

	[[ -d "$source" ]] || return 0

	for dir in "$source"/*/v*/"$platform"; do
		[[ -d "$dir" ]] || continue
		tool="$(basename "$(dirname "$(dirname "$dir")")")"
		version="$(basename "$(dirname "$dir")")"
		mkdir -p "$destination/$tool/$version/$platform"
		cp -a "$dir/." "$destination/$tool/$version/$platform/"
	done
}

# package_tree <directory> <archive>
# Package a directory as a zip (Windows) or a gzipped tar (everything
# else), keeping the directory itself as the archive's top-level entry.
package_tree() {
	local directory="$1"
	local archive="$2"
	local parent base
	parent="$(dirname "$directory")"
	base="$(basename "$directory")"

	if [[ "$archive" == *.zip ]]; then
		(
			cd "$parent" && zip -q -r "$archive" "$base"
		) || die "could not package $archive"
	else
		tar -czf "$archive" -C "$parent" "$base" ||
			die "could not package $archive"
	fi
}

package_binaries() {
	local entry platform goos goarch _ binary extension
	local lite_tree full_tree archive

	for entry in "${PLATFORMS[@]}"; do
		read -r platform goos goarch _ binary <<<"$entry"
		if [[ "$goos" == "windows" ]]; then
			extension="zip"
		else
			extension="tar.gz"
		fi

		# Lite: the program alone.
		lite_tree="$STAGE_DIR/pkg/formattington-lite-$platform"
		mkdir -p "$lite_tree"
		cp "$DIST_DIR/lite/$platform/$binary" "$lite_tree/$binary"
		archive="$DIST_DIR/formattington-lite-$platform.$extension"
		package_tree "$lite_tree" "$archive"
		log INFO "  packaged $(basename "$archive")"
		RELEASE_ASSETS+=("$archive")

		# Full: the self-contained program (the bundled formatters are
		# embedded inside the binary).
		full_tree="$STAGE_DIR/pkg/formattington-full-$platform"
		mkdir -p "$full_tree"
		cp "$DIST_DIR/full/$platform/$binary" "$full_tree/$binary"
		archive="$DIST_DIR/formattington-full-$platform.$extension"
		package_tree "$full_tree" "$archive"
		log INFO "  packaged $(basename "$archive")"
		RELEASE_ASSETS+=("$archive")
	done
}

# smoke_full_binary
# Run the freshly built full binary for this host under an empty HOME, so
# the embedded formatters must be unpacked from the binary itself. Passes
# when every fetched tool for the host bundle platform answers
# "--version" — through the CLI for the tools that are registered, and by
# direct invocation of the unpacked executable for the ones that are not
# (markdownlint). Fails the build otherwise. Skipped on platforms the
# matrix does not cover and when no formatter was fetched for the host.
smoke_full_binary() {
	local host_system host_machine host_bundle host_full entry release_platform bundle binary tool output

	host_system="$(uname -s)"
	host_machine="$(uname -m)"
	case "$host_system" in
	Darwin) host_system="macos" ;;
	Linux) host_system="linux" ;;
	MINGW* | MSYS* | CYGWIN*) host_system="windows" ;;
	*)
		log WARNING "smoke test skipped (unsupported host: $host_system)"
		return 0
		;;
	esac
	case "$host_machine" in
	x86_64 | amd64) host_machine="x86_64" ;;
	arm64 | aarch64) host_machine="arm64" ;;
	*)
		log WARNING "smoke test skipped (unsupported host architecture: $host_machine)"
		return 0
		;;
	esac
	host_bundle="$host_system-$host_machine"

	host_full=""
	for entry in "${PLATFORMS[@]}"; do
		# shellcheck disable=SC2034 # only the release platform, bundle and binary are needed
		read -r release_platform _ _ bundle binary <<<"$entry"
		if [[ "$bundle" == "$host_bundle" ]]; then
			host_full="$DIST_DIR/full/$release_platform/$binary"
			break
		fi
	done
	if [[ -z "$host_full" ]]; then
		log WARNING "smoke test skipped (no matrix entry for $host_bundle)"
		return 0
	fi

	local tools=()
	for tool_dir in "$STAGE_DIR"/formatters/*/v*/"$host_bundle"; do
		[[ -d "$tool_dir" ]] || continue
		tools+=("$(basename "$(dirname "$(dirname "$tool_dir")")")")
	done
	if [[ "${#tools[@]}" -eq 0 ]]; then
		log WARNING "smoke test skipped (no formatters were fetched for $host_bundle)"
		return 0
	fi

	# The formatters registered in the CLI. markdownlint is bundled but not
	# yet integrated, so it is exercised through its unpacked binary below.
	local cli_tools=(shfmt clang-format black prettier taplo)

	local smoke_home
	smoke_home="$(mktemp -d)"
	trap 'rm -rf "$smoke_home"' RETURN
	local failed=() exe exe_name direct_output
	for tool in "${tools[@]}"; do
		case " ${cli_tools[*]} " in
		*" $tool "*)
			# PATH intentionally excludes every location a formatter could
			# hide in, so a passing run proves the embedded bundle resolved.
			output="$(
				env -u FORMATTINGTON_FORMATTERS_DIR \
					HOME="$smoke_home" XDG_CACHE_HOME="$smoke_home/xdg" \
					PATH=/usr/bin:/bin "$host_full" "$tool" --version 2>&1
			)" || failed+=("$tool")
			log INFO "  $tool (via formattington): $(head -n 1 <<<"$output")"
			;;
		*)
			log INFO "  $tool is not integrated into the CLI; checking the unpacked binary"
			;;
		esac

		# Direct check: the unpacked executable must run on this host,
		# whether or not the CLI knows about it.
		exe_name="$tool"
		if [[ "$host_full" == *.exe ]]; then
			exe_name="$tool.exe"
		fi
		exe="$(find "$smoke_home" -type f -path "*/$tool/v*/$host_bundle/$exe_name" -print -quit)"
		if [[ -z "$exe" ]]; then
			failed+=("$tool (unpacked binary not found)")
			continue
		fi
		if direct_output="$("$exe" --version 2>&1)"; then
			log INFO "  $tool (direct): $(head -n 1 <<<"$direct_output")"
		else
			failed+=("$tool (unpacked binary exited non-zero)")
		fi
	done
	trap - RETURN
	rm -rf "$smoke_home"

	if [[ "${#failed[@]}" -ne 0 ]]; then
		die "smoke test failed for: ${failed[*]} (the embedded formatters do not run on this host)"
	fi
	log INFO "smoke test passed: the embedded formatters unpack and run on this host"
}

STAGE_DIR="$(mktemp -d)"

# Fetch the bundled formatter executables first: the full build embeds
# them, so the payload must be in place before cross-compiling.
log INFO "fetching the bundled formatters..."
fetch_formatters

build_binaries "$VERSION"
package_binaries
log INFO "built ${#RELEASE_ASSETS[@]} release assets in dist/"

# The freshly built full binary must run its embedded formatters on this
# host; a broken bundle must never ship.
smoke_full_binary

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
