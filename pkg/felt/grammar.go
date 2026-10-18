// Package felt describes reviewed visual connector behavior without game assets.
package felt

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed connectors.json
var connectorData []byte

type Variant struct {
	Index        int         `json:"variant"`
	Axis         string      `json:"axis,omitempty"`
	Role         string      `json:"role"`
	RendersAlong []string    `json:"renders_along,omitempty"`
	RotatesTo    map[int]int `json:"rotates_to,omitempty"`
	Evidence     string      `json:"evidence"`
	Note         string      `json:"note,omitempty"`
}

type Entry struct {
	UnitID   int       `json:"unit_const"`
	Name     string    `json:"name,omitempty"`
	Contexts []string  `json:"contexts,omitempty"`
	Grammar  []Variant `json:"grammar"`
}

type Catalog struct {
	Schema  string  `json:"schema"`
	Entries []Entry `json:"entries"`
}

// Load reads a felt compendium with per-entry grammar, or the built-in reviewed
// connector subset. Variant counts are supplied separately by the caller's DAT.
func Load(path string) (Catalog, error) {
	b := connectorData
	if path != "" {
		var err error
		b, err = os.ReadFile(path)
		if err != nil {
			return Catalog{}, err
		}
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func (c Catalog) Find(id int) (Entry, bool) {
	for _, e := range c.Entries {
		if e.UnitID == id {
			return e, true
		}
	}
	return Entry{}, false
}

func (c Catalog) Validate() error {
	ids := map[int]bool{}
	axisOK := func(s string) bool { return s == "x" || s == "y" || s == "diag_ne" || s == "diag_nw" || s == "none" }
	for _, e := range c.Entries {
		if e.UnitID < 0 || ids[e.UnitID] {
			return fmt.Errorf("invalid or duplicate grammar unit %d", e.UnitID)
		}
		ids[e.UnitID] = true
		seen := map[int]bool{}
		reviewed := map[int]bool{}
		for _, v := range e.Grammar {
			if v.Index < 0 || seen[v.Index] || (v.Axis != "" && !axisOK(v.Axis)) || v.Role == "" || v.Evidence == "" {
				return fmt.Errorf("invalid grammar for unit %d variant %d", e.UnitID, v.Index)
			}
			seen[v.Index] = true
			reviewed[v.Index] = v.Evidence != "unmapped"
			for _, axis := range v.RendersAlong {
				if !axisOK(axis) || axis == "none" {
					return fmt.Errorf("invalid renders_along for unit %d", e.UnitID)
				}
			}
		}
		for _, v := range e.Grammar {
			for degrees, target := range v.RotatesTo {
				if (degrees != 90 && degrees != 180 && degrees != 270) || !reviewed[target] || v.Evidence == "unmapped" {
					return fmt.Errorf("invalid rotation mapping for unit %d variant %d", e.UnitID, v.Index)
				}
			}
		}
		for _, context := range e.Contexts {
			if context != "grass" && context != "water" {
				return fmt.Errorf("invalid grammar context %q", context)
			}
		}
	}
	return nil
}

// Skeleton fills all DAT facings, leaving unreviewed fields explicitly unmapped.
func Skeleton(e Entry, count int) (Entry, error) {
	if count < 1 || count > 256 {
		return Entry{}, fmt.Errorf("variant count must be 1..256")
	}
	known := map[int]Variant{}
	for _, v := range e.Grammar {
		if v.Index >= count {
			return Entry{}, fmt.Errorf("grammar unit %d variant %d outside DAT count %d", e.UnitID, v.Index, count)
		}
		known[v.Index] = v
	}
	e.Grammar = make([]Variant, count)
	for i := range e.Grammar {
		v, ok := known[i]
		if !ok {
			v = Variant{Index: i, Role: "unmapped", Evidence: "unmapped"}
		}
		e.Grammar[i] = v
	}
	return e, nil
}

// Rotate requires an explicit reviewed mapping. An axis alone cannot prove
// which corner, end, facing or chirality should replace an existing variant.
func (c Catalog) Rotate(id, variant, degrees int) (int, error) {
	if degrees == 0 {
		return variant, nil
	}
	e, ok := c.Find(id)
	if ok {
		for _, v := range e.Grammar {
			if v.Index == variant {
				if target, ok := v.RotatesTo[degrees]; ok {
					return target, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("unit %d variant %d has no reviewed %d-degree rotation", id, variant, degrees)
}
