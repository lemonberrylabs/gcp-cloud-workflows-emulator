# REST API - Complete Reference

Source: Protocol Buffer definitions from googleapis/googleapis repository.

## Workflows API

**Host**: `workflows.googleapis.com`
**Base path**: `/v1`

### Service: Workflows

#### ListWorkflows

```
GET /v1/{parent=projects/*/locations/*/workflows}
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `parent` | string | Yes | Format: `projects/{project}/locations/{location}` |
| `page_size` | int | No | Default 500, max 1000 |
| `page_token` | string | No | Token for pagination |
| `filter` | string | No | AIP-160 compatible filter expression |
| `order_by` | string | No | Field to sort by |

**Response**: `ListWorkflowsResponse`
```json
{
  "workflows": [Workflow],
  "nextPageToken": "string",
  "unreachable": ["string"]
}
```

#### GetWorkflow

```
GET /v1/{name=projects/*/locations/*/workflows/*}
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Full resource name |
| `revision_id` | string | No | Specific revision to retrieve |

**Response**: `Workflow`

#### CreateWorkflow

```
POST /v1/{parent=projects/*/locations/*/workflows}?workflowId={workflow_id}
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `parent` | string | Yes | Format: `projects/{project}/locations/{location}` |
| `workflow` | Workflow | Yes | Workflow resource (in request body) |
| `workflow_id` | string | Yes | ID for the workflow. Pattern: `[a-zA-Z][a-zA-Z0-9_-]{0,63}[a-zA-Z0-9]` or single letter |

**Response**: `google.longrunning.Operation` (containing Workflow when complete)

**Workflow ID Rules**:
- 1-64 characters
- Letters, numbers, underscores, hyphens
- Must start with a letter
- Must end with a letter or number

#### UpdateWorkflow

```
PATCH /v1/{workflow.name=projects/*/locations/*/workflows/*}
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workflow` | Workflow | Yes | Updated workflow resource (in request body) |
| `update_mask` | FieldMask | No | Fields to update |

**Response**: `google.longrunning.Operation` (containing Workflow when complete)

#### DeleteWorkflow

```
DELETE /v1/{name=projects/*/locations/*/workflows/*}
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Full resource name |

**Response**: `google.longrunning.Operation`

#### ListWorkflowRevisions

```
GET /v1/{name=projects/*/locations/*/workflows/*}/revisions
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Full resource name |
| `page_size` | int | No | Default 20, max 100 |
| `page_token` | string | No | Pagination token |

**Response**: `ListWorkflowRevisionsResponse`
```json
{
  "workflows": [Workflow],
  "nextPageToken": "string"
}
```
Workflows returned in reverse chronological order.

### Workflow Resource

```json
{
  "name": "projects/{project}/locations/{location}/workflows/{workflow}",
  "description": "string (max 1000 Unicode chars)",
  "state": "ACTIVE|UNAVAILABLE|STATE_UNSPECIFIED",
  "revisionId": "string (output only)",
  "createTime": "timestamp (RFC 3339, output only)",
  "updateTime": "timestamp (RFC 3339, output only)",
  "revisionCreateTime": "timestamp (RFC 3339, output only)",
  "labels": {"key": "value"},
  "serviceAccount": "string",
  "sourceContents": "string (max 128 KB)",
  "cryptoKeyName": "string (optional)",
  "stateError": {
    "details": "string",
    "type": "TYPE_UNSPECIFIED|KMS_ERROR"
  },
  "callLogLevel": "CALL_LOG_LEVEL_UNSPECIFIED|LOG_ALL_CALLS|LOG_ERRORS_ONLY|LOG_NONE",
  "userEnvVars": {"key": "value (max 20 entries, 4 KiB each)"},
  "executionHistoryLevel": "EXECUTION_HISTORY_LEVEL_UNSPECIFIED|EXECUTION_HISTORY_BASIC|EXECUTION_HISTORY_DETAILED"
}
```

### Workflow.State Enum
| Value | Number | Description |
|-------|--------|-------------|
| STATE_UNSPECIFIED | 0 | Invalid state |
| ACTIVE | 1 | Workflow is deployed and ready |
| UNAVAILABLE | 2 | Workflow has issues (e.g., KMS key problems) |

### OperationMetadata

Long-running operations return this metadata:
```json
{
  "createTime": "timestamp",
  "endTime": "timestamp",
  "target": "string (resource name)",
  "verb": "string (create/delete/update)",
  "apiVersion": "string"
}
```

---

## Workflow Executions API

**Host**: `workflowexecutions.googleapis.com`
**Base path**: `/v1`

### Service: Executions

#### CreateExecution

```
POST /v1/{parent=projects/*/locations/*/workflows/*}/executions
```

**Request Body**: `Execution` object

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `parent` | string | Yes | Workflow resource name |
| `execution` | Execution | Yes | Execution resource (in request body) |

**Request Body Fields** (input):
```json
{
  "argument": "{\"key\": \"value\"} (JSON string, max 32 KB)",
  "callLogLevel": "CALL_LOG_LEVEL_UNSPECIFIED|LOG_ALL_CALLS|LOG_ERRORS_ONLY|LOG_NONE",
  "labels": {"key": "value"}
}
```

**Response**: `Execution` (with state ACTIVE or QUEUED)

#### GetExecution

```
GET /v1/{name=projects/*/locations/*/workflows/*/executions/*}
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Full execution resource name |
| `view` | ExecutionView | No | BASIC (default) or FULL |

**Response**: `Execution`

**ExecutionView**:
- `BASIC` (1): Includes most fields, may exclude large fields like result/argument
- `FULL` (2): Includes all fields

#### ListExecutions

```
GET /v1/{parent=projects/*/locations/*/workflows/*}/executions
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `parent` | string | Yes | Workflow resource name |
| `page_size` | int | No | Page size |
| `page_token` | string | No | Pagination token |
| `view` | ExecutionView | No | BASIC or FULL |
| `filter` | string | No | Filter expression |
| `order_by` | string | No | Sort order |

**Response**: `ListExecutionsResponse`
```json
{
  "executions": [Execution],
  "nextPageToken": "string"
}
```

#### CancelExecution

```
POST /v1/{name=projects/*/locations/*/workflows/*/executions/*}:cancel
```

**Request Parameters**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Full execution resource name |

**Response**: `Execution` (with state CANCELLED)

### Execution Resource

```json
{
  "name": "projects/{project}/locations/{location}/workflows/{workflow}/executions/{execution}",
  "startTime": "timestamp (RFC 3339, output only)",
  "endTime": "timestamp (RFC 3339, output only)",
  "duration": "duration string (output only)",
  "state": "STATE_UNSPECIFIED|ACTIVE|SUCCEEDED|FAILED|CANCELLED|UNAVAILABLE|QUEUED",
  "argument": "string (JSON, max 32 KB)",
  "result": "string (JSON, output only, only when state=SUCCEEDED)",
  "error": {
    "payload": "string (JSON-encoded error)",
    "context": "string",
    "stackTrace": {
      "elements": [{
        "step": "string",
        "routine": "string",
        "position": {
          "line": "int64",
          "column": "int64",
          "length": "int64"
        }
      }]
    }
  },
  "workflowRevisionId": "string (output only)",
  "callLogLevel": "CALL_LOG_LEVEL_UNSPECIFIED|LOG_ALL_CALLS|LOG_ERRORS_ONLY|LOG_NONE",
  "status": {
    "currentSteps": [{
      "routine": "string",
      "step": "string"
    }]
  },
  "labels": {"key": "value (max 64 entries, keys/values max 63 chars)"},
  "stateError": {
    "details": "string",
    "type": "TYPE_UNSPECIFIED|KMS_ERROR"
  }
}
```

### Execution.State Enum
| Value | Number | Description |
|-------|--------|-------------|
| STATE_UNSPECIFIED | 0 | Invalid state |
| ACTIVE | 1 | Execution is running |
| SUCCEEDED | 2 | Completed successfully; result field populated |
| FAILED | 3 | Failed with error; error field populated |
| CANCELLED | 4 | Cancelled by user; error field may be populated |
| UNAVAILABLE | 5 | Execution data unavailable (e.g., KMS issue) |
| QUEUED | 6 | Waiting for concurrency quota (backlogged) |

### State Transitions
```
QUEUED -> ACTIVE -> SUCCEEDED
                 -> FAILED
                 -> CANCELLED
ACTIVE -> CANCELLED (via cancel request)
```

---

## Callbacks (within Executions API)

### List Callbacks

```
GET /v1/{parent=projects/*/locations/*/workflows/*/executions/*}/callbacks
```

**Response**: List of callback objects

### Send Callback

```
POST {callback_url}
```

The callback URL is the full URL returned by `events.create_callback_endpoint()`.

**Request Body**: Any JSON payload

**Authentication**: Requires `workflows.callbacks.send` IAM permission (Workflows Invoker role)

**Response Codes**:
- 200: Callback received and processed
- 429: Callback slot occupied (first callback not yet processed)
- 404: Workflow execution completed/failed (callback no longer valid)

---

## Workflow Revisions

### Revision ID Format
The revision ID is auto-generated and follows this format:
- Two parts separated by a hyphen
- First part: zero-padded incrementing number starting at 1 (e.g., `000001`)
- Second part: random 3-character alphanumeric string
- Examples: `000001-27f` (initial), `000002-d52` (first update)

### Revision Creation
- A new revision is created automatically on every successful workflow update (PATCH)
- The `revisionId` field is output-only (server-generated)
- The `revisionCreateTime` records when each revision was created

### Execution Pinning to Revisions (CRITICAL for directory-watching emulator)
**Official GCW behavior**: "Updating a workflow does not affect in-progress executions. Only future executions of the workflow will use the updated configuration."

This means:
1. When an execution starts, it is pinned to the **current workflow revision** at that moment
2. The execution's `workflowRevisionId` field records which revision it uses
3. If the workflow is updated while the execution is running, the execution **continues using the OLD revision**
4. New executions created after the update use the **new revision**
5. `sys.get_env("GOOGLE_CLOUD_WORKFLOW_REVISION_ID")` returns the pinned revision ID

### Listing Revisions
```
GET /v1/{name=projects/*/locations/*/workflows/*}/revisions
```
Returns revisions in reverse chronological order (newest first).
Default page size: 20, max: 100.

### Getting a Specific Revision
```
GET /v1/{name=projects/*/locations/*/workflows/*}?revisionId=REVISION_ID
```
The `revision_id` query parameter on GetWorkflow retrieves a specific revision.

### Emulator Implementation for Directory Watching
When the emulator watches a directory and hot-reloads workflow files:
1. Each file change creates a new revision (increment the counter, generate a random suffix)
2. In-flight executions MUST continue using their pinned (old) workflow definition
3. New executions use the latest revision
4. The emulator must store multiple versions of a workflow definition simultaneously
5. Old revisions can be garbage-collected once no executions reference them

---

## Emulator Simplifications

Per PROJECT.md out-of-scope:
- Long-running Operations for workflow CRUD: Create/Update/Delete should return immediately
- IAM/Authentication: Accept all requests without auth checks
- Operation polling: Not needed since operations complete immediately
