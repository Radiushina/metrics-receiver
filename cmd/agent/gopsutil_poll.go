package main

import (
	"context"
	"sync"
	"time"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
)

// initGopsutilGauges подготавливает ключи gauge-метрик gopsutil в gaugeValues.
func initGopsutilGauges(gaugeValues map[string]float64, gopsutilNames []string) {
	for _, name := range gopsutilNames {
		gaugeValues[name] = 0
	}
}

// pollGopsutilOnce читает память и загрузку CPU через gopsutil и обновляет gaugeValues.
func pollGopsutilOnce(
	logg *zap.Logger,
	mu *sync.Mutex,
	gaugeValues map[string]float64,
	cpuCount int,
) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		logg.Sugar().Warnf("gopsutil poll: virtual memory: %v", err)
		return
	}

	percents, err := cpu.Percent(0, true)
	if err != nil {
		logg.Sugar().Warnf("gopsutil poll: cpu percent: %v", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	gaugeValues[models.TotalMemory] = float64(vm.Total)
	gaugeValues[models.FreeMemory] = float64(vm.Free)

	for i := 0; i < cpuCount && i < len(percents); i++ {
		gaugeValues[models.CPUUtilizationMetricName(i)] = percents[i]
	}

	logg.Info("poll: gopsutil TotalMemory, FreeMemory, CPUutilization updated")
}

// runGopsutilPollLoop — третья горутина агента: периодически собирает метрики через gopsutil.
// TotalMemory, FreeMemory и CPUutilization0…CPUutilization{N-1} (N = число логических CPU).
// Останавливается при отмене ctx.
func runGopsutilPollLoop(
	ctx context.Context,
	logg *zap.Logger,
	interval time.Duration,
	mu *sync.Mutex,
	gaugeValues map[string]float64,
	cpuCount int,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pollGopsutilOnce(logg, mu, gaugeValues, cpuCount)
		}
	}
}
