package metrics

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics interface {
	WriteInitialMetrics(ctx context.Context)
	WriteAfterScanMetrics(ctx context.Context, success bool)
	RecordRequest(ctx context.Context, endpoint, method, client string, status int32, elapsed int64)
	RecordPluginRequest(ctx context.Context, plugin, method string, ok bool, elapsed int64)
	GetHandler() http.Handler
}

type metrics struct {
	ds model.DataStore
}

func GetPrometheusInstance(ds model.DataStore) Metrics {
	if !conf.Server.Prometheus.Enabled {
		return noopMetrics{}
	}

	return singleton.GetInstance(func() *metrics {
		return &metrics{ds: ds}
	})
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

func (m *metrics) RecordRequest(_ context.Context, endpoint, method, client string, status int32, elapsed int64) {
	httpLabel := prometheus.Labels{
		"endpoint": endpoint,
		"method":   method,
		"client":   client,
		"status":   strconv.FormatInt(int64(status), 10),
	}
	getPrometheusMetrics().httpRequestCounter.With(httpLabel).Inc()

	httpLatencyLabel := prometheus.Labels{
		"endpoint": endpoint,
		"method":   method,
		"client":   client,
	}
	getPrometheusMetrics().httpRequestDuration.With(httpLatencyLabel).Observe(float64(elapsed))
}

func (m *metrics) RecordPluginRequest(_ context.Context, plugin, method string, ok bool, elapsed int64) {
	pluginLabel := prometheus.Labels{
		"plugin": plugin,
		"method": method,
		"ok":     strconv.FormatBool(ok),
	}
	getPrometheusMetrics().pluginRequestCounter.With(pluginLabel).Inc()

	pluginLatencyLabel := prometheus.Labels{
		"plugin": plugin,
		"method": method,
	}
	getPrometheusMetrics().pluginRequestDuration.With(pluginLatencyLabel).Observe(float64(elapsed))
}

func (m *metrics) GetHandler() http.Handler {
	r := chi.NewRouter()

	if conf.Server.Prometheus.Password != "" {
		r.Use(middleware.BasicAuth("metrics", map[string]string{
			consts.PrometheusAuthUser: conf.Server.Prometheus.Password,
		}))
	}

	// Enable created at timestamp to handle zero counter on create.
	// This requires --enable-feature=created-timestamp-zero-ingestion to be passed in Prometheus
	r.Handle("/", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		EnableOpenMetrics:                   true,
		EnableOpenMetricsTextCreatedSamples: true,
	}))
	return r
}

type prometheusMetrics struct {
	dbTotal               *prometheus.GaugeVec
	versionInfo           *prometheus.GaugeVec
	lastMediaScan         *prometheus.GaugeVec
	mediaScansCounter     *prometheus.CounterVec
	httpRequestCounter    *prometheus.CounterVec
	httpRequestDuration   *prometheus.SummaryVec
	pluginRequestCounter  *prometheus.CounterVec
	pluginRequestDuration *prometheus.SummaryVec
}

// Prometheus' metrics requires initialization. But not more than once
var getPrometheusMetrics = sync.OnceValue(func() *prometheusMetrics {
	quartilesToEstimate := map[float64]float64{0.5: 0.05, 0.75: 0.025, 0.9: 0.01, 0.99: 0.001}

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
		httpRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_request_count",
				Help: "Request types by status",
			},
			[]string{"endpoint", "method", "client", "status"},
		),
		httpRequestDuration: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "http_request_latency",
				Help:       "Latency (in ms) of HTTP requests",
				Objectives: quartilesToEstimate,
			},
			[]string{"endpoint", "method", "client"},
		),
		pluginRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "plugin_request_count",
				Help: "Plugin requests by method/status",
			},
			[]string{"plugin", "method", "ok"},
		),
		pluginRequestDuration: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "plugin_request_latency",
				Help:       "Latency (in ms) of plugin requests",
				Objectives: quartilesToEstimate,
			},
			[]string{"plugin", "method"},
		),
	}

	prometheus.DefaultRegisterer.MustRegister(
		instance.dbTotal,
		instance.versionInfo,
		instance.lastMediaScan,
		instance.mediaScansCounter,
		instance.httpRequestCounter,
		instance.httpRequestDuration,
		instance.pluginRequestCounter,
		instance.pluginRequestDuration,
	)

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

func (n noopMetrics) RecordRequest(context.Context, string, string, string, int32, int64) {}

func (n noopMetrics) RecordPluginRequest(context.Context, string, string, bool, int64) {}

func (n noopMetrics) GetHandler() http.Handler { return nil }
