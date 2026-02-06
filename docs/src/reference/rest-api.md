# REST API Reference

All endpoints are prefixed with `/v1/projects/{project}/locations/{location}`.

Default project is `my-project` and default location is `us-central1`. These are configurable via the `PROJECT` and `LOCATION` environment variables. See [CLI & Configuration](../guide/configuration.md).

## Workflows API

### Create Workflow

```
POST /v1/projects/{project}/locations/{location}/workflows?workflowId={id}
```

**Request body:**

```json
{
  "sourceContents": "main:\n  steps:\n    - done:\n        return: 42\n",
  "description": "Optional description"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sourceContents` | string | Yes | YAML or JSON workflow definition (max 128 KB) |
| `description` | string | No | Human-readable description (max 1000 chars) |

**Response:** The workflow resource (see below).

**Errors:**
- 400 if `workflowId` is missing, `sourceContents` is empty, or the workflow definition is invalid
- 409 if a workflow with the same ID already exists

Note: The real GCW API returns a long-running Operation. The emulator completes immediately and returns the workflow directly.

### Get Workflow

```
GET /v1/projects/{project}/locations/{location}/workflows/{workflowId}
```

**Response:**

```json
{
  "name": "projects/my-project/locations/us-central1/workflows/my-wf",
  "state": "ACTIVE",
  "revisionId": "000001-abc",
  "sourceContents": "main:\n  steps:\n    ...",
  "description": "",
  "createTime": "2026-01-15T10:00:00Z",
  "updateTime": "2026-01-15T10:00:00Z"
}
```

**Errors:** 404 if the workflow does not exist.

### List Workflows

```
GET /v1/projects/{project}/locations/{location}/workflows
```

**Response:**

```json
{
  "workflows": [
    {
      "name": "projects/my-project/locations/us-central1/workflows/my-wf",
      "state": "ACTIVE",
      ...
    }
  ]
}
```

### Update Workflow

```
PATCH /v1/projects/{project}/locations/{location}/workflows/{workflowId}
```

**Request body:** Same fields as Create (provide `sourceContents` and/or `description`).

**Response:** An Operation with `done: true` and the updated workflow in `response`.

**Errors:** 404 if the workflow does not exist.

### Delete Workflow

```
DELETE /v1/projects/{project}/locations/{location}/workflows/{workflowId}
```

**Response:** An Operation with `done: true`.

**Errors:** 404 if the workflow does not exist.

---

## Executions API

### Create Execution

```
POST /v1/projects/{project}/locations/{location}/workflows/{workflowId}/executions
```

**Request body:**

```json
{
  "argument": "{\"key\": \"value\"}"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `argument` | string | No | JSON-encoded string with execution arguments (max 32 KB) |

The `argument` field is a JSON-encoded **string**, not a JSON object. This matches the real GCW API format.

**Response:** The execution resource with `state: "ACTIVE"`.

The execution runs asynchronously. Poll the Get Execution endpoint to check for completion.

**Errors:** 404 if the workflow does not exist.

### Get Execution

```
GET /v1/projects/{project}/locations/{location}/workflows/{workflowId}/executions/{executionId}
```

**Successful response:**

```json
{
  "name": "projects/my-project/locations/us-central1/workflows/my-wf/executions/exec-abc123",
  "state": "SUCCEEDED",
  "result": "\"Hello, World!\"",
  "argument": "{\"name\": \"Alice\"}",
  "startTime": "2026-01-15T10:00:00Z",
  "endTime": "2026-01-15T10:00:01Z",
  "workflowRevisionId": "000001-abc"
}
```

**Failed response:**

```json
{
  "name": "...",
  "state": "FAILED",
  "error": {
    "payload": "{\"message\":\"division by zero\",\"tags\":[\"ZeroDivisionError\"]}",
    "context": "step: calculate"
  },
  "startTime": "...",
  "endTime": "..."
}
```

The `result` field is a JSON-encoded string. The `error.payload` field is also a JSON-encoded string containing the error map.

**Errors:** 404 if the execution does not exist.

### List Executions

```
GET /v1/projects/{project}/locations/{location}/workflows/{workflowId}/executions
```

**Response:**

```json
{
  "executions": [
    {
      "name": "...",
      "state": "SUCCEEDED",
      ...
    }
  ]
}
```

### Cancel Execution

```
POST /v1/projects/{project}/locations/{location}/workflows/{workflowId}/executions/{executionId}:cancel
```

Cancels an active execution. The execution state changes to `CANCELLED`.

**Errors:**
- 404 if the execution does not exist
- 400 if the execution is not in `ACTIVE` state

### Execution states

| State | Description |
|-------|-------------|
| `ACTIVE` | Currently running |
| `SUCCEEDED` | Completed successfully (check `result` field) |
| `FAILED` | Completed with error (check `error` field) |
| `CANCELLED` | Cancelled via the Cancel API |

State transitions: `ACTIVE` -> `SUCCEEDED`, `FAILED`, or `CANCELLED`.

---

## Callbacks API

### List Callbacks

```
GET /v1/projects/{project}/locations/{location}/workflows/{workflowId}/executions/{executionId}/callbacks
```

**Response:**

```json
{
  "callbacks": [
    {
      "name": "...",
      "method": "POST",
      "url": "http://localhost:8787/callbacks/abc123",
      "createTime": "2026-01-15T10:00:00Z"
    }
  ]
}
```

### Send Callback

```
POST /callbacks/{callbackId}
```

Sends data to a waiting `events.await_callback` step. The request body can be any JSON payload and will be available in the callback result's `http_request.body` field.

---

## gRPC API

The emulator also exposes a gRPC API on port 8788 (configurable via `GRPC_PORT` environment variable). The gRPC API implements the same operations as the REST API using the official Google Cloud Workflows protobuf definitions.

### Workflows service

| RPC | Description |
|-----|-------------|
| `ListWorkflows` | List workflows in a project/location |
| `GetWorkflow` | Get workflow details |
| `CreateWorkflow` | Create a new workflow (returns Operation) |
| `DeleteWorkflow` | Delete a workflow (returns Operation) |
| `UpdateWorkflow` | Update a workflow (returns Operation) |

### Executions service

| RPC | Description |
|-----|-------------|
| `ListExecutions` | List executions for a workflow |
| `CreateExecution` | Start a new execution |
| `GetExecution` | Get execution details |
| `CancelExecution` | Cancel a running execution |

### Connecting via gRPC

```go
import (
    workflowspb "cloud.google.com/go/workflows/apiv1/workflowspb"
    executionspb "cloud.google.com/go/workflows/executions/apiv1/executionspb"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

conn, err := grpc.Dial("localhost:8788", grpc.WithTransportCredentials(insecure.NewCredentials()))
workflowsClient := workflowspb.NewWorkflowsClient(conn)
executionsClient := executionspb.NewExecutionsClient(conn)
```

---

## Emulator simplifications

The emulator differs from the real GCW API in these ways:

| Real GCW | Emulator |
|----------|----------|
| Create/Update/Delete return long-running Operations that must be polled | Returns the result immediately |
| Requires IAM authentication | Accepts all requests without credentials |
| Supports pagination (page_size, page_token) | Returns all results in one response |
| Supports filter and order_by parameters | Not implemented |
