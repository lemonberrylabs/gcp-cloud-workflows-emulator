# Step Types - Complete Reference

## Overview

A workflow consists of one or more named steps. Steps execute sequentially by default. Each step is a single-property YAML object where the key is the step name and the value defines the step behavior.

```yaml
main:
  steps:
    - step_name:
        <step_type>: <value>
```

Minimum 1 step per workflow, maximum 100,000 steps per execution.

## Common Fields (available on most step types)

- `next`: Jump to a named step, or special values `end`, `break`, `continue`
- `result`: Store the result of a `call` step in a variable

## Step Type: assign

Assigns values to variables. Assignments execute sequentially within the step.

```yaml
- init:
    assign:
      - var1: "hello"
      - var2: 42
      - var3: ${var1 + " world"}
      - myList[0]: "new_value"
      - myMap.key: "value"
      - myMap["dynamic_key"]: "value"
```

### Constraints
- **Maximum 50 assignments** per assign step
- Each assignment is a single key-value pair
- Left-hand side can be:
  - Simple variable name: `varName`
  - List index: `varName[index]` or `varName[expression]`
  - Map property: `varName.key` or `varName["key"]` or `varName[expression]`
  - Nested: `varName.key1.key2[0]`
- Right-hand side: any expression, literal value, list, or map
- Assignments within a single assign step are evaluated sequentially (later assignments can reference earlier ones)

### Validation Rules
- Variable names: letters, digits, underscores; must start with letter or underscore
- Assigning to a non-existent nested path creates intermediate structures (for maps)
- Assigning to an out-of-bounds list index raises **IndexError**

## Step Type: call

Invokes an HTTP endpoint, standard library function, connector, or subworkflow.

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

### Fields
| Field | Required | Description |
|-------|----------|-------------|
| `call` | Yes | Target: `http.get`, `http.post`, `sys.log`, subworkflow name, etc. |
| `args` | Depends | Arguments passed to the call target. Required for HTTP, optional for some stdlib |
| `result` | No | Variable name to store the return value |

### HTTP Call Args
| Arg | Type | Description |
|-----|------|-------------|
| `url` | string | Required. Target URL (HTTP or HTTPS) |
| `method` | string | Only for `http.request`. HTTP method. |
| `headers` | map | Request headers |
| `body` | any | Request body. Auto-serialized to JSON if Content-Type not specified |
| `query` | map | URL query parameters |
| `auth` | map | Authentication: `{type: "OIDC"}` or `{type: "OAuth2"}` |
| `timeout` | number | Timeout in seconds (max 1800 = 30 minutes) |

### HTTP Response (stored in result)
```
result.body    - Response body (auto-parsed from JSON to map/list if Content-Type is application/json)
result.code    - HTTP status code (integer)
result.headers - Response headers (map, keys are lowercase)
```

### Auth Configuration
```yaml
auth:
  type: OIDC          # or OAuth2
  audience: "https://my-service.run.app"  # OIDC only, defaults to URL
  scope: "https://www.googleapis.com/auth/cloud-platform"  # OAuth2 scope
  scopes: "scope1 scope2"  # Alternative: space or comma separated
```

### Subworkflow Call
```yaml
- call_sub:
    call: my_subworkflow
    args:
      param1: "value1"
      param2: 42
    result: sub_result
```

Arguments are passed by name (matching subworkflow params).

### Calling Subworkflows as Expressions
```yaml
- result_step:
    assign:
      - output: ${my_subworkflow("positional_arg1", "positional_arg2")}
```

When called in an expression, arguments are positional.

## Step Type: switch

Conditional branching. Evaluates conditions in order; executes the first match.

```yaml
- check_value:
    switch:
      - condition: ${x < 10}
        next: handle_small
      - condition: ${x < 100}
        steps:
          - handle:
              assign:
                - category: "medium"
      - condition: true    # default/fallback condition
        next: handle_large
    next: default_next_step   # If no condition matches, go here
```

### Constraints
- **Maximum 50 conditions** per switch step
- Each condition entry has:
  - `condition`: expression that evaluates to boolean (required)
  - `next`: step name to jump to (optional)
  - `steps`: inline steps to execute (optional)
  - `assign`, `call`, `raise`, `return`: any step type can be inline
- `condition: true` acts as a default/else clause
- If no condition matches AND no `next` on the switch step, execution continues to the next sequential step
- If no condition matches AND the switch has a `next` field, jump to that step
- Each condition can have EITHER `next` OR `steps` (or other inline step content), not both

### Evaluation
- Conditions evaluated top-to-bottom
- First matching condition executes; all subsequent conditions are skipped
- Condition expressions MUST evaluate to boolean (non-boolean raises TypeError)

## Step Type: for

Iterates over lists, maps (via `keys()`), or numeric ranges.

### Iterate over a list
```yaml
- loop:
    for:
      value: v         # Required: loop variable for current element
      index: i         # Optional: loop variable for current index (0-based)
      in: ${my_list}   # Required: expression evaluating to a list
      steps:           # Required: steps to execute each iteration
        - process:
            assign:
              - sum: ${sum + v}
```

### Iterate over a range
```yaml
- loop_range:
    for:
      value: v
      range: [1, 9]   # Inclusive on BOTH ends: 1, 2, 3, ... 9
      steps:
        - process:
            assign:
              - sum: ${sum + v}
```

### Iterate over map keys
```yaml
- loop_map:
    for:
      value: key
      in: ${keys(my_map)}
      steps:
        - process:
            assign:
              - result: ${result + my_map[key]}
```

### Loop Control
- `next: break` — exit the for loop immediately
- `next: continue` — skip to the next iteration

### Variable Scoping
- **Variables created inside a for loop do NOT exist after the loop ends**
- Variables from the parent scope that are modified inside the loop **retain their changes** after the loop
- The loop variable (`value`) and index variable (`index`) are scoped to the loop body
- To use a result from a loop, assign to a variable declared BEFORE the loop

### Constraints
- `range` takes exactly 2 elements: `[start, end]`, both inclusive
- `range` values must be integers
- `in` and `range` are mutually exclusive (use one or the other)
- `steps` is required
- `value` is required; `index` is optional
- Empty list iteration: loop body executes 0 times, loop completes normally

### Known Issue
- Placing a `for` loop directly after a `try` block can cause deployment failure with error: `"loop step name should not be empty (Code: 3)"`. Workaround: wrap the `for` loop in a named step.

## Step Type: parallel

Concurrent execution of branches or parallel for loops.

### Parallel Branches
```yaml
- parallel_step:
    parallel:
      shared: [var1, var2]         # Variables writable by branches
      concurrency_limit: 5          # Optional: max concurrent branches
      exception_policy: continueAll # Optional: error handling policy
      branches:
        - branch_a:
            steps:
              - step1:
                  call: http.get
                  args:
                    url: https://api.com/a
                  result: var1
        - branch_b:
            steps:
              - step2:
                  call: http.get
                  args:
                    url: https://api.com/b
                  result: var2
```

### Parallel For Loop
```yaml
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

See `parallel-execution.md` for full details.

## Step Type: try / except / retry

Error handling with optional retry.

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

### try block
- Contains `steps` (required if multiple steps) or a single inline step
- Can contain any step types including nested try/except

### retry block (optional)
| Field | Type | Description |
|-------|------|-------------|
| `predicate` | expression | Reference to a retry predicate function/subworkflow |
| `max_retries` | int | Maximum number of retry attempts |
| `backoff` | map | Exponential backoff configuration |
| `backoff.initial_delay` | number | Initial delay in seconds |
| `backoff.max_delay` | number | Maximum delay in seconds |
| `backoff.multiplier` | number | Multiplier for each subsequent delay |

Delay calculation: `min(initial_delay * multiplier^attempt, max_delay)`

### except block (optional)
| Field | Description |
|-------|-------------|
| `as` | Variable name to receive the error map |
| `steps` | Steps to execute when an error is caught |

### Error Variable Structure
```
e.message  - Human-readable error description (string)
e.code     - Error code (integer, e.g. HTTP status code)
e.tags     - List of error type tags (e.g. ["HttpError", "NotFound"])
```

HTTP errors may also include:
```
e.headers  - Response headers
e.body     - Response body
```

See `error-model.md` for complete error details.

## Step Type: raise

Raises a custom error.

```yaml
# Raise a string error
- raise_string:
    raise: "Something went wrong"

# Raise a map error
- raise_map:
    raise:
      code: 55
      message: "Custom error occurred"

# Re-raise a caught error
- rethrow:
    raise: ${e}

# Raise with expression
- raise_dynamic:
    raise: ${"Error: " + detail}
```

### Accepted Values
- **String**: becomes the error message; tags will be empty, code will be 0
- **Map**: can include `code`, `message`, `tags` fields (all optional)
- **Expression**: evaluated and the result (string or map) is raised

### Behavior
- Execution immediately stops at the current scope
- Error propagates up through try/except blocks
- If unhandled, the execution fails with state FAILED

## Step Type: return

Returns a value from the current workflow or subworkflow.

```yaml
# Return a single value
- done:
    return: ${result}

# Return a map
- done:
    return:
      status: "ok"
      data: ${processed_data}

# Return from main workflow (terminates execution)
- done:
    return: "Success"
```

### Behavior
- In `main`: terminates the entire workflow execution; the value becomes the execution result
- In subworkflows: returns to the caller; value is stored in caller's `result` variable
- Return value can be any type: string, int, double, bool, null, list, map
- The returned value is serialized as JSON in the execution result

## Step Type: steps (nested steps)

Groups steps under a parent step for organization.

```yaml
- parent_step:
    steps:
      - child_step_1:
          assign:
            - x: 1
      - child_step_2:
          assign:
            - y: 2
```

### Behavior
- Nested steps execute sequentially within the parent step
- Variable scope is shared with the parent
- `next` within nested steps can target siblings or parent-level steps

## Flow Control: next

The `next` field controls which step executes after the current step.

```yaml
- step_one:
    assign:
      - x: 1
    next: step_three

- step_two:
    assign:
      - y: 2

- step_three:
    assign:
      - z: 3
    next: end
```

### Special next Values
| Value | Description |
|-------|-------------|
| `end` | Terminate the current workflow/subworkflow (return null) |
| `break` | Exit the enclosing for loop (only valid inside for loops) |
| `continue` | Skip to next iteration of enclosing for loop (only valid inside for loops) |

### Validation Rules
- `next` must reference a valid step name in the same scope, or a special value
- Forward and backward jumps are both allowed
- `break` and `continue` are only valid inside `for` loop steps
- `next` on the last step is optional (implicit end)
- Unreachable steps are allowed (no dead-code elimination)
