package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	defaultProject  = "my-project"
	defaultLocation = "us-central1"
	// parentPath is set in init() based on project/location.
	parentPath string
)

// testServer holds the base URL of a running emulator instance for tests.
var testServer string

func init() {
	// Support the standard GCP emulator host convention first.
	testServer = os.Getenv("WORKFLOWS_EMULATOR_HOST")
	if testServer == "" {
		// Fallback to explicit test URL.
		testServer = os.Getenv("GCW_EMULATOR_URL")
	}
	if testServer == "" {
		testServer = "http://localhost:8787"
	}
	// Ensure the URL has a scheme.
	if !strings.HasPrefix(testServer, "http://") && !strings.HasPrefix(testServer, "https://") {
		testServer = "http://" + testServer
	}

	// Allow overriding project/location to match emulator config.
	if p := os.Getenv("PROJECT"); p != "" {
		defaultProject = p
	}
	if l := os.Getenv("LOCATION"); l != "" {
		defaultLocation = l
	}
	parentPath = "projects/" + defaultProject + "/locations/" + defaultLocation
}

// loadWorkflow reads a YAML workflow definition from the testdata directory.
func loadWorkflow(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", "workflows", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load workflow %s: %v", name, err)
	}
	return string(data)
}

// apiURL builds a full URL for the given API path.
func apiURL(path string) string {
	return strings.TrimRight(testServer, "/") + "/v1/" + path
}

// createWorkflow deploys a workflow definition to the emulator via the
// Workflows CRUD API and returns the workflow resource name
// (e.g., "projects/test-project/locations/us-central1/workflows/my-wf").
func createWorkflow(t *testing.T, workflowID, sourceYAML string) string {
	t.Helper()

	body := map[string]interface{}{
		"sourceContents": sourceYAML,
	}
	data, _ := json.Marshal(body)

	url := apiURL(parentPath+"/workflows") + "?workflowId=" + workflowID
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("createWorkflow HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("createWorkflow failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("createWorkflow decode error: %v", err)
	}

	name, ok := result["name"].(string)
	if !ok {
		// For operations that return immediately, the name might be nested.
		if response, ok := result["response"].(map[string]interface{}); ok {
			name, _ = response["name"].(string)
		}
		if name == "" {
			name = parentPath + "/workflows/" + workflowID
		}
	}
	return name
}

// executeWorkflow starts an execution of the named workflow with optional
// arguments and waits for it to complete, returning the execution result.
func executeWorkflow(t *testing.T, workflowName string, args map[string]interface{}) executionResult {
	t.Helper()

	body := map[string]interface{}{}
	if args != nil {
		argsJSON, _ := json.Marshal(args)
		body["argument"] = string(argsJSON)
	}
	data, _ := json.Marshal(body)

	url := apiURL(workflowName + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("executeWorkflow HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("executeWorkflow failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var exec map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&exec); err != nil {
		t.Fatalf("executeWorkflow decode error: %v", err)
	}

	execName, _ := exec["name"].(string)
	if execName == "" {
		t.Fatalf("executeWorkflow: no execution name in response: %v", exec)
	}

	return waitForExecution(t, execName, 30*time.Second)
}

// executeWorkflowExpectError starts an execution that is expected to fail.
func executeWorkflowExpectError(t *testing.T, workflowName string, args map[string]interface{}) executionResult {
	t.Helper()

	body := map[string]interface{}{}
	if args != nil {
		argsJSON, _ := json.Marshal(args)
		body["argument"] = string(argsJSON)
	}
	data, _ := json.Marshal(body)

	url := apiURL(workflowName + "/executions")
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("executeWorkflow HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("executeWorkflow failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var exec map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&exec); err != nil {
		t.Fatalf("executeWorkflow decode error: %v", err)
	}

	execName, _ := exec["name"].(string)
	if execName == "" {
		t.Fatalf("executeWorkflow: no execution name in response: %v", exec)
	}

	return waitForExecution(t, execName, 30*time.Second)
}

// executionResult represents the outcome of a workflow execution.
type executionResult struct {
	Name   string
	State  string // SUCCEEDED, FAILED, CANCELLED
	Result interface{}
	Error  map[string]interface{}
	Raw    map[string]interface{}
}

// waitForExecution polls the execution until it reaches a terminal state.
func waitForExecution(t *testing.T, execName string, timeout time.Duration) executionResult {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("execution %s did not complete within %s", execName, timeout)
		}

		resp, err := http.Get(apiURL(execName))
		if err != nil {
			t.Fatalf("waitForExecution HTTP error: %v", err)
		}

		var exec map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&exec); err != nil {
			resp.Body.Close()
			t.Fatalf("waitForExecution decode error: %v", err)
		}
		resp.Body.Close()

		state, _ := exec["state"].(string)
		if state == "SUCCEEDED" || state == "FAILED" || state == "CANCELLED" {
			er := executionResult{
				Name:  execName,
				State: state,
				Raw:   exec,
			}

			if resultStr, ok := exec["result"].(string); ok {
				var parsed interface{}
				if err := json.Unmarshal([]byte(resultStr), &parsed); err == nil {
					er.Result = parsed
				} else {
					er.Result = resultStr
				}
			} else {
				er.Result = exec["result"]
			}

			if errMap, ok := exec["error"].(map[string]interface{}); ok {
				er.Error = errMap
			}

			return er
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// deployAndRun is a convenience that creates a workflow from inline YAML,
// executes it, and returns the result.
func deployAndRun(t *testing.T, workflowID, yaml string, args map[string]interface{}) executionResult {
	t.Helper()
	name := createWorkflow(t, workflowID, yaml)
	return executeWorkflow(t, name, args)
}

// deployAndRunExpectError is a convenience for workflows expected to fail.
func deployAndRunExpectError(t *testing.T, workflowID, yaml string, args map[string]interface{}) executionResult {
	t.Helper()
	name := createWorkflow(t, workflowID, yaml)
	return executeWorkflowExpectError(t, name, args)
}

// assertSucceeded checks that the execution succeeded.
func assertSucceeded(t *testing.T, er executionResult) {
	t.Helper()
	if er.State != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED but got %s; error: %v; raw: %v", er.State, er.Error, er.Raw)
	}
}

// assertFailed checks that the execution failed.
func assertFailed(t *testing.T, er executionResult) {
	t.Helper()
	if er.State != "FAILED" {
		t.Fatalf("expected FAILED but got %s; result: %v; raw: %v", er.State, er.Result, er.Raw)
	}
}

// assertResultEquals checks that the execution result matches the expected value.
func assertResultEquals(t *testing.T, er executionResult, expected interface{}) {
	t.Helper()
	assertSucceeded(t, er)

	// Normalize both sides to JSON for comparison.
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(er.Result)

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("result mismatch:\n  expected: %s\n  actual:   %s", string(expectedJSON), string(actualJSON))
	}
}

// assertResultContains checks that the execution result map contains the given key-value pairs.
func assertResultContains(t *testing.T, er executionResult, key string, expected interface{}) {
	t.Helper()
	assertSucceeded(t, er)

	resultMap, ok := er.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be a map, got %T: %v", er.Result, er.Result)
	}

	actual, exists := resultMap[key]
	if !exists {
		t.Fatalf("result map missing key %q; result: %v", key, resultMap)
	}

	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)
	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("result[%q] mismatch:\n  expected: %s\n  actual:   %s", key, string(expectedJSON), string(actualJSON))
	}
}

// parseErrorPayload extracts the structured error fields from the execution
// error. The API stores the WorkflowError as a JSON string in the "payload"
// field. If the payload is a JSON object with "message"/"tags", we parse it.
// Otherwise, we treat the raw payload string as the message.
func parseErrorPayload(er executionResult) (msg string, tags []string) {
	if er.Error == nil {
		return "", nil
	}

	// First check if error has direct "message" field (possible future format).
	if m, ok := er.Error["message"].(string); ok {
		msg = m
		if t, ok := er.Error["tags"].([]interface{}); ok {
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}
		return msg, tags
	}

	// Current format: error.payload is a JSON string containing {message, code, tags}.
	payload, _ := er.Error["payload"].(string)
	if payload == "" {
		return "", nil
	}

	// Try to parse as JSON object.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &parsed); err == nil {
		msg, _ = parsed["message"].(string)
		if t, ok := parsed["tags"].([]interface{}); ok {
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}
		return msg, tags
	}

	// Payload is a plain string error message.
	return payload, nil
}

// assertErrorContains checks that the execution failed and the error message contains the substring.
func assertErrorContains(t *testing.T, er executionResult, substr string) {
	t.Helper()
	assertFailed(t, er)

	if er.Error == nil {
		t.Fatalf("expected error map but got nil")
	}

	msg, _ := parseErrorPayload(er)
	if !strings.Contains(strings.ToLower(msg), strings.ToLower(substr)) {
		t.Errorf("error message %q does not contain %q; raw error: %v", msg, substr, er.Error)
	}
}

// assertErrorHasTag checks that the execution failed with an error that includes the given tag.
func assertErrorHasTag(t *testing.T, er executionResult, tag string) {
	t.Helper()
	assertFailed(t, er)

	if er.Error == nil {
		t.Fatalf("expected error map but got nil")
	}

	_, tags := parseErrorPayload(er)
	for _, t2 := range tags {
		if t2 == tag {
			return
		}
	}
	t.Errorf("error does not have tag %q; tags: %v; raw error: %v", tag, tags, er.Error)
}

// uniqueID generates a unique workflow ID for test isolation.
func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// deleteWorkflow removes a workflow by name (cleanup).
func deleteWorkflow(t *testing.T, workflowName string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, apiURL(workflowName), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("deleteWorkflow warning: %v", err)
		return
	}
	resp.Body.Close()
}
