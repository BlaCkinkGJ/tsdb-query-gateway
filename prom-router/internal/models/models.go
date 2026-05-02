package models

import "encoding/json"

// PromQueryRequest represents a basic Prometheus query.
type PromQueryRequest struct {
	Query string `json:"query"`
	Time  string `json:"time,omitempty"`
	Timeout string `json:"timeout,omitempty"`
}

// PromQueryRangeRequest represents a basic Prometheus query_range.
type PromQueryRangeRequest struct {
	Query string `json:"query"`
	Start string `json:"start"`
	End   string `json:"end"`
	Step  string `json:"step"`
	Timeout string `json:"timeout,omitempty"`
}

// PromResponse represents a Prometheus API response.
type PromResponse struct {
	Status    string          `json:"status"`
	Data      PromData        `json:"data,omitempty"`
	ErrorType string          `json:"errorType,omitempty"`
	Error     string          `json:"error,omitempty"`
	Warnings  []string        `json:"warnings,omitempty"`
}

// PromData represents the data section of a Prometheus response.
type PromData struct {
	ResultType string            `json:"resultType"`
	Result     json.RawMessage   `json:"result"`
}
