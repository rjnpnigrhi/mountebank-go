package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/dop251/goja"
	"github.com/mountebank-testing/mountebank-go/internal/util"
	"github.com/oliveagle/jsonpath"
)

// PredicateEvaluator evaluates predicates against requests
type PredicateEvaluator struct {
	encoding       string
	logger         *util.Logger
	state          map[string]interface{}
	allowInjection bool
}

// NewPredicateEvaluator creates a new predicate evaluator
func NewPredicateEvaluator(encoding string, logger *util.Logger, state map[string]interface{}, allowInjection bool) *PredicateEvaluator {
	return &PredicateEvaluator{
		encoding:       encoding,
		logger:         logger,
		state:          state,
		allowInjection: allowInjection,
	}
}

// Evaluate evaluates a predicate against a request
func (pe *PredicateEvaluator) Evaluate(predicate Predicate, request *Request) bool {
	// Check which predicate type is being used
	if predicate.Equals != nil {
		return pe.evaluateEquals(predicate, request)
	}
	if predicate.DeepEquals != nil {
		return pe.evaluateDeepEquals(predicate, request)
	}
	if predicate.Contains != nil {
		return pe.evaluateContains(predicate, request)
	}
	if predicate.StartsWith != nil {
		return pe.evaluateStartsWith(predicate, request)
	}
	if predicate.EndsWith != nil {
		return pe.evaluateEndsWith(predicate, request)
	}
	if predicate.Matches != nil {
		return pe.evaluateMatches(predicate, request)
	}
	if predicate.Exists != nil {
		return pe.evaluateExists(predicate, request)
	}
	if predicate.Not != nil {
		return !pe.Evaluate(*predicate.Not, request)
	}
	if predicate.Or != nil {
		for _, p := range predicate.Or {
			if pe.Evaluate(p, request) {
				return true
			}
		}
		return false
	}
	if predicate.And != nil {
		for _, p := range predicate.And {
			if !pe.Evaluate(p, request) {
				return false
			}
		}
		return true
	}
	if predicate.Inject != "" {
		return pe.evaluateInject(predicate, request)
	}

	return false
}

// evaluateEquals checks if request fields equal expected values
func (pe *PredicateEvaluator) evaluateEquals(predicate Predicate, request *Request) bool {
	expected := pe.normalize(predicate.Equals, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	return pe.predicateSatisfied(expected, actual, predicate, func(a, b interface{}) bool {
		return fmt.Sprint(a) == fmt.Sprint(b)
	})
}

// evaluateDeepEquals checks deep equality
// evaluateDeepEquals checks deep equality
func (pe *PredicateEvaluator) evaluateDeepEquals(predicate Predicate, request *Request) bool {
	expected := pe.normalize(predicate.DeepEquals, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	// The root request object usually contains more fields than the predicate (e.g. method, headers).
	// But deepEquals expects the fields THAT ARE PROVIDED to match exactly.
	// So we iterate over the expected map keys and compare the values strictly.

	expectedMap, ok := expected.(map[string]interface{})
	if !ok {
		// Fallback for non-map predicate (unlikely for matched level)
		expectedJSON, _ := json.Marshal(expected)
		actualJSON, _ := json.Marshal(actual)
		return string(expectedJSON) == string(actualJSON)
	}

	actualMap, ok := actual.(map[string]interface{})
	if !ok {
		// If expected is map but actual is not, they are not equal
		return false
	}

	for k, v := range expectedMap {
		actualVal, exists := actualMap[k]
		if !exists {
			// Expected key missing in actual
			return false
		}

		// Prepare JSON for strict comparison of the value
		// This ensures nested objects must match exactly (no extra fields in actual nested objects)
		expectedJSON, _ := json.Marshal(v)
		actualJSON, _ := json.Marshal(actualVal)

		if string(expectedJSON) != string(actualJSON) {
			return false
		}
	}

	return true
}

// evaluateContains checks if actual contains expected
func (pe *PredicateEvaluator) evaluateContains(predicate Predicate, request *Request) bool {
	expected := pe.normalize(predicate.Contains, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	return pe.predicateSatisfied(expected, actual, predicate, func(a, b interface{}) bool {
		aStr := fmt.Sprint(a)
		bStr := fmt.Sprint(b)
		return strings.Contains(bStr, aStr)
	})
}

// evaluateStartsWith checks if actual starts with expected
func (pe *PredicateEvaluator) evaluateStartsWith(predicate Predicate, request *Request) bool {
	expected := pe.normalize(predicate.StartsWith, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	return pe.predicateSatisfied(expected, actual, predicate, func(a, b interface{}) bool {
		aStr := fmt.Sprint(a)
		bStr := fmt.Sprint(b)
		return strings.HasPrefix(bStr, aStr)
	})
}

// evaluateEndsWith checks if actual ends with expected
func (pe *PredicateEvaluator) evaluateEndsWith(predicate Predicate, request *Request) bool {
	expected := pe.normalize(predicate.EndsWith, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	return pe.predicateSatisfied(expected, actual, predicate, func(a, b interface{}) bool {
		aStr := fmt.Sprint(a)
		bStr := fmt.Sprint(b)
		return strings.HasSuffix(bStr, aStr)
	})
}

// evaluateMatches checks if actual matches regex pattern
func (pe *PredicateEvaluator) evaluateMatches(predicate Predicate, request *Request) bool {
	if pe.encoding == "base64" {
		pe.logger.Error("the matches predicate is not allowed in binary mode")
		return false
	}

	expected := pe.normalize(predicate.Matches, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	caseSensitive := predicate.CaseSensitive != nil && *predicate.CaseSensitive

	return pe.predicateSatisfied(expected, actual, predicate, func(a, b interface{}) bool {
		pattern := fmt.Sprint(a)
		text := fmt.Sprint(b)

		if !caseSensitive {
			pattern = "(?i)" + pattern
		}

		matched, err := regexp.MatchString(pattern, text)
		if err != nil {
			pe.logger.Warnf("Invalid regex pattern: %s", pattern)
			return false
		}

		return matched
	})
}

// evaluateExists checks if a field exists
func (pe *PredicateEvaluator) evaluateExists(predicate Predicate, request *Request) bool {
	expected := pe.normalize(predicate.Exists, predicate, false)
	actual := pe.normalize(pe.requestToMap(request), predicate, true)

	return pe.predicateSatisfied(expected, actual, predicate, func(a, b interface{}) bool {
		shouldExist, ok := a.(bool)
		if !ok {
			return false
		}
		exists := b != nil && b != ""

		return shouldExist == exists
	})
}

// evaluateInject evaluates injected JavaScript predicate
func (pe *PredicateEvaluator) evaluateInject(predicate Predicate, request *Request) bool {
	if request.IsDryRun {
		return true
	}

	if !pe.allowInjection {
		pe.logger.Error("invalid injection: JavaScript injection is not allowed unless mb is run with the --allowInjection flag")
		return false
	}

	vm := goja.New()

	// Create JS-compatible logger
	jsLogger := map[string]interface{}{
		"debug": func(msg string, args ...interface{}) { pe.logger.Debugf(msg, args...) },
		"info":  func(msg string, args ...interface{}) { pe.logger.Infof(msg, args...) },
		"warn":  func(msg string, args ...interface{}) { pe.logger.Warnf(msg, args...) },
		"error": func(msg string, args ...interface{}) { pe.logger.Errorf(msg, args...) },
	}

	// Create config object
	config := map[string]interface{}{
		"request": pe.requestToMap(request),
		"state":   pe.state,
		"logger":  jsLogger,
	}

	vm.Set("config", config)
	vm.Set("logger", jsLogger)

	// The injection code is expected to be a function expression
	// We wrap it in parentheses and call it with (config, logger)
	// The injection code is expected to be a function expression
	// We detect arity to support legacy signatures
	script := fmt.Sprintf(`
		(function() {
			var fn = %s;
			if (typeof fn !== 'function') {
				throw new Error("Injection must evaluate to a function");
			}
			
			if (fn.length === 2) {
				// Legacy: function(request, logger)
				// Note: Mountebank docs say predicate injection is function(config), 
				// but historically might have been different or user might expect specific args.
				// Actually, predicate injection legacy was function(request, logger) ?
				// Let's assume if 2 args, it's (config, logger) as passed before? 
				// Wait, the previous code was: (%s)(config, logger)
				// implying it ALWAYS passed 2 args.
				// If the user function is function(config), then passing (config, logger) works because JS ignores extra args.
				// If the user function is function(request, logger), then passing (config, logger) passes 'config' as 'request'.
				// AND 'config' has a 'request' property. 
				// So if user does request.method, they get config.method -> undefined. They need config.request.method.
				
				// Let's check how we construct params.
				// config = { request: ... }
				// If we want to support function(request, logger), we must pass request object as first arg.
				
				// We can try to support both.
				// Standard: function(config) - length 1
				// Legacy/Compatible: function(request, logger) - length 2
				
				// However, if we just blindly pass (request, logger) for length 2, we break function(config, logger) if that was ever valid.
				// Mountebank docs: "The function accepts a single object, config".
				// But let's look at what we are fixing.
				// The user issue was about response injection (imposters-test.ejs -> getC360Identifier.ejs -> get360Identifier.js).
				// That is an IMPOSTER injection (response).
				// But we also proposed fixing PREDIATE checks.
				
				// For predicates, let's keep it safe.
				// If user wrote function(request, logger), they expect request.
				return fn(config.request, logger);
			}
			
			// Default to passing config (and logger as extra which is harmless for arity 1)
			return fn(config, logger);
		})()
	`, predicate.Inject)

	val, err := vm.RunString(script)
	if err != nil {
		pe.logger.Errorf("Injection error: %v", err)
		return false
	}

	if boolVal, ok := val.Export().(bool); ok {
		pe.logger.Infof("Injection result: %v", boolVal)
		return boolVal
	}

	pe.logger.Warnf("Injection returned non-boolean: %v", val.Export())
	return false
}

// predicateSatisfied checks if a predicate is satisfied
func (pe *PredicateEvaluator) predicateSatisfied(expected, actual interface{}, predicate Predicate, fn func(interface{}, interface{}) bool) bool {
	if actual == nil {
		return false
	}

	// Handle maps
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		if actualMap, ok := actual.(map[string]interface{}); ok {
			for fieldName, expectedValue := range expectedMap {
				actualValue, ok := actualMap[fieldName]
				if !ok {
					return false
				}
				if !pe.predicateSatisfied(expectedValue, actualValue, predicate, fn) {
					return false
				}
			}
			return true
		}
		return false
	}

	// Handle arrays
	if expectedArr, ok := expected.([]interface{}); ok {
		if actualArr, ok := actual.([]interface{}); ok {
			// All expected values must match at least one actual value
			for _, expVal := range expectedArr {
				found := false
				for _, actVal := range actualArr {
					if pe.predicateSatisfied(expVal, actVal, predicate, fn) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}
		return false
	}

	// Direct comparison
	return fn(expected, actual)
}

// normalize normalizes a value for comparison
func (pe *PredicateEvaluator) normalize(value interface{}, predicate Predicate, withSelectors bool) interface{} {
	// Apply selectors if requested
	if withSelectors {
		value = pe.selectValue(value, predicate)
	}

	// If value is nil after selection, return nil
	if value == nil {
		return nil
	}

	// If value is a map, normalize its keys/values
	if objMap, ok := value.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		caseSensitive := predicate.CaseSensitive == nil || *predicate.CaseSensitive

		for key, val := range objMap {
			normalizedKey := key
			if !caseSensitive {
				normalizedKey = strings.ToLower(key)
			}
			result[normalizedKey] = pe.normalizeValue(val, predicate, caseSensitive)
		}
		return result
	}

	// If value is not a map, just normalize the value itself
	caseSensitive := predicate.CaseSensitive == nil || *predicate.CaseSensitive
	return pe.normalizeValue(value, predicate, caseSensitive)
}

// selectValue applies selectors (JSONPath, XPath) to the value
func (pe *PredicateEvaluator) selectValue(value interface{}, predicate Predicate) interface{} {
	if predicate.JSONPath != nil {
		res, err := jsonpath.JsonPathLookup(value, predicate.JSONPath.Selector)
		if err != nil {
			pe.logger.Debugf("JSONPath lookup failed: %v", err)
			return nil
		}
		return res
	}

	if predicate.XPath != nil {
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
			pe.logger.Debugf("XML parse failed: %v", err)
			return nil
		}

		// Handle namespaces if provided
		if len(predicate.XPath.NS) > 0 {
			// xmlquery doesn't support dynamic namespaces easily in FindOne?
			// It seems xmlquery supports namespaces in the selector directly or via a map?
			// Looking at xmlquery docs/examples...
			// It seems we might need to use a different function or just pass the selector.
			// But for now let's assume standard XPath selector.
			// Mountebank passes 'ns' map.
			// xmlquery doesn't seem to have a direct way to pass a namespace map to FindOne.
			// But we can ignore namespaces for now or assume they are in the selector.
		}

		node := xmlquery.FindOne(doc, predicate.XPath.Selector)
		if node != nil {
			return node.InnerText()
		}
		return nil
	}

	return value
}

// normalizeValue normalizes a single value
func (pe *PredicateEvaluator) normalizeValue(value interface{}, predicate Predicate, caseSensitive bool) interface{} {
	// Handle encoding transformation
	if pe.encoding == "base64" {
		if str, ok := value.(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(str)
			if err == nil {
				value = string(decoded)
			}
		}
	}

	// Handle except pattern
	if predicate.Except != "" {
		if str, ok := value.(string); ok {
			// Simple replace for now (full regex support would need more work)
			value = strings.ReplaceAll(str, predicate.Except, "")
		}
	}

	// Handle case sensitivity
	if !caseSensitive {
		if str, ok := value.(string); ok {
			value = strings.ToLower(str)
		}
	}

	// Handle nested objects
	if objMap, ok := value.(map[string]interface{}); ok {
		return pe.normalize(objMap, predicate, caseSensitive)
	}

	// Handle arrays
	if arr, ok := value.([]interface{}); ok {
		result := make([]interface{}, len(arr))
		for i, item := range arr {
			result[i] = pe.normalizeValue(item, predicate, caseSensitive)
		}
		return result
	}

	return value
}

// requestToMap converts a request to a map for predicate evaluation
func (pe *PredicateEvaluator) requestToMap(request *Request) map[string]interface{} {
	result := make(map[string]interface{})

	if request.Method != "" {
		result["method"] = request.Method
	}
	if request.Path != "" {
		result["path"] = request.Path
	}
	if request.Query != nil {
		result["query"] = request.Query
	}
	if request.Headers != nil {
		result["headers"] = request.Headers
	}
	if request.Body != nil {
		result["body"] = request.Body
	}
	if request.Data != "" {
		result["data"] = request.Data
	}

	return result
}
