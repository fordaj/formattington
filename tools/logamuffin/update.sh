#!/usr/bin/env bash
# update.sh — refresh bundled logamuffin binaries from fordaj/logamuffin releases.
#
# Thin wrapper around tools/update.sh, the shared implementation for all
# bundled helper tools (downloads the latest release's per-platform
# executables via gh, verifies SHA256 checksums, stages and swaps the
# bundle atomically). See tools/update.sh.md for the shared logic.
#
# Usage:
#   tools/logamuffin/update.sh              # download latest, replace old versions
#   tools/logamuffin/update.sh --dry-run    # resolve tag + list assets, change nothing
#   tools/logamuffin/update.sh --help       # usage
#
# Prerequisites: gh (https://cli.github.com/) and tar/unzip (.7z assets
# additionally need 7z/7zz). Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback).
#
# Layout after a successful run (example):
#   tools/logamuffin/
#     update.sh
#     v<tag>/
#       macos-arm64/logamuffin
#       macos-x86_64/logamuffin
#       linux-arm64/logamuffin
#       linux-x86_64/logamuffin
#       windows-arm64/logamuffin
#       windows-x86_64/logamuffin
#
# Env:
#   LOGAMUFFIN_REPO  Repository slug (default: fordaj/logamuffin; overridden by --repo).
#   LOGAMUFFIN_TAG   Pinned tag (default: latest release; overridden by --tag).
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
exec "$SCRIPT_DIR/../update.sh" logamuffin "$@"
