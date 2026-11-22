#!/usr/bin/env bash
# Fails when a PR adds a //nolint directive to a Go file; a directive kept on an edited line is not new.
# Compares HEAD against $BASE_REF (default origin/master) and needs full history (fetch-depth: 0 in CI).
set -euo pipefail

BASE_REF="${BASE_REF:-origin/master}"

git diff --unified=0 --no-color "$BASE_REF"...HEAD -- '*.go' | awk '
function directives(s,    out) {
	out = ""
	while (match(s, /\/\/nolint(:[A-Za-z0-9_,-]+)?/)) {
		out = out " " substr(s, RSTART, RLENGTH)
		s = substr(s, RSTART + RLENGTH)
	}
	return out
}
/^diff --git / { header = 1; next }
/^@@/ {
	split($3, pos, ",")
	line = substr(pos[1], 2)
	header = 0
	delete removed
	next
}
header && /^\+\+\+ / { file = substr($0, 7); next }
header { next }
/^-/ {
	n = split(directives($0), ds, " ")
	for (i = 1; i <= n; i++) removed[ds[i]]++
	next
}
/^\+/ {
	n = split(directives($0), ds, " ")
	for (i = 1; i <= n; i++) {
		if (removed[ds[i]] > 0) { removed[ds[i]]--; continue }
		found++
		printf "%s:%d: new %s directive\n", file, line, ds[i]
		if (ENVIRON["GITHUB_ACTIONS"] == "true")
			printf "::error file=%s,line=%d::New %s directive. Fix the lint issue instead. If this is a real exception, a maintainer can add the allow-nolint label.\n", file, line, ds[i]
	}
	line++
}
END {
	if (found) {
		printf "\n%d new //nolint directive(s). Fix the lint issues instead; a maintainer can approve real exceptions with the allow-nolint label.\n", found
		exit 1
	}
}'
