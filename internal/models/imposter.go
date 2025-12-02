package models

import (
	"fmt"
	"sync"

	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// Imposter represents a virtual service
type Imposter struct {
	port              int
	protocol          string
	name              string
	stubs             *StubRepository
	logger            *util.Logger
	state             map[string]interface{}
	numberOfRequests  int
	recordRequests    bool
	closeFunc         func(func()) error
	encoding          string
	mu                sync.RWMutex
	predicateEvaluator *PredicateEvaluator
	behaviorExecutor   *BehaviorExecutor
	defaultResponse    *Response
	middleware         string
	allowInjection     bool
}

// ImposterInfo contains information about an imposter
type ImposterInfo struct {
	Port             int                    `json:"port"`
	Protocol         string                 `json:"protocol"`
	Name             string                 `json:"name,omitempty"`
	NumberOfRequests int                    `json:"numberOfRequests"`
	RecordRequests   bool                   `json:"recordRequests,omitempty"`
	Requests         []*Request             `json:"requests,omitempty"`
	Stubs            []Stub                 `json:"stubs,omitempty"`
	Middleware       string                 `json:"middleware,omitempty"`
}

// NewImposter creates a new imposter
func NewImposter(config *ImposterConfig, logger *util.Logger, allowInjection bool, closeFunc func(func()) error) *Imposter {
	state := make(map[string]interface{})
	stubs := NewStubRepository(config.Stubs, logger)
	encoding := "utf8"
	
	if config.Mode == "binary" {
		encoding = "base64"
	}

	imp := &Imposter{
		port:              config.Port,
		protocol:          config.Protocol,
		name:              config.Name,
		stubs:             stubs,
		logger:            logger,
		state:             state,
		numberOfRequests:  0,
		recordRequests:    config.RecordRequests,
		closeFunc:         closeFunc,
		encoding:          encoding,
		defaultResponse:   config.DefaultResponse,
		middleware:        config.Middleware,
		allowInjection:    allowInjection,
	}

	imp.predicateEvaluator = NewPredicateEvaluator(encoding, logger, state, allowInjection)
	imp.behaviorExecutor = NewBehaviorExecutor(logger, state, allowInjection)

	return imp
}

// GetResponseFor generates a response for a request
func (imp *Imposter) GetResponseFor(request *Request, requestDetails map[string]interface{}) (*Response, error) {
	imp.mu.Lock()
	imp.numberOfRequests++
	imp.mu.Unlock()

	// Record request if enabled
	if imp.recordRequests {
		imp.stubs.AddRequest(request)
	}

	// Execute middleware
	middlewareResponse, err := imp.executeMiddleware(request)
	if err != nil {
		return nil, err
	}
	if middlewareResponse != nil {
		return middlewareResponse, nil
	}

	// Find matching stub
	match, err := imp.findFirstMatch(request)
	if err != nil {
		return nil, err
	}

	if !match.Success {
		if imp.defaultResponse != nil {
			return imp.defaultResponse, nil
		}
		// Default to 200 OK empty body if no defaultResponse configured
		return &Response{StatusCode: 200}, nil
	}

	// Get response config
	responseConfig, err := match.NextResponse()
	if err != nil {
		return nil, err
	}

	// Generate response
	response, err := imp.resolveResponse(responseConfig, request, requestDetails)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// findFirstMatch finds the first stub that matches the request
func (imp *Imposter) findFirstMatch(request *Request) (*StubMatch, error) {
	filter := func(predicates []Predicate) bool {
		if len(predicates) == 0 {
			return true
		}

		for _, predicate := range predicates {
			if !imp.predicateEvaluator.Evaluate(predicate, request) {
				return false
			}
		}

		return true
	}

	return imp.stubs.First(filter)
}

// resolveResponse resolves a response configuration to an actual response
func (imp *Imposter) resolveResponse(config *ResponseConfig, request *Request, requestDetails map[string]interface{}) (*Response, error) {
	var response *Response

	// Handle different response types
	if config.Is != nil {
		// Static response
		response = config.Is
	} else if config.Proxy != nil {
		// Proxy response
		// TODO: Implement proxy support
		imp.logger.Warn("Proxy responses not yet implemented")
		response = &Response{
			StatusCode: 200,
			Body:       "Proxy not implemented",
		}
	} else if config.Inject != "" {
		// Injected response
		if !imp.allowInjection {
			return nil, fmt.Errorf("invalid injection: JavaScript injection is not allowed unless mb is run with the --allowInjection flag")
		}
		// TODO: Implement JavaScript injection
		imp.logger.Warn("Inject responses not yet implemented")
		response = &Response{
			StatusCode: 200,
			Body:       "Inject not implemented",
		}
	} else if config.Fault != nil {
		// Fault response
		// TODO: Implement fault injection
		imp.logger.Warn("Fault responses not yet implemented")
		response = &Response{
			StatusCode: 500,
			Body:       "Fault injection not implemented",
		}
	} else {
		// Default response
		response = &Response{
			StatusCode: 200,
		}
	}

	// Apply behaviors
	if len(config.Behaviors) > 0 {
		var err error
		response, err = imp.behaviorExecutor.Execute(request, response, config.Behaviors)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

// Stop stops the imposter
func (imp *Imposter) Stop() error {
	return imp.closeFunc(func() {
		imp.logger.Info("Imposter stopped")
	})
}

// ResetRequests clears all recorded requests
func (imp *Imposter) ResetRequests() error {
	imp.mu.Lock()
	imp.numberOfRequests = 0
	imp.mu.Unlock()

	return imp.stubs.DeleteSavedRequests()
}

// DeleteSavedProxyResponses removes all stubs recorded by a proxy
func (imp *Imposter) DeleteSavedProxyResponses() error {
	return imp.stubs.DeleteSavedProxyResponses()
}

// ToJSON converts the imposter to JSON format
func (imp *Imposter) ToJSON(options map[string]interface{}) *ImposterInfo {
	imp.mu.RLock()
	defer imp.mu.RUnlock()

	info := &ImposterInfo{
		Port:             imp.port,
		Protocol:         imp.protocol,
		Name:             imp.name,
		NumberOfRequests: imp.numberOfRequests,
		RecordRequests:   imp.recordRequests,
		Middleware:       imp.middleware,
	}

	// Include stubs
	allStubs := imp.stubs.GetAll()
	
	// Filter stubs based on options
	replayable := false
	if options != nil && options["replayable"] == true {
		replayable = true
	}
	
	removeProxies := false
	if options != nil && options["removeProxies"] == true {
		removeProxies = true
	}

	filteredStubs := make([]Stub, 0, len(allStubs))
	for _, stub := range allStubs {
		if removeProxies && stub.IsProxy {
			continue
		}
		
		if replayable {
			// Create a copy to remove matches
			stubCopy := stub
			stubCopy.Matches = nil
			filteredStubs = append(filteredStubs, stubCopy)
		} else {
			filteredStubs = append(filteredStubs, stub)
		}
	}
	info.Stubs = filteredStubs

	// Include requests if requested
	// If replayable is true, requests should be removed regardless of requests option
	if !replayable && (options == nil || options["requests"] == true) {
		info.Requests = imp.stubs.LoadRequests()
	}

	return info
}

// Port returns the imposter's port
func (imp *Imposter) Port() int {
	return imp.port
}

// Protocol returns the imposter's protocol
func (imp *Imposter) Protocol() string {
	return imp.protocol
}

// Stubs returns the stub repository
func (imp *Imposter) Stubs() *StubRepository {
	return imp.stubs
}

// executeMiddleware executes global middleware
func (imp *Imposter) executeMiddleware(request *Request) (*Response, error) {
	/*
	if imp.middleware == "" {
		return nil, nil
	}

	if !imp.allowInjection {
		return nil, fmt.Errorf("invalid injection: JavaScript injection is not allowed unless mb is run with the --allowInjection flag")
	}

	vm := goja.New()
    // ...
    */
    return nil, nil
}
