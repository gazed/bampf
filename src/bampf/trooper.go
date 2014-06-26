// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"sort"
	"vu"
	"vu/audio"
	"vu/math/lin"
)

// trooper is a cube that represents the players health and
// progress for a level. Each new level increases the size of the cube.
// Trooper is an attempt to keep polygon growth linear while the player statistics
// grows exponentially. This is done by rendering groups of cells as a single
// cube when possible.
//
// trooper works with single cubes (cells) of size 2 centered at the origin.
type trooper struct {
	part                  vu.Part   // Graphics container.
	lvl                   int       // Current game level of trooper.
	eng                   vu.Engine // Games engine.
	neo                   vu.Part   // Un-injured trooper
	bits                  []box     // Injured troopers have panels and edge cubes.
	ipos                  []int     // Remember the initial positions for resets.
	center                vu.Part   // Center always represented as one piece
	mid                   int       // Level entry number of cells.
	cloaked               bool      // Is cloaking turned on.
	cloakEnergy, cemax    int       // Energy available for cloaking.
	teleportEnergy, temax int       // Energy available for teleporting.

	// monitors and sounds.
	hms    map[string]healthMonitor    // Health event monitors.
	ems    map[string]energyMonitor    // Energy event monitors.
	noises map[string]audio.SoundMaker // Various sounds.
}

// newTrooper creates a trooper for the given level.
//    level 0: 1x1x1 :  0 edge cubes 0 panels, (only 1 cube)
//    level 1: 2x2x2 :  8 edge cubes + 6 panels of 0x0 cubes + 0x0x0 center.
//    level 2: 3x3x3 : 20 edge cubes + 6 panels of 1x1 cubes + 1x1x1 center.
//    level 3: 4x4x4 : 32 edge cubes + 6 panels of 2x2 cubes + 2x2x2 center.
//    ...
func newTrooper(eng vu.Engine, part vu.Part, level int) *trooper {
	tr := &trooper{}
	tr.lvl = level
	tr.eng = eng
	tr.part = part
	tr.bits = []box{}
	tr.ipos = []int{}
	tr.mid = tr.lvl*tr.lvl*tr.lvl*8 - (tr.lvl-1)*(tr.lvl-1)*(tr.lvl-1)*8
	tr.noises = make(map[string]audio.SoundMaker)

	// set max energies.
	tr.cemax, tr.temax = 1000, 1000

	// special case for a level 0 (start screen) trooper.
	if tr.lvl == 0 {
		cube := newCube(tr.part, 0, 0, 0, 1)
		cube.edgeSort(1)
		tr.bits = append(tr.bits, cube)
		return tr
	}

	// create the panels. These are used in each level after level 1.
	cubeSize := 1.0 / float64(tr.lvl+1)
	centerOffset := cubeSize * 0.5
	panelCenter := float64(tr.lvl) * centerOffset
	tr.bits = append(tr.bits, newPanel(eng, tr.part, panelCenter, 0.0, 0.0, tr.lvl))
	tr.bits = append(tr.bits, newPanel(eng, tr.part, -panelCenter, 0.0, 0.0, tr.lvl))
	tr.bits = append(tr.bits, newPanel(eng, tr.part, 0.0, panelCenter, 0.0, tr.lvl))
	tr.bits = append(tr.bits, newPanel(eng, tr.part, 0.0, -panelCenter, 0.0, tr.lvl))
	tr.bits = append(tr.bits, newPanel(eng, tr.part, 0.0, 0.0, panelCenter, tr.lvl))
	tr.bits = append(tr.bits, newPanel(eng, tr.part, 0.0, 0.0, -panelCenter, tr.lvl))

	// troopers are made out of cubes and panels.
	mx := float64(-tr.lvl)
	for cx := 0; cx <= tr.lvl; cx++ {
		my := float64(-tr.lvl)
		for cy := 0; cy <= tr.lvl; cy++ {
			mz := float64(-tr.lvl)
			for cz := 0; cz <= tr.lvl; cz++ {

				// create the outer edges.
				newCells := 0
				if (cx == 0 || cx == tr.lvl) && (cy == 0 || cy == tr.lvl) && (cz == 0 || cz == tr.lvl) {

					// corner cube
					newCells = 1
				} else if (cx == 0 || cx == tr.lvl) && (cy == 0 || cy == tr.lvl) ||
					(cx == 0 || cx == tr.lvl) && (cz == 0 || cz == tr.lvl) ||
					(cy == 0 || cy == tr.lvl) && (cz == 0 || cz == tr.lvl) {

					// edge cube
					newCells = 2
				} else if cx == 0 || cx == tr.lvl || cy == 0 || cy == tr.lvl || cz == 0 || cz == tr.lvl {

					// side cubes are added to a panel.
					x, y, z := mx*centerOffset, my*centerOffset, mz*centerOffset
					if cx == tr.lvl && x > y && x > z {
						tr.bits[0].(*panel).addCube(x, y, z, float64(cubeSize))
					} else if cx == 0 && x < y && x < z {
						tr.bits[1].(*panel).addCube(x, y, z, float64(cubeSize))
					} else if cy == tr.lvl && y > x && y > z {
						tr.bits[2].(*panel).addCube(x, y, z, float64(cubeSize))
					} else if cy == 0 && y < x && y < z {
						tr.bits[3].(*panel).addCube(x, y, z, float64(cubeSize))
					} else if cz == tr.lvl && z > x && z > y {
						tr.bits[4].(*panel).addCube(x, y, z, float64(cubeSize))
					} else if cz == 0 && z < x && z < y {
						tr.bits[5].(*panel).addCube(x, y, z, float64(cubeSize))
					}
				}
				if newCells > 0 {
					x, y, z := mx*centerOffset, my*centerOffset, mz*centerOffset
					cube := newCube(tr.part, x, y, z, float64(cubeSize))
					cube.edgeSort(newCells)
					tr.bits = append(tr.bits, cube)
				}
				mz += 2
			}
			my += 2
		}
		mx += 2
	}
	tr.addCenter()

	// its easier to remember the initial positions than recalculate them.
	tr.ipos = make([]int, len(tr.bits))
	for cnt, b := range tr.bits {
		tr.ipos[cnt] = b.box().ccnt
	}
	return tr
}

// fullHealth returns true if the player is at full health.
func (tr *trooper) fullHealth() bool { return tr.neo != nil }

// setScale changes the troopers size.
func (tr *trooper) setScale(scale float64) { tr.part.SetScale(scale, scale, scale) }

// loc gets the troopers current location.
func (tr *trooper) loc() (x, y, z float64) { return tr.part.Location() }
func (tr *trooper) setLoc(x, y, z float64) { tr.part.SetLocation(x, y, z) }

// addCenter creates the interior center of the trooper which is a single cube
// the size of the previous level. This will be nothing on the first level.
func (tr *trooper) addCenter() {
	if tr.lvl > 0 {
		cubeSize := 1.0 / float64(tr.lvl+1)
		scale := float64(tr.lvl-1) * cubeSize * 0.45 // leave a gap.
		tr.center = tr.part.AddPart().SetScale(scale, scale, scale)
		tr.center.SetRole("flata").SetMesh("cube").SetMaterial("tred")
		tr.center.Role().SetUniform("fd", 1000)
	}
}

// health returns the current cell count, the mid-point cell count
// (the starting number of cells for the level), and the maximum
// possible cell count for this level.
func (tr *trooper) health() (health, mid, max int) {
	for _, b := range tr.bits {
		health += b.box().ccnt
	}
	l0, l1, l2 := (tr.lvl-1)*2, tr.lvl*2, (tr.lvl+1)*2
	min, mid, max := l0*l0*l0, l1*l1*l1, l2*l2*l2
	return health, mid - min, max - min
}

// reset the troopers health to the level's minimum.
func (tr *trooper) reset() {
	tr.trash()
	tr.addCenter()
	for cnt, b := range tr.bits {
		b.reset(tr.ipos[cnt])
	}
	tr.healthChanged(tr.health())
}

// attach currently tries to attach new cells to the panels first.
// Otherwise add to an edge.
func (tr *trooper) attach() {
	for _, b := range tr.bits {
		if b.attach() {
			health, mid, max := tr.health()
			if health == max && tr.neo == nil {
				tr.merge()
			}
			tr.healthChanged(health, mid, max)
			return
		}
	}
}

// detach currently tries to remove cells from edges first.
// Otherwise remove from a panel.
func (tr *trooper) detach() {
	if tr.neo != nil {
		tr.demerge()
		tr.healthChanged(tr.health())
		return
	}
	for _, b := range tr.bits {
		if b.detach() {
			tr.healthChanged(tr.health())
			return
		}
	}
}

// detachCores removes the indicated number of cells.
func (tr *trooper) detachCores(loss int) {
	if loss <= 0 {
		return
	}
	h, _, _ := tr.health()
	if loss > h {
		loss = h
	}
	for cnt := loss; cnt > 0; cnt-- {
		if tr.neo != nil {
			tr.demerge()
			continue
		}
		for _, b := range tr.bits {
			if b.detach() {
				break
			}
		}
	}
	tr.healthChanged(tr.health())
}

// merge collapses all the troopers cubes into a single cube with an
// optional center cube.  Called when the trooper reaches full health.
func (tr *trooper) merge() {
	tr.trash()
	tr.neo = tr.part.AddPart().SetScale(0.5, 0.5, 0.5)
	tr.neo.SetRole("flata").SetMesh("cube").SetMaterial("tblue")
	tr.neo.Role().SetUniform("fd", 1000)
	tr.addCenter()
}

// demerge breaks the troopers single cube into smaller blocks. Expected to
// be called when a trooper at full health loses health.
func (tr *trooper) demerge() {
	tr.trash()
	tr.addCenter()
	for _, b := range tr.bits {
		b.reset(b.box().cmax)
	}
	tr.bits[0].detach()
}

// trash destroys all the troopers cells.
func (tr *trooper) trash() {
	for _, b := range tr.bits {
		b.trash()
	}
	if tr.center != nil {
		tr.part.RemPart(tr.center)
		tr.center = nil
	}
	tr.part.RemPart(tr.neo)
	tr.neo = nil
}

// addCloakEnergy is called to increase the amount of cloaking energy.
func (tr *trooper) addCloakEnergy() {
	tr.cloakEnergy += 100
	if tr.cloakEnergy > tr.cemax {
		tr.cloakEnergy = tr.cemax
	}
	tr.energyChanged()
}

// cloak toggles the players cloak ability. Cloaking is only enabled if
// there is sufficient energy.
func (tr *trooper) cloak(useCloak bool) {
	if useCloak && tr.cloakEnergy > 0 {
		tr.cloaked = true
		tr.eng.PlaceSoundListener(tr.loc())
		noise := tr.noises["cloak"]
		noise.SetLocation(tr.loc())
		noise.Play()
	} else if !useCloak {
		tr.cloaked = false
		tr.eng.PlaceSoundListener(tr.loc())
		noise := tr.noises["decloak"]
		noise.SetLocation(tr.loc())
		noise.Play()
	}
}

// teleport uses all of the teleport energy in one shot. Teleport only
// works if the full amount of teleport energy is available.
func (tr *trooper) teleport() bool {
	if tr.teleportEnergy >= tr.temax {
		tr.eng.PlaceSoundListener(tr.loc())
		teleportNoise := tr.noises["teleport"]
		teleportNoise.SetLocation(tr.loc())
		teleportNoise.Play()
		tr.teleportEnergy = 0
		tr.energyChanged()
		return true
	}
	return false
}

// energy returns the amount of energy available for cloaking and teleporting.
func (tr *trooper) energy() (teng, tmax, ceng, cmax int) {
	ce := tr.cloakEnergy
	if ce > tr.cemax { // can only happens with debugging hooks.
		ce = tr.cemax
	}
	return tr.teleportEnergy, tr.temax, ce, tr.cemax
}

// updateEnergy is called on a regular basis to refresh the players available
// teleport and cloaking energy.
func (tr *trooper) updateEnergy() {
	change := false

	// teleport energy increases to max.
	if tr.teleportEnergy < tr.temax {
		tr.teleportEnergy += 1
		change = true
	}

	// cloak energy is used until gone.
	if tr.cloaked {
		change = true
		tr.cloakEnergy -= 4
		if tr.cloakEnergy <= 0 {
			tr.cloakEnergy = 0
			tr.cloak(false)
		}
	}
	if change {
		tr.energyChanged()
	}
}

// resetEnergy is called at the start of a level.
func (tr *trooper) resetEnergy() {
	tr.teleportEnergy = tr.temax
	tr.cloakEnergy = 1000
}

// trooper
// ===========================================================================
// box & cbox

// box defines common cell behaviours.
type box interface {
	attach() bool
	detach() bool
	trash()
	merge()
	reset(count int)
	box() *cbox
}

// cbox is a base class for panels and cubes.
type cbox struct {
	ccnt, cmax     int     // Number of cells.
	cx, cy, cz     float64 // Center of the box.
	csize          float64 // Cell size where each side is the same dimension.
	trashc, mergec func()  // Set by super class.
	addc, remc     func()  // Set by super class.
}

// attach adds a cell to the cube, merging the cube when the cube is full.
// Attach returns true if a cell was added. A return of false indicates a
// full cube.
func (c *cbox) attach() bool {
	if c.ccnt >= 0 && c.ccnt < c.cmax {
		c.ccnt++ // only spot where this is incremented.
		if c.ccnt == c.cmax {
			c.mergec() // c.merge()
		} else {
			c.addc() // c.addCell()
		}
		return true
	}
	return false
}

// detach removes a cell from the cube, demerging a full cube if necessary.
// Detach returns true if a cell was detached. A return of false indicates
// an empty cube.
func (c *cbox) detach() bool {
	if c.ccnt > 0 && c.ccnt <= c.cmax {
		if c.ccnt == c.cmax {
			c.reset(c.cmax - 1)
		} else {
			c.remc() // c.removeCell()
			c.ccnt-- // only spot where this is decremented.
		}
		return true
	}
	return false
}

// reset clears the cbox and ensures the cell count is the given value.
func (c *cbox) reset(cellCount int) {
	c.trashc()
	c.ccnt = 0 // only spot where this is reset to 0
	if cellCount > c.cmax {
		cellCount = c.cmax
	}
	for cnt := 0; cnt < cellCount; cnt++ {
		c.attach()
	}
}

// box allows direct access to the cbox from a super class.
func (c *cbox) box() *cbox { return c }

// box & cbox
// ===========================================================================
// panel

// panel groups 0 or more cubes into the center of one of the troopers
// six sides.
type panel struct {
	eng   vu.Engine // Needed to create new cells.
	part  vu.Part   // Each panel needs its own part.
	lvl   int       // Used to scale slab.
	slab  vu.Part   // Un-injured panel is a single piece.
	cubes []*cube   // An injured panel is made of cubes.
	cbox
}

// newPanel creates a panel with no cubes. The cubes are added later using
// panel.addCube().
func newPanel(eng vu.Engine, part vu.Part, x, y, z float64, level int) *panel {
	p := &panel{}
	p.eng = eng
	p.part = part.AddPart()
	p.lvl = level
	p.cubes = []*cube{}
	p.cx, p.cy, p.cz = x, y, z
	p.ccnt, p.cmax = 0, (level-1)*(level-1)*8
	p.mergec = func() { p.merge() }
	p.trashc = func() { p.trash() }
	p.addc = func() { p.addCell() }
	p.remc = func() { p.removeCell() }
	return p
}

// addCube is only used at the begining to add cubes that are owned by this
// panel.
func (p *panel) addCube(x, y, z, cubeSize float64) {
	p.csize = cubeSize
	c := newCube(p.part, x, y, z, p.csize)
	if (p.cx > p.cy && p.cx > p.cz) || (p.cx < p.cy && p.cx < p.cz) {
		c.panelSort(1, 0, 0, 4)
	} else if (p.cy > p.cx && p.cy > p.cz) || (p.cy < p.cx && p.cy < p.cz) {
		c.panelSort(0, 1, 0, 4)
	} else if (p.cz > p.cx && p.cz > p.cy) || (p.cz < p.cx && p.cz < p.cy) {
		c.panelSort(0, 0, 1, 4)
	}
	if c != nil {
		p.ccnt += 4
		p.cubes = append(p.cubes, c)
	}
}

// addCell adds cells so that the new cells are spread amongst the panels cubes.
func (p *panel) addCell() {
	for addeven := 0; addeven < p.cubes[0].cmax; addeven++ {
		for _, c := range p.cubes {
			if c.ccnt <= addeven {
				c.attach()
				return
			}
		}
	}
	log.Printf("pc:panel addCell should never reach here. %d %d", p.ccnt, p.cmax)
}

// removeCell takes a piece out of a panel.
func (p *panel) removeCell() {
	for _, c := range p.cubes {
		if c.detach() {
			return
		}
	}
	log.Printf("pc:panel removeCell should never reach here.")
}

// merge turns all the cubes into a single panel.
func (p *panel) merge() {
	p.trash()
	size := p.csize * 0.5
	p.slab = p.part.AddPart().SetLocation(p.cx, p.cy, p.cz)
	scale := float64(p.lvl-1) * size
	if (p.cx > p.cy && p.cx > p.cz) || (p.cx < p.cy && p.cx < p.cz) {
		p.slab.SetScale(size, scale, scale)
	} else if (p.cy > p.cx && p.cy > p.cz) || (p.cy < p.cx && p.cy < p.cz) {
		p.slab.SetScale(scale, size, scale)
	} else if (p.cz > p.cx && p.cz > p.cy) || (p.cz < p.cx && p.cz < p.cy) {
		p.slab.SetScale(scale, scale, size)
	}
	p.slab.SetRole("flata").SetMesh("cube").SetMaterial("tblue")
	p.slab.Role().SetUniform("fd", 1000)
}

// trash clears any visible parts from the panel. It is up to calling methods
// to ensure the cell count is correct.
func (p *panel) trash() {
	if p.slab != nil {
		p.part.RemPart(p.slab)
		p.slab = nil
	}
	for _, cube := range p.cubes {
		cube.reset(0)
	}
}

// panel
// ===========================================================================
// cube

// cube is the building block for troopers and panels. Cube takes a size
// and location and creates an 8 part cube out of it. Cubes can be queried
// as to their current number of cells which is between 0 (nothing visible),
// 1-7 (partial) and 8 (merged).
type cube struct {
	eng     vu.Engine // Needed to create new cells.
	part    vu.Part   // For the merged cube.
	cells   []vu.Part // Max 8 cells per cube.
	centers csort     // Precalculated center location of each cell.
	cbox
}

// newCube's are often started with cube size of 1 corner, 2 edges,
// or 4 bottom side pieces.
func newCube(part vu.Part, x, y, z, cubeSize float64) *cube {
	c := &cube{}
	c.part = part.AddPart()
	c.cells = []vu.Part{}
	c.cx, c.cy, c.cz, c.csize = x, y, z, cubeSize
	c.ccnt, c.cmax = 0, 8
	c.mergec = func() { c.merge() }
	c.trashc = func() { c.trash() }
	c.addc = func() { c.addCell() }
	c.remc = func() { c.removeCell() }

	// calculate the cell center locations (unsorted)
	qs := c.csize * 0.25
	c.centers = csort{
		&lin.V3{x - qs, y - qs, z - qs},
		&lin.V3{x - qs, y - qs, z + qs},
		&lin.V3{x - qs, y + qs, z - qs},
		&lin.V3{x - qs, y + qs, z + qs},
		&lin.V3{x + qs, y - qs, z - qs},
		&lin.V3{x + qs, y - qs, z + qs},
		&lin.V3{x + qs, y + qs, z - qs},
		&lin.V3{x + qs, y + qs, z + qs},
	}
	return c
}

// edgeSort arranges the edge pieces so that cubes are added or removed in cube
// like looking pieces.
func (c *cube) edgeSort(startCount int) {
	sort.Sort(c.centers)
	c.reset(startCount)
}

// panelSort sorts cubes based on which panel they are in. Needed for orderly
// addition/removal of cubes.
func (c *cube) panelSort(rx, ry, rz float64, startCount int) {
	sorter := &ssort{c.centers, rx, ry, rz}
	sort.Sort(sorter)
	c.reset(startCount)
}

// addCell creates and adds a new cell to the cube.
func (c *cube) addCell() {
	center := c.centers[c.ccnt-1]
	cell := c.part.AddPart().SetLocation(center.X, center.Y, center.Z)
	scale := c.csize * 0.20 // leave a gap (0.25 for no gap).
	cell.SetScale(scale, scale, scale)
	cell.SetRole("flata").SetMesh("cube").SetMaterial("tgreen")
	cell.Role().SetUniform("fd", 1000)
	c.cells = append(c.cells, cell)
}

// removeCell removes the last cell from the list of cube cells.
func (c *cube) removeCell() {
	last := len(c.cells)
	c.part.RemPart(c.cells[last-1])
	c.cells[last-1] = nil
	c.cells = c.cells[:last-1]
}

// merge removes all cells and replaces them with a single cube. Expected
// to only be called by attach. The c.ccnt should be c.cmax before and after
// merge is called.
func (c *cube) merge() {
	c.trash()
	cell := c.part.AddPart().SetLocation(c.cx, c.cy, c.cz)
	cell.SetRole("flata").SetMesh("cube").SetMaterial("tgreen")
	cell.Role().SetUniform("fd", 1000)
	scale := (c.csize - (c.csize * 0.15)) * 0.5 // leave a gap (just c.csize for no gap)
	cell.SetScale(scale, scale, scale)
	c.cells = append(c.cells, cell)
}

// removes all visible cube parts.
func (c *cube) trash() {
	for len(c.cells) > 0 {
		c.removeCell()
	}
}

// cube
// ===========================================================================
// csort

// csort is used to sort the cube quadrants so that the quadrants closest
// to the origin are first in the list. This way the cells added first and
// removed last are those closest to the center.
//
// A reference point is necessary since the origin gets too far away for
// a flat panel to orient the quads properly.
type csort []*lin.V3 // list of quadrant centers.

func (c csort) Len() int               { return len(c) }
func (c csort) Swap(i, j int)          { c[i], c[j] = c[j], c[i] }
func (c csort) Less(i, j int) bool     { return c.Dtoc(c[i]) < c.Dtoc(c[j]) }
func (c csort) Dtoc(v *lin.V3) float64 { return v.X*v.X + v.Y*v.Y + v.Z*v.Z }

// ssort is used to sort the panel cube quadrants so that the quadrants
// to the inside origin plane are first in the list. A reference normal is
// necessary since the panels get large enough that the points on the
// "outside" get picked up due to the angle.
type ssort struct {
	c       []*lin.V3 // list of quadrant centers.
	x, y, z float64   // reference plane.
}

func (s ssort) Len() int           { return len(s.c) }
func (s ssort) Swap(i, j int)      { s.c[i], s.c[j] = s.c[j], s.c[i] }
func (s ssort) Less(i, j int) bool { return s.Dtoc(s.c[i]) < s.Dtoc(s.c[j]) }
func (s ssort) Dtoc(v *lin.V3) float64 {
	normal := &lin.V3{s.x, s.y, s.z}
	dot := v.Dot(normal)
	dx := normal.X * dot
	dy := normal.Y * dot
	dz := normal.Z * dot
	return dx*dx + dy*dy + dz*dz
}

// csort
// ===========================================================================
// healthMonitor

// healthMonitor is used to monitor troopers cell count changes.
type healthMonitor interface {
	healthUpdated(health, high, warn int) // called when cells are added or lost.
}

// monitorHealth adds a monitor for trooper health changes.
func (tr *trooper) monitorHealth(id string, mon healthMonitor) {
	if tr.hms == nil {
		tr.hms = make(map[string]healthMonitor)
	}
	tr.hms[id] = mon
}

// ignoreHealth removes a monitor.
func (tr *trooper) ignoreHealth(id string) {
	if tr.hms != nil {
		delete(tr.hms, id)
	}
}

// healthChanged is called to notify all monitors.
func (tr *trooper) healthChanged(health, mid, max int) {
	if tr.hms != nil {
		for _, monitor := range tr.hms {
			monitor.healthUpdated(health, mid, max)
		}
	}
}

// healthMonitor
// ===========================================================================
// energyMontior

// energyMonitor is used to monitor the troopers energy amount changes.
type energyMonitor interface {
	energyUpdated(teleportEnergy, tmax, cloakEnergy, cmax int) // called when cells are added or lost.
}

// monitorEnergy adds a monitor for trooper energy changes.
func (tr *trooper) monitorEnergy(id string, mon energyMonitor) {
	if tr.ems == nil {
		tr.ems = make(map[string]energyMonitor)
	}
	tr.ems[id] = mon
}

// ignoreEnergy removes a monitor.
func (tr *trooper) ignoreEnergy(id string) {
	if tr.ems != nil {
		delete(tr.ems, id)
	}
}

// energyChanged is called to notify all monitors.
func (tr *trooper) energyChanged() {
	if tr.ems != nil {
		for _, monitor := range tr.ems {
			monitor.energyUpdated(tr.energy())
		}
	}
}
