package datcodec

import (
	"sort"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

type EffectCommandMatrixReport struct {
	Version             string                     `json:"version"`
	Method              string                     `json:"method"`
	Verification        aoe2.VerificationClaim     `json:"verification"`
	Filters             EffectCommandMatrixFilters `json:"filters,omitempty"`
	EffectCount         int                        `json:"effect_count"`
	MatchingEffectCount int                        `json:"matching_effect_count,omitempty"`
	CommandCount        int                        `json:"command_count"`
	TypeCount           int                        `json:"type_count"`
	ExampleLimit        int                        `json:"example_limit"`
	CommandTypes        []EffectCommandTypeProfile `json:"command_types"`
}

type EffectCommandMatrixOptions struct {
	ExampleLimit   int
	CommandType    *int
	Operand        *int
	ReferenceKind  string
	ReferenceField string
	UnknownOnly    bool
	TypedOnly      bool
	CandidateOnly  bool
}

type EffectCommandMatrixFilters struct {
	CommandType    *int   `json:"command_type,omitempty"`
	Operand        *int   `json:"operand,omitempty"`
	ReferenceKind  string `json:"reference_kind,omitempty"`
	ReferenceField string `json:"reference_field,omitempty"`
	UnknownOnly    bool   `json:"unknown_only,omitempty"`
	TypedOnly      bool   `json:"typed_only,omitempty"`
	CandidateOnly  bool   `json:"candidate_only,omitempty"`
}

type EffectCommandTypeProfile struct {
	Type             uint8                         `json:"type"`
	TypeName         string                        `json:"type_name"`
	CommandCount     int                           `json:"command_count"`
	EffectCount      int                           `json:"effect_count"`
	Operands         EffectCommandOperandProfiles  `json:"operands"`
	TypedReferences  []EffectCommandReferenceCount `json:"typed_references,omitempty"`
	CandidateRefs    []EffectCommandReferenceCount `json:"candidate_references,omitempty"`
	Attributes       []EffectCommandValueCount     `json:"attributes,omitempty"`
	PackedTypeIDs    []EffectCommandValueCount     `json:"packed_type_ids,omitempty"`
	PackedAmounts    []EffectCommandValueCount     `json:"packed_amounts,omitempty"`
	ResourceIDs      []EffectCommandValueCount     `json:"resource_ids,omitempty"`
	OperationIDs     []EffectCommandValueCount     `json:"operation_ids,omitempty"`
	TechAttributeIDs []EffectCommandValueCount     `json:"tech_attribute_ids,omitempty"`
	Examples         []EffectCommandExample        `json:"examples,omitempty"`
}

type EffectCommandOperandProfiles struct {
	A EffectCommandOperandProfile `json:"a"`
	B EffectCommandOperandProfile `json:"b"`
	C EffectCommandOperandProfile `json:"c"`
	D EffectCommandOperandProfile `json:"d"`
}

type EffectCommandOperandProfile struct {
	Min             float64                   `json:"min"`
	Max             float64                   `json:"max"`
	Negative        int                       `json:"negative"`
	Zero            int                       `json:"zero"`
	Positive        int                       `json:"positive"`
	Integral        int                       `json:"integral"`
	Distinct        int                       `json:"distinct"`
	SampleValues    []EffectCommandValueCount `json:"sample_values,omitempty"`
	CommonValues    []EffectCommandValueCount `json:"common_values,omitempty"`
	LooksLikeIDList bool                      `json:"looks_like_id_list,omitempty"`
}

type EffectCommandValueCount struct {
	Value int    `json:"value"`
	Count int    `json:"count"`
	Name  string `json:"name,omitempty"`
}

type EffectCommandReferenceCount struct {
	Kind      string                    `json:"kind"`
	Field     string                    `json:"field"`
	Count     int                       `json:"count"`
	SampleIDs []EffectCommandValueCount `json:"sample_ids,omitempty"`
}

type EffectCommandExample struct {
	EffectID   int                              `json:"effect_id"`
	EffectName string                           `json:"effect_name"`
	Command    datfile.EffectCommand            `json:"command"`
	References []datfile.EffectCommandReference `json:"references,omitempty"`
}

type effectCommandTypeAccumulator struct {
	profile       EffectCommandTypeProfile
	effects       map[int]bool
	operandA      operandAccumulator
	operandB      operandAccumulator
	operandC      operandAccumulator
	operandD      operandAccumulator
	referenceMaps map[string]*referenceAccumulator
	candidateMaps map[string]*referenceAccumulator
	attributes    map[int]int
	packedTypes   map[int]int
	packedAmounts map[int]int
	resourceIDs   map[int]int
	operationIDs  map[int]int
	techAttrIDs   map[int]int
}

type operandAccumulator struct {
	seen   bool
	min    float64
	max    float64
	counts map[int]int
	out    EffectCommandOperandProfile
}

type referenceAccumulator struct {
	kind   string
	field  string
	count  int
	values map[int]int
}

func EffectCommandMatrix(idx *datfile.Index, exampleLimit int) EffectCommandMatrixReport {
	return EffectCommandMatrixWithOptions(idx, EffectCommandMatrixOptions{ExampleLimit: exampleLimit})
}

func EffectCommandMatrixWithOptions(idx *datfile.Index, opts EffectCommandMatrixOptions) EffectCommandMatrixReport {
	exampleLimit := opts.ExampleLimit
	if exampleLimit < 0 {
		exampleLimit = 0
	}
	byType := map[uint8]*effectCommandTypeAccumulator{}
	totalCommands := 0
	matchedEffects := map[int]bool{}
	for _, effect := range idx.Effects {
		for _, command := range effect.Commands {
			refs := effectCommandSemanticReferences(command)
			candidateRefs := effectCommandCandidateReferences(command)
			if !includeEffectCommandInMatrix(command, refs, candidateRefs, opts) {
				continue
			}
			totalCommands++
			matchedEffects[effect.Index] = true
			acc := byType[command.Type]
			if acc == nil {
				acc = &effectCommandTypeAccumulator{
					profile: EffectCommandTypeProfile{
						Type:     command.Type,
						TypeName: datfile.EffectCommandTypeName(command.Type),
					},
					effects:       map[int]bool{},
					referenceMaps: map[string]*referenceAccumulator{},
					candidateMaps: map[string]*referenceAccumulator{},
					attributes:    map[int]int{},
					packedTypes:   map[int]int{},
					packedAmounts: map[int]int{},
					resourceIDs:   map[int]int{},
					operationIDs:  map[int]int{},
					techAttrIDs:   map[int]int{},
				}
				byType[command.Type] = acc
			}
			acc.profile.CommandCount++
			acc.effects[effect.Index] = true
			acc.operandA.addInt(int(command.A))
			acc.operandB.addInt(int(command.B))
			acc.operandC.addInt(int(command.C))
			acc.operandD.addFloat(float64(command.D))
			if command.C >= 0 {
				acc.attributes[int(command.C)]++
			}
			if command.Semantic != nil {
				if command.Semantic.ResourceID != nil {
					acc.resourceIDs[int(*command.Semantic.ResourceID)]++
				}
				if command.Semantic.PackedTypeID != nil {
					acc.packedTypes[*command.Semantic.PackedTypeID]++
				}
				if command.Semantic.PackedAmount != nil {
					acc.packedAmounts[*command.Semantic.PackedAmount]++
				}
				if command.Semantic.OperationID != nil {
					acc.operationIDs[int(*command.Semantic.OperationID)]++
				}
				if command.Semantic.TechAttributeID != nil {
					acc.techAttrIDs[int(*command.Semantic.TechAttributeID)]++
				}
			}
			for _, ref := range refs {
				key := ref.Kind + "\x00" + ref.Field
				refAcc := acc.referenceMaps[key]
				if refAcc == nil {
					refAcc = &referenceAccumulator{kind: ref.Kind, field: ref.Field, values: map[int]int{}}
					acc.referenceMaps[key] = refAcc
				}
				refAcc.count++
				refAcc.values[ref.ID]++
			}
			for _, ref := range candidateRefs {
				key := ref.Kind + "\x00" + ref.Field
				refAcc := acc.candidateMaps[key]
				if refAcc == nil {
					refAcc = &referenceAccumulator{kind: ref.Kind, field: ref.Field, values: map[int]int{}}
					acc.candidateMaps[key] = refAcc
				}
				refAcc.count++
				refAcc.values[ref.ID]++
			}
			if len(acc.profile.Examples) < exampleLimit {
				acc.profile.Examples = append(acc.profile.Examples, EffectCommandExample{
					EffectID:   effect.Index,
					EffectName: effect.Name,
					Command:    command,
					References: refs,
				})
			}
		}
	}
	types := make([]int, 0, len(byType))
	for typ := range byType {
		types = append(types, int(typ))
	}
	sort.Ints(types)
	commandTypes := make([]EffectCommandTypeProfile, 0, len(types))
	for _, typ := range types {
		acc := byType[uint8(typ)]
		acc.profile.EffectCount = len(acc.effects)
		acc.profile.Operands = EffectCommandOperandProfiles{
			A: acc.operandA.profile(),
			B: acc.operandB.profile(),
			C: acc.operandC.profile(),
			D: acc.operandD.profile(),
		}
		acc.profile.TypedReferences = referenceCounts(acc.referenceMaps, idx)
		acc.profile.CandidateRefs = referenceCounts(acc.candidateMaps, idx)
		acc.profile.Attributes = attributeCounts(acc.attributes)
		acc.profile.PackedTypeIDs = valueCounts(acc.packedTypes, 16, true)
		acc.profile.PackedAmounts = valueCounts(acc.packedAmounts, 16, true)
		acc.profile.ResourceIDs = resourceCounts(acc.resourceIDs)
		acc.profile.OperationIDs = operationCounts(acc.operationIDs)
		acc.profile.TechAttributeIDs = techAttributeCounts(acc.techAttrIDs)
		commandTypes = append(commandTypes, acc.profile)
	}
	claim := aoe2.StructureVerification(true)
	claim.Note = "effect-command-matrix is a structural DAT census over decoded effect command rows; operand shape is evidence for investigation, not proof of in-engine behavior."
	return EffectCommandMatrixReport{
		Version:      Version,
		Method:       "group decoded DAT effect commands by command type, operand distribution, typed references, and examples",
		Verification: claim,
		Filters: EffectCommandMatrixFilters{
			CommandType:    opts.CommandType,
			Operand:        opts.Operand,
			ReferenceKind:  opts.ReferenceKind,
			ReferenceField: opts.ReferenceField,
			UnknownOnly:    opts.UnknownOnly,
			TypedOnly:      opts.TypedOnly,
			CandidateOnly:  opts.CandidateOnly,
		},
		EffectCount:         len(idx.Effects),
		MatchingEffectCount: len(matchedEffects),
		CommandCount:        totalCommands,
		TypeCount:           len(commandTypes),
		ExampleLimit:        exampleLimit,
		CommandTypes:        commandTypes,
	}
}

func effectCommandCandidateReferences(command datfile.EffectCommand) []datfile.EffectCommandReference {
	if len(effectCommandSemanticReferences(command)) > 0 {
		return nil
	}
	if !isCandidateUnitEffectCommandA(command.Type) || command.A < 0 {
		return nil
	}
	return []datfile.EffectCommandReference{{
		Kind:       "unit",
		Field:      "a_candidate_unit_id",
		ID:         int(command.A),
		Confidence: "candidate_effect_command_operand",
	}}
}

func includeEffectCommandInMatrix(command datfile.EffectCommand, refs []datfile.EffectCommandReference, candidateRefs []datfile.EffectCommandReference, opts EffectCommandMatrixOptions) bool {
	if opts.CommandType != nil && int(command.Type) != *opts.CommandType {
		return false
	}
	if opts.Operand != nil && !effectCommandMatrixOperandEquals(command, *opts.Operand) {
		return false
	}
	typeName := datfile.EffectCommandTypeName(command.Type)
	if opts.UnknownOnly && typeName != "unknown" {
		return false
	}
	if opts.TypedOnly && len(refs) == 0 {
		return false
	}
	if opts.CandidateOnly && len(candidateRefs) == 0 {
		return false
	}
	if opts.ReferenceKind == "" && opts.ReferenceField == "" {
		return true
	}
	filterRefs := refs
	if opts.CandidateOnly {
		filterRefs = candidateRefs
	}
	for _, ref := range filterRefs {
		if opts.ReferenceKind != "" && ref.Kind != opts.ReferenceKind {
			continue
		}
		if opts.ReferenceField != "" && ref.Field != opts.ReferenceField {
			continue
		}
		return true
	}
	return false
}

func effectCommandMatrixOperandEquals(command datfile.EffectCommand, value int) bool {
	if int(command.A) == value || int(command.B) == value || int(command.C) == value {
		return true
	}
	d := int(command.D)
	return command.D == float32(d) && d == value
}

func (acc *operandAccumulator) addInt(value int) {
	acc.addFloat(float64(value))
}

func (acc *operandAccumulator) addFloat(value float64) {
	if !acc.seen {
		acc.seen = true
		acc.min = value
		acc.max = value
		acc.counts = map[int]int{}
	}
	if value < acc.min {
		acc.min = value
	}
	if value > acc.max {
		acc.max = value
	}
	switch {
	case value < 0:
		acc.out.Negative++
	case value == 0:
		acc.out.Zero++
	default:
		acc.out.Positive++
	}
	intValue := int(value)
	if value == float64(intValue) {
		acc.out.Integral++
		acc.counts[intValue]++
	}
}

func (acc operandAccumulator) profile() EffectCommandOperandProfile {
	out := acc.out
	out.Min = acc.min
	out.Max = acc.max
	out.Distinct = len(acc.counts)
	out.SampleValues = valueCounts(acc.counts, 8, false)
	out.CommonValues = valueCounts(acc.counts, 8, true)
	out.LooksLikeIDList = out.Integral == out.Negative+out.Zero+out.Positive && out.Distinct > 8
	return out
}

func referenceCounts(in map[string]*referenceAccumulator, idx *datfile.Index) []EffectCommandReferenceCount {
	out := make([]EffectCommandReferenceCount, 0, len(in))
	for _, ref := range in {
		samples := valueCounts(ref.values, 8, false)
		for i := range samples {
			samples[i].Name = effectCommandReferenceName(idx, ref.kind, samples[i].Value)
		}
		out = append(out, EffectCommandReferenceCount{
			Kind:      ref.kind,
			Field:     ref.field,
			Count:     ref.count,
			SampleIDs: samples,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Field < out[j].Field
	})
	return out
}

func effectCommandReferenceName(idx *datfile.Index, kind string, id int) string {
	switch kind {
	case "tech":
		tech, ok := idx.Tech(id)
		if ok {
			return tech.Name
		}
	case "unit":
		for _, civ := range idx.Civs {
			if id < 0 || id >= len(civ.Units) {
				continue
			}
			unit := civ.Units[id]
			if unit.Present && unit.Name != "" {
				return unit.Name
			}
		}
	}
	return ""
}

func attributeCounts(counts map[int]int) []EffectCommandValueCount {
	out := valueCounts(counts, 16, true)
	for i := range out {
		out[i].Name = datfile.EffectAttributeName(int16(out[i].Value))
	}
	return out
}

func resourceCounts(counts map[int]int) []EffectCommandValueCount {
	out := valueCounts(counts, 16, true)
	for i := range out {
		out[i].Name = datfile.EffectResourceName(int16(out[i].Value))
	}
	return out
}

func operationCounts(counts map[int]int) []EffectCommandValueCount {
	out := valueCounts(counts, 16, true)
	for i := range out {
		out[i].Name = datfile.EffectOperationName(int16(out[i].Value))
	}
	return out
}

func techAttributeCounts(counts map[int]int) []EffectCommandValueCount {
	out := valueCounts(counts, 16, true)
	for i := range out {
		out[i].Name = datfile.EffectTechAttributeName(int16(out[i].Value))
	}
	return out
}

func valueCounts(counts map[int]int, limit int, byCount bool) []EffectCommandValueCount {
	values := make([]EffectCommandValueCount, 0, len(counts))
	for value, count := range counts {
		values = append(values, EffectCommandValueCount{Value: value, Count: count})
	}
	sort.Slice(values, func(i, j int) bool {
		if byCount && values[i].Count != values[j].Count {
			return values[i].Count > values[j].Count
		}
		return values[i].Value < values[j].Value
	})
	if limit > 0 && len(values) > limit {
		values = values[:limit]
	}
	return values
}
