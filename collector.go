package sentry

import (
	"log"
	"sort"
	"strings"
	"time"
)

const (
	DataChanSize        = 10240
	MaxAccumulatorCount = 32
	DpsBatchSize        = 20
)

type accumulator struct {
	value float64
	count int
}

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

type DataCollector struct {
	Curve
}

func (b *DataCollector) Put(value float64) {
	b.PutWithTime(value, time.Now().Unix())
}

func (b *DataCollector) PutWithTime(value float64, timestamp int64) {
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

func getDataCollector(metric string, tags map[string]string, aggregator CollectorType, interval int64) Collector {
	collector := &DataCollector{}
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

type BaseCollector interface {
	Put(value float64, ts int64)
	Aggregate(now int64) []AgentDataPoint
	Type() CollectorType
}

type MinMaxCollector struct {
	Curve
	values            [MaxAccumulatorCount]accumulator
	lastAggregateTime int64
}

func (m *MinMaxCollector) Type() CollectorType {
	return m.Curve.aggregator
}

func (m *MinMaxCollector) Put(value float64, ts int64) {
	index := ts / m.interval % MaxAccumulatorCount // calculate which slot to put the value

	if m.values[index].count == 0 {
		m.values[index].value = value
		m.values[index].count = 1
		return
	}

	if m.aggregator == Min {
		if value < m.values[index].value {
			m.values[index].value = value
		}
	} else {
		if value > m.values[index].value {
			m.values[index].value = value
		}
	}
	m.values[index].count++
}

func (m *MinMaxCollector) Aggregate(now int64) []AgentDataPoint {
	var dps []AgentDataPoint
	for m.lastAggregateTime+m.interval < now {
		index := m.lastAggregateTime / m.interval % MaxAccumulatorCount
		if m.values[index].count > 0 {
			dp := AgentDataPoint{
				Metric:    m.metric,
				Tags:      m.tags,
				Timestamp: m.lastAggregateTime,
				Value:     m.values[index].value,
			}

			dps = append(dps, dp)
		}

		m.lastAggregateTime += m.interval
		m.values[index].value = 0
		m.values[index].count = 0
	}

	return dps
}

type SumCollector struct {
	Curve
	values            [MaxAccumulatorCount]accumulator
	lastAggregateTime int64
}

func (s *SumCollector) Type() CollectorType {
	return s.Curve.aggregator
}

func (s *SumCollector) Put(value float64, ts int64) {
	index := ts / s.interval % MaxAccumulatorCount

	s.values[index].value += value
	s.values[index].count++
}

func (s *SumCollector) Aggregate(now int64) []AgentDataPoint {
	var dps []AgentDataPoint
	for s.lastAggregateTime+s.interval < now {
		index := s.lastAggregateTime / s.interval % MaxAccumulatorCount
		if s.values[index].count > 0 {
			value := s.values[index].value
			if s.aggregator == Avg {
				value = s.values[index].value / float64(s.values[index].count)
			}

			dp := AgentDataPoint{
				Metric:    s.metric,
				Tags:      s.tags,
				Timestamp: s.lastAggregateTime,
				Value:     value,
			}

			dps = append(dps, dp)
		}

		s.lastAggregateTime += s.interval
		s.values[index].value = 0
		s.values[index].count = 0
	}

	return dps
}

var dataChan chan DataPoint
var collectorMap = map[string]BaseCollector{}

func collect() {
	t := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-t.C:
			aggregate()
		case dp := <-dataChan:
			handleDataPoint(&dp)
		}
	}
}

func handleDataPoint(dp *DataPoint) {
	baseCollector, exist := collectorMap[dp.uniqueId]
	if exist {
		if baseCollector.Type() != dp.aggregator {
			log.Printf("discard the data, cause collector type is not match, %d != %d", baseCollector.Type(), dp.aggregator)
			return
		}

		baseCollector.Put(dp.value, dp.timestamp)
		return
	}

	lastAggregateTime := dp.timestamp - dp.timestamp%dp.interval
	switch dp.aggregator {
	case Sum, Avg:
		var collector = SumCollector{
			Curve:             dp.Curve,
			lastAggregateTime: lastAggregateTime,
		}
		collectorMap[dp.uniqueId] = BaseCollector(&collector)
		collector.Put(dp.value, dp.timestamp)
	case Min, Max:
		var collector = MinMaxCollector{
			Curve:             dp.Curve,
			lastAggregateTime: lastAggregateTime,
		}
		collectorMap[dp.uniqueId] = BaseCollector(&collector)
		collector.Put(dp.value, dp.timestamp)
	}
}

func aggregate() {
	now := time.Now().Unix()

	var agentDps []AgentDataPoint
	for _, collector := range collectorMap {
		dps := collector.Aggregate(now)
		if len(dps) > 0 {
			agentDps = append(agentDps, dps...)
			if len(agentDps) > DpsBatchSize {
				go Send(agentDps)
				agentDps = nil // clear
			}
		}
	}

	if len(agentDps) > 0 {
		go Send(agentDps)
	}
}
