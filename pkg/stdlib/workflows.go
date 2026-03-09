package stdlib

import (
	"fmt"
	"time"

	"github.com/lemonberrylabs/gcw-emulator/pkg/ast"
	"github.com/lemonberrylabs/gcw-emulator/pkg/parser"
	"github.com/lemonberrylabs/gcw-emulator/pkg/types"
)

// WorkflowStore is the interface needed to look up workflows for child execution.
type WorkflowStore interface {
	FindWorkflowByID(workflowID string) (WorkflowInfo, error)
}

// WorkflowInfo holds the data needed to execute a child workflow.
type WorkflowInfo struct {
	Name       string
	SourceCode string
}

// ChildExecutor runs a child workflow synchronously and returns the result.
// It is provided by the API layer which has access to the runtime engine.
type ChildExecutor func(wfAST *ast.Workflow, args types.Value) (types.Value, error)

// RegisterWorkflowExecution registers the googleapis.workflowexecutions.v1
// connector function for child workflow execution.
func (r *Registry) RegisterWorkflowExecution(
	store WorkflowStore,
	parsedCache map[string]*ast.Workflow,
	executor ChildExecutor,
) {
	r.Register(
		"googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run",
		func(args []types.Value) (types.Value, error) {
			return workflowExecutionsRun(args, store, parsedCache, executor)
		},
	)
}

func workflowExecutionsRun(
	args []types.Value,
	store WorkflowStore,
	parsedCache map[string]*ast.Workflow,
	executor ChildExecutor,
) (types.Value, error) {
	if len(args) == 0 {
		return types.Null, fmt.Errorf("googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run requires arguments")
	}

	m := args[0].AsMap()
	if m == nil {
		return types.Null, fmt.Errorf("googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run: expected map argument")
	}

	// Extract workflow_id (required)
	workflowIDVal, ok := m.Get("workflow_id")
	if !ok || workflowIDVal.Type() != types.TypeString {
		return types.Null, fmt.Errorf("googleapis.workflowexecutions.v1.projects.locations.workflows.executions.run: workflow_id is required and must be a string")
	}
	workflowID := workflowIDVal.AsString()

	// Extract argument (optional)
	var childArgs types.Value = types.Null
	if argVal, ok := m.Get("argument"); ok && !argVal.IsNull() {
		childArgs = argVal
	}

	// Look up the workflow
	wfInfo, err := store.FindWorkflowByID(workflowID)
	if err != nil {
		return types.Null, &types.WorkflowError{
			Message: fmt.Sprintf("workflow '%s' not found", workflowID),
			Code:    404,
			Tags:    []string{"NotFoundError"},
		}
	}

	// Get or parse the workflow AST
	wfAST, ok := parsedCache[wfInfo.Name]
	if !ok {
		parsed, err := parser.Parse([]byte(wfInfo.SourceCode))
		if err != nil {
			return types.Null, fmt.Errorf("failed to parse child workflow '%s': %v", workflowID, err)
		}
		wfAST = parsed
		parsedCache[wfInfo.Name] = wfAST
	}

	// Execute the child workflow synchronously
	startTime := time.Now()
	result, err := executor(wfAST, childArgs)
	endTime := time.Now()

	// Build the execution response object matching the GCW Execution resource
	execResult := types.NewOrderedMap()
	execResult.Set("name", types.NewString(fmt.Sprintf("%s/executions/child-%d", wfInfo.Name, time.Now().UnixNano())))
	execResult.Set("startTime", types.NewString(startTime.Format(time.RFC3339)))
	execResult.Set("endTime", types.NewString(endTime.Format(time.RFC3339)))

	if err != nil {
		execResult.Set("state", types.NewString("FAILED"))

		// Propagate the error to the parent workflow
		return types.Null, err
	}

	execResult.Set("state", types.NewString("SUCCEEDED"))
	execResult.Set("result", result)

	return types.NewMap(execResult), nil
}
