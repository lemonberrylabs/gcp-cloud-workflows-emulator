package integration

import (
	"testing"
)

// TestFor_ListIteration verifies basic for-in-list iteration.
func TestFor_ListIteration(t *testing.T) {
	yaml := loadWorkflow(t, "for_list.yaml")
	er := deployAndRun(t, uniqueID("for-list"), yaml, nil)
	// sum of [10, 20, 30] = 60
	assertResultEquals(t, er, float64(60))
}

// TestFor_ListWithIndex verifies for-in with index variable.
func TestFor_ListWithIndex(t *testing.T) {
	yaml := loadWorkflow(t, "for_list_index.yaml")
	er := deployAndRun(t, uniqueID("for-list-idx"), yaml, nil)
	assertResultEquals(t, er, []interface{}{"0:a", "1:b", "2:c"})
}

// TestFor_Range verifies for-range iteration (inclusive both ends).
func TestFor_Range(t *testing.T) {
	yaml := loadWorkflow(t, "for_range.yaml")
	er := deployAndRun(t, uniqueID("for-range"), yaml, nil)
	// sum of 1+2+3+4+5 = 15 (range [1,5] inclusive both ends)
	assertResultEquals(t, er, float64(15))
}

// TestFor_MapKeys verifies iterating over map keys.
func TestFor_MapKeys(t *testing.T) {
	yaml := loadWorkflow(t, "for_map_keys.yaml")
	er := deployAndRun(t, uniqueID("for-map-keys"), yaml, nil)
	// sum of values: 1+2+3 = 6
	assertResultEquals(t, er, float64(6))
}

// TestFor_Break verifies that break exits the loop early.
func TestFor_Break(t *testing.T) {
	yaml := loadWorkflow(t, "for_break.yaml")
	er := deployAndRun(t, uniqueID("for-break"), yaml, nil)
	// items [1,2,3,4,5], break when v > 3, so sum = 1+2+3 = 6
	assertResultEquals(t, er, float64(6))
}

// TestFor_Continue verifies that continue skips the current iteration.
func TestFor_Continue(t *testing.T) {
	yaml := loadWorkflow(t, "for_continue.yaml")
	er := deployAndRun(t, uniqueID("for-continue"), yaml, nil)
	// items [1,2,3,4,5], skip v==3, so sum = 1+2+4+5 = 12
	assertResultEquals(t, er, float64(12))
}

// TestFor_EmptyList verifies loop over empty list (zero iterations).
func TestFor_EmptyList(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - result: "unchanged"
    - loop:
        for:
          value: v
          in: []
          steps:
            - set:
                assign:
                  - result: "changed"
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("for-empty"), yaml, nil)
	assertResultEquals(t, er, "unchanged")
}

// TestFor_NestedLoops verifies nested for loops.
func TestFor_NestedLoops(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - total: 0
    - outer:
        for:
          value: i
          range: [1, 3]
          steps:
            - inner:
                for:
                  value: j
                  range: [1, 3]
                  steps:
                    - add:
                        assign:
                          - total: ${total + i * j}
    - done:
        return: ${total}
`
	er := deployAndRun(t, uniqueID("for-nested"), yaml, nil)
	// (1*1 + 1*2 + 1*3) + (2*1 + 2*2 + 2*3) + (3*1 + 3*2 + 3*3)
	// = 6 + 12 + 18 = 36
	assertResultEquals(t, er, float64(36))
}

// TestFor_VariableScoping verifies that variables created inside a loop
// don't leak outside.
func TestFor_VariableScoping(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - outer_var: "exists"
    - loop:
        for:
          value: v
          in: [1, 2, 3]
          steps:
            - inside:
                assign:
                  - loop_var: ${v}
    - check:
        try:
          steps:
            - access:
                assign:
                  - x: ${loop_var}
        except:
          as: e
          steps:
            - handle:
                return: "loop_var not accessible"
`
	er := deployAndRun(t, uniqueID("for-scope"), yaml, nil)
	// loop_var should not be accessible outside the for loop
	assertResultEquals(t, er, "loop_var not accessible")
}

// TestFor_ModifyParentScope verifies that modifying a parent-scope variable
// inside a for loop persists after the loop.
func TestFor_ModifyParentScope(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - counter: 0
    - loop:
        for:
          value: v
          range: [1, 5]
          steps:
            - inc:
                assign:
                  - counter: ${counter + 1}
    - done:
        return: ${counter}
`
	er := deployAndRun(t, uniqueID("for-parent"), yaml, nil)
	assertResultEquals(t, er, float64(5))
}

// TestFor_RangeReverse verifies that range with start > end produces no iterations.
func TestFor_RangeReverse(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - counter: 0
    - loop:
        for:
          value: v
          range: [5, 1]
          steps:
            - inc:
                assign:
                  - counter: ${counter + 1}
    - done:
        return: ${counter}
`
	er := deployAndRun(t, uniqueID("for-reverse"), yaml, nil)
	// GCW range [5, 1] should produce 0 iterations (start > end)
	assertResultEquals(t, er, float64(0))
}

// TestFor_SingleElement verifies loop with a single-element list.
func TestFor_SingleElement(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - sum: 0
    - loop:
        for:
          value: v
          in: [42]
          steps:
            - add:
                assign:
                  - sum: ${sum + v}
    - done:
        return: ${sum}
`
	er := deployAndRun(t, uniqueID("for-single"), yaml, nil)
	assertResultEquals(t, er, float64(42))
}

// TestFor_BreakInNestedLoop verifies that break only exits the innermost loop.
func TestFor_BreakInNestedLoop(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - result: []
    - outer:
        for:
          value: i
          range: [1, 3]
          steps:
            - inner:
                for:
                  value: j
                  range: [1, 5]
                  steps:
                    - check:
                        switch:
                          - condition: ${j > 2}
                            next: break
                    - collect:
                        assign:
                          - result: ${list.concat(result, [i * 10 + j])}
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("for-break-nested"), yaml, nil)
	// For each i (1,2,3), inner loop runs j=1,2 then breaks at j=3
	// So: [11, 12, 21, 22, 31, 32]
	assertResultEquals(t, er, []interface{}{
		float64(11), float64(12),
		float64(21), float64(22),
		float64(31), float64(32),
	})
}
