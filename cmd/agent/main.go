package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

func reportOnce(
	client *resty.Client,
	baseURL string,
	snapshot map[string]float64,
	delta int64,
) error {
	logger.Log.Sugar().Infof(
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
			logger.Log.Sugar().Warnf(
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
		logger.Log.Sugar().Warnf(
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
		logger.Log.Sugar().Warnf("failed to send counter PollCount+=%d: %v", delta, err)
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

func main() {
	flags := NewFlags()

	exitCode, err := flags.parse()
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}

	baseURL := flags.serverBaseURL()
	pollInterval := flags.pollEvery()
	reportInterval := flags.reportEvery()

	client := resty.New().
		SetTimeout(5 * time.Second)

	gaugeValues := make(map[string]float64, len(models.GaugeNames)+1)
	var ms runtime.MemStats
	var mu sync.Mutex
	var pollCountDelta int64
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	poll := func() {
		runtime.ReadMemStats(&ms)
		mu.Lock()
		models.UpdateGaugesFromMemStats(gaugeValues, &ms, rnd)
		mu.Unlock()
		atomic.AddInt64(&pollCountDelta, 1)
		logger.Log.Info("poll: MemStats + RandomValue updated")
	}

	poll()
	logger.Log.Sugar().Infof(
		"poll: initial update done; first metric report in %v",
		reportInterval,
	)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	go func() {
		for range pollTicker.C {
			poll()
		}
	}()

	go func() {
		snapshot, delta := takeReportSnapshot(&mu, gaugeValues, &pollCountDelta)
		if err := reportOnce(client, baseURL, snapshot, delta); err != nil {
			atomic.AddInt64(&pollCountDelta, delta)
		}
		for range reportTicker.C {
			snapshot, delta := takeReportSnapshot(&mu, gaugeValues, &pollCountDelta)
			if err := reportOnce(client, baseURL, snapshot, delta); err != nil {
				atomic.AddInt64(&pollCountDelta, delta)
			}
		}
	}()

	select {}
}
