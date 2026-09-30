package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

type Entry struct {
	Name         string `json:"name"`
	Family       string `json:"family,omitempty"`
	Description  string `json:"description,omitempty"`
	Tier         string `json:"tier,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
	TriggerCount int    `json:"trigger_count,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type Registry map[string]Entry

func Load(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Registry{}, nil
		}
		return nil, err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}
	if registry == nil {
		registry = Registry{}
	}
	return registry, nil
}

func Save(path string, registry Registry) error {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func Register(path, hash string, entry Entry) (Registry, error) {
	if hash == "" {
		return nil, fmt.Errorf("empty graph hash")
	}
	if entry.Name == "" {
		return nil, fmt.Errorf("registry entry needs a name")
	}
	registry, err := Load(path)
	if err != nil {
		return nil, err
	}
	registry[hash] = entry
	return registry, Save(path, registry)
}

func Keys(registry Registry) []string {
	keys := make([]string, 0, len(registry))
	for key := range registry {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
