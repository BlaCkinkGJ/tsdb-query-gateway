package integrationtest

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/client"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/handler"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/internal/service"
	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func setupIntegrationTest(t testing.TB, endpoints []string) *gin.Engine {
	t.Helper()
	promClient := client.NewPrometheusClient(5 * time.Second)
	gatewaySvc := service.NewGatewayService(endpoints, promClient)
	queryHandler := handler.NewQueryHandler(gatewaySvc)

	engine := gin.New()
	api := engine.Group("/api/v1")
	queryHandler.RegisterRoutes(api)
	return engine
}

func TestIntegration_Query_FanOutAndMerge(t *testing.T) {
	fake1 := NewFakePrometheus()
	fake2 := NewFakePrometheus()
	defer fake1.Close()
	defer fake2.Close()

	fake1.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result: json.RawMessage(`[
				{"metric":{"__name__":"up","instance":"host1"},"value":[1,"1"]}
			]`),
		},
	})
	fake2.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result: json.RawMessage(`[
				{"metric":{"__name__":"up","instance":"host2"},"value":[1,"1"]}
			]`),
		},
	})

	engine := setupIntegrationTest(t, []string{fake1.URL(), fake2.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var rsp models.PromResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "success", rsp.Status)
	assert.Equal(t, "vector", rsp.Data.ResultType)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(rsp.Data.Result, &results))
	assert.Len(t, results, 2, "expected 2 metrics merged from 2 endpoints")

	assert.Len(t, fake1.GetRequests(), 1)
	assert.Len(t, fake2.GetRequests(), 1)
}

func TestIntegration_QueryRange_FanOutAndMerge(t *testing.T) {
	fake1 := NewFakePrometheus()
	fake2 := NewFakePrometheus()
	defer fake1.Close()
	defer fake2.Close()

	fake1.SetQueryRangeResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "matrix",
			Result: json.RawMessage(`[
				{"metric":{"__name__":"cpu"},"values":[[1,"0.5"]]}
			]`),
		},
	})
	fake2.SetQueryRangeResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "matrix",
			Result: json.RawMessage(`[
				{"metric":{"__name__":"cpu"},"values":[[2,"0.6"]]}
			]`),
		},
	})

	engine := setupIntegrationTest(t, []string{fake1.URL(), fake2.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query_range?query=cpu&start=0&end=60&step=15", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var rsp models.PromResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "success", rsp.Status)
	assert.Equal(t, "matrix", rsp.Data.ResultType)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(rsp.Data.Result, &results))
	assert.Len(t, results, 2)
	assert.Len(t, fake1.GetRequests(), 1)
	assert.Len(t, fake2.GetRequests(), 1)
}

func TestIntegration_Query_DownstreamErrorPropagation(t *testing.T) {
	fake := NewFakePrometheus()
	defer fake.Close()

	fake.SetStatusCode(http.StatusBadRequest)
	fake.SetQueryResponse(&models.PromResponse{
		Status:    "error",
		ErrorType: "bad_data",
		Error:     "invalid parameter \"query\": 1:1: parse error: no expression found in input",
	})

	engine := setupIntegrationTest(t, []string{fake.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=bad", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "downstream 400 should be propagated to client")

	var rsp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "error", rsp["status"])
	assert.Equal(t, "bad_data", rsp["errorType"])
	assert.Contains(t, rsp["error"], "parse error")
	assert.NotContains(t, rsp["error"], "downstream error", "error message should not contain internal prefix")
}

func TestIntegration_QueryRange_DownstreamErrorPropagation(t *testing.T) {
	fake := NewFakePrometheus()
	defer fake.Close()

	fake.SetStatusCode(http.StatusBadRequest)
	fake.SetQueryRangeResponse(&models.PromResponse{
		Status:    "error",
		ErrorType: "bad_data",
		Error:     "invalid parameter \"step\": duration must be greater than 0",
	})

	engine := setupIntegrationTest(t, []string{fake.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query_range?query=up&start=0&end=1&step=0", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var rsp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "bad_data", rsp["errorType"])
}

func TestIntegration_Query_DownstreamFailurePropagates(t *testing.T) {
	fakeOK := NewFakePrometheus()
	fakeErr := NewFakePrometheus()
	defer fakeOK.Close()
	defer fakeErr.Close()

	fakeOK.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{"__name__":"up"},"value":[1,"1"]}]`),
		},
	})
	fakeErr.SetStatusCode(http.StatusServiceUnavailable)
	fakeErr.SetQueryResponse(&models.PromResponse{
		Status:    "error",
		ErrorType: "timeout",
		Error:     "query timed out",
	})

	engine := setupIntegrationTest(t, []string{fakeOK.URL(), fakeErr.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
	engine.ServeHTTP(w, req)

	// The gateway uses errgroup.Wait() which returns the first error.
	// A single failing endpoint causes the entire request to fail,
	// and the downstream error status is propagated as-is.
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestIntegration_Query_EmptyResult(t *testing.T) {
	fake := NewFakePrometheus()
	defer fake.Close()

	fake.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[]`),
		},
	})

	engine := setupIntegrationTest(t, []string{fake.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=nonexistent_metric", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var rsp models.PromResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(rsp.Data.Result, &results))
	assert.Len(t, results, 0)
}

func TestIntegration_Query_WarningsAggregation(t *testing.T) {
	fake1 := NewFakePrometheus()
	fake2 := NewFakePrometheus()
	defer fake1.Close()
	defer fake2.Close()

	fake1.SetQueryResponse(&models.PromResponse{
		Status:   "success",
		Warnings: []string{"deprecation warning"},
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{},"value":[1,"1"]}]`),
		},
	})
	fake2.SetQueryResponse(&models.PromResponse{
		Status:   "success",
		Warnings: []string{"partial response", "deprecation warning"},
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{},"value":[2,"2"]}]`),
		},
	})

	engine := setupIntegrationTest(t, []string{fake1.URL(), fake2.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var rsp models.PromResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.ElementsMatch(t, []string{"deprecation warning", "partial response"}, rsp.Warnings)
}

func TestIntegration_Query_MissingQueryParameter(t *testing.T) {
	engine := setupIntegrationTest(t, []string{"http://unused"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var rsp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "bad_data", rsp["errorType"])
	assert.Contains(t, rsp["error"], "missing query")
}

func TestIntegration_QueryRange_MissingRequiredParameters(t *testing.T) {
	engine := setupIntegrationTest(t, []string{"http://unused"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query_range?query=up", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var rsp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "bad_data", rsp["errorType"])
	assert.Contains(t, rsp["error"], "missing required parameters")
}

// TestIntegration_Query_ScalarResultType verifies that scalar results
// (format [timestamp, value]) are merged by concatenation, preserving
// the first endpoint's value in the combined result.
func TestIntegration_Query_ScalarResultType(t *testing.T) {
	fake1 := NewFakePrometheus()
	fake2 := NewFakePrometheus()
	defer fake1.Close()
	defer fake2.Close()

	fake1.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "scalar",
			Result:     json.RawMessage(`[1234567890, "42"]`),
		},
	})
	fake2.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "scalar",
			Result:     json.RawMessage(`[1234567890, "99"]`),
		},
	})

	engine := setupIntegrationTest(t, []string{fake1.URL(), fake2.URL()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=count(up)", nil)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var rsp models.PromResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
	assert.Equal(t, "scalar", rsp.Data.ResultType)

	// Scalar format is [timestamp, value]. The gateway concatenates raw
	// results, so the first endpoint's scalar appears first.
	var scalarResult []any
	require.NoError(t, json.Unmarshal(rsp.Data.Result, &scalarResult))
	require.Len(t, scalarResult, 2)
	assert.Equal(t, float64(1234567890), scalarResult[0])
	assert.Equal(t, "42", scalarResult[1])
	assert.Len(t, fake1.GetRequests(), 1)
	assert.Len(t, fake2.GetRequests(), 1)
}

func TestIntegration_Query_NoEndpoints(t *testing.T) {
	engine := setupIntegrationTest(t, []string{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
	engine.ServeHTTP(w, req)

	// Gateway service returns an error when no endpoints are configured.
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIntegration_Query_PostForm(t *testing.T) {
	fake := NewFakePrometheus()
	defer fake.Close()

	fake.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{"__name__":"up"},"value":[1,"1"]}]`),
		},
	})

	engine := setupIntegrationTest(t, []string{fake.URL()})

	w := httptest.NewRecorder()
	form := url.Values{"query": {"up"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify the fake received the POST form parameter.
	reqs := fake.GetRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/api/v1/query", reqs[0].Path)
	assert.Equal(t, "up", reqs[0].Form.Get("query"))
}

// BenchmarkIntegration_Query measures end-to-end throughput of the gateway
// with a single fake downstream.
func BenchmarkIntegration_Query(b *testing.B) {
	fake := NewFakePrometheus()
	defer fake.Close()

	fake.SetQueryResponse(&models.PromResponse{
		Status: "success",
		Data: models.PromData{
			ResultType: "vector",
			Result:     json.RawMessage(`[{"metric":{"__name__":"up"},"value":[1,"1"]}]`),
		},
	})

	engine := setupIntegrationTest(b, []string{fake.URL()})

	fake.DisableRecording()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

// ExampleFakePrometheus demonstrates configuring a response and inspecting
// recorded requests.
//
// Output: status=503 requests=1
func ExampleFakePrometheus() {
	fake := NewFakePrometheus()
	defer fake.Close()

	fake.SetStatusCode(http.StatusServiceUnavailable)

	resp, err := http.Get(fake.URL() + "/api/v1/query?query=up")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("status="+strconv.Itoa(resp.StatusCode), "requests="+strconv.Itoa(len(fake.GetRequests())))
}
