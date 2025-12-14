package models

import (
	"fmt"
	"sync"

	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// Imposter represents a virtual service
type Imposter struct {
	port               int
	protocol           string
	name               string
	stubs              *StubRepository
	logger             *util.Logger
	state              map[string]interface{}
	numberOfRequests   int
	recordRequests     bool
	closeFunc          func(func()) error
	encoding           string
	mu                 sync.RWMutex
	predicateEvaluator *PredicateEvaluator
	behaviorExecutor   *BehaviorExecutor
	defaultResponse    *Response
	middleware         string
	allowInjection     bool
	saveFunc           func(*Imposter) error

	// Config fields for persistence
	allowCORS  bool
	key        string
	cert       string
	mutualAuth bool
	mode       string
	host       string
}

// ImposterInfo contains information about an imposter
type ImposterInfo struct {
	Port             int            `json:"port"`
	Protocol         string         `json:"protocol"`
	Name             string         `json:"name,omitempty"`
	NumberOfRequests int            `json:"numberOfRequests"`
	RecordRequests   bool           `json:"recordRequests,omitempty"`
	Requests         *[]*Request    `json:"requests,omitempty"` // Changed to pointer to slice
	Stubs            []Stub         `json:"stubs,omitempty"`
	Middleware       string         `json:"middleware,omitempty"`
	DefaultResponse  *Response      `json:"defaultResponse,omitempty"`
	AllowCORS        bool           `json:"allowCORS,omitempty"`
	Key              string         `json:"key,omitempty"`
	Cert             string         `json:"cert,omitempty"`
	MutualAuth       bool           `json:"mutualAuth,omitempty"`
	Mode             string         `json:"mode,omitempty"`
	Host             string         `json:"host,omitempty"`
	Links            *ImposterLinks `json:"_links,omitempty"`
}

// ImposterLinks contains hypermedia links for an imposter
type ImposterLinks struct {
	Self  *Link `json:"self"`
	Stubs *Link `json:"stubs,omitempty"`
}

// Link represents a hypermedia link
type Link struct {
	Href string `json:"href"`
}

// NewImposter creates a new imposter
func NewImposter(config *ImposterConfig, logger *util.Logger, allowInjection bool, closeFunc func(func()) error, saveFunc func(*Imposter) error) *Imposter {
	state := make(map[string]interface{})
	encoding := "utf8"

	if config.Mode == "binary" {
		encoding = "base64"
	}

	imp := &Imposter{
		port:             config.Port,
		protocol:         config.Protocol,
		name:             config.Name,
		logger:           logger,
		state:            state,
		numberOfRequests: 0,
		recordRequests:   config.RecordRequests,
		closeFunc:        closeFunc,
		encoding:         encoding,
		defaultResponse:  config.DefaultResponse,
		middleware:       config.Middleware,
		allowInjection:   allowInjection,
		saveFunc:         saveFunc,
		allowCORS:        config.AllowCORS,
		key:              config.Key,
		cert:             config.Cert,
		mutualAuth:       config.MutualAuth,
		mode:             config.Mode,
		host:             config.Host,
	}

	onUpdate := func() {
		if saveFunc != nil {
			if err := saveFunc(imp); err != nil {
				logger.Errorf("Failed to save imposter: %v", err)
			}
		}
	}

	stubs := NewStubRepository(config.Stubs, config.Requests, logger, onUpdate)
	imp.stubs = stubs

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

		var err error
		response, err = imp.evaluateInject(config.Inject, request, requestDetails)
		if err != nil {
			return nil, err
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
		DefaultResponse:  imp.defaultResponse,
		AllowCORS:        imp.allowCORS,
		Key:              imp.key,
		Cert:             imp.cert,
		MutualAuth:       imp.mutualAuth,
		Mode:             imp.mode,
		Host:             imp.host,
	}

	// Include stubs if requested
	includeStubs := true
	if options != nil {
		if val, ok := options["stubs"]; ok {
			if b, ok := val.(bool); ok {
				includeStubs = b
			}
		}
	}

	// Filter stubs based on options
	replayable := false
	if options != nil && options["replayable"] == true {
		replayable = true
	}

	removeProxies := false
	if options != nil && options["removeProxies"] == true {
		removeProxies = true
	}

	if includeStubs {
		allStubs := imp.stubs.GetAll()

		filteredStubs := make([]Stub, 0, len(allStubs))
		for i, stub := range allStubs {
			if removeProxies && stub.IsProxy {
				continue
			}

			if replayable {
				// Create a copy to remove matches and links
				stubCopy := stub
				stubCopy.Matches = nil
				stubCopy.Links = nil
				filteredStubs = append(filteredStubs, stubCopy)
			} else {
				// Add hypermedia links to stub
				stubCopy := stub
				stubCopy.Links = &StubLinks{
					Self: &Link{
						Href: fmt.Sprintf("http://localhost:2525/imposters/%d/stubs/%d", imp.port, i),
					},
				}
				filteredStubs = append(filteredStubs, stubCopy)
			}
		}
		info.Stubs = filteredStubs
	}

	// Include requests if requested
	// If replayable is true, requests should be removed regardless of requests option
	if !replayable && (options == nil || options["requests"] == true) {
		reqs := imp.stubs.LoadRequests()
		// Ensure non-nil slice so pointer is not nil even if empty
		if reqs == nil {
			reqs = make([]*Request, 0)
		}
		info.Requests = &reqs
	}

	// Add hypermedia links (unless replayable is true)
	if !replayable {
		info.Links = &ImposterLinks{
			Self: &Link{
				Href: fmt.Sprintf("http://localhost:2525/imposters/%d", imp.port),
			},
			Stubs: &Link{
				Href: fmt.Sprintf("http://localhost:2525/imposters/%d/stubs", imp.port),
			},
		}
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
