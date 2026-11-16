#!/usr/bin/env bash
# update.sh — refresh bundled barnacle binaries from fordaj/barnacle releases.
#
# Thin wrapper around tools/update.sh, the shared implementation for all
# bundled helper tools (downloads the latest release's per-platform
# executables via gh, verifies SHA256 checksums, stages and swaps the
# bundle atomically). See tools/update.sh.md for the shared logic.
#
# Usage:
#   tools/barnacle/update.sh              # download latest, replace old versions
#   tools/barnacle/update.sh --dry-run    # resolve tag + list assets, change nothing
#   tools/barnacle/update.sh --help       # usage
#
# Prerequisites: gh (https://cli.github.com/) and tar/unzip (.7z assets
# additionally need 7z/7zz). Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback).
#
# Layout after a successful run (example):
#   tools/barnacle/
#     update.sh
#     v<tag>/
#       macos-arm64/barnacle
#       macos-x86_64/barnacle
#       linux-arm64/barnacle
#       linux-x86_64/barnacle
#       windows-arm64/barnacle
#       windows-x86_64/barnacle
#
# Env:
#   BARNACLE_REPO  Repository slug (default: fordaj/barnacle; overridden by --repo).
#   BARNACLE_TAG   Pinned tag (default: latest release; overridden by --tag).
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
exec "$SCRIPT_DIR/../update.sh" barnacle "$@"
