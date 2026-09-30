#!/bin/sh
# Regenerates the host wrapper here and the Go/Rust clients the test plugins use.
set -e
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
(cd ../../cmd/ndpgen && go run . -host-wrappers -input=../../internal/streamtest -package=streamtest)
(cd ../../cmd/ndpgen && go run . -input=../../internal/streamtest -output="$out" -package=host -go -rust)
mkdir -p ../../testdata/test-stream-service/host ../../testdata/test-stream-service-rs/src
cp "$out/go/host/nd_host_streamtest.go" ../../testdata/test-stream-service/host/
cp "$out/rust/nd-pdk-host/src/nd_host_streamtest.rs" ../../testdata/test-stream-service-rs/src/streamtest.rs
