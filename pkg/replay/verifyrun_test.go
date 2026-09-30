package replay

import "testing"

func TestVerifyDataSetCitesDataModLock(t *testing.T) {
	story := &StoryReport{DataSet: DataSetIdentity{Status: "vanilla", Source: "embedded_mod_block", Confidence: "parsed"}}
	claims := verifyDataSet(&DataSetExpectation{Status: "modded", ActiveDataSets: []string{"Example Mod"}}, story)
	if len(claims) != 1 {
		t.Fatalf("claims len = %d, want 1", len(claims))
	}
	claim := claims[0]
	if claim.Status != "fail" || claim.Gotcha != "Data-Mod Lock" || claim.Remediation == "" {
		t.Fatalf("claim = %+v", claim)
	}
}

func TestVerifyRenderExpectationsAreUnknown(t *testing.T) {
	claims := verifyRenderExpectations([]RenderExpectation{{ID: "caption", Description: "Caption should be visible"}})
	if len(claims) != 1 {
		t.Fatalf("claims len = %d, want 1", len(claims))
	}
	claim := claims[0]
	if claim.Status != "unknown" || claim.Tier != "render_verified" || claim.Source != "render_oracle_missing" {
		t.Fatalf("claim = %+v", claim)
	}
}

func TestTelemetryExpectationMatchesFields(t *testing.T) {
	line := "telemetry SDS EVT mod_loaded name=Example version=1"
	if !telemetryLineMatches(line, TelemetryExpectation{Name: "mod_loaded", Fields: map[string]string{"name": "Example"}}) {
		t.Fatal("expected telemetry line to match")
	}
	if telemetryLineMatches(line, TelemetryExpectation{Name: "mod_loaded", Fields: map[string]string{"name": "Other"}}) {
		t.Fatal("expected telemetry line not to match")
	}
}

func TestVerifyPlayers(t *testing.T) {
	story := &StoryReport{Players: []FeedbackPlayer{{PlayerID: 1}, {PlayerID: 2}}}
	claims := verifyPlayers(&PlayersExpectation{MinCount: 2, ExactCount: 2, RequiredIDs: []int{1}, ForbiddenIDs: []int{3}}, story)
	if len(claims) != 4 {
		t.Fatalf("claims len = %d, want 4", len(claims))
	}
	for _, claim := range claims {
		if claim.Status != "pass" {
			t.Fatalf("claim = %+v", claim)
		}
	}
	claims = verifyPlayers(&PlayersExpectation{ExactCount: 3, RequiredIDs: []int{4}, ForbiddenIDs: []int{2}}, story)
	for _, claim := range claims {
		if claim.Status != "fail" {
			t.Fatalf("claim = %+v, want fail", claim)
		}
	}
}

func TestVerifyPlayerProfile(t *testing.T) {
	profile := &PlayerProfileReport{
		Coverage: ProfileCoverage{ReplayActions: 100, DecodedPercent: 95},
		Players:  []PlayerProfile{{PlayerID: 1, Actions: 20, DecodedActions: 18}},
	}
	claims := verifyPlayerProfile(&ProfileExpectation{
		MinReplayActions:  50,
		MinDecodedPercent: 90,
		Players:           []PlayerActionExpectation{{PlayerID: 1, MinActions: 10, MinDecodedPercent: 80}},
	}, profile)
	if len(claims) != 4 {
		t.Fatalf("claims len = %d, want 4", len(claims))
	}
	for _, claim := range claims {
		if claim.Status != "pass" {
			t.Fatalf("claim = %+v", claim)
		}
	}
	claims = verifyPlayerProfile(&ProfileExpectation{
		MinReplayActions:  101,
		MinDecodedPercent: 96,
		Players:           []PlayerActionExpectation{{PlayerID: 1, MinActions: 21, MinDecodedPercent: 91}, {PlayerID: 2, MinActions: 1}},
	}, profile)
	for _, claim := range claims {
		if claim.Status != "fail" {
			t.Fatalf("claim = %+v, want fail", claim)
		}
	}
}
