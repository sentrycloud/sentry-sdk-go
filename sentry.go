package sentry

//	sentry sdk API definition, for sdk user only

type CollectorType int

const (
	Sum CollectorType = iota
	Avg
	Max
	Min
)

type Collector interface {
	Put(value float64)
	PutWithTime(value float64, timestamp int64)
}

// GetCollector used for most scenarios, user just have to get a Collector, then use that to Put metric data
func GetCollector(metric string, tags map[string]string, aggregator CollectorType, interval int64) Collector {
	return getDataCollector(metric, tags, aggregator, interval)
}

// StartCollectGC start collect gc in a separation goroutine
func StartCollectGC(appName string) {
	go startCollectGC(appName)
}

// SetReportURL set reportURL to a server report URL, replace the default local agent URL
func SetReportURL(url string) {
	reportURL = url
}
