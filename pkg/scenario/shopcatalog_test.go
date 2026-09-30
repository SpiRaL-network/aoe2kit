package scenario

import (
	"strings"
	"testing"
)

func TestShopCatalogRecipeGeneratesTokenShopRecipe(t *testing.T) {
	x := 100.5
	y := 101.5
	ref := 5001
	catalog := ShopCatalog{
		Name:             "generic shop",
		Player:           2,
		SetupTriggerName: "Shop Setup",
		ClearTriggers:    true,
		Cleanup:          true,
		Area:             &ShopCatalogArea{X1: 90, Y1: 90, X2: 120, Y2: 120},
		Shops: []ShopCatalogShop{
			{
				Name:        "Weapon Tent",
				Description: "Train a renamed token to buy an item.",
				ShopUnit:    1097,
				X:           &x,
				Y:           &y,
				ReferenceID: &ref,
				Caption:     "WEAPON SHOP",
				Items: []ShopCatalogItem{
					{
						Label:       "Quickbrand",
						Description: "Placeholder weapon.",
						TokenUnit:   74,
						Costs:       []ShopCatalogItemCost{{Resource: "gold", Amount: 100}},
					},
					{
						Label:       "Sentinel",
						Description: "Placeholder shield.",
						TokenUnit:   93,
						Button:      4,
						Costs: []ShopCatalogItemCost{
							{Resource: "wood", Amount: 1},
							{Resource: "stone", Amount: 275},
						},
					},
				},
			},
		},
	}

	report, err := ShopCatalogRecipe(catalog)
	if err != nil {
		t.Fatalf("ShopCatalogRecipe: %v", err)
	}
	if report.ShopCount != 1 || report.ItemCount != 2 {
		t.Fatalf("summary got shops=%d items=%d", report.ShopCount, report.ItemCount)
	}
	if got := len(report.Recipe.Triggers); got != 4 {
		t.Fatalf("trigger count = %d, want clear + setup + 2 cleanup", got)
	}
	if report.Recipe.Triggers[0].Op != "clear_triggers" {
		t.Fatalf("first trigger op = %q, want clear_triggers", report.Recipe.Triggers[0].Op)
	}
	setup := report.Recipe.Triggers[1]
	if setup.Name != "Shop Setup" {
		t.Fatalf("setup trigger name = %q", setup.Name)
	}
	if len(report.Recipe.Units) != 1 {
		t.Fatalf("unit count = %d, want 1", len(report.Recipe.Units))
	}
	if report.Recipe.Units[0].CaptionString != "WEAPON SHOP" || report.Recipe.Units[0].ReferenceID == nil || *report.Recipe.Units[0].ReferenceID != ref {
		t.Fatalf("shop unit not carried through: %+v", report.Recipe.Units[0])
	}

	counts := map[string]int{}
	for _, effect := range setup.Effects {
		counts[effect.Op]++
	}
	for _, want := range []struct {
		op    string
		count int
	}{
		{"change_object_name", 3},
		{"change_object_description", 3},
		{"enable_disable_object", 2},
		{"add_train_location", 2},
		{"change_object_cost", 2},
	} {
		if counts[want.op] != want.count {
			t.Fatalf("%s effects = %d, want %d", want.op, counts[want.op], want.count)
		}
	}
	var quickButton, sentinelButton int
	var sentinelCost EffectRecipe
	for _, effect := range setup.Effects {
		if effect.Op == "add_train_location" && effect.ObjectListUnitID != nil && *effect.ObjectListUnitID == 74 {
			quickButton = *effect.ButtonLocation
		}
		if effect.Op == "add_train_location" && effect.ObjectListUnitID != nil && *effect.ObjectListUnitID == 93 {
			sentinelButton = *effect.ButtonLocation
		}
		if effect.Op == "change_object_cost" && effect.ObjectListUnitID != nil && *effect.ObjectListUnitID == 93 {
			sentinelCost = effect
		}
	}
	if quickButton != 1 || sentinelButton != 4 {
		t.Fatalf("buttons quick=%d sentinel=%d, want 1 and 4", quickButton, sentinelButton)
	}
	if sentinelCost.Resource1 == nil || *sentinelCost.Resource1 != 1 || sentinelCost.Resource1Quantity == nil || *sentinelCost.Resource1Quantity != 1 {
		t.Fatalf("sentinel first cost wrong: %+v", sentinelCost)
	}
	if sentinelCost.Resource2 == nil || *sentinelCost.Resource2 != 2 || sentinelCost.Resource2Quantity == nil || *sentinelCost.Resource2Quantity != 275 {
		t.Fatalf("sentinel second cost wrong: %+v", sentinelCost)
	}
	if report.Recipe.Triggers[2].Effects[0].Op != "remove_object" || report.Recipe.Triggers[3].Conditions[0].Op != "own_objects" {
		t.Fatalf("cleanup triggers not generated as expected: %+v %+v", report.Recipe.Triggers[2], report.Recipe.Triggers[3])
	}
}

func TestShopCatalogRecipeValidation(t *testing.T) {
	catalog := ShopCatalog{
		Player: 1,
		Shops: []ShopCatalogShop{
			{
				Name:     "Bad Shop",
				ShopUnit: 1097,
				Items: []ShopCatalogItem{
					{Label: "A", TokenUnit: 74, Button: 1},
					{Label: "B", TokenUnit: 74, Button: 2},
				},
			},
		},
	}
	if _, err := ShopCatalogRecipe(catalog); err == nil || !strings.Contains(err.Error(), "reused") {
		t.Fatalf("expected reused token error, got %v", err)
	}

	catalog.Shops[0].Items[1].TokenUnit = 93
	catalog.Shops[0].Items[1].Button = 17
	if _, err := ShopCatalogRecipe(catalog); err == nil || !strings.Contains(err.Error(), "button 17") {
		t.Fatalf("expected button range error, got %v", err)
	}

	catalog.Shops[0].Items[1].Button = 2
	catalog.Shops[0].Items[1].Costs = []ShopCatalogItemCost{
		{Resource: "food", Amount: 1},
		{Resource: "wood", Amount: 1},
		{Resource: "stone", Amount: 1},
		{Resource: "gold", Amount: 1},
	}
	if _, err := ShopCatalogRecipe(catalog); err == nil || !strings.Contains(err.Error(), "at most 3") {
		t.Fatalf("expected cost limit error, got %v", err)
	}
}
