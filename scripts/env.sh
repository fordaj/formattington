#!/usr/bin/env bash
# env.sh — source the Formattington env cascade and select the
# host-appropriate versioner binary.
#
# Sourced usage (bash):
#   source scripts/env.sh   # or: . scripts/env.sh
#   "$VERSIONER_BIN" -quiet
#
# Executed usage (prints export statements for eval in any shell):
#   eval "$(bash scripts/env.sh)"
#   bash scripts/env.sh
#
# Cascade (all committed): scripts/.common.env (base) then
# scripts/.$PLATFORM.env (os-specific, e.g. .macos-x86_64.env),
# where $PLATFORM is the hyphenated <os>-<arch> from uname
# (macos/linux/windows + x86_64/arm64, matching get_platform_dir()).
# env.sh then resolves $VERSIONER_BIN to the newest bundled binary
# (tools/versioner/v*/$PLATFORM/versioner[.exe]; tools/versioner/update.bash
# keeps exactly one v*/ dir), falling back to `versioner` on PATH.
#
# Exports:
#   FORMATTINGTON_PLATFORM  host platform (e.g. macos-x86_64)
#   VERSIONER_BIN           absolute path to versioner ("" if not found)
#   VERSIONER_BIN_NAME      versioner or versioner.exe
#   VERSIONER_REPO, VERSIONER_TAG_PATTERN, VERSIONER_CONFIG_FILE
#   VERSIONER_BUNDLE_DIR (repo-relative)

# Resolve this file's directory (works sourced or executed, from anywhere).
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
    _env_sh_path="${BASH_SOURCE[0]}"
else
    _env_sh_path="$0"
fi
_env_sh_dir=$(cd -- "$(dirname -- "$_env_sh_path")" &>/dev/null && pwd)
_repo_root=$(dirname -- "$_env_sh_dir")

# Detect host platform (mirrors get_platform_dir()).
_host_os_raw=$(uname -s | tr '[:upper:]' '[:lower:]')
_host_arch_raw=$(uname -m | tr '[:upper:]' '[:lower:]')
case "$_host_os_raw" in
darwin*) _host_os="macos" ;;
linux*) _host_os="linux" ;;
msys* | mingw* | cygwin* | windows* | win*) _host_os="windows" ;;
*) _host_os="$_host_os_raw" ;;
esac
case "$_host_arch_raw" in
x86_64 | amd64) _host_arch="x86_64" ;;
arm64 | aarch64) _host_arch="arm64" ;;
*) _host_arch="$_host_arch_raw" ;;
esac
_detected_platform="$_host_os-$_host_arch"

# Source cascade (base, then platform-specific). set -a auto-exports.
set -a
# shellcheck disable=SC1090,SC1091
. "$_env_sh_dir/.common.env"
if [[ -f "$_env_sh_dir/.$_detected_platform.env" ]]; then
    # shellcheck disable=SC1090,SC1091
    . "$_env_sh_dir/.$_detected_platform.env"
else
    echo "env.sh: warning: no $_env_sh_dir/.$_detected_platform.env; using defaults" >&2
    FORMATTINGTON_PLATFORM="$_detected_platform"
    if [[ "$_host_os" == "windows" ]]; then
        VERSIONER_BIN_NAME="versioner.exe"
    else
        VERSIONER_BIN_NAME="versioner"
    fi
fi
set +a

FORMATTINGTON_PLATFORM="${FORMATTINGTON_PLATFORM:-$_detected_platform}"
VERSIONER_BIN_NAME="${VERSIONER_BIN_NAME:-versioner}"
VERSIONER_BUNDLE_DIR="${VERSIONER_BUNDLE_DIR:-tools/versioner}"

# Resolve the newest bundled binary (single v*/ dir per update.bash invariant).
VERSIONER_BIN=""
_bundle_root="$_repo_root/$VERSIONER_BUNDLE_DIR"
for _version_dir in "$_bundle_root"/v*/; do
    if [[ ! -d "$_version_dir" ]]; then
        continue
    fi
    _cand="$_version_dir$FORMATTINGTON_PLATFORM/$VERSIONER_BIN_NAME"
    if [[ -f "$_cand" ]]; then
        VERSIONER_BIN="$_cand"
    fi
done
if [[ -z "$VERSIONER_BIN" || ! -x "$VERSIONER_BIN" ]]; then
    if command -v versioner >/dev/null 2>&1; then
        VERSIONER_BIN=$(command -v versioner)
    else
        VERSIONER_BIN=""
    fi
fi

export FORMATTINGTON_PLATFORM VERSIONER_BIN VERSIONER_BIN_NAME
export VERSIONER_REPO VERSIONER_TAG_PATTERN VERSIONER_CONFIG_FILE VERSIONER_BUNDLE_DIR

unset _env_sh_path _env_sh_dir _repo_root _host_os_raw _host_arch_raw
unset _host_os _host_arch _detected_platform _bundle_root _version_dir _cand

# Sourced vs executed: when executed, print export statements.
if [[ -n "${BASH_SOURCE[0]:-}" && "${BASH_SOURCE[0]:-}" == "$0" ]]; then
    printf 'export FORMATTINGTON_PLATFORM=%q\n' "$FORMATTINGTON_PLATFORM"
    printf 'export VERSIONER_BIN=%q\n' "$VERSIONER_BIN"
    printf 'export VERSIONER_BIN_NAME=%q\n' "$VERSIONER_BIN_NAME"
    printf 'export VERSIONER_REPO=%q\n' "${VERSIONER_REPO:-}"
    printf 'export VERSIONER_TAG_PATTERN=%q\n' "${VERSIONER_TAG_PATTERN:-}"
    printf 'export VERSIONER_CONFIG_FILE=%q\n' "${VERSIONER_CONFIG_FILE:-}"
    printf 'export VERSIONER_BUNDLE_DIR=%q\n' "${VERSIONER_BUNDLE_DIR:-}"
fi
