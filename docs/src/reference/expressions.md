# Expression Language

Expressions are enclosed in `${}` and evaluated at runtime. They can appear anywhere a value is expected: assignments, conditions, function arguments, URLs, and more.

## Syntax

```yaml
- step:
    assign:
      - result: ${a + b}
      - greeting: '${"Hello, " + name + "!"}'
      - check: ${x > 10 and y < 20}
```

When an expression starts a YAML value, wrap the entire value in quotes to avoid YAML parsing issues with colons and special characters.

**Maximum expression length:** 400 characters.

## Data types

| Type | Examples | `type()` result | Notes |
|------|----------|-----------------|-------|
| int | `1`, `-5`, `0` | `"int"` | 64-bit signed integer |
| double | `4.1`, `-0.5`, `3.14e10` | `"double"` | 64-bit IEEE 754 |
| string | `"hello"`, `'world'` | `"string"` | Max 256 KB (UTF-8) |
| bool | `true`, `false` | `"bool"` | Also `True`/`False`, `TRUE`/`FALSE` |
| null | `null` | `"null"` | Distinct type, not zero or empty string |
| list | `[1, 2, 3]`, `[]` | `"list"` | Ordered, 0-indexed |
| map | `{"key": "value"}`, `{}` | `"map"` | String keys only |
| bytes | (no literal syntax) | `"bytes"` | Created via `text.encode()` or `base64.decode()` |

## Operators

### Arithmetic

| Operator | Description | Example | Notes |
|----------|-------------|---------|-------|
| `+` | Addition / string concat | `${a + b}`, `${"hi " + name}` | `string + int` is TypeError |
| `-` | Subtraction | `${a - b}` | |
| `*` | Multiplication | `${a * b}` | |
| `/` | Division | `${10 / 3}` -> `3.333...` | Always returns double |
| `%` | Modulo | `${10 % 3}` -> `1` | |
| `//` | Integer division | `${10 // 3}` -> `3` | Floor division (truncates toward negative infinity) |
| `-` (unary) | Negation | `${-x}` | |

**Type promotion:** When int and double are mixed, int is promoted to double. Division `/` always returns double, even `${4 / 2}` = `2.0`.

**No implicit string conversion:** `${"count: " + 42}` is a TypeError. Use `${"count: " + string(42)}`.

### Comparison

| Operator | Description |
|----------|-------------|
| `==` | Equal (deep equality for maps and lists) |
| `!=` | Not equal |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

- `null == null` is `true`
- `null == <anything_else>` is `false`
- Comparing incompatible types with `<`, `>`, `<=`, `>=` raises TypeError
- `==` and `!=` between incompatible types returns `false` (no error)
- Maps and lists use deep equality

### Logical

| Operator | Description |
|----------|-------------|
| `and` | Short-circuit AND |
| `or` | Short-circuit OR |
| `not` | Logical NOT |

Operands must be boolean. `not "hello"` or `true and 1` raises TypeError. Short-circuit: if the left operand determines the result, the right operand is not evaluated.

### Membership

| Operator | Description | Example |
|----------|-------------|---------|
| `in` | Key exists in map, or value in list | `${"key" in my_map}`, `${item in my_list}` |
| `not in` | Negation of `in` | `${"key" not in my_map}` |

### Property and index access

| Syntax | Description |
|--------|-------------|
| `obj.key` | Map property access |
| `obj["key"]` | Map key access (supports dynamic keys) |
| `list[0]` | List index access (0-based) |
| `obj.a.b[0].c` | Nested access |

- Missing map key raises KeyError
- Out-of-bounds list index raises IndexError
- Negative list indices are **not** supported (raises IndexError)
- Use `map.get(obj, "key", default)` for safe access

## Operator precedence

From highest to lowest:

1. `.`, `[]` -- property/index access
2. `not`, `-` (unary) -- negation
3. `*`, `/`, `%`, `//` -- multiplicative
4. `+`, `-` -- additive
5. `<`, `>`, `<=`, `>=` -- relational
6. `==`, `!=` -- equality
7. `in`, `not in` -- membership
8. `and` -- logical AND
9. `or` -- logical OR

Use parentheses `()` to override precedence.

## Function calls in expressions

Built-in functions and standard library functions can be called inside `${}`:

```yaml
- step:
    assign:
      - count: ${len(items)}
      - upper_name: ${text.to_upper(name)}
      - safe_value: ${default(map.get(config, "key"), "fallback")}
      - data_type: ${type(value)}
      - id: ${uuid.generate()}
```

Subworkflows can also be called with positional arguments:

```yaml
- step:
    assign:
      - result: ${my_subworkflow("arg1", "arg2")}
```

See the [Standard Library](./stdlib.md) for all available functions.

## Null handling

```yaml
# Accessing a missing map key raises KeyError -- NOT null
- bad: ${myMap.missingKey}              # KeyError!

# Safe access with map.get (returns null if not found)
- safe: ${map.get(myMap, "key")}        # null if missing

# Safe access with a default value
- safe: ${map.get(myMap, "key", 0)}     # 0 if missing

# default() handles null but does NOT catch KeyError
- val: ${default(map.get(myMap, "key"), "fallback")}

# Null comparisons
- is_null: ${value == null}             # true if value is null
- both_null: ${null == null}            # true
```

## Common patterns

### String building

```yaml
- step:
    assign:
      - url: '${"http://localhost:9090/users/" + string(user_id) + "/orders"}'
      - message: '${"Found " + string(len(items)) + " items"}'
```

### Conditional defaults

```yaml
- step:
    assign:
      - timeout: ${default(map.get(config, "timeout"), 30)}
      - name: ${default(map.get(args, "name"), "Anonymous")}
```

### Type checking

```yaml
- step:
    switch:
      - condition: ${type(value) == "list"}
        assign:
          - count: ${len(value)}
      - condition: ${type(value) == "string"}
        assign:
          - count: 1
```

### Checking for map keys

```yaml
- step:
    switch:
      - condition: ${"email" in user}
        next: send_email
      - condition: true
        next: skip_notification
```

## Edge cases

| Case | Behavior |
|------|----------|
| `${x / 0}` | ZeroDivisionError |
| `${"hi" + 5}` | TypeError (use `string(5)`) |
| `${not "hello"}` | TypeError (operand must be boolean) |
| `${myList[-1]}` | IndexError (negative indices not supported) |
| `${}` | Invalid (empty expression) |
| `${${x}}` | Invalid (nested expressions not supported) |
| `${10 / 2}` | `2.0` (division always returns double) |
| `${-10 // 3}` | `-4` (floor division, not truncation toward zero) |
| Integer overflow | Wraps (64-bit signed) |
