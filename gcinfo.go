package sentry

import (
	"runtime"
	"runtime/debug"
	"time"
)

const GcInterval = 10

var (
	goNumCollector   Collector
	gcNumCollector   Collector
	gcPauseCollector Collector

	lastGCNum   int64         = 0
	lastGCPause time.Duration = 0
)

func startCollectGC(appName string) {
	tags := map[string]string{"appName": appName}
	goNumCollector = GetCollector("sentry_go_num", tags, Sum, GcInterval)
	gcNumCollector = GetCollector("sentry_gc_num", tags, Sum, GcInterval)
	gcPauseCollector = GetCollector("sentry_gc_pause", tags, Sum, GcInterval)

	t := time.NewTicker(GcInterval * time.Second)
	for {
		select {
		case <-t.C:
			collectGC()
		}
	}
}

func collectGC() {
	goNum := runtime.NumGoroutine()
	goNumCollector.Put(float64(goNum))

	stats := &debug.GCStats{}
	debug.ReadGCStats(stats)

	gcNum := stats.NumGC - lastGCNum
	pauseTime := stats.PauseTotal - lastGCPause

	gcNumCollector.Put(float64(gcNum))
	gcPauseCollector.Put(float64(pauseTime.Milliseconds()))

	lastGCNum = stats.NumGC
	lastGCPause = stats.PauseTotal
}
