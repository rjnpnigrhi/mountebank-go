package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/dop251/goja"
	"github.com/mountebank-testing/mountebank-go/internal/util"
	"github.com/oliveagle/jsonpath"
)

// BehaviorExecutor executes response behaviors
type BehaviorExecutor struct {
	logger *util.Logger
	state  map[string]interface{}
}

// NewBehaviorExecutor creates a new behavior executor
func NewBehaviorExecutor(logger *util.Logger, state map[string]interface{}) *BehaviorExecutor {
	return &BehaviorExecutor{
		logger: logger,
		state:  state,
	}
}

// Execute executes all behaviors on a response
func (be *BehaviorExecutor) Execute(request *Request, response *Response, behaviors []Behavior) (*Response, error) {
	result := response
	
	for _, behavior := range behaviors {
		var err error
		result, err = be.executeBehavior(request, result, behavior)
		if err != nil {
			return nil, err
		}
	}
	
	return result, nil
}

// executeBehavior executes a single behavior
func (be *BehaviorExecutor) executeBehavior(request *Request, response *Response, behavior Behavior) (*Response, error) {
	if behavior.Wait != nil {
		return be.executeWait(response, behavior.Wait)
	}
	
	if behavior.Decorate != "" {
		return be.executeDecorate(request, response, behavior.Decorate)
	}
	
	if behavior.Copy != nil {
		return be.executeCopy(request, response, behavior.Copy)
	}
	
	if behavior.Lookup != nil {
		return be.executeLookup(request, response, behavior.Lookup)
	}
	
	if behavior.ShellTransform != "" {
		return be.executeShellTransform(request, response, behavior.ShellTransform)
	}
	
	return response, nil
}

// executeWait adds latency to the response
func (be *BehaviorExecutor) executeWait(response *Response, wait *WaitBehavior) (*Response, error) {
	if wait.Milliseconds > 0 {
		time.Sleep(time.Duration(wait.Milliseconds) * time.Millisecond)
	}
	return response, nil
}

// executeDecorate modifies the response using JavaScript
func (be *BehaviorExecutor) executeDecorate(request *Request, response *Response, code string) (*Response, error) {
	vm := goja.New()

	// Create JS-compatible logger
	jsLogger := map[string]interface{}{
		"debug": func(msg string, args ...interface{}) { be.logger.Debugf(msg, args...) },
		"info":  func(msg string, args ...interface{}) { be.logger.Infof(msg, args...) },
		"warn":  func(msg string, args ...interface{}) { be.logger.Warnf(msg, args...) },
		"error": func(msg string, args ...interface{}) { be.logger.Errorf(msg, args...) },
	}

	// Prepare config object
	config := map[string]interface{}{
		"request":  be.requestToMap(request),
		"response": be.responseToMap(response),
		"logger":   jsLogger,
		"state":    be.state,
	}

	vm.Set("config", config)
	vm.Set("logger", jsLogger)

	// Wrap code in a function call
	script := fmt.Sprintf("(%s)(config, config.response, logger)", code)

	val, err := vm.RunString(script)
	if err != nil {
		be.logger.Errorf("Decorate error: %v", err)
		return response, nil
	}

	// If the function returns a value, use it as the new response
	// Otherwise, assume the response object in config was modified in place
	var newResponseMap map[string]interface{}

	if val != nil && !util.IsUndefined(val) && !util.IsNull(val) {
		// Try to export as map
		if exported, ok := val.Export().(map[string]interface{}); ok {
			newResponseMap = exported
		}
	}

	// If no return value, use the modified config.response
	if newResponseMap == nil {
		if respObj, ok := config["response"].(map[string]interface{}); ok {
			newResponseMap = respObj
		}
	}

	if newResponseMap != nil {
		// Convert map back to Response struct
		be.mapToResponse(newResponseMap, response)
	}

	return response, nil
}

// executeCopy copies values from request to response
func (be *BehaviorExecutor) executeCopy(request *Request, response *Response, copies []CopyBehavior) (*Response, error) {
	for _, copy := range copies {
		value := be.extractValue(request, copy.From)
		if value != nil {
			// Apply selector if present
			if copy.Using != nil {
				value = be.applySelector(value, copy.Using)
			}

			if value != nil {
				be.injectValue(response, copy.Into, value)
			}
		}
	}
	return response, nil
}

// applySelector applies a selector to a value
func (be *BehaviorExecutor) applySelector(value interface{}, selector *CopySelector) interface{} {
	if selector == nil {
		return value
	}

	strValue := fmt.Sprintf("%v", value)

	if selector.Method == "regex" {
		re, err := regexp.Compile(selector.Selector)
		if err != nil {
			be.logger.Warnf("Invalid regex selector: %s", selector.Selector)
			return value
		}

		// Find first match
		match := re.FindStringSubmatch(strValue)
		if len(match) > 1 {
			return match[1] // Return the first capture group
		} else if len(match) > 0 {
			return match[0] // Return the whole match
		}
		return nil // No match
	}

	if selector.Method == "jsonpath" {
		res, err := jsonpath.JsonPathLookup(value, selector.Selector)
		if err != nil {
			be.logger.Warnf("JSONPath lookup failed: %v", err)
			return nil
		}
		return res
	}

	if selector.Method == "xpath" {
		var xmlBody string
		if m, ok := value.(map[string]interface{}); ok {
			if b, ok := m["body"].(string); ok {
				xmlBody = b
			}
		} else if s, ok := value.(string); ok {
			xmlBody = s
		}

		if xmlBody == "" {
			return nil
		}

		doc, err := xmlquery.Parse(strings.NewReader(xmlBody))
		if err != nil {
			be.logger.Debugf("XML parse failed: %v", err)
			return nil
		}

		node := xmlquery.FindOne(doc, selector.Selector)
		if node != nil {
			return node.InnerText()
		}
		return nil
	}

	return value
}

// executeLookup looks up values from a data source
func (be *BehaviorExecutor) executeLookup(request *Request, response *Response, lookup *LookupBehavior) (*Response, error) {
	// TODO: Implement lookup behavior with CSV support
	be.logger.Warn("Lookup behavior not yet implemented")
	return response, nil
}

// executeShellTransform transforms response using shell command
func (be *BehaviorExecutor) executeShellTransform(request *Request, response *Response, command string) (*Response, error) {
	// TODO: Implement shell transform
	be.logger.Warn("ShellTransform behavior not yet implemented")
	return response, nil
}

// extractValue extracts a value from a request using a path
func (be *BehaviorExecutor) extractValue(request *Request, path string) interface{} {
	// Simple path extraction (e.g., "body.field" or "headers.Content-Type")
	requestMap := be.requestToMap(request)
	return be.getNestedValue(requestMap, path)
}

// injectValue injects a value into a response using a token
func (be *BehaviorExecutor) injectValue(response *Response, token string, value interface{}) {
	strValue := fmt.Sprintf("%v", value)

	// Replace in body if it's a string
	if strBody, ok := response.Body.(string); ok {
		response.Body = strings.ReplaceAll(strBody, token, strValue)
	}

	// Replace in headers
	for k, v := range response.Headers {
		if strHeader, ok := v.(string); ok {
			response.Headers[k] = strings.ReplaceAll(strHeader, token, strValue)
		}
	}
}

// getNestedValue gets a value from a nested map using dot notation
func (be *BehaviorExecutor) getNestedValue(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

// setNestedValue sets a value in a nested map using dot notation
func (be *BehaviorExecutor) setNestedValue(obj map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	var current map[string]interface{} = obj

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		if val, exists := current[part]; exists {
			if m, ok := val.(map[string]interface{}); ok {
				current = m
			} else {
				// Overwrite non-map value with new map
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		} else {
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
}

// requestToMap converts a request to a map
func (be *BehaviorExecutor) requestToMap(request *Request) map[string]interface{} {
	data, _ := json.Marshal(request)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// responseToMap converts a response to a map
func (be *BehaviorExecutor) responseToMap(response *Response) map[string]interface{} {
	data, _ := json.Marshal(response)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// mapToResponse updates a response from a map
func (be *BehaviorExecutor) mapToResponse(m map[string]interface{}, response *Response) {
	data, _ := json.Marshal(m)
	json.Unmarshal(data, response)
}
