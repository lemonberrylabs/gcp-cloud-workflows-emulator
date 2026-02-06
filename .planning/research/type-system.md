# Type System - Complete Reference

## Data Types

Google Cloud Workflows supports 8 data types:

### int (Integer)
- **Size**: 64-bit signed integer
- **Range**: -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807
- **Literals**: `1`, `-5`, `0`, `42`
- **Conversion**: `int(value)` from string or double
  - From double: truncates toward zero (`int(2.7)` = 2, `int(-2.7)` = -2)
  - From string: parses decimal (`int("42")` = 42)
- **Type name**: `"int"` (from `type()` function)

### double (Floating-point)
- **Size**: 64-bit IEEE 754 double-precision
- **Literals**: `4.1`, `-0.5`, `1.0`, `3.14e10`
- **Conversion**: `double(value)` from string or int
  - From int: exact for values within double precision
  - From string: parses decimal notation (`double("2.7")` = 2.7)
- **Special values**: Infinity, -Infinity, NaN may occur from arithmetic
- **Type name**: `"double"` (from `type()` function)

### string
- **Encoding**: Unicode (UTF-8)
- **Max length**: 256 KB (UTF-8 encoded size)
- **Literals**: `"hello"`, `'hello'` (both single and double quotes in YAML)
- **Escaping**: Standard escape sequences: `\"`, `\\`, `\n`, `\t`, `\r`
- **Concatenation**: `+` operator (only string + string; no auto-coercion)
- **Conversion**: `string(value)` from int, double, or bool
- **Type name**: `"string"` (from `type()` function)
- **Multi-line**: Supported via YAML multi-line string syntax (folded `>`, literal `|`)

### bool (Boolean)
- **Values**: `true`, `false` (also `True`, `False`, `TRUE`, `FALSE` in YAML)
- **Conversion**: `bool(value)` from string ("true"/"false")
- **Logical operators**: `and`, `or`, `not` (short-circuit evaluation)
- **Type name**: `"bool"` (from `type()` function)

### null
- **Value**: `null` (single value of null type)
- **Behavior**:
  - `null == null` is `true`
  - `null` compared with any non-null value using `==` is `false`
  - `null` is falsy in boolean context (but using null in boolean operators may raise TypeError)
  - `default(null, "fallback")` returns `"fallback"`
- **Type name**: `"null"` (from `type()` function)
- **Assignment**: Variables can be set to `null` to free memory

### list (Array)
- **Literals**: `[1, 2, 3]`, `["a", "b"]`, `[]`
- **Indexing**: 0-based, non-negative integers only
  - `myList[0]` - first element
  - `myList[len(myList) - 1]` - last element
  - Negative indices NOT supported (raises IndexError)
- **Nested**: `[[1, 2], [3, 4]]`, access via `myList[0][1]`
- **Mutation**:
  - By index: `myList[0]: "new_value"` (in assign step)
  - By expression: `myList[i + 1]: "value"`
  - Append: `myList: ${list.concat(myList, newItem)}` (creates new list)
  - Prepend: `myList: ${list.prepend(myList, newItem)}` (creates new list)
- **Membership**: `value in myList` returns boolean
- **Length**: `len(myList)` returns int
- **Type name**: `"list"` (from `type()` function)
- **Empty list**: `[]` - valid, length 0

### map (Dictionary/Object)
- **Literals**: `{"key": "value"}` or YAML map syntax
- **Keys**: Always strings
- **Access**:
  - Dot notation: `myMap.key`
  - Bracket notation: `myMap["key"]`, `myMap[keyVariable]`
  - Both raise **KeyError** for missing keys
- **Safe access**: `map.get(myMap, "key", defaultValue)` - returns default if key missing
- **Nested**: `myMap.nested.deep.key`, `myMap["nested"]["deep"]["key"]`
- **Mutation**:
  - Dot assignment: `myMap.newKey: "value"` (in assign step)
  - Bracket assignment: `myMap["newKey"]: "value"`
  - Dynamic key: `myMap[expression]: "value"`
  - Creates intermediate maps for nested assignment
- **Special characters in keys**: Use bracket notation: `myMap["special!key"]`
- **Key existence**: `"key" in myMap` returns boolean
- **Key listing**: `keys(myMap)` returns list of strings
- **Length**: `len(myMap)` returns count of key-value pairs
- **Merge**: `map.merge(map1, map2)` - shallow merge, map2 overrides
- **Deep merge**: `map.merge_nested(map1, map2)` - recursive merge
- **Delete key**: `map.delete(myMap, "key")` - returns new map
- **Type name**: `"map"` (from `type()` function)
- **Empty map**: `{}` - valid, length 0

### bytes
- **Creation**: From `text.encode(string)`, `base64.decode(string)`, `json.encode(value)`
- **Conversion**: `text.decode(bytes)` to string, `base64.encode(bytes)` to base64 string
- **Usage**: Binary data for HTTP request/response bodies, hash functions
- **Type name**: `"bytes"` (from `type()` function)
- **Not directly constructible**: No bytes literal syntax

## Type Coercion Rules

### Implicit Coercion (automatic)
- **int + double arithmetic**: int promoted to double, result is double
- **HTTP response body**: JSON strings auto-parsed to maps/lists if Content-Type is application/json

### No Implicit Coercion (explicit conversion required)
- string + int: **TypeError** (use `string(intValue)`)
- string + double: **TypeError** (use `string(doubleValue)`)
- string + bool: **TypeError** (use `string(boolValue)`)
- bool in arithmetic: **TypeError**
- null in arithmetic: **TypeError**
- Comparing different types (string vs int): **TypeError** for ordering operators

### Explicit Conversion Functions
| Function | From Types | To Type |
|----------|-----------|---------|
| `int()` | string, double | int |
| `double()` | string, int | double |
| `string()` | int, double, bool | string |
| `bool()` | string | bool |

## Null Handling

- Accessing a missing map key: **KeyError** (not null)
- `map.get(m, "missing_key")`: returns `null` (not an error)
- `map.get(m, "missing_key", default)`: returns `default`
- `default(null, fallback)`: returns `fallback`
- `default(non_null_value, fallback)`: returns `non_null_value`
- Assigning a variable to `null`: frees the memory for that variable's previous value
- `null` comparisons: `null == null` is true, `null == anything_else` is false

## Memory and Size Limits

| Resource | Limit |
|----------|-------|
| Total variable memory (all variables, arguments, events) | 512 KB |
| Maximum string length (UTF-8) | 256 KB |
| HTTP response size | 2 MB |
| Execution argument size | 32 KB |

### Memory Management Tips
- Assign unused variables to `null` to free memory
- Only store essential portions of API responses
- Filter API results at the source when possible
- Delegate large data processing to external functions

## Equality and Comparison

### Deep Equality
- Maps compared with `==` perform deep equality (all keys and values must match)
- Lists compared with `==` perform deep equality (all elements must match at same indices)
- Order of map keys does not matter for equality

### Type Compatibility for Comparison
- int and double can be compared (int promoted to double)
- Comparing incompatible types with `<`, `>`, `<=`, `>=` raises TypeError
- `==` and `!=` between incompatible types returns false (no error)

## JSON Serialization

All GCW types map to JSON:
| GCW Type | JSON Type |
|----------|-----------|
| int | number |
| double | number |
| string | string |
| bool | boolean |
| null | null |
| list | array |
| map | object |
| bytes | base64-encoded string |
