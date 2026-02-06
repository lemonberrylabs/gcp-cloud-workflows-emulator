# Error Handling

## Error structure

All errors in Google Cloud Workflows are represented as maps:

```json
{
  "message": "Human-readable error description",
  "code": 404,
  "tags": ["HttpError"]
}
```

HTTP errors include additional fields:

```json
{
  "message": "HTTP request failed with status 404",
  "code": 404,
  "tags": ["HttpError"],
  "headers": {"content-type": "application/json"},
  "body": {"error": "not found"}
}
```

## try / except

Catch errors and handle them:

```yaml
- safe_call:
    try:
      steps:
        - fetch:
            call: http.get
            args:
              url: http://localhost:9090/data
            result: response
    except:
      as: e
      steps:
        - check_error:
            switch:
              - condition: ${"ConnectionFailedError" in e.tags}
                return: "Service is not running"
              - condition: ${"HttpError" in e.tags and e.code == 404}
                return: null
              - condition: true
                raise: ${e}
```

The error variable (`e` in the `as` field) is a map with `message`, `code`, and `tags` fields.

## try / retry / except

Add retry with exponential backoff:

```yaml
- resilient_call:
    try:
      steps:
        - fetch:
            call: http.get
            args:
              url: http://localhost:9090/data
            result: response
    retry:
      predicate: ${http.default_retry}
      max_retries: 5
      backoff:
        initial_delay: 1
        max_delay: 60
        multiplier: 2
    except:
      as: e
      steps:
        - handle:
            return: '${"Failed after retries: " + e.message}'
```

**Key behavior:** Retries re-execute the **entire try block** from the beginning, not just the failed step.

**Retry count:** `max_retries: 3` means 1 initial attempt + 3 retries = 4 total attempts.

**Backoff formula:** `delay = min(initial_delay * multiplier^attempt, max_delay)`

## Error tags

The emulator supports all 17 Google Cloud Workflows error tags:

| Tag | When raised | Typical code |
|-----|-------------|--------------|
| `AuthError` | Generating credentials fails | 0 |
| `ConnectionError` | Connection broke mid-transfer | 0 |
| `ConnectionFailedError` | Connection never established (service down, DNS failure) | 0 |
| `HttpError` | Non-2xx HTTP response | HTTP status code |
| `IndexError` | List index out of range | 0 |
| `KeyError` | Map key not found, or unknown env var in `sys.get_env` | 0 |
| `OperationError` | Long-running operation failure | 0 |
| `ParallelNestingError` | Parallel nesting exceeds depth 2 | 0 |
| `RecursionError` | Call stack depth exceeds 20 | 0 |
| `ResourceLimitError` | Memory, step count, or other resource limits exceeded | 0 |
| `ResponseTypeError` | Unexpected response type from operation | 0 |
| `SystemError` | Internal system error | 0 |
| `TimeoutError` | HTTP request or callback await timed out | 0 |
| `TypeError` | Type mismatch (e.g., `"hi" + 5`, `not "string"`) | 0 |
| `UnhandledBranchError` | Raised after `continueAll` parallel when branches had errors | 0 |
| `ValueError` | Correct type but invalid value (e.g., `int("abc")`) | 0 |
| `ZeroDivisionError` | Division or modulo by zero | 0 |

Errors can have multiple tags. For example, an HTTP 404 error has `tags: ["HttpError"]`.

### ConnectionFailedError vs ConnectionError

This distinction is critical for local development:

| Error | Meaning | Common cause |
|-------|---------|--------------|
| **ConnectionFailedError** | Connection was never established | Service not running, port not listening, DNS failure |
| **ConnectionError** | Connection established but broke during transfer | Service crashed mid-response |

When your local service is not running, the emulator raises `ConnectionFailedError`. This is the error you'll see most often during development.

## Catching errors by tag

```yaml
- step:
    try:
      call: http.get
      args:
        url: http://localhost:9090/api
      result: response
    except:
      as: e
      steps:
        - route_error:
            switch:
              - condition: ${"ConnectionFailedError" in e.tags}
                return: "Service is down"
              - condition: ${"HttpError" in e.tags and e.code == 429}
                next: retry_later
              - condition: ${"HttpError" in e.tags and e.code >= 500}
                return: "Server error"
              - condition: ${"TimeoutError" in e.tags}
                return: "Request timed out"
              - condition: true
                raise: ${e}
```

## raise

Throw a custom error:

```yaml
# String error -> {message: "...", code: 0, tags: []}
- fail:
    raise: "validation failed"

# Map error with custom fields
- fail:
    raise:
      code: 400
      message: "Invalid order ID"
      tags: ["ValidationError"]

# Re-raise a caught error
- rethrow:
    raise: ${e}
```

## Error propagation

1. Error occurs in a step
2. If inside a `try` block with `retry`: retry is attempted first
3. If retry exhausted or not configured: `except` block executes (if present)
4. If no `except` or `except` re-raises: error propagates up
5. In a subworkflow: propagates to the caller
6. In a parallel branch: depends on exception policy (`unhandled` aborts all; `continueAll` collects)
7. At the top level of `main`: execution fails with state `FAILED`

## Variable scoping with try/except

Variables declared inside `except` are not visible outside:

```yaml
# WRONG: error_msg is not accessible after the try/except block
- handle:
    try:
      call: http.get
      args:
        url: http://localhost:9090/data
      result: response
    except:
      as: e
      steps:
        - save:
            assign:
              - error_msg: ${e.message}   # Only in scope inside except

# CORRECT: declare the variable before the try/except
- init:
    assign:
      - error_msg: null
- handle:
    try:
      call: http.get
      args:
        url: http://localhost:9090/data
      result: response
    except:
      as: e
      steps:
        - save:
            assign:
              - error_msg: ${e.message}   # Modifies parent-scope variable
- use:
    return: ${error_msg}                  # Works
```

## Built-in retry policies

| Policy | Retries on | Does NOT retry |
|--------|-----------|----------------|
| `http.default_retry` | 429, 502, 503, 504, ConnectionError, TimeoutError | **500** |
| `http.default_retry_non_idempotent` | Same as above | **500** |
| `retry.always` | Everything | (nothing) |
| `retry.never` | (nothing) | Everything |

`retry.default_backoff`: initial_delay 1s, max_delay 60s, multiplier 1.25.

See the [Standard Library > Custom retry predicates](./stdlib.md#custom-retry-predicates) section for writing predicates that retry on HTTP 500.
