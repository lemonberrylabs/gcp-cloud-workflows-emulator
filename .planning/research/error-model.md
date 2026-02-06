# Error Model - Complete Reference

## Error Structure

All errors in Google Cloud Workflows are represented as maps with this structure:

```json
{
  "message": "Human-readable error description",
  "code": 404,
  "tags": ["HttpError", "NotFound"]
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `message` | string | Human-readable error description |
| `code` | int | Error code (often HTTP status code, or custom code) |
| `tags` | list of strings | Error type classification tags |

HTTP errors may include additional fields:
| Field | Type | Description |
|-------|------|-------------|
| `headers` | map | Response headers |
| `body` | any | Response body (parsed if JSON) |

## Error Types (Tags) — OFFICIAL COMPLETE LIST

Source: Official GCW documentation page "Workflow errors" (verified).

Errors can have **multiple tags**. The primary tag identifies the error category, and secondary tags provide specificity.

### AuthError
- **Tags**: `["AuthError"]`
- **When**: Generating credentials for an HTTP request fails
- **Note**: This is the OFFICIAL tag name (not "AuthenticationError")

### ConnectionError
- **Tags**: `["ConnectionError"]`
- **When**: Connection is successfully established with the endpoint but there is a problem with the connection during data transfer
- **Key distinction**: Connection WAS established, but broke during transfer
- **Code**: Typically 0 or not set

### ConnectionFailedError
- **Tags**: `["ConnectionFailedError"]`
- **When**: Connection is NOT established with the API endpoint — due to incorrect domain name, DNS resolution issues, or other network problems
- **Key distinction**: Connection NEVER established (e.g., "connection refused" on localhost)
- **Code**: Typically 0 or not set
- **CRITICAL for emulator**: This is what fires when a localhost service is down

### HttpError
- **Tags**: `["HttpError"]` plus additional tags based on HTTP status code
- **When**: HTTP request fails with an HTTP error status (non-2xx)
- **Fields**: `code` (HTTP status), `message`, `headers`, `body`
- **Additional tags by status code** (follows Google gRPC-to-HTTP mapping):
  - 400: `["HttpError"]` (Bad Request / INVALID_ARGUMENT / FAILED_PRECONDITION / OUT_OF_RANGE)
  - 401: `["HttpError"]` (Unauthorized / UNAUTHENTICATED)
  - 403: `["HttpError"]` (Forbidden / PERMISSION_DENIED)
  - 404: `["HttpError"]` (Not Found / NOT_FOUND)
  - 409: `["HttpError"]` (Conflict / ALREADY_EXISTS / ABORTED)
  - 429: `["HttpError"]` (Too Many Requests / RESOURCE_EXHAUSTED)
  - 499: `["HttpError"]` (Client Closed Request / CANCELLED)
  - 500: `["HttpError"]` (Internal Server Error / UNKNOWN / INTERNAL / DATA_LOSS)
  - 501: `["HttpError"]` (Not Implemented / UNIMPLEMENTED)
  - 502: `["HttpError"]` (Bad Gateway)
  - 503: `["HttpError"]` (Service Unavailable / UNAVAILABLE)
  - 504: `["HttpError"]` (Gateway Timeout / DEADLINE_EXCEEDED)

### IndexError
- **Tags**: `["IndexError"]`
- **When**: A sequence subscript is an out-of-range integer
- **Examples**:
  - `${myList[99]}` when list has fewer elements
  - `${myList[-1]}` (negative indices not supported)

### KeyError
- **Tags**: `["KeyError"]`
- **When**: A map key is not found in the set of existing keys
- **Examples**:
  - `${myMap.nonExistentKey}`
  - `${myMap["missing"]}`
  - `sys.get_env("UNKNOWN_VAR")`

### OperationError
- **Tags**: `["OperationError"]`
- **When**: A long-running operation finishes unsuccessfully

### ParallelNestingError
- **Tags**: `["ParallelNestingError"]`
- **When**: Maximum depth that parallel steps can be nested is exceeded (limit: 2)

### RecursionError
- **Tags**: `["RecursionError"]`
- **When**: The interpreter detects that the maximum call stack depth is exceeded (limit: 20)

### ResourceLimitError
- **Tags**: `["ResourceLimitError"]`
- **When**: Some resource limit is exhausted
- **Subtypes** (may have additional tags):
  - Step limit exceeded (100,000 steps per execution)
  - Memory limit exceeded
  - Result size limit exceeded

### ResponseTypeError
- **Tags**: `["ResponseTypeError"]`
- **When**: A long-running operation returns a response of the wrong type

### SystemError
- **Tags**: `["SystemError"]`
- **When**: The interpreter finds an internal error

### TimeoutError
- **Tags**: `["TimeoutError"]`
- **When**: A system function times out at the system level
- **Triggers**: HTTP request exceeds timeout (max 1800s), callback await exceeds timeout

### TypeError
- **Tags**: `["TypeError"]`
- **When**: An operation or function is applied to an object of incompatible type
- **Examples**:
  - `"hello" + 5` (string + int without conversion)
  - `not "hello"` (logical not on non-boolean)
  - Passing wrong type to a function

### UnhandledBranchError
- **Tags**: `["UnhandledBranchError"]`
- **When**: One or more branches or iterations encounters an unhandled runtime error
- **Context**: Raised after a parallel step completes when using `continueAll` exception policy

### ValueError
- **Tags**: `["ValueError"]`
- **When**: An operation or function receives an argument that has the correct type but an incorrect value
- **Examples**:
  - `int("abc")` (unparseable string)
  - `json.decode("not json")`
  - Invalid regex pattern

### ZeroDivisionError
- **Tags**: `["ZeroDivisionError"]`
- **When**: The second argument of a division or modulo operation is zero
- **Examples**: `${x / 0}`, `${x // 0}`, `${x % 0}`

## Error Propagation

### Unhandled Errors
1. Error occurs in a step
2. If inside a `try` block: jumps to `except` handler
3. If inside a `for` loop within a `try`: exits loop, goes to `except`
4. If inside a subworkflow: propagates to caller
5. If inside a parallel branch: depends on exception policy
6. If unhandled at top level: execution fails with state `FAILED`

### try/except Behavior
1. Error in `try` block triggers `retry` first (if configured)
2. If retry exhausted or not configured, `except` block executes
3. Error available as variable specified in `as` field
4. If `except` re-raises (`raise: ${e}`), error propagates up
5. If `except` completes normally, execution continues after the try/except step

### Parallel Branch Errors
- **Default policy (no exception_policy set)**: Not well documented; see parallel-execution.md
- **`continueAll` policy**: Other branches continue; unhandled exceptions collected (max 100)
- Errors from parallel branches can be caught by a try/except wrapping the parallel step

## Custom Errors

### Raising String Errors
```yaml
- raise_error:
    raise: "Something went wrong"
```
Creates error: `{message: "Something went wrong", code: 0, tags: []}`

### Raising Map Errors
```yaml
- raise_error:
    raise:
      code: 55
      message: "Custom error"
```
Creates error: `{message: "Custom error", code: 55, tags: []}`

### Raising with Custom Tags
```yaml
- raise_error:
    raise:
      code: 1001
      message: "Validation failed"
      tags: ["ValidationError"]
```

## Checking Error Types

### By tag membership
```yaml
- condition: ${"HttpError" in e.tags}
```

### By code
```yaml
- condition: ${e.code == 404}
```

### Combined
```yaml
- condition: ${("HttpError" in e.tags) and (e.code == 429)}
```

### Non-HTTP error detection
```yaml
- condition: ${not("HttpError" in e.tags)}
  # This is a connection or other non-HTTP error
```

## Custom Retry Predicates

A custom retry predicate is a subworkflow that:
- Accepts a single parameter (the error map)
- Returns `true` to retry or `false` to stop
- Can inspect `e.tags`, `e.code`, `e.message`

```yaml
my_retry_predicate:
  params: [e]
  steps:
    - check:
        switch:
          - condition: ${"ConnectionError" in e.tags}
            return: true
          - condition: ${"TimeoutError" in e.tags}
            return: true
          - condition: ${("HttpError" in e.tags) and (e.code in [429, 500, 502, 503, 504])}
            return: true
    - default:
        return: false
```
