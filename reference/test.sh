#!/usr/bin/env bash
# End-to-end check: compile each example, assemble+link with gcc, run it,
# and compare stdout to the matching .out file.
set -euo pipefail
cd "$(dirname "$0")"

go build -o tinc .
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

fail=0
for tin in examples/*.tin; do
    base=$(basename "$tin" .tin)
    ./tinc "$tin" > "$tmp/$base.s"
    gcc "$tmp/$base.s" -o "$tmp/$base"
    got=$("$tmp/$base")
    want=$(cat "examples/$base.out")
    if [ "$got" = "$want" ]; then
        echo "ok   $base"
    else
        echo "FAIL $base"
        echo "  got:  $got"
        echo "  want: $want"
        fail=1
    fi
done
exit $fail
