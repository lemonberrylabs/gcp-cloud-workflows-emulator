# Callbacks - Complete Reference

## Overview

Callbacks allow a workflow execution to pause and wait for an external HTTP request before resuming. This enables event-driven patterns without polling.

## Creating a Callback Endpoint

```yaml
- create_callback:
    call: events.create_callback_endpoint
    args:
      http_callback_method: "POST"  # Optional, default "POST"
    result: callback_details
```

### Parameters
- `http_callback_method` (string, optional): HTTP method the endpoint accepts
  - Supported: GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH
  - Default: POST

### Return Value
A map containing:
- `url` (string): The callback URL

### Callback URL Format
```
https://workflowexecutions.googleapis.com/v1/projects/{projectId}/locations/{location}/workflows/{workflowName}/executions/{executionId}/callbacks/{callbackId}
```

## Awaiting a Callback

```yaml
- await:
    call: events.await_callback
    args:
      callback: ${callback_details}
      timeout: 3600  # Optional, default 43200 seconds (12 hours)
    result: callback_request
```

### Parameters
- `callback` (map, required): The callback details map from `create_callback_endpoint`
- `timeout` (double, optional): Maximum seconds to wait. Default: 43200 (12 hours)

### Return Value (callback_request)
```json
{
  "http_request": {
    "body": "<parsed body - JSON becomes map, text stays string, otherwise bytes>",
    "headers": {"header-name": "value"},
    "method": "POST",
    "query": "key=value&key2=value2",
    "url": "/callbacks/{callbackId}"
  },
  "received_time": "2023-01-15T10:30:00Z",
  "type": "HTTP"
}
```

### Body Parsing
- If Content-Type is `application/json`: body is parsed into map/list
- If Content-Type is text: body is string
- Otherwise: body is raw bytes

### Timeout Behavior
- If timeout elapses before a callback is received: **TimeoutError** is raised
- Can be caught by a try/except block

## Sending Data to a Callback

External services call the callback URL:

```bash
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "completed", "data": {"key": "value"}}' \
  "$CALLBACK_URL"
```

### Authentication Requirements
- The caller must have `workflows.callbacks.send` IAM permission
- This is included in the `Workflows Invoker` IAM role
- For the emulator: authentication is not enforced (per project scope)

## Callback Slot Mechanics

### Slot Model
1. `create_callback_endpoint` creates the endpoint but does NOT open a slot
2. `await_callback` opens a slot to receive one callback
3. When a callback request arrives, it fills the slot
4. The slot is emptied when `await_callback` processes the callback

### Multiple Requests to Same Callback
Response codes when receiving callbacks:

| Scenario | HTTP Response |
|----------|--------------|
| First callback received, await not yet processing | **429** Too Many Requests (callback queued) |
| First callback processed, second arrives | **200** OK (second stored but may be discarded) |
| Workflow execution completed/failed | **404** Not Found (callback no longer valid) |

### Key Constraint
**Callback endpoints can only be awaited in the execution where they were created.** Cannot share callbacks across executions.

## Patterns

### Basic Wait for External Event
```yaml
main:
  steps:
    - create_cb:
        call: events.create_callback_endpoint
        args:
          http_callback_method: "POST"
        result: callback_details
    - log_url:
        call: sys.log
        args:
          data: ${callback_details.url}
    - wait_for_event:
        call: events.await_callback
        args:
          callback: ${callback_details}
          timeout: 86400  # 24 hours
        result: event_data
    - process_event:
        assign:
          - result: ${event_data.http_request.body}
    - done:
        return: ${result}
```

### Callback with Timeout Handling
```yaml
- wait_step:
    try:
      call: events.await_callback
      args:
        callback: ${callback_details}
        timeout: 300
      result: callback_data
    except:
      as: e
      steps:
        - check_timeout:
            switch:
              - condition: ${"TimeoutError" in e.tags}
                assign:
                  - callback_data: null
              - condition: true
                raise: ${e}
```

### Parallel Callbacks
```yaml
- parallel_callbacks:
    parallel:
      shared: [results]
      branches:
        - wait_a:
            steps:
              - create_a:
                  call: events.create_callback_endpoint
                  result: cb_a
              - await_a:
                  call: events.await_callback
                  args:
                    callback: ${cb_a}
                  result: result_a
              - store_a:
                  assign:
                    - results: ${map.merge(results, {"a": result_a})}
        - wait_b:
            steps:
              - create_b:
                  call: events.create_callback_endpoint
                  result: cb_b
              - await_b:
                  call: events.await_callback
                  args:
                    callback: ${cb_b}
                  result: result_b
              - store_b:
                  assign:
                    - results: ${map.merge(results, {"b": result_b})}
```

## Edge Cases

1. **Callback created but never awaited**: The endpoint exists but will never be filled. Eventually, the execution may timeout (1 year max).

2. **Multiple awaits on same callback**: After the first await processes, a second await on the same callback will wait for a new callback request.

3. **Callback URL reuse**: Each call to `create_callback_endpoint` generates a unique callback ID.

4. **Redeployment**: Workflows using callbacks that were deployed before January 11, 2022 may need redeployment.

5. **Execution cancellation**: If the execution is cancelled while awaiting a callback, any pending callback requests will receive 404.
