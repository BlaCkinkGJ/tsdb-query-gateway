package service

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/client"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

type GatewayService struct {
	promEndpoints []string
	promClient    client.PrometheusClient
}

func NewGatewayService(promEndpoints []string, promClient client.PrometheusClient) *GatewayService {
	endpoints := make([]string, len(promEndpoints))
	copy(endpoints, promEndpoints)
	return &GatewayService{
		promEndpoints: endpoints,
		promClient:    promClient,
	}
}

// Query implements the QueryService interface.
func (s *GatewayService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
	if len(s.promEndpoints) == 0 {
		return nil, fmt.Errorf("no prometheus endpoints configured")
	}

	results := make([]*models.PromResponse, len(s.promEndpoints))
	g, gCtx := errgroup.WithContext(ctx)

	for i, endpoint := range s.promEndpoints {
		idx, targetURL := i, endpoint // capture variables for closure
		g.Go(func() error {
			res, err := s.promClient.Query(gCtx, targetURL, req)
			if err != nil {
				return err
			}
			results[idx] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error querying downstream prometheus: %w", err)
	}

	return s.mergeResults(results)
}

// QueryRange implements the QueryService interface.
func (s *GatewayService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
	if len(s.promEndpoints) == 0 {
		return nil, fmt.Errorf("no prometheus endpoints configured")
	}

	results := make([]*models.PromResponse, len(s.promEndpoints))
	g, gCtx := errgroup.WithContext(ctx)

	for i, endpoint := range s.promEndpoints {
		idx, targetURL := i, endpoint // capture variables for closure
		g.Go(func() error {
			res, err := s.promClient.QueryRange(gCtx, targetURL, req)
			if err != nil {
				return err
			}
			results[idx] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error querying downstream prometheus: %w", err)
	}

	return s.mergeResults(results)
}

// mergeResults merges the data from multiple Prometheus responses.
func (s *GatewayService) mergeResults(results []*models.PromResponse) (*models.PromResponse, error) {
	if len(results) == 0 {
		return nil, nil
	}

	var mergedResultType string
	allMetrics := []json.RawMessage{}
	var allWarnings []string

	for _, res := range results {
		if res == nil || res.Status != "success" || len(res.Data.Result) == 0 {
			continue
		}

		// Collect warnings from all downstream responses
		allWarnings = append(allWarnings, res.Warnings...)

		if mergedResultType == "" {
			mergedResultType = res.Data.ResultType
		} else if mergedResultType != res.Data.ResultType {
			return nil, fmt.Errorf("inconsistent result types in downstream responses: expected %q, got %q", mergedResultType, res.Data.ResultType)
		}

		// Scalar and string result types are [timestamp, "value"] tuples,
		// not a list of metric objects. Merging across instances is not
		// meaningful for these types — return the first valid result as-is.
		if mergedResultType == "scalar" || mergedResultType == "string" {
			res.Warnings = allWarnings
			return res, nil
		}

		// For vector, matrix, and unknown types, assume a concatenatable list of metrics.
		var metrics []json.RawMessage
		if err := json.Unmarshal(res.Data.Result, &metrics); err != nil {
			// If the payload isn't a list (e.g., a malformed scalar/string), fail the merge.
			return nil, fmt.Errorf("failed to unmarshal downstream result payload: %w", err)
		}
		allMetrics = append(allMetrics, metrics...)
	}

	if mergedResultType == "" {
		// Fallback to the first non-nil successful result if any
		for _, r := range results {
			if r != nil && r.Status == "success" {
				return r, nil
			}
		}
		return nil, fmt.Errorf("all downstream queries failed to produce valid result data")
	}

	mergedRaw, err := json.Marshal(allMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged results: %w", err)
	}

	resp := &models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: mergedResultType,
			Result:     mergedRaw,
		},
	}
	if len(allWarnings) > 0 {
		resp.Warnings = allWarnings
	}
	return resp, nil
}
