#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <test log>" >&2
  exit 1
fi

log_file="$1"

awk '
  /^\s*\[FAIL\]/ {
    name=$0
    sub(/^\s*\[FAIL\]\s*/, "", name)
    getline
    if (match($0, /([^ ]+\.go):([0-9]+)/, m)) {
      file=m[1]; line_no=m[2]
      ws=ENVIRON["GITHUB_WORKSPACE"]
      if (ws != "" && index(file, ws) == 1) {
        file=substr(file, length(ws)+2)
      }
      printf "::error file=%s,line=%s,title=Test Failure::%s\n", file, line_no, name
    }
  }
' "$log_file"

