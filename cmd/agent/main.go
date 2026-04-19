package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

func main() {
	exitCode, err := parseFlags()
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}

	baseURL := serverBaseURL()
	pollInterval := getPollInterval()
	reportInterval := getReportInterval()

	log.Printf("agent: server %s, poll %v, report %v", baseURL, pollInterval, reportInterval)

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
		log.Printf("poll: MemStats + RandomValue updated")
	}

	poll()
	log.Printf("poll: initial update done; first metric report in %v", reportInterval)

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
		report := func() {
			delta := atomic.SwapInt64(&pollCountDelta, 0)

			mu.Lock()
			snapshot := make(map[string]float64, len(gaugeValues))
			for k, v := range gaugeValues {
				snapshot[k] = v
			}
			mu.Unlock()

			log.Printf("report: sending %d gauges + RandomValue + PollCount(+%d)…", len(models.GaugeNames), delta)

			for _, name := range models.GaugeNames {
				if err := agent.PostMetric(client, baseURL, name, models.Gauge, snapshot[name]); err != nil {
					log.Printf("failed to send gauge %s=%v: %v", name, snapshot[name], err)
				}
			}
			if err := agent.PostMetric(client, baseURL, "RandomValue", models.Gauge, snapshot["RandomValue"]); err != nil {
				log.Printf("failed to send gauge RandomValue=%v: %v", snapshot["RandomValue"], err)
			}

			if err := agent.PostIntMetric(client, baseURL, models.PollCount, models.Counter, delta); err != nil {
				log.Printf("failed to send counter PollCount+=%d: %v", delta, err)
				atomic.AddInt64(&pollCountDelta, delta)
			}
		}

		report()
		for range reportTicker.C {
			report()
		}
	}()

	select {}
}
