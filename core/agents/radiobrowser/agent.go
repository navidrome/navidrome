package radiobrowser

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

const (
	fetchAgentTimeout = 100000000000000 // 100s
	minTimeToRefresh  = 24 * time.Hour
)

type RadioBrowserAgent struct {
	ds      model.DataStore
	baseUrl string
	client  *Client
}

func RadioBrowserConstructor(ds model.DataStore) RadioBrowserAgent {
	r := RadioBrowserAgent{
		ds:      ds,
		baseUrl: conf.Server.RadioBrowser.BaseUrl,
	}
	hc := &http.Client{
		Timeout: fetchAgentTimeout,
	}
	r.client = NewClient(r.baseUrl, hc)
	return r
}

func (r *RadioBrowserAgent) ShouldPerformInitialScan(ctx context.Context) bool {
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

func (r *RadioBrowserAgent) GetRadioInfo(ctx context.Context) error {
	radios, err := r.client.GetAllRadios(ctx)
	if err != nil {
		return err
	}
	return r.ds.WithTx(func(tx model.DataStore) error {
		err := r.ds.RadioInfo(ctx).Purge()

		if err != nil {
			return err
		}

		for _, radio := range *radios {
			err = r.ds.RadioInfo(ctx).Insert(&model.RadioInfo{
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
			})

			if err != nil {
				return err
			}
		}

		millis := time.Now().UnixNano() / int64(time.Millisecond)
		r.ds.Property(ctx).Put(model.PropLastRefresh, fmt.Sprint(millis))

		return nil
	})
}
