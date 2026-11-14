#!/usr/bin/env bash
set -euo pipefail
(
    SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
    cd $SCRIPT_DIR

    keep=(
        "clang-format"
        "clang-format.exe"
        ".clang-format"
        "LICENSE.TXT"
        "README.md"
    )

    for file in * .[^.]*; do
        [[ -e "$file" ]] || continue

        keep_file=false
        for allowed in "${keep[@]}"; do
            if [[ "$file" == "$allowed" ]]; then
                keep_file=true
                break
            fi
        done

        if [[ "$keep_file" == false ]]; then
            # rm -rf -- "$file"
            :
        else
            echo "$file found"
        fi
    done
)
