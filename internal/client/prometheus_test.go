package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

func TestPrometheusClient_non2xxStructuredError(t *testing.T) {
	// Simulate a Prometheus API that returns a structured error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.PromResponse{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "invalid parameter \"query\": 1:1: parse error: no expression found in input",
		})
	}))
	defer server.Close()

	client := NewPrometheusClient(5 * time.Second)
	_, err := client.Query(context.Background(), server.URL, models.PromQueryRequest{Query: "bad"})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}

	// The error should contain the structured Prometheus error fields, not raw body
	if !strings.Contains(err.Error(), "bad_data") {
		t.Errorf("expected error to contain Prometheus errorType, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid parameter") {
		t.Errorf("expected error to contain Prometheus error message, got: %v", err)
	}
}

func TestPrometheusClient_non2xxRawBody(t *testing.T) {
	// Simulate a plain text error response (non-JSON)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewPrometheusClient(5 * time.Second)
	_, err := client.Query(context.Background(), server.URL, models.PromQueryRequest{Query: "up"})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}

	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("expected raw body in error, got: %v", err)
	}
}
