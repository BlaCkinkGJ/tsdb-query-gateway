package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/models"
)

type mockPromClient struct {
	result []interface{}
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

func TestRouterService_Query(t *testing.T) {
	mockClient := &mockPromClient{
		result: []interface{}{
			map[string]interface{}{"metric": map[string]string{"__name__": "up"}},
		},
	}

	endpoints := []string{"http://prom1", "http://prom2"}
	svc := NewRouterService(endpoints, mockClient)

	req := models.PromQueryRequest{Query: "up"}
	res, err := svc.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if res.Status != "success" {
		t.Errorf("expected success status, got %s", res.Status)
	}

	var parsedResult []interface{}
	err = json.Unmarshal(res.Data.Result, &parsedResult)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// We have 2 endpoints, each returning 1 metric, so we expect 2 metrics in the merged result
	if len(parsedResult) != 2 {
		t.Errorf("expected 2 merged metrics, got %d", len(parsedResult))
	}
}
