package parser

import (
	"testing"
)

func TestParseBasicWorkflow(t *testing.T) {
	src := []byte(`
main:
  steps:
    - init:
        assign:
          - x: 1
          - y: "hello"
    - done:
        return: ${x}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wf.Main == nil {
		t.Fatal("main workflow is nil")
	}
	if wf.Main.Name != "main" {
		t.Errorf("expected name 'main', got %q", wf.Main.Name)
	}
	if len(wf.Main.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(wf.Main.Steps))
	}

	// Check assign step
	init := wf.Main.Steps[0]
	if init.Name != "init" {
		t.Errorf("expected step name 'init', got %q", init.Name)
	}
	if len(init.Assign) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(init.Assign))
	}
	if init.Assign[0].Target != "x" {
		t.Errorf("expected target 'x', got %q", init.Assign[0].Target)
	}
	if init.Assign[0].Value != int64(1) {
		t.Errorf("expected value 1, got %v (%T)", init.Assign[0].Value, init.Assign[0].Value)
	}
	if init.Assign[1].Target != "y" {
		t.Errorf("expected target 'y', got %q", init.Assign[1].Target)
	}

	// Check return step
	done := wf.Main.Steps[1]
	if done.Name != "done" {
		t.Errorf("expected step name 'done', got %q", done.Name)
	}
	if !done.HasReturn {
		t.Error("expected HasReturn to be true")
	}
}

func TestParseWorkflowWithParams(t *testing.T) {
	src := []byte(`
main:
  params: [args]
  steps:
    - init:
        assign:
          - name: ${args.name}
    - done:
        return: ${name}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(wf.Main.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(wf.Main.Params))
	}
	if wf.Main.Params[0].Name != "args" {
		t.Errorf("expected param name 'args', got %q", wf.Main.Params[0].Name)
	}
}

func TestParseSubworkflow(t *testing.T) {
	src := []byte(`
main:
  steps:
    - call_sub:
        call: greet
        args:
          first_name: "Ada"
        result: message
    - done:
        return: ${message}

greet:
  params:
    - first_name
    - last_name: "Unknown"
  steps:
    - build:
        return: ${"Hello " + first_name}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wf.Main == nil {
		t.Fatal("main is nil")
	}

	greet, ok := wf.Subworkflows["greet"]
	if !ok {
		t.Fatal("subworkflow 'greet' not found")
	}
	if len(greet.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(greet.Params))
	}
	if greet.Params[0].Name != "first_name" {
		t.Errorf("expected param 'first_name', got %q", greet.Params[0].Name)
	}
	if greet.Params[0].HasDefault {
		t.Error("first_name should not have a default")
	}
	if greet.Params[1].Name != "last_name" {
		t.Errorf("expected param 'last_name', got %q", greet.Params[1].Name)
	}
	if !greet.Params[1].HasDefault {
		t.Error("last_name should have a default")
	}
	if greet.Params[1].Default != "Unknown" {
		t.Errorf("expected default 'Unknown', got %v", greet.Params[1].Default)
	}

	// Check call step
	callStep := wf.Main.Steps[0]
	if callStep.Call == nil {
		t.Fatal("call is nil")
	}
	if callStep.Call.Function != "greet" {
		t.Errorf("expected function 'greet', got %q", callStep.Call.Function)
	}
	if callStep.Result != "message" {
		t.Errorf("expected result 'message', got %q", callStep.Result)
	}
}

func TestParseSwitchStep(t *testing.T) {
	src := []byte(`
main:
  steps:
    - check:
        switch:
          - condition: ${x < 10}
            next: handle_small
          - condition: ${x >= 10}
            steps:
              - set:
                  assign:
                    - category: "big"
    - handle_small:
        return: "small"
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	check := wf.Main.Steps[0]
	if len(check.Switch) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(check.Switch))
	}
	if check.Switch[0].Next != "handle_small" {
		t.Errorf("expected next 'handle_small', got %q", check.Switch[0].Next)
	}
	if check.Switch[1].Steps == nil {
		t.Error("expected inline steps in second condition")
	}
}

func TestParseForLoop(t *testing.T) {
	src := []byte(`
main:
  steps:
    - init:
        assign:
          - total: 0
    - loop:
        for:
          value: item
          index: i
          in: ${my_list}
          steps:
            - add:
                assign:
                  - total: ${total + item}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loop := wf.Main.Steps[1]
	if loop.For == nil {
		t.Fatal("for is nil")
	}
	if loop.For.Value != "item" {
		t.Errorf("expected value 'item', got %q", loop.For.Value)
	}
	if loop.For.Index != "i" {
		t.Errorf("expected index 'i', got %q", loop.For.Index)
	}
	if loop.For.HasRange {
		t.Error("expected HasRange to be false")
	}
}

func TestParseForRange(t *testing.T) {
	src := []byte(`
main:
  steps:
    - loop:
        for:
          value: v
          range: [1, 10]
          steps:
            - step:
                assign:
                  - x: ${v}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loop := wf.Main.Steps[0]
	if !loop.For.HasRange {
		t.Error("expected HasRange to be true")
	}
	if loop.For.Range[0] != int64(1) {
		t.Errorf("expected range start 1, got %v", loop.For.Range[0])
	}
	if loop.For.Range[1] != int64(10) {
		t.Errorf("expected range end 10, got %v", loop.For.Range[1])
	}
}

func TestParseParallelBranches(t *testing.T) {
	src := []byte(`
main:
  steps:
    - parallel_step:
        parallel:
          shared: [result_a, result_b]
          concurrency_limit: 5
          exception_policy: continueAll
          branches:
            - branch_a:
                steps:
                  - get_a:
                      assign:
                        - result_a: "a"
            - branch_b:
                steps:
                  - get_b:
                      assign:
                        - result_b: "b"
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ps := wf.Main.Steps[0]
	if ps.Parallel == nil {
		t.Fatal("parallel is nil")
	}
	if len(ps.Parallel.Shared) != 2 {
		t.Errorf("expected 2 shared vars, got %d", len(ps.Parallel.Shared))
	}
	if ps.Parallel.ConcurrencyLimit != 5 {
		t.Errorf("expected concurrency_limit 5, got %d", ps.Parallel.ConcurrencyLimit)
	}
	if ps.Parallel.ExceptionPolicy != "continueAll" {
		t.Errorf("expected exception_policy 'continueAll', got %q", ps.Parallel.ExceptionPolicy)
	}
	if len(ps.Parallel.Branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(ps.Parallel.Branches))
	}
	if ps.Parallel.Branches[0].Name != "branch_a" {
		t.Errorf("expected branch name 'branch_a', got %q", ps.Parallel.Branches[0].Name)
	}
}

func TestParseTryExceptRetry(t *testing.T) {
	src := []byte(`
main:
  steps:
    - handle_errors:
        try:
          steps:
            - risky:
                call: http.get
                args:
                  url: https://example.com
                result: response
        retry:
          predicate: ${http.default_retry}
          max_retries: 5
          backoff:
            initial_delay: 1
            max_delay: 60
            multiplier: 2
        except:
          as: e
          steps:
            - log_error:
                assign:
                  - error_msg: ${e.message}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	step := wf.Main.Steps[0]
	if step.Try == nil {
		t.Fatal("try is nil")
	}
	if len(step.Try.Try) != 1 {
		t.Fatalf("expected 1 try step, got %d", len(step.Try.Try))
	}
	if step.Try.Retry == nil {
		t.Fatal("retry is nil")
	}
	if step.Try.Retry.MaxRetries != 5 {
		t.Errorf("expected max_retries 5, got %d", step.Try.Retry.MaxRetries)
	}
	if step.Try.Retry.Backoff == nil {
		t.Fatal("backoff is nil")
	}
	if step.Try.Retry.Backoff.InitialDelay != 1 {
		t.Errorf("expected initial_delay 1, got %f", step.Try.Retry.Backoff.InitialDelay)
	}
	if step.Try.Except == nil {
		t.Fatal("except is nil")
	}
	if step.Try.Except.As != "e" {
		t.Errorf("expected as 'e', got %q", step.Try.Except.As)
	}
}

func TestParseTryInlineCall(t *testing.T) {
	// GCP Cloud Workflows allows a try block to contain a direct call/args/result
	// without wrapping in steps. This is common with retry predicates.
	src := []byte(`
main:
  params: [input]
  steps:
    - init:
        assign:
          - serviceUrl: "https://example.com"
          - authToken: "tok"
          - workflow_id: ${input.workflowId}
          - config: ${input.config}
    - run_pipeline:
        try:
          steps:
            - generate_domain_research:
                try:
                  call: http.post
                  args:
                    url: '${serviceUrl + "/api/research"}'
                    headers:
                      Content-Type: application/json
                      x-auth: ${authToken}
                    body:
                      config: ${config}
                      workflowId: ${workflow_id}
                    timeout: 120
                  result: domainResearchResult
                retry:
                  predicate: ${retry_predicate}
                  max_retries: 3
                  backoff:
                    initial_delay: 2
                    max_delay: 30
                    multiplier: 2
            - use_result:
                assign:
                  - output: ${domainResearchResult.body}
        except:
          as: e
          steps:
            - handle:
                raise: ${e}
retry_predicate:
  params: [e]
  steps:
    - check:
        switch:
          - condition: ${e.code == 429}
            return: true
    - default:
        return: false
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// run_pipeline step should have a try block
	runPipeline := wf.Main.Steps[1]
	if runPipeline.Try == nil {
		t.Fatal("run_pipeline: try is nil")
	}
	if len(runPipeline.Try.Try) != 2 {
		t.Fatalf("expected 2 try steps in run_pipeline, got %d", len(runPipeline.Try.Try))
	}

	// generate_domain_research should have an inline try (call without steps wrapper)
	genStep := runPipeline.Try.Try[0]
	if genStep.Try == nil {
		t.Fatal("generate_domain_research: try is nil")
	}
	if len(genStep.Try.Try) != 1 {
		t.Fatalf("expected 1 inline try step, got %d", len(genStep.Try.Try))
	}
	inlineStep := genStep.Try.Try[0]
	if inlineStep.Call == nil {
		t.Fatal("inline try step: call is nil")
	}
	if inlineStep.Call.Function != "http.post" {
		t.Errorf("expected call function 'http.post', got %q", inlineStep.Call.Function)
	}
	if inlineStep.Call.Result != "domainResearchResult" {
		t.Errorf("expected result 'domainResearchResult', got %q", inlineStep.Call.Result)
	}

	// retry should be parsed
	if genStep.Try.Retry == nil {
		t.Fatal("generate_domain_research: retry is nil")
	}
	if genStep.Try.Retry.MaxRetries != 3 {
		t.Errorf("expected max_retries 3, got %d", genStep.Try.Retry.MaxRetries)
	}

	// retry_predicate subworkflow should exist
	if _, ok := wf.Subworkflows["retry_predicate"]; !ok {
		t.Fatal("retry_predicate subworkflow not found")
	}
}

func TestParseRaiseStep(t *testing.T) {
	src := []byte(`
main:
  steps:
    - raise_error:
        raise:
          code: 55
          message: "Something went wrong"
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	step := wf.Main.Steps[0]
	if step.Raise == nil {
		t.Fatal("raise is nil")
	}
	raiseMap, ok := step.Raise.(map[string]interface{})
	if !ok {
		t.Fatalf("expected raise to be a map, got %T", step.Raise)
	}
	if raiseMap["code"] != int64(55) {
		t.Errorf("expected code 55, got %v", raiseMap["code"])
	}
}

func TestParseNextStep(t *testing.T) {
	src := []byte(`
main:
  steps:
    - step1:
        assign:
          - x: 1
        next: step3
    - step2:
        assign:
          - y: 2
    - step3:
        return: ${x}
`)

	wf, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wf.Main.Steps[0].Next != "step3" {
		t.Errorf("expected next 'step3', got %q", wf.Main.Steps[0].Next)
	}
}

func TestParseRejectsNoMain(t *testing.T) {
	src := []byte(`
helper:
  steps:
    - step1:
        return: 1
`)

	_, err := Parse(src)
	if err == nil {
		t.Fatal("expected error for missing main")
	}
}

func TestParseRejectsOversize(t *testing.T) {
	src := make([]byte, MaxSourceSize+1)
	for i := range src {
		src[i] = 'a'
	}

	_, err := Parse(src)
	if err == nil {
		t.Fatal("expected error for oversize source")
	}
}

func TestParseLargeAssignmentSucceeds(t *testing.T) {
	// Limits are enforced at runtime, not parse time.
	// Parsing should succeed even with many assignments.
	yaml := "main:\n  steps:\n    - big:\n        assign:\n"
	for i := 0; i <= MaxAssignments; i++ {
		yaml += "          - x" + string(rune('a'+i%26)) + ": 1\n"
	}

	_, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("expected parse to succeed (limits are runtime-enforced), got: %v", err)
	}
}
