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

	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

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

	pollOnce(&ms, &mu, gaugeValues, rnd, &pollCountDelta)
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
			pollOnce(&ms, &mu, gaugeValues, rnd, &pollCountDelta)
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
