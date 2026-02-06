package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestAPIExecutions_Create verifies creating an execution via POST.
func TestAPIExecutions_Create(t *testing.T) {
	wfID := uniqueID("exec-create")
	yaml := `
main:
  steps:
    - done:
        return: "executed"
`
	name := createWorkflow(t, wfID, yaml)

	body, _ := json.Marshal(map[string]interface{}{})
	url := apiURL(name + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200/201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var exec map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exec)

	execName, _ := exec["name"].(string)
	if execName == "" {
		t.Fatal("expected execution name in response")
	}

	state, _ := exec["state"].(string)
	// State should be ACTIVE or SUCCEEDED (if instant)
	if state != "ACTIVE" && state != "SUCCEEDED" {
		t.Logf("initial state: %s", state)
	}
}

// TestAPIExecutions_Get verifies getting an execution status via GET.
func TestAPIExecutions_Get(t *testing.T) {
	wfID := uniqueID("exec-get")
	yaml := `
main:
  steps:
    - done:
        return: "test"
`
	name := createWorkflow(t, wfID, yaml)

	// Create execution
	body, _ := json.Marshal(map[string]interface{}{})
	url := apiURL(name + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var exec map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exec)
	execName, _ := exec["name"].(string)

	// Wait a bit, then GET execution
	time.Sleep(500 * time.Millisecond)

	getResp, err := http.Get(apiURL(execName))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(getResp.Body)
		t.Fatalf("expected 200, got %d: %s", getResp.StatusCode, string(respBody))
	}

	var execResult map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&execResult)

	resultName, _ := execResult["name"].(string)
	if resultName != execName {
		t.Errorf("expected name %q, got %q", execName, resultName)
	}

	state, _ := execResult["state"].(string)
	t.Logf("execution state: %s", state)
}

// TestAPIExecutions_List verifies listing executions via GET.
func TestAPIExecutions_List(t *testing.T) {
	wfID := uniqueID("exec-list")
	yaml := `
main:
  steps:
    - done:
        return: "test"
`
	name := createWorkflow(t, wfID, yaml)

	// Create a few executions
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]interface{}{})
		url := apiURL(name + "/executions")
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("HTTP error: %v", err)
		}
		resp.Body.Close()
	}

	// Wait for executions to complete
	time.Sleep(1 * time.Second)

	// List executions
	resp, err := http.Get(apiURL(name + "/executions"))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	executions, ok := result["executions"].([]interface{})
	if !ok {
		t.Fatalf("expected executions array in response, got: %v", result)
	}

	if len(executions) < 3 {
		t.Errorf("expected at least 3 executions, got %d", len(executions))
	}
}

// TestAPIExecutions_WithArgument verifies execution with argument parameter.
func TestAPIExecutions_WithArgument(t *testing.T) {
	wfID := uniqueID("exec-args")
	yaml := `
main:
  params: [args]
  steps:
    - done:
        return: ${args.greeting}
`
	name := createWorkflow(t, wfID, yaml)

	args, _ := json.Marshal(map[string]interface{}{
		"greeting": "hello",
	})
	body, _ := json.Marshal(map[string]interface{}{
		"argument": string(args),
	})

	url := apiURL(name + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var exec map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exec)
	execName, _ := exec["name"].(string)

	er := waitForExecution(t, execName, 30*time.Second)
	assertResultEquals(t, er, "hello")
}

// TestAPIExecutions_Cancel verifies cancelling a running execution.
func TestAPIExecutions_Cancel(t *testing.T) {
	wfID := uniqueID("exec-cancel")
	// A long-running workflow using sys.sleep
	yaml := `
main:
  steps:
    - wait:
        call: sys.sleep
        args:
          seconds: 60
    - done:
        return: "should not reach"
`
	name := createWorkflow(t, wfID, yaml)

	// Start execution
	body, _ := json.Marshal(map[string]interface{}{})
	url := apiURL(name + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var exec map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exec)
	execName, _ := exec["name"].(string)

	// Wait a moment, then cancel
	time.Sleep(500 * time.Millisecond)

	cancelURL := apiURL(execName + ":cancel")
	cancelBody, _ := json.Marshal(map[string]interface{}{})
	cancelResp, err := http.Post(cancelURL, "application/json", bytes.NewReader(cancelBody))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer cancelResp.Body.Close()

	if cancelResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(cancelResp.Body)
		t.Fatalf("expected 200, got %d: %s", cancelResp.StatusCode, string(respBody))
	}

	// Verify execution is cancelled
	er := waitForExecution(t, execName, 10*time.Second)
	if er.State != "CANCELLED" {
		t.Errorf("expected CANCELLED, got %s", er.State)
	}
}

// TestAPIExecutions_CompletedResult verifies that completed execution has result.
func TestAPIExecutions_CompletedResult(t *testing.T) {
	wfID := uniqueID("exec-result")
	yaml := `
main:
  steps:
    - done:
        return:
          status: "ok"
          count: 42
`
	name := createWorkflow(t, wfID, yaml)
	er := executeWorkflow(t, name, nil)

	assertSucceeded(t, er)
	assertResultContains(t, er, "status", "ok")
	assertResultContains(t, er, "count", float64(42))
}

// TestAPIExecutions_FailedResult verifies that failed execution has error info.
func TestAPIExecutions_FailedResult(t *testing.T) {
	wfID := uniqueID("exec-fail")
	yaml := `
main:
  steps:
    - fail:
        raise:
          code: 500
          message: "intentional failure"
`
	name := createWorkflow(t, wfID, yaml)
	er := executeWorkflowExpectError(t, name, nil)

	assertFailed(t, er)
	if er.Error == nil {
		t.Fatal("expected error info in failed execution")
	}
}
