package integration

import (
	"fmt"
	"strings"
	"testing"
)

// TestAssign_SimpleTypes verifies assignment of all basic types.
func TestAssign_SimpleTypes(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - my_int: 100
          - my_double: 2.718
          - my_string: "test"
          - my_bool: false
          - my_null: null
    - done:
        return:
          int: ${my_int}
          double: ${my_double}
          string: ${my_string}
          bool: ${my_bool}
          "null": ${my_null}
`
	er := deployAndRun(t, uniqueID("assign-types"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "int", float64(100))
	assertResultContains(t, er, "double", 2.718)
	assertResultContains(t, er, "string", "test")
	assertResultContains(t, er, "bool", false)
	assertResultContains(t, er, "null", nil)
}

// TestAssign_ListOperations verifies list creation and index assignment.
func TestAssign_ListOperations(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - items: [1, 2, 3]
          - items[0]: 10
          - items[2]: 30
    - done:
        return: ${items}
`
	er := deployAndRun(t, uniqueID("assign-list"), yaml, nil)
	assertResultEquals(t, er, []interface{}{float64(10), float64(2), float64(30)})
}

// TestAssign_MapOperations verifies map creation and key assignment.
func TestAssign_MapOperations(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - data:
              name: "Alice"
              age: 30
          - data.email: "alice@example.com"
          - data["phone"]: "555-1234"
    - done:
        return: ${data}
`
	er := deployAndRun(t, uniqueID("assign-map"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "name", "Alice")
	assertResultContains(t, er, "email", "alice@example.com")
}

// TestAssign_NestedMapAssignment verifies deep nested map key assignment.
func TestAssign_NestedMapAssignment(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - data:
              address:
                city: "Portland"
          - data.address.zip: "97201"
    - done:
        return: ${data.address}
`
	er := deployAndRun(t, uniqueID("assign-nested"), yaml, nil)
	assertResultEquals(t, er, map[string]interface{}{
		"city": "Portland",
		"zip":  "97201",
	})
}

// TestAssign_ExpressionValues verifies that assignments can use expressions.
func TestAssign_ExpressionValues(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - a: 10
          - b: ${a * 2}
          - c: ${a + b}
    - done:
        return:
          a: ${a}
          b: ${b}
          c: ${c}
`
	er := deployAndRun(t, uniqueID("assign-expr"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "a", float64(10))
	assertResultContains(t, er, "b", float64(20))
	assertResultContains(t, er, "c", float64(30))
}

// TestAssign_SequentialDependency verifies that assignments within one step
// are evaluated sequentially (later ones can reference earlier ones).
func TestAssign_SequentialDependency(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 5
          - y: ${x + 1}
          - z: ${y + 1}
    - done:
        return: ${z}
`
	er := deployAndRun(t, uniqueID("assign-seq"), yaml, nil)
	assertResultEquals(t, er, float64(7))
}

// TestAssign_Reassignment verifies that variables can be reassigned.
func TestAssign_Reassignment(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 1
    - update:
        assign:
          - x: ${x + 10}
    - done:
        return: ${x}
`
	er := deployAndRun(t, uniqueID("assign-reassign"), yaml, nil)
	assertResultEquals(t, er, float64(11))
}

// TestAssign_MaxAssignments verifies that exactly 50 assignments are allowed.
func TestAssign_MaxAssignments(t *testing.T) {
	// Build a YAML with exactly 50 assignments
	var assigns []string
	for i := 0; i < 50; i++ {
		assigns = append(assigns, fmt.Sprintf("          - v%d: %d", i, i))
	}
	yaml := fmt.Sprintf(`
main:
  steps:
    - init:
        assign:
%s
    - done:
        return: ${v49}
`, strings.Join(assigns, "\n"))

	er := deployAndRun(t, uniqueID("assign-max50"), yaml, nil)
	assertResultEquals(t, er, float64(49))
}

// TestAssign_ExceedMaxAssignments verifies that 51+ assignments produce an error.
func TestAssign_ExceedMaxAssignments(t *testing.T) {
	var assigns []string
	for i := 0; i < 51; i++ {
		assigns = append(assigns, fmt.Sprintf("          - v%d: %d", i, i))
	}
	yaml := fmt.Sprintf(`
main:
  steps:
    - init:
        assign:
%s
    - done:
        return: ${v50}
`, strings.Join(assigns, "\n"))

	er := deployAndRunExpectError(t, uniqueID("assign-over50"), yaml, nil)
	assertFailed(t, er)
}

// TestAssign_EmptyMap verifies assigning an empty map.
func TestAssign_EmptyMap(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - m: {}
          - m.key: "value"
    - done:
        return: ${m}
`
	er := deployAndRun(t, uniqueID("assign-empty-map"), yaml, nil)
	assertResultEquals(t, er, map[string]interface{}{"key": "value"})
}

// TestAssign_EmptyList verifies assigning an empty list.
func TestAssign_EmptyList(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - items: []
    - done:
        return: ${len(items)}
`
	er := deployAndRun(t, uniqueID("assign-empty-list"), yaml, nil)
	assertResultEquals(t, er, float64(0))
}
