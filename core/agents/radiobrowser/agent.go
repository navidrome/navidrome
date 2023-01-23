package radiobrowser

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

const (
	fetchAgentTimeout = 100000000000000 // 100s
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

		return nil
	})
}
