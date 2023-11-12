package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var reportURL = "http://127.0.0.1:50001/agent/api/putMetrics"

type AgentDataPoint struct {
	Metric    string            `json:"metric"`
	Tags      map[string]string `json:"tags"`
	Timestamp int64             `json:"timestamp"`
	Value     float64           `json:"value"`
}

func Send(dps []AgentDataPoint) error {
	data, err := json.Marshal(dps)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	body := bytes.NewReader(data)
	req, err := http.NewRequest("POST", reportURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status error: %d", resp.StatusCode)
	}

	return nil
}

// SetReportURL set reportURL to a server report URL, replace the default local agent URL
func SetReportURL(url string) {
	reportURL = url
}
