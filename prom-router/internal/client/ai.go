package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/example/prom-router/internal/models"
)

type AIClient interface {
	Process(ctx context.Context, endpoint string, data *models.PromResponse) (*models.PromResponse, error)
}

type aiClientImpl struct {
	httpClient *http.Client
}

func NewAIClient() AIClient {
	return &aiClientImpl{
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // AI operations might take longer
		},
	}
}

func (c *aiClientImpl) Process(ctx context.Context, endpoint string, data *models.PromResponse) (*models.PromResponse, error) {
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI endpoint error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var modifiedResp models.PromResponse
	if err := json.Unmarshal(body, &modifiedResp); err != nil {
		return nil, err
	}

	return &modifiedResp, nil
}
