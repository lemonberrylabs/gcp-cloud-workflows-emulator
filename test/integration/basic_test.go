package integration

import (
	"testing"
)

// TestBasic_ReturnLiteral verifies that a workflow can return a literal string.
func TestBasic_ReturnLiteral(t *testing.T) {
	yaml := loadWorkflow(t, "basic_return.yaml")
	er := deployAndRun(t, uniqueID("basic-return"), yaml, nil)
	assertResultEquals(t, er, "hello world")
}

// TestBasic_AssignAndReturn verifies simple variable assignment and return.
func TestBasic_AssignAndReturn(t *testing.T) {
	yaml := loadWorkflow(t, "basic_assign_return.yaml")
	er := deployAndRun(t, uniqueID("basic-assign"), yaml, nil)
	assertResultEquals(t, er, float64(42))
}

// TestBasic_MultiStep verifies sequential step execution with variable references.
func TestBasic_MultiStep(t *testing.T) {
	yaml := loadWorkflow(t, "basic_multi_step.yaml")
	er := deployAndRun(t, uniqueID("basic-multi"), yaml, nil)
	assertResultEquals(t, er, float64(30))
}

// TestBasic_WorkflowParams verifies that main workflow can receive params.
func TestBasic_WorkflowParams(t *testing.T) {
	yaml := loadWorkflow(t, "basic_params.yaml")
	er := deployAndRun(t, uniqueID("basic-params"), yaml, map[string]interface{}{
		"name": "Alice",
	})
	assertResultEquals(t, er, "Alice")
}

// TestBasic_StringConcat verifies string concatenation in expressions.
func TestBasic_StringConcat(t *testing.T) {
	yaml := loadWorkflow(t, "basic_string_concat.yaml")
	er := deployAndRun(t, uniqueID("basic-concat"), yaml, nil)
	assertResultEquals(t, er, "Hello, World!")
}

// TestBasic_AllTypes verifies that the emulator correctly handles all GCW types.
func TestBasic_AllTypes(t *testing.T) {
	yaml := loadWorkflow(t, "basic_types.yaml")
	er := deployAndRun(t, uniqueID("basic-types"), yaml, nil)
	assertSucceeded(t, er)

	assertResultContains(t, er, "int", float64(42))
	assertResultContains(t, er, "double", 3.14)
	assertResultContains(t, er, "string", "hello")
	assertResultContains(t, er, "bool", true)
	assertResultContains(t, er, "null", nil)
	assertResultContains(t, er, "list", []interface{}{float64(1), float64(2), float64(3)})
	assertResultContains(t, er, "map", map[string]interface{}{"key": "value"})
}

// TestBasic_InlineReturn verifies inline return in a single step workflow.
func TestBasic_InlineReturn(t *testing.T) {
	yaml := `
main:
  steps:
    - done:
        return: 42
`
	er := deployAndRun(t, uniqueID("basic-inline"), yaml, nil)
	assertResultEquals(t, er, float64(42))
}

// TestBasic_AssignMultipleVars verifies assigning multiple variables in one step.
func TestBasic_AssignMultipleVars(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - a: 1
          - b: 2
          - c: 3
    - done:
        return:
          a: ${a}
          b: ${b}
          c: ${c}
`
	er := deployAndRun(t, uniqueID("basic-multi-assign"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "a", float64(1))
	assertResultContains(t, er, "b", float64(2))
	assertResultContains(t, er, "c", float64(3))
}

// TestBasic_AssignListIndex verifies assigning to list indices.
func TestBasic_AssignListIndex(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - my_list: ["a", "b", "c"]
          - my_list[1]: "B"
    - done:
        return: ${my_list}
`
	er := deployAndRun(t, uniqueID("basic-list-idx"), yaml, nil)
	assertResultEquals(t, er, []interface{}{"a", "B", "c"})
}

// TestBasic_AssignMapKey verifies assigning to map keys using dot notation.
func TestBasic_AssignMapKey(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - my_map:
              name: "Alice"
          - my_map.age: 30
    - done:
        return: ${my_map}
`
	er := deployAndRun(t, uniqueID("basic-map-key"), yaml, nil)
	assertResultEquals(t, er, map[string]interface{}{"name": "Alice", "age": float64(30)})
}

// TestBasic_NoReturn verifies that a workflow without explicit return returns null.
func TestBasic_NoReturn(t *testing.T) {
	yaml := `
main:
  steps:
    - noop:
        assign:
          - x: 1
`
	er := deployAndRun(t, uniqueID("basic-no-return"), yaml, nil)
	assertSucceeded(t, er)
	// GCW returns null when no explicit return statement
	if er.Result != nil {
		t.Errorf("expected nil result, got %v", er.Result)
	}
}
