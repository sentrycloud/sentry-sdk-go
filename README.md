# sentry-sdk-go
sentry sdk for sending metrics to sentry_agent or sentry_server in golang

# Install 
```
https://github.com/sentrycloud/sentry-sdk-go
```

# Usage

```
package main

import "github.com/sentrycloud/sentry-sdk-go"

func main() {
	metric := "testApp_http_qps"
	tags := map[string]string {
		"from":  "iOS",
		"aggregator": "sum",
	}

	qpsCollector := sentry.GetCollector(metric, tags, sentry.Sum, 10)
	qpsCollector.Put(1)
}
```
