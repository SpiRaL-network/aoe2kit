package compose

// WoodenRoad is the measured visual-index convention for the Wooden Road kit
// (normally unit 2748). Geometry is preview-derived; it does not make a bridge
// passable. Supplying a different ID assumes it shares the same facet layout.
type WoodenRoad struct {
	Scene  *Scene
	UnitID int
}

func (r WoodenRoad) Square(p Point) { r.Scene.Gaia(r.UnitID, p, 0); r.Scene.Claim(p, 1.1) }

// Curve joins axis strips around block corner (cx,cy), each component +/-1.
// Mouths: (cx,-cy/2) and (-cx/2,cy), measured in tile offsets from the anchor.
func (r WoodenRoad) Curve(p Point, cx, cy int) {
	if (cx != -1 && cx != 1) || (cy != -1 && cy != 1) {
		r.Scene.fail("curve corner must be +/-1 per axis")
		return
	}
	v := 12
	if cx == 1 {
		v += 2
	}
	if cy == 1 {
		v++
	}
	r.Scene.Gaia(r.UnitID, p, v)
	r.Scene.Claim(p, 1.8)
}
func (r WoodenRoad) Ring(p Point) {
	r.Curve(Point{p.X + 1, p.Y + 1}, -1, -1)
	r.Curve(Point{p.X + 1, p.Y - 1}, -1, 1)
	r.Curve(Point{p.X - 1, p.Y + 1}, 1, -1)
	r.Curve(Point{p.X - 1, p.Y - 1}, 1, 1)
}

// AxisStrip uses v0 squares at one-tile pitch; it works in all four directions.
func (r WoodenRoad) AxisStrip(start Point, dx, dy, n int) {
	if !cardinal(dx, dy) || n < 0 || n > 10000 {
		r.Scene.fail("invalid axis strip")
		return
	}
	for k := 0; k < n && r.Scene.Err() == nil; k++ {
		r.Square(Point{start.X + float64(k*dx), start.Y + float64(k*dy)})
	}
}
func cardinal(x, y int) bool {
	return (x == 0 && (y == -1 || y == 1)) || (y == 0 && (x == -1 || x == 1))
}

type roadMouth struct {
	x, y, dx, dy float64
	v            int
}

var diagonalRoadMouths = [4][2]roadMouth{
	{{-.33, -1.05, -1, -1, 5}, {-.34, 1.02, -1, 1, 7}},
	{{1.03, .31, 1, 1, 5}, {-1.04, .33, -1, 1, 7}},
	{{-1.02, -.34, -1, -1, 5}, {1.05, -.35, 1, -1, 7}},
	{{.34, 1.02, 1, 1, 5}, {.35, -1.04, 1, -1, 7}},
}

// DiagonalCorner uses measured mouth offsets for v16..19; v5/v7 strips
// attach at 0.685 along their diagonal and repeat every 1.37 per coordinate.
func (r WoodenRoad) DiagonalCorner(p Point, variant, strips int) {
	if variant < 16 || variant > 19 || strips < 0 || strips > 10000 {
		r.Scene.fail("invalid diagonal corner")
		return
	}
	r.Scene.Gaia(r.UnitID, p, variant)
	r.Scene.Claim(p, 1.8)
	for _, m := range diagonalRoadMouths[variant-16] {
		for k := 0; k < strips && r.Scene.Err() == nil; k++ {
			d := .685 + 1.37*float64(k)
			q := Point{p.X + m.x + m.dx*d, p.Y + m.y + m.dy*d}
			r.Scene.Gaia(r.UnitID, q, m.v)
			r.Scene.Claim(q, 1.1)
		}
	}
}

// Band squares a (1,1) two-square staircase with v9/v8 triangle shoulders.
// Evidence: previewed joins; integer x,y name the first tile, not its centre.
func (r WoodenRoad) Band(x, y, n int) {
	if n < 1 || n > 10000 {
		r.Scene.fail("invalid diagonal band length")
		return
	}
	for k := 0; k < n && r.Scene.Err() == nil; k++ {
		r.Square(Point{float64(x+k) + .5, float64(y+k) + .5})
		r.Square(Point{float64(x+k+1) + .5, float64(y+k) + .5})
	}
	for k := 1; k < n && r.Scene.Err() == nil; k++ {
		r.Scene.Gaia(r.UnitID, Point{float64(x+k) + .5, float64(y+k-2) + .5}, 9)
		r.Scene.Gaia(r.UnitID, Point{float64(x+k) + .5, float64(y+k+1) + .5}, 8)
	}
}

type Turtle struct {
	Road   WoodenRoad
	Next   Point
	DX, DY int
	laid   bool
}

func (t *Turtle) Forward(n int) {
	if !cardinal(t.DX, t.DY) || n < 0 || n > 10000 {
		t.Road.Scene.fail("invalid turtle forward")
		return
	}
	for k := 0; k < n && t.Road.Scene.Err() == nil; k++ {
		t.Road.Square(t.Next)
		t.Next.X += float64(t.DX)
		t.Next.Y += float64(t.DY)
		t.laid = true
	}
}

// Turn must be a right-angle turn after Forward. The resulting Next is the
// centre of the first outgoing square, matched to the curve's measured mouth.
func (t *Turtle) Turn(dx, dy int) {
	if !t.laid || !cardinal(t.DX, t.DY) || !cardinal(dx, dy) || t.DX*dx+t.DY*dy != 0 {
		t.Road.Scene.fail("turtle turn needs a previous square and perpendicular cardinal heading")
		return
	}
	lx, ly := t.Next.X-float64(t.DX), t.Next.Y-float64(t.DY)
	var cx, cy int
	var p Point
	if t.DX != 0 {
		cx, cy = -t.DX, dy
		p = Point{lx - float64(cx)*1.5, ly + float64(cy)/2}
		t.Next = Point{p.X - float64(cx)/2, p.Y + float64(cy)*1.5}
	} else {
		cx, cy = dx, -t.DY
		p = Point{lx + float64(cx)/2, ly - float64(cy)*1.5}
		t.Next = Point{p.X + float64(cx)*1.5, p.Y - float64(cy)/2}
	}
	t.Road.Curve(p, cx, cy)
	t.DX, t.DY = dx, dy
	t.laid = false
}
