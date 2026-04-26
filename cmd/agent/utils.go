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
	baseURL string,
	snapshot map[string]float64,
	delta int64,
) error {
	logg.Sugar().Infof(
		"report: sending %d gauges + RandomValue + PollCount(+%d)…",
		len(models.GaugeNames),
		delta,
	)

	for _, name := range models.GaugeNames {
		if err := agent.PostMetric(
			client,
			baseURL,
			name,
			models.Gauge,
			snapshot[name],
		); err != nil {
			logg.Sugar().Warnf(
				"failed to send gauge %s=%v: %v",
				name,
				snapshot[name],
				err,
			)
		}
	}

	if err := agent.PostMetric(
		client,
		baseURL,
		"RandomValue",
		models.Gauge,
		snapshot["RandomValue"],
	); err != nil {
		logg.Sugar().Warnf(
			"failed to send gauge RandomValue=%v: %v",
			snapshot["RandomValue"],
			err,
		)
	}

	if err := agent.PostIntMetric(
		client,
		baseURL,
		models.PollCount,
		models.Counter,
		delta,
	); err != nil {
		logg.Sugar().Warnf("failed to send counter PollCount+=%d: %v", delta, err)
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
