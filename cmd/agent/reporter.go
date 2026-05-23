package main

import (
	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

// metricJob — задача на отправку одной метрики воркеру пула.
type metricJob struct {
	name         string
	metricType   models.MetricType
	gaugeValue   float64
	counterDelta int64
	done         chan error // сюда воркер передаёт результат HTTP-запроса
}

// metricSender — пул воркеров с общей очередью задач (паттерн worker pool).
type metricSender struct {
	jobs               chan metricJob
	gopsutilGaugeNames []string
}

// newMetricSender создаёт пул из workers воркеров и запускает их горутины.
// workers — верхняя граница числа одновременных исходящих HTTP-запросов (RATE_LIMIT / -l).
// gopsutilGaugeNames — имена gauge-метрик из gopsutil (TotalMemory, FreeMemory, CPUutilization*).
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
		jobs:               make(chan metricJob),
		gopsutilGaugeNames: gopsutilGaugeNames,
	}

	for range workers {
		go metricWorker(s.jobs, client, secretKey, baseURL)
	}

	return s
}

// metricWorker читает задачи из jobs и отправляет метрики на сервер.
// Завершается при закрытии канала jobs (в текущей реализации канал не закрывается — воркеры живут всё время работы агента).
func metricWorker(
	jobs <-chan metricJob,
	client *resty.Client,
	secretKey, baseURL string,
) {
	for job := range jobs {
		var err error
		switch job.metricType {
		case models.Gauge:
			err = agent.PostGaugeMetric(client, secretKey, baseURL, job.name, models.Gauge, job.gaugeValue)
		case models.Counter:
			err = agent.PostCounterMetric(client, secretKey, baseURL, job.name, models.Counter, job.counterDelta)
		}
		job.done <- err
	}
}

// sendSnapshot ставит в очередь все метрики из снимка и ждёт завершения отправки.
// snapshot — копия gauge-значений; delta — приращение PollCount за прошедший интервал опроса.
// Возвращает первую встреченную ошибку или nil, если все запросы успешны.
func (s *metricSender) sendSnapshot(snapshot map[string]float64, delta int64) error {
	tasks := make([]metricJob, 0, len(models.GaugeNames)+len(s.gopsutilGaugeNames)+2)

	for _, name := range models.GaugeNames {
		v := snapshot[name]
		tasks = append(tasks, metricJob{
			name:       name,
			metricType: models.Gauge,
			gaugeValue: v,
		})
	}

	for _, name := range s.gopsutilGaugeNames {
		v := snapshot[name]
		tasks = append(tasks, metricJob{
			name:       name,
			metricType: models.Gauge,
			gaugeValue: v,
		})
	}

	rv := snapshot["RandomValue"]
	tasks = append(tasks, metricJob{
		name:       "RandomValue",
		metricType: models.Gauge,
		gaugeValue: rv,
	})

	tasks = append(tasks, metricJob{
		name:         models.PollCount,
		metricType:   models.Counter,
		counterDelta: delta,
	})

	var firstErr error
	for i := range tasks {
		tasks[i].done = make(chan error, 1)
		s.jobs <- tasks[i]
	}

	for i := range tasks {
		if err := <-tasks[i].done; err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
