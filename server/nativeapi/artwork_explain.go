package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// explainableKinds is every kind the endpoint accepts, matching the CLI's explainKinds: a kind
// with no chain to walk still has stored state and config to report.
var explainableKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork, model.KindDiscArtwork,
	model.KindMediaFileArtwork, model.KindPlaylistArtwork, model.KindRadioArtwork,
}

type traceStepDTO struct {
	Candidate string `json:"candidate"`
	Outcome   string `json:"outcome"`
	Detail    string `json:"detail,omitempty"`
}

type storedDTO struct {
	Source      string `json:"source"`
	SourcePath  string `json:"sourcePath,omitempty"`
	AttemptedAt string `json:"attemptedAt,omitempty"`
}

type queuedDTO struct {
	Priority     int    `json:"priority"`
	PriorityName string `json:"priorityName"`
	Attempts     int    `json:"attempts"`
	RetryAt      string `json:"retryAt,omitempty"`
}

type configDTO struct {
	Setting string `json:"setting"`
	Value   string `json:"value"`
}

type explainDTO struct {
	Kind              string         `json:"kind"`
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Result            string         `json:"result"`
	ChainOrigin       string         `json:"chainOrigin"`
	Steps             []traceStepDTO `json:"steps"`
	Stored            *storedDTO     `json:"stored,omitempty"`
	Queued            *queuedDTO     `json:"queued,omitempty"`
	LastAttemptFailed []traceStepDTO `json:"lastAttemptFailed,omitempty"`
	GaveUpAfter       []traceStepDTO `json:"gaveUpAfter,omitempty"`
	Config            *configDTO     `json:"config,omitempty"`
	Agents            string         `json:"agents,omitempty"`
}

func (api *Router) addArtworkExplainRoute(r chi.Router) {
	r.Get("/artwork/explain", api.explainArtwork())
}

func (api *Router) explainArtwork() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		kind, ok := model.ParseKind(r.URL.Query().Get("kind"))
		if !ok || !slices.Contains(explainableKinds, kind) {
			http.Error(w, "invalid artwork kind", http.StatusBadRequest)
			return
		}
		id := r.URL.Query().Get("id")

		// No Walk: the endpoint reports history only, so it can never reach the network.
		rep, err := artwork.Explain(ctx, api.ds, api.agents, kind, id, artwork.ExplainOptions{})
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Error(ctx, "Error explaining artwork", "kind", kind, "id", id, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toExplainDTO(rep)); err != nil {
			log.Error(ctx, "Error encoding artwork explain response", "kind", kind, "id", id, err)
		}
	}
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func toStepDTOs(steps []artwork.TraceStep) []traceStepDTO {
	// Never nil: the UI maps over this unconditionally.
	out := make([]traceStepDTO, 0, len(steps))
	for _, s := range steps {
		out = append(out, traceStepDTO{Candidate: s.Candidate, Outcome: string(s.Outcome), Detail: s.Detail})
	}
	return out
}

func toExplainDTO(rep artwork.ExplainReport) explainDTO {
	dto := explainDTO{
		Kind:        rep.Kind.Prefix(),
		ID:          rep.ID,
		Name:        rep.Name,
		Result:      rep.Result(),
		ChainOrigin: rep.ChainOrigin(),
		Steps:       toStepDTOs(rep.Steps),
		Agents:      rep.Agents,
	}
	if rep.Stored != nil {
		dto.Stored = &storedDTO{
			Source:      rep.Stored.Source,
			SourcePath:  rep.Stored.SourcePath,
			AttemptedAt: rfc3339(rep.Stored.AttemptedAt),
		}
	}
	if rep.Queued != nil {
		dto.Queued = &queuedDTO{
			Priority:     rep.Queued.Priority,
			PriorityName: artwork.PriorityName(rep.Queued.Priority),
			Attempts:     rep.Queued.Attempts,
			RetryAt:      rfc3339(rep.Queued.RetryAt),
		}
	}
	if steps := rep.LastAttemptFailed(); len(steps) > 0 {
		dto.LastAttemptFailed = toStepDTOs(steps)
	}
	if steps := rep.GaveUpAfter(); len(steps) > 0 {
		dto.GaveUpAfter = toStepDTOs(steps)
	}
	if setting, value := artwork.ConfigFor(rep.Kind); setting != "" {
		dto.Config = &configDTO{Setting: setting, Value: value}
	}
	return dto
}
