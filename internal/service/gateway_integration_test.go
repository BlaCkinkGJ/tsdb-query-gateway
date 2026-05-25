package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/client"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

func staticPromHandler(t *testing.T, resp *models.PromResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}
}

func errorPromHandler(statusCode int, resp *models.PromResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if resp != nil {
			_ = json.NewEncoder(w).Encode(resp)
		}
	}
}

func staticPromServer(t *testing.T, resp *models.PromResponse) *httptest.Server {
	return httptest.NewServer(staticPromHandler(t, resp))
}

func TestGatewayService_Query_MultipleEndpoints_MergeVectors(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{"__name__":"up","instance":"host1"},"value":[1716000000,"1"]}]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{"__name__":"up","instance":"host2"},"value":[1716000000,"1"]}]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))

	res, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "up"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != "success" {
		t.Fatalf("expected status success, got %s", res.Status)
	}
	if res.Data.ResultType != "vector" {
		t.Fatalf("expected vector, got %s", res.Data.ResultType)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(res.Data.Result, &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 merged metrics, got %d", len(parsed))
	}

	instances := make(map[string]struct{})
	for _, m := range parsed {
		metric, ok := m["metric"].(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected metric structure: %v", m)
		}
		instances[metric["instance"].(string)] = struct{}{}
	}
	if _, ok := instances["host1"]; !ok {
		t.Fatalf("expected host1 in merged result")
	}
	if _, ok := instances["host2"]; !ok {
		t.Fatalf("expected host2 in merged result")
	}
}

func TestGatewayService_QueryRange_MultipleEndpoints_MergeMatrix(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "matrix",
			Result: json.RawMessage(`[
				{"metric":{"__name__":"cpu","instance":"host1"},"values":[[1716000000,"0.5"],[1716000060,"0.6"]]}
			]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "matrix",
			Result: json.RawMessage(`[
				{"metric":{"__name__":"cpu","instance":"host2"},"values":[[1716000000,"0.8"],[1716000060,"0.9"]]}
			]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))

	res, err := svc.QueryRange(context.Background(), models.PromQueryRangeRequest{
		Query: "cpu",
		Start: "1716000000",
		End:   "1716000060",
		Step:  "60s",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Data.ResultType != "matrix" {
		t.Fatalf("expected matrix, got %s", res.Data.ResultType)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(res.Data.Result, &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 merged series, got %d", len(parsed))
	}

	instances := make(map[string]struct{})
	for _, s := range parsed {
		metric, ok := s["metric"].(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected metric structure: %v", s)
		}
		instances[metric["instance"].(string)] = struct{}{}
	}
	if _, ok := instances["host1"]; !ok {
		t.Fatalf("expected host1 in merged result")
	}
	if _, ok := instances["host2"]; !ok {
		t.Fatalf("expected host2 in merged result")
	}
}

func TestGatewayService_Query_MultipleQueries(t *testing.T) {
	// Two servers return distinct data based on PromQL query expression.
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var resp models.PromResponse
		switch query {
		case "up":
			resp = models.PromResponse{
				Status: "success",
				Data: models.PromData{
					ResultType: "vector",
					Result:     json.RawMessage(`[{"metric":{"__name__":"up","job":"server1"},"value":[1716000000,"1"]}]`),
				},
			}
		case "process_cpu_seconds_total":
			resp = models.PromResponse{
				Status: "success",
				Data: models.PromData{
					ResultType: "vector",
					Result:     json.RawMessage(`[{"metric":{"__name__":"process_cpu_seconds_total","job":"server1"},"value":[1716000000,"42.5"]}]`),
				},
			}
		default:
			resp = models.PromResponse{Status: "success", Data: models.PromData{ResultType: "vector", Result: json.RawMessage(`[]`)}}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer s1.Close()

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var resp models.PromResponse
		switch query {
		case "up":
			resp = models.PromResponse{
				Status: "success",
				Data: models.PromData{
					ResultType: "vector",
					Result:     json.RawMessage(`[{"metric":{"__name__":"up","job":"server2"},"value":[1716000000,"1"]}]`),
				},
			}
		case "process_cpu_seconds_total":
			resp = models.PromResponse{
				Status: "success",
				Data: models.PromData{
					ResultType: "vector",
					Result:     json.RawMessage(`[{"metric":{"__name__":"process_cpu_seconds_total","job":"server2"},"value":[1716000000,"99.1"]}]`),
				},
			}
		default:
			resp = models.PromResponse{Status: "success", Data: models.PromData{ResultType: "vector", Result: json.RawMessage(`[]`)}}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	ctx := context.Background()

	// Test query "up"
	resUp, err := svc.Query(ctx, models.PromQueryRequest{Query: "up"})
	if err != nil {
		t.Fatalf("query 'up' failed: %v", err)
	}
	var upResult []map[string]interface{}
	if err := json.Unmarshal(resUp.Data.Result, &upResult); err != nil {
		t.Fatalf("unmarshal 'up' result: %v", err)
	}
	if len(upResult) != 2 {
		t.Fatalf("expected 2 results for 'up', got %d", len(upResult))
	}

	jobs := make(map[string]struct{})
	for _, m := range upResult {
		metric := m["metric"].(map[string]interface{})
		jobs[metric["job"].(string)] = struct{}{}
	}
	if _, ok := jobs["server1"]; !ok {
		t.Fatalf("expected server1 in 'up' result")
	}
	if _, ok := jobs["server2"]; !ok {
		t.Fatalf("expected server2 in 'up' result")
	}

	// Test query "process_cpu_seconds_total"
	resCPU, err := svc.Query(ctx, models.PromQueryRequest{Query: "process_cpu_seconds_total"})
	if err != nil {
		t.Fatalf("query 'process_cpu_seconds_total' failed: %v", err)
	}
	var cpuResult []map[string]interface{}
	if err := json.Unmarshal(resCPU.Data.Result, &cpuResult); err != nil {
		t.Fatalf("unmarshal 'process_cpu_seconds_total' result: %v", err)
	}
	if len(cpuResult) != 2 {
		t.Fatalf("expected 2 results for 'process_cpu_seconds_total', got %d", len(cpuResult))
	}
}

func TestGatewayService_Query_MultipleEndpoints_ScalarFirstWins(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "scalar",
			Result:     json.RawMessage(`[1716000000, "42"]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "scalar",
			Result:     json.RawMessage(`[1716000000, "99"]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	res, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "scalar(count(up))"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Data.ResultType != "scalar" {
		t.Fatalf("expected scalar, got %s", res.Data.ResultType)
	}
	if string(res.Data.Result) != `[1716000000,"42"]` {
		t.Errorf("unexpected scalar result: %s", string(res.Data.Result))
	}
}

func TestGatewayService_Query_MultipleEndpoints_StringFirstWins(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "string",
			Result:     json.RawMessage(`[1716000000, "hello"]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "string",
			Result:     json.RawMessage(`[1716000000, "world"]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	res, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "label_replace(up, \"x\", \"hello\", \"\", \"\")"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Data.ResultType != "string" {
		t.Fatalf("expected string, got %s", res.Data.ResultType)
	}
	if string(res.Data.Result) != `[1716000000,"hello"]` {
		t.Errorf("unexpected string result: %s", string(res.Data.Result))
	}
}

func TestGatewayService_Query_MultipleEndpoints_WarningsDedup(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status:   "success",
		Warnings: []string{"partial timeout"},
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{}}]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status:   "success",
		Warnings: []string{"partial timeout", "data gap"},
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{}}]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	res, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "up"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("expected 2 deduplicated warnings, got %d: %v", len(res.Warnings), res.Warnings)
	}

	seen := make(map[string]struct{})
	for _, w := range res.Warnings {
		seen[w] = struct{}{}
	}
	if _, ok := seen["partial timeout"]; !ok {
		t.Fatalf("expected 'partial timeout' warning")
	}
	if _, ok := seen["data gap"]; !ok {
		t.Fatalf("expected 'data gap' warning")
	}
}

func TestGatewayService_Query_MultipleEndpoints_PartialFailure(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{}}]`),
		},
	})
	defer s1.Close()

	s2 := httptest.NewServer(errorPromHandler(http.StatusInternalServerError, &models.PromResponse{
		Status:    "error",
		ErrorType: "internal",
		Error:     "downstream crash",
	}))
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	_, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "up"})
	if err == nil {
		t.Fatal("expected error for partial downstream failure, got nil")
	}
	if !strings.Contains(err.Error(), "downstream crash") {
		t.Fatalf("expected downstream error in message, got: %v", err)
	}
}

func TestGatewayService_Query_MultipleEndpoints_AllFail(t *testing.T) {
	s1 := httptest.NewServer(errorPromHandler(http.StatusServiceUnavailable, &models.PromResponse{
		Status:    "error",
		ErrorType: "timeout",
		Error:     "query timed out",
	}))
	defer s1.Close()

	s2 := httptest.NewServer(errorPromHandler(http.StatusBadRequest, &models.PromResponse{
		Status:    "error",
		ErrorType: "bad_data",
		Error:     "invalid query",
	}))
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	_, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "up"})
	if err == nil {
		t.Fatal("expected error when all downstreams fail, got nil")
	}
}

func TestGatewayService_Query_MultipleEndpoints_InconsistentTypes(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "matrix",
			Result:     json.RawMessage(`[]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	_, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "up"})
	if err == nil {
		t.Fatal("expected error for inconsistent result types, got nil")
	}
	if !strings.Contains(err.Error(), "inconsistent result types") {
		t.Fatalf("expected inconsistent result types error, got: %v", err)
	}
}

func TestGatewayService_Query_MultipleEndpoints_MixedEmptyAndNonEmpty(t *testing.T) {
	s1 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[]`),
		},
	})
	defer s1.Close()

	s2 := staticPromServer(t, &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{"__name__":"up"},"value":[1716000000,"1"]}]`),
		},
	})
	defer s2.Close()

	svc := NewGatewayService([]string{s1.URL, s2.URL}, client.NewPrometheusClient(5*time.Second))
	res, err := svc.Query(context.Background(), models.PromQueryRequest{Query: "up"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(res.Data.Result, &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 merged metric, got %d", len(parsed))
	}
}
