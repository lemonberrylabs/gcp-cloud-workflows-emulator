# Parallel Execution - Complete Reference

## Overview

Parallel steps enable concurrent execution of either:
1. **Branches**: Multiple independent step sequences
2. **Parallel for**: Same operations applied to each item in a list concurrently

## Syntax

### Parallel Branches

```yaml
- parallel_step:
    parallel:
      shared: [var1, var2]
      concurrency_limit: 5
      exception_policy: continueAll
      branches:
        - branch_name_1:
            steps:
              - step1:
                  call: http.get
                  args:
                    url: https://api.com/endpoint1
                  result: var1
        - branch_name_2:
            steps:
              - step2:
                  call: http.get
                  args:
                    url: https://api.com/endpoint2
                  result: var2
```

### Parallel For Loop

```yaml
- parallel_loop:
    parallel:
      shared: [total]
      concurrency_limit: 10
      for:
        value: item
        index: i
        in: ${items}
        steps:
          - process:
              assign:
                - total: ${total + 1}
```

Parallel for also supports `range`:
```yaml
- parallel_range:
    parallel:
      shared: [results]
      for:
        value: v
        range: [0, 9]
        steps:
          - process:
              assign:
                - results: ${list.concat(results, v * 2)}
```

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `shared` | list of strings | Yes (if writing to parent vars) | Variables with parent scope writable by branches |
| `concurrency_limit` | int or expression | No | Max concurrent branches/iterations |
| `exception_policy` | string | No | Error handling policy |
| `branches` | list of branch objects | Yes (for branch mode) | Named branch definitions |
| `for` | for config | Yes (for parallel for mode) | Loop configuration |

`branches` and `for` are mutually exclusive.

## Shared Variables

### Declaration
- Variables listed in `shared` MUST exist in the parent scope before the parallel step
- They must be explicitly declared; only listed variables are writable
- Variables not in `shared` are read-only copies within each branch

### Atomicity
- All variable reads and writes to shared variables are **atomic**
- Individual read or write operations are guaranteed to be atomic
- However, compound operations like `total: ${total + 1}` are NOT atomic as a unit:
  - The read of `total`, addition, and write are separate atomic operations
  - Race conditions CAN occur between the read and write
- No mutex/lock mechanism is provided

### Visibility
- Writes to shared variables are **immediately visible** to other branches
- There is no eventual consistency; writes are seen right away
- But the order of execution across branches is non-deterministic

### Scope
- Each branch has its own local scope for non-shared variables
- Variables created within a branch do NOT exist after the parallel step
- The `result` variable from a call step inside a branch is local to that branch unless it is in `shared`

## Concurrency Limit

- Optional positive integer (literal or expression)
- Controls maximum number of concurrently executing branches or iterations
- **Applies to a single parallel step only** (does not cascade to nested parallel steps)
- Excess branches/iterations are queued and executed as slots become available
- Default: up to 20 concurrent (the system limit)

## Exception Policies

| Policy | Behavior |
|--------|----------|
| (default / not set) | Abort on first unhandled exception; cancel other branches |
| `continueAll` | Other branches continue executing; unhandled exceptions are collected |

### continueAll Details
- When a branch throws an unhandled exception, other branches keep running
- Up to **100 unhandled exceptions** can be collected per execution
- After all branches complete, if any unhandled exceptions exist, the parallel step raises an error containing all collected exceptions
- The error can be caught by a try/except wrapping the parallel step

### Default Policy Details
- First unhandled exception in any branch causes:
  - Other branches to be cancelled
  - The exception to propagate up from the parallel step

## Limits

| Limit | Value |
|-------|-------|
| Branches per parallel step | **10** |
| Maximum concurrent branches/iterations | **20** |
| Parallel nesting depth | **2** (parallel within parallel) |
| Uncaught exceptions per execution | **100** |

## Nesting

- Parallel steps can contain nested parallel steps up to depth **2**
- Each nested parallel step has its own `shared` declaration
- The nested parallel step's `shared` variables must be in the parent parallel step's scope

## Execution Order

- Branches and iterations execute in **any order**
- The order may differ between executions
- No guarantees about scheduling or ordering
- For parallel for: results are NOT guaranteed to be in input order (unlike `experimental.executions.map`)

## Performance Considerations

- **Blocking calls benefit most**: HTTP requests, sleep, callbacks
- **Non-blocking operations gain no advantage**: assignments, conditions, expressions
- System overhead can increase total time for workflows without blocking calls
- Example: 5 independent API calls each taking 2 seconds:
  - Sequential: ~10 seconds
  - Parallel: ~2 seconds

## Error Handling in Parallel Steps

### Try/except within a branch
```yaml
- parallel_step:
    parallel:
      shared: [results]
      branches:
        - branch_a:
            steps:
              - try_call:
                  try:
                    call: http.get
                    args:
                      url: https://unstable-api.com
                    result: response
                  except:
                    as: e
                    steps:
                      - handle:
                          assign:
                            - results: ${list.concat(results, "branch_a failed")}
```

### Try/except wrapping the parallel step
```yaml
- handle_parallel:
    try:
      steps:
        - parallel_step:
            parallel:
              exception_policy: continueAll
              shared: [results]
              branches:
                - branch_a:
                    steps:
                      - risky_call:
                          call: http.get
                          args:
                            url: https://may-fail.com
    except:
      as: e
      steps:
        - log_error:
            call: sys.log
            args:
              data: ${e}
```

## Parallel Callbacks

- Callback endpoints can be created in the parent scope and awaited in parallel branches
- Each `await_callback` within a parallel step opens a new callback slot
- Multiple callbacks can be awaited concurrently across branches
- Callbacks can only be awaited in the same execution where they were created
