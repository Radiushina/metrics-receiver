package main

import (
	"crypto/rsa"
	"sync"

	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/pool"
	"github.com/go-resty/resty/v2"
)

// metricsPool переиспользует *models.Metrics между циклами отправки батча.
var metricsPool = pool.New(func() *models.Metrics {
	return &models.Metrics{}
})

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
	wg                 sync.WaitGroup
}

// newMetricSender создаёт пул из workers воркеров и запускает их горутины.
// workers (RATE_LIMIT / -l) — сколько батч-запросов к /updates/ может выполняться одновременно.
func newMetricSender(
	workers int,
	client *resty.Client,
	secretKey, baseURL, localIP string,
	gopsutilGaugeNames []string,
	publicKey *rsa.PublicKey,
) *metricSender {
	if workers < 1 {
		workers = 1
	}

	s := &metricSender{
		jobs:               make(chan batchJob, workers),
		gopsutilGaugeNames: gopsutilGaugeNames,
	}

	for range workers {
		s.wg.Go(func() {
			batchWorker(s.jobs, client, secretKey, baseURL, localIP, publicKey)
		})
	}

	return s
}

// Close закрывает очередь задач и ждёт завершения воркеров (в том числе in-flight отправок).
func (s *metricSender) Close() {
	close(s.jobs)
	s.wg.Wait()
}

// batchWorker читает batchJob из jobs и отправляет весь снимок одним вызовом PostMetricsBatch.
func batchWorker(
	jobs <-chan batchJob,
	client *resty.Client,
	secretKey, baseURL, localIP string,
	publicKey *rsa.PublicKey,
) {
	for job := range jobs {
		err := agent.PostMetricsBatch(client, secretKey, baseURL, localIP, job.metrics, publicKey)
		job.done <- err
	}
}

// buildMetricsFromSnapshot собирает []models.Metrics для одного батч-запроса из снимка gauge и delta PollCount.
func (s *metricSender) buildMetricsFromSnapshot(snapshot map[string]float64, delta int64) []models.Metrics {
	capacity := len(models.GaugeNames) + len(s.gopsutilGaugeNames) + 2
	metrics := make([]models.Metrics, 0, capacity)

	appendGauge := func(name string, value float64) {
		m := metricsPool.Get()
		m.ID = name
		m.MType = models.Gauge
		m.Value = &value
		m.Delta = nil
		metrics = append(metrics, *m)
		m.Value = nil
		m.Delta = nil
		metricsPool.Put(m)
	}

	for _, name := range models.GaugeNames {
		appendGauge(name, snapshot[name])
	}

	for _, name := range s.gopsutilGaugeNames {
		appendGauge(name, snapshot[name])
	}

	appendGauge("RandomValue", snapshot["RandomValue"])

	m := metricsPool.Get()
	m.ID = models.PollCount
	m.MType = models.Counter
	m.Delta = &delta
	m.Value = nil
	metrics = append(metrics, *m)
	m.Delta = nil
	m.Value = nil
	metricsPool.Put(m)

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
