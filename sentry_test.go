package sentry

import (
	"sync"
	"testing"
	"time"
)

const TestCount = 21

func aggregatorToString(aggregator CollectorType) string {
	switch aggregator {
	case Sum:
		return "sum"
	case Avg:
		return "avg"
	case Max:
		return "max"
	case Min:
		return "min"
	default:
		return ""
	}
}

func testCollector(aggregator CollectorType, value float64) {
	metric := "testApp_http_qps"
	tags := map[string]string{
		"from":       "iOS",
		"aggregator": aggregatorToString(aggregator),
	}

	collector := GetCollector(metric, tags, aggregator, 10)
	collector.Put(value)
}

func TestAllCollectors(t *testing.T) {
	for i := 0; i < TestCount; i++ {
		testCollector(Sum, float64(i))
		testCollector(Avg, float64(i))
		testCollector(Max, float64(i))
		testCollector(Min, float64(i))
		time.Sleep(1 * time.Second)
	}
}

func testInGoroutine(wg *sync.WaitGroup, aggregator CollectorType) {
	defer wg.Done()
	for i := 0; i < TestCount; i++ {
		testCollector(aggregator, float64(i))
		time.Sleep(1 * time.Second)
	}
}

func TestAllCollectorsInGoroutine(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(4)
	go testInGoroutine(&wg, Sum)
	go testInGoroutine(&wg, Avg)
	go testInGoroutine(&wg, Max)
	go testInGoroutine(&wg, Min)
	wg.Wait()
}
