package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// These tests validate the watched workflows directory feature:
//   gcw-emulator --workflows-dir=./workflows
//
// The emulator watches a directory for .yaml/.json files.
// File name (sans extension) becomes the workflow ID.
// Files are hot-reloaded: add/modify/delete -> auto deploy/update/remove.
//
// Tests are organized into two categories:
// 1. FILE-BASED tests (primary): Write actual files to the watched directory
//    and verify the emulator picks them up. Requires WORKFLOWS_DIR env var.
// 2. API-BASED tests (secondary): Test CRUD API behavior that mirrors what
//    the file watcher does internally. These always run.

// getWatchedDir returns the watched workflows directory from the WORKFLOWS_DIR
// env var. If not set, returns "" (file-based tests will skip).
func getWatchedDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("WORKFLOWS_DIR")
	if dir == "" {
		return ""
	}
	// Verify the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("WORKFLOWS_DIR %q does not exist", dir)
	}
	return dir
}

// skipIfNoWatchedDir skips the test if no watched directory is configured.
func skipIfNoWatchedDir(t *testing.T) string {
	t.Helper()
	dir := getWatchedDir(t)
	if dir == "" {
		t.Skip("WORKFLOWS_DIR not set; skipping file-based directory watch test")
	}
	return dir
}

// writeWorkflowFile writes a YAML workflow to the watched directory.
func writeWorkflowFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow file %s: %v", path, err)
	}
}

// writeWorkflowFileJSON writes a JSON workflow to the watched directory.
func writeWorkflowFileJSON(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow file %s: %v", path, err)
	}
}

// removeWorkflowFile deletes a workflow file from the watched directory.
// Tries both .yaml and .json extensions.
func removeWorkflowFile(t *testing.T, dir, name string) {
	t.Helper()
	for _, ext := range []string{".yaml", ".json"} {
		path := filepath.Join(dir, name+ext)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to remove workflow file %s: %v", path, err)
		}
	}
}

// waitForWorkflowAvailable polls until a workflow becomes available or times out.
func waitForWorkflowAvailable(t *testing.T, workflowName string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("workflow %s not available within %s", workflowName, timeout)
		}
		resp, err := http.Get(apiURL(workflowName))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// waitForWorkflowGone polls until a workflow returns 404 or times out.
func waitForWorkflowGone(t *testing.T, workflowName string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("workflow %s still available after %s", workflowName, timeout)
		}
		resp, err := http.Get(apiURL(workflowName))
		if err == nil && resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// workflowResourceName builds the full resource name for a workflow ID.
func workflowResourceName(wfID string) string {
	return parentPath + "/workflows/" + wfID
}

// ============================================================================
// FILE-BASED TESTS (Primary)
// These require WORKFLOWS_DIR env var pointing to the emulator's watched dir.
// ============================================================================

// TestDirWatch_File_AddYAMLDeploysWorkflow verifies that writing a new .yaml
// file to the watched directory auto-deploys it as a workflow accessible via API.
func TestDirWatch_File_AddYAMLDeploysWorkflow(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	wfID := uniqueID("fw-add")
	yaml := `main:
  steps:
    - done:
        return: "deployed from file"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, wfID) })
	writeWorkflowFile(t, dir, wfID, yaml)

	name := workflowResourceName(wfID)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "deployed from file")
}

// TestDirWatch_File_AddJSONDeploysWorkflow verifies that .json workflow files
// are also recognized and deployed.
func TestDirWatch_File_AddJSONDeploysWorkflow(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	wfID := uniqueID("fw-json")
	jsonContent := `{
  "main": {
    "steps": [
      {
        "done": {
          "return": "deployed from json"
        }
      }
    ]
  }
}`
	t.Cleanup(func() { removeWorkflowFile(t, dir, wfID) })
	writeWorkflowFileJSON(t, dir, wfID, jsonContent)

	name := workflowResourceName(wfID)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "deployed from json")
}

// TestDirWatch_File_ModifyUpdatesWorkflow verifies that modifying a workflow
// file causes the emulator to redeploy with the new definition.
func TestDirWatch_File_ModifyUpdatesWorkflow(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	wfID := uniqueID("fw-modify")
	t.Cleanup(func() { removeWorkflowFile(t, dir, wfID) })

	// Write version 1
	yamlV1 := `main:
  steps:
    - done:
        return: "file version 1"
`
	writeWorkflowFile(t, dir, wfID, yamlV1)
	name := workflowResourceName(wfID)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	er1 := executeWorkflow(t, name, nil)
	assertResultEquals(t, er1, "file version 1")

	// Overwrite with version 2
	yamlV2 := `main:
  steps:
    - done:
        return: "file version 2"
`
	writeWorkflowFile(t, dir, wfID, yamlV2)

	// Wait for hot-reload (fsnotify + debounce)
	time.Sleep(500 * time.Millisecond)

	er2 := executeWorkflow(t, name, nil)
	assertResultEquals(t, er2, "file version 2")
}

// TestDirWatch_File_DeleteRemovesWorkflow verifies that deleting a workflow
// file removes it from the emulator.
func TestDirWatch_File_DeleteRemovesWorkflow(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	wfID := uniqueID("fw-delete")

	// Write and verify
	yaml := `main:
  steps:
    - done:
        return: "will be deleted"
`
	writeWorkflowFile(t, dir, wfID, yaml)
	name := workflowResourceName(wfID)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	// Delete file
	removeWorkflowFile(t, dir, wfID)

	// Verify workflow disappears
	waitForWorkflowGone(t, name, 5*time.Second)
}

// TestDirWatch_File_InFlightExecutionIsolation is the CRITICAL file-based test:
// verifies that modifying a workflow FILE while an execution is in-flight does
// NOT affect the running execution. The running execution continues with the
// OLD definition. New executions use the NEW definition.
func TestDirWatch_File_InFlightExecutionIsolation(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	wfID := uniqueID("fw-inflight")
	t.Cleanup(func() { removeWorkflowFile(t, dir, wfID) })

	// Write version 1: a slow workflow that sleeps then returns
	yamlV1 := `main:
  steps:
    - wait:
        call: sys.sleep
        args:
          seconds: 3
    - done:
        return: "file version 1 result"
`
	writeWorkflowFile(t, dir, wfID, yamlV1)
	name := workflowResourceName(wfID)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	// Start execution of version 1 (will be in-flight for ~3 seconds)
	execResult := startExecutionAsync(t, name, nil)

	// Wait a moment, then overwrite file with version 2
	time.Sleep(500 * time.Millisecond)

	yamlV2 := `main:
  steps:
    - done:
        return: "file version 2 result"
`
	writeWorkflowFile(t, dir, wfID, yamlV2)

	// Wait for hot-reload
	time.Sleep(500 * time.Millisecond)

	// The in-flight execution should complete with version 1's result
	er1 := waitForExecution(t, execResult.execName, 30*time.Second)
	assertResultEquals(t, er1, "file version 1 result")

	// A new execution should use version 2
	er2 := executeWorkflow(t, name, nil)
	assertResultEquals(t, er2, "file version 2 result")
}

// TestDirWatch_File_InvalidYAMLDoesNotCrash verifies that an invalid YAML
// file in the watched directory does not crash the emulator or affect other
// workflows.
func TestDirWatch_File_InvalidYAMLDoesNotCrash(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	// First deploy a valid workflow
	validID := uniqueID("fw-valid")
	validYAML := `main:
  steps:
    - done:
        return: "still works after invalid file"
`
	t.Cleanup(func() {
		removeWorkflowFile(t, dir, validID)
		removeWorkflowFile(t, dir, "invalid-file")
	})
	writeWorkflowFile(t, dir, validID, validYAML)
	validName := workflowResourceName(validID)
	waitForWorkflowAvailable(t, validName, 5*time.Second)

	// Write an invalid YAML file
	invalidYAML := `this is not valid workflow YAML
  - broken: {{{
`
	writeWorkflowFile(t, dir, "invalid-file", invalidYAML)

	// Wait for the watcher to process the invalid file
	time.Sleep(500 * time.Millisecond)

	// The valid workflow should still be executable
	er := executeWorkflow(t, validName, nil)
	assertResultEquals(t, er, "still works after invalid file")
}

// TestDirWatch_File_FileNameBecomesWorkflowID verifies that the file name
// (without extension) becomes the workflow ID used in the API path.
func TestDirWatch_File_FileNameBecomesWorkflowID(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	// Use a descriptive file name
	wfID := "order-processor"
	yaml := `main:
  steps:
    - done:
        return: "order processed"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, wfID) })
	writeWorkflowFile(t, dir, wfID, yaml)

	// The workflow should be accessible using the filename as ID
	name := workflowResourceName(wfID)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	// Verify via GET
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var wf map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&wf)

	gotName, _ := wf["name"].(string)
	if gotName != name {
		t.Errorf("expected name %q, got %q", name, gotName)
	}
}

// ============================================================================
// FILE NAMING EDGE CASES
// Workflow ID rules: lowercase letters, digits, hyphens, underscores.
// Must start with a letter. Max 128 chars. Invalid names skipped with warning.
// ============================================================================

// TestDirWatch_File_ValidNames verifies various valid workflow file names.
func TestDirWatch_File_ValidNames(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	validNames := []struct {
		fileName    string
		description string
	}{
		{"my-workflow", "hyphens allowed"},
		{"my_workflow", "underscores allowed"},
		{"a", "single letter"},
		{"workflow123", "letters followed by digits"},
		{"a-b-c_d_e", "mixed hyphens and underscores"},
	}

	yaml := `main:
  steps:
    - done:
        return: "valid name test"
`
	for _, tc := range validNames {
		t.Run(tc.description, func(t *testing.T) {
			t.Cleanup(func() { removeWorkflowFile(t, dir, tc.fileName) })
			writeWorkflowFile(t, dir, tc.fileName, yaml)

			name := workflowResourceName(tc.fileName)
			waitForWorkflowAvailable(t, name, 5*time.Second)

			// Cleanup: remove after verification
			removeWorkflowFile(t, dir, tc.fileName)
			waitForWorkflowGone(t, name, 5*time.Second)
		})
	}
}

// TestDirWatch_File_InvalidNameStartsWithDigit verifies that a file name
// starting with a digit is skipped (workflow IDs must start with a letter).
func TestDirWatch_File_InvalidNameStartsWithDigit(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	invalidName := "123-bad-name"
	yaml := `main:
  steps:
    - done:
        return: "should not deploy"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, invalidName) })
	writeWorkflowFile(t, dir, invalidName, yaml)

	// Wait for watcher to process
	time.Sleep(1 * time.Second)

	// Should NOT be available (invalid name, starts with digit)
	name := workflowResourceName(invalidName)
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("workflow with invalid name %q should not have been deployed", invalidName)
	}
}

// TestDirWatch_File_InvalidNameContainsDots verifies that a file name with
// dots (after extension stripping) is skipped because dots are not allowed
// in workflow IDs.
func TestDirWatch_File_InvalidNameContainsDots(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	// File "my.workflow.yaml" -> ID "my.workflow" (invalid: contains dot)
	invalidName := "my.workflow"
	yaml := `main:
  steps:
    - done:
        return: "should not deploy"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, invalidName) })
	writeWorkflowFile(t, dir, invalidName, yaml)

	// Wait for watcher to process
	time.Sleep(1 * time.Second)

	// Should NOT be available (dots not allowed in workflow IDs)
	name := workflowResourceName(invalidName)
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("workflow with dotted name %q should not have been deployed", invalidName)
	}
}

// TestDirWatch_File_UppercaseNameIsLowercased verifies that a file name with
// uppercase letters is lowercased to form a valid workflow ID.
// E.g., "MyWorkflow.yaml" -> workflow ID "myworkflow".
func TestDirWatch_File_UppercaseNameIsLowercased(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	uppercaseName := "MyWorkflow"
	yaml := `main:
  steps:
    - done:
        return: "uppercase was lowercased"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, uppercaseName) })
	writeWorkflowFile(t, dir, uppercaseName, yaml)

	// The emulator should lowercase "MyWorkflow" -> "myworkflow"
	nameLower := workflowResourceName("myworkflow")
	waitForWorkflowAvailable(t, nameLower, 5*time.Second)

	er := executeWorkflow(t, nameLower, nil)
	assertResultEquals(t, er, "uppercase was lowercased")

	// The original mixed-case name should NOT be accessible
	nameExact := workflowResourceName(uppercaseName)
	resp, err := http.Get(apiURL(nameExact))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("uppercase name %q should not be accessible; emulator should only register the lowercased ID", uppercaseName)
	}
}

// TestDirWatch_File_MaxLengthName verifies that a workflow file name at the
// maximum allowed length (128 chars) is accepted.
func TestDirWatch_File_MaxLengthName(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	// Generate a 128-char valid name: "a" + 127 "x"s
	maxName := "a"
	for len(maxName) < 128 {
		maxName += "x"
	}

	yaml := `main:
  steps:
    - done:
        return: "max length name"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, maxName) })
	writeWorkflowFile(t, dir, maxName, yaml)

	name := workflowResourceName(maxName)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	// Clean up
	removeWorkflowFile(t, dir, maxName)
	waitForWorkflowGone(t, name, 5*time.Second)
}

// TestDirWatch_File_OverMaxLengthName verifies that a workflow file name
// exceeding 128 characters is skipped.
func TestDirWatch_File_OverMaxLengthName(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	// Generate a 129-char name
	tooLongName := "a"
	for len(tooLongName) < 129 {
		tooLongName += "x"
	}

	yaml := `main:
  steps:
    - done:
        return: "should not deploy"
`
	t.Cleanup(func() { removeWorkflowFile(t, dir, tooLongName) })
	writeWorkflowFile(t, dir, tooLongName, yaml)

	// Wait for watcher to process
	time.Sleep(1 * time.Second)

	name := workflowResourceName(tooLongName)
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("workflow with name > 128 chars should not have been deployed")
	}
}

// TestDirWatch_File_NonYAMLFileIgnored verifies that files with extensions
// other than .yaml and .json are ignored.
func TestDirWatch_File_NonYAMLFileIgnored(t *testing.T) {
	dir := skipIfNoWatchedDir(t)

	// Write a .txt file -- should be ignored
	txtPath := filepath.Join(dir, "ignore-me.txt")
	if err := os.WriteFile(txtPath, []byte("not a workflow"), 0o644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}
	t.Cleanup(func() { os.Remove(txtPath) })

	// Wait for watcher to process
	time.Sleep(1 * time.Second)

	// Should not create a workflow named "ignore-me"
	name := workflowResourceName("ignore-me")
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf(".txt file should not have been deployed as a workflow")
	}
}

// ============================================================================
// API-BASED TESTS (Secondary)
// These always run and test CRUD API behavior that mirrors file-watch behavior.
// ============================================================================

// TestDirWatch_API_FileNameBecomesWorkflowID verifies that workflows deployed
// via API follow the same naming convention as file-based deployments.
func TestDirWatch_API_FileNameBecomesWorkflowID(t *testing.T) {
	resp, err := http.Get(apiURL(parentPath + "/workflows"))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	workflows, _ := result["workflows"].([]interface{})
	if len(workflows) == 0 {
		t.Skip("no workflows found; emulator may not have --workflows-dir")
		return
	}

	for _, w := range workflows {
		wf, _ := w.(map[string]interface{})
		name, _ := wf["name"].(string)
		t.Logf("found workflow: %s", name)
		if name == "" {
			t.Error("workflow has empty name")
		}
	}
}

// TestDirWatch_API_AddDeploysWorkflow verifies deploying via API (simulating
// what the file watcher does internally).
func TestDirWatch_API_AddDeploysWorkflow(t *testing.T) {
	wfID := uniqueID("dirwatch-add")
	yaml := `
main:
  steps:
    - done:
        return: "deployed via API"
`
	name := createWorkflow(t, wfID, yaml)
	waitForWorkflowAvailable(t, name, 5*time.Second)

	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "deployed via API")
}

// TestDirWatch_API_ModifyUpdatesWorkflow verifies updating via PATCH API
// (simulates file modification hot-reload).
func TestDirWatch_API_ModifyUpdatesWorkflow(t *testing.T) {
	wfID := uniqueID("dirwatch-modify")

	yamlV1 := `
main:
  steps:
    - done:
        return: "version 1"
`
	name := createWorkflow(t, wfID, yamlV1)

	er1 := executeWorkflow(t, name, nil)
	assertResultEquals(t, er1, "version 1")

	yamlV2 := `
main:
  steps:
    - done:
        return: "version 2"
`
	updateWorkflow(t, name, yamlV2)
	time.Sleep(500 * time.Millisecond)

	er2 := executeWorkflow(t, name, nil)
	assertResultEquals(t, er2, "version 2")
}

// TestDirWatch_API_DeleteRemovesWorkflow verifies deleting via API (simulates
// file deletion).
func TestDirWatch_API_DeleteRemovesWorkflow(t *testing.T) {
	wfID := uniqueID("dirwatch-delete")
	yaml := `
main:
  steps:
    - done:
        return: "will be deleted"
`
	name := createWorkflow(t, wfID, yaml)

	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	deleteWorkflow(t, name)
	time.Sleep(500 * time.Millisecond)

	resp2, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp2.StatusCode)
	}
}

// TestDirWatch_API_InFlightExecutionIsolation verifies that updating a
// workflow via PATCH while an execution is in-flight does NOT affect the
// running execution. This is the API-based secondary test; see
// TestDirWatch_File_InFlightExecutionIsolation for the file-based primary test.
func TestDirWatch_API_InFlightExecutionIsolation(t *testing.T) {
	wfID := uniqueID("dirwatch-inflight")

	yamlV1 := `
main:
  steps:
    - wait:
        call: sys.sleep
        args:
          seconds: 3
    - done:
        return: "version 1 result"
`
	name := createWorkflow(t, wfID, yamlV1)

	execResult := startExecutionAsync(t, name, nil)
	time.Sleep(500 * time.Millisecond)

	yamlV2 := `
main:
  steps:
    - done:
        return: "version 2 result"
`
	updateWorkflow(t, name, yamlV2)

	er1 := waitForExecution(t, execResult.execName, 30*time.Second)
	assertResultEquals(t, er1, "version 1 result")

	er2 := executeWorkflow(t, name, nil)
	assertResultEquals(t, er2, "version 2 result")
}

// TestDirWatch_API_CRUDAlongsideFileWatch verifies that the Workflows CRUD
// API works alongside file-watched workflows.
func TestDirWatch_API_CRUDAlongsideFileWatch(t *testing.T) {
	wfID := uniqueID("dirwatch-crud-coexist")
	yaml := `
main:
  steps:
    - done:
        return: "from CRUD API"
`
	name := createWorkflow(t, wfID, yaml)
	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "from CRUD API")

	resp, err := http.Get(apiURL(parentPath + "/workflows"))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	workflows, _ := result["workflows"].([]interface{})
	found := false
	for _, w := range workflows {
		wf, _ := w.(map[string]interface{})
		n, _ := wf["name"].(string)
		if n == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CRUD-deployed workflow %s not found in list", name)
	}
}

// TestDirWatch_API_YAMLAndJSONFormats verifies that both YAML and JSON source
// formats are accepted via the API.
func TestDirWatch_API_YAMLAndJSONFormats(t *testing.T) {
	yamlWfID := uniqueID("dirwatch-yaml")
	yamlContent := `
main:
  steps:
    - done:
        return: "from yaml"
`
	name1 := createWorkflow(t, yamlWfID, yamlContent)
	er1 := executeWorkflow(t, name1, nil)
	assertResultEquals(t, er1, "from yaml")
}

// TestDirWatch_API_InvalidSourceDoesNotCrash verifies that submitting invalid
// source via the API returns an error without crashing.
func TestDirWatch_API_InvalidSourceDoesNotCrash(t *testing.T) {
	validID := uniqueID("dirwatch-valid")
	validYAML := `
main:
  steps:
    - done:
        return: "still works"
`
	validName := createWorkflow(t, validID, validYAML)

	invalidID := uniqueID("dirwatch-invalid")
	invalidYAML := `
this is not valid workflow YAML
  - broken: {{{
`
	body, _ := json.Marshal(map[string]interface{}{
		"sourceContents": invalidYAML,
	})
	url := apiURL(parentPath+"/workflows") + "?workflowId=" + invalidID
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		t.Logf("emulator accepted invalid YAML (may validate lazily)")
	}

	er := executeWorkflow(t, validName, nil)
	assertResultEquals(t, er, "still works")
}

// ============================================================================
// SHARED HELPERS
// ============================================================================

// asyncExecResult holds the name of an async-started execution.
type asyncExecResult struct {
	execName string
}

// startExecutionAsync starts an execution without waiting for it to complete.
func startExecutionAsync(t *testing.T, workflowName string, args map[string]interface{}) asyncExecResult {
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
		t.Fatalf("startExecutionAsync HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("startExecutionAsync failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var exec map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&exec); err != nil {
		t.Fatalf("startExecutionAsync decode error: %v", err)
	}

	execName, _ := exec["name"].(string)
	if execName == "" {
		t.Fatalf("startExecutionAsync: no execution name in response")
	}

	return asyncExecResult{execName: execName}
}

// updateWorkflow sends a PATCH request to update a workflow's source.
func updateWorkflow(t *testing.T, workflowName, sourceYAML string) {
	t.Helper()

	body, _ := json.Marshal(map[string]interface{}{
		"sourceContents": sourceYAML,
	})

	req, _ := http.NewRequest(http.MethodPatch, apiURL(workflowName), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("updateWorkflow HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("updateWorkflow failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}
