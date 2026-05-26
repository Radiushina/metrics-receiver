package main

import (
	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

// batchJob — задача воркеру: отправить один пакет метрик на POST /updates/.
type batchJob struct {
	metrics []models.Metrics
	done    chan error // воркер передаёт сюда результат PostMetricsBatch
}

// metricSender — пул воркеров (worker pool) для исходящих батч-запросов.
type metricSender struct {
	// jobs — очередь батчей; буфер = workers, чтобы отправитель не блокировался,
	// пока воркеры обрабатывают предыдущие запросы.
	jobs               chan batchJob
	gopsutilGaugeNames []string
}

// newMetricSender создаёт пул из workers воркеров и запускает их горутины.
// workers (RATE_LIMIT / -l) — сколько батч-запросов к /updates/ может выполняться одновременно.
func newMetricSender(
	workers int,
	client *resty.Client,
	secretKey, baseURL string,
	gopsutilGaugeNames []string,
) *metricSender {
	if workers < 1 {
		workers = 1
	}

	s := &metricSender{
		jobs:               make(chan batchJob, workers),
		gopsutilGaugeNames: gopsutilGaugeNames,
	}

	for range workers {
		go batchWorker(s.jobs, client, secretKey, baseURL)
	}

	return s
}

// batchWorker читает batchJob из jobs и отправляет весь снимок одним вызовом PostMetricsBatch.
func batchWorker(
	jobs <-chan batchJob,
	client *resty.Client,
	secretKey, baseURL string,
) {
	for job := range jobs {
		err := agent.PostMetricsBatch(client, secretKey, baseURL, job.metrics)
		job.done <- err
	}
}

// buildMetricsFromSnapshot собирает []models.Metrics для одного батч-запроса из снимка gauge и delta PollCount.
func (s *metricSender) buildMetricsFromSnapshot(snapshot map[string]float64, delta int64) []models.Metrics {
	capacity := len(models.GaugeNames) + len(s.gopsutilGaugeNames) + 2
	metrics := make([]models.Metrics, 0, capacity)

	for _, name := range models.GaugeNames {
		v := snapshot[name]
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}

	for _, name := range s.gopsutilGaugeNames {
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

	return metrics
}

// sendSnapshot ставит в очередь один батч со всеми метриками снимка и ждёт его отправки.
// За цикл отчёта — один HTTP-запрос на /updates/; RATE_LIMIT ограничивает число одновременных батчей.
func (s *metricSender) sendSnapshot(snapshot map[string]float64, delta int64) error {
	metrics := s.buildMetricsFromSnapshot(snapshot, delta)

	done := make(chan error, 1)
	s.jobs <- batchJob{metrics: metrics, done: done}

	return <-done
}
