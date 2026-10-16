package compose

import "math"

// RNG is the workshop generators' xorshift64 sequence. It is reproducible,
// not cryptographic. A zero-value RNG initializes to state 1.
type RNG struct{ state uint64 }

func NewRNG(seed uint64) *RNG { return &RNG{state: seed*2654435761 + 1} }
func (r *RNG) Float64() float64 {
	if r.state == 0 {
		r.state = 1
	}
	r.state ^= r.state << 13
	r.state ^= r.state >> 7
	r.state ^= r.state << 17
	return float64(r.state>>11) / float64(uint64(1)<<53)
}
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		panic("compose RNG: nonpositive bound")
	}
	return int(r.Float64()*float64(n)) % n
}
func (r *RNG) Pick(values ...int) int        { return values[r.Intn(len(values))] }
func (r *RNG) Jitter(radius float64) float64 { return (r.Float64() - .5) * 2 * radius }

func noiseHash(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*1442695041
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h) / float64(math.MaxUint32)
}

// Noise is smooth value noise on the integer lattice, in [0,1].
func Noise(x, y float64, seed int) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	fx = fx * fx * (3 - 2*fx)
	fy = fy * fy * (3 - 2*fy)
	a, b := noiseHash(int(x0), int(y0), seed), noiseHash(int(x0)+1, int(y0), seed)
	c, d := noiseHash(int(x0), int(y0)+1, seed), noiseHash(int(x0)+1, int(y0)+1, seed)
	return (a*(1-fx)+b*fx)*(1-fy) + (c*(1-fx)+d*fx)*fy
}

// FBM is the generators' four-octave value noise: frequency doubles, amplitude
// halves, and each octave offsets the seed by 101. No global RNG is consumed.
func FBM(x, y float64, seed int) float64 {
	v, amp, f, total := 0.0, 1.0, 1.0, 0.0
	for o := 0; o < 4; o++ {
		v += amp * Noise(x*f, y*f, seed+o*101)
		total += amp
		amp *= .5
		f *= 2
	}
	return v / total
}
