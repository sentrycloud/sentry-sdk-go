package sentry

import (
	"sort"
	"strings"
	"time"
)

type CollectorType int

const (
	Sum CollectorType = iota
	Avg
	Max
	Min
)

type Curve struct {
	metric     string
	tags       map[string]string
	aggregator CollectorType
	interval   int64
	uniqueId   string // this is calculated from metric and tags
}

type DataPoint struct {
	Curve
	timestamp int64
	value     float64
}

type Collector struct {
	Curve
}

func (b *Collector) Put(value float64) {
	b.PutWithTime(value, time.Now().Unix())
}

func (b *Collector) PutWithTime(value float64, timestamp int64) {
	var dp = DataPoint{}
	dp.Curve = b.Curve
	dp.timestamp = timestamp
	dp.value = value

	// send in non-blocking mode, discard data point when channel is full
	select {
	case dataChan <- dp:
	default:
	}
}

func GetCollector(metric string, tags map[string]string, aggregator CollectorType, interval int64) *Collector {
	collector := &Collector{}
	collector.metric = metric
	collector.tags = make(map[string]string)
	for k, v := range tags {
		collector.tags[k] = v
	}
	collector.aggregator = aggregator
	collector.interval = interval
	collector.uniqueId = serializeUniqueId(metric, tags)
	return collector
}

func serializeUniqueId(metric string, tags map[string]string) string {
	// cause go map is random, we need to sort keys to make sure the same tags serialize to the same id
	var keys []string
	for k := range tags {
		keys = append(keys, k)
	}

	var id strings.Builder
	id.WriteString(metric)
	id.WriteString("@")

	sort.Strings(keys)
	for _, k := range keys {
		id.WriteString(k)
		id.WriteString(":")
		id.WriteString(tags[k])
		id.WriteString(",")
	}
	return id.String()
}
