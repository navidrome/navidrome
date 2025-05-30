package nativeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

// sensitiveFieldsPartialMask contains configuration field names that should be redacted
// using partial masking (first and last character visible, middle replaced with *).
// For values with 7+ characters: "secretvalue123" becomes "s***********3"
// For values with <7 characters: "short" becomes "****"
// Add field paths using dot notation (e.g., "LastFM.ApiKey", "Spotify.Secret")
var sensitiveFieldsPartialMask = []string{
	"LastFM.ApiKey",
	"LastFM.Secret",
	"Prometheus.MetricsPath",
	"Spotify.ID",
	"Spotify.Secret",
	"DevAutoLoginUsername",
}

// sensitiveFieldsFullMask contains configuration field names that should always be
// completely masked with "****" regardless of their length.
// Add field paths using dot notation for any fields that should never show any content.
var sensitiveFieldsFullMask = []string{
	"DevAutoCreateAdminPassword",
	"PasswordEncryptionKey",
	"Prometheus.Password",
}

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

func redactValue(key string, value string) string {
	// Return empty values as-is
	if len(value) == 0 {
		return value
	}

	// Check if this field should be fully masked
	for _, field := range sensitiveFieldsFullMask {
		if field == key {
			return "****"
		}
	}

	// Check if this field should be partially masked
	for _, field := range sensitiveFieldsPartialMask {
		if field == key {
			if len(value) < 7 {
				return "****"
			}
			// Show first and last character with * in between
			return string(value[0]) + strings.Repeat("*", len(value)-2) + string(value[len(value)-1])
		}
	}

	// Return original value if not sensitive
	return value
}

func flatten(ctx context.Context, entries *[]configEntry, prefix string, v reflect.Value) {
	if v.Kind() == reflect.Struct && v.Type().PkgPath() != "time" {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			flatten(ctx, entries, prefix+"."+t.Field(i).Name, v.Field(i))
		}
		return
	}

	key := strings.TrimPrefix(prefix, ".")
	envVar := "ND_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	var val interface{}
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		b, err := json.Marshal(v.Interface())
		if err != nil {
			log.Error(ctx, "Error marshalling config value", "key", key, err)
			val = "error marshalling value"
		} else {
			val = string(b)
		}
	default:
		originalValue := fmt.Sprint(v.Interface())
		val = redactValue(key, originalValue)
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
		flatten(ctx, &entries, fieldType.Name, fieldVal)
	}

	resp := configResponse{ID: "config", ConfigFile: conf.Server.ConfigFile, Config: entries}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(ctx, "Error encoding config response", err)
	}
}
