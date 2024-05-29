package req

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
)

type Values struct {
	*http.Request
}

func Params(r *http.Request) *Values {
	return &Values{r}
}

var (
	ErrMissingParam = errors.New("missing parameter")
	ErrInvalidParam = errors.New("invalid parameter")
)

func newError(err error, param string) error {
	return fmt.Errorf("%w: '%s'", err, param)
}
func (r *Values) String(param string) (string, error) {
	v := r.URL.Query().Get(param)
	if v == "" {
		return "", newError(ErrMissingParam, param)
	}
	return v, nil
}

func (r *Values) StringOr(param, def string) string {
	v, _ := r.String(param)
	if v == "" {
		return def
	}
	return v
}

func (r *Values) Strings(param string) ([]string, error) {
	values := r.URL.Query()[param]
	if len(values) == 0 {
		return nil, newError(ErrMissingParam, param)
	}
	return values, nil
}

func (r *Values) TimeOr(param string, def time.Time) time.Time {
	v, _ := r.String(param)
	if v == "" || v == "-1" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	t := time.UnixMilli(value)
	if t.Before(time.Date(1970, time.January, 2, 0, 0, 0, 0, time.UTC)) {
		return def
	}
	return t
}

func (r *Values) Times(param string) ([]time.Time, error) {
	pStr, err := r.Strings(param)
	if err != nil {
		return nil, err
	}
	times := make([]time.Time, len(pStr))
	for i, t := range pStr {
		ti, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			log.Warn(r.Context(), "Ignoring invalid time param", "time", t, err)
			times[i] = time.Now()
			continue
		}
		times[i] = time.UnixMilli(ti)
	}
	return times, nil
}

func (r *Values) Int64(param string) (int64, error) {
	v, err := r.String(param)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w '%s': expected integer, got '%s'", ErrInvalidParam, param, v)
	}
	return value, nil
}

func (r *Values) Int(param string) (int, error) {
	v, err := r.Int64(param)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

func (r *Values) IntOr(param string, def int) int {
	v, err := r.Int64(param)
	if err != nil {
		return def
	}
	return int(v)
}

func (r *Values) Int64Or(param string, def int64) int64 {
	v, err := r.Int64(param)
	if err != nil {
		return def
	}
	return v
}

func (r *Values) Ints(param string) ([]int, error) {
	pStr, err := r.Strings(param)
	if err != nil {
		return nil, err
	}
	ints := make([]int, 0, len(pStr))
	for _, s := range pStr {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			ints = append(ints, int(i))
		}
	}
	return ints, nil
}

func (r *Values) Bool(param string) (bool, error) {
	v, err := r.String(param)
	if err != nil {
		return false, err
	}
	return strings.Contains("/true/on/1/", "/"+strings.ToLower(v)+"/"), nil
}

func (r *Values) BoolOr(param string, def bool) bool {
	v, err := r.Bool(param)
	if err != nil {
		return def
	}
	return v
}
