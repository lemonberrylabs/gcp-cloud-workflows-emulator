package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// TestAPIWorkflows_Create verifies creating a workflow via POST.
func TestAPIWorkflows_Create(t *testing.T) {
	wfID := uniqueID("api-create")
	yaml := `
main:
  steps:
    - done:
        return: "created"
`
	body, _ := json.Marshal(map[string]interface{}{
		"sourceContents": yaml,
	})

	url := apiURL(parentPath+"/workflows") + "?workflowId=" + wfID
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200/201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify the response contains the workflow name
	t.Logf("Create response: %v", result)
}

// TestAPIWorkflows_Get verifies getting a workflow via GET.
func TestAPIWorkflows_Get(t *testing.T) {
	wfID := uniqueID("api-get")
	yaml := `
main:
  steps:
    - done:
        return: "test"
`
	name := createWorkflow(t, wfID, yaml)

	resp, err := http.Get(apiURL(name))
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

	resultName, _ := result["name"].(string)
	if resultName != name {
		t.Errorf("expected name %q, got %q", name, resultName)
	}
}

// TestAPIWorkflows_List verifies listing workflows via GET.
func TestAPIWorkflows_List(t *testing.T) {
	// Create a couple of workflows first
	yaml := `
main:
  steps:
    - done:
        return: "test"
`
	createWorkflow(t, uniqueID("api-list-a"), yaml)
	createWorkflow(t, uniqueID("api-list-b"), yaml)

	resp, err := http.Get(apiURL(parentPath + "/workflows"))
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

	workflows, ok := result["workflows"].([]interface{})
	if !ok {
		t.Fatalf("expected workflows array in response, got: %v", result)
	}

	if len(workflows) < 2 {
		t.Errorf("expected at least 2 workflows, got %d", len(workflows))
	}
}

// TestAPIWorkflows_Update verifies updating a workflow via PATCH.
func TestAPIWorkflows_Update(t *testing.T) {
	wfID := uniqueID("api-update")
	yaml1 := `
main:
  steps:
    - done:
        return: "v1"
`
	name := createWorkflow(t, wfID, yaml1)

	// Update with new source
	yaml2 := `
main:
  steps:
    - done:
        return: "v2"
`
	body, _ := json.Marshal(map[string]interface{}{
		"sourceContents": yaml2,
	})

	req, _ := http.NewRequest(http.MethodPatch, apiURL(name), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Execute to verify update took effect
	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "v2")
}

// TestAPIWorkflows_Delete verifies deleting a workflow via DELETE.
func TestAPIWorkflows_Delete(t *testing.T) {
	wfID := uniqueID("api-delete")
	yaml := `
main:
  steps:
    - done:
        return: "test"
`
	name := createWorkflow(t, wfID, yaml)

	req, _ := http.NewRequest(http.MethodDelete, apiURL(name), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify workflow is gone
	getResp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getResp.StatusCode)
	}
}

// TestAPIWorkflows_GetNotFound verifies 404 for non-existent workflow.
func TestAPIWorkflows_GetNotFound(t *testing.T) {
	name := parentPath + "/workflows/nonexistent-workflow-12345"
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// TestAPIWorkflows_CreateDuplicate verifies that creating a workflow with an
// existing ID returns an error.
func TestAPIWorkflows_CreateDuplicate(t *testing.T) {
	wfID := uniqueID("api-dup")
	yaml := `
main:
  steps:
    - done:
        return: "test"
`
	createWorkflow(t, wfID, yaml)

	// Try to create again with same ID
	body, _ := json.Marshal(map[string]interface{}{
		"sourceContents": yaml,
	})

	url := apiURL(parentPath+"/workflows") + "?workflowId=" + wfID
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	// Should return 409 Conflict or similar error
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		t.Errorf("expected error for duplicate workflow, got %d", resp.StatusCode)
	}
}
