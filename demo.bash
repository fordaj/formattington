#!/usr/bin/env bash
# demo.bash — end-to-end tour of Formattington, logged with logamuffin.
#
# Shows the main features an end user would use — one interface, tool
# and language selection, check/dry-run modes, sequential `all`,
# config files, exit codes, and bundled-vs-PATH resolution — without
# modifying your repository. Every formattington call is non-destructive
# (`--help`, `--version`, `--verbose`, `--check`, or `--dry-run`) and
# targets messy sample files with explicit paths inside a temporary
# sandbox (including a path with spaces) that is removed on exit.
# Discovery is shown by running from the sandbox so `all` stays scoped
# to the demo files instead of the whole repository.
#
# All narration goes through `log <level> <message>` from
# scripts/env.sh (logamuffin, with printf fallback), never by calling
# the logamuffin binary directly. The per-call LOG_SOURCE field shows
# demo.bash file:line as messages log.
#
# Usage:
#   ./demo.bash          # run the whole tour (single non-interactive run)
#   ./demo.bash --help   # usage
#
# Prerequisites: uv (or `formattington` on PATH), and the bundled tools
# under tools/ for offline use. `black`, `prettier`, and `pwsh` are
# PATH-only — when one is missing its call exits 127 and the tour
# continues (see section 8). No network, gh, or versioner refresh is
# needed. Python 3.14 or newer is required (via uv).
#
# Env:
#   VERSIONER_BIN   Path to versioner (default: via scripts/env.sh, then PATH).
#   LOGAMUFFIN_BIN  Path to logamuffin for log() (default: via scripts/env.sh, then PATH).
#   LOG_LEVEL       Minimum level for log() (default: INFO; try TRACE to see more).
#   FORMATTINGTON_VERBOSE  Set to 1/true/yes/on to show resolved executables.

set -euo pipefail

# Logging via logamuffin (log <level> <message> from scripts/env.sh,
# with printf fallback until tools/logamuffin is populated). Source the
# env cascade first so every later message goes through log().
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
if [[ ! -f "$SCRIPT_DIR/scripts/env.sh" ]]; then
    printf 'demo.bash: error: env cascade not found: %s/scripts/env.sh\n' "$SCRIPT_DIR" >&2
    exit 1
fi
# shellcheck disable=SC1091
. "$SCRIPT_DIR/scripts/env.sh"

usage() {
    cat <<'EOF'
Usage: ./demo.bash [--help]

Run a non-destructive, single-pass tour of Formattington: logamuffin
logging, --help/--version, --verbose resolution, single-tool and
language-alias selection with explicit files, --check vs --dry-run,
`all` with --series, a .formattington.json config, and exit codes.
Messy samples live in a temporary sandbox (removed on exit); the
repository is never modified.

Flags:
  --help, -h      Show this help.

Env:
  VERSIONER_BIN    Path to versioner (default: via scripts/env.sh, then PATH).
  LOGAMUFFIN_BIN   Path to logamuffin for log() (default: via scripts/env.sh, then PATH).
  LOG_LEVEL        Minimum level for log() (default: INFO).
  FORMATTINGTON_VERBOSE  Show resolved executables (also via --verbose).
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
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

# --- Launcher: prefer uv (project-pinned), then PATH, then python -m. --
# `uv run --project $REPO_ROOT --python 3.14` keeps discovery cwd-scoped
# to the sandbox while still running this repo's checkout (plain
# `uv run` from the sandbox would fail to find the project, and a
# monorepo parent pin can force Python 3.13 otherwise).
REPO_ROOT="$SCRIPT_DIR"
cd "$REPO_ROOT"

if command -v uv >/dev/null 2>&1; then
    FMT_LAUNCHER=(uv run --project "$REPO_ROOT" --python 3.14 formattington)
    log INFO "formattington launcher: uv run --project $REPO_ROOT --python 3.14 formattington"
elif command -v formattington >/dev/null 2>&1; then
    FMT_LAUNCHER=(formattington)
    log INFO "formattington launcher: formattington (PATH)"
elif command -v python3.14 >/dev/null 2>&1; then
    PYTHONPATH="$REPO_ROOT/src:${PYTHONPATH:-}"
    export PYTHONPATH
    FMT_LAUNCHER=(python3.14 -m formattington)
    log INFO "formattington launcher: python3.14 -m formattington (PYTHONPATH=$REPO_ROOT/src)"
elif command -v python3 >/dev/null 2>&1; then
    PYTHONPATH="$REPO_ROOT/src:${PYTHONPATH:-}"
    export PYTHONPATH
    FMT_LAUNCHER=(python3 -m formattington)
    log INFO "formattington launcher: python3 -m formattington (PYTHONPATH=$REPO_ROOT/src)"
else
    die "no formattington launcher found (install uv, or formattington on PATH, or python3/python3.14)"
fi

fmt() {
    "${FMT_LAUNCHER[@]}" "$@"
}

# demo_run <formattington-args...> — log the command, run it without
# letting `set -e` abort the tour on the expected non-zero exits
# (1 changes required, 2 usage, 127 missing), then log the exit code
# with its meaning. Always returns 0 so the tour continues.
demo_run() {
    log INFO "run: formattington $*"
    local rc=0
    fmt "$@" || rc=$?
    local meaning="tool error"
    case "$rc" in
    0) meaning="ok" ;;
    1) meaning="formatting changes required" ;;
    2) meaning="usage error" ;;
    127) meaning="formatter not found (bundled missing and not on PATH)" ;;
    *) meaning="formatter returned $rc (preserved)" ;;
    esac
    log INFO "exit: $rc ($meaning)"
    return 0
}

# demo_run_tmp <formattington-args...> — same as demo_run, but from the
# sandbox so discovery is scoped to the demo files (relative paths) and
# every `all`/no-file-args call stays inside the sandbox.
demo_run_tmp() {
    log INFO "run (cwd=$TMP_DIR): formattington $*"
    local rc=0
    (cd "$TMP_DIR" && fmt "$@") || rc=$?
    local meaning="tool error"
    case "$rc" in
    0) meaning="ok" ;;
    1) meaning="formatting changes required" ;;
    2) meaning="usage error" ;;
    127) meaning="formatter not found (bundled missing and not on PATH)" ;;
    *) meaning="formatter returned $rc (preserved)" ;;
    esac
    log INFO "exit: $rc ($meaning)"
    return 0
}

section() {
    log INFO "=== $1 ==="
}

# --- Temp sandbox (explicit targets; path with spaces on purpose). ----
TMP_DIR=""
cleanup() {
    if [[ -n "${TMP_DIR:-}" && -d "$TMP_DIR" ]]; then
        rm -rf -- "$TMP_DIR"
    fi
}
trap cleanup EXIT

TMP_DIR="$(mktemp -d)"
SAMPLE_DIR="$TMP_DIR/demo files"
mkdir -p "$SAMPLE_DIR"
# shellcheck disable=SC2016 # $x is the messy script's variable, not ours.
printf 'if [ "$x" = "1" ]; then echo hi;fi\n' >"$SAMPLE_DIR/messy.sh"
printf 'x=1\n' >"$SAMPLE_DIR/messy.py"
printf '#Title\ntext   with   spaces\n' >"$SAMPLE_DIR/messy.md"
printf 'x=1\n' >"$SAMPLE_DIR/messy.toml"
printf 'int main(){return 0;}\n' >"$SAMPLE_DIR/messy.cpp"
MESSY_SH="$SAMPLE_DIR/messy.sh"
MESSY_PY="$SAMPLE_DIR/messy.py"
MESSY_MD="$SAMPLE_DIR/messy.md"
MESSY_TOML="$SAMPLE_DIR/messy.toml"
MESSY_CPP="$SAMPLE_DIR/messy.cpp"

section "1/8 logamuffin in action (log <level> <message>, never direct calls)"
log INFO "env cascade: FORMATTINGTON_PLATFORM=${FORMATTINGTON_PLATFORM:-unknown}"
log INFO "LOGAMUFFIN_BIN=${LOGAMUFFIN_BIN:-<not found; log() falls back to printf>}"
log INFO "versioner (direct call is versioner, not logamuffin): VERSIONER_BIN=${VERSIONER_BIN:-<not found>}"
if [[ -n "${VERSIONER_BIN:-}" ]]; then
    log INFO "run: versioner --version"
    "$VERSIONER_BIN" --version
    log INFO "versioner exit: 0 (ok)"
else
    log WARNING "versioner not found (run tools/versioner/update.bash, or set VERSIONER_BIN)"
fi
log INFO "threshold is LOG_LEVEL=${LOG_LEVEL:-INFO} by default: TRACE/DEBUG are suppressed, INFO and above show"
log TRACE "suppressed TRACE: nothing printed above for this line"
log DEBUG "suppressed DEBUG: nothing printed above for this line"
log INFO "above: two lines were suppressed (no TRACE/DEBUG output)"
log INFO "hello from INFO (stdout, with file:line source)"
LOG_LEVEL=TRACE log TRACE "TRACE via per-call LOG_LEVEL=TRACE (now shown)"
LOG_LEVEL=TRACE log DEBUG "DEBUG via per-call LOG_LEVEL=TRACE (now shown)"
log WARNING "hello from WARNING (stdout; WARNING is also an alias for WARN)"
log ERROR "hello from ERROR (stderr, level >= 40 goes to stderr)"
log CRITICAL "hello from CRITICAL (stderr, level 50)"
log 50 "numeric 50 resolves to CRITICAL (same as above)"
log 20 "numeric 20 resolves to INFO (same as INFO above)"
log INFO "watch the source field: it is this script's file:line per call, set by log(), not handwritten"
LOG_LEVEL=WARNING log INFO "filtered: LOG_LEVEL=WARNING suppresses INFO (rc still 0, nothing printed)"
log INFO "would-be failing demo_run below uses log INFO for exits, so expected 1/2/127 never abort the tour"

section "2/8 one interface: --help and --version (no formatter needed)"
demo_run --help
demo_run --version

section "3/8 bundled first, then PATH (--verbose / FORMATTINGTON_VERBOSE=1)"
log INFO "--verbose before the tool shows find_tool() resolution on stderr (bundled vs PATH)"
demo_run --verbose shfmt --version
demo_run --verbose clang-format --version
log INFO "note: clang-format on macos-x86_64 resolves to the legacy v22.1.8 bundle (see tools/README.md); other platforms use v23.1.3"
demo_run --verbose taplo --version
demo_run --verbose black --version
demo_run --verbose prettier --version
log INFO "env spelling works the same:"
log INFO "run: FORMATTINGTON_VERBOSE=1 formattington shfmt --version"
rc=0
FORMATTINGTON_VERBOSE=1 fmt shfmt --version || rc=$?
log INFO "exit: $rc (ok)"

section "4/8 single tool vs language alias, explicit files (paths with spaces)"
log INFO "explicit targets skip discovery; args stay discrete so 'demo files/messy.sh' is one path"
log INFO "bash -> shfmt (.sh/.bash)"
demo_run --check bash "$MESSY_SH"
log INFO "python/py -> black (.py, PATH-only; missing exits 127 and the tour continues)"
demo_run --check python "$MESSY_PY"
log INFO "md/markdown/html -> prettier (Markdown formatting lives in Prettier; .md/.markdown)"
demo_run --check md "$MESSY_MD"
log INFO "toml -> taplo (taplo fmt, .toml)"
demo_run --check toml "$MESSY_TOML"
log INFO "c/cpp/c++ -> clang-format (C/C++, bundled)"
demo_run --check c "$MESSY_CPP"

section "5/8 modes: --check lists (exit 1) vs --dry-run diffs (never -w)"
log INFO "--check before the tool is Formattington-owned (shfmt -l below); after the tool is forwarded verbatim (shfmt --version in section 3)"
demo_run --check shfmt "$MESSY_SH"
demo_run --dry-run shfmt "$MESSY_SH"
log INFO "per-tool mappings (same flags, different natives): shfmt -l/-d, clang-format --dry-run --Werror / --dry-run, black --check / --diff, prettier --check / --list-different, taplo fmt --check / --diff, powershell lint (no in-place)"
demo_run --dry-run black "$MESSY_PY"

section "6/8 all, --series, discovery, and exit aggregation (sandbox-scoped)"
log INFO "no-file-args discovery walks cwd, skipping hidden dirs and __pycache__/node_modules/vendor/venv/dist/build/tools/"
demo_run_tmp --dry-run shfmt
log INFO "default all is concurrent (output may interleave)"
demo_run_tmp --check all
log INFO "--series (before or after all) runs shfmt, clang-format, black, prettier, powershell, taplo in order"
demo_run_tmp --check --series all
log INFO "exit is the last non-zero in that order (missing = 127) or 0; a missing formatter never stops the others (watch powershell 127 above if pwsh is absent)"

section "7/8 config file: .formattington.json (exact argv lists, not shell strings)"
log INFO "repo sample (read-only, not auto-discovered):"
cat "$REPO_ROOT/.formattington.example.json"
log INFO "precedence (highest first): --config PATH (before or after all), FORMATTINGTON_CONFIG_PATH, ./.formattington.json in cwd only, then full-registry all"
log INFO "sandbox config: native flags verbatim per tool (shfmt -l, black --check), spaces stay single entries"
printf '{\n  "commands": [\n    ["shfmt", "-l", "demo files/messy.sh"],\n    ["black", "--check", "demo files/messy.py"]\n  ]\n}\n' >"$TMP_DIR/.formattington.json"
cat "$TMP_DIR/.formattington.json"
demo_run_tmp --series all --config .formattington.json
log INFO "empty commands [] would run nothing and exit 0; malformed/missing config exits 2 without falling back"

section "8/8 exit codes, friendly errors, and summary"
log INFO "unknown selector is a usage error (argparse prints usage, exit 2)"
demo_run not-a-tool
log INFO "check and dry-run together are exclusive (exit 2)"
demo_run --check --dry-run shfmt "$MESSY_SH"
log INFO "missing PATH-only backend is one line, no traceback, exit 127 (powershell needs pwsh + PSScriptAnalyzer)"
demo_run powershell --help
log INFO "you saw: one interface, aliases, --check/--dry-run, --series all, --config, --verbose, bundled-vs-PATH, spaces-safe explicit paths, discovery, and 0/1/2/127 exits"
log INFO "all demo files lived in $TMP_DIR and were not part of the repo; nothing in $REPO_ROOT was modified (besides .logs/log.log, gitignored)"
log INFO "next: read README.md (usage), run 'uv run pytest' for the suite, or ./release.bash --dry-run for the versioner dogfood"
