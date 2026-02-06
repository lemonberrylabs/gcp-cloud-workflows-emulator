package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestHTTP_GetCall verifies http.get call step.
func TestHTTP_GetCall(t *testing.T) {
	// Start a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "hello from mock",
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
        result: response
    - done:
        return:
          body: ${response.body}
          code: ${response.code}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-get"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "code", float64(200))
}

// TestHTTP_PostCall verifies http.post call step with body.
func TestHTTP_PostCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"received": body,
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.post
        args:
          url: %s
          body:
            name: "test"
            value: 42
        result: response
    - done:
        return:
          code: ${response.code}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-post"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "code", float64(200))
}

// TestHTTP_CustomHeaders verifies that custom headers are sent.
func TestHTTP_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"x_custom": r.Header.Get("X-Custom-Header"),
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
          headers:
            X-Custom-Header: "my-value"
        result: response
    - done:
        return: ${response.body.x_custom}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-headers"), yaml, nil)
	assertResultEquals(t, er, "my-value")
}

// TestHTTP_QueryParams verifies that query params are sent.
func TestHTTP_QueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"action": r.URL.Query().Get("action"),
			"page":   r.URL.Query().Get("page"),
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
          query:
            action: "search"
            page: "1"
        result: response
    - done:
        return: ${response.body}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-query"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "action", "search")
	assertResultContains(t, er, "page", "1")
}

// TestHTTP_ErrorStatusCode verifies that non-2xx status codes raise HttpError.
func TestHTTP_ErrorStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "not found",
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - try_call:
        try:
          steps:
            - call_api:
                call: http.get
                args:
                  url: %s
                result: response
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  code: ${e.code}
                  has_http_error: ${"HttpError" in e.tags}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-error"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	assertResultContains(t, er, "code", float64(404))
	assertResultContains(t, er, "has_http_error", true)
}

// TestHTTP_ResponseStructure verifies that HTTP response has body, code, headers.
func TestHTTP_ResponseStructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test", "response-header")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": "test",
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
        result: response
    - done:
        return:
          has_body: ${response.body != null}
          code: ${response.code}
          has_headers: ${response.headers != null}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-structure"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "has_body", true)
	assertResultContains(t, er, "code", float64(200))
	assertResultContains(t, er, "has_headers", true)
}

// TestHTTP_AllMethods verifies http.put, http.patch, http.delete methods.
func TestHTTP_AllMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"method": r.Method,
		})
	}))
	defer server.Close()

	methods := []struct {
		name     string
		callFunc string
		expected string
	}{
		{"put", "http.put", "PUT"},
		{"patch", "http.patch", "PATCH"},
		{"delete", "http.delete", "DELETE"},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: %s
        args:
          url: %s
        result: response
    - done:
        return: ${response.body.method}
`, m.callFunc, server.URL)

			er := deployAndRun(t, uniqueID("http-"+m.name), yaml, nil)
			assertResultEquals(t, er, m.expected)
		})
	}
}

// TestHTTP_JSONResponseAutoParsed verifies that JSON responses (Content-Type:
// application/json) are automatically parsed into map/list, not returned as
// raw string.
func TestHTTP_JSONResponseAutoParsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"name": "Bob"},
			},
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
        result: response
    - done:
        return:
          body_type: ${type(response.body)}
          user_count: ${len(response.body.users)}
          first_user: ${response.body.users[0].name}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-json-parse"), yaml, nil)
	assertSucceeded(t, er)
	// response.body should be a map (auto-parsed JSON), not a string
	assertResultContains(t, er, "body_type", "map")
	assertResultContains(t, er, "user_count", float64(2))
	assertResultContains(t, er, "first_user", "Alice")
}

// TestHTTP_PlainTextResponseNotParsed verifies that non-JSON responses are
// returned as raw strings.
func TestHTTP_PlainTextResponseNotParsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello plain text"))
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
        result: response
    - done:
        return:
          body_type: ${type(response.body)}
          body: ${response.body}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-plain"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "body_type", "string")
	assertResultContains(t, er, "body", "hello plain text")
}

// TestHTTP_ResponseHeadersLowercased verifies that response headers are
// exposed with lowercased keys, matching GCW behavior.
func TestHTTP_ResponseHeadersLowercased(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.Header().Set("X-Another", "another-value")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        call: http.get
        args:
          url: %s
        result: response
    - done:
        return:
          content_type: ${response.headers["content-type"]}
          custom: ${response.headers["x-custom-header"]}
          another: ${response.headers["x-another"]}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-headers-lower"), yaml, nil)
	assertSucceeded(t, er)
	// Headers should be accessible via lowercase keys
	assertResultContains(t, er, "custom", "custom-value")
	assertResultContains(t, er, "another", "another-value")
}

// TestHTTP_NonSuccessErrorStructure verifies that non-2xx responses raise
// HttpError with full error structure: code, message, tags, headers, body.
func TestHTTP_NonSuccessErrorStructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Error-Id", "err-42")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "access denied",
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - try_call:
        try:
          steps:
            - call_api:
                call: http.get
                args:
                  url: %s
                result: response
        except:
          as: e
          steps:
            - handle:
                return:
                  code: ${e.code}
                  message: ${e.message}
                  has_http_error: ${"HttpError" in e.tags}
                  has_headers: ${e.headers != null}
                  has_body: ${e.body != null}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-err-struct"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "code", float64(403))
	assertResultContains(t, er, "has_http_error", true)
	// HttpError should include response headers and body
	assertResultContains(t, er, "has_headers", true)
	assertResultContains(t, er, "has_body", true)
}

// TestHTTP_ConnectionRefused verifies that connecting to a closed port raises
// ConnectionFailedError (NOT ConnectionError, which is for mid-transfer failures).
func TestHTTP_ConnectionRefused(t *testing.T) {
	// Start and immediately close a server to get a port that refuses connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := server.URL
	server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - try_call:
        try:
          steps:
            - call_api:
                call: http.get
                args:
                  url: %s
                result: response
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  tags: ${e.tags}
                  has_conn_failed: ${"ConnectionFailedError" in e.tags}
`, closedURL)

	er := deployAndRun(t, uniqueID("http-conn-refused"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	// Should be ConnectionFailedError, not ConnectionError
	assertResultContains(t, er, "has_conn_failed", true)
}

// TestHTTP_DefaultRetryDoesNotRetry500 verifies that http.default_retry does
// NOT retry on HTTP 500. GCW's default retry only retries on 429, 502, 503,
// 504, ConnectionError, and TimeoutError.
func TestHTTP_DefaultRetryDoesNotRetry500(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "internal server error",
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - try_call:
        try:
          steps:
            - call_api:
                call: http.get
                args:
                  url: %s
                result: response
        retry:
          predicate: ${http.default_retry}
          max_retries: 5
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  code: ${e.code}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-no-retry-500"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	assertResultContains(t, er, "code", float64(500))

	// http.default_retry should NOT retry 500, so only 1 call
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != 1 {
		t.Errorf("expected exactly 1 call (no retry for 500), got %d", finalCount)
	}
}

// TestHTTP_DefaultRetryDoesRetry503 verifies that http.default_retry DOES
// retry on HTTP 503.
func TestHTTP_DefaultRetryDoesRetry503(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "service unavailable",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": "success",
		})
	}))
	defer server.Close()

	yaml := fmt.Sprintf(`
main:
  steps:
    - call_api:
        try:
          steps:
            - request:
                call: http.get
                args:
                  url: %s
                result: response
        retry:
          predicate: ${http.default_retry}
          max_retries: 5
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
    - done:
        return:
          result: ${response.body.result}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-retry-503"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "result", "success")

	// Should have retried: 2 failures + 1 success = 3 calls
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount < 3 {
		t.Errorf("expected at least 3 calls (retry on 503), got %d", finalCount)
	}
}

// TestHTTP_RetryReExecutesEntireTryBlock verifies that retry re-executes the
// entire try block, not just the failed step. This is critical behavior --
// any state changes made by earlier steps in the try block happen again.
func TestHTTP_RetryReExecutesEntireTryBlock(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	// The try block has two steps:
	// 1. Increment a counter (should run on every retry)
	// 2. Make HTTP call (may fail)
	// If retry re-executes the entire try block, the counter will be
	// incremented on each attempt.
	yaml := fmt.Sprintf(`
main:
  steps:
    - init:
        assign:
          - attempt_count: 0
    - try_block:
        try:
          steps:
            - count_attempt:
                assign:
                  - attempt_count: ${attempt_count + 1}
            - call_api:
                call: http.get
                args:
                  url: %s
                result: response
        retry:
          predicate: ${http.default_retry}
          max_retries: 5
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
    - done:
        return:
          attempts: ${attempt_count}
          success: ${response.body.ok}
`, server.URL)

	er := deployAndRun(t, uniqueID("http-retry-whole"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "success", true)
	// attempt_count should be 3 (retry re-executes entire try block including
	// the counter increment step)
	assertResultContains(t, er, "attempts", float64(3))
}
