# Edge Cases and Gotchas - Complete Reference

## Known Issues (from official Google Cloud documentation)

### 1. For Loop After Try Block
**Issue**: Placing a `for` loop directly after a `try` block causes deployment failure.
**Error**: `"loop step name should not be empty (Code: 3)"`
**Workaround**: Wrap the `for` loop in a named step within the `try` block structure.
**Emulator implication**: Consider whether to replicate this parser bug or just handle it correctly.

### 2. HTTP Request Timeout with Cloud Build
**Issue**: Workflows calling Cloud Build may fail with `"HTTP request lost"` and `"RPC connection timed out"`.
**Workaround**: Implement retry policy or try/except.
**Emulator implication**: Not directly relevant (no Cloud Build integration).

### 3. Secret Data in Logs
**Issue**: When call logging is `log-all-calls`, secret values from Secret Manager appear unredacted in logs.
**Emulator implication**: If implementing logging, be aware of this behavior.

### 4. GKE IP-Based Endpoints
**Issue**: HTTP requests to IP-based GKE cluster control planes are not supported.
**Emulator implication**: Not relevant for local emulator.

## YAML Parsing Gotchas

### Colons in Expressions
**Issue**: `${}` expression syntax is non-standard YAML. Colons inside expressions confuse YAML parsers.
```yaml
# This may fail:
- step:
    assign:
      - x: ${myMap: "value"}

# Fix: wrap in quotes:
- step:
    assign:
      - x: '${myMap["key"]}'
```
**Emulator implication**: The YAML parser must handle `${}` expressions that contain colons. Using a standard YAML parser and then post-processing expressions is recommended.

### Boolean Literals
**Issue**: YAML auto-converts `true`/`false`, `True`/`False`, `TRUE`/`FALSE`, `yes`/`no`, `on`/`off` to booleans.
**Emulator implication**: The parser must handle all YAML boolean representations correctly. `yes` and `on` should NOT be treated as booleans in the GCW expression context - only `true`/`false` variants.

### Null Literals
**Issue**: YAML treats `null`, `Null`, `NULL`, `~` as null values.
**Emulator implication**: All should map to GCW `null`.

## Expression Edge Cases

### Division Operators
```yaml
# Regular division always returns double
- x: ${10 / 3}     # 3.3333...
- y: ${10 / 2}     # 5.0 (NOT 5)

# Integer division truncates toward negative infinity
- z: ${10 // 3}    # 3
- w: ${-10 // 3}   # -4 (floor division, NOT -3)

# Modulo
- m: ${10 % 3}     # 1
```

### String Concatenation
```yaml
# MUST explicitly convert non-strings
- ok: ${"count: " + string(42)}       # Works
- fail: ${"count: " + 42}             # TypeError!

# Boolean to string
- ok: ${"active: " + string(true)}    # Works
- fail: ${"active: " + true}          # TypeError!
```

### Null Comparisons
```yaml
# Equality
- a: ${null == null}                   # true
- b: ${null == 0}                      # false
- c: ${null == ""}                     # false
- d: ${null == false}                  # false

# Ordering - behavior may vary
- e: ${null < 1}                       # Likely TypeError
```

### Empty Collections
```yaml
- empty_list: []
- empty_map: {}
- len_list: ${len([])}                 # 0
- len_map: ${len({})}                  # 0
- in_empty: ${"x" in {}}              # false
- in_empty_list: ${1 in []}           # false
```

### Negative List Indices
```yaml
# Negative indices are NOT supported
- x: ${myList[-1]}                     # IndexError (NOT last element)
```

### Map Key Types
```yaml
# Keys are always strings in GCW
- m: {1: "one"}    # Key "1" is a string (YAML may parse numeric keys)
# Accessing: ${m["1"]} or ${m[string(1)]}
```

## Variable Scoping Edge Cases

### For Loop Scope
```yaml
- init:
    assign:
      - total: 0
- loop:
    for:
      value: item
      in: [1, 2, 3]
      steps:
        - add:
            assign:
              - total: ${total + item}    # Modifies parent-scope variable
              - temp: ${item * 2}         # Creates loop-local variable
- after_loop:
    assign:
      - result: ${total}                  # Works: total = 6
      # - bad: ${temp}                    # KeyError! temp doesn't exist here
      # - bad: ${item}                    # KeyError! item doesn't exist here
```

### Except Block Scope
```yaml
# Variable declared inside except is NOT visible outside
- handle:
    try:
      steps:
        - risky:
            call: http.get
            args:
              url: https://may-fail.com
            result: response
    except:
      as: e
      steps:
        - save_error:
            assign:
              - error_info: ${e.message}   # Only available inside except
# - use_error:
#     assign:
#       - x: ${error_info}                # KeyError! Not in scope

# Fix: declare variable before try/except
- init:
    assign:
      - error_info: null
- handle:
    try:
      # ...
    except:
      as: e
      steps:
        - save:
            assign:
              - error_info: ${e.message}   # Now modifies parent-scope var
- use:
    assign:
      - x: ${error_info}                  # Works
```

### Parallel Branch Scope
```yaml
- parallel_step:
    parallel:
      shared: [result]
      branches:
        - branch_a:
            steps:
              - work:
                  assign:
                    - local_var: "only in branch_a"  # NOT visible outside
                    - result: "from a"               # Visible (shared)
# After parallel: local_var does NOT exist, result = whatever branch wrote last
```

## HTTP Request Edge Cases

### Auto Content-Type
```yaml
# If body is provided but no Content-Type header:
# -> Content-Type: application/json; charset=utf-8 (for non-bytes)
# -> Content-Type: application/octet-stream (for bytes)

# If body is empty/null: no Content-Type added
```

### Response Body Parsing
```yaml
# application/json -> auto-parsed to map/list
# text/* -> string
# Everything else -> bytes
# Malformed JSON with application/json Content-Type -> may return raw string or error
```

### User-Agent Header
```yaml
# Whatever you set gets APPENDED to, not replaced:
# Your value + " GoogleCloudWorkflows; (+https://cloud.google.com/workflows/docs)"
```

### Query Parameter Encoding
```yaml
# Query parameters are URL-encoded and appended to URL
- step:
    call: http.get
    args:
      url: https://api.com/search
      query:
        q: "hello world"  # Becomes ?q=hello%20world
```

## Retry Edge Cases

### Retry Counter
```yaml
# max_retries does NOT include the initial attempt
# max_retries: 3 means: 1 initial attempt + 3 retries = 4 total attempts
```

### Backoff Calculation
```yaml
# Delay for attempt N (0-indexed retries):
# delay = min(initial_delay * multiplier^N, max_delay)
# Plus jitter (implementation detail)
```

### Retry with Multiple Steps
```yaml
# When retry is configured and the try block has multiple steps:
# The ENTIRE try block is re-executed from the beginning, not just the failed step
```

### Custom Predicate Error
```yaml
# If the custom predicate subworkflow itself throws an error:
# The retry is aborted and the original error propagates
```

## Parallel Execution Edge Cases

### Race Conditions on Shared Variables
```yaml
# This is NOT safe for accumulation:
- parallel:
    shared: [counter]
    for:
      value: item
      in: ${items}
      steps:
        - increment:
            assign:
              - counter: ${counter + 1}
# counter may be less than len(items) due to read-modify-write race conditions
# Read and write are individually atomic, but the compound operation is not
```

### Parallel Branch Ordering
```yaml
# Branch execution order is non-deterministic
# Even with the same input, branches may execute in different orders
# Results from parallel for are NOT guaranteed to be in input order
```

### Exception Collection Limit
```yaml
# With continueAll policy:
# If more than 100 unhandled exceptions occur, behavior is undefined
# The 101st exception may be dropped or cause a system error
```

## Subworkflow Edge Cases

### Max Call Depth
```yaml
# Maximum 20 levels of nested calls
# Recursive subworkflows that exceed this raise RecursionError
# This includes all call chain depth, not just direct recursion
```

### Parameter Passing
```yaml
# Named parameters (call step):
- step:
    call: my_sub
    args:
      param1: "value1"    # Named arguments

# Positional parameters (expression):
- step:
    assign:
      - result: ${my_sub("value1", "value2")}  # Positional arguments
```

### Main Workflow Parameters
```yaml
# main accepts a SINGLE parameter which must be a map (dict)
main:
  params: [args]          # args receives the JSON execution argument as a map
  steps:
    - use:
        assign:
          - x: ${args.key}

# Subworkflows can accept multiple named parameters with defaults
my_sub:
  params: [required_param, optional_param: "default_value"]
```

### No Parameters
```yaml
# If main has no params, execution argument must be null/empty
# If main has params but no argument is passed, params receives null
```

## Assignment Edge Cases

### Sequential Evaluation
```yaml
# Assignments within an assign step are sequential:
- step:
    assign:
      - x: 1
      - y: ${x + 1}     # y = 2 (x is already 1)
      - x: ${x + 10}    # x = 11 (uses current x = 1)
```

### Creating Nested Structures
```yaml
# Assigning to a nested map path creates intermediate maps:
- step:
    assign:
      - myMap: {}
      - myMap.a.b.c: "deep"  # Creates myMap = {a: {b: {c: "deep"}}}

# But list indices must already exist:
- step:
    assign:
      - myList: []
      - myList[0]: "first"   # MAY raise IndexError (list is empty)
```

## Workflow Definition Edge Cases

### Step Names
```yaml
# Step names must be unique within their scope (sibling steps)
# Steps in different branches/subworkflows can share names
# Step names are used as targets for `next` jumps
```

### Duplicate Subworkflow Names
```yaml
# Cannot define two subworkflows with the same name in the same workflow file
# Results in deployment/validation error
```

### Empty Workflow
```yaml
# Minimum 1 step per workflow
# An empty steps list is invalid
```

### Source Code Size
```yaml
# Maximum 128 KB for the entire workflow definition
# Includes all subworkflows, comments, whitespace
# Measured as UTF-8 encoded YAML/JSON
```
