package core

import (
	"context"
	"fmt"
	"strconv"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics requieres initialization. But not more than once
var prometheusMetricsInstance *PrometheusMetrics

type PrometheusMetrics struct {
	DbTotal           *prometheus.GaugeVec
	VersionInfo       *prometheus.GaugeVec
	LastMediaScan     *prometheus.GaugeVec
	MediaScansCounter *prometheus.CounterVec
}

func GetPrometheusMetrics() *PrometheusMetrics {
	if prometheusMetricsInstance == nil {
		var err error
		err, prometheusMetricsInstance = NewPrometheusMetrics()
		if prometheusMetricsInstance == nil {
			panic(fmt.Sprintf("Unable to create Prometheus metrics instance. Error: %v", err))
		}
		fmt.Printf("GetPrometheusMetrics: %v\n", prometheusMetricsInstance)
	}
	return prometheusMetricsInstance
}

func NewPrometheusMetrics() (error, *PrometheusMetrics) {
	res := &PrometheusMetrics{
		DbTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_model_totals",
				Help: "Total number of DB items per model",
			},
			[]string{"model"},
		),
		VersionInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "navidrome_info",
				Help: "Information about Navidrome version",
			},
			[]string{"version"},
		),
		LastMediaScan: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "media_scan_last",
				Help: "Last media scan timestamp by success",
			},
			[]string{"success"},
		),
		MediaScansCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "media_scans",
				Help: "Total success media scans by success",
			},
			[]string{"success"},
		),
	}

	err := prometheus.DefaultRegisterer.Register(res.DbTotal)
	if err != nil {
		log.Error("Unable to register db_model_totals metrics")
		return err, nil
	}
	err = prometheus.DefaultRegisterer.Register(res.VersionInfo)
	if err != nil {
		log.Error("Unable to register navidrome_info metrics")
		return err, nil
	}
	err = prometheus.DefaultRegisterer.Register(res.LastMediaScan)
	if err != nil {
		log.Error("Unable to register media_scan_last metrics")
		return err, nil
	}
	err = prometheus.DefaultRegisterer.Register(res.MediaScansCounter)
	if err != nil {
		log.Error("Unable to register media_scans metrics")
		return err, nil
	}
	return nil, res
}

func processSqlAggregateMetrics(ctx context.Context, dataStore model.DataStore, targetGauge *prometheus.GaugeVec) {
	albums_count, err := dataStore.Album(ctx).CountAll()
	if err != nil {
		log.Error("album CountAll error: %v\n", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "album"}).Set(float64(albums_count))

	songs_count, err := dataStore.MediaFile(ctx).CountAll()
	if err != nil {
		log.Error("media CountAll error: %v\n", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "media"}).Set(float64(songs_count))

	users_count, err := dataStore.User(ctx).CountAll()
	if err != nil {
		log.Error("user CountAll error: %v\n", err)
		return
	}
	targetGauge.With(prometheus.Labels{"model": "user"}).Set(float64(users_count))
}

func WriteInitialMetrics(metrics *PrometheusMetrics) {
	metrics.VersionInfo.With(prometheus.Labels{"version": consts.Version}).Set(1)
}

func WriteAfterScanMetrics(ctx context.Context, metrics *PrometheusMetrics, success bool) {
	sqlDB := db.Db()
	dataStore := persistence.New(sqlDB)
	processSqlAggregateMetrics(ctx, dataStore, metrics.DbTotal)

	scanLabels := prometheus.Labels{"success": strconv.FormatBool(success)}
	metrics.LastMediaScan.With(scanLabels).SetToCurrentTime()
	metrics.MediaScansCounter.With(scanLabels).Inc()
}
