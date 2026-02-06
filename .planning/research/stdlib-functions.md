# Standard Library Functions - Complete Reference

## Expression Helpers (built-in, no module prefix)

These are called directly in expressions without a module prefix.

### default(value, default_value)
- **Parameters**: `value` (any), `default_value` (any)
- **Returns**: `value` if it is not null; otherwise `default_value`
- **Use case**: Null-safe access: `${default(map.get(m, "key"), "fallback")}`
- **Edge case**: Does NOT catch KeyError; only handles null values. Use with `map.get()` for safe map access.

### keys(map)
- **Parameters**: `map` (map)
- **Returns**: list of strings (the map's keys)
- **Error**: TypeError if argument is not a map
- **Edge case**: Order of keys is not guaranteed to be consistent

### len(value)
- **Parameters**: `value` (string, list, or map)
- **Returns**: int
  - For strings: number of characters (Unicode code points)
  - For lists: number of elements
  - For maps: number of key-value pairs
- **Error**: TypeError for unsupported types

### type(value)
- **Parameters**: `value` (any)
- **Returns**: string describing the type
  - `"int"` for integers
  - `"double"` for doubles
  - `"string"` for strings
  - `"bool"` for booleans
  - `"list"` for lists
  - `"map"` for maps
  - `"null"` for null
  - `"bytes"` for bytes

### int(value)
- **Parameters**: `value` (string, double, or int)
- **Returns**: int (64-bit signed)
- **Conversion rules**:
  - String: parses as integer (e.g., `int("42")` = 42). Fails on non-numeric strings with ValueError.
  - Double: truncates toward zero (e.g., `int(2.7)` = 2, `int(-2.7)` = -2)
  - Int: returns as-is
- **Error**: ValueError for unparseable strings, TypeError for other types

### double(value)
- **Parameters**: `value` (string, int, or double)
- **Returns**: double (64-bit IEEE 754)
- **Conversion rules**:
  - String: parses as floating-point (e.g., `double("2.7")` = 2.7)
  - Int: converts to double
  - Double: returns as-is
- **Error**: ValueError for unparseable strings, TypeError for other types

### string(value)
- **Parameters**: `value` (int, double, bool, or string)
- **Returns**: string representation
- **Conversion rules**:
  - Int: decimal string (e.g., `string(42)` = "42")
  - Double: decimal string (e.g., `string(1.7)` = "1.7")
  - Bool: "true" or "false"
  - String: returns as-is
- **Error**: TypeError for maps, lists, null, bytes

### bool(value)
- **Parameters**: `value` (string or bool)
- **Returns**: boolean
- **Conversion rules**:
  - String: `"true"` -> true, `"false"` -> false (case-insensitive likely)
  - Bool: returns as-is
- **Error**: ValueError for other string values, TypeError for non-string/non-bool

---

## http module

### http.get(args)
### http.post(args)
### http.put(args)
### http.patch(args)
### http.delete(args)

All HTTP methods accept the same `args` structure:

| Arg | Type | Required | Description |
|-----|------|----------|-------------|
| `url` | string | Yes | Target URL |
| `headers` | map | No | Request headers |
| `body` | any | No | Request body (auto-serialized to JSON if no Content-Type set) |
| `query` | map | No | URL query parameters (appended to URL) |
| `auth` | map | No | Auth config: `{type: "OIDC", audience: "..."}` or `{type: "OAuth2", scope: "..."}` |
| `timeout` | int | No | Timeout in seconds (max 1800, default 1800) |

**Returns**: map with `body`, `code` (int), `headers` (map)

**Auto-behaviors**:
- If no Content-Type header and body exists: sets `Content-Type: application/json; charset=utf-8`
- If body is bytes: sets `Content-Type: application/octet-stream`
- User-Agent header always appends `GoogleCloudWorkflows; (+https://cloud.google.com/workflows/docs)`
- Response body auto-parsed from JSON if Content-Type is `application/json`
- Supported Content-Types: `application/json`, `application/x-www-form-urlencoded`, text types

**Errors**:
- Non-2xx status: raises error with tags `["HttpError"]` plus additional tags based on status
- Connection failure: raises error with tags `["ConnectionError"]`
- Timeout: raises error with tags `["TimeoutError"]`

**Response size limit**: 2 MB

### http.request(args)
Same as above but with an additional required `method` field:
| Arg | Type | Required | Description |
|-----|------|----------|-------------|
| `method` | string | Yes | HTTP method: "GET", "POST", "PUT", "PATCH", "DELETE", etc. |

### http.default_retry
Pre-configured retry policy for idempotent HTTP calls:
- Retries on: 429, 502, 503, 504, ConnectionError, TimeoutError
- Does NOT retry on 500
- max_retries: 5
- Uses `retry.default_backoff`

### http.default_retry_non_idempotent
Same as `http.default_retry` but for non-idempotent calls (POST, PATCH).
- Retries on: 429, 502, 503, 504, ConnectionError, TimeoutError
- Does NOT retry on 500
- max_retries: 5
- Uses `retry.default_backoff`

### http.default_retry_predicate
Predicate function only (no backoff/max_retries config):
- Returns true for: 429, 502, 503, 504, ConnectionError, TimeoutError
- Returns false for all others (including 500)

### http.default_retry_predicate_non_idempotent
Same predicate as above, for non-idempotent calls.

---

## sys module

### sys.get_env(name)
- **Call style**: expression `${sys.get_env("VAR_NAME")}` or call step
- **Parameters**: `name` (string) - environment variable name
- **Returns**: string value of the environment variable
- **Supported built-in variables**:
  - `GOOGLE_CLOUD_PROJECT_ID` - Project ID
  - `GOOGLE_CLOUD_PROJECT_NUMBER` - Project number
  - `GOOGLE_CLOUD_LOCATION` - Deployment region
  - `GOOGLE_CLOUD_WORKFLOW_ID` - Workflow name
  - `GOOGLE_CLOUD_WORKFLOW_REVISION_ID` - Current revision ID
  - `GOOGLE_CLOUD_WORKFLOW_EXECUTION_ID` - Current execution ID
  - `GOOGLE_CLOUD_WORKFLOW_EXECUTION_ATTEMPT` - Current retry attempt number
- **User-defined env vars**: Up to 20 variables, max 4 KiB each. Cannot start with "GOOGLE" or "WORKFLOWS".
- **Error**: KeyError if variable name not found

### sys.log(data, severity)
- **Call style**: call step
- **Parameters**:
  - `data` (any, required): data to log (serialized as JSON if map/list)
  - `severity` (string, optional): log severity level
  - Also accepts: `text` (string) as alias for `data`, `json` (map) for structured logging
- **Severity values**: DEFAULT, DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY
- **Returns**: null
- **Behavior**: Writes to Cloud Logging

### sys.now()
- **Call style**: expression `${sys.now()}` or call step
- **Parameters**: none
- **Returns**: double - Unix timestamp (seconds since epoch as a floating-point number)
- **Precision**: Seconds (with potential sub-second fractional component)

### sys.sleep(seconds)
- **Call style**: call step
- **Parameters**: `seconds` (int or double) - duration to sleep
- **Returns**: null
- **Behavior**: Pauses execution for the specified number of seconds
- **Constraint**: Part of execution duration limit (1 year max total)

### sys.sleep_until(timestamp)
- **Call style**: call step
- **Parameters**: `timestamp` (double) - Unix timestamp to sleep until
- **Returns**: null
- **Behavior**: Pauses execution until the specified Unix timestamp
- **Edge case**: If timestamp is in the past, returns immediately

---

## events module

### events.create_callback_endpoint(http_callback_method)
- **Call style**: call step
- **Parameters**:
  - `http_callback_method` (string, optional): HTTP method to accept. Default: "POST"
  - Supported: GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH
- **Returns**: map with:
  - `url` (string): The callback URL in format `https://workflowexecutions.googleapis.com/v1/projects/{project}/locations/{location}/workflows/{workflow}/executions/{execution}/callbacks/{callbackId}`
- **Behavior**: Creates a callback endpoint that external services can invoke

### events.await_callback(callback, timeout)
- **Call style**: call step
- **Parameters**:
  - `callback` (map, required): The callback details map from `create_callback_endpoint`
  - `timeout` (double, optional): Maximum seconds to wait. Default: 43200 (12 hours)
- **Returns**: map with:
  - `http_request.body` - Parsed request body (JSON->map, text, or raw bytes)
  - `http_request.headers` - Request headers map
  - `http_request.method` - HTTP method used
  - `http_request.query` - Query parameters
  - `http_request.url` - Callback endpoint path
  - `received_time` - Timestamp when callback was received
  - `type` - "HTTP"
- **Error**: TimeoutError if timeout elapses before callback received
- **Edge case**: Only one callback request fills the slot at a time. Additional requests:
  - HTTP 429 if first not yet processed
  - HTTP 200 if first processed (second stored but may be discarded)
  - HTTP 404 if workflow completed/failed

---

## text module

### text.decode(data, charset)
- **Parameters**: `data` (bytes), `charset` (string, optional, default "UTF-8")
- **Returns**: string
- **Use case**: Convert bytes to string

### text.encode(data, charset)
- **Parameters**: `data` (string), `charset` (string, optional, default "UTF-8")
- **Returns**: bytes
- **Use case**: Convert string to bytes

### text.find_all(source, substr)
- **Parameters**: `source` (string), `substr` (string)
- **Returns**: list of maps, each containing:
  - `index` (int): position of match
  - `match` (string): the matched substring

### text.find_all_regex(source, pattern)
- **Parameters**: `source` (string), `pattern` (string - RE2 regex)
- **Returns**: list of maps, each containing:
  - `index` (int): position of match
  - `match` (string): the matched substring
  - Groups captured by regex are also accessible
- **Regex engine**: RE2 (not PCRE)

### text.match_regex(source, pattern)
- **Parameters**: `source` (string), `pattern` (string - RE2 regex)
- **Returns**: boolean - true if the pattern matches the source
- **Regex engine**: RE2

### text.replace_all(source, substr, replacement)
- **Parameters**: `source` (string), `substr` (string), `replacement` (string)
- **Returns**: string with all occurrences of `substr` replaced by `replacement`

### text.replace_all_regex(source, pattern, replacement)
- **Parameters**: `source` (string), `pattern` (string - RE2 regex), `replacement` (string)
- **Returns**: string with all regex matches replaced
- **Regex engine**: RE2
- **Backreferences**: `\0` for full match, `\1`-`\9` for capture groups

### text.split(source, separator)
- **Parameters**: `source` (string), `separator` (string)
- **Returns**: list of strings

### text.substring(source, start, end)
- **Parameters**: `source` (string), `start` (int), `end` (int)
- **Returns**: string - substring from `start` (inclusive) to `end` (exclusive)
- **Indexing**: 0-based

### text.to_lower(source)
- **Parameters**: `source` (string)
- **Returns**: string in lowercase

### text.to_upper(source)
- **Parameters**: `source` (string)
- **Returns**: string in uppercase

### text.url_decode(source)
- **Parameters**: `source` (string) - URL-encoded string
- **Returns**: decoded string

### text.url_encode(source)
- **Parameters**: `source` (string)
- **Returns**: URL-encoded string (percent-encoding)

### text.url_encode_plus(source)
- **Parameters**: `source` (string)
- **Returns**: URL-encoded string with spaces as `+` instead of `%20`

---

## json module

### json.decode(data)
- **Parameters**: `data` (string or bytes)
- **Returns**: Parsed value (map, list, string, number, bool, or null)
- **Error**: ValueError if data is not valid JSON

### json.encode(value)
- **Parameters**: `value` (any)
- **Returns**: bytes - JSON-encoded bytes

### json.encode_to_string(value)
- **Parameters**: `value` (any)
- **Returns**: string - JSON-encoded string
- **Use case**: When you need JSON as a string rather than bytes

---

## base64 module

### base64.decode(data)
- **Parameters**: `data` (string) - base64-encoded string
- **Returns**: bytes
- **Error**: ValueError for invalid base64

### base64.encode(data)
- **Parameters**: `data` (bytes or string)
- **Returns**: string - base64-encoded

---

## math module

### math.abs(value)
- **Parameters**: `value` (int or double)
- **Returns**: Same type as input, absolute value

### math.floor(value)
- **Parameters**: `value` (double)
- **Returns**: int - largest integer less than or equal to value

### math.max(a, b)
- **Parameters**: `a` (int or double), `b` (int or double)
- **Returns**: The larger of the two values

### math.min(a, b)
- **Parameters**: `a` (int or double), `b` (int or double)
- **Returns**: The smaller of the two values

---

## list module

### list.concat(list, element)
- **Parameters**: `list` (list), `element` (any)
- **Returns**: new list with `element` appended to the end
- **Note**: Does NOT modify the original list; returns a copy
- **Use case**: `${list.concat(myList, newItem)}`

### list.prepend(list, element)
- **Parameters**: `list` (list), `element` (any)
- **Returns**: new list with `element` added at the beginning
- **Note**: Does NOT modify the original list; returns a copy

---

## map module

### map.get(map, key, default?)
- **Parameters**: `map` (map), `key` (string), `default` (any, optional)
- **Returns**: Value associated with key, or `default` if key not found, or `null` if key not found and no default provided
- **Key difference from `[]` access**: Does NOT raise KeyError for missing keys
- **Use case**: Safe map access: `${map.get(m, "key", "fallback")}`

### map.delete(map, key)
- **Parameters**: `map` (map), `key` (string)
- **Returns**: new map without the specified key
- **Note**: Does NOT modify the original map; returns a copy
- **Edge case**: If key doesn't exist, returns a copy of the original map

### map.merge(map1, map2)
- **Parameters**: `map1` (map), `map2` (map)
- **Returns**: new map with all key-value pairs from both maps
- **Conflict resolution**: Values from `map2` override values from `map1` for duplicate keys
- **Note**: Shallow merge only

### map.merge_nested(map1, map2)
- **Parameters**: `map1` (map), `map2` (map)
- **Returns**: new map with recursively merged nested maps
- **Behavior**: For keys present in both maps:
  - If both values are maps: recursively merge
  - Otherwise: value from `map2` overrides

---

## time module

### time.format(timestamp, timezone?)
- **Parameters**:
  - `timestamp` (double): Unix timestamp (seconds since epoch)
  - `timezone` (string, optional): IANA timezone name (e.g., "America/New_York"). Default: UTC
- **Returns**: string in RFC 3339 format (e.g., "2023-01-15T10:30:00.000Z")

### time.parse(value)
- **Parameters**: `value` (string) - RFC 3339 formatted timestamp string
- **Returns**: double - Unix timestamp (seconds since epoch)
- **Error**: ValueError for invalid format

---

## hash module

### hash.compute_checksum(data, algorithm)
- **Parameters**:
  - `data` (string or bytes): Data to hash
  - `algorithm` (string): Hash algorithm - "SHA256", "SHA384", "SHA512", "MD5", "SHA1"
- **Returns**: bytes - The computed hash
- **Use case**: Often combined with `base64.encode()` for string representation

### hash.compute_hmac(data, key, algorithm)
- **Parameters**:
  - `data` (string or bytes): Data to authenticate
  - `key` (string or bytes): Secret key
  - `algorithm` (string): HMAC algorithm - "SHA256", "SHA384", "SHA512", "MD5", "SHA1"
- **Returns**: bytes - The computed HMAC
- **Use case**: Message authentication codes

---

## uuid module

### uuid.generate()
- **Parameters**: none
- **Returns**: string - UUID v4 (random) in standard format (e.g., "550e8400-e29b-41d4-a716-446655440000")

---

## retry module

### retry.always
- **Type**: retry predicate (not a function call)
- **Behavior**: Always retries (returns true for any error)

### retry.never
- **Type**: retry predicate
- **Behavior**: Never retries (returns false for any error)

### retry.default_backoff
- **Type**: backoff configuration
- **Values**:
  - initial_delay: 1 second
  - max_delay: 60 seconds
  - multiplier: 1.25

---

## experimental.executions module

### experimental.executions.run(workflow_id, argument)
- **Parameters**:
  - `workflow_id` (string): Workflow to execute
  - `argument` (any, optional): Arguments for the workflow
- **Returns**: Execution result
- **Behavior**: Starts a child workflow execution and waits for it

### experimental.executions.map(workflow_id, arguments)
- **Parameters**:
  - `workflow_id` (string): Workflow to execute for each argument
  - `arguments` (list): List of argument maps
- **Returns**: list of execution results in argument order
- **Behavior**: Executes the workflow for each argument in parallel, waits for all to complete
- **Note**: Recommended to use parallel steps instead (this is legacy)
