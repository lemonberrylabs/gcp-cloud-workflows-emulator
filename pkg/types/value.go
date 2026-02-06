// Package types defines the core value types used throughout the GCW emulator.
// It implements the GCW type system: int, double, string, bool, null, list, map, bytes.
package types

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ValueType represents the type of a GCW value.
type ValueType int

const (
	TypeNull   ValueType = iota
	TypeBool             // bool
	TypeInt              // int64
	TypeDouble           // float64
	TypeString           // string
	TypeBytes            // []byte
	TypeList             // []Value
	TypeMap              // ordered map of string -> Value
)

// String returns the GCW type name as returned by the type() stdlib function.
func (t ValueType) String() string {
	switch t {
	case TypeNull:
		return "null"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeDouble:
		return "double"
	case TypeString:
		return "string"
	case TypeBytes:
		return "bytes"
	case TypeList:
		return "list"
	case TypeMap:
		return "map"
	default:
		return "unknown"
	}
}

// Value represents a GCW runtime value. It uses a tagged union approach for efficiency.
type Value struct {
	typ        ValueType
	boolVal    bool
	intVal     int64
	doubleVal  float64
	stringVal  string
	bytesVal   []byte
	listVal    []Value
	mapVal     *OrderedMap
}

// OrderedMap maintains insertion order for map keys, matching GCW map behavior.
type OrderedMap struct {
	keys   []string
	values map[string]Value
}

// NewOrderedMap creates a new empty ordered map.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0),
		values: make(map[string]Value),
	}
}

// NewOrderedMapFromPairs creates an ordered map from alternating key-value pairs.
func NewOrderedMapFromPairs(pairs ...interface{}) *OrderedMap {
	m := NewOrderedMap()
	for i := 0; i+1 < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			continue
		}
		val, ok := pairs[i+1].(Value)
		if !ok {
			continue
		}
		m.Set(key, val)
	}
	return m
}

// Get retrieves a value by key. Returns the value and whether it exists.
func (m *OrderedMap) Get(key string) (Value, bool) {
	v, ok := m.values[key]
	return v, ok
}

// Set adds or updates a key-value pair, preserving insertion order.
func (m *OrderedMap) Set(key string, val Value) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = val
}

// Delete removes a key from the map.
func (m *OrderedMap) Delete(key string) {
	if _, exists := m.values[key]; !exists {
		return
	}
	delete(m.values, key)
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

// Keys returns the keys in insertion order.
func (m *OrderedMap) Keys() []string {
	result := make([]string, len(m.keys))
	copy(result, m.keys)
	return result
}

// Len returns the number of entries.
func (m *OrderedMap) Len() int {
	return len(m.keys)
}

// Clone creates a deep copy of the ordered map.
func (m *OrderedMap) Clone() *OrderedMap {
	c := NewOrderedMap()
	for _, k := range m.keys {
		c.Set(k, m.values[k].Clone())
	}
	return c
}

// Null is the singleton null value.
var Null = Value{typ: TypeNull}

// NewBool creates a boolean value.
func NewBool(v bool) Value {
	return Value{typ: TypeBool, boolVal: v}
}

// NewInt creates an integer value (64-bit).
func NewInt(v int64) Value {
	return Value{typ: TypeInt, intVal: v}
}

// NewDouble creates a double value (64-bit float).
func NewDouble(v float64) Value {
	return Value{typ: TypeDouble, doubleVal: v}
}

// NewString creates a string value.
func NewString(v string) Value {
	return Value{typ: TypeString, stringVal: v}
}

// NewBytes creates a bytes value.
func NewBytes(v []byte) Value {
	return Value{typ: TypeBytes, bytesVal: v}
}

// NewList creates a list value from a slice of values.
func NewList(v []Value) Value {
	return Value{typ: TypeList, listVal: v}
}

// NewMap creates a map value from an OrderedMap.
func NewMap(v *OrderedMap) Value {
	return Value{typ: TypeMap, mapVal: v}
}

// NewMapFromGoMap creates a map value from a Go map (keys sorted alphabetically for determinism).
func NewMapFromGoMap(m map[string]Value) Value {
	om := NewOrderedMap()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		om.Set(k, m[k])
	}
	return Value{typ: TypeMap, mapVal: om}
}

// Type returns the value's type.
func (v Value) Type() ValueType {
	return v.typ
}

// IsNull returns true if the value is null.
func (v Value) IsNull() bool {
	return v.typ == TypeNull
}

// AsBool returns the boolean value. Panics if not a bool.
func (v Value) AsBool() bool {
	if v.typ != TypeBool {
		panic(fmt.Sprintf("AsBool called on %s value", v.typ))
	}
	return v.boolVal
}

// AsInt returns the integer value. Panics if not an int.
func (v Value) AsInt() int64 {
	if v.typ != TypeInt {
		panic(fmt.Sprintf("AsInt called on %s value", v.typ))
	}
	return v.intVal
}

// AsDouble returns the double value. Panics if not a double.
func (v Value) AsDouble() float64 {
	if v.typ != TypeDouble {
		panic(fmt.Sprintf("AsDouble called on %s value", v.typ))
	}
	return v.doubleVal
}

// AsString returns the string value. Panics if not a string.
func (v Value) AsString() string {
	if v.typ != TypeString {
		panic(fmt.Sprintf("AsString called on %s value", v.typ))
	}
	return v.stringVal
}

// AsBytes returns the bytes value. Panics if not bytes.
func (v Value) AsBytes() []byte {
	if v.typ != TypeBytes {
		panic(fmt.Sprintf("AsBytes called on %s value", v.typ))
	}
	return v.bytesVal
}

// AsList returns the list value. Panics if not a list.
func (v Value) AsList() []Value {
	if v.typ != TypeList {
		panic(fmt.Sprintf("AsList called on %s value", v.typ))
	}
	return v.listVal
}

// AsMap returns the map value. Panics if not a map.
func (v Value) AsMap() *OrderedMap {
	if v.typ != TypeMap {
		panic(fmt.Sprintf("AsMap called on %s value", v.typ))
	}
	return v.mapVal
}

// AsNumber returns the numeric value as float64. Works for int and double types.
func (v Value) AsNumber() (float64, bool) {
	switch v.typ {
	case TypeInt:
		return float64(v.intVal), true
	case TypeDouble:
		return v.doubleVal, true
	default:
		return 0, false
	}
}

// Truthy returns the truthiness of a value per GCW semantics.
// Only false and null are falsy; 0, empty string, empty list, empty map are truthy.
func (v Value) Truthy() bool {
	switch v.typ {
	case TypeNull:
		return false
	case TypeBool:
		return v.boolVal
	default:
		return true
	}
}

// Clone creates a deep copy of the value.
func (v Value) Clone() Value {
	switch v.typ {
	case TypeList:
		items := make([]Value, len(v.listVal))
		for i, item := range v.listVal {
			items[i] = item.Clone()
		}
		return NewList(items)
	case TypeMap:
		return NewMap(v.mapVal.Clone())
	case TypeBytes:
		b := make([]byte, len(v.bytesVal))
		copy(b, v.bytesVal)
		return NewBytes(b)
	default:
		return v // scalar types are value-copied
	}
}

// Equal tests deep equality between two values.
func (v Value) Equal(other Value) bool {
	if v.typ != other.typ {
		// int and double can be compared
		if (v.typ == TypeInt || v.typ == TypeDouble) && (other.typ == TypeInt || other.typ == TypeDouble) {
			a, _ := v.AsNumber()
			b, _ := other.AsNumber()
			return a == b
		}
		return false
	}
	switch v.typ {
	case TypeNull:
		return true
	case TypeBool:
		return v.boolVal == other.boolVal
	case TypeInt:
		return v.intVal == other.intVal
	case TypeDouble:
		return v.doubleVal == other.doubleVal
	case TypeString:
		return v.stringVal == other.stringVal
	case TypeBytes:
		if len(v.bytesVal) != len(other.bytesVal) {
			return false
		}
		for i := range v.bytesVal {
			if v.bytesVal[i] != other.bytesVal[i] {
				return false
			}
		}
		return true
	case TypeList:
		if len(v.listVal) != len(other.listVal) {
			return false
		}
		for i := range v.listVal {
			if !v.listVal[i].Equal(other.listVal[i]) {
				return false
			}
		}
		return true
	case TypeMap:
		if v.mapVal.Len() != other.mapVal.Len() {
			return false
		}
		for _, k := range v.mapVal.Keys() {
			ov, ok := other.mapVal.Get(k)
			if !ok {
				return false
			}
			mv, _ := v.mapVal.Get(k)
			if !mv.Equal(ov) {
				return false
			}
		}
		return true
	}
	return false
}

// String returns a human-readable representation of the value for debugging.
func (v Value) String() string {
	switch v.typ {
	case TypeNull:
		return "null"
	case TypeBool:
		if v.boolVal {
			return "true"
		}
		return "false"
	case TypeInt:
		return fmt.Sprintf("%d", v.intVal)
	case TypeDouble:
		if v.doubleVal == math.Trunc(v.doubleVal) && !math.IsInf(v.doubleVal, 0) {
			return fmt.Sprintf("%.1f", v.doubleVal)
		}
		return fmt.Sprintf("%g", v.doubleVal)
	case TypeString:
		return v.stringVal
	case TypeBytes:
		return fmt.Sprintf("<bytes len=%d>", len(v.bytesVal))
	case TypeList:
		parts := make([]string, len(v.listVal))
		for i, item := range v.listVal {
			parts[i] = item.String()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case TypeMap:
		parts := make([]string, 0, v.mapVal.Len())
		for _, k := range v.mapVal.Keys() {
			val, _ := v.mapVal.Get(k)
			parts = append(parts, fmt.Sprintf("%s: %s", k, val.String()))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return "<unknown>"
}

// MarshalJSON converts a Value to JSON, matching GCW serialization.
func (v Value) MarshalJSON() ([]byte, error) {
	switch v.typ {
	case TypeNull:
		return []byte("null"), nil
	case TypeBool:
		if v.boolVal {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case TypeInt:
		return json.Marshal(v.intVal)
	case TypeDouble:
		return json.Marshal(v.doubleVal)
	case TypeString:
		return json.Marshal(v.stringVal)
	case TypeBytes:
		// GCW serializes bytes as their UTF-8 string representation in JSON
		return json.Marshal(string(v.bytesVal))
	case TypeList:
		items := make([]json.RawMessage, len(v.listVal))
		for i, item := range v.listVal {
			b, err := item.MarshalJSON()
			if err != nil {
				return nil, err
			}
			items[i] = b
		}
		return json.Marshal(items)
	case TypeMap:
		// Use ordered iteration
		buf := []byte{'{'}
		for i, k := range v.mapVal.Keys() {
			if i > 0 {
				buf = append(buf, ',')
			}
			keyBytes, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			buf = append(buf, keyBytes...)
			buf = append(buf, ':')
			val, _ := v.mapVal.Get(k)
			valBytes, err := val.MarshalJSON()
			if err != nil {
				return nil, err
			}
			buf = append(buf, valBytes...)
		}
		buf = append(buf, '}')
		return buf, nil
	}
	return nil, fmt.Errorf("cannot marshal unknown type %d", v.typ)
}

// ValueFromJSON converts a Go interface{} (from json.Unmarshal) into a Value.
func ValueFromJSON(v interface{}) Value {
	if v == nil {
		return Null
	}
	switch val := v.(type) {
	case bool:
		return NewBool(val)
	case float64:
		// JSON numbers are float64; convert to int if no fractional part
		if val == math.Trunc(val) && !math.IsInf(val, 0) && val >= math.MinInt64 && val <= math.MaxInt64 {
			return NewInt(int64(val))
		}
		return NewDouble(val)
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return NewInt(i)
		}
		if f, err := val.Float64(); err == nil {
			return NewDouble(f)
		}
		return NewString(val.String())
	case string:
		return NewString(val)
	case []interface{}:
		items := make([]Value, len(val))
		for i, item := range val {
			items[i] = ValueFromJSON(item)
		}
		return NewList(items)
	case map[string]interface{}:
		m := NewOrderedMap()
		// JSON maps don't have guaranteed order, sort keys for determinism
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			m.Set(k, ValueFromJSON(val[k]))
		}
		return NewMap(m)
	default:
		return NewString(fmt.Sprintf("%v", val))
	}
}

// ToGoValue converts a Value to a plain Go interface{} suitable for JSON marshaling.
func (v Value) ToGoValue() interface{} {
	switch v.typ {
	case TypeNull:
		return nil
	case TypeBool:
		return v.boolVal
	case TypeInt:
		return v.intVal
	case TypeDouble:
		return v.doubleVal
	case TypeString:
		return v.stringVal
	case TypeBytes:
		return v.bytesVal
	case TypeList:
		result := make([]interface{}, len(v.listVal))
		for i, item := range v.listVal {
			result[i] = item.ToGoValue()
		}
		return result
	case TypeMap:
		result := make(map[string]interface{}, v.mapVal.Len())
		for _, k := range v.mapVal.Keys() {
			val, _ := v.mapVal.Get(k)
			result[k] = val.ToGoValue()
		}
		return result
	}
	return nil
}
