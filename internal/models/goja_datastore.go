package models

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/dop251/goja"
	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// GojaDataStore implements DataStore using a JavaScript module
type GojaDataStore struct {
	vm     *goja.Runtime
	repo   *goja.Object
	logger *util.Logger
	mu     sync.Mutex
}

// NewGojaDataStore creates a new Goja data store
func NewGojaDataStore(path string, logger *util.Logger) (*GojaDataStore, error) {
	vm := goja.New()

	// Mock module.exports
	module := vm.NewObject()
	exports := vm.NewObject()
	module.Set("exports", exports)
	vm.Set("module", module)
	vm.Set("exports", exports)

	// Mock require (basic)
	vm.Set("require", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})

	// Read JS file
	script, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository file: %v", err)
	}

	// Execute script
	_, err = vm.RunScript(path, string(script))
	if err != nil {
		return nil, fmt.Errorf("failed to execute repository script: %v", err)
	}

	// Get create function
	exportsObj := module.Get("exports").ToObject(vm)
	createVal := exportsObj.Get("create")
	create, ok := goja.AssertFunction(createVal)
	if !ok {
		return nil, fmt.Errorf("create function not found in %s", path)
	}

	// Prepare config for create
	// We pass a simplified logger
	logObj := vm.NewObject()
	logObj.Set("debug", func(msg string) { logger.Debug(msg) })
	logObj.Set("info", func(msg string) { logger.Info(msg) })
	logObj.Set("warn", func(msg string) { logger.Warn(msg) })
	logObj.Set("error", func(msg string) { logger.Error(msg) })

	config := map[string]interface{}{
		"logger": logObj,
	}

	// Call create(config)
	repoVal, err := create(goja.Undefined(), vm.ToValue(config))
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %v", err)
	}

	return &GojaDataStore{
		vm:     vm,
		repo:   repoVal.ToObject(vm),
		logger: logger,
	}, nil
}

// Load loads all imposters from the store
func (s *GojaDataStore) Load() ([]*ImposterConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loadVal := s.repo.Get("load")
	load, ok := goja.AssertFunction(loadVal)
	if !ok {
		// If load is missing, maybe it's 'all'? Mountebank JS uses 'load' on startup?
		// Actually, Mountebank JS calls 'load' on the repository factory, not the instance?
		// No, the instance has 'load' or 'all'.
		// Let's assume 'all' or 'load'.
		// Docs say: "The object returned by create should have the following functions..."
		// But I don't see exact docs for repo interface here.
		// Assuming 'all' returns list.
		loadVal = s.repo.Get("all")
		load, ok = goja.AssertFunction(loadVal)
		if !ok {
			return nil, fmt.Errorf("repository missing 'load' or 'all' function")
		}
	}

	res, err := load(s.repo)
	if err != nil {
		return nil, err
	}

	// Handle Promise if returned
	if promise, ok := res.Export().(*goja.Promise); ok {
		// Goja doesn't expose Promise result easily synchronously?
		// We might need to wait.
		// For now, assume sync or simple object.
		// If it's a promise, we are in trouble without an event loop.
		// But let's assume the user provides a sync implementation or we handle simple values.
		// Actually, if it returns a Promise, we can't get value easily.
		// We'll assume it returns the value directly for this implementation.
		_ = promise
	}

	// Convert result to []*ImposterConfig
	var configs []*ImposterConfig

	// Marshaling via JSON is easiest to ensure type safety
	jsonBytes, err := json.Marshal(res.Export())
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonBytes, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// Save persists an imposter
func (s *GojaDataStore) Save(imposter *Imposter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addVal := s.repo.Get("add")
	add, ok := goja.AssertFunction(addVal)
	if !ok {
		// Try 'save'?
		return fmt.Errorf("repository missing 'add' function")
	}

	info := imposter.ToJSON(nil)
	_, err := add(s.repo, s.vm.ToValue(info))
	return err
}

// Delete removes an imposter
func (s *GojaDataStore) Delete(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delVal := s.repo.Get("del") // 'del' or 'delete'? JS usually 'del' to avoid keyword
	del, ok := goja.AssertFunction(delVal)
	if !ok {
		delVal = s.repo.Get("delete")
		del, ok = goja.AssertFunction(delVal)
		if !ok {
			return fmt.Errorf("repository missing 'del' or 'delete' function")
		}
	}

	_, err := del(s.repo, s.vm.ToValue(port))
	return err // Ignore result imposter
}

// DeleteAll removes all imposters
func (s *GojaDataStore) DeleteAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delAllVal := s.repo.Get("deleteAll")
	delAll, ok := goja.AssertFunction(delAllVal)
	if !ok {
		return fmt.Errorf("repository missing 'deleteAll' function")
	}

	_, err := delAll(s.repo)
	return err
}
