#!/usr/bin/env bash

gofmtcmd=`which goimports || echo "gofmt"`

gofiles=$(git diff --name-only --diff-filter=ACM | grep '.go$')
[ -z "$gofiles" ] && exit 0

unformatted=`$gofmtcmd -l $gofiles`
[ -z "$unformatted" ] && exit 0

for f in $unformatted; do
    $gofmtcmd -w -l "$f"
    gofmt -s -w -l "$f"
done