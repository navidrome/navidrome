package core

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/prometheus/client_golang/prometheus"
)

func WriteInitialMetrics() {
	getPrometheusMetrics().versionInfo.With(prometheus.Labels{"version": consts.Version}).Set(1)
}

func WriteAfterScanMetrics(ctx context.Context, dataStore model.DataStore, success bool) {
	processSqlAggregateMetrics(ctx, dataStore, getPrometheusMetrics().dbTotal)

	scanLabels := prometheus.Labels{"success": strconv.FormatBool(success)}
	getPrometheusMetrics().lastMediaScan.With(scanLabels).SetToCurrentTime()
	getPrometheusMetrics().mediaScansCounter.With(scanLabels).Inc()
}

// Prometheus' metrics requires initialization. But not more than once
var (
	prometheusMetricsInstance *prometheusMetrics
	prometheusOnce            sync.Once
)

type prometheusMetrics struct {
	dbTotal           *prometheus.GaugeVec
	versionInfo       *prometheus.GaugeVec
	lastMediaScan     *prometheus.GaugeVec
	mediaScansCounter *prometheus.CounterVec
}

func getPrometheusMetrics() *prometheusMetrics {
	prometheusOnce.Do(func() {
		var err error
		prometheusMetricsInstance, err = newPrometheusMetrics()
		if err != nil {
			log.Fatal("Unable to create Prometheus metrics instance.", err)
		}
	})
	return prometheusMetricsInstance
}

func newPrometheusMetrics() (*prometheusMetrics, error) {
	res := &prometheusMetrics{
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

	err := prometheus.DefaultRegisterer.Register(res.dbTotal)
	if err != nil {
		return nil, fmt.Errorf("unable to register db_model_totals metrics: %w", err)
	}
	err = prometheus.DefaultRegisterer.Register(res.versionInfo)
	if err != nil {
		return nil, fmt.Errorf("unable to register navidrome_info metrics: %w", err)
	}
	err = prometheus.DefaultRegisterer.Register(res.lastMediaScan)
	if err != nil {
		return nil, fmt.Errorf("unable to register media_scan_last metrics: %w", err)
	}
	err = prometheus.DefaultRegisterer.Register(res.mediaScansCounter)
	if err != nil {
		return nil, fmt.Errorf("unable to register media_scans metrics: %w", err)
	}
	return res, nil
}

func processSqlAggregateMetrics(ctx context.Context, dataStore model.DataStore, targetGauge *prometheus.GaugeVec) {
	albumsCount, err := dataStore.Album(ctx).CountAll()
	if err != nil {
		log.Warn("album CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "album"}).Set(float64(albumsCount))

	songsCount, err := dataStore.MediaFile(ctx).CountAll()
	if err != nil {
		log.Warn("media CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "media"}).Set(float64(songsCount))

	usersCount, err := dataStore.User(ctx).CountAll()
	if err != nil {
		log.Warn("user CountAll error", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "user"}).Set(float64(usersCount))
}
