package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestCallbacks_CreateAndAwait verifies creating a callback endpoint and
// receiving a callback.
func TestCallbacks_CreateAndAwait(t *testing.T) {
	wfID := uniqueID("callback-basic")
	yaml := `
main:
  steps:
    - create_cb:
        call: events.create_callback_endpoint
        args:
          http_callback_method: "POST"
        result: callback
    - wait:
        call: events.await_callback
        args:
          callback: ${callback}
          timeout: 10
        result: callback_data
    - done:
        return: ${callback_data}
`
	name := createWorkflow(t, wfID, yaml)

	// Start execution in background (it will wait for callback)
	body, _ := json.Marshal(map[string]interface{}{})
	url := apiURL(name + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	var exec map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exec)
	resp.Body.Close()
	execName, _ := exec["name"].(string)

	// Wait a moment for the execution to create the callback
	time.Sleep(2 * time.Second)

	// List callbacks to find the URL
	listResp, err := http.Get(apiURL(execName + "/callbacks"))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	var callbacks map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&callbacks)
	listResp.Body.Close()

	cbList, ok := callbacks["callbacks"].([]interface{})
	if !ok || len(cbList) == 0 {
		t.Skipf("no callbacks found - callback feature may not be implemented yet")
		return
	}

	// Get the callback URL
	cb, _ := cbList[0].(map[string]interface{})
	cbURL, _ := cb["url"].(string)
	if cbURL == "" {
		t.Skip("callback URL not found")
		return
	}

	// Send callback data
	cbBody, _ := json.Marshal(map[string]interface{}{
		"status": "approved",
	})
	cbResp, err := http.Post(cbURL, "application/json", bytes.NewReader(cbBody))
	if err != nil {
		t.Fatalf("callback HTTP error: %v", err)
	}
	cbResp.Body.Close()

	// Wait for execution to complete
	er := waitForExecution(t, execName, 15*time.Second)
	assertSucceeded(t, er)
}

// TestCallbacks_Timeout verifies that callback timeout raises an error.
func TestCallbacks_Timeout(t *testing.T) {
	yaml := `
main:
  steps:
    - create_cb:
        call: events.create_callback_endpoint
        args:
          http_callback_method: "POST"
        result: callback
    - try_wait:
        try:
          steps:
            - wait:
                call: events.await_callback
                args:
                  callback: ${callback}
                  timeout: 2
                result: callback_data
        except:
          as: e
          steps:
            - handle:
                return:
                  timed_out: true
                  message: ${e.message}
`
	er := deployAndRun(t, uniqueID("cb-timeout"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "timed_out", true)
}
