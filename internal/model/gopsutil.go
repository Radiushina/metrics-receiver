package models

import "fmt"

const (
	// TotalMemory — общий объём оперативной памяти (байты), gauge из gopsutil.
	TotalMemory = "TotalMemory"
	// FreeMemory — свободная оперативная память (байты), gauge из gopsutil.
	FreeMemory = "FreeMemory"
)

// CPUUtilizationMetricName возвращает имя gauge-метрики загрузки CPU по индексу ядра.
// Индексация с нуля: CPUutilization0, CPUutilization1, …
func CPUUtilizationMetricName(cpuIndex int) string {
	return fmt.Sprintf("CPUutilization%d", cpuIndex)
}

// GopsutilGaugeNames возвращает имена gauge-метрик gopsutil для заданного числа CPU.
func GopsutilGaugeNames(cpuCount int) []string {
	names := make([]string, 0, 2+cpuCount)
	names = append(names, TotalMemory, FreeMemory)
	for i := range cpuCount {
		names = append(names, CPUUtilizationMetricName(i))
	}
	return names
}
