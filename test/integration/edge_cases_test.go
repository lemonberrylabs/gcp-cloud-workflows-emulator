package integration

import (
	"testing"
)

// TestEdge_NextEnd terminates workflow early with next: end.
func TestEdge_NextEnd(t *testing.T) {
	yaml := loadWorkflow(t, "next_end.yaml")
	er := deployAndRun(t, uniqueID("edge-next-end"), yaml, nil)
	assertSucceeded(t, er)
	// next: end should stop execution after step1, result should be null
	// because no return was executed
	if er.Result != nil {
		t.Errorf("expected nil result after next: end, got %v", er.Result)
	}
}

// TestEdge_NextJump verifies forward jump skipping steps.
func TestEdge_NextJump(t *testing.T) {
	yaml := loadWorkflow(t, "next_jump.yaml")
	er := deployAndRun(t, uniqueID("edge-next-jump"), yaml, nil)
	// step1 jumps to step3, skipping step2
	assertResultEquals(t, er, "step1,step3")
}

// TestEdge_EmptyListIteration verifies zero-iteration loop.
func TestEdge_EmptyListIteration(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - ran: false
    - loop:
        for:
          value: v
          in: []
          steps:
            - set:
                assign:
                  - ran: true
    - done:
        return: ${ran}
`
	er := deployAndRun(t, uniqueID("edge-empty-loop"), yaml, nil)
	assertResultEquals(t, er, false)
}

// TestEdge_NullInExpressions verifies null arithmetic/comparison behavior.
func TestEdge_NullInExpressions(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - null_eq_null: ${null == null}
          - null_neq_1: ${null != 1}
    - done:
        return:
          null_eq_null: ${null_eq_null}
          null_neq_1: ${null_neq_1}
`
	er := deployAndRun(t, uniqueID("edge-null-expr"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "null_eq_null", true)
	assertResultContains(t, er, "null_neq_1", true)
}

// TestEdge_MapLiteralInExpression verifies inline map creation in expressions.
func TestEdge_MapLiteralInExpression(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - data: ${{  "key": "value", "num": 42  }}
    - done:
        return: ${data}
`
	er := deployAndRun(t, uniqueID("edge-map-literal"), yaml, nil)
	assertResultEquals(t, er, map[string]interface{}{"key": "value", "num": float64(42)})
}

// TestEdge_ListLiteralInExpression verifies inline list creation in expressions.
func TestEdge_ListLiteralInExpression(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - data: ${[1, "two", true, null]}
    - done:
        return: ${data}
`
	er := deployAndRun(t, uniqueID("edge-list-literal"), yaml, nil)
	assertResultEquals(t, er, []interface{}{float64(1), "two", true, nil})
}

// TestEdge_NestedMapAccess verifies deeply nested property access.
func TestEdge_NestedMapAccess(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - data:
              level1:
                level2:
                  level3:
                    value: "deep"
    - done:
        return: ${data.level1.level2.level3.value}
`
	er := deployAndRun(t, uniqueID("edge-deep-access"), yaml, nil)
	assertResultEquals(t, er, "deep")
}

// TestEdge_BooleanExpressions verifies truthy/falsy evaluation.
func TestEdge_BooleanExpressions(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - true_and_true: ${true and true}
          - true_and_false: ${true and false}
          - false_or_true: ${false or true}
          - not_not_true: ${not not true}
    - done:
        return:
          tt: ${true_and_true}
          tf: ${true_and_false}
          ft: ${false_or_true}
          nnt: ${not_not_true}
`
	er := deployAndRun(t, uniqueID("edge-bool"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "tt", true)
	assertResultContains(t, er, "tf", false)
	assertResultContains(t, er, "ft", true)
	assertResultContains(t, er, "nnt", true)
}

// TestEdge_StringEscaping verifies string escape handling.
func TestEdge_StringEscaping(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - with_quotes: "hello \"world\""
          - with_newline: "line1\nline2"
    - done:
        return:
          quotes: ${with_quotes}
          newline: ${with_newline}
`
	er := deployAndRun(t, uniqueID("edge-escape"), yaml, nil)
	assertSucceeded(t, er)
}

// TestEdge_ExprInReturn verifies complex expression in return step.
func TestEdge_ExprInReturn(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 10
          - y: 20
    - done:
        return: ${x * y + 5}
`
	er := deployAndRun(t, uniqueID("edge-expr-return"), yaml, nil)
	assertResultEquals(t, er, float64(205))
}

// TestEdge_MultipleSubworkflows verifies a workflow with several subworkflows.
func TestEdge_MultipleSubworkflows(t *testing.T) {
	yaml := `
main:
  steps:
    - call_add:
        call: add
        args:
          a: 10
          b: 20
        result: sum
    - call_mul:
        call: multiply
        args:
          a: ${sum}
          b: 3
        result: product
    - done:
        return: ${product}

add:
  params: [a, b]
  steps:
    - compute:
        return: ${a + b}

multiply:
  params: [a, b]
  steps:
    - compute:
        return: ${a * b}
`
	er := deployAndRun(t, uniqueID("edge-multi-sub"), yaml, nil)
	// (10+20) * 3 = 90
	assertResultEquals(t, er, float64(90))
}

// TestEdge_LoopWithSubworkflowCall verifies calling subworkflows inside loops.
func TestEdge_LoopWithSubworkflowCall(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - items: [1, 2, 3, 4, 5]
          - sum: 0
    - loop:
        for:
          value: v
          in: ${items}
          steps:
            - call_double:
                call: double
                args:
                  n: ${v}
                result: doubled
            - add:
                assign:
                  - sum: ${sum + doubled}
    - done:
        return: ${sum}

double:
  params: [n]
  steps:
    - compute:
        return: ${n * 2}
`
	er := deployAndRun(t, uniqueID("edge-loop-sub"), yaml, nil)
	// sum of 2+4+6+8+10 = 30
	assertResultEquals(t, er, float64(30))
}

// TestEdge_SwitchInLoop verifies switch inside a for loop.
func TestEdge_SwitchInLoop(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - items: [1, 2, 3, 4, 5]
          - evens: 0
          - odds: 0
    - loop:
        for:
          value: v
          in: ${items}
          steps:
            - classify:
                switch:
                  - condition: ${v % 2 == 0}
                    assign:
                      - evens: ${evens + 1}
                  - condition: true
                    assign:
                      - odds: ${odds + 1}
    - done:
        return:
          evens: ${evens}
          odds: ${odds}
`
	er := deployAndRun(t, uniqueID("edge-switch-loop"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "evens", float64(2))
	assertResultContains(t, er, "odds", float64(3))
}

// TestEdge_TryInLoop verifies error handling inside a for loop.
func TestEdge_TryInLoop(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - items: [1, 0, 2, 0, 3]
          - results: []
    - loop:
        for:
          value: v
          in: ${items}
          steps:
            - try_div:
                try:
                  steps:
                    - compute:
                        assign:
                          - val: ${10 / v}
                          - results: ${list.concat(results, [val])}
                except:
                  as: e
                  steps:
                    - handle:
                        assign:
                          - results: ${list.concat(results, [-1])}
    - done:
        return: ${results}
`
	er := deployAndRun(t, uniqueID("edge-try-loop"), yaml, nil)
	assertSucceeded(t, er)
	// 10/1=10, 10/0=error(-1), 10/2=5, 10/0=error(-1), 10/3=3.33...
	// Results should have 5 elements
	resultList, ok := er.Result.([]interface{})
	if !ok {
		t.Fatalf("expected list result, got %T", er.Result)
	}
	if len(resultList) != 5 {
		t.Errorf("expected 5 results, got %d", len(resultList))
	}
}

// TestEdge_IntegerDivisionVsFloat verifies // vs / behavior.
func TestEdge_IntegerDivisionVsFloat(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - float_div: ${7 / 2}
          - int_div: ${7 // 2}
    - done:
        return:
          float_div: ${float_div}
          int_div: ${int_div}
`
	er := deployAndRun(t, uniqueID("edge-div-types"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "float_div", 3.5)
	assertResultContains(t, er, "int_div", float64(3))
}
