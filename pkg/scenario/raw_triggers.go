package scenario

import "fmt"

// RawTriggerSection is the preservation-first trigger codec. It owns record
// framing and leaves every field inside a record opaque until evidence earns a
// typed decoder.
//
// WE3_Hautevilles_3 is intentionally parked here: the original/editor-saved
// Rosetta pair proves a five-byte separator after a valid first record,
// tombstone/padding records, and a legacy condition-marker form, but does not
// yet provide a second oracle for the per-effect legacy layout. The original
// and twin are retained in the private workshop under runs/convert-official;
// do not promote guessed semantics. Opaque preservation remains exact.
type RawTriggerSection struct {
	Prefix   []byte
	Records  []RawTriggerRecord
	Suffix   []byte
	Original []byte
}

type RawTriggerRecord struct {
	Index int
	Bytes []byte
}

func (f *File) RawTriggerSection() (RawTriggerSection, error) {
	if f.root == nil {
		return RawTriggerSection{}, fmt.Errorf("scenario body is not parsed")
	}
	section := f.root.section("Triggers")
	if section == nil {
		return RawTriggerSection{}, fmt.Errorf("scenario has no Triggers section")
	}
	raw := section.raw()
	field := section.field("trigger_data")
	if field == nil {
		return RawTriggerSection{}, fmt.Errorf("Triggers has no trigger_data record field")
	}
	out := RawTriggerSection{Original: append([]byte(nil), raw...)}
	base := section.Start
	previous := 0
	for i, node := range field.Elements {
		start, end := node.Start-base, node.End-base
		if start < previous || end < start || end > len(raw) {
			return RawTriggerSection{}, fmt.Errorf("trigger record %d span %d:%d outside section length %d", i, start, end, len(raw))
		}
		if i == 0 {
			out.Prefix = append([]byte(nil), raw[:start]...)
		} else if start > previous {
			out.Suffix = append(out.Suffix, raw[previous:start]...)
		}
		out.Records = append(out.Records, RawTriggerRecord{Index: i, Bytes: append([]byte(nil), raw[start:end]...)})
		previous = end
	}
	if len(out.Records) == 0 {
		out.Prefix = append([]byte(nil), raw...)
	} else if previous < len(raw) {
		out.Suffix = append(out.Suffix, raw[previous:]...)
	}
	return out, nil
}

func (s RawTriggerSection) Rebuild() []byte {
	out := make([]byte, 0, len(s.Original))
	out = append(out, s.Prefix...)
	for _, record := range s.Records {
		out = append(out, record.Bytes...)
	}
	out = append(out, s.Suffix...)
	return out
}

func (s RawTriggerSection) ReplaceRecord(index int, payload []byte) error {
	if index < 0 || index >= len(s.Records) {
		return fmt.Errorf("trigger record index %d out of range", index)
	}
	s.Records[index].Bytes = append([]byte(nil), payload...)
	return nil
}

func (s RawTriggerSection) Unchanged() bool {
	got := s.Rebuild()
	if len(got) != len(s.Original) {
		return false
	}
	for i := range got {
		if got[i] != s.Original[i] {
			return false
		}
	}
	return true
}
