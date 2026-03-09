package integration

import (
	"testing"
)

// TestChildWorkflow_BasicRun verifies that a parent workflow can invoke a child
// workflow using the googleapis.workflowexecutions.v1 connector run method.
func TestChildWorkflow_BasicRun(t *testing.T) {
	// Deploy the child workflow first.
	childID := uniqueID("child-basic")
	childYAML := `
main:
  steps:
    - done:
        return: "hello from child"
`
	createWorkflow(t, childID, childYAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+childID)

	// Parent workflow calls the child via the connector.
	parentYAML := `
main:
  steps:
    - run_child:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: ` + childID + `
        result: child_execution
    - done:
        return: ${child_execution.result}
`
	er := deployAndRun(t, uniqueID("parent-basic"), parentYAML, nil)
	assertResultEquals(t, er, "hello from child")
}

// TestChildWorkflow_PassArguments verifies that arguments are correctly passed
// from parent to child workflow via the connector.
func TestChildWorkflow_PassArguments(t *testing.T) {
	childID := uniqueID("child-args")
	childYAML := `
main:
  params: [args]
  steps:
    - done:
        return:
          greeting: ${"Hello, " + args.name + "!"}
          doubled: ${args.value * 2}
`
	createWorkflow(t, childID, childYAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+childID)

	parentYAML := `
main:
  steps:
    - run_child:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: ` + childID + `
          argument:
            name: "World"
            value: 21
        result: child_execution
    - done:
        return: ${child_execution.result}
`
	er := deployAndRun(t, uniqueID("parent-args"), parentYAML, nil)
	assertSucceeded(t, er)
	resultMap, ok := er.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be a map, got %T: %v", er.Result, er.Result)
	}
	if resultMap["greeting"] != "Hello, World!" {
		t.Errorf("expected greeting 'Hello, World!', got %v", resultMap["greeting"])
	}
	if resultMap["doubled"] != float64(42) {
		t.Errorf("expected doubled 42, got %v", resultMap["doubled"])
	}
}

// TestChildWorkflow_ChildReturnsMap verifies that complex return values from
// the child workflow are accessible in the parent.
func TestChildWorkflow_ChildReturnsMap(t *testing.T) {
	childID := uniqueID("child-map")
	childYAML := `
main:
  params: [args]
  steps:
    - done:
        return:
          status: "ok"
          items:
            - "a"
            - "b"
            - "c"
          count: 3
`
	createWorkflow(t, childID, childYAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+childID)

	parentYAML := `
main:
  steps:
    - run_child:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: ` + childID + `
          argument: {}
        result: child_execution
    - done:
        return:
          child_status: ${child_execution.result.status}
          child_count: ${child_execution.result.count}
`
	er := deployAndRun(t, uniqueID("parent-map"), parentYAML, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "child_status", "ok")
	assertResultContains(t, er, "child_count", float64(3))
}

// TestChildWorkflow_ChildFailure verifies that when a child workflow fails,
// the error propagates to the parent and can be caught with try/except.
func TestChildWorkflow_ChildFailure(t *testing.T) {
	childID := uniqueID("child-fail")
	childYAML := `
main:
  steps:
    - fail:
        raise:
          code: 99
          message: "child workflow error"
`
	createWorkflow(t, childID, childYAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+childID)

	parentYAML := `
main:
  steps:
    - try_child:
        try:
          steps:
            - run_child:
                call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
                args:
                  workflow_id: ` + childID + `
                result: child_execution
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  error_message: ${e.message}
`
	er := deployAndRun(t, uniqueID("parent-fail"), parentYAML, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
}

// TestChildWorkflow_NotFound verifies that referencing a non-existent workflow
// causes an error.
func TestChildWorkflow_NotFound(t *testing.T) {
	parentYAML := `
main:
  steps:
    - run_child:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: this-workflow-does-not-exist
        result: child_execution
    - done:
        return: ${child_execution.result}
`
	er := deployAndRunExpectError(t, uniqueID("parent-notfound"), parentYAML, nil)
	assertFailed(t, er)
	msg, _ := parseErrorPayload(er)
	if msg == "" {
		t.Errorf("expected error message about workflow not found, got empty")
	}
}

// TestChildWorkflow_ChainedExecution verifies that a parent can call multiple
// child workflows in sequence and pass results between them.
func TestChildWorkflow_ChainedExecution(t *testing.T) {
	child1ID := uniqueID("child-chain1")
	child1YAML := `
main:
  params: [args]
  steps:
    - done:
        return: ${args.x + 10}
`
	createWorkflow(t, child1ID, child1YAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+child1ID)

	child2ID := uniqueID("child-chain2")
	child2YAML := `
main:
  params: [args]
  steps:
    - done:
        return: ${args.x * 2}
`
	createWorkflow(t, child2ID, child2YAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+child2ID)

	parentYAML := `
main:
  steps:
    - step1:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: ` + child1ID + `
          argument:
            x: 5
        result: exec1
    - step2:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: ` + child2ID + `
          argument:
            x: ${exec1.result}
        result: exec2
    - done:
        return: ${exec2.result}
`
	// 5 + 10 = 15, then 15 * 2 = 30
	er := deployAndRun(t, uniqueID("parent-chain"), parentYAML, nil)
	assertResultEquals(t, er, float64(30))
}

// TestChildWorkflow_InForLoop verifies that a parent workflow can invoke a
// child workflow inside a for loop (the delete-account use case).
func TestChildWorkflow_InForLoop(t *testing.T) {
	childID := uniqueID("child-loop")
	childYAML := `
main:
  params: [args]
  steps:
    - done:
        return:
          processed: ${args.item}
`
	createWorkflow(t, childID, childYAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+childID)

	parentYAML := `
main:
  steps:
    - init:
        assign:
          - items: ["user-1", "user-2", "user-3"]
          - results: []
    - process_loop:
        for:
          value: item
          in: ${items}
          steps:
            - run_child:
                call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
                args:
                  workflow_id: ` + childID + `
                  argument:
                    item: ${item}
                result: child_exec
            - collect:
                assign:
                  - results: ${list.concat(results, [child_exec.result.processed])}
    - done:
        return: ${results}
`
	er := deployAndRun(t, uniqueID("parent-loop"), parentYAML, nil)
	assertResultEquals(t, er, []interface{}{"user-1", "user-2", "user-3"})
}

// TestChildWorkflow_ExecutionMetadata verifies that the returned execution
// object contains expected metadata fields (name, state, startTime, etc.).
func TestChildWorkflow_ExecutionMetadata(t *testing.T) {
	childID := uniqueID("child-meta")
	childYAML := `
main:
  steps:
    - done:
        return: 42
`
	createWorkflow(t, childID, childYAML)
	defer deleteWorkflow(t, parentPath+"/workflows/"+childID)

	parentYAML := `
main:
  steps:
    - run_child:
        call: googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run
        args:
          workflow_id: ` + childID + `
        result: child_execution
    - done:
        return:
          has_name: ${child_execution.name != null}
          state: ${child_execution.state}
          result_value: ${child_execution.result}
`
	er := deployAndRun(t, uniqueID("parent-meta"), parentYAML, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "state", "SUCCEEDED")
	assertResultContains(t, er, "result_value", float64(42))
}
