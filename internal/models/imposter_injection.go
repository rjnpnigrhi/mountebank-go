package models

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
)

// evaluateInject executes the injection function
func (imp *Imposter) evaluateInject(injectFunction string, request *Request, requestDetails map[string]interface{}) (*Response, error) {
	vm := goja.New()

	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// Set request
	reqVal := vm.ToValue(request)
	vm.Set("request", reqVal)

	// Set logger
	logObj := vm.NewObject()
	logObj.Set("debug", func(msg string) { imp.logger.Debug(msg) })
	logObj.Set("info", func(msg string) { imp.logger.Info(msg) })
	logObj.Set("warn", func(msg string) { imp.logger.Warn(msg) })
	logObj.Set("error", func(msg string) { imp.logger.Error(msg) })
	vm.Set("logger", logObj)

	// Polyfill console to map to logger
	consoleObj := vm.NewObject()
	logFn := func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		imp.logger.Info(fmt.Sprint(args...))
		return goja.Undefined()
	}
	warnFn := func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		imp.logger.Warn(fmt.Sprint(args...))
		return goja.Undefined()
	}
	errorFn := func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		imp.logger.Error(fmt.Sprint(args...))
		return goja.Undefined()
	}

	consoleObj.Set("log", logFn)
	consoleObj.Set("info", logFn)
	consoleObj.Set("warn", warnFn)
	consoleObj.Set("error", errorFn)
	vm.Set("console", consoleObj)

	// Wrap in a function call
	script := fmt.Sprintf(`
		(function() {
			var fn = %s;
			return fn(request, state, logger);
		})()
	`, injectFunction)

	// Execute with lock to protect state
	imp.mu.Lock()
	defer imp.mu.Unlock()

	// Bind state
	vm.Set("state", imp.state)

	val, err := vm.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("injection execution failed: %w", err)
	}

	// Parse response
	// Expecting object with statusCode, headers, body
	export := val.Export()

	// Convert export to Response
	// Easiest is via JSON roundtrip for type safety
	jsonBytes, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal injection result: %w", err)
	}

	var response Response
	if err := json.Unmarshal(jsonBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal injection result to Response: %w", err)
	}

	return &response, nil
}
