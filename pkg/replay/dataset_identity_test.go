package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestDataSetIdentitySinglePlayerScenarioAndMultiplayerControls(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantStatus   string
		wantDataSet  string
		wantChecksum uint32
		wantWorkshop uint32
		wantPlayers  int
		allowDataSet bool
	}{
		{
			name:         "fireball single-player data mod",
			path:         testfixtures.Path(t, "save-analysis/diagnostics/fireball_run.aoe2record"),
			wantStatus:   "modded",
			wantDataSet:  "Chraes Scenario Playground",
			wantPlayers:  1,
			allowDataSet: true,
		},
		{
			name:         "fireball local data mod checksum",
			path:         testfixtures.Path(t, "save-analysis/fb_run_A.aoe2record"),
			wantStatus:   "modded",
			wantDataSet:  "Chraes Scenario Playground",
			wantChecksum: 413432973,
			wantWorkshop: 516252,
			wantPlayers:  1,
			allowDataSet: true,
		},
		{
			name:         "fireball subscribed data mod checksum",
			path:         testfixtures.Path(t, "save-analysis/fb_run_B.aoe2record"),
			wantStatus:   "modded",
			wantDataSet:  "Chraes Scenario Playground",
			wantChecksum: 2462983196,
			wantWorkshop: 516252,
			wantPlayers:  1,
			allowDataSet: true,
		},
		{
			name:        "v3 diagnostic single-player with ai slot",
			path:        testfixtures.Path(t, "save-analysis/diagnostics/diag_v3_run.aoe2record"),
			wantStatus:  "vanilla",
			wantPlayers: 2,
		},
		{
			name:        "v4 diagnostic single-player with two ai slots",
			path:        testfixtures.Path(t, "save-analysis/diagnostics/v4_run_20260722.aoe2record"),
			wantStatus:  "vanilla",
			wantPlayers: 3,
		},
		{
			name:        "latest CBA multiplayer control",
			path:        testfixtures.Path(t, "save-analysis/latest_cba.aoe2record"),
			wantStatus:  "vanilla",
			wantPlayers: 8,
		},
		{
			name:        "latest Decima multiplayer control",
			path:        testfixtures.Path(t, "save-analysis/rec_latest_20260721.aoe2record"),
			wantStatus:  "vanilla",
			wantPlayers: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat(tt.path); err != nil {
				t.Skipf("golden replay not present: %v", err)
			}
			rec, err := Open(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if rec.DataSet.Error != "" {
				t.Fatalf("data-set identity error = %q", rec.DataSet.Error)
			}
			if rec.DataSet.Confidence != "parsed" {
				t.Fatalf("confidence = %q, want parsed", rec.DataSet.Confidence)
			}
			if rec.DataSet.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", rec.DataSet.Status, tt.wantStatus)
			}
			if tt.wantDataSet != "" && rec.DataSet.ActiveDataSet != tt.wantDataSet {
				t.Fatalf("active data set = %q, want %q", rec.DataSet.ActiveDataSet, tt.wantDataSet)
			}
			if tt.wantChecksum != 0 && rec.DataSet.Checksum != tt.wantChecksum {
				t.Fatalf("checksum = %d, want %d", rec.DataSet.Checksum, tt.wantChecksum)
			}
			if tt.wantWorkshop != 0 && rec.DataSet.WorkshopID != tt.wantWorkshop {
				t.Fatalf("workshop id = %d, want %d", rec.DataSet.WorkshopID, tt.wantWorkshop)
			}
			if !tt.allowDataSet && rec.DataSet.ActiveDataSet != "" {
				t.Fatalf("active data set = %q, want empty", rec.DataSet.ActiveDataSet)
			}
			if !tt.allowDataSet && rec.DataSet.Checksum != 0 {
				t.Fatalf("checksum = %d, want zero for non-modded replay", rec.DataSet.Checksum)
			}
			if len(rec.Players) != tt.wantPlayers {
				t.Fatalf("players = %d, want %d", len(rec.Players), tt.wantPlayers)
			}
		})
	}
}

func TestScanScenarioPathDataSetSuffix(t *testing.T) {
	header := []byte("prefix DAT Command Semantics Test.aoe2scenario:KitSemanticsTest:false suffix")
	if got := scanScenarioPathDataSetSuffix(header); got != "KitSemanticsTest" {
		t.Fatalf("scanScenarioPathDataSetSuffix = %q, want KitSemanticsTest", got)
	}
	if got := scanScenarioPathDataSetSuffix([]byte("prefix Scenario.aoe2scenario::false")); got != "" {
		t.Fatalf("empty dataset suffix = %q, want empty", got)
	}
}
