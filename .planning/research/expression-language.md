# Expression Language - Complete Reference

## Overview

Expressions in Google Cloud Workflows are wrapped in `${}` delimiters. They can appear in:
- Assignment values: `- x: ${1 + 2}`
- Condition predicates: `condition: ${x > 5}`
- String interpolation within `${}`: `${"Hello " + name}`
- Function arguments: `url: ${"https://api.com/" + id}`
- Any field that accepts a value

## Expression Syntax

### Delimiters
- Expressions MUST be wrapped in `${...}`
- The entire value must be a single expression (no mixing literal text and expressions outside `${}`)
- Inside `${}`, string concatenation achieves interpolation: `${"Hello " + name + "!"}`

### Maximum Length
- **400 characters** maximum for any single expression

### YAML Gotcha
- Colons inside `${}` can confuse YAML parsers (interpreted as key-value separators)
- Best practice: wrap expressions containing colons in quotes: `'${...}'`

## Operators

### Arithmetic Operators
| Operator | Description | Operand Types | Result Type |
|----------|-------------|---------------|-------------|
| `+` | Addition / String concatenation | int+int, double+double, int+double, string+string | int, double, double, string |
| `-` | Subtraction | int, double | Same as operands (double if mixed) |
| `*` | Multiplication | int, double | Same as operands (double if mixed) |
| `/` | Division | int, double | double (always floating-point) |
| `%` | Modulo | int, double | Same as operands |
| `//` | Integer division (floor division) | int, double | int (truncates toward zero) |
| `-` (unary) | Negation | int, double | Same as operand |

### Type Promotion Rules for Arithmetic
- When an `int` and `double` are combined, the `int` is promoted to `double`
- Division `/` always returns `double` (even `4 / 2` = `2.0`)
- Integer division `//` always returns `int` (truncates toward negative infinity)
- String `+` string performs concatenation
- String `+` non-string is a **TypeError** (must explicitly convert with `string()`)

### Comparison Operators
| Operator | Description |
|----------|-------------|
| `==` | Equal |
| `!=` | Not equal |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

- Comparison between incompatible types (e.g., string vs int) raises **TypeError**
- `null == null` is `true`
- `null` compared with any non-null value is `false` for `==`

### Logical Operators
| Operator | Description |
|----------|-------------|
| `and` | Logical AND (short-circuit) |
| `or` | Logical OR (short-circuit) |
| `not` | Logical NOT |

- `and` and `or` use **short-circuit evaluation**: if the left operand determines the result, the right operand is NOT evaluated
- Operands must be boolean; non-boolean operands raise **TypeError**

### Membership Operators
| Operator | Description |
|----------|-------------|
| `in` | Membership test |
| `not in` | Negative membership test |

- For **maps**: `"key" in myMap` tests if the key exists
- For **lists**: `value in myList` tests if the value is an element
- Returns `true` / `false`

### Property/Index Access
| Operator | Description |
|----------|-------------|
| `.` | Map property access: `myMap.key` |
| `[]` | Index/key access: `myList[0]`, `myMap["key"]`, `myMap[varName]` |

- Accessing a non-existent map key via `.` or `[]` raises **KeyError**
- Accessing an out-of-bounds list index raises **IndexError**
- Negative indices are NOT supported (raises IndexError)
- Nested access is supported: `myMap.nested.key`, `myList[0][1]`
- Dynamic key access: `myMap[keyVariable]`, `myList[indexExpression]`

## Operator Precedence (highest to lowest)

1. `.`, `[]` (property/index access)
2. `not`, `-` (unary negation)
3. `*`, `/`, `%`, `//` (multiplicative)
4. `+`, `-` (additive)
5. `<`, `>`, `<=`, `>=` (relational)
6. `==`, `!=` (equality)
7. `in`, `not in` (membership)
8. `and` (logical AND)
9. `or` (logical OR)

Parentheses `()` can be used to override precedence.

## Function Calls in Expressions

Functions can be called directly in expressions:
```yaml
- x: ${len(myList)}
- y: ${int("42")}
- z: ${text.to_upper(name)}
- w: ${map.get(myMap, "key", "default")}
```

Subworkflows can also be called as functions in expressions:
```yaml
- result: ${mySubworkflow("arg1", "arg2")}
```

When calling subworkflows as functions in expressions, arguments are passed positionally (not by name).

## Escaping

- To include a literal `${` in a string, there is no standard escape mechanism documented
- String literals within expressions use standard escaping: `\"`, `\\`, `\n`, `\t`
- Unicode escape sequences are supported in strings

## Type Coercion in Expressions

Google Cloud Workflows does **NOT** perform implicit type coercion in most cases:
- `"hello" + 5` is a **TypeError** (must use `"hello" + string(5)`)
- `true + 1` is a **TypeError**
- Arithmetic between `int` and `double` promotes `int` to `double` (this is the one exception)

## Edge Cases

1. **Division by zero**: `x / 0` or `x // 0` or `x % 0` raises **ZeroDivisionError**
2. **Integer overflow**: 64-bit signed integers; overflow behavior wraps (follows Go/Java semantics)
3. **Floating-point special values**: `double` follows IEEE 754; infinity and NaN may occur
4. **Empty expressions**: `${}` is invalid
5. **Nested expressions**: `${${x}}` is NOT valid; expressions cannot be nested
6. **Multi-line expressions**: Expressions CAN span multiple lines (useful for long SQL queries in BigQuery calls)
7. **Boolean literals**: `true`/`false`, `True`/`False`, `TRUE`/`FALSE` are all valid
8. **Null literal**: `null` is the only null value
9. **String concatenation with `+`**: Only works between two strings; no automatic conversion
10. **Map/List equality**: Maps and lists compared with `==` perform deep equality comparison
