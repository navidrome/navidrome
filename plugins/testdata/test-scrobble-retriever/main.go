package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

func main() {

}

type TestScrobbleTimestampOutput struct {
	Timestamp *int64 `json:"timestamp"`
}

//go:wasmexport call_get_first_timestamp
func callGetFirstTimestamp() int32 {
	username := pdk.InputString()

	time, err := host.ScrobbleRetrieverGetFirstTimestamp(username)
	if err != nil {
		pdk.SetErrorString("failed to call scrobble retriever api " + err.Error())
		return 1
	}

	pdk.OutputJSON(TestScrobbleTimestampOutput{Timestamp: time})
	return 0
}

//go:wasmexport call_get_last_timestamp
func callGetLastTimestamp() int32 {
	username := pdk.InputString()

	time, err := host.ScrobbleRetrieverGetLastTimestamp(username)
	if err != nil {
		pdk.SetErrorString("failed to call scrobble retriever api " + err.Error())
		return 1
	}

	pdk.OutputJSON(TestScrobbleTimestampOutput{Timestamp: time})
	return 0
}
