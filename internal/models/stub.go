package models

import (
	"sync"

	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// StubRepository manages stubs for an imposter
type StubRepository struct {
	stubs    []Stub
	requests []*Request
	mu       sync.RWMutex
	logger   *util.Logger
}

// NewStubRepository creates a new stub repository
func NewStubRepository(stubs []Stub, logger *util.Logger) *StubRepository {
	return &StubRepository{
		stubs:    stubs,
		requests: make([]*Request, 0),
		logger:   logger,
	}
}

// First finds the first stub matching the filter
func (sr *StubRepository) First(filter func([]Predicate) bool) (*StubMatch, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	for i, stub := range sr.stubs {
		if filter(stub.Predicates) {
			return &StubMatch{
				Success:   true,
				Stub:      &sr.stubs[i],
				StubIndex: i,
			}, nil
		}
	}
	
	// Return no match
	return &StubMatch{
		Success:   false,
		Stub:      nil,
		StubIndex: -1,
	}, nil
}

// Add adds a new stub
func (sr *StubRepository) Add(stub Stub) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	sr.stubs = append(sr.stubs, stub)
	return nil
}

// InsertAtIndex inserts a stub at a specific index
func (sr *StubRepository) InsertAtIndex(stub Stub, index int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if index < 0 || index > len(sr.stubs) {
		sr.stubs = append(sr.stubs, stub)
		return nil
	}
	
	// Insert at index
	sr.stubs = append(sr.stubs[:index], append([]Stub{stub}, sr.stubs[index:]...)...)
	return nil
}

// DeleteAtIndex deletes a stub at a specific index
func (sr *StubRepository) DeleteAtIndex(index int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if index < 0 || index >= len(sr.stubs) {
		return util.NewValidationError("invalid stub index", index)
	}
	
	sr.stubs = append(sr.stubs[:index], sr.stubs[index+1:]...)
	return nil
}

// ReplaceAtIndex replaces a stub at a specific index
func (sr *StubRepository) ReplaceAtIndex(stub Stub, index int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if index < 0 || index >= len(sr.stubs) {
		return util.NewValidationError("invalid stub index", index)
	}
	
	sr.stubs[index] = stub
	return nil
}

// ReplaceAll replaces all stubs
func (sr *StubRepository) ReplaceAll(stubs []Stub) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	sr.stubs = stubs
	return nil
}

// GetAll returns all stubs
func (sr *StubRepository) GetAll() []Stub {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	return sr.stubs
}

// AddRequest records a request
func (sr *StubRepository) AddRequest(request *Request) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	sr.requests = append(sr.requests, request)
	return nil
}

// LoadRequests returns all recorded requests
func (sr *StubRepository) LoadRequests() []*Request {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	return sr.requests
}

// DeleteSavedRequests clears all recorded requests
func (sr *StubRepository) DeleteSavedRequests() error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	sr.requests = make([]*Request, 0)
	return nil
}

// StubMatch represents the result of a stub match
type StubMatch struct {
	Success   bool
	Stub      *Stub
	StubIndex int
}

// NextResponse returns the next response from the stub
func (sm *StubMatch) NextResponse() (*ResponseConfig, error) {
	if sm.Stub == nil || len(sm.Stub.Responses) == 0 {
		return &ResponseConfig{
			Is: &Response{},
		}, nil
	}
	
	// For now, just return the first response
	// TODO: Implement response rotation and repeat logic
	return &sm.Stub.Responses[0], nil
}

// RecordMatch records a match for testing purposes
func (sm *StubMatch) RecordMatch(request *Request, response *Response, responseConfig *ResponseConfig, duration int) error {
	// TODO: Implement match recording for debugging
	return nil
}

// StubIndex returns the index of the matched stub
func (rc *ResponseConfig) StubIndex() (int, error) {
	// This would be set by the stub repository during matching
	return 0, nil
}
