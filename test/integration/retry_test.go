package integration

import (
	"testing"
)

// TestRetry_BasicRetry verifies basic retry behavior with max_retries.
func TestRetry_BasicRetry(t *testing.T) {
	// This test needs an HTTP mock server. For now we use a subworkflow
	// that tracks attempts and fails until a threshold.
	yaml := `
main:
  steps:
    - init:
        assign:
          - attempt: 0
    - try_step:
        try:
          steps:
            - increment:
                assign:
                  - attempt: ${attempt + 1}
            - check:
                switch:
                  - condition: ${attempt < 3}
                    steps:
                      - fail:
                          raise:
                            code: 500
                            message: "transient error"
        retry:
          max_retries: 5
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
    - done:
        return:
          attempts: ${attempt}
`
	er := deployAndRun(t, uniqueID("retry-basic"), yaml, nil)
	assertSucceeded(t, er)
	// Should succeed on attempt 3
	assertResultContains(t, er, "attempts", float64(3))
}

// TestRetry_ExceedMaxRetries verifies that exceeding max_retries fails.
func TestRetry_ExceedMaxRetries(t *testing.T) {
	yaml := `
main:
  steps:
    - try_step:
        try:
          steps:
            - fail:
                raise:
                  code: 500
                  message: "always fails"
        retry:
          max_retries: 2
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
        except:
          as: e
          steps:
            - handle:
                return:
                  caught: true
                  message: ${e.message}
`
	er := deployAndRun(t, uniqueID("retry-exceed"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "caught", true)
}

// TestRetry_WithCustomPredicate verifies retry with a custom predicate subworkflow.
func TestRetry_WithCustomPredicate(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - attempt: 0
    - try_step:
        try:
          steps:
            - increment:
                assign:
                  - attempt: ${attempt + 1}
            - fail:
                raise:
                  code: 503
                  message: "service unavailable"
                  tags: ["HttpError"]
        retry:
          predicate: ${should_retry}
          max_retries: 3
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
        except:
          as: e
          steps:
            - handle:
                return:
                  attempts: ${attempt}
                  message: ${e.message}

should_retry:
  params: [e]
  steps:
    - check:
        switch:
          - condition: ${"HttpError" in e.tags}
            return: true
    - done:
        return: false
`
	er := deployAndRun(t, uniqueID("retry-predicate"), yaml, nil)
	assertSucceeded(t, er)
	// Should have retried 3 times + original = 4 attempts
	assertResultContains(t, er, "attempts", float64(4))
}

// TestRetry_NeverPolicy verifies that retry.never disables retries.
func TestRetry_NeverPolicy(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - attempt: 0
    - try_step:
        try:
          steps:
            - increment:
                assign:
                  - attempt: ${attempt + 1}
            - fail:
                raise: "error"
        retry:
          predicate: ${retry.never}
          max_retries: 10
          backoff:
            initial_delay: 0.1
            max_delay: 1
            multiplier: 2
        except:
          as: e
          steps:
            - handle:
                return: ${attempt}
`
	er := deployAndRun(t, uniqueID("retry-never"), yaml, nil)
	assertSucceeded(t, er)
	// retry.never means no retries, only 1 attempt
	assertResultEquals(t, er, float64(1))
}
