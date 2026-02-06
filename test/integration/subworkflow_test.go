package integration

import (
	"testing"
)

// TestSubworkflow_BasicCall verifies calling a subworkflow with params and defaults.
func TestSubworkflow_BasicCall(t *testing.T) {
	yaml := loadWorkflow(t, "subworkflow_basic.yaml")
	er := deployAndRun(t, uniqueID("sub-basic"), yaml, nil)
	// greet("Ada") with last_name default "Unknown"
	assertResultEquals(t, er, "Hello Ada Unknown")
}

// TestSubworkflow_WithAllParams verifies calling a subworkflow with all params specified.
func TestSubworkflow_WithAllParams(t *testing.T) {
	yaml := `
main:
  steps:
    - call_sub:
        call: greet
        args:
          first_name: "Ada"
          last_name: "Lovelace"
        result: message
    - done:
        return: ${message}

greet:
  params: [first_name, last_name: "Unknown"]
  steps:
    - build:
        return: ${"Hello " + first_name + " " + last_name}
`
	er := deployAndRun(t, uniqueID("sub-allparams"), yaml, nil)
	assertResultEquals(t, er, "Hello Ada Lovelace")
}

// TestSubworkflow_Nested verifies nested subworkflow calls.
func TestSubworkflow_Nested(t *testing.T) {
	yaml := loadWorkflow(t, "subworkflow_nested.yaml")
	er := deployAndRun(t, uniqueID("sub-nested"), yaml, nil)
	// outer(5) -> inner(5*2=10) -> 10+10=20 -> 20+1=21
	assertResultEquals(t, er, float64(21))
}

// TestSubworkflow_Recursion verifies recursive subworkflow calls.
func TestSubworkflow_Recursion(t *testing.T) {
	yaml := loadWorkflow(t, "subworkflow_recursion.yaml")
	er := deployAndRun(t, uniqueID("sub-recurse"), yaml, nil)
	// factorial(5) = 120
	assertResultEquals(t, er, float64(120))
}

// TestSubworkflow_RecursionOverflow verifies that exceeding max call stack
// depth (20) raises RecursionError.
func TestSubworkflow_RecursionOverflow(t *testing.T) {
	yaml := loadWorkflow(t, "subworkflow_recursion_overflow.yaml")
	er := deployAndRunExpectError(t, uniqueID("sub-overflow"), yaml, nil)
	assertFailed(t, er)
}

// TestSubworkflow_ReturnValue verifies that subworkflow return values are
// properly captured in the caller's result variable.
func TestSubworkflow_ReturnValue(t *testing.T) {
	yaml := `
main:
  steps:
    - get_data:
        call: make_data
        result: data
    - done:
        return:
          name: ${data.name}
          count: ${data.count}

make_data:
  steps:
    - build:
        return:
          name: "test"
          count: 42
`
	er := deployAndRun(t, uniqueID("sub-return"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "name", "test")
	assertResultContains(t, er, "count", float64(42))
}

// TestSubworkflow_VariableIsolation verifies that subworkflow variables
// do not leak into caller scope.
func TestSubworkflow_VariableIsolation(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: "main_value"
    - call_sub:
        call: modify
        result: sub_result
    - check:
        return:
          x: ${x}
          sub_result: ${sub_result}

modify:
  steps:
    - set:
        assign:
          - x: "sub_value"
          - local_var: "only_in_sub"
    - done:
        return: ${x}
`
	er := deployAndRun(t, uniqueID("sub-isolation"), yaml, nil)
	assertSucceeded(t, er)
	// x in main should remain "main_value"
	assertResultContains(t, er, "x", "main_value")
	// sub_result should be "sub_value" (what the subworkflow returned)
	assertResultContains(t, er, "sub_result", "sub_value")
}

// TestSubworkflow_MultipleReturns verifies that a subworkflow can have
// conditional returns.
func TestSubworkflow_MultipleReturns(t *testing.T) {
	yaml := `
main:
  steps:
    - call_classify:
        call: classify
        args:
          value: 50
        result: label
    - done:
        return: ${label}

classify:
  params: [value]
  steps:
    - check:
        switch:
          - condition: ${value < 0}
            return: "negative"
          - condition: ${value == 0}
            return: "zero"
          - condition: ${value < 100}
            return: "small"
          - condition: true
            return: "large"
`
	er := deployAndRun(t, uniqueID("sub-multi-ret"), yaml, nil)
	assertResultEquals(t, er, "small")
}

// TestSubworkflow_NoParams verifies calling a subworkflow with no params.
func TestSubworkflow_NoParams(t *testing.T) {
	yaml := `
main:
  steps:
    - call_sub:
        call: get_value
        result: val
    - done:
        return: ${val}

get_value:
  steps:
    - done:
        return: 42
`
	er := deployAndRun(t, uniqueID("sub-no-params"), yaml, nil)
	assertResultEquals(t, er, float64(42))
}

// TestSubworkflow_RecursionDepth20 verifies that recursion up to depth 20 works.
func TestSubworkflow_RecursionDepth20(t *testing.T) {
	yaml := `
main:
  steps:
    - call_deep:
        call: count_down
        args:
          n: 20
        result: val
    - done:
        return: ${val}

count_down:
  params: [n]
  steps:
    - base:
        switch:
          - condition: ${n <= 1}
            return: 1
    - recurse:
        call: count_down
        args:
          n: ${n - 1}
        result: sub
    - done:
        return: ${sub + 1}
`
	er := deployAndRun(t, uniqueID("sub-depth20"), yaml, nil)
	assertResultEquals(t, er, float64(20))
}
