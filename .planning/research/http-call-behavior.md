# HTTP Call Behavior — Deep Dive (Critical for Emulator Fidelity)

This document covers the exact semantics of `http.*` call steps, which is the **core value proposition** of the emulator. Developers will run their services locally and use the emulator to orchestrate them via `http.*` steps. Getting this behavior exactly right is essential.

## HTTP Methods

All methods share the same args structure. The only difference is the implied HTTP method.

| Function | HTTP Method |
|----------|-------------|
| `http.get` | GET |
| `http.post` | POST |
| `http.put` | PUT |
| `http.patch` | PATCH |
| `http.delete` | DELETE |
| `http.request` | Specified via `method` arg |

## Request Construction

### URL
- Required field
- Both HTTP and HTTPS are supported
- Query parameters from `query` arg are appended to the URL
- IP-based endpoints of GKE cluster control planes are NOT supported (but the emulator can ignore this restriction)

### Headers
- Map of header name -> value
- **User-Agent**: The runtime ALWAYS appends `GoogleCloudWorkflows; (+https://cloud.google.com/workflows/docs)` to any user-specified User-Agent value. If none set, this becomes the User-Agent.
- **Content-Type auto-detection** (when no Content-Type header is explicitly set):
  - If body is present and NOT bytes: `Content-Type: application/json; charset=utf-8`
  - If body is bytes: `Content-Type: application/octet-stream`
  - If body is absent/null: no Content-Type header added

### Body Encoding
- **No Content-Type set + body is map/list/primitive**: Body is JSON-encoded, Content-Type set to `application/json; charset=utf-8`
- **Content-Type: application/json**: Body is JSON-encoded
- **Content-Type: application/x-www-form-urlencoded**: Body must be provided as an unencoded string (the body is sent as-is)
- **Content-Type: text/***: Body sent as string
- **Body is bytes**: Sent as raw bytes with `application/octet-stream` if no Content-Type

### Query Parameters
- Map of key -> value
- Values are URL-encoded automatically
- Appended to URL as `?key1=value1&key2=value2`

### Authentication (Emulator behavior)
In real GCW:
- `auth.type: OIDC` - Uses OpenID Connect token
- `auth.type: OAuth2` - Uses OAuth 2.0 access token
- `auth.audience` - OIDC audience (defaults to URL)
- `auth.scope` / `auth.scopes` - OAuth2 scope(s)

**For the emulator**: Auth should be a no-op. Do NOT add auth headers. The emulator should accept the `auth` config without error but not enforce it.

### Timeout
- Default: 1800 seconds (30 minutes)
- Maximum: 1800 seconds
- If exceeded: **TimeoutError** is raised
- Configurable per-call via `timeout` arg

## Response Handling

### Response Structure
The result variable from an HTTP call is always a map with exactly 3 fields:

```
{
  "body": <parsed response body>,
  "code": <int HTTP status code>,
  "headers": <map of lowercase header names to values>
}
```

### Response Body Parsing (AUTO)
| Response Content-Type | Body Type | Behavior |
|----------------------|-----------|----------|
| `application/json` | map/list/primitive | Automatically parsed from JSON |
| `text/*` | string | Raw text |
| Other / no Content-Type | bytes | Raw bytes |
| `application/json` but invalid JSON | string (or error) | May return raw string |

**Critical for emulator**: When a localhost service returns `application/json`, the body MUST be auto-parsed to a map/list. This is what developers expect and test against.

### Response Headers
- Headers are returned as a map
- Header names are **lowercased**
- If multiple values exist for a header, behavior is implementation-defined

### Response Size Limit
- Maximum 2 MB response body
- Exceeding this raises a **ResourceLimitError**

## Error Mapping for HTTP Calls

### Successful Responses (2xx)
- Status codes 200-299: No error raised
- Response stored in `result` variable with body, code, headers

### Client/Server Errors (non-2xx)
- **Any non-2xx status code** raises an error with `tags: ["HttpError"]`
- The error map contains:
  - `code`: The HTTP status code (int)
  - `message`: Error description
  - `tags`: `["HttpError"]` (may include additional tags)
  - `headers`: Response headers
  - `body`: Response body (parsed if JSON)

### Connection Errors
| Scenario | Error Tag | Description |
|----------|-----------|-------------|
| DNS failure / wrong domain | `ConnectionFailedError` | Connection never established |
| Connection refused (service not running) | `ConnectionFailedError` | Target not listening |
| Connection reset during transfer | `ConnectionError` | Connection broke mid-transfer |
| SSL/TLS handshake failure | `ConnectionFailedError` | TLS negotiation failed |

**CRITICAL for emulator**: When a developer's localhost service is not running, the emulator MUST raise a `ConnectionFailedError`, NOT a `ConnectionError`. This distinction matters:
- `ConnectionFailedError` = "I couldn't even connect" (service down, wrong port)
- `ConnectionError` = "I connected but the transfer broke" (service crashed mid-response)

### Timeout Errors
- `TimeoutError` raised when the request exceeds the configured timeout
- Default timeout: 1800 seconds
- The timeout covers the entire request lifecycle (connection + transfer + response)

## HTTP Redirect Behavior

Not explicitly documented in GCW docs. Based on standard Go HTTP client behavior (which GCW likely uses internally):
- 301, 302, 303: Follow redirect, change method to GET
- 307, 308: Follow redirect, preserve method
- Maximum redirects: implementation-defined (typically 10)

**Emulator recommendation**: Follow standard Go `http.Client` redirect behavior (which follows redirects by default, up to 10 hops).

## Built-in Retry Policies

### http.default_retry
Complete pre-configured retry policy:
- **Retries on**: HTTP 429, 502, 503, 504, ConnectionError, TimeoutError
- **Does NOT retry on**: HTTP 500 (this surprises many users)
- **max_retries**: 5
- **backoff**: Uses `retry.default_backoff` (initial_delay: 1s, max_delay: 60s, multiplier: 1.25)

### http.default_retry_non_idempotent
Same as above but intended for non-idempotent methods (POST, PATCH):
- **Retries on**: HTTP 429, 502, 503, 504, ConnectionError, TimeoutError
- **Does NOT retry on**: HTTP 500
- **max_retries**: 5
- **backoff**: Uses `retry.default_backoff`

### http.default_retry_predicate
Predicate function only (returns true/false, no backoff config):
- Returns `true` for: 429, 502, 503, 504, ConnectionError, TimeoutError
- Returns `false` for everything else

### http.default_retry_predicate_non_idempotent
Same predicate as above for non-idempotent calls.

### retry.default_backoff
Default backoff configuration:
- `initial_delay`: 1 second
- `max_delay`: 60 seconds
- `multiplier`: 1.25

### Retry Mechanics
```
Attempt 1: Execute step
  If error matches predicate:
    Wait initial_delay seconds
Attempt 2: Re-execute ENTIRE try block from beginning
  If error matches predicate:
    Wait min(initial_delay * multiplier, max_delay) seconds
Attempt 3: Re-execute ENTIRE try block from beginning
  ...
After max_retries retries: Error propagates to except block (or fails execution)
```

- **Total attempts** = 1 (initial) + max_retries
- Retries re-execute the ENTIRE try block, not just the failed step
- If the retry predicate itself throws an error, retry is aborted

## Custom Retry Predicates

Custom predicates are subworkflows that receive the error map and return boolean:

```yaml
my_retry_predicate:
  params: [e]
  steps:
    - check:
        switch:
          - condition: ${"ConnectionFailedError" in e.tags}
            return: true
          - condition: ${"ConnectionError" in e.tags}
            return: true
          - condition: ${"TimeoutError" in e.tags}
            return: true
          - condition: ${("HttpError" in e.tags) and (e.code in [429, 500, 502, 503, 504])}
            return: true
    - no_retry:
        return: false
```

## Emulator-Specific Considerations

### Localhost HTTP Calls
When the emulator executes `http.*` steps, it makes **real HTTP requests** to the URLs specified. For local development:
- `url: http://localhost:8081/api/users` will make a real HTTP call to port 8081
- The developer runs their service locally on that port
- The emulator orchestrates the calls exactly as real GCW would

### Error Fidelity Priorities (for the emulator)
1. **ConnectionFailedError** when target service is not running (connection refused)
2. **HttpError** with correct `code` for non-2xx responses
3. **TimeoutError** when requests take too long
4. **Response auto-parsing** (JSON -> map/list) based on Content-Type
5. **Correct User-Agent** header behavior (append, don't replace)
6. **Correct Content-Type** auto-detection for request body

### What NOT to Emulate
- IAM authentication (accept `auth` config but don't enforce)
- Service Directory private endpoints
- IAP-secured endpoints
- VPC Service Controls
