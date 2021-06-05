package utils

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ParamString(r *http.Request, param string) string {
	return r.URL.Query().Get(param)
}

func ParamStringDefault(r *http.Request, param, def string) string {
	v := ParamString(r, param)
	if v == "" {
		return def
	}
	return v
}

func ParamStrings(r *http.Request, param string) []string {
	return r.URL.Query()[param]
}

func ParamTimes(r *http.Request, param string) []time.Time {
	pStr := ParamStrings(r, param)
	times := make([]time.Time, len(pStr))
	for i, t := range pStr {
		ti, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			times[i] = ToTime(ti)
		}
	}
	return times
}

func ParamTime(r *http.Request, param string, def time.Time) time.Time {
	v := ParamString(r, param)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return ToTime(value)
}

func ParamInt(r *http.Request, param string, def int) int {
	v := ParamString(r, param)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return def
	}
	return int(value)
}

func ParamInt64(r *http.Request, param string, def int64) int64 {
	v := ParamString(r, param)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return value
}

func ParamInts(r *http.Request, param string) []int {
	pStr := ParamStrings(r, param)
	ints := make([]int, 0, len(pStr))
	for _, s := range pStr {
		i, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			ints = append(ints, int(i))
		}
	}
	return ints
}

func ParamBool(r *http.Request, param string, def bool) bool {
	p := strings.ToLower(ParamString(r, param))
	if p == "" {
		return def
	}
	return strings.Contains("/true/on/1/", "/"+p+"/")
}
