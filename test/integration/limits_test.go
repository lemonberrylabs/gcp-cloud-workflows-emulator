package integration

import (
	"fmt"
	"strings"
	"testing"
)

// TestLimits_MaxAssignments verifies the 50-assignment-per-step limit.
func TestLimits_MaxAssignments(t *testing.T) {
	// Build 51 assignments
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
        return: 1
`, strings.Join(assigns, "\n"))

	er := deployAndRunExpectError(t, uniqueID("limit-assign"), yaml, nil)
	assertFailed(t, er)
}

// TestLimits_MaxSwitchConditions verifies the 50-conditions-per-switch limit.
func TestLimits_MaxSwitchConditions(t *testing.T) {
	// Build 51 conditions
	var conditions []string
	for i := 0; i < 51; i++ {
		conditions = append(conditions, fmt.Sprintf("          - condition: ${x == %d}\n            assign:\n              - result: %d", i, i))
	}
	yaml := fmt.Sprintf(`
main:
  steps:
    - init:
        assign:
          - x: 0
    - check:
        switch:
%s
    - done:
        return: ${result}
`, strings.Join(conditions, "\n"))

	er := deployAndRunExpectError(t, uniqueID("limit-switch"), yaml, nil)
	assertFailed(t, er)
}

// TestLimits_RecursionDepth verifies the 20-call-stack-depth limit.
func TestLimits_RecursionDepth(t *testing.T) {
	// This should exceed the 20-level call stack limit
	yaml := `
main:
  steps:
    - call:
        call: recurse
        args:
          depth: 0
        result: val
    - done:
        return: ${val}

recurse:
  params: [depth]
  steps:
    - go:
        call: recurse
        args:
          depth: ${depth + 1}
        result: val
    - done:
        return: ${val}
`
	er := deployAndRunExpectError(t, uniqueID("limit-recurse"), yaml, nil)
	assertFailed(t, er)
}

// TestLimits_ExpressionLength verifies the 400-character expression limit.
func TestLimits_ExpressionLength(t *testing.T) {
	// Create an expression longer than 400 characters
	longExpr := strings.Repeat("1 + ", 100) + "1"
	yaml := fmt.Sprintf(`
main:
  steps:
    - compute:
        assign:
          - x: ${%s}
    - done:
        return: ${x}
`, longExpr)

	er := deployAndRunExpectError(t, uniqueID("limit-expr"), yaml, nil)
	assertFailed(t, er)
}

// TestLimits_MaxBranchesPerParallel verifies the 10-branch limit.
func TestLimits_MaxBranchesPerParallel(t *testing.T) {
	// Build 11 parallel branches
	var branches []string
	for i := 0; i < 11; i++ {
		branches = append(branches, fmt.Sprintf(`            - branch_%d:
                steps:
                  - step_%d:
                      assign:
                        - x: %d`, i, i, i))
	}
	yaml := fmt.Sprintf(`
main:
  steps:
    - init:
        assign:
          - x: 0
    - par:
        parallel:
          shared: [x]
          branches:
%s
    - done:
        return: ${x}
`, strings.Join(branches, "\n"))

	er := deployAndRunExpectError(t, uniqueID("limit-branches"), yaml, nil)
	assertFailed(t, er)
}

// TestLimits_ValidAssignCount verifies that exactly 50 assignments succeed.
func TestLimits_ValidAssignCount(t *testing.T) {
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

	er := deployAndRun(t, uniqueID("limit-assign-ok"), yaml, nil)
	assertResultEquals(t, er, float64(49))
}

// TestLimits_ValidSwitchCount verifies that exactly 50 conditions succeed.
func TestLimits_ValidSwitchCount(t *testing.T) {
	var conditions []string
	for i := 0; i < 50; i++ {
		conditions = append(conditions, fmt.Sprintf("          - condition: ${x == %d}\n            assign:\n              - result: %d", i, i))
	}
	yaml := fmt.Sprintf(`
main:
  steps:
    - init:
        assign:
          - x: 25
          - result: -1
    - check:
        switch:
%s
    - done:
        return: ${result}
`, strings.Join(conditions, "\n"))

	er := deployAndRun(t, uniqueID("limit-switch-ok"), yaml, nil)
	assertResultEquals(t, er, float64(25))
}
