package util

import (
	"encoding/json"
	"net"
	"reflect"

	"github.com/dop251/goja"
)

// IsUndefined checks if a goja value is undefined
func IsUndefined(val goja.Value) bool {
	return val == nil || val == goja.Undefined()
}

// IsNull checks if a goja value is null
func IsNull(val goja.Value) bool {
	return val == nil || val == goja.Null()
}

// Clone creates a deep copy of an object
func Clone(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	// Use JSON marshaling for deep clone
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}

	var dst interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		return src
	}

	return dst
}

// Merge merges two maps
func Merge(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy dst
	for k, v := range dst {
		result[k] = v
	}
	
	// Merge src
	for k, v := range src {
		result[k] = v
	}
	
	return result
}

// Defined checks if a value is defined (not nil)
func Defined(v interface{}) bool {
	if v == nil {
		return false
	}
	
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Ptr, reflect.Interface:
		return !val.IsNil()
	default:
		return true
	}
}

// IsObject checks if a value is an object (map or struct)
func IsObject(v interface{}) bool {
	if v == nil {
		return false
	}
	
	val := reflect.ValueOf(v)
	kind := val.Kind()
	
	return kind == reflect.Map || kind == reflect.Struct
}

// SocketName returns a name for a socket connection
func SocketName(conn net.Conn) string {
	return conn.RemoteAddr().String()
}

// SetDeep sets a value deep in a nested structure
func SetDeep(obj map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}
	
	if len(path) == 1 {
		obj[path[0]] = value
		return
	}
	
	key := path[0]
	if _, ok := obj[key]; !ok {
		obj[key] = make(map[string]interface{})
	}
	
	if nested, ok := obj[key].(map[string]interface{}); ok {
		SetDeep(nested, path[1:], value)
	}
}

// ObjFilter filters an object based on ignore list
func ObjFilter(obj map[string]interface{}, ignore map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range obj {
		if _, shouldIgnore := ignore[k]; !shouldIgnore {
			result[k] = v
		}
	}
	
	return result
}

// Contains checks if a slice contains a value
func Contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// ToJSON converts an object to JSON string
func ToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// FromJSON parses JSON string to object
func FromJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
