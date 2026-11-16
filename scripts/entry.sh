#!/usr/bin/env bash
set -euo pipefail

ENTRY_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$ENTRY_DIR/env.sh"

# level_value <word-or-number> — map a log level to its numeric value.
# Unknown words map to INFO (20).
level_value() {
  local level="$1"
  if [[ "$level" =~ ^-?[0-9]+$ ]]; then
    printf '%s' "$level"
    return
  fi
  local upper
  upper="$(printf '%s' "$level" | tr '[:lower:]' '[:upper:]')"
  case "$upper" in
  TRACE) printf '%s' 5 ;;
  DEBUG) printf '%s' 10 ;;
  INFO) printf '%s' 20 ;;
  WARNING | WARN) printf '%s' 30 ;;
  ERROR) printf '%s' 40 ;;
  CRITICAL | FATAL) printf '%s' 50 ;;
  ALWAYS) printf '%s' -1 ;;
  *) printf '%s' 20 ;;
  esac
}

# log_fallback <SRC> <LEVEL> <MESSAGE...> — printf implementation of the
# log contract used when no logamuffin binary is available (bootstrap /
# fresh clone). LEVEL may be a word (INFO) or a number (50). Messages at
# or above the LOG_LEVEL/LOGAMUFFIN_LEVEL threshold are printed; ERROR
# and above go to stderr. Returns 2 for a missing LEVEL.
log_fallback() {
  local src="$1"
  shift
  if [[ $# -eq 0 ]]; then
    printf '%s\n' "log: usage: log LEVEL MESSAGE..." >&2
    return 2
  fi
  local level_arg="$1"
  shift
  local msg="$*"

  local level threshold
  level="$(level_value "$level_arg")"
  threshold="$(level_value "${LOG_LEVEL:-${LOGAMUFFIN_LEVEL:-INFO}}")"
  if [[ "$level" -ne -1 && "$level" -lt "$threshold" ]]; then
    return 0
  fi
  if [[ "$level" -ge 40 ]]; then
    printf '%s: [%s] %s\n' "$src" "$level_arg" "$msg" >&2
  else
    printf '%s: [%s] %s\n' "$src" "$level_arg" "$msg"
  fi
}

# log LEVEL [MESSAGE ...]
#   LEVEL may be a word (INFO) or a number (50). When a logamuffin
#   binary is available all arguments are forwarded to it with
#   LOG_SOURCE set from the call site; otherwise a printf fallback
#   keeps the same `log LEVEL MESSAGE` call shape working (see the
#   header for filtering/stream semantics). The exit code is
#   preserved (2 for usage errors, 0 otherwise for the fallback).
log() {
  local bin
  bin="$LOGAMUFFIN_BIN"

  # Caller context -> LOG_SOURCE="file[:function]:line". BASH_SOURCE[1]
  # is the caller's file, BASH_LINENO[0] is the line that called
  # log(), and FUNCNAME[1] is the calling function (empty at top
  # level). Defaults keep `set -u` safe.
  local caller_file="${BASH_SOURCE[1]:-}"
  local caller_line="${BASH_LINENO[0]:-1}"
  local caller_func=""
  if ((${#FUNCNAME[@]} > 1)); then
    caller_func="${FUNCNAME[1]:-}"
  fi
  if [[ -z "$caller_file" ]]; then
    caller_file="${0:-log}"
  fi

  local src
  if [[ -n "$caller_func" && "$caller_func" != "log" && "$caller_func" != "main" ]]; then
    src="${caller_file}:${caller_func}:${caller_line}"
  else
    src="${caller_file}:${caller_line}"
  fi

  if [[ ! -x "$bin" ]]; then
    log_fallback "$src" "$@"
    return $?
  fi

  LOG_SOURCE="$src" "$bin" "$@"
  # Preserve `logamuffin`'s exit code (0 success/filtered,
  # 2 usage/config errors) even when `set -e` is active.
  return $?
}
