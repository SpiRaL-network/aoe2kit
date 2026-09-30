package scenario

import (
	"fmt"
	"math"
	"sort"
)

type RegionGuessReport struct {
	Path     string        `json:"path,omitempty"`
	Regions  []RegionGuess `json:"regions"`
	Warnings []string      `json:"warnings,omitempty"`
}

type RegionGuess struct {
	Name       string    `json:"name"`
	Box        []float64 `json:"box"`
	Method     string    `json:"method"`
	Confidence string    `json:"confidence"`
	UnitCount  int       `json:"unit_count,omitempty"`
	Player     int       `json:"player,omitempty"`
}

func GuessRegionsFile(path string) (RegionGuessReport, error) {
	file, err := Open(path)
	if err != nil {
		return RegionGuessReport{}, err
	}
	report := file.GuessRegions()
	report.Path = path
	return report, nil
}

func (f *File) GuessRegions() RegionGuessReport {
	report := RegionGuessReport{Path: f.Path}
	if f.Map != nil {
		report.Regions = append(report.Regions, RegionGuess{
			Name:       "Whole Map",
			Box:        []float64{0, 0, float64(f.Map.Width), float64(f.Map.Height)},
			Method:     "map_bounds",
			Confidence: "parsed",
		})
	}
	if f.Units == nil {
		report.Warnings = append(report.Warnings, "unit section unavailable; only whole-map region can be emitted")
		return report
	}
	for _, section := range f.Units.Sections {
		if len(section.Units) == 0 {
			continue
		}
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
		for _, unit := range section.Units {
			if unit.X < minX {
				minX = unit.X
			}
			if unit.X > maxX {
				maxX = unit.X
			}
			if unit.Y < minY {
				minY = unit.Y
			}
			if unit.Y > maxY {
				maxY = unit.Y
			}
		}
		if minX == math.MaxFloat64 {
			continue
		}
		margin := 3.0
		report.Regions = append(report.Regions, RegionGuess{
			Name:       fmt.Sprintf("P%d Unit Cluster", section.Player),
			Box:        []float64{math.Max(0, minX-margin), math.Max(0, minY-margin), maxX + margin, maxY + margin},
			Method:     "per_player_unit_bounds",
			Confidence: "heuristic",
			UnitCount:  len(section.Units),
			Player:     section.Player,
		})
	}
	sort.Slice(report.Regions, func(i, j int) bool {
		if report.Regions[i].Method != report.Regions[j].Method {
			return report.Regions[i].Method < report.Regions[j].Method
		}
		return report.Regions[i].Name < report.Regions[j].Name
	})
	return report
}
