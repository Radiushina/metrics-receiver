package models

import (
	"math/rand"
	"runtime"
)

var GaugeNames = []string{
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

func UpdateGaugesFromMemStats(gaugeValues map[string]float64, ms *runtime.MemStats, rnd *rand.Rand) {
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

	pick := GaugeNames[rnd.Intn(len(GaugeNames))]
	gaugeValues["RandomValue"] = gaugeValues[pick]
}
