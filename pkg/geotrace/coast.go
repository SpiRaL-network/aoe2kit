package geotrace

// These are conservative visualization policy thresholds, not source accuracy claims.
const coastRefinementMaxTileMeters = 250.0
const coastPreservationMaxTileMeters = 1500.0

func knownWorldCoverClass(class int) bool {
	switch class {
	case 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 100:
		return true
	}
	return false
}

func worldCoverInlandWater(sourceLand bool, sample LandcoverSample) bool {
	return sourceLand && sample.Valid && sample.Class == 80 && sample.CoastalLandFraction == 0
}

type CoastRefinement struct {
	Policy         string  `json:"policy"`
	Enabled        bool    `json:"enabled"`
	MaxTileMeters  float64 `json:"max_tile_meters"`
	MaxPixelMeters float64 `json:"max_pixel_meters"`
	Eligible       int     `json:"eligible_cells"`
	Resolved       int     `json:"resolved_cells"`
	Fallback       int     `json:"fallback_cells"`
	WaterToLand    int     `json:"water_to_land"`
	LandToWater    int     `json:"land_to_water"`
	Mode           string  `json:"mode,omitempty"`
	MinLandPixels  int     `json:"min_land_pixels,omitempty"`
}

func refineWorldCoverCoast(provider string, tileMeters float64) bool {
	return provider == "worldcover" && tileMeters > 0 && tileMeters <= coastPreservationMaxTileMeters
}

func refineCoastMask(mask []bool, samples []LandcoverSample, provider string, tileMeters float64) CoastRefinement {
	d := CoastRefinement{Policy: "WorldCover at tiles <=250m and selected pixels <=tile/3 refines both directions before shore distance; unrecognized/nodata/coarser samples retain source mask. Explicit inland-water polygons still override.", Enabled: refineWorldCoverCoast(provider, tileMeters), MaxTileMeters: coastRefinementMaxTileMeters, MaxPixelMeters: tileMeters / 3}
	d.Mode = "majority"
	coarse := tileMeters > coastRefinementMaxTileMeters
	if coarse {
		d.Mode = "thin_land_preservation"
		d.MaxTileMeters = coastPreservationMaxTileMeters
		d.MinLandPixels = 1
		d.MaxPixelMeters = tileMeters / 12
		d.Policy = "At tiles >250m and <=1500m, retain source land and promote water if any recognized WorldCover land pixel occurs in the approximately tile-sized sampling footprint (pixels <=tile/12). Use a land-only representative class at mixed coastal cells. This is cartographic exaggeration, not exact rotated-cell area coverage; explicit inland-water polygons override."
	}
	if !d.Enabled {
		return d
	}
	for i, land := range mask {
		d.Eligible++
		if i >= len(samples) {
			d.Fallback++
			continue
		}
		s := samples[i]
		known := knownWorldCoverClass(s.Class)
		if !s.Valid || !known || !finite(s.PixelMeters) || s.PixelMeters <= 0 || s.PixelMeters > d.MaxPixelMeters {
			d.Fallback++
			continue
		}
		d.Resolved++
		next := s.Class != 80
		if coarse {
			fraction := 0.0
			for _, stat := range s.Histogram {
				if stat.Class != 80 && knownWorldCoverClass(stat.Class) {
					fraction += stat.Weight
				}
			}
			next = land || fraction > 0
			if next && fraction > 0 {
				samples[i].CoastalLandFraction = fraction
			}
		}
		if next && !land {
			d.WaterToLand++
		}
		if !next && land {
			d.LandToWater++
		}
		mask[i] = next
	}
	return d
}
