package integration

import (
	"testing"
)

// TestExpr_Arithmetic verifies all arithmetic operators.
func TestExpr_Arithmetic(t *testing.T) {
	yaml := loadWorkflow(t, "expr_arithmetic.yaml")
	er := deployAndRun(t, uniqueID("expr-arith"), yaml, nil)
	assertSucceeded(t, er)

	// 10 + 3 = 13
	assertResultContains(t, er, "add", float64(13))
	// 10 - 3 = 7
	assertResultContains(t, er, "sub", float64(7))
	// 10 * 3 = 30
	assertResultContains(t, er, "mul", float64(30))
	// 10 / 3 = 3.333...
	// Note: GCW integer division with / produces a double
	assertResultContains(t, er, "div", float64(10)/float64(3))
	// 10 % 3 = 1
	assertResultContains(t, er, "mod", float64(1))
	// 10 // 3 = 3 (integer division, floor)
	assertResultContains(t, er, "intdiv", float64(3))
}

// TestExpr_Comparison verifies all comparison operators.
func TestExpr_Comparison(t *testing.T) {
	yaml := loadWorkflow(t, "expr_comparison.yaml")
	er := deployAndRun(t, uniqueID("expr-comp"), yaml, nil)
	assertSucceeded(t, er)

	assertResultContains(t, er, "eq", true)
	assertResultContains(t, er, "neq", true)
	assertResultContains(t, er, "lt", true)
	assertResultContains(t, er, "gt", true)
	assertResultContains(t, er, "lte", true)
	assertResultContains(t, er, "gte", true)
}

// TestExpr_Logical verifies all logical operators.
func TestExpr_Logical(t *testing.T) {
	yaml := loadWorkflow(t, "expr_logical.yaml")
	er := deployAndRun(t, uniqueID("expr-logic"), yaml, nil)
	assertSucceeded(t, er)

	assertResultContains(t, er, "and_true", true)
	assertResultContains(t, er, "and_false", false)
	assertResultContains(t, er, "or_true", true)
	assertResultContains(t, er, "or_false", false)
	assertResultContains(t, er, "not_true", true)
	assertResultContains(t, er, "not_false", false)
}

// TestExpr_Membership verifies 'in' operator for lists and maps.
func TestExpr_Membership(t *testing.T) {
	yaml := loadWorkflow(t, "expr_membership.yaml")
	er := deployAndRun(t, uniqueID("expr-member"), yaml, nil)
	assertSucceeded(t, er)

	assertResultContains(t, er, "in_list", true)
	assertResultContains(t, er, "not_in_list", false)
	assertResultContains(t, er, "in_map", true)
	assertResultContains(t, er, "not_in_map", false)
}

// TestExpr_PropertyAccess verifies dot and bracket notation property access.
func TestExpr_PropertyAccess(t *testing.T) {
	yaml := loadWorkflow(t, "expr_property_access.yaml")
	er := deployAndRun(t, uniqueID("expr-prop"), yaml, nil)
	assertSucceeded(t, er)

	assertResultContains(t, er, "dot_access", "Alice")
	assertResultContains(t, er, "bracket_access", "Alice")
	assertResultContains(t, er, "nested_dot", "Wonderland")
	assertResultContains(t, er, "list_index", float64(85))
}

// TestExpr_NegativeNumbers verifies negative number handling.
func TestExpr_NegativeNumbers(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - neg: ${-5}
          - neg_add: ${-5 + 3}
          - neg_mul: ${-2 * -3}
    - done:
        return:
          neg: ${neg}
          neg_add: ${neg_add}
          neg_mul: ${neg_mul}
`
	er := deployAndRun(t, uniqueID("expr-neg"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "neg", float64(-5))
	assertResultContains(t, er, "neg_add", float64(-2))
	assertResultContains(t, er, "neg_mul", float64(6))
}

// TestExpr_StringConcatWithNumbers verifies string + int concatenation.
func TestExpr_StringConcatWithNumbers(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - result: '${"count " + string(42)}'
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("expr-str-num"), yaml, nil)
	assertResultEquals(t, er, "count 42")
}

// TestExpr_TypeConversion verifies type conversion functions.
func TestExpr_TypeConversion(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - str_to_int: ${int("42")}
          - str_to_double: ${double("3.14")}
          - int_to_str: ${string(42)}
          - double_to_int: ${int(3.99)}
          - bool_to_str: ${string(true)}
    - done:
        return:
          str_to_int: ${str_to_int}
          str_to_double: ${str_to_double}
          int_to_str: ${int_to_str}
          double_to_int: ${double_to_int}
          bool_to_str: ${bool_to_str}
`
	er := deployAndRun(t, uniqueID("expr-convert"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "str_to_int", float64(42))
	assertResultContains(t, er, "str_to_double", 3.14)
	assertResultContains(t, er, "int_to_str", "42")
	// int() truncates toward zero
	assertResultContains(t, er, "double_to_int", float64(3))
	assertResultContains(t, er, "bool_to_str", "true")
}

// TestExpr_NullHandling verifies null comparisons and default().
func TestExpr_NullHandling(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: null
    - compute:
        assign:
          - is_null: ${x == null}
          - with_default: ${default(x, "fallback")}
          - type_of_null: ${type(x)}
    - done:
        return:
          is_null: ${is_null}
          with_default: ${with_default}
          type_of_null: ${type_of_null}
`
	er := deployAndRun(t, uniqueID("expr-null"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "is_null", true)
	assertResultContains(t, er, "with_default", "fallback")
	assertResultContains(t, er, "type_of_null", "null")
}

// TestExpr_TypeFunction verifies the type() function for all types.
func TestExpr_TypeFunction(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - t_int: ${type(42)}
          - t_double: ${type(3.14)}
          - t_string: ${type("hello")}
          - t_bool: ${type(true)}
          - t_null: ${type(null)}
          - t_list: ${type([1,2])}
          - mymap:
              a: 1
          - t_map: ${type(mymap)}
    - done:
        return:
          int: ${t_int}
          double: ${t_double}
          string: ${t_string}
          bool: ${t_bool}
          "null": ${t_null}
          list: ${t_list}
          map: ${t_map}
`
	er := deployAndRun(t, uniqueID("expr-type"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "int", "int")
	assertResultContains(t, er, "double", "double")
	assertResultContains(t, er, "string", "string")
	assertResultContains(t, er, "bool", "bool")
	assertResultContains(t, er, "null", "null")
	assertResultContains(t, er, "list", "list")
	assertResultContains(t, er, "map", "map")
}

// TestExpr_LenFunction verifies the len() function for strings, lists, and maps.
func TestExpr_LenFunction(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - str_len: ${len("hello")}
          - list_len: ${len([1, 2, 3])}
          - mymap:
              a: 1
              b: 2
          - map_len: ${len(mymap)}
          - empty_str: ${len("")}
          - empty_list: ${len([])}
    - done:
        return:
          str_len: ${str_len}
          list_len: ${list_len}
          map_len: ${map_len}
          empty_str: ${empty_str}
          empty_list: ${empty_list}
`
	er := deployAndRun(t, uniqueID("expr-len"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "str_len", float64(5))
	assertResultContains(t, er, "list_len", float64(3))
	assertResultContains(t, er, "map_len", float64(2))
	assertResultContains(t, er, "empty_str", float64(0))
	assertResultContains(t, er, "empty_list", float64(0))
}

// TestExpr_KeysFunction verifies the keys() function on maps.
func TestExpr_KeysFunction(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - m:
              z: 1
              a: 2
              m: 3
    - compute:
        assign:
          - k: ${keys(m)}
    - done:
        return: ${len(k)}
`
	er := deployAndRun(t, uniqueID("expr-keys"), yaml, nil)
	assertSucceeded(t, er)
	// keys() should return 3 keys
	assertResultEquals(t, er, float64(3))
}

// TestExpr_DefaultFunctionWithNonNull verifies default() returns value when not null.
func TestExpr_DefaultFunctionWithNonNull(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 42
    - done:
        return: ${default(x, 99)}
`
	er := deployAndRun(t, uniqueID("expr-default"), yaml, nil)
	assertResultEquals(t, er, float64(42))
}

// TestExpr_NestedExpressions verifies nested function calls in expressions.
func TestExpr_NestedExpressions(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - result: ${string(int("42") + len("hello"))}
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("expr-nested"), yaml, nil)
	assertResultEquals(t, er, "47")
}

// TestExpr_ZeroDivision verifies that division by zero raises ZeroDivisionError.
func TestExpr_ZeroDivision(t *testing.T) {
	yaml := `
main:
  steps:
    - try_div:
        try:
          steps:
            - divide:
                assign:
                  - x: ${1 / 0}
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  tags: ${e.tags}
`
	er := deployAndRun(t, uniqueID("expr-zero-div"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
}
