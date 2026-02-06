package integration

import (
	"testing"
)

// --- text.* functions ---

// TestStdlib_TextToLower verifies text.to_lower.
func TestStdlib_TextToLower(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: text.to_lower
        args:
          source: "HELLO World"
        result: val
    - done:
        return: ${val}
`
	er := deployAndRun(t, uniqueID("stdlib-lower"), yaml, nil)
	assertResultEquals(t, er, "hello world")
}

// TestStdlib_TextToUpper verifies text.to_upper.
func TestStdlib_TextToUpper(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: text.to_upper
        args:
          source: "Hello World"
        result: val
    - done:
        return: ${val}
`
	er := deployAndRun(t, uniqueID("stdlib-upper"), yaml, nil)
	assertResultEquals(t, er, "HELLO WORLD")
}

// TestStdlib_TextSplit verifies text.split.
func TestStdlib_TextSplit(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: text.split
        args:
          source: "a,b,c"
          separator: ","
        result: val
    - done:
        return: ${val}
`
	er := deployAndRun(t, uniqueID("stdlib-split"), yaml, nil)
	assertResultEquals(t, er, []interface{}{"a", "b", "c"})
}

// TestStdlib_TextSubstring verifies text.substring.
func TestStdlib_TextSubstring(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: text.substring
        args:
          source: "Hello World"
          start: 6
          end: 11
        result: val
    - done:
        return: ${val}
`
	er := deployAndRun(t, uniqueID("stdlib-substr"), yaml, nil)
	assertResultEquals(t, er, "World")
}

// TestStdlib_TextReplaceAll verifies text.replace_all.
func TestStdlib_TextReplaceAll(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: text.replace_all
        args:
          source: "hello world hello"
          substr: "hello"
          replacement: "hi"
        result: val
    - done:
        return: ${val}
`
	er := deployAndRun(t, uniqueID("stdlib-replace"), yaml, nil)
	assertResultEquals(t, er, "hi world hi")
}

// TestStdlib_TextMatchRegex verifies text.match_regex.
func TestStdlib_TextMatchRegex(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - match1: true
          - match2: false
    - check1:
        call: text.match_regex
        args:
          source: "hello123"
          regex: "[a-z]+[0-9]+"
        result: match1
    - check2:
        call: text.match_regex
        args:
          source: "hello"
          regex: "^[0-9]+$"
        result: match2
    - done:
        return:
          match1: ${match1}
          match2: ${match2}
`
	er := deployAndRun(t, uniqueID("stdlib-regex"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "match1", true)
	assertResultContains(t, er, "match2", false)
}

// TestStdlib_TextUrlEncode verifies text.url_encode.
func TestStdlib_TextUrlEncode(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: text.url_encode
        args:
          source: "hello world&foo=bar"
        result: val
    - done:
        return: ${val}
`
	er := deployAndRun(t, uniqueID("stdlib-urlencode"), yaml, nil)
	assertSucceeded(t, er)
	// Should URL-encode spaces and special chars
}

// --- json.* functions ---

// TestStdlib_JsonEncodeDecode verifies json.encode_to_string and json.decode.
func TestStdlib_JsonEncodeDecode(t *testing.T) {
	yaml := `
main:
  steps:
    - encode:
        call: json.encode_to_string
        args:
          data:
            name: "test"
            value: 42
        result: json_str
    - decode:
        call: json.decode
        args:
          data: ${json_str}
        result: parsed
    - done:
        return:
          name: ${parsed.name}
          value: ${parsed.value}
`
	er := deployAndRun(t, uniqueID("stdlib-json"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "name", "test")
	assertResultContains(t, er, "value", float64(42))
}

// --- base64.* functions ---

// TestStdlib_Base64EncodeDecode verifies base64.encode and base64.decode.
func TestStdlib_Base64EncodeDecode(t *testing.T) {
	yaml := `
main:
  steps:
    - encode:
        call: base64.encode
        args:
          data: "Hello, World!"
        result: encoded
    - decode:
        call: base64.decode
        args:
          data: ${encoded}
        result: decoded
    - done:
        return:
          encoded: ${encoded}
          decoded: ${decoded}
`
	er := deployAndRun(t, uniqueID("stdlib-base64"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "encoded", "SGVsbG8sIFdvcmxkIQ==")
	assertResultContains(t, er, "decoded", "Hello, World!")
}

// --- math.* functions ---

// TestStdlib_MathFunctions verifies math.abs, math.floor, math.max, math.min.
func TestStdlib_MathFunctions(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - abs_val: ${math.abs(-42)}
          - floor_val: ${math.floor(3.7)}
          - max_val: ${math.max(10, 20)}
          - min_val: ${math.min(10, 20)}
    - done:
        return:
          abs: ${abs_val}
          floor: ${floor_val}
          max: ${max_val}
          min: ${min_val}
`
	er := deployAndRun(t, uniqueID("stdlib-math"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "abs", float64(42))
	assertResultContains(t, er, "floor", float64(3))
	assertResultContains(t, er, "max", float64(20))
	assertResultContains(t, er, "min", float64(10))
}

// --- list.* functions ---

// TestStdlib_ListConcat verifies list.concat.
func TestStdlib_ListConcat(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - a: [1, 2]
          - b: [3, 4]
          - result: ${list.concat(a, b)}
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("stdlib-list-concat"), yaml, nil)
	assertResultEquals(t, er, []interface{}{float64(1), float64(2), float64(3), float64(4)})
}

// TestStdlib_ListPrepend verifies list.prepend.
func TestStdlib_ListPrepend(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - items: [2, 3]
          - result: ${list.prepend(items, 1)}
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("stdlib-list-prepend"), yaml, nil)
	assertResultEquals(t, er, []interface{}{float64(1), float64(2), float64(3)})
}

// --- map.* functions ---

// TestStdlib_MapGet verifies map.get with default value.
func TestStdlib_MapGet(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - m:
              a: 1
    - compute:
        assign:
          - exists: ${map.get(m, "a")}
          - missing: ${map.get(m, "b", "default")}
    - done:
        return:
          exists: ${exists}
          missing: ${missing}
`
	er := deployAndRun(t, uniqueID("stdlib-map-get"), yaml, nil)
	assertSucceeded(t, er)
	assertResultContains(t, er, "exists", float64(1))
	assertResultContains(t, er, "missing", "default")
}

// TestStdlib_MapDelete verifies map.delete.
func TestStdlib_MapDelete(t *testing.T) {
	yaml := `
main:
  steps:
    - init:
        assign:
          - m:
              a: 1
              b: 2
              c: 3
    - compute:
        assign:
          - result: ${map.delete(m, "b")}
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("stdlib-map-delete"), yaml, nil)
	assertResultEquals(t, er, map[string]interface{}{"a": float64(1), "c": float64(3)})
}

// TestStdlib_MapMerge verifies map.merge.
func TestStdlib_MapMerge(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        assign:
          - a:
              x: 1
              y: 2
          - b:
              y: 20
              z: 30
          - result: ${map.merge(a, b)}
    - done:
        return: ${result}
`
	er := deployAndRun(t, uniqueID("stdlib-map-merge"), yaml, nil)
	assertResultEquals(t, er, map[string]interface{}{
		"x": float64(1),
		"y": float64(20),
		"z": float64(30),
	})
}

// --- sys.* functions ---

// TestStdlib_SysNow verifies sys.now returns a timestamp.
func TestStdlib_SysNow(t *testing.T) {
	yaml := `
main:
  steps:
    - compute:
        call: sys.now
        result: ts
    - done:
        return: ${ts}
`
	er := deployAndRun(t, uniqueID("stdlib-sys-now"), yaml, nil)
	assertSucceeded(t, er)
	// sys.now returns a Unix epoch timestamp (number)
	if er.Result == nil {
		t.Fatal("expected non-nil timestamp from sys.now")
	}
}

// TestStdlib_SysGetEnv verifies sys.get_env returns workflow metadata.
func TestStdlib_SysGetEnv(t *testing.T) {
	yaml := `
main:
  steps:
    - get_project:
        call: sys.get_env
        args:
          name: "GOOGLE_CLOUD_PROJECT_ID"
        result: project
    - done:
        return: ${project}
`
	er := deployAndRun(t, uniqueID("stdlib-sys-env"), yaml, nil)
	assertSucceeded(t, er)
	// Emulator should return something for this env var
}

// --- uuid.* functions ---

// TestStdlib_UuidGenerate verifies uuid.generate returns a valid UUID string.
func TestStdlib_UuidGenerate(t *testing.T) {
	yaml := `
main:
  steps:
    - gen:
        call: uuid.generate
        result: id
    - done:
        return:
          id: ${id}
          length: ${len(id)}
`
	er := deployAndRun(t, uniqueID("stdlib-uuid"), yaml, nil)
	assertSucceeded(t, er)
	// UUID v4 format: 8-4-4-4-12 = 36 chars
	assertResultContains(t, er, "length", float64(36))
}
