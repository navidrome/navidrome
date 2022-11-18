package core

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/prometheus/client_golang/prometheus"
)

// typical Prometheus scrape interval is in 10..30 seconds range
const METRICS_INTERVAL = 15 * time.Second

type PrometheusMetrics struct {
	DbTotal *prometheus.GaugeVec
}

func NewPrometheusMetrics() *PrometheusMetrics {
	res := &PrometheusMetrics{
		DbTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_model_totals",
				Help: "Total number of DB items per model",
			},
			[]string{"model"},
		),
	}
	err := prometheus.DefaultRegisterer.Register(res.DbTotal)
	if err != nil {
		log.Error("Unable to register db_model_totals metrics")
		return nil
	}
	return res
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

func processMetrics(ctx context.Context, dataStore model.DataStore, metrics *PrometheusMetrics) {
	processSqlAggregateMetrics(ctx, dataStore, metrics.DbTotal)
}

func MetricsWorker() {
	sqlDB := db.Db()
	dataStore := persistence.New(sqlDB)
	ctx := context.Background()
	metrics := NewPrometheusMetrics()
	if metrics == nil {
		log.Error("Unable to create Prometheus metrics")
		return
	}

	for {
		begin_at := float64(time.Now().UnixNano()) / 1000_000_000
		processMetrics(ctx, dataStore, metrics)
		elapsed := float64(time.Now().UnixNano())/1000_000_000 - begin_at
		log.Debug(fmt.Sprintf("Metrics collecting takes %.5f s\n", elapsed))

		time.Sleep(METRICS_INTERVAL)
	}
}
