package kit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gfx"
)

type VerifyReport struct {
	Root          string                 `json:"root"`
	Version       string                 `json:"version"`
	Profile       string                 `json:"profile,omitempty"`
	Generation    int                    `json:"generation,omitempty"`
	SourceBuildOK bool                   `json:"source_build_ok"`
	DocsOK        bool                   `json:"docs_ok"`
	ManifestOK    bool                   `json:"manifest_ok"`
	InventoryOK   bool                   `json:"inventory_ok"`
	ArtifactsOK   bool                   `json:"artifacts_ok"`
	GoAvailable   bool                   `json:"go_available"`
	Verification  aoe2.VerificationClaim `json:"verification"`
	Warnings      []string               `json:"warnings,omitempty"`
	Errors        []string               `json:"errors,omitempty"`
}

func Verify(root string, runGoChecks bool) VerifyReport {
	report := VerifyReport{Root: root, Version: Version}

	// Hold the tree to the contract it declares, not to the fullest one. A
	// handoff archive that carries no docs/ is complete for what it claims to be.
	marker, markerErr := ReadProfileMarker(root)
	report.Profile = marker.Profile
	report.Generation = marker.Generation
	if markerErr != nil {
		// An archive whose declared contract cannot be read is not verifiable.
		report.Errors = append(report.Errors, markerErr.Error())
	}
	requiredDocs, reqErr := RequiredDocsIn(root, PackProfile(marker.Profile))
	if reqErr != nil {
		report.Errors = append(report.Errors, reqErr.Error())
	}
	for _, doc := range requiredDocs {
		if !exists(filepath.Join(root, doc)) {
			report.Errors = append(report.Errors, "missing required artifact: "+doc)
		}
	}
	report.DocsOK = len(report.Errors) == 0

	generated, err := ManifestJSON()
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("generate manifest: %v", err))
	} else {
		snapshotPath := filepath.Join(root, "KIT_MANIFEST.json")
		snapshot, err := os.ReadFile(snapshotPath)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("read KIT_MANIFEST.json: %v", err))
		} else if !bytes.Equal(bytes.TrimSpace(snapshot), bytes.TrimSpace(generated)) {
			report.Errors = append(report.Errors, "KIT_MANIFEST.json drift: regenerate with `kit manifest > KIT_MANIFEST.json`")
		} else {
			report.ManifestOK = true
		}
	}

	inventory, err := Scan(root)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("inventory scan: %v", err))
	} else {
		report.InventoryOK = true
		for _, warning := range inventory.Warnings {
			report.Warnings = append(report.Warnings, "inventory: "+warning)
		}
		for _, mod := range inventory.ModDirs {
			for _, err := range mod.Errors {
				report.Errors = append(report.Errors, fmt.Sprintf("mod %s: %s", mod.Path, err))
			}
		}
		artifactErrorsBefore := len(report.Errors)
		for _, file := range inventory.Files {
			switch file.Kind {
			case "dat":
				if _, err := datfile.Open(file.Path); err != nil {
					report.Errors = append(report.Errors, fmt.Sprintf("dat parse %s: %v", file.Path, err))
				}
			case "gfx":
				if _, err := gfx.OpenSLD(file.Path); err != nil {
					report.Errors = append(report.Errors, fmt.Sprintf("gfx parse %s: %v", file.Path, err))
				}
			}
		}
		report.ArtifactsOK = len(report.Errors) == artifactErrorsBefore
	}

	if _, err := exec.LookPath("go"); err == nil {
		report.GoAvailable = true
		if err := runCommand(root, "go", "build", "./..."); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("go build ./...: %v", err))
		} else {
			report.SourceBuildOK = true
		}
		if runGoChecks {
			if err := runCommand(root, "go", "test", "./..."); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("go test ./...: %v", err))
			}
		}
	} else {
		report.Errors = append(report.Errors, "Go toolchain not found; source build cannot be verified")
	}

	report.Verification = aoe2.StructureVerification(report.OK())
	return report
}

func (r VerifyReport) OK() bool {
	return len(r.Errors) == 0
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if len(out) == 0 {
		return err
	}
	return errors.New(string(out))
}
