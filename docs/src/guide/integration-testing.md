# Integration Testing

The primary use case for this emulator is integration testing. You define workflows in YAML, run them against your local services, and verify the orchestration works correctly.

## Pattern

1. Start the emulator (as a separate process or in `TestMain`)
2. Deploy a workflow via the REST API
3. Execute it
4. Poll for completion
5. Assert on the result

## Go test example

```go
package myservice_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "os"
    "testing"
    "time"
)

var emulatorURL string

func TestMain(m *testing.M) {
    emulatorURL = os.Getenv("WORKFLOWS_EMULATOR_HOST")
    if emulatorURL == "" {
        emulatorURL = "http://localhost:8787"
    }
    os.Exit(m.Run())
}

func deployWorkflow(t *testing.T, id, source string) {
    t.Helper()
    body, _ := json.Marshal(map[string]string{"sourceContents": source})
    url := emulatorURL + "/v1/projects/my-project/locations/us-central1/workflows?workflowId=" + id
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatal(err)
    }
    resp.Body.Close()
    if resp.StatusCode != 200 {
        t.Fatalf("deploy failed: %d", resp.StatusCode)
    }
}

func runWorkflow(t *testing.T, id string, args map[string]interface{}) map[string]interface{} {
    t.Helper()

    body := map[string]interface{}{}
    if args != nil {
        argsJSON, _ := json.Marshal(args)
        body["argument"] = string(argsJSON)
    }
    data, _ := json.Marshal(body)

    url := emulatorURL + "/v1/projects/my-project/locations/us-central1/workflows/" + id + "/executions"
    resp, err := http.Post(url, "application/json", bytes.NewReader(data))
    if err != nil {
        t.Fatal(err)
    }
    var exec map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&exec)
    resp.Body.Close()

    // Poll for completion
    name := exec["name"].(string)
    getURL := emulatorURL + "/v1/" + name
    for i := 0; i < 100; i++ {
        resp, _ = http.Get(getURL)
        json.NewDecoder(resp.Body).Decode(&exec)
        resp.Body.Close()
        state := exec["state"].(string)
        if state == "SUCCEEDED" || state == "FAILED" || state == "CANCELLED" {
            return exec
        }
        time.Sleep(100 * time.Millisecond)
    }
    t.Fatal("execution timed out")
    return nil
}

func TestOrderWorkflow(t *testing.T) {
    deployWorkflow(t, "order-flow", `
main:
  params: [args]
  steps:
    - validate:
        call: http.post
        args:
          url: http://localhost:9090/api/validate
          body:
            order_id: ${args.order_id}
        result: resp
    - done:
        return: ${resp.body}
`)

    result := runWorkflow(t, "order-flow", map[string]interface{}{
        "order_id": "ORD-123",
    })

    if result["state"] != "SUCCEEDED" {
        t.Fatalf("expected SUCCEEDED, got %s: %v", result["state"], result["error"])
    }
}
```

## Running tests

Start the emulator in one terminal:

```bash
go run ./cmd/gcw-emulator
```

Run your tests in another:

```bash
WORKFLOWS_EMULATOR_HOST=http://localhost:8787 go test -v ./...
```

## Tips

- Use unique workflow IDs per test (e.g., append a timestamp) to avoid conflicts when running tests in parallel
- The emulator runs executions asynchronously -- always poll `GET .../executions/{id}` until you see a terminal state
- For tests that expect failures, check `exec["state"] == "FAILED"` and inspect `exec["error"]`
- The `argument` field in create-execution is a JSON **string**, not a JSON object. Marshal your args to JSON and pass as a string.
