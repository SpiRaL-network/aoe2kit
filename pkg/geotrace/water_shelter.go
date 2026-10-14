package geotrace

// classifyShelteredWater is a gameplay habitat proxy: land in at least four of
// eight directions within 32 tiles, including an opposing pair of banks.
// it does not claim measured salinity, depth, or wave exposure.
func classifyShelteredWater(r *Report) {
	n := r.GridSize
	if len(r.Cells) != n*n {
		return
	}
	for i := range r.Cells {
		c := &r.Cells[i]
		c.ShelteredWater = false
		if c.Land == nil || *c.Land || c.InlandWater {
			continue
		}
		hits := 0
		banks := [8]bool{}
		for i, d := range [][2]int{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}} {
			for step := 1; step <= 32; step++ {
				if step*step*(d[0]*d[0]+d[1]*d[1]) > 32*32 {
					break
				}
				x, y := c.X+d[0]*step, c.Y+d[1]*step
				if x < 0 || x >= n || y < 0 || y >= n {
					break
				}
				land := r.Cells[y*n+x].Land
				if land != nil && *land {
					hits++
					banks[i] = true
					break
				}
			}
		}
		opposed := false
		for i := 0; i < 4; i++ {
			opposed = opposed || (banks[i] && banks[i+4])
		}
		c.ShelteredWater = hits >= 4 && opposed
	}
}
