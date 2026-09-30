package scenario

import "errors"

// openMapReadable parses only the structurally stable prefix through Map.
// This is also the safe read path for a newer scenario format whose later
// sections have not been mapped yet. It intentionally does not enable writes.
func openMapReadable(path string) (*File, error) {
	return OpenWithOptions(path, ParseOptions{StopBeforeSection: "Units"})
}

// MapFile reads map metadata without requiring later sections such as Units or
// Triggers to be understood.
func MapFile(path string) (*MapInfo, error) {
	file, err := openMapReadable(path)
	if err != nil {
		return nil, err
	}
	if file.Map == nil {
		return nil, errors.New("missing Map section")
	}
	mapInfo := *file.Map
	return &mapInfo, nil
}
