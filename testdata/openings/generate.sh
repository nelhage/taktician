#!/bin/bash
set -eu
dir=$1
size=$(basename "$dir")
rm -f "$dir/*.ptn"

i=1
while read line; do
    {
        echo "[Size \"$size\"]"
        echo
        echo "$line"
    } > "$dir/$i.ptn"
    let i=i+1
done < "$dir/openings.txt"
