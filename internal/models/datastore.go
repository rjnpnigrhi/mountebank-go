package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// DataStore interface for imposter persistence
type DataStore interface {
	// Load loads all imposters from the store
	Load() ([]*ImposterConfig, error)
	
	// Save persists an imposter to the store
	Save(imposter *Imposter) error
	
	// Delete removes an imposter from the store
	Delete(port int) error
	
	// DeleteAll removes all imposters from the store
	DeleteAll() error
}

// NoOpDataStore is a data store that does nothing
type NoOpDataStore struct{}

func (s *NoOpDataStore) Load() ([]*ImposterConfig, error) { return nil, nil }
func (s *NoOpDataStore) Save(imposter *Imposter) error    { return nil }
func (s *NoOpDataStore) Delete(port int) error            { return nil }
func (s *NoOpDataStore) DeleteAll() error                 { return nil }

// FileSystemDataStore implements DataStore using the filesystem
type FileSystemDataStore struct {
	datadir string
	logger  *util.Logger
}

// NewFileSystemDataStore creates a new file system data store
func NewFileSystemDataStore(datadir string, logger *util.Logger) *FileSystemDataStore {
	return &FileSystemDataStore{
		datadir: datadir,
		logger:  logger,
	}
}

// Load loads all imposters from the datadir
func (s *FileSystemDataStore) Load() ([]*ImposterConfig, error) {
	if s.datadir == "" {
		return nil, nil
	}

	files, err := os.ReadDir(s.datadir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var configs []*ImposterConfig

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filename := filepath.Join(s.datadir, file.Name())
			data, err := os.ReadFile(filename)
			if err != nil {
				s.logger.Errorf("Failed to read imposter file %s: %v", filename, err)
				continue
			}

			var config ImposterConfig
			if err := json.Unmarshal(data, &config); err != nil {
				s.logger.Errorf("Failed to parse imposter file %s: %v", filename, err)
				continue
			}
			configs = append(configs, &config)
		}
	}
	return configs, nil
}

// Save persists the imposter to disk
func (s *FileSystemDataStore) Save(imposter *Imposter) error {
	if s.datadir == "" {
		return nil
	}

	// Ensure datadir exists
	if err := os.MkdirAll(s.datadir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(s.datadir, fmt.Sprintf("%d.json", imposter.Port()))

	// Convert to JSON
	info := imposter.ToJSON(nil)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Delete removes an imposter from disk
func (s *FileSystemDataStore) Delete(port int) error {
	if s.datadir == "" {
		return nil
	}

	filename := filepath.Join(s.datadir, fmt.Sprintf("%d.json", port))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DeleteAll removes all imposters from disk
func (s *FileSystemDataStore) DeleteAll() error {
	if s.datadir == "" {
		return nil
	}

	files, err := os.ReadDir(s.datadir)
	if err != nil {
		return nil // Ignore error if dir doesn't exist
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filename := filepath.Join(s.datadir, file.Name())
			if err := os.Remove(filename); err != nil {
				s.logger.Errorf("Failed to remove imposter file %s: %v", filename, err)
			}
		}
	}
	return nil
}
