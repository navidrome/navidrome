gofmtcmd="go run golang.org/x/tools/cmd/goimports"

gofiles=$(find . -name '*.go' | grep -v '_gen.go$')
[ -z "$gofiles" ] && exit 0

unformatted=$($gofmtcmd -l $gofiles)
[ -z "$unformatted" ] && exit 0

# Some files are not gofmt'd. Print message and fail.

echo >&2 "Go files must be formatted with '$gofmtcmd'. Please run:"
for fn in $unformatted; do
	echo >&2 "  $gofmtcmd -w $PWD/$fn"
done

exit 1
