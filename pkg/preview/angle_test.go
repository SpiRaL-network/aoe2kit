package preview

import "testing"

func TestAngleFrameOrdinal(t *testing.T) {
	cases := []struct {
		rotation, angles, frames, want int
	}{
		{4, 42, 1, 4},     // static art: rotation is the frame (oak)
		{6, 3, 1, 6},      // out-of-range static stays out of range, so it is reported
		{0, 16, 60, 0},    // animated: first frame of the first facing
		{7, 16, 60, 420},  // first frame of facing 7
		{15, 16, 60, 900}, // last facing
		{16, 16, 60, -1},  // past the last facing: reported, not wrapped
		{3, 1, 10, 3},     // single-angle animation keeps the rotation
	}
	for _, c := range cases {
		if got := angleFrameOrdinal(c.rotation, c.angles, c.frames); got != c.want {
			t.Errorf("angleFrameOrdinal(%d, %d, %d) = %d, want %d", c.rotation, c.angles, c.frames, got, c.want)
		}
	}
}
