package geotrace

import (
	"aoe2kit/pkg/scenario"
	"math"
)

type RegionDiagnostics struct {
	Name        string `json:"name"`
	RoadCells   int    `json:"road_cells"`
	BorderCells int    `json:"border_cells"`
}

type AxisDiagnostics struct {
	Name              string  `json:"name"`
	Visible           bool    `json:"visible"`
	AngleFromSouthDeg float64 `json:"angle_from_south_deg"`
}

func boundsContain(outer BBox, inner *BBox) bool {
	return inner == nil || (bboxContains(outer, inner.MinLat, inner.MinLon) && bboxContains(outer, inner.MaxLat, inner.MaxLon))
}

func applyProfileOverlays(report *Report) {
	proj := report.projection()
	for _, overlay := range report.EcologyProfile.Overlays {
		if !boundsContain(report.BBox, overlay.RequireBounds) {
			continue
		}
		for i := 1; i < len(overlay.Points); i++ {
			a, b := overlay.Points[i-1], overlay.Points[i]
			for _, idx := range proj.geoSegmentIndices(report.GridSize, point{a.Lon, a.Lat}, point{b.Lon, b.Lat}) {
				switch overlay.Kind {
				case "inferred_corridor":
					report.Cells[idx].Road = true
					report.Cells[idx].Causeway = true
				case "admin_line":
					markBorderCell(report, idx, overlay.Radius)
				}
			}
		}
	}
}

func landmarkCells(report Report, landmark LandmarkCluster) []Cell {
	if !boundsContain(report.BBox, landmark.RequireBounds) {
		return nil
	}
	var out []Cell
	for _, cell := range report.Cells {
		if !cellBuildableForGaiaPlacement(cell) || cell.Land == nil || !*cell.Land || cell.Road || cell.Roadside || cell.Border {
			continue
		}
		if !bboxContains(landmark.Bounds, cell.Lat, cell.Lon) {
			continue
		}
		matches := len(landmark.LandcoverKeys) == 0 && len(landmark.TerrainIDs) == 0
		for _, key := range landmark.LandcoverKeys {
			matches = matches || key == cell.LandcoverKey
		}
		for _, id := range landmark.TerrainIDs {
			matches = matches || id == cell.TerrainID
		}
		if matches {
			out = append(out, cell)
		}
	}
	return out
}

func measureProfileRegions(report *Report) {
	report.RegionDiagnostics = nil
	for _, region := range report.EcologyProfile.DiagnosticRegions {
		d := RegionDiagnostics{Name: region.Name}
		for _, cell := range report.Cells {
			if !bboxContains(region.Bounds, cell.Lat, cell.Lon) {
				continue
			}
			if cell.Road {
				d.RoadCells++
			}
			if cell.Border {
				d.BorderCells++
			}
		}
		report.RegionDiagnostics = append(report.RegionDiagnostics, d)
	}
}

func measureCalibrationAxes(report *Report) {
	report.CalibrationAxes = nil
	p := report.projection()
	for _, axis := range report.EcologyProfile.CalibrationAxes {
		sx0, sy0, ok0 := screenPointForLatLon(p, axis.From.Lat, axis.From.Lon)
		sx1, sy1, ok1 := screenPointForLatLon(p, axis.To.Lat, axis.To.Lon)
		d := AxisDiagnostics{Name: axis.Name, Visible: ok0 && ok1}
		if d.Visible {
			d.AngleFromSouthDeg = math.Atan2(sx1-sx0, sy1-sy0) * 180 / math.Pi
		}
		report.CalibrationAxes = append(report.CalibrationAxes, d)
	}
}

func buildLandmarks(report Report, profile EcologyProfile) []scenario.UnitRecipe {
	var out []scenario.UnitRecipe
	for _, l := range profile.Landmarks {
		cells := landmarkCells(report, l)
		if len(cells) == 0 || l.CellsPerUnit <= 0 {
			continue
		}
		count := densityScaledCount(minInt(l.MaxCount, maxInt(l.MinCount, int(float64(len(cells))/l.CellsPerUnit))), l.Density)
		exact := false
		rotationChoices := visualVariantRotationChoices(l.UnitConst, profile.Biome)
		if len(rotationChoices) == 0 {
			rotationChoices = rotationAngles(16)
		}
		out = append(out, scenario.UnitRecipe{Op: "cluster_scatter", Player: 0, UnitConst: l.UnitConst, Count: intPtr(count),
			Centers: representativeCenters(cells, minInt(l.CenterLimit, maxInt(4, count/90))), Snap: "auto", Distribution: "dispersed", Jitter: float64Ptr(0.25),
			Spread: float64Ptr(l.Spread), MinDistance: float64Ptr(l.MinDistance), AllowedTerrainIDs: pruneUnbuildableTerrainIDs(l.AllowedTerrainIDs), Exact: &exact,
			Seed: intPtr(layerSeed(l.Name, report.Seed)), RotationChoices: rotationChoices,
			TargetAreaX1: float64Ptr(0), TargetAreaY1: float64Ptr(0), TargetAreaX2: float64Ptr(float64(report.GridSize)), TargetAreaY2: float64Ptr(float64(report.GridSize))})
	}
	return out
}
