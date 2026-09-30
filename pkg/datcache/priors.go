package datcache

import "sort"

// SemanticPriorsReport summarizes DAT-derived hypotheses. It deliberately
// does not claim engine semantics; callers should compare it with a harness
// observation before upgrading any row to an engine fact.
type SemanticPriorsReport struct {
	Version                string               `json:"version"`
	CivCount               int                  `json:"civ_count"`
	SourceRows             int                  `json:"source_rows"`
	UnitIdentities         int                  `json:"unit_identities"`
	CollisionSignatures    []CollisionSignature `json:"collision_signatures"`
	RemovalBuckets         []RemovalBucket      `json:"removal_buckets"`
	ZeroCollision          int                  `json:"zero_collision_identities"`
	ZeroCollisionCreatable int                  `json:"zero_collision_creatable"`
	Verification           string               `json:"verification"`
}

type CollisionSignature struct {
	XPositive bool `json:"x_positive"`
	YPositive bool `json:"y_positive"`
	ZPositive bool `json:"z_positive"`
	Units     int  `json:"units"`
	Creatable int  `json:"creatable"`
}

type RemovalBucket struct {
	Name  string `json:"name"`
	Units int    `json:"units"`
}

// SemanticPriors derives the first-pass prediction tables from the cached DAT.
// Unit identities are grouped by unit_index so civ duplication does not skew
// the result. A max aggregation mirrors the canonical SQL used in the ledger.
func (c *Cache) SemanticPriors() (SemanticPriorsReport, error) {
	rows, err := c.Units()
	if err != nil {
		return SemanticPriorsReport{}, err
	}
	type identity struct {
		hitPoints int16
		class     int16
		x, y, z   float32
		creatable bool
	}
	identities := make(map[int]identity)
	for _, row := range rows {
		current := identities[row.Index]
		if row.HitPoints > current.hitPoints {
			current.hitPoints = row.HitPoints
		}
		if row.Class > current.class {
			current.class = row.Class
		}
		if row.CollisionSizeX > current.x {
			current.x = row.CollisionSizeX
		}
		if row.CollisionSizeY > current.y {
			current.y = row.CollisionSizeY
		}
		if row.CollisionSizeZ > current.z {
			current.z = row.CollisionSizeZ
		}
		current.creatable = current.creatable || row.HasCreatable
		identities[row.Index] = current
	}

	collision := make(map[[3]bool]CollisionSignature)
	removal := map[string]int{"class14_hp1": 0, "class14_other": 0, "hp1_other": 0}
	zeroCollisionCreatable := 0
	for _, unit := range identities {
		signature := [3]bool{unit.x > 0, unit.y > 0, unit.z > 0}
		bucket := collision[signature]
		bucket.XPositive, bucket.YPositive, bucket.ZPositive = signature[0], signature[1], signature[2]
		bucket.Units++
		if unit.creatable {
			bucket.Creatable++
		}
		collision[signature] = bucket
		if !signature[0] && !signature[1] && !signature[2] {
			if unit.creatable {
				zeroCollisionCreatable++
			}
		}
		if unit.class == 14 && unit.hitPoints == 1 {
			removal["class14_hp1"]++
		} else if unit.class == 14 {
			removal["class14_other"]++
		} else if unit.hitPoints == 1 {
			removal["hp1_other"]++
		}
	}

	collisions := make([]CollisionSignature, 0, len(collision))
	for _, bucket := range collision {
		collisions = append(collisions, bucket)
	}
	sort.Slice(collisions, func(i, j int) bool {
		if collisions[i].XPositive != collisions[j].XPositive {
			return !collisions[i].XPositive
		}
		if collisions[i].YPositive != collisions[j].YPositive {
			return !collisions[i].YPositive
		}
		return !collisions[i].ZPositive
	})
	removalBuckets := make([]RemovalBucket, 0, len(removal))
	for _, name := range []string{"class14_hp1", "class14_other", "hp1_other"} {
		removalBuckets = append(removalBuckets, RemovalBucket{Name: name, Units: removal[name]})
	}
	return SemanticPriorsReport{
		Version: c.Version, CivCount: c.CivCount, SourceRows: len(rows),
		UnitIdentities: len(identities), CollisionSignatures: collisions,
		RemovalBuckets: removalBuckets, ZeroCollision: collision[[3]bool{false, false, false}].Units,
		ZeroCollisionCreatable: zeroCollisionCreatable,
		Verification:           "dat_derived_prior_not_engine_verified",
	}, nil
}
