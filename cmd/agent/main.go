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

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	// сюда будет складыать называния метрики (ключ) и их значения метрики (значение мапы)
	gaugeValues := make(map[string]float64, len(gaugeNames)+1)
	// pollCountDelta (тип counter) - счетчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime,
	// на каждый pollInterval
	var pollCountDelta int64

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	pollCountDelta = 1

	updateGaugesFromMemStats(gaugeValues, &ms)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			fmt.Printf("Обновляем метрики [%d, %f]\n", pollCountDelta, gaugeValues["RandomValue"])
			runtime.ReadMemStats(&ms)
			pollCountDelta++
			updateGaugesFromMemStats(gaugeValues, &ms)
		case <-reportTicker.C:
			fmt.Printf("\nНачинаем слать запрос\n\n")

			for _, name := range gaugeNames {
				value := gaugeValues[name]
				if err := postMetric(client, serverAddr, metricTypeGauge, name, value); err != nil {
					log.Printf("failed to send gauge %s=%v: %v", name, value, err)
				}
			}

			if err := postMetric(client, serverAddr, metricTypeGauge, "RandomValue", gaugeValues["RandomValue"]); err != nil {
				log.Printf("failed to send gauge RandomValue=%v: %v", gaugeValues["RandomValue"], err)
			}

			if pollCountDelta > 0 {
				if err := postIntMetric(client, serverAddr, metricTypeCounter, "PollCount", pollCountDelta); err != nil {
					log.Printf("failed to send counter PollCount+=%d: %v", pollCountDelta, err)
				} else {
					pollCountDelta = 0
				}
			}
		}
	}
}

func updateGaugesFromMemStats(gaugeValues map[string]float64, ms *runtime.MemStats) {
	// Обновляем gaugeValues
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

	pick := gaugeNames[rand.Intn(len(gaugeNames))]
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

	// Эта строка читает и выбрасывает начало тела ответа сервера.
	// После client.Do проверяем StatusCode, но тело часто не нужно. Если сразу выйти, не прочитав тело,
	// соединение иногда нельзя нормально переиспользовать.
	//Поэтому частично дренируют тело: читают до 512 байт (обычно ответ пустой или короткий — этого хватает),
	//потом defer resp.Body.Close() закрывает поток.
	_, _ = io.CopyN(io.Discard, resp.Body, 512)

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
	}
	return nil
}
