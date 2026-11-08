#!/usr/bin/env bash
set -euo pipefail

(
    SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
    cd $SCRIPT_DIR
    for archive in *.tar.xz; do
        [ -f "$archive" ] || continue
        dir="${archive%.tar.xz}"
        mkdir -p "$dir"
        tar -xJf "$archive" -C "$dir" --strip-components=1
    done
)
