package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// These tests validate the core Firebase-emulator-style use case:
// Workflow http.* steps make real HTTP calls to localhost services.
// Developers run their services locally and the emulator orchestrates them.

// TestLocalhost_WorkflowCallsLocalService verifies that a workflow can call
// a local HTTP service and get a response, simulating the real GCW calling
// Cloud Run / Cloud Functions.
func TestLocalhost_WorkflowCallsLocalService(t *testing.T) {
	// Start a mock "local service" that returns user data.
	userService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    1,
			"name":  "Alice",
			"email": "alice@example.com",
		})
	}))
	defer userService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - get_user:
        call: http.get
        args:
          url: ${args.user_service_url + "/users/1"}
        result: response
    - done:
        return:
          user_name: ${response.body.name}
          user_email: ${response.body.email}
          status_code: ${response.code}
`
	er := deployAndRun(t, uniqueID("local-svc"), yaml, map[string]interface{}{
		"user_service_url": userService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "user_name", "Alice")
	assertResultContains(t, er, "user_email", "alice@example.com")
	assertResultContains(t, er, "status_code", float64(200))
}

// TestLocalhost_WorkflowOrchestratesMultipleServices verifies that a workflow
// can call multiple local services in sequence, simulating a real orchestration
// pattern (e.g., call user service, then notification service).
func TestLocalhost_WorkflowOrchestratesMultipleServices(t *testing.T) {
	// Service 1: User service
	userService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":  "Bob",
			"email": "bob@example.com",
		})
	}))
	defer userService.Close()

	// Service 2: Notification service (accepts POST with user data)
	var notifReceived map[string]interface{}
	notifService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&notifReceived)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sent": true,
		})
	}))
	defer notifService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - get_user:
        call: http.get
        args:
          url: ${args.user_url}
        result: user_response
    - send_notification:
        call: http.post
        args:
          url: ${args.notif_url}
          body:
            to: ${user_response.body.email}
            message: ${"Welcome, " + user_response.body.name + "!"}
        result: notif_response
    - done:
        return:
          user: ${user_response.body.name}
          notification_sent: ${notif_response.body.sent}
`
	er := deployAndRun(t, uniqueID("local-multi-svc"), yaml, map[string]interface{}{
		"user_url":  userService.URL,
		"notif_url": notifService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "user", "Bob")
	assertResultContains(t, er, "notification_sent", true)

	// Verify the notification service actually received the correct data
	if notifReceived != nil {
		if to, ok := notifReceived["to"].(string); ok && to != "bob@example.com" {
			t.Errorf("notification service received wrong email: %s", to)
		}
	}
}

// TestLocalhost_WorkflowRetryOnServiceFailure verifies that when a local
// service temporarily fails, the workflow retry mechanism retries the call.
func TestLocalhost_WorkflowRetryOnServiceFailure(t *testing.T) {
	var callCount int32

	// Service that fails twice, then succeeds
	flakyService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "service temporarily unavailable",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": "success",
		})
	}))
	defer flakyService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - call_flaky:
        try:
          steps:
            - request:
                call: http.get
                args:
                  url: ${args.service_url}
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
`
	er := deployAndRun(t, uniqueID("local-retry"), yaml, map[string]interface{}{
		"service_url": flakyService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "result", "success")

	// Verify the service was called multiple times (retried)
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount < 3 {
		t.Errorf("expected at least 3 calls to flaky service, got %d", finalCount)
	}
}

// TestLocalhost_WorkflowParallelServiceCalls verifies that a workflow can
// call multiple local services in parallel branches.
func TestLocalhost_WorkflowParallelServiceCalls(t *testing.T) {
	// Service A
	serviceA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "from_a"})
	}))
	defer serviceA.Close()

	// Service B
	serviceB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "from_b"})
	}))
	defer serviceB.Close()

	yaml := `
main:
  params: [args]
  steps:
    - init:
        assign:
          - result_a: ""
          - result_b: ""
    - parallel_calls:
        parallel:
          shared: [result_a, result_b]
          branches:
            - call_a:
                steps:
                  - request_a:
                      call: http.get
                      args:
                        url: ${args.service_a_url}
                      result: resp_a
                  - save_a:
                      assign:
                        - result_a: ${resp_a.body.data}
            - call_b:
                steps:
                  - request_b:
                      call: http.get
                      args:
                        url: ${args.service_b_url}
                      result: resp_b
                  - save_b:
                      assign:
                        - result_b: ${resp_b.body.data}
    - done:
        return:
          a: ${result_a}
          b: ${result_b}
`
	er := deployAndRun(t, uniqueID("local-parallel"), yaml, map[string]interface{}{
		"service_a_url": serviceA.URL,
		"service_b_url": serviceB.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "a", "from_a")
	assertResultContains(t, er, "b", "from_b")
}

// TestLocalhost_WorkflowLoopOverServiceCalls verifies that a workflow can
// iterate over a list and call a local service for each item.
func TestLocalhost_WorkflowLoopOverServiceCalls(t *testing.T) {
	var mu sync.Mutex
	var processedItems []string

	processingService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		item, _ := body["item"].(string)
		mu.Lock()
		processedItems = append(processedItems, item)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"processed": item,
			"status":    "ok",
		})
	}))
	defer processingService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - init:
        assign:
          - items: ["order-1", "order-2", "order-3"]
          - results: []
    - process_loop:
        for:
          value: item
          in: ${items}
          steps:
            - call_service:
                call: http.post
                args:
                  url: ${args.service_url}
                  body:
                    item: ${item}
                result: response
            - collect:
                assign:
                  - results: ${list.concat(results, [response.body.processed])}
    - done:
        return: ${results}
`
	er := deployAndRun(t, uniqueID("local-loop-svc"), yaml, map[string]interface{}{
		"service_url": processingService.URL,
	})
	assertSucceeded(t, er)
	assertResultEquals(t, er, []interface{}{"order-1", "order-2", "order-3"})

	// Verify all items were processed by the local service
	mu.Lock()
	defer mu.Unlock()
	if len(processedItems) != 3 {
		t.Errorf("expected 3 processed items, got %d", len(processedItems))
	}
}

// TestLocalhost_WorkflowConditionalServiceRouting verifies that a workflow
// can route to different local services based on conditions (saga pattern).
func TestLocalhost_WorkflowConditionalServiceRouting(t *testing.T) {
	// Payment service
	paymentService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		amount, _ := body["amount"].(float64)

		w.Header().Set("Content-Type", "application/json")
		if amount > 1000 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"approved": false,
				"reason":   "amount exceeds limit",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"approved":   true,
				"payment_id": "pay-123",
			})
		}
	}))
	defer paymentService.Close()

	// Fulfillment service (only called if payment approved)
	fulfillmentService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"shipped": true,
			"tracking": "TRACK-456",
		})
	}))
	defer fulfillmentService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - process_payment:
        call: http.post
        args:
          url: ${args.payment_url}
          body:
            amount: ${args.amount}
        result: payment
    - check_payment:
        switch:
          - condition: ${payment.body.approved == true}
            steps:
              - fulfill:
                  call: http.post
                  args:
                    url: ${args.fulfillment_url}
                    body:
                      payment_id: ${payment.body.payment_id}
                  result: shipment
              - success:
                  return:
                    status: "completed"
                    tracking: ${shipment.body.tracking}
          - condition: true
            steps:
              - rejected:
                  return:
                    status: "rejected"
                    reason: ${payment.body.reason}
`
	// Test approved path (amount under limit)
	t.Run("approved", func(t *testing.T) {
		er := deployAndRun(t, uniqueID("local-route-ok"), yaml, map[string]interface{}{
			"payment_url":     paymentService.URL,
			"fulfillment_url": fulfillmentService.URL,
			"amount":          float64(500),
		})
		assertSucceeded(t, er)
		assertResultContains(t, er, "status", "completed")
		assertResultContains(t, er, "tracking", "TRACK-456")
	})

	// Test rejected path (amount over limit)
	t.Run("rejected", func(t *testing.T) {
		er := deployAndRun(t, uniqueID("local-route-rej"), yaml, map[string]interface{}{
			"payment_url":     paymentService.URL,
			"fulfillment_url": fulfillmentService.URL,
			"amount":          float64(5000),
		})
		assertSucceeded(t, er)
		assertResultContains(t, er, "status", "rejected")
		assertResultContains(t, er, "reason", "amount exceeds limit")
	})
}

// TestLocalhost_WorkflowErrorHandlingForServiceFailure verifies that when
// a local service returns an error, the workflow's try/except catches it.
func TestLocalhost_WorkflowErrorHandlingForServiceFailure(t *testing.T) {
	errorService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "internal server error",
		})
	}))
	defer errorService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - try_call:
        try:
          steps:
            - call_failing:
                call: http.get
                args:
                  url: ${args.service_url}
                result: response
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  error_code: ${e.code}
                  is_http_error: ${"HttpError" in e.tags}
`
	er := deployAndRun(t, uniqueID("local-err"), yaml, map[string]interface{}{
		"service_url": errorService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	assertResultContains(t, er, "error_code", float64(500))
	assertResultContains(t, er, "is_http_error", true)
}

// TestLocalhost_WorkflowPassesRequestHeaders verifies that workflow http.* calls
// correctly forward custom headers to the local service.
func TestLocalhost_WorkflowPassesRequestHeaders(t *testing.T) {
	headerCheckService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"auth":         r.Header.Get("Authorization"),
			"content_type": r.Header.Get("Content-Type"),
			"custom":       r.Header.Get("X-Custom"),
		})
	}))
	defer headerCheckService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - call_with_headers:
        call: http.get
        args:
          url: ${args.service_url}
          headers:
            Authorization: "Bearer test-token-123"
            X-Custom: "my-custom-value"
        result: response
    - done:
        return:
          auth: ${response.body.auth}
          custom: ${response.body.custom}
`
	er := deployAndRun(t, uniqueID("local-headers"), yaml, map[string]interface{}{
		"service_url": headerCheckService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "auth", "Bearer test-token-123")
	assertResultContains(t, er, "custom", "my-custom-value")
}

// TestLocalhost_WorkflowPassesQueryParams verifies that workflow http.* calls
// correctly forward query parameters to the local service.
func TestLocalhost_WorkflowPassesQueryParams(t *testing.T) {
	queryCheckService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"search":   r.URL.Query().Get("q"),
			"page":     r.URL.Query().Get("page"),
			"per_page": r.URL.Query().Get("per_page"),
		})
	}))
	defer queryCheckService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - call_with_query:
        call: http.get
        args:
          url: ${args.service_url}
          query:
            q: "test search"
            page: "2"
            per_page: "10"
        result: response
    - done:
        return: ${response.body}
`
	er := deployAndRun(t, uniqueID("local-query"), yaml, map[string]interface{}{
		"service_url": queryCheckService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "search", "test search")
	assertResultContains(t, er, "page", "2")
	assertResultContains(t, er, "per_page", "10")
}

// TestLocalhost_WorkflowResponseStructure verifies that the emulator correctly
// exposes HTTP response body, code, and headers to the workflow.
func TestLocalhost_WorkflowResponseStructure(t *testing.T) {
	echoService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-789")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "response body",
		})
	}))
	defer echoService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - call_service:
        call: http.get
        args:
          url: ${args.service_url}
        result: response
    - done:
        return:
          body_message: ${response.body.message}
          status_code: ${response.code}
          has_headers: ${response.headers != null}
`
	er := deployAndRun(t, uniqueID("local-response"), yaml, map[string]interface{}{
		"service_url": echoService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "body_message", "response body")
	assertResultContains(t, er, "status_code", float64(200))
	assertResultContains(t, er, "has_headers", true)
}

// TestLocalhost_WorkflowChainedServiceCalls verifies a multi-step orchestration
// pattern where data from one service call feeds into the next.
func TestLocalhost_WorkflowChainedServiceCalls(t *testing.T) {
	// Step 1: Auth service returns a token
	authService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "jwt-abc-123",
		})
	}))
	defer authService.Close()

	// Step 2: Data service requires the token from auth service
	dataService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer jwt-abc-123" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "unauthorized",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{"item1", "item2", "item3"},
		})
	}))
	defer dataService.Close()

	yaml := `
main:
  params: [args]
  steps:
    - authenticate:
        call: http.post
        args:
          url: ${args.auth_url}
          body:
            client_id: "test-client"
        result: auth_response
    - fetch_data:
        call: http.get
        args:
          url: ${args.data_url}
          headers:
            Authorization: ${"Bearer " + auth_response.body.token}
        result: data_response
    - done:
        return:
          token: ${auth_response.body.token}
          data_count: ${len(data_response.body.data)}
`
	er := deployAndRun(t, uniqueID("local-chained"), yaml, map[string]interface{}{
		"auth_url": authService.URL,
		"data_url": dataService.URL,
	})
	assertSucceeded(t, er)
	assertResultContains(t, er, "token", "jwt-abc-123")
	assertResultContains(t, er, "data_count", float64(3))
}
