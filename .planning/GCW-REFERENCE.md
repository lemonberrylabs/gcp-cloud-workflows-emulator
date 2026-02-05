# Google Cloud Workflows — Complete Reference

## 1. Product Overview

Google Cloud Workflows is a fully managed serverless orchestration platform that executes services in a defined order. It combines custom services (Cloud Run, Cloud Functions), Google Cloud services, and any HTTP-based API into automated workflows.

**Key characteristics:**
- Serverless: No infrastructure to manage; scales automatically
- Stateful: Holds state across steps; can retry, poll, or wait for up to 1 year
- Pay-per-use: No charges while idle
- Regional: Workflows are deployed and execute within a specific region
- Independent executions: All executions are independent

## 2. Workflow Definition Language

Workflows are defined in YAML or JSON. A workflow consists of a series of steps executed sequentially by default.

### Basic Structure

```yaml
main:
  params: [args]
  steps:
    - step_name:
        <step_type>: <value>
    - another_step:
        <step_type>: <value>
```

Steps execute sequentially by default. Each step can optionally include a `next` field to jump to another step or `end` to finish.

## 3. Step Types

### 3.1 assign

Assigns values to variables. Max 50 assignments per step.

```yaml
- initialize:
    assign:
      - my_integer: 1
      - my_string: "hello"
      - my_list: ["zero", "one"]
      - my_map:
          name: Lila
      - my_list[0]: "new_value"
      - my_map.newKey: "new_value"
```

### 3.2 call

Invokes HTTP endpoint, stdlib function, connector, or subworkflow.

```yaml
- make_request:
    call: http.get
    args:
      url: https://example.com/api
      headers:
        Content-Type: "application/json"
      query:
        action: opensearch
      auth:
        type: OIDC
      timeout: 300
    result: api_response
```

HTTP response structure:
- `${result.body}` — response body (auto-parsed from JSON)
- `${result.code}` — HTTP status code
- `${result.headers}` — response headers map

### 3.3 switch

Conditional branching. Evaluates conditions in order; executes first match. Max 50 conditions.

```yaml
- check_value:
    switch:
      - condition: ${my_integer < 10}
        next: handle_small
      - condition: ${my_integer < 100}
        steps:
          - handle:
              assign:
                - category: "medium"
```

### 3.4 for

Iterates over lists, maps (via keys()), or ranges.

```yaml
- loop_list:
    for:
      value: v
      index: i
      in: ${my_list}
      steps:
        - process:
            assign:
              - sum: ${sum + v}

- loop_range:
    for:
      value: v
      range: [1, 9]  # inclusive both ends
      steps:
        - process:
            assign:
              - sum: ${sum + v}
```

Loop control: `next: break` exits loop, `next: continue` skips to next iteration.

Variables created inside a for loop do NOT exist outside. Variables from parent scope modified inside retain changes.

### 3.5 parallel

Concurrent branches or parallel for loops.

```yaml
- parallel_step:
    parallel:
      shared: [user, notification]
      concurrency_limit: 5
      exception_policy: continueAll
      branches:
        - getUser:
            steps:
              - call: http.get
                args:
                  url: https://example.com/users/1
                result: user
        - getNotification:
            steps:
              - call: http.get
                args:
                  url: https://example.com/notif/1
                result: notification

- parallel_loop:
    parallel:
      shared: [total]
      for:
        value: item
        in: ${items}
        steps:
          - process:
              assign:
                - total: ${total + 1}
```

Limits: 10 branches, 20 concurrent, nesting depth 2, 100 max unhandled exceptions.
Shared variables: atomic reads/writes, immediately visible.
Exception policies: `unhandled` (default, abort on first) or `continueAll`.

### 3.6 try/except/retry

```yaml
- handle_errors:
    try:
      steps:
        - step_a:
            call: http.get
            args:
              url: https://example.com
            result: response
    retry:
      predicate: ${http.default_retry_predicate}
      max_retries: 10
      backoff:
        initial_delay: 1
        max_delay: 90
        multiplier: 3
    except:
      as: e
      steps:
        - handle:
            switch:
              - condition: ${e.code == 404}
                return: "Not found"
        - rethrow:
            raise: ${e}
```

Built-in retry policies:
- `${http.default_retry}` — 429, 502, 503, 504, ConnectionError, TimeoutError
- `${http.default_retry_non_idempotent}`
- `${http.default_retry_predicate}` — predicate only
- `${retry.always}`, `${retry.never}`, `${retry.default_backoff}`

### 3.7 raise

```yaml
- raise_error:
    raise:
      code: 55
      message: "Something went wrong"
```

### 3.8 return

```yaml
- done:
    return: ${result}
```

### 3.9 next

```yaml
- step_one:
    assign:
      - x: 1
    next: step_three
```

Special values: `end`, `break`, `continue`.

## 4. Standard Library

### Expression Helpers
`default(value, default_value)`, `keys(map)`, `len(value)`, `type(value)`, `int(value)`, `double(value)`, `string(value)`, `bool(value)`

### http.*
`http.get`, `http.post`, `http.put`, `http.patch`, `http.delete`, `http.request`

### sys.*
`sys.get_env(name)`, `sys.log(data, severity)`, `sys.now()`, `sys.sleep(seconds)`, `sys.sleep_until(timestamp)`

Severities: DEFAULT, DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY

Environment variables: GOOGLE_CLOUD_PROJECT_ID, GOOGLE_CLOUD_PROJECT_NUMBER, GOOGLE_CLOUD_LOCATION, GOOGLE_CLOUD_WORKFLOW_ID, GOOGLE_CLOUD_WORKFLOW_REVISION_ID, GOOGLE_CLOUD_WORKFLOW_EXECUTION_ID, GOOGLE_CLOUD_WORKFLOW_EXECUTION_ATTEMPT

### events.*
`events.create_callback_endpoint(http_callback_method)`, `events.await_callback(callback, timeout)`

### text.*
`text.decode`, `text.encode`, `text.find_all`, `text.find_all_regex`, `text.match_regex`, `text.replace_all`, `text.replace_all_regex`, `text.split`, `text.substring`, `text.to_lower`, `text.to_upper`, `text.url_decode`, `text.url_encode`, `text.url_encode_plus`

### json.*
`json.decode`, `json.encode`, `json.encode_to_string`

### base64.*
`base64.decode`, `base64.encode`

### math.*
`math.abs`, `math.floor`, `math.max`, `math.min`

### list.*
`list.concat`, `list.prepend`

### map.*
`map.get`, `map.delete`, `map.merge`, `map.merge_nested`

### time.*
`time.format`, `time.parse`

### hash.*
`hash.compute_checksum`, `hash.compute_hmac`

### uuid.*
`uuid.generate`

### retry.*
`retry.always`, `retry.never`, `retry.default_backoff`

## 5. Expression Language

Expressions wrapped in `${}`.

### Operators
- Arithmetic: `+`, `-`, `*`, `/`, `%`, `//` (integer division)
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Logical: `and`, `or`, `not`
- String: `+` (concatenation)
- Membership: `in`, `not in`
- Property/Index: `.`, `[]`

Max expression length: 400 characters.

### Null Handling
- `null` is a distinct type
- Accessing missing map key throws KeyError
- Use `map.get(m, key)` for safe access
- Use `default(value, fallback)` for null-safe access

## 6. Subworkflows

```yaml
main:
  params: [args]
  steps:
    - call_sub:
        call: greet
        args:
          first_name: "Ada"
        result: message

greet:
  params: [first_name, last_name: "Unknown"]
  steps:
    - build:
        return: ${"Hello " + first_name + " " + last_name}
```

Max call stack depth: 20. Main accepts single dict param; subworkflows accept multiple named params with optional defaults.

## 7. Error Types

All errors are maps: `{message, code, tags}`

Tags: HttpError, ConnectionError, TimeoutError, SystemError, TypeError, ValueError, KeyError, IndexError, ZeroDivisionError, RecursionError, ResourceLimitError, MemoryLimitExceededError, ResultSizeLimitExceededError, OperationError, ResponseTypeError, AuthenticationError

Errors can have multiple tags (e.g., `["HttpError", "NotFound"]`).

## 8. REST APIs

### Workflows API — Base: `https://workflows.googleapis.com/v1`

| Method | HTTP | Path |
|--------|------|------|
| create | POST | `/{parent}/workflows?workflowId={id}` |
| get | GET | `/{name}` |
| list | GET | `/{parent}/workflows` |
| patch | PATCH | `/{name}` |
| delete | DELETE | `/{name}` |

### Executions API — Base: `https://workflowexecutions.googleapis.com/v1`

| Method | HTTP | Path |
|--------|------|------|
| create | POST | `/{parent}/executions` |
| get | GET | `/{name}` |
| list | GET | `/{parent}/executions` |
| cancel | POST | `/{name}:cancel` |

### Callbacks

| Method | HTTP | Path |
|--------|------|------|
| list | GET | `/{parent}/callbacks` |
| send | POST | `{callback_url}` |

## 9. System Limits

| Limit | Value |
|-------|-------|
| Assignments per step | 50 |
| Conditions per switch | 50 |
| Max call stack depth | 20 |
| Branches per parallel step | 10 |
| Parallel nesting depth | 2 |
| Max concurrent branches | 20 |
| Unhandled exceptions (parallel) | 100 |
| Source code size | 128 KB |
| HTTP response size | 2 MB |
| Expression length | 400 chars |
| Variable memory | 512 KB |
| Max string length | 256 KB |
| Max steps per execution | 100,000 |
| Execution duration max | 1 year |
| HTTP request timeout | 1800s |
