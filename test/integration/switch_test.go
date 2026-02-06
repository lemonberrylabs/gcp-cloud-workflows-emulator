package integration

import (
	"testing"
)

// TestSwitch_BasicConditions verifies switch with multiple conditions.
func TestSwitch_BasicConditions(t *testing.T) {
	yaml := loadWorkflow(t, "switch_basic.yaml")
	er := deployAndRun(t, uniqueID("switch-basic"), yaml, nil)
	// x=50, should match x < 100 -> "medium"
	assertResultEquals(t, er, "medium")
}

// TestSwitch_WithNext verifies switch conditions that use next jumps.
func TestSwitch_WithNext(t *testing.T) {
	yaml := loadWorkflow(t, "switch_with_next.yaml")
	er := deployAndRun(t, uniqueID("switch-next"), yaml, nil)
	// x=5 < 10, should jump to handle_small
	assertResultEquals(t, er, "small")
}

// TestSwitch_WithEmbeddedSteps verifies switch conditions with embedded step blocks.
func TestSwitch_WithEmbeddedSteps(t *testing.T) {
	yaml := loadWorkflow(t, "switch_with_steps.yaml")
	er := deployAndRun(t, uniqueID("switch-steps"), yaml, nil)
	// x=42 > 40, so result = 42*2 = 84
	assertResultEquals(t, er, float64(84))
}

// TestSwitch_DefaultCondition verifies that condition: true acts as default.
func TestSwitch_DefaultCondition(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 999
    - check:
        switch:
          - condition: ${x == 1}
            assign:
              - result: "one"
          - condition: ${x == 2}
            assign:
              - result: "two"
          - condition: true
            assign:
              - result: "default"
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("switch-default"), yaml, nil)
	assertResultEquals(t, er, "default")
}

// TestSwitch_FirstMatchWins verifies that only the first matching condition executes.
func TestSwitch_FirstMatchWins(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 5
    - check:
        switch:
          - condition: ${x < 10}
            assign:
              - result: "first"
          - condition: ${x < 20}
            assign:
              - result: "second"
          - condition: true
            assign:
              - result: "default"
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("switch-first"), yaml, nil)
	// Both x<10 and x<20 are true, but first match wins
	assertResultEquals(t, er, "first")
}

// TestSwitch_NoMatch verifies that execution continues when no condition matches.
func TestSwitch_NoMatch(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 100
          - result: "unchanged"
    - check:
        switch:
          - condition: ${x == 1}
            assign:
              - result: "one"
          - condition: ${x == 2}
            assign:
              - result: "two"
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("switch-nomatch"), yaml, nil)
	// No conditions match, so result stays "unchanged" and execution falls through
	assertResultEquals(t, er, "unchanged")
}

// TestSwitch_NestedSteps verifies switch with complex embedded step blocks.
func TestSwitch_NestedSteps(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - x: 10
    - check:
        switch:
          - condition: ${x >= 10}
            steps:
              - step_a:
                  assign:
                    - x: ${x + 5}
              - step_b:
                  assign:
                    - x: ${x * 2}
              - return_step:
                  return: ${x}
`
	er := deployAndRun(t, uniqueID("switch-nested"), yaml, nil)
	// x=10, +5=15, *2=30
	assertResultEquals(t, er, float64(30))
}

// TestSwitch_TableDriven verifies various switch condition outcomes.
func TestSwitch_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"small", 5, "small"},
		{"medium", 50, "medium"},
		{"large", 500, "large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
main:
  params: [args]
  steps:
    - check:
        switch:
          - condition: ${args.x < 10}
            assign:
              - result: "small"
          - condition: ${args.x < 100}
            assign:
              - result: "medium"
          - condition: true
            assign:
              - result: "large"
    - done:
        return: ${result}
`
			er := deployAndRun(t, uniqueID("switch-td-"+tt.name), yaml, map[string]interface{}{
				"x": tt.input,
			})
			assertResultEquals(t, er, tt.expected)
		})
	}
}
