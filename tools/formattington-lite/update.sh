#!/usr/bin/env bash
# update.sh — refresh bundled formattington-lite binaries from fordaj/formattington releases.
#
# Thin wrapper around tools/update.sh, the shared implementation for all
# bundled helper tools (downloads the latest release's per-platform
# executables via gh, verifies SHA256 checksums, stages and swaps the
# bundle atomically). See tools/update.sh.md for the shared logic.
#
# Usage:
#   tools/formattington-lite/update.sh              # download latest, replace old versions
#   tools/formattington-lite/update.sh --dry-run    # resolve tag + list assets, change nothing
#   tools/formattington-lite/update.sh --help       # usage
#
# Prerequisites: gh (https://cli.github.com/) and tar/unzip (.7z assets
# additionally need 7z/7zz). Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback).
#
# Layout after a successful run (example):
#   tools/formattington-lite/
#     update.sh
#     v<tag>/
#       macos-arm64/formattington
#       macos-x86_64/formattington
#       linux-arm64/formattington
#       linux-x86_64/formattington
#       windows-arm64/formattington
#       windows-x86_64/formattington
#
# Env:
#   FORMATTINGTON_REPO  Repository slug (default: fordaj/formattington; overridden by --repo).
#   FORMATTINGTON_TAG   Pinned tag (default: latest release; overridden by --tag).
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
exec "$SCRIPT_DIR/../update.sh" formattington-lite "$@"
