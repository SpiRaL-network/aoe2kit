package geotrace

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// Observed coverage is not a claim that a source contains every real feature.
type SourceCoverage struct {
	State                 string `json:"state"`
	EligibleCells         int    `json:"eligible_cells"`
	CoveredCells          int    `json:"covered_cells"`
	MissingCells          int    `json:"missing_cells"`
	OutsideNominalCells   int    `json:"outside_nominal_cells"`
	OutsideNominalCovered int    `json:"outside_nominal_covered"`
	QueryBounds           *BBox  `json:"query_bounds,omitempty"`
	GeometryBounds        *BBox  `json:"geometry_bounds,omitempty"`
	Note                  string `json:"note,omitempty"`
}

func projectionEnvelope(p geoProjection) BBox {
	if p.orient == "diamond" {
		return p.b
	}
	var corners []Cell
	for _, u := range []float64{-0.5, 0.5} {
		for _, v := range []float64{-0.5, 0.5} {
			lat, lon, _ := p.gridToLatLon(u, v)
			corners = append(corners, Cell{Lat: lat, Lon: lon})
		}
	}
	return bboxFromCells(corners)
}

func measureSourceCoverage(r *Report) {
	if r.SourceCoverage == nil {
		r.SourceCoverage = map[string]SourceCoverage{}
	}
	for _, name := range []string{"landmask", "landcover", "usgs_woody"} {
		d := SourceCoverage{State: "sampled"}
		for _, c := range r.Cells {
			if name != "landmask" && (c.Land == nil || !*c.Land) {
				continue
			}
			d.EligibleCells++
			if c.OutsideTrace {
				d.OutsideNominalCells++
			}
			covered := c.Land != nil
			if name == "landcover" {
				covered = c.LandcoverKey != ""
			}
			if name == "usgs_woody" {
				covered = len(c.USGSWoodySpecies) > 0
			}
			if covered {
				d.CoveredCells++
				if c.OutsideTrace {
					d.OutsideNominalCovered++
				}
			}
		}
		d.MissingCells = d.EligibleCells - d.CoveredCells
		if (name == "landcover" && r.LandcoverProvider == "") || (name == "usgs_woody" && r.USGSWoodyProvider == "") || (name == "landmask" && r.LandmaskProvider == "") {
			d.State = "unavailable"
		} else if d.MissingCells > 0 {
			d.State = "partial"
		}
		if name == "usgs_woody" {
			d.Note = "Cells with mapped species in a nearest atlas point within 50 km; presence is not abundance."
		}
		if name == "landmask" {
			d.Note = "Every evaluated polygon point is known land/water; a source file's geographic completeness is not inferred from absence of polygons."
		}
		r.SourceCoverage[name] = d
		if d.MissingCells > 0 && d.State != "unavailable" {
			r.Warnings = append(r.Warnings, fmt.Sprintf("%s missing evidence for %d/%d eligible cells; outside requested bounds covered %d/%d", name, d.MissingCells, d.EligibleCells, d.OutsideNominalCovered, d.OutsideNominalCells))
		}
	}
	if d, ok := r.SourceCoverage["roads"]; ok {
		if d.QueryBounds == nil {
			r.Warnings = append(r.Warnings, "road source query extent is unknown; observed geometry bounds do not prove coverage of the projected map")
		} else if !boundsContain(*d.QueryBounds, &r.OverlayBBox) {
			r.Warnings = append(r.Warnings, "road source query bounds do not contain the projected map envelope")
		}
	}
}

func roadSourceCoverage(ways []roadWay, opts Options) SourceCoverage {
	d := SourceCoverage{State: "query_extent_known", QueryBounds: &opts.BBox, Note: "Requested extent, not proof that every real road is present."}
	b := BBox{MinLat: math.Inf(1), MinLon: math.Inf(1), MaxLat: math.Inf(-1), MaxLon: math.Inf(-1)}
	count := 0
	for _, w := range ways {
		for _, p := range w.Geometry {
			b.MinLat = math.Min(b.MinLat, p.Lat)
			b.MaxLat = math.Max(b.MaxLat, p.Lat)
			b.MinLon = math.Min(b.MinLon, p.Lon)
			b.MaxLon = math.Max(b.MaxLon, p.Lon)
			count++
		}
	}
	if count > 0 {
		d.GeometryBounds = &b
	}
	if opts.RoadsPath != "" {
		d.State = "query_extent_unknown"
		d.QueryBounds = nil
		d.Note = "Local geometry extent is observed; original query extent is unknown unless declared in kit_query_bbox."
		var metadata struct {
			BBox *BBox `json:"kit_query_bbox"`
		}
		if raw, err := os.ReadFile(opts.RoadsPath); err == nil && json.Unmarshal(raw, &metadata) == nil && metadata.BBox != nil {
			b := metadata.BBox
			if finite(b.MinLat) && finite(b.MaxLat) && finite(b.MinLon) && finite(b.MaxLon) && b.MinLat >= -90 && b.MaxLat <= 90 && b.MinLon >= -180 && b.MaxLon <= 180 && b.MinLat < b.MaxLat && b.MinLon < b.MaxLon {
				d.QueryBounds = b
				d.State = "query_extent_declared"
			}
		}
	}
	return d
}
