package geotrace

import (
	"encoding/json"
	"os"
	"strings"
)

// SourceAttribution travels with reports, including unknown local query extents.
type SourceAttribution struct {
	Source      string `json:"source"`
	Text        string `json:"attribution"`
	License     string `json:"license"`
	URL         string `json:"url"`
	Dataset     string `json:"dataset,omitempty"`
	QueryBounds *BBox  `json:"query_bounds"`
	ExtentState string `json:"query_extent_state"`
}

func worldCoverAttribution(bounds BBox) SourceAttribution {
	return SourceAttribution{Source: "ESA WorldCover", Text: "(c) ESA WorldCover project 2021 / Contains modified Copernicus Sentinel data (2021) processed by ESA WorldCover consortium", License: "CC BY 4.0", URL: "https://esa-worldcover.org/en/data-access", Dataset: "ESA WorldCover 10 m 2021 v200; https://doi.org/10.5281/zenodo.7254221", QueryBounds: &bounds, ExtentState: "sampling_extent"}
}

func osmRoadAttribution(opts Options, coverage SourceCoverage) (SourceAttribution, bool) {
	provider := strings.ToLower(strings.TrimSpace(opts.RoadsProvider))
	osm := strings.HasPrefix(provider, "osm-")
	if !osm && (provider == "" || provider == "overpass-json") && opts.RoadsPath != "" {
		var doc struct {
			Elements json.RawMessage `json:"elements"`
		}
		if raw, err := os.ReadFile(opts.RoadsPath); err == nil && json.Unmarshal(raw, &doc) == nil && len(doc.Elements) > 0 {
			osm = true
		}
	}
	if !osm {
		return SourceAttribution{}, false
	}
	return SourceAttribution{Source: "OpenStreetMap", Text: "(c) OpenStreetMap contributors", License: "ODbL 1.0", URL: "https://www.openstreetmap.org/copyright", QueryBounds: coverage.QueryBounds, ExtentState: coverage.State}, true
}
