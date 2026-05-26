package main

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"go.uber.org/zap"
)

func reportOnce(
	logg *zap.Logger,
	sender *metricSender,
	snapshot map[string]float64,
	delta int64,
) error {
	logg.Sugar().Infof(
		"report: sending batch (%d runtime + gopsutil gauges + RandomValue + PollCount(+%d)) via worker pool…",
		len(models.GaugeNames),
		delta,
	)

	if err := sender.sendSnapshot(snapshot, delta); err != nil {
		logg.Sugar().Warnf("failed to send metrics: %v", err)
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

// runRuntimePollLoop — первая горутина агента: опрос runtime.MemStats и PollCount.
func runRuntimePollLoop(
	logg *zap.Logger,
	interval time.Duration,
	ms *runtime.MemStats,
	mu *sync.Mutex,
	gaugeValues map[string]float64,
	rnd *rand.Rand,
	pollCountDelta *int64,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		pollOnce(logg, ms, mu, gaugeValues, rnd, pollCountDelta)
	}
}

// runReportLoop — горутина отправки: снимает метрики и шлёт их на сервер через worker pool.
func runReportLoop(
	logg *zap.Logger,
	interval time.Duration,
	mu *sync.Mutex,
	gaugeValues map[string]float64,
	pollCountDelta *int64,
	sender *metricSender,
) {
	snapshot, delta := takeReportSnapshot(mu, gaugeValues, pollCountDelta)
	if err := reportOnce(logg, sender, snapshot, delta); err != nil {
		atomic.AddInt64(pollCountDelta, delta)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		snapshot, delta := takeReportSnapshot(mu, gaugeValues, pollCountDelta)
		if err := reportOnce(logg, sender, snapshot, delta); err != nil {
			atomic.AddInt64(pollCountDelta, delta)
		}
	}
}
