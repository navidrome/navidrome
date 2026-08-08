package main

import (
	"strconv"

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

type TestScrobbleOptions struct {
	Username      string `json:"username"`
	FromTimestamp *int64 `json:"fromTimestamp,omitempty"`
	ToTimestamp   *int64 `json:"toTimestamp,omitempty"`
	Descending    bool   `json:"descending"`
	Offset        int    `json:"offset,omitempty"`
	MaxItems      int    `json:"maxItems"`
}

// TestScrobblePage mirrors the multi-return shape so tests can assert on both values
type TestScrobblePage struct {
	Scrobbles []host.ScrobbleRef    `json:"scrobbles"`
	Next      *host.ScrobbleOptions `json:"next"`
}

//go:wasmexport call_get_scrobbles
func callGetScrobbles() int32 {
	var options TestScrobbleOptions
	err := pdk.InputJSON(&options)

	if err != nil {
		pdk.SetErrorString("failed to deserialize input " + err.Error())
		return 1
	}

	scrobbles, next, err := host.ScrobbleRetrieverGetScrobbles(options.Username, host.ScrobbleOptions{
		FromTimestamp: options.FromTimestamp,
		ToTimestamp:   options.ToTimestamp,
		MaxItems:      options.MaxItems,
		Descending:    options.Descending,
		Offset:        options.Offset,
	})

	if err != nil {
		pdk.SetErrorString("failed to call scrobble retriever api " + err.Error())
		return 1
	}

	if scrobbles == nil {
		scrobbles = []host.ScrobbleRef{}
	}

	pdk.OutputJSON(TestScrobblePage{Scrobbles: scrobbles, Next: next})
	return 0
}

type TestScrobbleCountOptions struct {
	Username      string `json:"username"`
	FromTimestamp *int64 `json:"fromTimestamp,omitempty"`
	ToTimestamp   *int64 `json:"toTimestamp,omitempty"`
}

//go:wasmexport call_get_scrobbles_count
func callGetScrobblesCount() int32 {
	var options TestScrobbleOptions
	err := pdk.InputJSON(&options)

	if err != nil {
		pdk.SetErrorString("failed to deserialize input " + err.Error())
		return 1
	}

	count, err := host.ScrobbleRetrieverGetScrobbleCount(options.Username, host.ScrobbleCountOptions{
		FromTimestamp: options.FromTimestamp,
		ToTimestamp:   options.ToTimestamp,
	})

	if err != nil {
		pdk.SetErrorString("failed to call scrobble retriever api " + err.Error())
		return 1
	}

	pdk.OutputString(strconv.FormatInt(count, 10))
	return 0
}
