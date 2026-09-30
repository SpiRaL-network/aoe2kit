package scenario

import "fmt"

// LayoutDocument is the migration seam between the three layers: byte
// preservation, semantic values, and raw-preserving variable sections.
type LayoutDocument struct {
	Raw      PreservedDocument
	Semantic SemanticDocument
	Triggers RawTriggerSection
}

// OpenLayoutDocument builds the owned document model while the legacy parser
// still supplies temporary section locations. No bytes are discarded, and the
// trigger codec does not interpret unknown record fields.
func OpenLayoutDocument(path string) (LayoutDocument, error) {
	file, err := Open(path)
	if err != nil {
		return LayoutDocument{}, err
	}
	triggers, err := file.RawTriggerSection()
	if err != nil {
		return LayoutDocument{}, fmt.Errorf("raw triggers: %w", err)
	}
	doc := LayoutDocument{
		Raw:      file.PreservedDocument(),
		Semantic: file.SemanticDocument(),
		Triggers: triggers,
	}
	if err := ValidateSemanticMirrors(doc.Semantic); err != nil {
		return LayoutDocument{}, err
	}
	return doc, nil
}

func (d LayoutDocument) Write(path string) error {
	return d.Raw.Write(path)
}
