package scenario

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	ShopCatalogDefaultPlayer    = 1
	ShopCatalogDefaultTrainTime = 1
	ShopCatalogDefaultHotkey    = 0
	ShopCatalogDefaultAreaMin   = 0
	ShopCatalogDefaultAreaMax   = 479
)

type ShopCatalog struct {
	Name             string            `json:"name,omitempty"`
	Player           int               `json:"player,omitempty"`
	SetupTriggerName string            `json:"setup_trigger_name,omitempty"`
	Cleanup          bool              `json:"cleanup,omitempty"`
	ClearTriggers    bool              `json:"clear_triggers,omitempty"`
	RenameShops      *bool             `json:"rename_shops,omitempty"`
	Area             *ShopCatalogArea  `json:"area,omitempty"`
	Shops            []ShopCatalogShop `json:"shops"`
}

type ShopCatalogShop struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ShopUnit    int               `json:"shop_unit"`
	Player      int               `json:"player,omitempty"`
	X           *float64          `json:"x,omitempty"`
	Y           *float64          `json:"y,omitempty"`
	ReferenceID *int              `json:"reference_id,omitempty"`
	Caption     string            `json:"caption,omitempty"`
	Items       []ShopCatalogItem `json:"items"`
}

type ShopCatalogItem struct {
	Label       string                `json:"label"`
	Description string                `json:"description,omitempty"`
	TokenUnit   int                   `json:"token_unit"`
	Button      int                   `json:"button,omitempty"`
	TrainTime   *int                  `json:"train_time,omitempty"`
	Hotkey      *int                  `json:"hotkey,omitempty"`
	Costs       []ShopCatalogItemCost `json:"costs,omitempty"`
}

type ShopCatalogItemCost struct {
	ResourceID *int   `json:"resource_id,omitempty"`
	Resource   string `json:"resource,omitempty"`
	Amount     int    `json:"amount"`
}

type ShopCatalogArea struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

type ShopCatalogReport struct {
	Name          string               `json:"name,omitempty"`
	Player        int                  `json:"player"`
	ShopCount     int                  `json:"shop_count"`
	ItemCount     int                  `json:"item_count"`
	Cleanup       bool                 `json:"cleanup"`
	ClearTriggers bool                 `json:"clear_triggers"`
	Recipe        Recipe               `json:"recipe"`
	ShopSummaries []ShopCatalogSummary `json:"shop_summaries"`
	Warnings      []string             `json:"warnings,omitempty"`
	Verification  string               `json:"verification"`
}

type ShopCatalogSummary struct {
	Name      string `json:"name"`
	ShopUnit  int    `json:"shop_unit"`
	Player    int    `json:"player"`
	ItemCount int    `json:"item_count"`
	ButtonMin int    `json:"button_min,omitempty"`
	ButtonMax int    `json:"button_max,omitempty"`
	HasUnit   bool   `json:"has_unit"`
}

func LoadShopCatalogFile(path string) (ShopCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ShopCatalog{}, err
	}
	var catalog ShopCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return ShopCatalog{}, err
	}
	return catalog, nil
}

func ShopCatalogRecipe(catalog ShopCatalog) (ShopCatalogReport, error) {
	if catalog.Player == 0 {
		catalog.Player = ShopCatalogDefaultPlayer
	}
	if catalog.SetupTriggerName == "" {
		catalog.SetupTriggerName = "A2K Shop Catalog Setup"
	}
	if catalog.Area == nil {
		catalog.Area = &ShopCatalogArea{X1: ShopCatalogDefaultAreaMin, Y1: ShopCatalogDefaultAreaMin, X2: ShopCatalogDefaultAreaMax, Y2: ShopCatalogDefaultAreaMax}
	}
	renameShops := true
	if catalog.RenameShops != nil {
		renameShops = *catalog.RenameShops
	}
	if err := validateShopCatalog(catalog); err != nil {
		return ShopCatalogReport{}, err
	}

	active := true
	noLoop := false
	loop := true
	timer := 0
	setup := TriggerRecipe{
		Op:      "add_trigger",
		Name:    catalog.SetupTriggerName,
		Enabled: &active,
		Looping: &noLoop,
		Conditions: []ConditionRecipe{
			{Op: "timer", Timer: &timer},
		},
	}
	var prefixTriggers []TriggerRecipe
	if catalog.ClearTriggers {
		prefixTriggers = append(prefixTriggers, TriggerRecipe{Op: "clear_triggers"})
	}
	var cleanupTriggers []TriggerRecipe
	var units []UnitRecipe
	var summaries []ShopCatalogSummary
	itemCount := 0
	for _, shop := range catalog.Shops {
		player := catalog.Player
		if shop.Player != 0 {
			player = shop.Player
		}
		if renameShops {
			setup.Effects = append(setup.Effects, shopRenameEffects(player, shop)...)
		}
		if shop.X != nil && shop.Y != nil {
			units = append(units, shopUnitRecipe(player, shop))
		}
		buttonMin := 0
		buttonMax := 0
		for i, item := range shop.Items {
			button := item.Button
			if button == 0 {
				button = i + 1
			}
			if buttonMin == 0 || button < buttonMin {
				buttonMin = button
			}
			if button > buttonMax {
				buttonMax = button
			}
			item.Button = button
			setup.Effects = append(setup.Effects, itemSetupEffects(player, shop.ShopUnit, item)...)
			if catalog.Cleanup {
				cleanupTriggers = append(cleanupTriggers, itemCleanupTrigger(player, item, *catalog.Area, &active, &loop))
			}
			itemCount++
		}
		summaries = append(summaries, ShopCatalogSummary{
			Name:      shop.Name,
			ShopUnit:  shop.ShopUnit,
			Player:    player,
			ItemCount: len(shop.Items),
			ButtonMin: buttonMin,
			ButtonMax: buttonMax,
			HasUnit:   shop.X != nil && shop.Y != nil,
		})
	}
	triggers := append(prefixTriggers, setup)
	triggers = append(triggers, cleanupTriggers...)
	recipe := Recipe{Triggers: triggers, Units: units}
	report := ShopCatalogReport{
		Name:          catalog.Name,
		Player:        catalog.Player,
		ShopCount:     len(catalog.Shops),
		ItemCount:     itemCount,
		Cleanup:       catalog.Cleanup,
		ClearTriggers: catalog.ClearTriggers,
		Recipe:        recipe,
		ShopSummaries: summaries,
		Verification:  "recipe_generated_not_applied",
	}
	return report, nil
}

func validateShopCatalog(catalog ShopCatalog) error {
	if catalog.Player < 0 || catalog.Player > 8 {
		return fmt.Errorf("player %d out of range 0..8", catalog.Player)
	}
	if len(catalog.Shops) == 0 {
		return fmt.Errorf("shop catalog requires at least one shop")
	}
	tokenOwners := map[int]string{}
	for _, shop := range catalog.Shops {
		if shop.ShopUnit <= 0 {
			return fmt.Errorf("shop %q requires positive shop_unit", shop.Name)
		}
		if shop.Player < 0 || shop.Player > 8 {
			return fmt.Errorf("shop %q player %d out of range 0..8", shop.Name, shop.Player)
		}
		if len(shop.Items) == 0 {
			return fmt.Errorf("shop %q requires at least one item", shop.Name)
		}
		buttons := map[int]string{}
		for i, item := range shop.Items {
			if strings.TrimSpace(item.Label) == "" {
				return fmt.Errorf("shop %q item %d requires label", shop.Name, i)
			}
			if item.TokenUnit <= 0 {
				return fmt.Errorf("shop %q item %q requires positive token_unit", shop.Name, item.Label)
			}
			if owner, ok := tokenOwners[item.TokenUnit]; ok {
				return fmt.Errorf("token_unit %d reused by %q and %q", item.TokenUnit, owner, item.Label)
			}
			tokenOwners[item.TokenUnit] = item.Label
			button := item.Button
			if button == 0 {
				button = i + 1
			}
			if button < 1 || button > 16 {
				return fmt.Errorf("shop %q item %q button %d out of range 1..16", shop.Name, item.Label, button)
			}
			if owner, ok := buttons[button]; ok {
				return fmt.Errorf("shop %q button %d reused by %q and %q", shop.Name, button, owner, item.Label)
			}
			buttons[button] = item.Label
			if len(item.Costs) > 3 {
				return fmt.Errorf("shop %q item %q has %d costs; change_object_cost supports at most 3", shop.Name, item.Label, len(item.Costs))
			}
			for _, cost := range item.Costs {
				if _, err := costResourceID(cost); err != nil {
					return fmt.Errorf("shop %q item %q: %w", shop.Name, item.Label, err)
				}
				if cost.Amount < 0 {
					return fmt.Errorf("shop %q item %q cost amount %d is negative", shop.Name, item.Label, cost.Amount)
				}
			}
		}
	}
	return nil
}

func shopRenameEffects(player int, shop ShopCatalogShop) []EffectRecipe {
	effects := []EffectRecipe{
		{Op: "change_object_name", SourcePlayer: &player, ObjectListUnitID: &shop.ShopUnit, Message: shop.Name},
	}
	if shop.Description != "" {
		effects = append(effects, EffectRecipe{Op: "change_object_description", SourcePlayer: &player, ObjectListUnitID: &shop.ShopUnit, Message: shop.Description})
	}
	return effects
}

func itemSetupEffects(player, shopUnit int, item ShopCatalogItem) []EffectRecipe {
	trainTime := ShopCatalogDefaultTrainTime
	if item.TrainTime != nil {
		trainTime = *item.TrainTime
	}
	hotkey := ShopCatalogDefaultHotkey
	if item.Hotkey != nil {
		hotkey = *item.Hotkey
	}
	effects := []EffectRecipe{
		{Op: "enable_disable_object", SourcePlayer: &player, ObjectListUnitID: &item.TokenUnit, Enabled: intPtrScenario(1)},
		{
			Op:                "add_train_location",
			SourcePlayer:      &player,
			ObjectListUnitID:  &item.TokenUnit,
			ObjectListUnitID2: &shopUnit,
			ButtonLocation:    &item.Button,
			TrainTime:         &trainTime,
			Hotkey:            &hotkey,
		},
		{Op: "change_object_name", SourcePlayer: &player, ObjectListUnitID: &item.TokenUnit, Message: item.Label},
	}
	if item.Description != "" {
		effects = append(effects, EffectRecipe{Op: "change_object_description", SourcePlayer: &player, ObjectListUnitID: &item.TokenUnit, Message: item.Description})
	}
	effects = append(effects, itemCostEffect(player, item))
	return effects
}

func itemCostEffect(player int, item ShopCatalogItem) EffectRecipe {
	resources := []int{-1, -1, -1}
	amounts := []int{-1, -1, -1}
	for i, cost := range item.Costs {
		if i >= 3 {
			break
		}
		res, _ := costResourceID(cost)
		resources[i] = res
		amounts[i] = cost.Amount
	}
	return EffectRecipe{
		Op:                "change_object_cost",
		SourcePlayer:      &player,
		ObjectListUnitID:  &item.TokenUnit,
		Resource1:         &resources[0],
		Resource1Quantity: &amounts[0],
		Resource2:         &resources[1],
		Resource2Quantity: &amounts[1],
		Resource3:         &resources[2],
		Resource3Quantity: &amounts[2],
	}
}

func itemCleanupTrigger(player int, item ShopCatalogItem, area ShopCatalogArea, active *bool, loop *bool) TriggerRecipe {
	qty := 1
	return TriggerRecipe{
		Op:      "add_trigger",
		Name:    fmt.Sprintf("A2K Shop Cleanup %s", sanitizeShopTriggerName(item.Label)),
		Enabled: active,
		Looping: loop,
		Conditions: []ConditionRecipe{
			{Op: "own_objects", SourcePlayer: &player, ObjectList: &item.TokenUnit, Quantity: &qty},
		},
		Effects: []EffectRecipe{
			{
				Op:               "remove_object",
				SourcePlayer:     &player,
				ObjectListUnitID: &item.TokenUnit,
				AreaX1:           &area.X1,
				AreaY1:           &area.Y1,
				AreaX2:           &area.X2,
				AreaY2:           &area.Y2,
				MaxUnitsAffected: &qty,
			},
		},
	}
}

func shopUnitRecipe(player int, shop ShopCatalogShop) UnitRecipe {
	status := 2
	z := 0.0
	unit := UnitRecipe{
		Op:            "add_unit",
		Player:        player,
		UnitConst:     shop.ShopUnit,
		X:             shop.X,
		Y:             shop.Y,
		Z:             &z,
		Status:        &status,
		ReferenceID:   shop.ReferenceID,
		CaptionString: shop.Caption,
	}
	return unit
}

func costResourceID(cost ShopCatalogItemCost) (int, error) {
	if cost.ResourceID != nil {
		if *cost.ResourceID < 0 {
			return 0, fmt.Errorf("resource_id %d is negative", *cost.ResourceID)
		}
		return *cost.ResourceID, nil
	}
	switch strings.ToLower(strings.TrimSpace(cost.Resource)) {
	case "food", "mp":
		return 0, nil
	case "wood", "ap":
		return 1, nil
	case "stone":
		return 2, nil
	case "gold":
		return 3, nil
	case "pop", "population", "pop_cap":
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown resource %q; use food, wood, stone, gold, pop, or resource_id", cost.Resource)
	}
}

func sanitizeShopTriggerName(label string) string {
	replacer := strings.NewReplacer("+", "plus", "-", "minus", "/", "_", "%", "pct", ",", "", ":", "", ";", "", " ", "_")
	name := replacer.Replace(label)
	if len(name) > 54 {
		name = name[:54]
	}
	if name == "" {
		return "item"
	}
	return name
}

func intPtrScenario(v int) *int {
	return &v
}

func ShopCatalogExample() ShopCatalog {
	return ShopCatalog{
		Name:    "example token shop",
		Player:  1,
		Cleanup: true,
		Shops: []ShopCatalogShop{
			{
				Name:        "Example Shop",
				Description: "Generated by kit scen shop-catalog.",
				ShopUnit:    1097,
				Items: []ShopCatalogItem{
					{Label: "Quickbrand", Description: "Placeholder item.", TokenUnit: 74, Button: 1, Costs: []ShopCatalogItemCost{{Resource: "gold", Amount: 100}}},
					{Label: "Sentinel", Description: "Dual-cost placeholder item.", TokenUnit: 93, Button: 2, Costs: []ShopCatalogItemCost{{Resource: "wood", Amount: 1}, {Resource: "stone", Amount: 275}}},
				},
			},
		},
	}
}

func SortedShopCatalogCostResources(costs []ShopCatalogItemCost) []int {
	out := make([]int, 0, len(costs))
	for _, cost := range costs {
		res, err := costResourceID(cost)
		if err == nil {
			out = append(out, res)
		}
	}
	sort.Ints(out)
	return out
}
