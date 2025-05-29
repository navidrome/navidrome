package nativeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model/request"
)

type configEntry struct {
	Key    string      `json:"key"`
	EnvVar string      `json:"envVar"`
	Value  interface{} `json:"value"`
}

type configResponse struct {
	ID         string        `json:"id"`
	ConfigFile string        `json:"configFile"`
	Config     []configEntry `json:"config"`
}

func flatten(entries *[]configEntry, prefix string, v reflect.Value) {
	if v.Kind() == reflect.Struct && v.Type().PkgPath() != "time" {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			flatten(entries, prefix+"."+t.Field(i).Name, v.Field(i))
		}
		return
	}

	key := strings.TrimPrefix(prefix, ".")
	envVar := "ND_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	var val interface{}
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		b, _ := json.Marshal(v.Interface())
		val = string(b)
	default:
		val = fmt.Sprint(v.Interface())
	}

	*entries = append(*entries, configEntry{Key: key, EnvVar: envVar, Value: val})
}

func getConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)
	if !user.IsAdmin {
		http.Error(w, "Config endpoint is only available to admin users", http.StatusUnauthorized)
		return
	}

	entries := make([]configEntry, 0)
	v := reflect.ValueOf(*conf.Server)
	t := reflect.TypeOf(*conf.Server)
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)
		flatten(&entries, fieldType.Name, fieldVal)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })

	resp := configResponse{ID: "config", ConfigFile: conf.Server.ConfigFile, Config: entries}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
