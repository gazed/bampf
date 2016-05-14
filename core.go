// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

// Energy core related code is grouped here.

import (
	"math/rand"
	"time"

	"github.com/gazed/vu"
)

// coreControl tracks available core drop locations and regulates how fast
// new cores appear.
type coreControl struct {
	cores   []vu.Pov      // cores available to be collected.
	tiles   []gridSpot    // core drop locations.
	saved   []gridSpot    // remember the core drop locations for resets.
	last    time.Time     // last time a core was dropped.
	holdoff time.Duration // time delay between core drops.
	units   float64       // eng.Units injected on creation is...
	spot    *gridSpot     // ...used to translate between grid and game coordinates.
	ani     *animator     // Handles short animations.
}

// newCoreControl returns an initialized coreControl structure.
func newCoreControl(units int, ani *animator) *coreControl {
	cc := &coreControl{}
	cc.ani = ani
	cc.units = float64(units)
	cc.cores = []vu.Pov{}
	cc.saved = []gridSpot{}
	cc.tiles = []gridSpot{}
	cc.spot = &gridSpot{}
	cc.holdoff, _ = time.ParseDuration("200ms")
	return cc
}

// timeToDrop regulates how fast the new cores appear.
func (cc *coreControl) timeToDrop() bool {
	if time.Now().After(cc.last.Add(cc.holdoff)) {
		cc.last = time.Now()
		return true
	}
	return false
}

// canDrop is called to determine if a new core could/should be dropped.
// Cores are dropped if there is not enough dropped cores to get the player
// to the next level (coresNeeded) and if there are available drop locations.
func (cc *coreControl) canDrop(coresNeeded int) bool {
	return len(cc.cores) < coresNeeded && len(cc.tiles) > 0
}

// dropSpot picks a random free core drop location. Return the potential
// gridx, gridy drop location
func (cc *coreControl) dropSpot() (gridx, gridy int) {
	index := rand.Intn(len(cc.tiles))
	spot := cc.tiles[index]
	return spot.x, spot.y
}

// dropCore creates a new core. Create it high so that it drops.
// Return the x, z game location of the dropped core.
func (cc *coreControl) dropCore(root vu.Pov, fade float64, gridx, gridy int) (gamex, gamez float64) {

	// remove the dropped spot from the list of available spots.
	removed := false // sanity check.
	for index, xy := range cc.tiles {
		if gridx == xy.x && gridy == xy.y {
			cc.tiles = append(cc.tiles[:index], cc.tiles[index+1:]...)
			removed = true
			break
		}
	}
	if !removed {
		logf("core.dropCore: failed to locate what should be a valid drop location")
		return 0, 0
	}
	core := cc.createCore(root, fade)

	// add the core to the list of dropped cores.
	cc.cores = append(cc.cores, core)
	gamex, gamez = toGame(gridx, gridy, cc.units)
	core.SetLocation(gamex, 10, gamez) // start high and animate drop to floor level.
	cc.ani.addAnimation(&coreDropAnimation{core: core})
	return gamex, gamez
}

// remCore destroys the indicated core. The drop spot is now available for new
// cores. Return the game location of the removed core.
func (cc *coreControl) remCore(index int) (gamex, gamez float64) {
	core := cc.cores[index]
	cc.cores = append(cc.cores[:index], cc.cores[index+1:]...)

	// remove the core from the display and minimap.
	core.Dispose(vu.PovNode)
	gamex, _, gamez = core.Location()
	gridx, gridy := toGrid(gamex, 0, gamez, cc.units)

	// make the tile available for a new drop. Use the old core location.
	cc.tiles = append(cc.tiles, gridSpot{gridx, gridy})
	return gamex, gamez
}

// hitCore returns the core index if the given location is in the same grid location
// as a core. Return -1 if no core was hit.
func (cc *coreControl) hitCore(gamex, gamez float64) (coreIndex int) {
	coreIndex = -1
	gridx, gridy := toGrid(gamex, 0, gamez, cc.units)
	for index, core := range cc.cores {
		x, y, z := core.Location()
		corex, corey := toGrid(x, y, z, cc.units)
		if gridx == corex && gridy == corey {
			coreIndex = index
			break
		}
	}
	return coreIndex
}

// addDropLocation adds a spot where cores are allowed to be dropped.
// The coordinates are specified in grid coordinates.
func (cc *coreControl) addDropLocation(gridx, gridy int) {
	cc.saved = append(cc.saved, gridSpot{gridx, gridy})
	cc.tiles = append(cc.tiles, gridSpot{gridx, gridy})
}

// reset puts the core control back to the initial conditions before cores
// starting dropping. Expected to be called for cleaning up the current
// level before transitioning to a new level.
func (cc *coreControl) reset() {
	for _, core := range cc.cores {
		core.Dispose(vu.PovNode)
	}
	cc.cores = []vu.Pov{}
	cc.tiles = []gridSpot{}
	for _, spot := range cc.saved {
		cc.tiles = append(cc.tiles, gridSpot{spot.x, spot.y})
	}
}

// createCore makes the new core model.
// Create a core image using a single multi-texture shader.
func (cc *coreControl) createCore(root vu.Pov, fade float64) vu.Pov {
	core := root.NewPov().SetScale(0.25, 0.25, 0.25)
	model := core.NewModel("spinball").LoadMesh("billboard")
	model.AddTex("ele").AddTex("ele").AddTex("halo").AddTex("halo")
	model.SetAlpha(0.6)
	model.SetUniform("fd", fade)
	return core
}

// coreControl
// ===========================================================================
// gridSpot is used by coreControl and sentinel.

// gridSpot is used to track grid locations. It can be used to store grid
// locations and to convert back and forth between grid and game locations.
type gridSpot struct{ x, y int }

// toGame takes a grid location and translates into a game location.
// Game locations are where models of cores, walls, and tiles are placed.
func toGame(gridx, gridy int, units float64) (gamex, gamez float64) {
	return float64(gridx) * units, float64(-gridy) * units
}

// toGrid takes the current game location and translates into a grid location.
// Grid locations are where cores are dropped or fetched.
func toGrid(gamex, gamey, gamez, units float64) (gridx, gridy int) {
	inv := 1.0 / units
	adj := units * 0.5
	xadj := adj
	if gamex < 0 {
		xadj = -xadj
	}
	yadj := adj
	if gamez > 0 {
		yadj = -yadj
	}
	return int((gamex + xadj) * inv), int((-gamez + yadj) * inv)
}

// ===========================================================================
// coreDropAnimation

// coreDropAnimation shows cores falling when they are first created.
type coreDropAnimation struct {
	core    vu.Pov  // core to animate.
	x, y, z float64 // core location.
	drop    float64 // the amount to fall each tick.
	rest    float64 // final resting location.
	ticks   int     // how many game ticks to animate.
	state   int
}

// Animate implements animation. Drop the core.
func (ca *coreDropAnimation) Animate(dt float64) bool {
	switch ca.state {
	case 0:
		ca.ticks = 50                         // total animation time.
		ca.rest = 0.25                        // final core height.
		ca.x, ca.y, ca.z = ca.core.Location() // initial location.
		ca.drop = (ca.rest - ca.y) / float64(ca.ticks)
		ca.state = 1
		return true
	case 1:
		if ca.ticks > 0 {
			ca.y += ca.drop
			ca.core.SetLocation(ca.x, ca.y, ca.z)
			ca.ticks--
			return true // animation not done.
		}
		ca.Wrap()
		return false // animation done.
	default:
		return false // animation done.
	}
}

// Wrap finishes the core drop by ensuring the core is at its
// final location.
func (ca *coreDropAnimation) Wrap() {
	ca.core.SetLocation(ca.x, ca.rest, ca.z)
	ca.state = 2
}
