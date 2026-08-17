package main

import (
	"context"
	"crypto/rsa"
	"sync"

	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/pool"
	pb "github.com/Radiushina/metrics-receiver.git/internal/proto"
	"github.com/go-resty/resty/v2"
	"google.golang.org/grpc"
)

// metricsPool переиспользует *models.Metrics между циклами отправки батча.
var metricsPool = pool.New(func() *models.Metrics {
	return &models.Metrics{}
})

// batchJob — задача воркеру: отправить один пакет метрик.
type batchJob struct {
	metrics []models.Metrics
	done    chan error
}

// batchPoster отправляет пакет метрик на сервер (HTTP или gRPC).
type batchPoster interface {
	PostBatch(metrics []models.Metrics) error
	Close() error
}

type httpPoster struct {
	client    *resty.Client
	secretKey string
	baseURL   string
	publicKey *rsa.PublicKey
}

func (p *httpPoster) PostBatch(metrics []models.Metrics) error {
	return agent.PostMetricsBatch(p.client, p.secretKey, p.baseURL, metrics, p.publicKey)
}

func (*httpPoster) Close() error { return nil }

type grpcPoster struct {
	conn    *grpc.ClientConn
	client  pb.MetricsClient
	localIP string
}

func (p *grpcPoster) PostBatch(metrics []models.Metrics) error {
	return agent.PostMetricsBatchGRPC(context.Background(), p.client, p.localIP, metrics)
}

func (p *grpcPoster) Close() error {
	if p.conn == nil {
		return nil
	}
	return p.conn.Close()
}

// metricSender — пул воркеров (worker pool) для исходящих батч-запросов.
type metricSender struct {
	jobs               chan batchJob
	gopsutilGaugeNames []string
	poster             batchPoster
	wg                 sync.WaitGroup
	closeOnce          sync.Once
}

// newMetricSender создаёт пул из workers воркеров и запускает их горутины.
// workers (RATE_LIMIT / -l) — сколько батч-запросов может выполняться одновременно.
func newMetricSender(workers int, poster batchPoster, gopsutilGaugeNames []string) *metricSender {
	if workers < 1 {
		workers = 1
	}

	s := &metricSender{
		jobs:               make(chan batchJob, workers),
		gopsutilGaugeNames: gopsutilGaugeNames,
		poster:             poster,
	}

	for range workers {
		s.wg.Go(func() {
			batchWorker(s.jobs, poster)
		})
	}

	return s
}

// Close закрывает очередь задач, ждёт воркеров и освобождает соединение poster.
func (s *metricSender) Close() {
	s.closeOnce.Do(func() {
		close(s.jobs)
		s.wg.Wait()
		if s.poster != nil {
			_ = s.poster.Close()
		}
	})
}

func batchWorker(jobs <-chan batchJob, poster batchPoster) {
	for job := range jobs {
		job.done <- poster.PostBatch(job.metrics)
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
func (s *metricSender) sendSnapshot(snapshot map[string]float64, delta int64) error {
	metrics := s.buildMetricsFromSnapshot(snapshot, delta)

	done := make(chan error, 1)
	s.jobs <- batchJob{metrics: metrics, done: done}

	return <-done
}
