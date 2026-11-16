#!/usr/bin/env bash
# update.sh — refresh bundled versioner binaries from fordaj/Versioner releases.
#
# Thin wrapper around tools/update.sh, the shared implementation for all
# bundled helper tools (downloads the latest release's per-platform
# executables via gh, verifies SHA256 checksums, stages and swaps the
# bundle atomically). See tools/update.sh.md for the shared logic.
#
# Usage:
#   tools/versioner/update.sh              # download latest, replace old versions
#   tools/versioner/update.sh --dry-run    # resolve tag + list assets, change nothing
#   tools/versioner/update.sh --help       # usage
#
# Prerequisites: gh (https://cli.github.com/) and tar/unzip (.7z assets
# additionally need 7z/7zz). Logging uses log <level> <message> from
# scripts/entry.sh (logamuffin, with printf fallback).
#
# Layout after a successful run (example):
#   tools/versioner/
#     update.sh
#     v<tag>/
#       macos-arm64/versioner
#       macos-x86_64/versioner
#       linux-arm64/versioner
#       linux-x86_64/versioner
#       windows-arm64/versioner
#       windows-x86_64/versioner
#
# Env:
#   VERSIONER_REPO  Repository slug (default: fordaj/Versioner; overridden by --repo).
#   VERSIONER_TAG   Pinned tag (default: latest release; overridden by --tag).
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
exec "$SCRIPT_DIR/../update.sh" versioner "$@"
