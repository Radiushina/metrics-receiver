package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

var gaugeNames = []string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
	"MCacheSys",
	"MSpanInuse",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
}

/*const (
	metricTypeGauge   = "gauge"
	metricTypeCounter = "counter"
)*/

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

	gaugeValues := make(map[string]float64, len(gaugeNames)+1)
	var ms runtime.MemStats
	var mu sync.Mutex
	var pollCountDelta int64
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	poll := func() {
		runtime.ReadMemStats(&ms)
		mu.Lock()
		updateGaugesFromMemStats(gaugeValues, &ms, rnd)
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
		for range reportTicker.C {
			delta := atomic.SwapInt64(&pollCountDelta, 0)

			mu.Lock()
			snapshot := make(map[string]float64, len(gaugeValues))
			for k, v := range gaugeValues {
				snapshot[k] = v
			}
			mu.Unlock()

			log.Printf("report: sending %d gauges + RandomValue + PollCount(+%d)…", len(gaugeNames), delta)

			for _, name := range gaugeNames {
				if err := postMetric(client, baseURL, name, models.Gauge, snapshot[name]); err != nil {
					log.Printf("failed to send gauge %s=%v: %v", name, snapshot[name], err)
				}
			}
			if err := postMetric(client, baseURL, "RandomValue", models.Gauge, snapshot["RandomValue"]); err != nil {
				log.Printf("failed to send gauge RandomValue=%v: %v", snapshot["RandomValue"], err)
			}

			if delta > 0 {
				if err := postIntMetric(client, baseURL, models.PollCount, models.Counter, delta); err != nil {
					log.Printf("failed to send counter PollCount+=%d: %v", delta, err)
				}
			}
		}
	}()

	select {}
}

func updateGaugesFromMemStats(gaugeValues map[string]float64, ms *runtime.MemStats, rnd *rand.Rand) {
	gaugeValues["Alloc"] = float64(ms.Alloc)
	gaugeValues["BuckHashSys"] = float64(ms.BuckHashSys)
	gaugeValues["Frees"] = float64(ms.Frees)
	gaugeValues["GCCPUFraction"] = ms.GCCPUFraction
	gaugeValues["GCSys"] = float64(ms.GCSys)
	gaugeValues["HeapAlloc"] = float64(ms.HeapAlloc)
	gaugeValues["HeapIdle"] = float64(ms.HeapIdle)
	gaugeValues["HeapInuse"] = float64(ms.HeapInuse)
	gaugeValues["HeapObjects"] = float64(ms.HeapObjects)
	gaugeValues["HeapReleased"] = float64(ms.HeapReleased)
	gaugeValues["HeapSys"] = float64(ms.HeapSys)
	gaugeValues["LastGC"] = float64(ms.LastGC)
	gaugeValues["Lookups"] = float64(ms.Lookups)
	gaugeValues["MCacheInuse"] = float64(ms.MCacheInuse)
	gaugeValues["MCacheSys"] = float64(ms.MCacheSys)
	gaugeValues["MSpanInuse"] = float64(ms.MSpanInuse)
	gaugeValues["MSpanSys"] = float64(ms.MSpanSys)
	gaugeValues["Mallocs"] = float64(ms.Mallocs)
	gaugeValues["NextGC"] = float64(ms.NextGC)
	gaugeValues["NumForcedGC"] = float64(ms.NumForcedGC)
	gaugeValues["NumGC"] = float64(ms.NumGC)
	gaugeValues["OtherSys"] = float64(ms.OtherSys)
	gaugeValues["PauseTotalNs"] = float64(ms.PauseTotalNs)
	gaugeValues["StackInuse"] = float64(ms.StackInuse)
	gaugeValues["StackSys"] = float64(ms.StackSys)
	gaugeValues["Sys"] = float64(ms.Sys)
	gaugeValues["TotalAlloc"] = float64(ms.TotalAlloc)

	pick := gaugeNames[rnd.Intn(len(gaugeNames))]
	gaugeValues["RandomValue"] = gaugeValues[pick]
}

func postMetric(client *resty.Client, baseURL, name string, metricType models.MetricType, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("invalid float value: %v", value)
	}
	return basePostMetric(client, baseURL, name, metricType, &value, nil)
}

func postIntMetric(client *resty.Client, baseURL, name string, metricType models.MetricType, delta int64) error {
	return basePostMetric(client, baseURL, name, metricType, nil, &delta)
}

func basePostMetric(client *resty.Client, baseURL, name string, metricType models.MetricType, value *float64, delta *int64) error {
	switch metricType {
	case models.Gauge:
		if value == nil || delta != nil {
			return fmt.Errorf("gauge metric requires value, delta must be omitted")
		}
	case models.Counter:
		if delta == nil || value != nil {
			return fmt.Errorf("counter metric requires delta, value must be omitted")
		}
	default:
		return fmt.Errorf("unsupported metric type: %q", metricType)
	}

	metric := models.Metrics{
		ID:    name,
		MType: metricType,
		Delta: delta,
		Value: value,
	}

	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(baseURL)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status())
	}
	return nil
}
