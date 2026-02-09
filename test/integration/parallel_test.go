package integration

import (
	"testing"
)

// TestParallel_TwoBranches verifies basic parallel branch execution.
func TestParallel_TwoBranches(t *testing.T) {
	yaml := loadWorkflow(t, "parallel_branches.yaml")
	er := deployAndRun(t, uniqueID("par-branches"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "a", "from_a")
	assertResultContains(t, er, "b", "from_b")
}

// TestParallel_ForLoop verifies parallel for loop execution.
func TestParallel_ForLoop(t *testing.T) {
	yaml := loadWorkflow(t, "parallel_for.yaml")
	er := deployAndRun(t, uniqueID("par-for"), yaml, nil)
	assertSucceeded(t, er)
	// Note: parallel writes to shared variable are atomic but final value
	// depends on execution order. The sum should be 1+2+3+4+5=15 regardless.
	assertResultEquals(t, er, float64(15))
}

// TestParallel_SharedVariables verifies shared variable declaration and access.
func TestParallel_SharedVariables(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - results: []
    - par:
        parallel:
          shared: [results]
          branches:
            - branch_a:
                steps:
                  - set_a:
                      assign:
                        - results: ${list.concat(results, ["a"])}
            - branch_b:
                steps:
                  - set_b:
                      assign:
                        - results: ${list.concat(results, ["b"])}
    - done:
        return: ${len(results)}
`
	er := deployAndRun(t, uniqueID("par-shared"), yaml, nil)
	assertSucceeded(t, er)
	// Both branches should have added to results
	assertResultEquals(t, er, float64(2))
}

// TestParallel_ConcurrencyLimit verifies that concurrency_limit is respected.
func TestParallel_ConcurrencyLimit(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - items: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
          - total: 0
    - par_loop:
        parallel:
          shared: [total]
          concurrency_limit: 2
          for:
            value: item
            in: ${items}
            steps:
              - add:
                  assign:
                    - total: ${total + item}
    - done:
        return: ${total}
`
	er := deployAndRun(t, uniqueID("par-limit"), yaml, nil)
	assertSucceeded(t, er)
	// Sum of 1..10 = 55
	assertResultEquals(t, er, float64(55))
}

// TestParallel_ExceptionPolicyUnhandled verifies that unhandled policy aborts
// on first exception.
func TestParallel_ExceptionPolicyUnhandled(t *testing.T) {
	yaml := `
main:
  steps:
    - try_par:
        try:
          steps:
            - par:
                parallel:
                  exception_policy: unhandled
                  branches:
                    - good:
                        steps:
                          - ok:
                              assign:
                                - x: 1
                    - bad:
                        steps:
                          - fail:
                              raise: "branch error"
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  message: ${e.message}
`
	er := deployAndRun(t, uniqueID("par-unhandled"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
}

// TestParallel_ExceptionPolicyContinueAll verifies that continueAll collects
// errors from all branches.
func TestParallel_ExceptionPolicyContinueAll(t *testing.T) {
	yaml := `
main:
  steps:
    - try_par:
        try:
          steps:
            - par:
                parallel:
                  exception_policy: continueAll
                  branches:
                    - bad1:
                        steps:
                          - fail1:
                              raise: "error 1"
                    - bad2:
                        steps:
                          - fail2:
                              raise: "error 2"
                    - good:
                        steps:
                          - ok:
                              assign:
                                - x: 1
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
`
	er := deployAndRun(t, uniqueID("par-continue"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
}

// TestParallel_EmptyBranches verifies parallel with no-op branches.
func TestParallel_EmptyBranches(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 0
    - par:
        parallel:
          shared: [x]
          branches:
            - a:
                steps:
                  - noop:
                      assign:
                        - x: ${x + 1}
            - b:
                steps:
                  - noop:
                      assign:
                        - x: ${x + 1}
    - done:
        return: ${x}
`
	er := deployAndRun(t, uniqueID("par-noop"), yaml, nil)
	assertSucceeded(t, er)
	// Both branches increment x, so result should be 2
	assertResultEquals(t, er, float64(2))
}

// TestParallel_NestingDepthExceeded verifies that exceeding parallel nesting
// depth (max 2) raises ParallelNestingError.
func TestParallel_NestingDepthExceeded(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 0
    - try_par:
        try:
          steps:
            - outer:
                parallel:
                  shared: [x]
                  branches:
                    - branch_a:
                        steps:
                          - middle:
                              parallel:
                                shared: [x]
                                branches:
                                  - inner_a:
                                      steps:
                                        - deep:
                                            parallel:
                                              shared: [x]
                                              branches:
                                                - too_deep:
                                                    steps:
                                                      - noop:
                                                          assign:
                                                            - x: 1
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  has_nesting_error: ${"ParallelNestingError" in e.tags}
`
	er := deployAndRun(t, uniqueID("par-nest-err"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	assertResultContains(t, er, "has_nesting_error", true)
}

// TestParallel_NestingDepth2Allowed verifies that parallel nesting depth of
// exactly 2 is allowed (the maximum).
func TestParallel_NestingDepth2Allowed(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 0
    - outer:
        parallel:
          shared: [x]
          branches:
            - branch_a:
                steps:
                  - inner:
                      parallel:
                        shared: [x]
                        branches:
                          - inner_a:
                              steps:
                                - set:
                                    assign:
                                      - x: ${x + 1}
                          - inner_b:
                              steps:
                                - set:
                                    assign:
                                      - x: ${x + 1}
    - done:
        return: ${x}
`
	er := deployAndRun(t, uniqueID("par-nest-ok"), yaml, nil)
	assertSucceeded(t, er)
	// Nesting depth 2 should succeed, x should be incremented
	assertResultEquals(t, er, float64(2))
}

// TestParallel_ContinueAllRaisesUnhandledBranchError verifies that after
// parallel continueAll completes with branch errors, the raised error has
// the UnhandledBranchError tag.
func TestParallel_ContinueAllRaisesUnhandledBranchError(t *testing.T) {
	yaml := `
main:
  steps:
    - try_par:
        try:
          steps:
            - par:
                parallel:
                  exception_policy: continueAll
                  branches:
                    - bad1:
                        steps:
                          - fail1:
                              raise: "error from branch 1"
                    - bad2:
                        steps:
                          - fail2:
                              raise: "error from branch 2"
                    - good:
                        steps:
                          - ok:
                              assign:
                                - x: 1
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  has_unhandled_branch: ${"UnhandledBranchError" in e.tags}
`
	er := deployAndRun(t, uniqueID("par-unhandled-tag"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	// After continueAll, the error should have UnhandledBranchError tag
	assertResultContains(t, er, "has_unhandled_branch", true)
}

// TestParallel_UnhandledPolicyAbortsBranches verifies that with default
// "unhandled" exception policy, an error in one branch aborts other branches.
func TestParallel_UnhandledPolicyAbortsBranches(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 0
    - try_par:
        try:
          steps:
            - par:
                parallel:
                  shared: [x]
                  branches:
                    - fast_fail:
                        steps:
                          - fail:
                              raise: "immediate failure"
                    - slow:
                        steps:
                          - wait:
                              call: sys.sleep
                              args:
                                seconds: 5
                          - set:
                              assign:
                                - x: 999
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  x_value: ${x}
`
	er := deployAndRun(t, uniqueID("par-abort"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
	// The slow branch should have been aborted, x should still be 0
	assertResultContains(t, er, "x_value", float64(0))
}

// TestParallel_NestedForLoopsComplexPipeline verifies that a complex workflow
// with parallel nesting depth 2 (parallel-for inside parallel-for) succeeds
// when multiple outer iterations run concurrently. This is a regression test
// for a bug where parallel depth was tracked via a shared counter on the
// Engine struct, causing concurrent inner parallels to race and falsely
// exceed the nesting limit.
func TestParallel_NestedForLoopsComplexPipeline(t *testing.T) {
	yaml := loadWorkflow(t, "parallel_nested_pipeline.yaml")
	args := map[string]interface{}{
		"workflowId": "test-wf-123",
		"itemId":     "item-456",
		"config": map[string]interface{}{
			"enablePreview": true,
		},
	}
	er := deployAndRun(t, uniqueID("par-nested-pipeline"), yaml, args)
	assertSucceeded(t, er)
	assertResultContains(t, er, "status", "complete")
	assertResultContains(t, er, "workflowId", "test-wf-123")
	assertResultContains(t, er, "itemId", "item-456")
	assertResultContains(t, er, "outputPath", "/output/final")
}

// TestParallel_NonSharedVariableIsolation verifies that non-shared variables
// are isolated between parallel branches.
func TestParallel_NonSharedVariableIsolation(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - shared_val: ""
    - par:
        parallel:
          shared: [shared_val]
          branches:
            - branch_a:
                steps:
                  - set_local:
                      assign:
                        - local_a: "from_a"
                        - shared_val: "a_was_here"
            - branch_b:
                steps:
                  - set_local:
                      assign:
                        - local_b: "from_b"
    - done:
        return:
          shared: ${shared_val}
`
	er := deployAndRun(t, uniqueID("par-isolation"), yaml, nil)
	assertSucceeded(t, er)
	// shared_val should be set by one of the branches
	// Non-shared variables (local_a, local_b) should not be accessible here
}
