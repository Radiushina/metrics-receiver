package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/buildinfo"
	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v4/cpu"
	"go.uber.org/zap"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	runAgent()
}

func runAgent() {
	buildinfo.Print(buildVersion, buildDate, buildCommit)
	flags := NewFlags()

	exitCode, err := flags.parse()
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			if _, printErr := fmt.Fprintln(os.Stderr, err); printErr != nil {
				os.Exit(1)
			}
		}
		os.Exit(exitCode)
	}

	baseURL := flags.serverBaseURL()
	pollInterval := flags.pollEvery()
	reportInterval := flags.reportEvery()
	secretKey := flags.key
	rateLimit := flags.rateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}

	logg, err := logger.New("info")
	if err != nil {
		logg = zap.NewNop()
	}
	defer func() { _ = logg.Sync() }()

	client := resty.New().
		SetTimeout(5 * time.Second)

	cpuCount, err := cpu.Counts(true)
	if err != nil {
		logg.Sugar().Warnf("gopsutil: cpu count: %v, using 1", err)
		cpuCount = 1
	}
	gopsutilGaugeNames := models.GopsutilGaugeNames(cpuCount)

	sender := newMetricSender(int(rateLimit), client, secretKey, baseURL, gopsutilGaugeNames)

	gaugeValues := make(map[string]float64, len(models.GaugeNames)+len(gopsutilGaugeNames)+1)
	initGopsutilGauges(gaugeValues, gopsutilGaugeNames)

	var ms runtime.MemStats
	var mu sync.Mutex
	var pollCountDelta int64
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	pollOnce(logg, &ms, &mu, gaugeValues, rnd, &pollCountDelta)
	pollGopsutilOnce(logg, &mu, gaugeValues, cpuCount)
	logg.Sugar().Infof(
		"poll: initial update done; first metric report in %v (rate limit %d, cpus %d)",
		reportInterval,
		rateLimit,
		cpuCount,
	)

	var wg sync.WaitGroup

	// Горутина 1: runtime.MemStats (Alloc, HeapAlloc, …) и RandomValue.
	wg.Go(func() {
		runRuntimePollLoop(logg, pollInterval, &ms, &mu, gaugeValues, rnd, &pollCountDelta)
	})

	// Горутина 2: отправка метрик на сервер (worker pool, RATE_LIMIT).
	wg.Go(func() {
		runReportLoop(logg, reportInterval, &mu, gaugeValues, &pollCountDelta, sender)
	})

	// Горутина 3: gopsutil — TotalMemory, FreeMemory, CPUutilization0…N-1.
	wg.Go(func() {
		runGopsutilPollLoop(logg, pollInterval, &mu, gaugeValues, cpuCount)
	})

	wg.Wait()
}
