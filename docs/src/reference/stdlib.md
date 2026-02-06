# Standard Library

The emulator supports all Google Cloud Workflows standard library functions.

## Built-in expression helpers

These functions are called directly in expressions without a module prefix.

| Function | Description | Example |
|----------|-------------|---------|
| `default(value, fallback)` | Returns `value` if not null, otherwise `fallback` | `${default(x, 0)}` |
| `keys(map)` | List of map keys (strings) | `${keys(my_map)}` |
| `len(value)` | Length of string, list, or map | `${len(items)}` |
| `type(value)` | Type name as string | `${type(x)}` returns `"int"`, `"string"`, etc. |
| `int(value)` | Convert to integer | `${int("42")}`, `${int(2.7)}` -> `2` |
| `double(value)` | Convert to double | `${double("3.14")}`, `${double(42)}` |
| `string(value)` | Convert to string | `${string(42)}` -> `"42"` |
| `bool(value)` | Convert string to boolean | `${bool("true")}` -> `true` |

**Notes:**
- `default()` only handles null. It does not catch KeyError. Combine with `map.get()` for safe map access: `${default(map.get(m, "key"), "fallback")}`.
- `int()` from double truncates toward zero: `int(-2.7)` = `-2`.
- `string()` does not work on maps, lists, or null. Use `json.encode_to_string()` for those.
- `keys()` does not guarantee key order.

---

## http

HTTP client functions. All HTTP call steps make **real HTTP requests** to the target URL.

### Methods

```yaml
- step:
    call: http.get    # or http.post, http.put, http.patch, http.delete
    args:
      url: http://localhost:9090/api/data
      headers:
        Authorization: "Bearer ${token}"
        Content-Type: "application/json"
      body:
        key: "value"
      query:
        limit: "10"
        offset: "0"
      timeout: 30
    result: response
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `url` | string | Yes | Target URL (HTTP or HTTPS) |
| `headers` | map | No | Request headers |
| `body` | any | No | Request body (auto-serialized to JSON if no Content-Type) |
| `query` | map | No | URL query parameters (URL-encoded automatically) |
| `auth` | map | No | Auth config (accepted but not enforced by emulator) |
| `timeout` | int | No | Timeout in seconds (max 1800, default 1800) |

### http.request

Generic HTTP call with explicit method:

```yaml
- step:
    call: http.request
    args:
      method: "PUT"
      url: http://localhost:9090/api/resource
      body:
        key: "value"
    result: response
```

### Response structure

```yaml
response.body     # Parsed body (JSON auto-parsed to map/list; text stays as string)
response.code     # HTTP status code (integer)
response.headers  # Response headers (map, keys are lowercased)
```

### Auto-behaviors

- **Request body**: If no Content-Type header is set and body is not bytes, the body is JSON-encoded and Content-Type is set to `application/json; charset=utf-8`.
- **Response parsing**: If the response Content-Type is `application/json`, the body is automatically parsed from JSON to a map/list. Text content types return a string. Everything else returns bytes.
- **Response headers**: Header names are lowercased.
- **Non-2xx responses**: Raise an error with tag `HttpError` containing the status code, response body, and headers.

### Error behavior

| Scenario | Error tag |
|----------|-----------|
| Target service not running (connection refused) | `ConnectionFailedError` |
| Connection broke mid-transfer | `ConnectionError` |
| Request exceeded timeout | `TimeoutError` |
| Non-2xx HTTP response | `HttpError` |

### Retry policies

| Policy | Retries on | Does NOT retry |
|--------|-----------|----------------|
| `http.default_retry` | 429, 502, 503, 504, ConnectionError, TimeoutError | **500** |
| `http.default_retry_non_idempotent` | Same as above | **500** |

Both use `retry.default_backoff` (initial 1s, max 60s, multiplier 1.25) with max_retries 5.

**Important:** `http.default_retry` does **not** retry HTTP 500 errors. This surprises many users. If you need to retry 500s, write a [custom retry predicate](#custom-retry-predicates).

---

## sys

### sys.get_env(name)

Returns the value of an environment variable as a string.

```yaml
- step:
    assign:
      - project: ${sys.get_env("GOOGLE_CLOUD_PROJECT_ID")}
```

**Built-in variables** provided by the emulator:

| Variable | Description |
|----------|-------------|
| `GOOGLE_CLOUD_PROJECT_ID` | Project ID (from `PROJECT` env var) |
| `GOOGLE_CLOUD_LOCATION` | Location (from `LOCATION` env var) |
| `GOOGLE_CLOUD_WORKFLOW_ID` | Current workflow ID |
| `GOOGLE_CLOUD_WORKFLOW_REVISION_ID` | Current revision ID |
| `GOOGLE_CLOUD_WORKFLOW_EXECUTION_ID` | Current execution ID |

Raises KeyError if the variable name is not found.

### sys.log(data, severity)

Logs a message. The emulator prints to stdout.

```yaml
- step:
    call: sys.log
    args:
      data: "Processing started"
      severity: "INFO"
```

Also accepts `text` as an alias for `data`, and `json` for structured logging.

Severity values: DEFAULT, DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY.

### sys.now()

Returns the current Unix timestamp as a double (seconds since epoch).

```yaml
- step:
    assign:
      - timestamp: ${sys.now()}
```

### sys.sleep(seconds)

Pauses execution for the specified number of seconds.

```yaml
- step:
    call: sys.sleep
    args:
      seconds: 5
```

---

## events

Callback support for event-driven async patterns. See also [Callbacks](../reference/rest-api.md#callbacks-api) in the REST API reference.

### events.create_callback_endpoint(http_callback_method)

Creates a callback endpoint that external services can invoke.

```yaml
- create:
    call: events.create_callback_endpoint
    args:
      http_callback_method: "POST"   # Optional, default "POST"
    result: callback_details
# callback_details.url contains the full callback URL
```

Supported methods: GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH.

### events.await_callback(callback, timeout)

Pauses execution until a callback request is received or timeout elapses.

```yaml
- wait:
    call: events.await_callback
    args:
      callback: ${callback_details}
      timeout: 3600                    # Optional, default 43200 (12 hours)
    result: callback_data
```

The returned `callback_data` contains:

```yaml
callback_data.http_request.body     # Parsed request body
callback_data.http_request.headers  # Request headers
callback_data.http_request.method   # HTTP method used
callback_data.received_time         # Timestamp
callback_data.type                  # "HTTP"
```

Raises TimeoutError if timeout elapses before a callback is received.

### Callback pattern example

```yaml
main:
  steps:
    - create_callback:
        call: events.create_callback_endpoint
        args:
          http_callback_method: "POST"
        result: cb
    - log_url:
        call: sys.log
        args:
          data: ${cb.url}
    - wait_for_approval:
        try:
          call: events.await_callback
          args:
            callback: ${cb}
            timeout: 300
          result: approval
        except:
          as: e
          steps:
            - check_timeout:
                switch:
                  - condition: ${"TimeoutError" in e.tags}
                    return: "Timed out waiting for approval"
            - rethrow:
                raise: ${e}
    - done:
        return: ${approval.http_request.body}
```

---

## text

String manipulation functions. All regex functions use RE2 syntax (not PCRE).

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `text.find_all` | `source`, `substr` | list of `{index, match}` | Find all substring occurrences |
| `text.find_all_regex` | `source`, `pattern` | list of `{index, match}` | Find all regex matches |
| `text.match_regex` | `source`, `pattern` | bool | Test if regex matches |
| `text.replace_all` | `source`, `substr`, `replacement` | string | Replace all occurrences |
| `text.replace_all_regex` | `source`, `pattern`, `replacement` | string | Replace regex matches (`\0` full match, `\1`-`\9` groups) |
| `text.split` | `source`, `separator` | list of strings | Split string |
| `text.substring` | `source`, `start`, `end` | string | Substring (0-based, start inclusive, end exclusive) |
| `text.to_lower` | `source` | string | Lowercase |
| `text.to_upper` | `source` | string | Uppercase |
| `text.url_encode` | `source` | string | Percent-encode |
| `text.url_decode` | `source` | string | Percent-decode |
| `text.url_encode_plus` | `source` | string | Percent-encode with `+` for spaces |
| `text.decode` | `data`, `charset` | string | Bytes to string (default UTF-8) |
| `text.encode` | `data`, `charset` | bytes | String to bytes (default UTF-8) |

---

## json

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `json.decode` | `data` (string or bytes) | any | Parse JSON |
| `json.encode` | `value` | bytes | Encode to JSON bytes |
| `json.encode_to_string` | `value` | string | Encode to JSON string |

```yaml
- step:
    assign:
      - parsed: ${json.decode("{\"key\": \"value\"}")}
      - encoded: ${json.encode_to_string(my_map)}
```

---

## base64

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `base64.decode` | `data` (string) | bytes | Decode base64 |
| `base64.encode` | `data` (bytes or string) | string | Encode to base64 |

---

## math

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `math.abs` | `value` | same type | Absolute value |
| `math.floor` | `value` (double) | int | Floor (largest int <= value) |
| `math.max` | `a`, `b` | larger value | Maximum |
| `math.min` | `a`, `b` | smaller value | Minimum |

---

## list

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `list.concat` | `list`, `element` | new list | Append element (does not modify original) |
| `list.prepend` | `list`, `element` | new list | Prepend element (does not modify original) |

```yaml
- step:
    assign:
      - items: ${list.concat(items, "new_item")}
      - items: ${list.prepend(items, "first")}
```

---

## map

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `map.get` | `map`, `key`, `default?` | value | Get value without KeyError. Returns null (or default) if missing |
| `map.delete` | `map`, `key` | new map | Remove key (does not modify original) |
| `map.merge` | `map1`, `map2` | new map | Shallow merge (`map2` overrides) |
| `map.merge_nested` | `map1`, `map2` | new map | Deep merge (recursively merges nested maps) |

```yaml
- step:
    assign:
      - value: ${map.get(config, "timeout", 30)}
      - cleaned: ${map.delete(response, "internal_field")}
      - combined: ${map.merge(defaults, overrides)}
```

---

## uuid

| Function | Parameters | Returns | Description |
|----------|-----------|---------|-------------|
| `uuid.generate` | (none) | string | Random UUID v4 (e.g., `"550e8400-e29b-41d4-a716-446655440000"`) |

---

## retry

Built-in retry policies for use with `try/retry` blocks.

| Policy | Description |
|--------|-------------|
| `retry.always` | Always retry (returns true for any error) |
| `retry.never` | Never retry (returns false for any error) |
| `retry.default_backoff` | Default backoff: initial_delay 1s, max_delay 60s, multiplier 1.25 |

### Custom retry predicates

Define a subworkflow that receives the error map and returns true/false:

```yaml
main:
  steps:
    - call_with_retry:
        try:
          call: http.get
          args:
            url: http://localhost:9090/api
          result: response
        retry:
          predicate: ${my_retry_predicate}
          max_retries: 5
          backoff:
            initial_delay: 1
            max_delay: 60
            multiplier: 2

my_retry_predicate:
  params: [e]
  steps:
    - check:
        switch:
          - condition: ${"ConnectionFailedError" in e.tags}
            return: true
          - condition: ${"TimeoutError" in e.tags}
            return: true
          - condition: ${("HttpError" in e.tags) and (e.code in [429, 500, 502, 503, 504])}
            return: true
    - no_retry:
        return: false
```

This predicate retries on connection failures, timeouts, and specific HTTP status codes including 500 (which `http.default_retry` does not retry).
