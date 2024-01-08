package sentry

func init() {
	dataChan = make(chan DataPoint, DataChanSize)

	go collect()
}
