package preview

import (
	"image"
	"image/color"
	"math"
)

// Screen directions clockwise from top, transformed into scenario coordinates
// by the same rotate270 orientation used by the terrain renderer.
var blendNeighbors = [8]image.Point{{1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}}

// blendMaskTiles selects directional records, not a generated approximation of
// their shape. Record meanings come from the classic format documentation;
// applicability to DE and the coordinate variant choice remain candidates.
func blendMaskTiles(neighbors uint8, x, y int) (ids [5]int, count int) {
	sides := (neighbors>>1)&1 | (neighbors>>2)&2 | (neighbors>>3)&4 | (neighbors>>4)&8
	// NE, SE, SW, NW in the low four bits; 0..15 have four variants per side.
	sideTiles := [16]int{-1, 4, 0, 25, 8, 20, 24, 26, 12, 23, 21, 29, 22, 28, 27, 30}
	if id := sideTiles[sides]; id >= 0 {
		if sides == 1 || sides == 4 {
			id += x & 3
		} else if sides == 2 || sides == 8 {
			id += y & 3
		}
		ids[count], count = id, count+1
	}
	for corner, id := range [4]int{18, 16, 17, 19} {
		bit := corner * 2
		adjacent := uint8(1<<uint((bit+7)%8) | 1<<uint((bit+1)%8))
		if neighbors&(1<<uint(bit)) != 0 && neighbors&adjacent == 0 {
			ids[count], count = id, count+1
		}
	}
	return
}

// Classic pair-to-mode facts, expressed by class precedence and exceptions.
// No claim is made that DAT blend_type alone is a blendomatic mode index.
func blendMode(a, b int32) int {
	if a < 0 || b < 0 || a > 7 || b > 7 {
		return -1
	}
	if a > b {
		a, b = b, a
	}
	if b == 7 && a == 1 {
		return 3
	}
	if b == 6 && a == 2 {
		return 1
	}
	if a >= 3 && b == 4 {
		return 7
	}
	if a == 3 && b == 3 {
		return 0
	}
	if a == 1 && b == 2 {
		return 3
	}
	return [8]int{2, 3, 2, 1, 1, 6, 5, 4}[b]
}

func blendAlpha(mask *image.Gray, u, v float64) float64 {
	// The packed diamond includes its four tips. Nearest sampling retains the
	// installed mask's hard edge pixels rather than imposing a radial cutoff.
	x := int(math.Round((u + v) * 48))
	y := int(math.Round((1 - u + v) * 24))
	// Round inside the scanline at diamond boundaries, never into its padding.
	half := 2 * min(y, 48-y)
	x = max(48-half, min(48+half, x))
	// In the measured file, zero occupies the named incoming edge/corner;
	// 128 occupies the retained interior. The stored weight retains the base.
	return 1 - float64(mask.GrayAt(x, y).Y)/255
}

func terrainEdgePixel(tiles []Tile, width, height int, mats map[int]terrainMaterial, x, y float64, opts TerrainTextureOptions) color.RGBA {
	tx, ty := int(x), int(y)
	current := tiles[ty*width+tx]
	base := mats[tileMaterial(current)]
	c := layeredPixel(current, mats, x, y, opts.Repeat, opts.TextureRepeat)
	type influence struct {
		tile      Tile
		neighbors uint8
	}
	var groups [8]influence
	n := 0
	for bit, delta := range blendNeighbors {
		nx, ny := tx+delta.X, ty+delta.Y
		if nx < 0 || ny < 0 || nx >= width || ny >= height {
			continue
		}
		t := tiles[ny*width+nx]
		m := mats[tileMaterial(t)]
		if m.priority <= base.priority {
			continue
		}
		at := n
		for i := 0; i < n; i++ {
			if groups[i].tile.TerrainID == t.TerrainID && tileMaterial(groups[i].tile) == tileMaterial(t) {
				at = i
				break
			}
		}
		if at == n {
			groups[n].tile = t
			n++
		}
		groups[at].neighbors |= 1 << uint(bit)
	}
	// Deterministic dominance order; equal-priority ties use discovery order.
	for i := 1; i < n; i++ {
		for j := i; j > 0 && mats[tileMaterial(groups[j].tile)].priority < mats[tileMaterial(groups[j-1].tile)].priority; j-- {
			groups[j], groups[j-1] = groups[j-1], groups[j]
		}
	}
	for _, g := range groups[:n] {
		m := mats[tileMaterial(g.tile)]
		mode := blendMode(base.blendType, m.blendType)
		if mode < 0 || mode >= len(m.blends) {
			continue
		}
		ids, count := blendMaskTiles(g.neighbors, tx, ty)
		alpha := 0.0
		for _, id := range ids[:count] {
			a := blendAlpha(m.blends[mode].Alpha[id], x-float64(tx), y-float64(ty))
			alpha += (1 - alpha) * a
		}
		c = mixTerrain(c, layeredPixel(g.tile, mats, x, y, opts.Repeat, opts.TextureRepeat), alpha)
	}
	return c
}
