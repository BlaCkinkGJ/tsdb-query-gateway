# API Reference

## Endpoints

All endpoints are prefixed with `/api/v1`.

### GET /api/v1/query

Execute an instant query against all configured Prometheus endpoints.

**Query Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | PromQL expression |
| `time` | string | No | Evaluation timestamp (Unix or RFC3339) |
| `timeout` | string | No | Query timeout (e.g., `30s`). Capped by the gateway's internal 30s HTTP client timeout. |

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/query?query=up"
```

**Example Response:**
```json
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {"metric": {"__name__": "up", "instance": "localhost:9090"}, "value": [1716000000, "1"]},
      {"metric": {"__name__": "up", "instance": "localhost:9091"}, "value": [1716000000, "1"]}
    ]
  }
}
```

**Error Response:**
```json
{
  "status": "error",
  "errorType": "bad_data",
  "error": "invalid parameter \"query\": 1:1: parse error: no expression found in input"
}
```

### GET /api/v1/query_range

Execute a range query against all configured Prometheus endpoints.

**Query Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | PromQL expression |
| `start` | string | Yes | Start timestamp |
| `end` | string | Yes | End timestamp |
| `step` | string | Yes | Query resolution step width |
| `timeout` | string | No | Query timeout |

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/query_range?query=up&start=1716000000&end=1716003600&step=15s"
```

**Example Response:**
```json
{
  "status": "success",
  "data": {
    "resultType": "matrix",
    "result": [
      {
        "metric": {"__name__": "up"},
        "values": [
          [1716000000, "1"],
          [1716000015, "1"]
        ]
      }
    ]
  }
}
```

### POST Support

Both `/api/v1/query` and `/api/v1/query_range` also accept `POST` requests with form-encoded parameters, matching the Prometheus API specification.

## Response Types

### PromResponse

```go
type PromResponse struct {
    Status    string   `json:"status"`
    Data      PromData `json:"data,omitempty"`
    ErrorType string   `json:"errorType,omitempty"`
    Error     string   `json:"error,omitempty"`
    Warnings  []string `json:"warnings,omitempty"`
}
```

### PromData

```go
type PromData struct {
    ResultType string          `json:"resultType"`
    Result     json.RawMessage `json:"result"`
}
```

## Error Handling

The gateway propagates downstream Prometheus errors with their original HTTP status codes:

| Downstream Status | Meaning |
|-------------------|---------|
| 400 | `bad_data` — invalid query syntax or parameters |
| 422 | `execution` — query execution failed |
| 503 | `timeout` or `cancelled` — downstream unavailable |
| 500 | `internal` — unexpected downstream error |

If the downstream returns an unstructured error, the gateway falls back to HTTP 500 with `server_error`.

## Merged Results

When querying multiple endpoints:

- **`vector`/`matrix`**: Results are concatenated. Two endpoints each returning one metric produce a merged response with two metrics.
- **`scalar`/`string`**: Only the first valid result is returned (these are singleton values, not mergeable).
- **Warnings**: Collected from all endpoints, deduplicated, and returned in the merged response.
