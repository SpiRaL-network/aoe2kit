package triggergraph

import "fmt"

type NeighborhoodOptions struct {
	TriggerIndex *int `json:"trigger_index,omitempty"`
	Depth        int  `json:"depth,omitempty"`
	Limit        int  `json:"limit,omitempty"`
}

type NeighborhoodReport struct {
	Verification string              `json:"verification"`
	GraphSHA256  string              `json:"graph_sha256"`
	TriggerCount int                 `json:"trigger_count"`
	Query        NeighborhoodOptions `json:"query"`
	Nodes        []NeighborhoodNode  `json:"nodes"`
	Edges        []NeighborhoodEdge  `json:"edges"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type NeighborhoodNode struct {
	Index          int      `json:"index"`
	Name           string   `json:"name,omitempty"`
	Enabled        int      `json:"enabled,omitempty"`
	Looping        int      `json:"looping,omitempty"`
	EffectTypes    []string `json:"effect_types,omitempty"`
	ConditionTypes []string `json:"condition_types,omitempty"`
	Distance       int      `json:"distance"`
}

type NeighborhoodEdge struct {
	From           int    `json:"from"`
	To             int    `json:"to"`
	Kind           string `json:"kind"`
	EffectIndex    *int   `json:"effect_index,omitempty"`
	ConditionIndex *int   `json:"condition_index,omitempty"`
}

func Neighborhood(graph *Graph, opts NeighborhoodOptions) (*NeighborhoodReport, error) {
	if graph == nil {
		return nil, fmt.Errorf("trigger graph is nil")
	}
	if opts.TriggerIndex == nil {
		return nil, fmt.Errorf("--trigger is required")
	}
	if *opts.TriggerIndex < 0 || *opts.TriggerIndex >= len(graph.Triggers) {
		return nil, fmt.Errorf("trigger index %d out of range 0..%d", *opts.TriggerIndex, len(graph.Triggers)-1)
	}
	depth := opts.Depth
	if depth <= 0 {
		depth = 1
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 200
	}
	edges := graphControlEdges(graph)
	distance := map[int]int{*opts.TriggerIndex: 0}
	queue := []int{*opts.TriggerIndex}
	for len(queue) > 0 && len(distance) < limit {
		current := queue[0]
		queue = queue[1:]
		if distance[current] >= depth {
			continue
		}
		for _, edge := range edges {
			if edge.From != current && edge.To != current {
				continue
			}
			next := edge.To
			if edge.To == current {
				next = edge.From
			}
			if next < 0 || next >= len(graph.Triggers) {
				continue
			}
			if _, seen := distance[next]; seen {
				continue
			}
			distance[next] = distance[current] + 1
			queue = append(queue, next)
			if len(distance) >= limit {
				break
			}
		}
	}
	report := &NeighborhoodReport{
		Verification: "structure_verified_embedded_trigger_graph_not_engine_runtime_verified",
		GraphSHA256:  graph.SHA256,
		TriggerCount: graph.TriggerCount,
		Query:        opts,
		Warnings:     graph.Warnings,
	}
	for index := range distance {
		report.Nodes = append(report.Nodes, triggerNode(index, graph.Triggers[index], distance[index]))
	}
	sortNeighborhoodNodes(report.Nodes)
	for _, edge := range edges {
		_, fromOK := distance[edge.From]
		_, toOK := distance[edge.To]
		if fromOK && toOK {
			report.Edges = append(report.Edges, edge)
		}
	}
	return report, nil
}

func graphControlEdges(graph *Graph) []NeighborhoodEdge {
	var edges []NeighborhoodEdge
	for triggerIndex, trigger := range graph.Triggers {
		for effectIndex, effect := range listOfMaps(trigger["effects"]) {
			effectType := intValue(effect["type"])
			if effectType != 8 && effectType != 9 {
				continue
			}
			target, ok := knownIntField(effect, "trigger_id")
			if !ok {
				continue
			}
			if target < 0 || target >= len(graph.Triggers) {
				continue
			}
			i := effectIndex
			kind := "activate_trigger"
			if effectType == 9 {
				kind = "deactivate_trigger"
			}
			edges = append(edges, NeighborhoodEdge{From: triggerIndex, To: target, Kind: kind, EffectIndex: &i})
		}
		for conditionIndex, condition := range listOfMaps(trigger["conditions"]) {
			conditionType := intValue(condition["type"])
			if conditionType != 12 {
				continue
			}
			target, ok := knownIntField(condition, "trigger_id")
			if !ok {
				continue
			}
			if target < 0 || target >= len(graph.Triggers) {
				continue
			}
			i := conditionIndex
			edges = append(edges, NeighborhoodEdge{From: target, To: triggerIndex, Kind: "ai_signal_trigger_id_condition", ConditionIndex: &i})
		}
	}
	return edges
}

func knownIntField(row map[string]any, field string) (int, bool) {
	known := knownFieldMap(row)
	if known == nil {
		return 0, false
	}
	_, ok := known[field]
	if !ok {
		return 0, false
	}
	return intValue(known[field]), true
}

func triggerNode(index int, trigger map[string]any, distance int) NeighborhoodNode {
	node := NeighborhoodNode{
		Index:    index,
		Name:     stringValue(trigger["name"]),
		Enabled:  intValue(trigger["enabled"]),
		Looping:  intValue(trigger["looping"]),
		Distance: distance,
	}
	seenEffects := map[string]bool{}
	for _, effect := range listOfMaps(trigger["effects"]) {
		name := stringValue(effect["type_name"])
		if name == "" {
			name = fmt.Sprintf("effect_%d", intValue(effect["type"]))
		}
		if !seenEffects[name] {
			node.EffectTypes = append(node.EffectTypes, name)
			seenEffects[name] = true
		}
	}
	seenConditions := map[string]bool{}
	for _, condition := range listOfMaps(trigger["conditions"]) {
		name := stringValue(condition["type_name"])
		if name == "" {
			name = fmt.Sprintf("condition_%d", intValue(condition["type"]))
		}
		if !seenConditions[name] {
			node.ConditionTypes = append(node.ConditionTypes, name)
			seenConditions[name] = true
		}
	}
	sortStrings(node.EffectTypes)
	sortStrings(node.ConditionTypes)
	return node
}

func sortNeighborhoodNodes(nodes []NeighborhoodNode) {
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[j].Distance < nodes[i].Distance || (nodes[j].Distance == nodes[i].Distance && nodes[j].Index < nodes[i].Index) {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
