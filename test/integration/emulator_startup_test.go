package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// Tests for the emulator startup and configuration behavior.
// These verify the Firebase-emulator-style usage model:
//   gcw-emulator --workflows-dir=./workflows

// TestStartup_WatchedDirWorkflowsAvailable verifies that workflows loaded from
// the watched directory (--workflows-dir flag) are immediately available for
// execution without needing to deploy via the Workflows CRUD API.
func TestStartup_WatchedDirWorkflowsAvailable(t *testing.T) {
	// Check if any dir-loaded workflows exist by listing.
	resp, err := http.Get(apiURL(parentPath + "/workflows"))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	workflows, _ := result["workflows"].([]interface{})
	if len(workflows) == 0 {
		t.Skip("no dir-loaded workflows found; emulator may not have --workflows-dir")
		return
	}

	// Get the first workflow name and try to execute it
	wf, _ := workflows[0].(map[string]interface{})
	wfName, _ := wf["name"].(string)
	if wfName == "" {
		t.Skip("could not determine workflow name")
		return
	}

	body, _ := json.Marshal(map[string]interface{}{})
	execResp, err := http.Post(apiURL(wfName+"/executions"), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer execResp.Body.Close()

	if execResp.StatusCode != http.StatusOK && execResp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(execResp.Body)
		t.Fatalf("expected 200/201, got %d: %s", execResp.StatusCode, string(respBody))
	}

	t.Logf("Successfully triggered execution of dir-loaded workflow: %s", wfName)
}

// TestStartup_CRUDApiWorksAlongsideDirWatch verifies that the CRUD API
// works alongside directory-watched workflows. Both sources coexist.
func TestStartup_CRUDApiWorksAlongsideDirWatch(t *testing.T) {
	wfID := uniqueID("startup-crud")
	yaml := `
main:
  steps:
    - done:
        return: "deployed via API"
`
	name := createWorkflow(t, wfID, yaml)
	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "deployed via API")
}

// TestStartup_APIPathsMatchGCP verifies that the emulator's REST API paths
// match the real GCP Workflows API format.
func TestStartup_APIPathsMatchGCP(t *testing.T) {
	wfID := uniqueID("startup-path")
	yaml := `
main:
  steps:
    - done:
        return: "path test"
`
	name := createWorkflow(t, wfID, yaml)

	// Verify the name follows GCP format:
	// projects/{project}/locations/{location}/workflows/{workflowId}
	expectedPrefix := "projects/" + defaultProject + "/locations/" + defaultLocation + "/workflows/"
	if len(name) <= len(expectedPrefix) || name[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("workflow name %q does not match expected GCP format prefix %q", name, expectedPrefix)
	}

	// Verify we can GET the workflow using the full GCP-style path
	resp, err := http.Get(apiURL(name))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 for GCP-style path, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestStartup_ExecutionPathsMatchGCP verifies that execution API paths
// follow the GCP Executions API format.
func TestStartup_ExecutionPathsMatchGCP(t *testing.T) {
	wfID := uniqueID("startup-exec-path")
	yaml := `
main:
  steps:
    - done:
        return: "exec path test"
`
	name := createWorkflow(t, wfID, yaml)

	// Create execution and check the name format
	body, _ := json.Marshal(map[string]interface{}{})
	resp, err := http.Post(apiURL(name+"/executions"), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var exec map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exec)

	execName, _ := exec["name"].(string)
	if execName == "" {
		t.Fatal("execution name is empty")
	}

	// Execution name should follow: {workflow_name}/executions/{execution_id}
	expectedBase := name + "/executions/"
	if len(execName) <= len(expectedBase) || execName[:len(expectedBase)] != expectedBase {
		t.Errorf("execution name %q does not match expected format %s{id}", execName, expectedBase)
	}
}

// TestStartup_EmulatorHostEnvVar verifies that the WORKFLOWS_EMULATOR_HOST
// env var convention is respected. This test validates that our test harness
// itself reads the env var correctly.
func TestStartup_EmulatorHostEnvVar(t *testing.T) {
	// The init() function in helpers_test.go reads WORKFLOWS_EMULATOR_HOST.
	// Just verify the test server URL is set and reachable.
	if testServer == "" {
		t.Fatal("testServer URL is empty")
	}

	// Verify the emulator is reachable at the configured address
	resp, err := http.Get(apiURL(parentPath + "/workflows"))
	if err != nil {
		t.Fatalf("emulator not reachable at %s: %v", testServer, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Logf("emulator responded with status %d (may have no workflows yet)", resp.StatusCode)
	}
}

// TestStartup_APIOnlyModeWorksWithoutWorkflowsDir verifies that when
// --workflows-dir is NOT provided, the emulator still starts and accepts
// workflow deployments via the CRUD API. This is "API-only mode".
func TestStartup_APIOnlyModeWorksWithoutWorkflowsDir(t *testing.T) {
	// Deploy a workflow via CRUD API -- this should always work regardless
	// of whether --workflows-dir is configured.
	wfID := uniqueID("startup-apionly")
	yaml := `
main:
  steps:
    - done:
        return: "API-only mode works"
`
	name := createWorkflow(t, wfID, yaml)
	er := executeWorkflow(t, name, nil)
	assertResultEquals(t, er, "API-only mode works")
}

// TestStartup_CLIFlags verifies the expected CLI flags are documented.
// This is a documentation test -- it verifies the emulator binary supports
// expected flags: --port, --project, --location, --workflows-dir.
func TestStartup_CLIFlags(t *testing.T) {
	// This test is more of a behavioral contract check.
	// The emulator should respond on the configured port with expected
	// project/location in the API paths.
	wfID := uniqueID("startup-flags")
	yaml := `
main:
  steps:
    - get_project:
        call: sys.get_env
        args:
          name: "GOOGLE_CLOUD_PROJECT_ID"
        result: project
    - get_location:
        call: sys.get_env
        args:
          name: "GOOGLE_CLOUD_LOCATION"
        result: location
    - done:
        return:
          project: ${project}
          location: ${location}
`
	name := createWorkflow(t, wfID, yaml)
	er := executeWorkflow(t, name, nil)
	assertSucceeded(t, er)

	// The emulator should expose project and location via sys.get_env
	// These should match the configured --project and --location flags
	resultMap, ok := er.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", er.Result)
	}

	project, _ := resultMap["project"].(string)
	location, _ := resultMap["location"].(string)
	if project == "" {
		t.Error("GOOGLE_CLOUD_PROJECT_ID not set in emulator")
	}
	if location == "" {
		t.Error("GOOGLE_CLOUD_LOCATION not set in emulator")
	}
	t.Logf("emulator project=%s location=%s", project, location)
}
