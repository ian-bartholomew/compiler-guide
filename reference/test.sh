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

# negative check: a program with no main must be rejected, not passed to gcc
echo 'func other(){ return 0; }' > "$tmp/nomain.tin"
if ./tinc "$tmp/nomain.tin" >/dev/null 2>&1; then
    echo "FAIL nomain (compiler accepted a program with no main)"
    fail=1
else
    echo "ok   nomain"
fi

exit $fail
