#!/usr/bin/env bash

gofiles=$(git diff --name-only --diff-filter=ACM | grep '.go$')
[ -z "$gofiles" ] && exit 0

unformatted=$(gofmt -l $gofiles)
[ -z "$unformatted" ] && exit 0

for f in $unformatted; do
    go fmt "$f"
done