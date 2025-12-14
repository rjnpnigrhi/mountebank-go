package models

import (
	"fmt"
	"sync"

	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// ImposterRepository manages all imposters
type ImposterRepository struct {
	imposters map[int]*Imposter
	mu        sync.RWMutex
	logger    *util.Logger
	dataStore DataStore
}

// NewImposterRepository creates a new imposter repository
func NewImposterRepository(logger *util.Logger, dataStore DataStore) *ImposterRepository {
	return &ImposterRepository{
		imposters: make(map[int]*Imposter),
		logger:    logger,
		dataStore: dataStore,
	}
}

// Save persists the imposter to the data store
func (ir *ImposterRepository) Save(imposter *Imposter) error {
	if ir.dataStore == nil {
		return nil
	}
	return ir.dataStore.Save(imposter)
}

// Add adds an imposter to the repository
func (ir *ImposterRepository) Add(imposter *Imposter) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	port := imposter.Port()
	if _, exists := ir.imposters[port]; exists {
		return util.NewValidationError(fmt.Sprintf("port %d is already in use", port), port)
	}

	ir.imposters[port] = imposter
	
	// Save to data store
	if err := ir.Save(imposter); err != nil {
		ir.logger.Errorf("Failed to save imposter on port %d to data store: %v", port, err)
	}

	ir.logger.Infof("Added imposter on port %d", port)

	return nil
}

// Get retrieves an imposter by port
func (ir *ImposterRepository) Get(port int) (*Imposter, error) {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	imposter, exists := ir.imposters[port]
	if !exists {
		return nil, util.NewMissingResourceError(fmt.Sprintf("imposter not found on port %d", port), port)
	}

	return imposter, nil
}

// Delete removes an imposter
func (ir *ImposterRepository) Delete(port int) (*Imposter, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	imposter, exists := ir.imposters[port]
	if !exists {
		return nil, util.NewMissingResourceError(fmt.Sprintf("imposter not found on port %d", port), port)
	}

	// Stop the imposter
	if err := imposter.Stop(); err != nil {
		ir.logger.Errorf("Error stopping imposter on port %d: %v", port, err)
	}

	delete(ir.imposters, port)
	
	// Remove from data store
	if ir.dataStore != nil {
		if err := ir.dataStore.Delete(port); err != nil {
			ir.logger.Errorf("Failed to remove imposter on port %d from data store: %v", port, err)
		}
	}

	ir.logger.Infof("Deleted imposter on port %d", port)

	return imposter, nil
}

// DeleteAll removes all imposters
func (ir *ImposterRepository) DeleteAll() ([]*Imposter, error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	imposters := make([]*Imposter, 0, len(ir.imposters))

	for port, imposter := range ir.imposters {
		// Stop the imposter
		if err := imposter.Stop(); err != nil {
			ir.logger.Errorf("Error stopping imposter on port %d: %v", port, err)
		}

		imposters = append(imposters, imposter)
	}

	ir.imposters = make(map[int]*Imposter)
	
	// Remove all from data store
	if ir.dataStore != nil {
		if err := ir.dataStore.DeleteAll(); err != nil {
			ir.logger.Errorf("Failed to remove all imposters from data store: %v", err)
		}
	}

	ir.logger.Info("Deleted all imposters")

	return imposters, nil
}

// GetAll returns all imposters
func (ir *ImposterRepository) GetAll() []*Imposter {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	imposters := make([]*Imposter, 0, len(ir.imposters))
	for _, imposter := range ir.imposters {
		imposters = append(imposters, imposter)
	}

	return imposters
}

// Exists checks if an imposter exists on a port
func (ir *ImposterRepository) Exists(port int) bool {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	_, exists := ir.imposters[port]
	return exists
}

// StopAll stops all imposters
func (ir *ImposterRepository) StopAll() error {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	for port, imposter := range ir.imposters {
		if err := imposter.Stop(); err != nil {
			ir.logger.Errorf("Error stopping imposter on port %d: %v", port, err)
		}
	}

	return nil
}
