package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
)

// evaluateInject executes the injection function
func (imp *Imposter) evaluateInject(injectFunction string, request *Request, requestDetails map[string]interface{}) (*Response, error) {
	vm := goja.New()

	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// Set request
	// We need to handle the case where Body is an object (map[string]interface{})
	// but the injection script expects a string (to call JSON.parse).
	// So we create a custom map for the request, and strict-stringify the body if it's an object.
	reqMap := make(map[string]interface{})

	// Convert Struct to Map first to get all fields
	tmpBytes, _ := json.Marshal(request)
	json.Unmarshal(tmpBytes, &reqMap)

	// Custom Body handling
	if request.Body != nil {
		switch v := request.Body.(type) {
		case map[string]interface{}, []interface{}, []map[string]interface{}:
			// It's a structured object/array. Convert to JSON string.
			bodyBytes, err := json.Marshal(v)
			if err == nil {
				reqMap["body"] = string(bodyBytes)
			}
		}
	}

	// Ensure body is present (default to empty string if nil/missing) to avoid undefined in JS
	if _, ok := reqMap["body"]; !ok {
		reqMap["body"] = ""
	}

	// Add 'Body' alias to support scripts using request.Body (deprecated but used in templates)
	reqMap["Body"] = reqMap["body"]

	vm.Set("request", reqMap)

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
	// Polyfill Buffer
	bufferObj := vm.NewObject()

	// Buffer.from(string, encoding)
	bufferObj.Set("from", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Null()
		}

		input := call.Arguments[0].String()
		encoding := "utf8"
		if len(call.Arguments) > 1 {
			encoding = call.Arguments[1].String()
		}

		var data []byte
		if encoding == "base64" {
			// Ignore error for now, similar to how Node might handle invalid input leniently or throw
			// But for injection, panic might catch it.
			d, _ := base64.StdEncoding.DecodeString(input)
			data = d
		} else {
			data = []byte(input)
		}

		// Create a buffer instance object
		bufInstance := vm.NewObject()
		bufInstance.Set("toString", func(call goja.FunctionCall) goja.Value {
			outEncoding := "utf8"
			if len(call.Arguments) > 0 {
				outEncoding = call.Arguments[0].String()
			}

			if outEncoding == "base64" {
				return vm.ToValue(base64.StdEncoding.EncodeToString(data))
			}
			return vm.ToValue(string(data))
		})

		return bufInstance
	})

	// Buffer.alloc(size)
	bufferObj.Set("alloc", func(call goja.FunctionCall) goja.Value {
		size := 0
		if len(call.Arguments) > 0 {
			size = int(call.Arguments[0].ToInteger())
		}
		data := make([]byte, size)

		bufInstance := vm.NewObject()
		bufInstance.Set("toString", func(call goja.FunctionCall) goja.Value {
			outEncoding := "utf8"
			if len(call.Arguments) > 0 {
				outEncoding = call.Arguments[0].String()
			}

			if outEncoding == "base64" {
				return vm.ToValue(base64.StdEncoding.EncodeToString(data))
			}
			return vm.ToValue(string(data))
		})

		return bufInstance
	})

	vm.Set("Buffer", bufferObj)

	vm.Set("console", consoleObj)

	// Wrap in a function call
	// Create JS-compatible logger map
	jsLogger := map[string]interface{}{
		"debug": func(msg string) { imp.logger.Debug(msg) },
		"info":  func(msg string) { imp.logger.Info(msg) },
		"warn":  func(msg string) { imp.logger.Warn(msg) },
		"error": func(msg string) { imp.logger.Error(msg) },
	}

	// Prepare config object
	config := map[string]interface{}{
		"request": reqMap,
		"state":   imp.state,
		"logger":  jsLogger,
	}

	vm.Set("config", config)
	vm.Set("request", reqMap)
	vm.Set("state", imp.state)
	vm.Set("logger", logObj) // Keep global logger for backward compatibility if needed

	// Wrap in a function call
	// We pass 'config' as the first argument to support the standard signature function(config)
	// We also pass request, state, logger for legacy signature function(request, state, logger)
	// Note: If the function is defined as function(config), it gets config.
	// If it is function(request, state, logger), it gets config as the first arg, which might be an issue?
	// Mountebank Node.js inspects function arguments to decide?
	// Actually, Mountebank Node.js passes (config) and relies on users using function(config).
	// Older versions passed (request, response, logger).
	// But let's assume complex_imposter_collection uses function(config).

	// We detect the number of arguments the function expects (function.length)
	// If it expects 1 argument (or 0?), we pass (config).
	// If it expects 3 arguments, we pass (request, state, logger).
	// This supports both legacy and modern Mountebank signatures.
	// We wrap in an IIFE that inspects 'fn'.
	script := fmt.Sprintf(`
		(function() {
			var fn = %s;
			if (typeof fn !== 'function') {
				throw new Error("Injection must evaluate to a function");
			}
			
			// Check arity
			if (fn.length === 3) {
				// Legacy: function(request, state, logger)
				return fn(request, state, logger);
			} else {
				// Standard: function(config)
				return fn(config);
			}
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
