package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
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

const (
	serverAddr = "http://localhost:8080"

	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second

	metricTypeGauge   = "gauge"
	metricTypeCounter = "counter"
)

func main() {
	log.Printf("agent: server %s, poll %v, report %v", serverAddr, pollInterval, reportInterval)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

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
				if err := postMetric(client, serverAddr, metricTypeGauge, name, snapshot[name]); err != nil {
					log.Printf("failed to send gauge %s=%v: %v", name, snapshot[name], err)
				}
			}
			if err := postMetric(client, serverAddr, metricTypeGauge, "RandomValue", snapshot["RandomValue"]); err != nil {
				log.Printf("failed to send gauge RandomValue=%v: %v", snapshot["RandomValue"], err)
			}

			if delta > 0 {
				if err := postIntMetric(client, serverAddr, metricTypeCounter, "PollCount", delta); err != nil {
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

func postMetric(client *http.Client, baseURL, metricType, name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("invalid float value: %v", value)
	}

	valueStr := strconv.FormatFloat(value, 'g', -1, 64)
	return basePostMetric(client, baseURL, metricType, name, valueStr)
}

func postIntMetric(client *http.Client, baseURL, metricType, name string, value int64) error {
	valueStr := strconv.FormatInt(value, 10)
	return basePostMetric(client, baseURL, metricType, name, valueStr)
}

func basePostMetric(client *http.Client, baseURL, metricType, name, valueStr string) error {
	nameEsc := url.PathEscape(name)
	valueEsc := url.PathEscape(valueStr)

	fullURL := fmt.Sprintf("%s/update/%s/%s/%s", baseURL, metricType, nameEsc, valueEsc)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, _ = io.CopyN(io.Discard, resp.Body, 512)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}
