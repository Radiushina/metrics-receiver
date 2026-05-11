package main

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

func reportOnce(
	logg *zap.Logger,
	client *resty.Client,
	secretKey, baseURL string,
	snapshot map[string]float64,
	delta int64,
) error {
	logg.Sugar().Infof(
		"report: sending %d gauges + RandomValue + PollCount(+%d) as batch…",
		len(models.GaugeNames),
		delta,
	)

	metrics := make([]models.Metrics, 0, len(models.GaugeNames)+2)

	for _, name := range models.GaugeNames {
		v := snapshot[name]
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}

	rv := snapshot["RandomValue"]
	metrics = append(metrics, models.Metrics{
		ID:    "RandomValue",
		MType: models.Gauge,
		Value: &rv,
	})

	metrics = append(metrics, models.Metrics{
		ID:    models.PollCount,
		MType: models.Counter,
		Delta: &delta,
	})

	if err := agent.PostMetricsBatch(client, secretKey, baseURL, metrics); err != nil {
		logg.Sugar().Warnf("failed to send batch metrics: %v", err)
		return err
	}
	return nil
}

func takeReportSnapshot(
	mu *sync.Mutex,
	gaugeValues map[string]float64,
	pollCountDelta *int64,
) (snapshot map[string]float64, delta int64) {
	delta = atomic.SwapInt64(pollCountDelta, 0)

	mu.Lock()
	snapshot = make(map[string]float64, len(gaugeValues))
	for k, v := range gaugeValues {
		snapshot[k] = v
	}
	mu.Unlock()

	return snapshot, delta
}

func pollOnce(
	logg *zap.Logger,
	ms *runtime.MemStats,
	mu *sync.Mutex,
	gaugeValues map[string]float64,
	rnd *rand.Rand,
	pollCountDelta *int64,
) {
	runtime.ReadMemStats(ms)

	mu.Lock()
	models.UpdateGaugesFromMemStats(gaugeValues, ms, rnd)
	mu.Unlock()

	atomic.AddInt64(pollCountDelta, 1)
	logg.Info("poll: MemStats + RandomValue updated")
}
