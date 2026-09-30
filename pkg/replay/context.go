package replay

import (
	"encoding/json"
	"fmt"
	"os"
)

type Context struct {
	Name              string          `json:"name,omitempty"`
	Regions           []ContextRegion `json:"regions,omitempty"`
	TelemetryPrefixes []string        `json:"telemetry_prefixes,omitempty"`
}

type ContextRegion struct {
	Name string    `json:"name"`
	Box  []float64 `json:"box,omitempty"`
}

func LoadContext(path string) (*Context, error) {
	if path == "" {
		return &Context{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var context Context
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, fmt.Errorf("%s is not valid replay context JSON: %w", path, err)
	}
	return &context, nil
}

func (c *Context) MatchRegion(x, y float64) string {
	if c == nil {
		return ""
	}
	for _, region := range c.Regions {
		if len(region.Box) != 4 || region.Name == "" {
			continue
		}
		minX, minY, maxX, maxY := region.Box[0], region.Box[1], region.Box[2], region.Box[3]
		if minX > maxX {
			minX, maxX = maxX, minX
		}
		if minY > maxY {
			minY, maxY = maxY, minY
		}
		if x >= minX && x <= maxX && y >= minY && y <= maxY {
			return region.Name
		}
	}
	return ""
}

func AnnotateRegions(events []ReplayEvent, context *Context) {
	if context == nil {
		return
	}
	for i := range events {
		if events[i].Region != "" {
			continue
		}
		if events[i].X == 0 && events[i].Y == 0 {
			continue
		}
		events[i].Region = context.MatchRegion(events[i].X, events[i].Y)
	}
}
