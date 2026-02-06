# Workflow Syntax

Workflows are defined in YAML (or JSON). A workflow file contains a `main` workflow and optional subworkflows. Steps execute sequentially by default.

## Basic structure

```yaml
main:
  params: [args]
  steps:
    - step_name:
        <step_type>: <value>
    - another_step:
        <step_type>: <value>
```

## Step types

### assign

Set variables. Up to 50 assignments per step. Assignments within a step execute sequentially -- later assignments can reference earlier ones.

```yaml
- init:
    assign:
      - name: "Alice"
      - age: 30
      - greeting: '${"Hello, " + name}'
      - items: [1, 2, 3]
      - config:
          debug: true
          level: "info"
```

Index and key assignment:

```yaml
- update:
    assign:
      - items[0]: "first"
      - config.debug: false
      - config["new_key"]: "value"
```

Assigning to a nested map path creates intermediate maps: `myMap.a.b.c: "deep"` creates `{a: {b: {c: "deep"}}}`.

### call

Call HTTP endpoints, [standard library functions](./stdlib.md), or subworkflows.

**HTTP call:**

```yaml
- get_data:
    call: http.get
    args:
      url: http://localhost:9090/api/data
      headers:
        Authorization: "Bearer ${token}"
      query:
        limit: "10"
      timeout: 30
    result: response
```

**Standard library call:**

```yaml
- log_it:
    call: sys.log
    args:
      data: "Processing complete"
      severity: "INFO"
```

**Subworkflow call:**

```yaml
- process:
    call: my_subworkflow
    args:
      input: ${data}
    result: output
```

The `result` field stores the return value in a variable. See the [Standard Library](./stdlib.md) for all available functions.

### switch

Conditional branching. Conditions are evaluated top-to-bottom; the first match executes. Up to 50 conditions.

```yaml
- check:
    switch:
      - condition: ${age >= 18}
        steps:
          - adult:
              assign:
                - category: "adult"
      - condition: ${age >= 13}
        next: teenager_step
      - condition: true  # default/fallback
        steps:
          - child:
              assign:
                - category: "child"
```

Each condition entry can have `next` (jump), `steps` (inline steps), or other inline step content (`assign`, `call`, `return`, `raise`).

If no condition matches and the switch step has no `next`, execution continues to the next sequential step.

### for

Iterate over lists, map keys (via `keys()`), or ranges.

**List iteration:**

```yaml
- loop:
    for:
      value: item
      in: ${my_list}
      steps:
        - process:
            call: http.post
            args:
              url: http://localhost:9090/process
              body:
                item: ${item}
```

**With index:**

```yaml
- loop:
    for:
      value: item
      index: i
      in: ${my_list}
      steps:
        - log:
            call: sys.log
            args:
              text: '${"Item " + string(i) + ": " + item}'
```

**Range (inclusive both ends):**

```yaml
- loop:
    for:
      value: i
      range: [1, 10]
      steps:
        - process:
            assign:
              - total: ${total + i}
```

**Map iteration:**

```yaml
- loop:
    for:
      value: key
      in: ${keys(my_map)}
      steps:
        - use:
            assign:
              - val: ${my_map[key]}
```

**Loop control:** Use `next: break` to exit the loop, `next: continue` to skip to the next iteration.

**Scoping:** Variables created inside a for loop do **not** exist after the loop ends. Variables from the parent scope that are modified inside the loop retain their changes. The loop variable (`value`) and index variable are scoped to the loop body.

### parallel

Execute branches concurrently or iterate in parallel.

**Parallel branches:**

```yaml
- fetch_all:
    parallel:
      shared: [results]
      branches:
        - get_users:
            steps:
              - fetch:
                  call: http.get
                  args:
                    url: http://localhost:9090/users
                  result: users
              - save:
                  assign:
                    - results: ${map.merge(results, {"users": users.body})}
        - get_orders:
            steps:
              - fetch:
                  call: http.get
                  args:
                    url: http://localhost:9091/orders
                  result: orders
              - save:
                  assign:
                    - results: ${map.merge(results, {"orders": orders.body})}
```

**Parallel for:**

```yaml
- process_batch:
    parallel:
      shared: [processed]
      for:
        value: item
        in: ${items}
        steps:
          - process:
              call: http.post
              args:
                url: http://localhost:9090/process
                body: ${item}
```

**Options:**

| Field | Description |
|-------|-------------|
| `shared` | Variables from parent scope writable by branches (must be declared before the parallel step) |
| `concurrency_limit` | Max concurrent branches/iterations (default: up to 20) |
| `exception_policy` | `unhandled` (default -- abort on first error) or `continueAll` (collect up to 100 errors) |

**Shared variables:** Individual reads and writes are atomic, but compound operations like `total: ${total + 1}` are **not** atomic as a unit -- race conditions can occur. Variables not in `shared` are read-only copies within each branch.

**Limits:** 10 branches per step, 20 max concurrent, nesting depth 2.

### try / except / retry

Error handling with optional retry. See [Error Handling](./errors.md) for the full error model.

```yaml
- safe_call:
    try:
      call: http.get
      args:
        url: http://localhost:9090/unstable
      result: response
    except:
      as: e
      steps:
        - handle:
            assign:
              - response:
                  error: ${e.message}
```

**With retry:**

```yaml
- retry_call:
    try:
      call: http.get
      args:
        url: http://localhost:9090/flaky
      result: response
    retry:
      predicate: ${http.default_retry}
      max_retries: 5
      backoff:
        initial_delay: 1
        max_delay: 30
        multiplier: 2
    except:
      as: e
      steps:
        - fallback:
            return: "service unavailable"
```

Retries re-execute the **entire try block** from the beginning, not just the failed step.

### raise

Throw an error. Accepts a string or a map.

```yaml
# String error
- fail:
    raise: "something went wrong"

# Structured error
- fail:
    raise:
      code: 400
      message: "invalid input"
      tags: ["ValidationError"]

# Re-raise a caught error
- rethrow:
    raise: ${e}
```

### return

Return a value from the current workflow or subworkflow.

```yaml
- done:
    return: ${result}
```

In `main`, this terminates the execution and the value becomes the execution result. In subworkflows, the value is returned to the caller.

### next

Jump to another step.

```yaml
- check:
    switch:
      - condition: ${x > 10}
        next: big_number
- small:
    assign:
      - category: "small"
    next: done
- big_number:
    assign:
      - category: "big"
- done:
    return: ${category}
```

Special targets: `end` (stop workflow), `break` (exit loop), `continue` (next iteration).

### steps

Nested step grouping for organization. Variables share the parent scope.

```yaml
- outer:
    steps:
      - inner1:
          assign:
            - x: 1
      - inner2:
          assign:
            - y: 2
```

## Subworkflows

Define reusable subworkflows alongside `main`:

```yaml
main:
  steps:
    - call_helper:
        call: add_numbers
        args:
          a: 10
          b: 20
        result: sum
    - done:
        return: ${sum}

add_numbers:
  params: [a, b]
  steps:
    - calc:
        return: ${a + b}
```

**Key points:**

- `main` accepts a single parameter (the execution argument as a map)
- Subworkflows accept multiple named parameters with optional defaults: `params: [required, optional: "default"]`
- Subworkflows can be called from `call` steps (named args) or expressions (positional args): `${add_numbers(10, 20)}`
- Variables are isolated per subworkflow -- a subworkflow cannot access the caller's variables
- Subworkflows can call other subworkflows and themselves (recursion)
- Maximum call stack depth: 20

## Parameters

```yaml
# main receives execution argument as a single map
main:
  params: [args]
  steps:
    - use:
        return: ${args.name}

# Subworkflow with named params and defaults
greet:
  params: [first_name, last_name: "Unknown"]
  steps:
    - build:
        return: '${"Hello " + first_name + " " + last_name}'
```

When triggering an execution, the argument is a JSON-encoded **string**:

```bash
curl -X POST .../executions \
  -H "Content-Type: application/json" \
  -d '{"argument": "{\"name\": \"Alice\", \"age\": 30}"}'
```

If `main` has no `params`, the execution argument must be null or omitted.
