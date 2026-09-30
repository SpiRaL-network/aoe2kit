package scenario

import (
	"encoding/hex"
	"errors"
	"math"
	"sort"
)

type TerrainOptions struct {
	ID           *int
	IncludeTiles bool
}

type TerrainReport struct {
	Path         string             `json:"path,omitempty"`
	Width        int                `json:"width"`
	Height       int                `json:"height"`
	TileCount    int                `json:"tile_count"`
	Returned     int                `json:"returned"`
	Filter       *TerrainFilter     `json:"filter,omitempty"`
	Aggregates   []TerrainAggregate `json:"aggregates"`
	Tiles        []TerrainTile      `json:"tiles,omitempty"`
	Verification string             `json:"verification"`
}

type TerrainFilter struct {
	ID int `json:"id"`
}

type TerrainTile struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	TerrainID int    `json:"terrain_id"`
	Elevation int    `json:"elevation"`
	Layer     int    `json:"layer"`
	UnusedHex string `json:"unused_hex,omitempty"`
}

type TerrainAggregate struct {
	TerrainID        int                `json:"terrain_id"`
	Count            int                `json:"count"`
	Bounds           TerrainBounds      `json:"bounds"`
	Centroid         TerrainPoint       `json:"centroid"`
	Components       int                `json:"components"`
	LargestComponent int                `json:"largest_component"`
	ComponentBounds  []TerrainComponent `json:"component_bounds,omitempty"`
}

type TerrainBounds struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

type TerrainPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TerrainComponent struct {
	Index    int           `json:"index"`
	Count    int           `json:"count"`
	Bounds   TerrainBounds `json:"bounds"`
	Centroid TerrainPoint  `json:"centroid"`
}

func TerrainFile(path string, opts TerrainOptions) (TerrainReport, error) {
	file, err := openMapReadable(path)
	if err != nil {
		return TerrainReport{}, err
	}
	report, err := file.Terrain(opts)
	if err != nil {
		return TerrainReport{}, err
	}
	report.Path = path
	return report, nil
}

func (f *File) Terrain(opts TerrainOptions) (TerrainReport, error) {
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return TerrainReport{}, err
	}
	if len(tiles) != width*height {
		return TerrainReport{}, errors.New("terrain tile count does not match map dimensions")
	}
	report := TerrainReport{
		Path:         f.Path,
		Width:        width,
		Height:       height,
		TileCount:    len(tiles),
		Verification: "structure_verified_not_engine_verified",
	}
	if opts.ID != nil {
		report.Filter = &TerrainFilter{ID: *opts.ID}
	}
	byID := map[int][]TerrainTile{}
	for i, node := range tiles {
		x := i % width
		y := i / width
		terrainID, _ := node.intValue("terrain_id")
		elevation, _ := node.intValue("elevation")
		layer, _ := node.intValue("layer")
		unusedHex := ""
		if field := node.field("unused"); field != nil && len(field.Raw) > 0 {
			unusedHex = hex.EncodeToString(field.Raw)
		}
		tile := TerrainTile{
			X:         x,
			Y:         y,
			TerrainID: terrainID,
			Elevation: elevation,
			Layer:     layer,
			UnusedHex: unusedHex,
		}
		byID[terrainID] = append(byID[terrainID], tile)
		if opts.IncludeTiles || opts.ID != nil && terrainID == *opts.ID {
			report.Tiles = append(report.Tiles, tile)
		}
	}
	report.Returned = len(report.Tiles)
	ids := make([]int, 0, len(byID))
	if opts.ID != nil {
		if _, ok := byID[*opts.ID]; ok {
			ids = append(ids, *opts.ID)
		}
	} else {
		for id := range byID {
			ids = append(ids, id)
		}
		sort.Ints(ids)
	}
	for _, id := range ids {
		report.Aggregates = append(report.Aggregates, aggregateTerrain(id, byID[id], width, height, opts.ID != nil))
	}
	return report, nil
}

func aggregateTerrain(id int, tiles []TerrainTile, width, height int, includeComponents bool) TerrainAggregate {
	agg := TerrainAggregate{
		TerrainID: id,
		Count:     len(tiles),
		Bounds: TerrainBounds{
			MinX: math.MaxInt,
			MinY: math.MaxInt,
			MaxX: -1,
			MaxY: -1,
		},
	}
	var sumX, sumY int
	present := make([]bool, width*height)
	for _, tile := range tiles {
		sumX += tile.X
		sumY += tile.Y
		if tile.X < agg.Bounds.MinX {
			agg.Bounds.MinX = tile.X
		}
		if tile.Y < agg.Bounds.MinY {
			agg.Bounds.MinY = tile.Y
		}
		if tile.X > agg.Bounds.MaxX {
			agg.Bounds.MaxX = tile.X
		}
		if tile.Y > agg.Bounds.MaxY {
			agg.Bounds.MaxY = tile.Y
		}
		present[tile.Y*width+tile.X] = true
	}
	if len(tiles) > 0 {
		agg.Centroid = TerrainPoint{X: float64(sumX) / float64(len(tiles)), Y: float64(sumY) / float64(len(tiles))}
	}
	components := terrainComponents(present, width, height)
	agg.Components = len(components)
	for _, component := range components {
		if component.Count > agg.LargestComponent {
			agg.LargestComponent = component.Count
		}
	}
	if includeComponents {
		agg.ComponentBounds = components
	}
	return agg
}

func terrainComponents(present []bool, width, height int) []TerrainComponent {
	visited := make([]bool, len(present))
	var components []TerrainComponent
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !present[idx] || visited[idx] {
				continue
			}
			component := floodTerrainComponent(present, visited, width, height, x, y, len(components))
			components = append(components, component)
		}
	}
	sort.Slice(components, func(i, j int) bool {
		if components[i].Count != components[j].Count {
			return components[i].Count > components[j].Count
		}
		if components[i].Bounds.MinY != components[j].Bounds.MinY {
			return components[i].Bounds.MinY < components[j].Bounds.MinY
		}
		return components[i].Bounds.MinX < components[j].Bounds.MinX
	})
	for i := range components {
		components[i].Index = i
	}
	return components
}

func floodTerrainComponent(present, visited []bool, width, height, startX, startY, index int) TerrainComponent {
	queue := []mapPoint{{X: startX, Y: startY}}
	visited[startY*width+startX] = true
	component := TerrainComponent{
		Index: index,
		Bounds: TerrainBounds{
			MinX: startX,
			MinY: startY,
			MaxX: startX,
			MaxY: startY,
		},
	}
	var sumX, sumY int
	for len(queue) > 0 {
		point := queue[0]
		queue = queue[1:]
		component.Count++
		sumX += point.X
		sumY += point.Y
		if point.X < component.Bounds.MinX {
			component.Bounds.MinX = point.X
		}
		if point.Y < component.Bounds.MinY {
			component.Bounds.MinY = point.Y
		}
		if point.X > component.Bounds.MaxX {
			component.Bounds.MaxX = point.X
		}
		if point.Y > component.Bounds.MaxY {
			component.Bounds.MaxY = point.Y
		}
		for _, next := range []mapPoint{
			{X: point.X - 1, Y: point.Y},
			{X: point.X + 1, Y: point.Y},
			{X: point.X, Y: point.Y - 1},
			{X: point.X, Y: point.Y + 1},
		} {
			if next.X < 0 || next.Y < 0 || next.X >= width || next.Y >= height {
				continue
			}
			idx := next.Y*width + next.X
			if present[idx] && !visited[idx] {
				visited[idx] = true
				queue = append(queue, next)
			}
		}
	}
	component.Centroid = TerrainPoint{X: float64(sumX) / float64(component.Count), Y: float64(sumY) / float64(component.Count)}
	return component
}
