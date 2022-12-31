package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

func (api *Router) CreateInternetRadio(r *http.Request) (*responses.Subsonic, error) {
	streamUrl, err := requiredParamString(r, "streamUrl")
	if err != nil {
		return nil, err
	}

	name, err := requiredParamString(r, "name")
	if err != nil {
		return nil, err
	}

	homepageUrl := utils.ParamString(r, "homepageUrl")
	ctx := r.Context()

	radio := &model.Radio{
		StreamUrl:   streamUrl,
		HomePageURL: homepageUrl,
		Name:        name,
	}

	err = api.ds.Radio(ctx).Put(radio)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) DeleteInternetRadio(r *http.Request) (*responses.Subsonic, error) {
	id, err := requiredParamString(r, "id")

	if err != nil {
		return nil, err
	}

	err = api.ds.Radio(r.Context()).Delete(id)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetInternetRadios(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	radios, err := api.ds.Radio(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	res := make([]responses.Radio, len(radios))
	for i, g := range radios {
		res[i] = responses.Radio{
			ID:          g.ID,
			Name:        g.Name,
			StreamUrl:   g.StreamUrl,
			HomepageUrl: g.HomePageURL,
		}
	}

	response := newResponse()
	response.InternetRadioStations = &responses.InternetRadioStations{
		Radios: res,
	}

	return response, nil
}

func (api *Router) UpdateInternetRadios(r *http.Request) (*responses.Subsonic, error) {
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}

	streamUrl, err := requiredParamString(r, "streamUrl")
	if err != nil {
		return nil, err
	}

	name, err := requiredParamString(r, "name")
	if err != nil {
		return nil, err
	}

	homepageUrl := utils.ParamString(r, "homepageUrl")
	ctx := r.Context()

	radio := &model.Radio{
		ID:          id,
		StreamUrl:   streamUrl,
		HomePageURL: homepageUrl,
		Name:        name,
	}

	err = api.ds.Radio(ctx).Put(radio)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}
