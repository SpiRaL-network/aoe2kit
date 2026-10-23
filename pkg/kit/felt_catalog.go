package kit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var feltCatalogPaths = []string{
	"compendiums/felt/data/art-felt-compendium.json",
	"compendiums/felt/data/terrain-felt-compendium.json",
}

type feltCatalogFile struct {
	Entries []map[string]json.RawMessage `json:"entries"`
}

type updateProvenance struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Date string `json:"date"`
}

// validateFeltCatalogs checks the optional update_provenance field when a
// public Felt catalog is present. Catalogs without that field remain valid.
func validateFeltCatalogs(root string) error {
	for _, rel := range feltCatalogPaths {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("felt catalog %s: stat: %w", rel, err)
		}
		if err := validateFeltCatalogFile(path); err != nil {
			return fmt.Errorf("felt catalog %s: %w", rel, err)
		}
	}
	return nil
}

func validateFeltCatalogFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var catalog feltCatalogFile
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	if catalog.Entries == nil {
		return errors.New("missing entries array")
	}
	for i, entry := range catalog.Entries {
		if entry == nil {
			return fmt.Errorf("entries[%d] must be an object", i)
		}
		raw, ok := entry["update_provenance"]
		if !ok {
			continue
		}
		if err := validateUpdateProvenance(raw); err != nil {
			return fmt.Errorf("entries[%d].update_provenance: %w", i, err)
		}
	}
	return nil
}

func validateUpdateProvenance(raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return errors.New("must be an array")
	}
	var items []json.RawMessage
	if err := json.Unmarshal(trimmed, &items); err != nil {
		return fmt.Errorf("parse array: %w", err)
	}
	for i, rawItem := range items {
		var item updateProvenance
		dec := json.NewDecoder(bytes.NewReader(rawItem))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("[%d] must contain only kind, id, and date: %w", i, err)
		}
		if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
			return fmt.Errorf("[%d] must contain one object", i)
		}
		if item.Kind != "feltboard_comment" && item.Kind != "tol_claim" {
			return fmt.Errorf("[%d].kind must be feltboard_comment or tol_claim", i)
		}
		if item.ID == "" || item.ID != strings.TrimSpace(item.ID) {
			return fmt.Errorf("[%d].id must be a non-empty identifier without surrounding whitespace", i)
		}
		if item.Kind == "tol_claim" && (!strings.HasPrefix(item.ID, "claim.felt.") || len(item.ID) == len("claim.felt.")) {
			return fmt.Errorf("[%d].id for tol_claim must be a non-empty claim.felt.* identifier", i)
		}
		date, err := time.Parse("2006-01-02", item.Date)
		if err != nil || date.Format("2006-01-02") != item.Date {
			return fmt.Errorf("[%d].date must be an ISO date in YYYY-MM-DD form", i)
		}
	}
	return nil
}
