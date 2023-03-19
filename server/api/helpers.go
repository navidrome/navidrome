package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

func toAPITrack(mf model.MediaFile) Track {
	return Track{
		Type: "track",
		Id:   mf.ID,
		Attributes: &TrackAttributes{
			Album: mf.Album,
			//Albumartist:   mf.AlbumArtist,
			Artist: mf.Artist,
			//Bitrate:       mf.BitRate,
			//Bpm:           p(mf.Bpm),
			//Channels:      mf.Channels,
			//Comments:      p(mf.Comment),
			//Disc:          p(mf.DiscNumber),
			Duration: mf.Duration,
			//Genre:         p(mf.Genre),
			//Mimetype:      mf.ContentType(),
			//RecordingMbid: p(mf.MbzTrackID),
			//Size:          int(mf.Size),
			Title: mf.Title,
			//Track:         mf.TrackNumber,
			//TrackMbid:     p(mf.MbzReleaseTrackID),
			//Year:          p(mf.Year),
		},
	}
}

func toAPITracks(mfs model.MediaFiles) []Track {
	tracks := make([]Track, len(mfs))
	for i := range mfs {
		tracks[i] = toAPITrack(mfs[i])
	}
	return tracks
}

func p[T comparable](t T) *T {
	var zero T
	if t == zero {
		return nil
	}
	return &t
}

func v[T comparable](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// toQueryOptions convert a params struct to a model.QueryOptions struct, to be used by the
// GetAll and CountAll functions. It assumes all GetXxxxParams functions have the exact same structure.
func toQueryOptions(params GetTracksParams) model.QueryOptions {
	var filters squirrel.And
	parseFilter := func(fs *[]string, op func(f, v string) squirrel.Sqlizer) {
		if fs != nil {
			for _, f := range *fs {
				parts := strings.SplitN(f, ":", 2)
				filters = append(filters, op(parts[0], parts[1]))
			}
		}
	}
	parseFilter(params.FilterEquals, func(f, v string) squirrel.Sqlizer { return squirrel.Eq{f: v} })
	parseFilter(params.FilterContains, func(f, v string) squirrel.Sqlizer { return squirrel.Like{f: "%" + v + "%"} })
	parseFilter(params.FilterStartsWith, func(f, v string) squirrel.Sqlizer { return squirrel.Like{f: v + "%"} })
	parseFilter(params.FilterEndsWith, func(f, v string) squirrel.Sqlizer { return squirrel.Like{f: "%" + v} })
	parseFilter(params.FilterGreaterThan, func(f, v string) squirrel.Sqlizer { return squirrel.Gt{f: v} })
	parseFilter(params.FilterGreaterOrEqual, func(f, v string) squirrel.Sqlizer { return squirrel.GtOrEq{f: v} })
	parseFilter(params.FilterLessThan, func(f, v string) squirrel.Sqlizer { return squirrel.Lt{f: v} })
	parseFilter(params.FilterLessOrEqual, func(f, v string) squirrel.Sqlizer { return squirrel.LtOrEq{f: v} })
	offset := v(params.PageOffset)
	limit := v(params.PageLimit)
	return model.QueryOptions{Max: int(limit), Offset: int(offset), Filters: filters}
}

func apiErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var res ErrorObject
	switch err {
	case model.ErrNotAuthorized:
		res = ErrorObject{Status: p(strconv.Itoa(http.StatusForbidden)), Title: p(http.StatusText(http.StatusForbidden))}
	case model.ErrNotFound:
		res = ErrorObject{Status: p(strconv.Itoa(http.StatusNotFound)), Title: p(http.StatusText(http.StatusNotFound))}
	default:
		res = ErrorObject{Status: p(strconv.Itoa(http.StatusInternalServerError)), Title: p(http.StatusText(http.StatusInternalServerError))}
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(403)

	json.NewEncoder(w).Encode(ErrorList{[]ErrorObject{res}})
}

func validationErrorHandler(w http.ResponseWriter, message string, statusCode int) {
	_ = GetTracks400JSONResponse{BadRequestJSONResponse{Errors: []ErrorObject{
		{
			Status: p(strconv.Itoa(statusCode)),
			Title:  p(http.StatusText(statusCode)),
			Detail: p(message),
		},
	}}}.VisitGetTracksResponse(w)
}

func buildPaginationLinksAndMeta(totalItems int32, params GetTracksParams, resourceName string) (PaginationLinks, PaginationMeta) {
	pageLimit := *params.PageLimit
	pageOffset := *params.PageOffset

	totalPages := (totalItems + pageLimit - 1) / pageLimit
	currentPage := pageOffset/pageLimit + 1

	meta := PaginationMeta{
		CurrentPage: &currentPage,
		TotalItems:  &totalItems,
		TotalPages:  &totalPages,
	}

	var first, last, next, prev *string

	buildLink := func(page int32) *string {
		query := url.Values{}
		query.Add("page[offset]", strconv.Itoa(int(page*pageLimit)))
		query.Add("page[limit]", strconv.Itoa(int(pageLimit)))

		addFilterParams := func(paramName string, values *[]string) {
			if values == nil {
				return
			}
			for _, value := range *values {
				query.Add(paramName, value)
			}
		}

		addFilterParams("filter[equals]", params.FilterEquals)
		addFilterParams("filter[contains]", params.FilterContains)
		addFilterParams("filter[lessThan]", params.FilterLessThan)
		addFilterParams("filter[lessOrEqual]", params.FilterLessOrEqual)
		addFilterParams("filter[greaterThan]", params.FilterGreaterThan)
		addFilterParams("filter[greaterOrEqual]", params.FilterGreaterOrEqual)
		addFilterParams("filter[startsWith]", params.FilterStartsWith)
		addFilterParams("filter[endsWith]", params.FilterEndsWith)

		if params.Sort != nil {
			query.Add("sort", string(*params.Sort))
		}
		if params.Include != nil {
			query.Add("include", string(*params.Include))
		}

		link := resourceName
		if len(query) > 0 {
			link += "&" + query.Encode()
		}
		return &link
	}

	if totalPages > 0 {
		firstLink := buildLink(0)
		first = firstLink

		lastLink := buildLink(totalPages - 1)
		last = lastLink
	}

	if currentPage < totalPages {
		nextLink := buildLink(currentPage)
		next = nextLink
	}

	if currentPage > 1 {
		prevLink := buildLink(currentPage - 2)
		prev = prevLink
	}

	links := PaginationLinks{
		First: first,
		Last:  last,
		Next:  next,
		Prev:  prev,
	}

	return links, meta
}
