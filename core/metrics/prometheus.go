package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics interface {
	WriteInitialMetrics(ctx context.Context)
	WriteAfterScanMetrics(ctx context.Context, success bool)
	GetHandler() http.Handler
}

type metrics struct {
	ds model.DataStore
}

func NewPrometheusInstance(ds model.DataStore) Metrics {
	if conf.Server.Prometheus.Enabled {
		return &metrics{ds: ds}
	}
	return noopMetrics{}
}

func NewNoopInstance() Metrics {
	return noopMetrics{}
}

func (m *metrics) WriteInitialMetrics(ctx context.Context) {
	getPrometheusMetrics().versionInfo.With(prometheus.Labels{"version": consts.Version}).Set(1)
	processSqlAggregateMetrics(ctx, m.ds, getPrometheusMetrics().dbTotal)
}

func (m *metrics) WriteAfterScanMetrics(ctx context.Context, success bool) {
	processSqlAggregateMetrics(ctx, m.ds, getPrometheusMetrics().dbTotal)

	scanLabels := prometheus.Labels{"success": strconv.FormatBool(success)}
	getPrometheusMetrics().lastMediaScan.With(scanLabels).SetToCurrentTime()
	getPrometheusMetrics().mediaScansCounter.With(scanLabels).Inc()
}

func (m *metrics) GetHandler() http.Handler {
	r := chi.NewRouter()

	if conf.Server.Prometheus.Password != "" {
		r.Use(middleware.BasicAuth("metrics", map[string]string{
			consts.PrometheusAuthUser: conf.Server.Prometheus.Password,
		}))
	}
	r.Handle("/", promhttp.Handler())

	return r
}

type prometheusMetrics struct {
	dbTotal           *prometheus.GaugeVec
	versionInfo       *prometheus.GaugeVec
	lastMediaScan     *prometheus.GaugeVec
	mediaScansCounter *prometheus.CounterVec
}

// Prometheus' metrics requires initialization. But not more than once
var getPrometheusMetrics = sync.OnceValue(func() *prometheusMetrics {
	instance := &prometheusMetrics{
		dbTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_model_totals",
				Help: "Total number of DB items per model",
			},
			[]string{"model"},
		),
		versionInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "navidrome_info",
				Help: "Information about Navidrome version",
			},
			[]string{"version"},
		),
		lastMediaScan: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "media_scan_last",
				Help: "Last media scan timestamp by success",
			},
			[]string{"success"},
		),
		mediaScansCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "media_scans",
				Help: "Total success media scans by success",
			},
			[]string{"success"},
		),
	}
	err := prometheus.DefaultRegisterer.Register(instance.dbTotal)
	if err != nil {
		log.Fatal("Unable to create Prometheus metric instance", fmt.Errorf("unable to register db_model_totals metrics: %w", err))
	}
	err = prometheus.DefaultRegisterer.Register(instance.versionInfo)
	if err != nil {
		log.Fatal("Unable to create Prometheus metric instance", fmt.Errorf("unable to register navidrome_info metrics: %w", err))
	}
	err = prometheus.DefaultRegisterer.Register(instance.lastMediaScan)
	if err != nil {
		log.Fatal("Unable to create Prometheus metric instance", fmt.Errorf("unable to register media_scan_last metrics: %w", err))
	}
	err = prometheus.DefaultRegisterer.Register(instance.mediaScansCounter)
	if err != nil {
		log.Fatal("Unable to create Prometheus metric instance", fmt.Errorf("unable to register media_scans metrics: %w", err))
	}
	return instance
})

func processSqlAggregateMetrics(ctx context.Context, ds model.DataStore, targetGauge *prometheus.GaugeVec) {
	albumsCount, err := ds.Album(ctx).CountAll()
	if err != nil {
		log.Warn("album CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "album"}).Set(float64(albumsCount))

	artistCount, err := ds.Artist(ctx).CountAll()
	if err != nil {
		log.Warn("artist CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "artist"}).Set(float64(artistCount))

	songsCount, err := ds.MediaFile(ctx).CountAll()
	if err != nil {
		log.Warn("media CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "media"}).Set(float64(songsCount))

	usersCount, err := ds.User(ctx).CountAll()
	if err != nil {
		log.Warn("user CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "user"}).Set(float64(usersCount))
}

type noopMetrics struct {
}

func (n noopMetrics) WriteInitialMetrics(context.Context) {}

func (n noopMetrics) WriteAfterScanMetrics(context.Context, bool) {}

func (n noopMetrics) GetHandler() http.Handler { return nil }
