// Package ast defines the Abstract Syntax Tree types for parsed GCW workflows.
// These types represent the structure of a workflow after YAML/JSON parsing
// and before execution.
package ast

// Workflow represents a complete parsed workflow with its subworkflows.
type Workflow struct {
	// Main is the entry-point workflow (always named "main").
	Main *Subworkflow

	// Subworkflows maps subworkflow names to their definitions.
	// Does not include "main".
	Subworkflows map[string]*Subworkflow
}

// Subworkflow represents a single workflow or subworkflow definition.
type Subworkflow struct {
	// Name is the workflow/subworkflow identifier.
	Name string

	// Params defines the parameter list.
	// For "main", this is either empty or a single parameter name (receives a map).
	// For subworkflows, these are named params that may have defaults.
	Params []Param

	// Steps is the ordered list of steps in this workflow.
	Steps []*Step
}

// Param represents a workflow/subworkflow parameter.
type Param struct {
	// Name is the parameter name.
	Name string

	// Default is the default value expression (nil if required).
	Default interface{}

	// HasDefault indicates whether a default was specified.
	HasDefault bool
}

// Step represents a single workflow step.
type Step struct {
	// Name is the step identifier, unique within its containing step list.
	Name string

	// Assign holds assignment operations (non-nil for assign steps).
	Assign []Assignment

	// Call holds a function/subworkflow call (non-nil for call steps).
	Call *CallExpr

	// Switch holds conditional branches (non-nil for switch steps).
	Switch []SwitchCondition

	// For holds a for-loop definition (non-nil for for steps).
	For *ForExpr

	// Parallel holds parallel execution (non-nil for parallel steps).
	Parallel *ParallelExpr

	// Try holds try/except/retry (non-nil for try steps).
	Try *TryExpr

	// Raise holds a raise expression (non-nil for raise steps).
	Raise interface{} // string expression or map

	// Return holds a return expression (non-nil for return steps).
	Return interface{} // any expression

	// HasReturn distinguishes return:null from no return.
	HasReturn bool

	// Next is the next step name ("end", "break", "continue", or step name).
	Next string

	// Steps holds nested step grouping (non-nil for steps steps).
	Steps []*Step

	// Result is the variable name to store call results.
	Result string
}

// Assignment represents a single variable assignment within an assign step.
type Assignment struct {
	// Target is the assignment target (variable name, possibly with property/index paths).
	// Examples: "x", "my_map.key", "my_list[0]"
	Target string

	// Value is the expression to evaluate and assign.
	Value interface{}
}

// CallExpr represents a call step: HTTP call, stdlib call, or subworkflow call.
type CallExpr struct {
	// Function is the fully qualified function name (e.g., "http.get", "sys.log", "my_subworkflow").
	Function string

	// Args maps argument names to their values/expressions.
	Args map[string]interface{}

	// Result is the variable name to store the return value.
	Result string
}

// SwitchCondition represents a single condition branch in a switch step.
type SwitchCondition struct {
	// Condition is the expression to evaluate (must be truthy to match).
	Condition interface{}

	// Next is the step to jump to if this condition matches.
	Next string

	// Steps are inline steps to execute if this condition matches.
	Steps []*Step

	// Assign holds inline assignments if condition matches.
	Assign []Assignment

	// Return holds an inline return value if condition matches.
	Return interface{}

	// HasReturn distinguishes return:null from no return.
	HasReturn bool

	// Raise holds an inline raise if condition matches.
	Raise interface{}
}

// ForExpr represents a for-loop step.
type ForExpr struct {
	// Value is the loop variable name for the current element.
	Value string

	// Index is the optional loop variable name for the current index.
	Index string

	// In is the expression producing the list/map to iterate over.
	In interface{}

	// Range specifies [start, end] inclusive range for numeric iteration.
	Range [2]interface{} // [start_expr, end_expr]

	// HasRange indicates whether Range (not In) should be used.
	HasRange bool

	// Steps is the loop body.
	Steps []*Step
}

// ParallelExpr represents a parallel execution step.
type ParallelExpr struct {
	// Shared lists variable names accessible across branches.
	Shared []string

	// Branches holds named parallel branches (nil if using for-loop).
	Branches []*ParallelBranch

	// For holds a parallel for-loop (nil if using branches).
	For *ForExpr

	// ConcurrencyLimit is the max concurrent goroutines (0 = use default of 20).
	ConcurrencyLimit int

	// ExceptionPolicy is "unhandled" (default) or "continueAll".
	ExceptionPolicy string
}

// ParallelBranch represents a single named branch within a parallel step.
type ParallelBranch struct {
	// Name is the branch identifier.
	Name string

	// Steps is the branch body.
	Steps []*Step
}

// TryExpr represents a try/except/retry step.
type TryExpr struct {
	// Try is the steps to attempt.
	Try []*Step

	// Except handles errors from the try block.
	Except *ExceptExpr

	// Retry configures automatic retry behavior.
	Retry *RetryExpr
}

// ExceptExpr represents the except clause of a try step.
type ExceptExpr struct {
	// As is the variable name to bind the caught error to.
	As string

	// Steps is the error handling body.
	Steps []*Step
}

// RetryExpr represents the retry clause of a try step.
type RetryExpr struct {
	// Predicate is the retry predicate expression
	// (e.g., "${http.default_retry}" or a subworkflow name).
	Predicate interface{}

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// Backoff configures exponential backoff.
	Backoff *BackoffExpr
}

// BackoffExpr defines exponential backoff parameters for retry.
type BackoffExpr struct {
	// InitialDelay in seconds.
	InitialDelay float64

	// MaxDelay in seconds.
	MaxDelay float64

	// Multiplier for exponential growth.
	Multiplier float64
}
