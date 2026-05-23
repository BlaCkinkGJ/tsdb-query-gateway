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
	seenWarnings := make(map[string]struct{})
	var allWarnings []string
	var firstScalarString *models.PromResponse

	addWarning := func(w string) {
		if _, ok := seenWarnings[w]; !ok {
			seenWarnings[w] = struct{}{}
			allWarnings = append(allWarnings, w)
		}
	}

	for _, res := range results {
		if res == nil || res.Status != "success" {
			continue
		}

		// Collect warnings from ALL successful downstream responses,
		// even those with empty result data.
		for _, w := range res.Warnings {
			addWarning(w)
		}

		if len(res.Data.Result) == 0 {
			continue
		}

		if mergedResultType == "" {
			mergedResultType = res.Data.ResultType
		} else if mergedResultType != res.Data.ResultType {
			return nil, fmt.Errorf("inconsistent result types in downstream responses: expected %q, got %q", mergedResultType, res.Data.ResultType)
		}

		// Scalar and string result types are [timestamp, "value"] tuples,
		// not a list of metric objects. Merging across instances is not
		// meaningful for these types — remember the first valid result but
		// continue the loop to gather warnings and validate consistency.
		if mergedResultType == "scalar" || mergedResultType == "string" {
			if firstScalarString == nil {
				firstScalarString = res
			}
			continue
		}

		// Only vector and matrix types are concatenatable lists of metrics.
		if mergedResultType != "vector" && mergedResultType != "matrix" {
			return nil, fmt.Errorf("unsupported result type for merging: %s", mergedResultType)
		}
		var metrics []json.RawMessage
		if err := json.Unmarshal(res.Data.Result, &metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal downstream result payload: %w", err)
		}
		allMetrics = append(allMetrics, metrics...)
	}

	if mergedResultType == "" {
		// Fallback to the first non-nil successful result if any
		for _, r := range results {
			if r != nil && r.Status == "success" {
				r.Warnings = allWarnings
				return r, nil
			}
		}
		return nil, fmt.Errorf("all downstream queries failed to produce valid result data")
	}

	// For scalar/string, return the first valid result with all collected warnings.
	if mergedResultType == "scalar" || mergedResultType == "string" {
		if firstScalarString != nil {
			firstScalarString.Warnings = allWarnings
			return firstScalarString, nil
		}
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
