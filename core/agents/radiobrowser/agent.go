package radiobrowser

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
)

const (
	fetchAgentTimeout = 100000000000000 // 100s
	minTimeToRefresh  = 24 * time.Hour
	maxQueryVariables = 1000 // You can go higher than this, but just a precaution
)

type RadioBrowserAgent interface {
	GetRadioInfo(ctx context.Context) error
	ShouldPerformInitialScan(ctx context.Context) bool
	SubmitClick(ctx context.Context, id string) error
}

func GetRadioBrowser(ds model.DataStore) RadioBrowserAgent {
	return singleton.GetInstance(func() *radioBrowserAgent {
		return RadioBrowserConstructor(ds)
	})
}

func RadioBrowserConstructor(ds model.DataStore) *radioBrowserAgent {
	r := radioBrowserAgent{
		ds:      ds,
		baseUrl: conf.Server.RadioBrowser.BaseUrl,
	}
	hc := &http.Client{
		Timeout: fetchAgentTimeout,
	}
	r.client = NewClient(r.baseUrl, hc)
	return &r
}

type radioBrowserAgent struct {
	ds      model.DataStore
	baseUrl string
	client  *Client
}

func (r *radioBrowserAgent) ShouldPerformInitialScan(ctx context.Context) bool {
	ms, err := r.ds.Property(ctx).Get(model.PropLastRefresh)
	if err != nil {
		return true
	}
	if ms == "" {
		return true
	}
	i, _ := strconv.ParseInt(ms, 10, 64)
	since := time.Since(time.Unix(0, i*int64(time.Millisecond)))

	return since >= minTimeToRefresh
}

func (r *radioBrowserAgent) GetRadioInfo(ctx context.Context) error {
	radios, err := r.client.GetAllRadios(ctx)
	if err != nil {
		return err
	}
	return r.ds.WithTx(func(tx model.DataStore) error {
		existing, err := r.ds.RadioInfo(ctx).GetAllIds()

		if err != nil {
			return err
		}

		for _, radio := range *radios {
			name := strings.TrimSpace(radio.Name)

			// for some reason, some of the radios are only whitespace
			if len(name) == 0 {
				continue
			}

			model := &model.RadioInfo{
				ID:       radio.StationID,
				Name:     radio.Name,
				Url:      radio.Url,
				Homepage: radio.Homepage,
				Favicon:  radio.Favicon,
				BaseRadioInfo: model.BaseRadioInfo{
					Tags:        radio.Tags,
					Country:     radio.Country,
					CountryCode: radio.CountryCode,
					Codec:       radio.Codec,
					Bitrate:     radio.Bitrate,
				},
			}
			_, ok := existing[model.ID]

			if ok {
				err = r.ds.RadioInfo(ctx).Update(model)
				delete(existing, radio.ID)
			} else {
				err = r.ds.RadioInfo(ctx).Insert(model)
			}

			if err != nil {
				return err
			}
		}

		toDelete := []string{}

		for id := range existing {
			toDelete = append(toDelete, id)
			if len(toDelete) == maxQueryVariables {
				err := r.ds.RadioInfo(ctx).DeleteMany(toDelete)
				if err != nil {
					return err
				}

				toDelete = []string{}
			}
		}

		if len(toDelete) != 0 {
			err := r.ds.RadioInfo(ctx).DeleteMany(toDelete)
			if err != nil {
				return err
			}
		}

		millis := time.Now().UnixNano() / int64(time.Millisecond)
		_ = r.ds.Property(ctx).Put(model.PropLastRefresh, fmt.Sprint(millis))

		return nil
	})
}

func (r *radioBrowserAgent) SubmitClick(ctx context.Context, id string) error {
	return r.client.Click(ctx, id)
}
