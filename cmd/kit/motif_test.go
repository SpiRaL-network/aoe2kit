package main

import (
	"reflect"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestMotifPatchOptionsKeepNormalPatchFlags(t *testing.T) {
	o, rest, e := parseMotifPatchOptions([]string{"--text", "--at", "10.5,20", "--rotate", "90", "--grammar", "review.json", "--game-dir", "game"})
	if e != nil || o.At == nil || o.At.X != 10.5 || o.At.Y != 20 || o.Rotate != 90 || o.Grammar != "review.json" || !reflect.DeepEqual(rest, []string{"--text", "--game-dir", "game"}) {
		t.Fatalf("%+v %v %v", o, rest, e)
	}
	for _, args := range [][]string{{"--at"}, {"--at", "1,2,3"}, {"--at", "x,2"}, {"--rotate", "45"}, {"--grammar"}} {
		if _, _, e := parseMotifPatchOptions(args); e == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestApplyMotifOptionsPreservesOrdinaryRecipes(t *testing.T) {
	r := scenario.Recipe{Units: []scenario.UnitRecipe{{Op: "add_unit", UnitConst: 1062}}}
	got, err := applyMotifOptions(r, motifPatchOptions{})
	if err != nil || !reflect.DeepEqual(got, r) {
		t.Fatalf("ordinary recipe changed: %+v %v", got, err)
	}
	if _, err := applyMotifOptions(r, motifPatchOptions{Grammar: "unused.json"}); err == nil {
		t.Fatal("accepted grammar without transform")
	}
	if _, err := applyMotifOptions(r, motifPatchOptions{At: &scenario.PointRecipe{X: 1, Y: 2}}); err == nil {
		t.Fatal("accepted transform on ordinary recipe")
	}
}
