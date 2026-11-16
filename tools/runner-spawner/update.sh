#!/usr/bin/env bash
# update.sh — refresh bundled runner-spawner binaries from fordaj/runner-spawner releases.
#
# Thin wrapper around tools/update.sh, the shared implementation for all
# bundled helper tools (downloads the latest release's per-platform
# executables via gh, verifies SHA256 checksums, stages and swaps the
# bundle atomically). See tools/update.sh.md for the shared logic.
#
# Usage:
#   tools/runner-spawner/update.sh              # download latest, replace old versions
#   tools/runner-spawner/update.sh --dry-run    # resolve tag + list assets, change nothing
#   tools/runner-spawner/update.sh --help       # usage
#
# Prerequisites: gh (https://cli.github.com/) and tar/unzip (.7z assets
# additionally need 7z/7zz). Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback).
#
# Layout after a successful run (example):
#   tools/runner-spawner/
#     update.sh
#     v<tag>/
#       macos-arm64/runner-spawner
#       macos-x86_64/runner-spawner
#       linux-arm64/runner-spawner
#       linux-x86_64/runner-spawner
#       windows-arm64/runner-spawner
#       windows-x86_64/runner-spawner
#
# Env:
#   RUNNER_SPAWNER_TOOL_REPO  Repository slug (default: fordaj/runner-spawner; overridden by --repo).
#   RUNNER_SPAWNER_TAG   Pinned tag (default: latest release; overridden by --tag).
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
exec "$SCRIPT_DIR/../update.sh" runner-spawner "$@"
