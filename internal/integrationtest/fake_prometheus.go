package integrationtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"

	"github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

// FakePrometheus is an HTTP test server that mimics the Prometheus API.
// It records incoming requests and returns configured responses.
// Close may be called multiple times safely (idempotent via sync.Once).
type FakePrometheus struct {
	server *httptest.Server

	mu       sync.RWMutex
	queryRsp *models.PromResponse
	rangeRsp *models.PromResponse
	status   int // HTTP status to return; 0 means 200 OK

	recordDisabled bool
	requests       []RecordedRequest

	closeOnce sync.Once
}

// RecordedRequest captures a single request received by the fake server.
type RecordedRequest struct {
	Path string
	Form url.Values
}

// NewFakePrometheus creates a new fake Prometheus server with default success responses.
func NewFakePrometheus() *FakePrometheus {
	f := &FakePrometheus{
		queryRsp: &models.PromResponse{
			Status: "success",
			Data: models.PromData{
				ResultType: "vector",
				Result:     json.RawMessage(`[]`),
			},
		},
		rangeRsp: &models.PromResponse{
			Status: "success",
			Data: models.PromData{
				ResultType: "matrix",
				Result:     json.RawMessage(`[]`),
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/query", f.handleQuery)
	mux.HandleFunc("/api/v1/query_range", f.handleQueryRange)
	f.server = httptest.NewServer(mux)
	return f
}

// Close shuts down the test server idempotently.
func (f *FakePrometheus) Close() {
	f.closeOnce.Do(func() {
		f.server.Close()
	})
}

// URL returns the base URL of the fake server.
func (f *FakePrometheus) URL() string {
	return f.server.URL
}

// SetQueryResponse configures the response returned by /api/v1/query.
func (f *FakePrometheus) SetQueryResponse(rsp *models.PromResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queryRsp = rsp.Clone()
}

// SetQueryRangeResponse configures the response returned by /api/v1/query_range.
func (f *FakePrometheus) SetQueryRangeResponse(rsp *models.PromResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rangeRsp = rsp.Clone()
}

// SetStatusCode forces a non-200 status code for all endpoints.
func (f *FakePrometheus) SetStatusCode(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = code
}

// GetRequests returns a deep copy of the recorded request history. Safe for concurrent use.
func (f *FakePrometheus) GetRequests() []RecordedRequest {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]RecordedRequest, len(f.requests))
	for i, req := range f.requests {
		formCopy := make(url.Values, len(req.Form))
		for k, v := range req.Form {
			vCopy := make([]string, len(v))
			copy(vCopy, v)
			formCopy[k] = vCopy
		}
		out[i] = RecordedRequest{
			Path: req.Path,
			Form: formCopy,
		}
	}
	return out
}

// ResetRequests clears the recorded request history.
func (f *FakePrometheus) ResetRequests() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = nil
}

// DisableRecording stops request recording. Useful in benchmarks to avoid memory bloat.
func (f *FakePrometheus) DisableRecording() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recordDisabled = true
}

// EnableRecording resumes request recording.
func (f *FakePrometheus) EnableRecording() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recordDisabled = false
}

func (f *FakePrometheus) handleQuery(w http.ResponseWriter, r *http.Request) {
	f.record(r, "/api/v1/query")
	f.mu.RLock()
	status := f.status
	rsp := f.queryRsp
	f.mu.RUnlock()
	writeResponse(w, status, rsp)
}

func (f *FakePrometheus) handleQueryRange(w http.ResponseWriter, r *http.Request) {
	f.record(r, "/api/v1/query_range")
	f.mu.RLock()
	status := f.status
	rsp := f.rangeRsp
	f.mu.RUnlock()
	writeResponse(w, status, rsp)
}

// record captures the request path and form data. Malformed request
// bodies are not expected in these tests, so ParseForm errors are ignored.
func (f *FakePrometheus) record(r *http.Request, path string) {
	_ = r.ParseForm()
	formCopy := make(url.Values, len(r.Form))
	for k, v := range r.Form {
		vCopy := make([]string, len(v))
		copy(vCopy, v)
		formCopy[k] = vCopy
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recordDisabled {
		return
	}
	f.requests = append(f.requests, RecordedRequest{
		Path: path,
		Form: formCopy,
	})
}

func writeResponse(w http.ResponseWriter, status int, body *models.PromResponse) {
	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if body != nil {
		if err := json.NewEncoder(w).Encode(body); err != nil {
			panic(fmt.Sprintf("fake_prometheus: json encode: %v", err))
		}
	}
}
