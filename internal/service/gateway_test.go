package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

type mockPromClient struct {
	result []json.RawMessage
}

func (m *mockPromClient) Query(ctx context.Context, targetURL string, req models.PromQueryRequest) (*models.PromResponse, error) {
	raw, _ := json.Marshal(m.result)
	return &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     raw,
		},
	}, nil
}

func (m *mockPromClient) QueryRange(ctx context.Context, targetURL string, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	return nil, nil // mock stub
}

func TestGatewayService_Query(t *testing.T) {
	mockClient := &mockPromClient{
		result: []json.RawMessage{
			json.RawMessage(`{"metric": {"__name__": "up"}}`),
		},
	}

	endpoints := []string{"http://prom1", "http://prom2"}
	svc := NewGatewayService(endpoints, mockClient)

	req := models.PromQueryRequest{Query: "up"}
	res, err := svc.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if res.Status != "success" {
		t.Errorf("expected success status, got %s", res.Status)
	}

	var parsedResult []json.RawMessage
	err = json.Unmarshal(res.Data.Result, &parsedResult)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// We have 2 endpoints, each returning 1 metric, so we expect 2 metrics in the merged result
	if len(parsedResult) != 2 {
		t.Errorf("expected 2 merged metrics, got %d", len(parsedResult))
	}
}

func TestGatewayService_mergeResults_scalar(t *testing.T) {
	svc := NewGatewayService(nil, nil)
	results := []*models.PromResponse{
		{
			Status: "success",
			Data: models.PromData{
				ResultType: "scalar",
				Result:     json.RawMessage(`[1234567890, "42"]`),
			},
		},
	}
	res, err := svc.mergeResults(results)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Data.ResultType != "scalar" {
		t.Errorf("expected scalar, got %s", res.Data.ResultType)
	}
	if string(res.Data.Result) != `[1234567890, "42"]` {
		t.Errorf("unexpected scalar result: %s", string(res.Data.Result))
	}
}

func TestGatewayService_mergeResults_string(t *testing.T) {
	svc := NewGatewayService(nil, nil)
	results := []*models.PromResponse{
		{
			Status: "success",
			Data: models.PromData{
				ResultType: "string",
				Result:     json.RawMessage(`[1234567890, "hello"]`),
			},
		},
	}
	res, err := svc.mergeResults(results)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Data.ResultType != "string" {
		t.Errorf("expected string, got %s", res.Data.ResultType)
	}
}

func TestGatewayService_mergeResults_emptyResult(t *testing.T) {
	svc := NewGatewayService(nil, nil)
	results := []*models.PromResponse{
		{Status: "success", Data: models.PromData{ResultType: "vector", Result: json.RawMessage(`[]`)}},
		{Status: "success", Data: models.PromData{ResultType: "vector", Result: json.RawMessage(`[]`)}},
	}
	res, err := svc.mergeResults(results)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	var parsed []json.RawMessage
	if err := json.Unmarshal(res.Data.Result, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("expected empty array, got %d elements", len(parsed))
	}
	if string(res.Data.Result) != "[]" {
		t.Errorf("expected JSON [], got %s", string(res.Data.Result))
	}
}

func TestGatewayService_mergeResults_warnings(t *testing.T) {
	svc := NewGatewayService(nil, nil)
	results := []*models.PromResponse{
		{
			Status:   "success",
			Warnings: []string{"warning1"},
			Data:     models.PromData{ResultType: "vector", Result: json.RawMessage(`[{"metric":{}}]`)},
		},
		{
			Status:   "success",
			Warnings: []string{"warning2", "warning3"},
			Data:     models.PromData{ResultType: "vector", Result: json.RawMessage(`[{"metric":{}}]`)},
		},
	}
	res, err := svc.mergeResults(results)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(res.Warnings) != 3 {
		t.Errorf("expected 3 warnings, got %d", len(res.Warnings))
	}
}

func TestGatewayService_mergeResults_inconsistentTypes(t *testing.T) {
	svc := NewGatewayService(nil, nil)
	results := []*models.PromResponse{
		{Status: "success", Data: models.PromData{ResultType: "vector", Result: json.RawMessage(`[]`)}},
		{Status: "success", Data: models.PromData{ResultType: "matrix", Result: json.RawMessage(`[]`)}},
	}
	_, err := svc.mergeResults(results)
	if err == nil {
		t.Fatal("expected error for inconsistent result types, got nil")
	}
}

func TestGatewayService_mergeResults_unknownType(t *testing.T) {
	svc := NewGatewayService(nil, nil)
	results := []*models.PromResponse{
		{Status: "success", Data: models.PromData{ResultType: "custom", Result: json.RawMessage(`[]`)}},
	}
	_, err := svc.mergeResults(results)
	if err == nil {
		t.Fatal("expected error for unknown result type, got nil")
	}
}

func TestGatewayService_defensiveCopy(t *testing.T) {
	endpoints := []string{"http://prom1", "http://prom2"}
	svc := NewGatewayService(endpoints, &mockPromClient{})

	// Mutate the original slice
	endpoints[0] = "http://tampered"

	// The service should still have the original value
	if svc.promEndpoints[0] != "http://prom1" {
		t.Errorf("expected defensive copy, got %s", svc.promEndpoints[0])
	}
}
