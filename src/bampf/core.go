// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

// Energy core related code is grouped here.

import (
	"log"
	"math/rand"
	"time"
	"vu"
)

// coreControl tracks available core drop locations and regulates how fast
// new cores appear.
type coreControl struct {
	cores   []vu.Part     // cores available to be collected.
	tiles   []gridSpot    // core drop locations.
	saved   []gridSpot    // remember the core drop locations for resets.
	last    time.Time     // last time a core was dropped.
	holdoff time.Duration // time delay between core drops.
	units   float64       // eng.Units injected on creation is...
	spot    *gridSpot     // ...used to translate between grid and game coordinates.
}

// newCoreControl returns an initialized coreControl structure.
func newCoreControl(units int) *coreControl {
	cc := &coreControl{}
	cc.units = float64(units)
	cc.cores = []vu.Part{}
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
	random := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	index := random.Intn(len(cc.tiles))
	spot := cc.tiles[index]
	return spot.x, spot.y
}

// dropCore creates a new core. Create it high so that it drops.
// Return the x, z game location of the dropped core.
func (cc *coreControl) dropCore(sc vu.Scene, fade float64, gridx, gridy int) (gamex, gamez float64) {

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
		log.Printf("core.dropCore: failed to locate what should be a valid drop location")
		return
	}
	core := cc.createCore(sc, fade)

	// add the core to the list of dropped cores.
	cc.cores = append(cc.cores, core)
	gamex, gamez = cc.spot.toGame(gridx, gridy, cc.units)
	core.SetLocation(gamex, 10, gamez)
	return
}

// remCore destroys the indicated core. The drop spot is now available for new
// cores. Return the game location of the removed core.
func (cc *coreControl) remCore(sc vu.Scene, index int) (gamex, gamez float64) {
	core := cc.cores[index]
	cc.cores = append(cc.cores[:index], cc.cores[index+1:]...)

	// remove the core from the display and minimap.
	core.Dispose()
	sc.RemPart(core)
	gamex, _, gamez = core.Location()
	x, y := cc.spot.toGrid(gamex, 0, gamez, cc.units)

	// make the tile available for a new drop. Use the old core location.
	cc.tiles = append(cc.tiles, gridSpot{x, y})
	return gamex, gamez
}

// hitCore returns the core index if the given location is in the same grid location
// as a core. Return -1 if no core was hit.
func (cc *coreControl) hitCore(gamex, gamez float64) (coreIndex int) {
	coreIndex = -1
	gridx, gridy := cc.playerToGrid(gamex, 0, gamez)
	for index, core := range cc.cores {
		x, y, z := core.Location()
		corex, corey := cc.spot.toGrid(x, y, z, cc.units)
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
func (cc *coreControl) reset(sc vu.Scene) {
	for _, core := range cc.cores {
		core.Dispose()
		sc.RemPart(core)
	}
	cc.cores = []vu.Part{}
	cc.tiles = []gridSpot{}
	for _, spot := range cc.saved {
		cc.tiles = append(cc.tiles, gridSpot{spot.x, spot.y})
	}
}

// createCore makes the new core model.
func (cc *coreControl) createCore(sc vu.Scene, fade float64) vu.Part {
	core := sc.AddPart()
	core.SetBody(vu.Sphere(0.15), 1, 0.8)

	// combine billboards to get an effect with some movement.
	cimg := core.AddPart().SetScale(0.25, 0.25, 0.25)
	cimg.SetCullable(false)
	cimg.SetRole("bbra").SetMesh("billboard").AddTex("ele")
	cimg.Role().SetAlpha(0.6)
	cimg.Role().SetUniform("spin", 1.93)
	cimg.Role().SetUniform("fd", fade)
	cimg.Role().Set2D()

	// same billboard rotating the other way.
	cimg = core.AddPart().SetScale(0.25, 0.25, 0.25)
	cimg.SetCullable(false)
	cimg.SetRole("bbra").SetMesh("billboard").AddTex("ele")
	cimg.Role().SetAlpha(0.6)
	cimg.Role().SetUniform("spin", -1.3)
	cimg.Role().SetUniform("fd", fade)
	cimg.Role().Set2D()

	// halo billboard rotating one way.
	cimg = core.AddPart().SetScale(0.25, 0.25, 0.25)
	cimg.SetCullable(false)
	cimg.SetRole("bbra").SetMesh("billboard").AddTex("halo")
	cimg.Role().SetAlpha(0.4)
	cimg.Role().SetUniform("spin", -2.0)
	cimg.Role().SetUniform("fd", fade)
	cimg.Role().Set2D()

	// halo billboard rotating the other way.
	cimg = core.AddPart().SetScale(0.25, 0.25, 0.25)
	cimg.SetCullable(false)
	cimg.SetRole("bbra").SetMesh("billboard").AddTex("halo")
	cimg.Role().SetAlpha(0.4)
	cimg.Role().SetUniform("spin", 1.0)
	cimg.Role().SetUniform("fd", fade)
	cimg.Role().Set2D()
	return core
}

// playerToGrid maps to rectangular areas around grid centers.  This is needed to
// interpret a player game location as an integer grid location.
func (cc *coreControl) playerToGrid(gamex, gamey, gamez float64) (gridx, gridy int) {
	inv := float64(1) / float64(cc.units)
	adj := float64(cc.units) * 0.5
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

// coreControl
// ===========================================================================
// gridSpot is used by coreControl and sentinel.

// gridSpot is used to track grid locations. It can be used to store grid
// locations and to convert back and forth between grid and game locations.
type gridSpot struct{ x, y int }

// toGame takes a grid location and translates into a game location.
// Game locations are where models of cores, walls, and tiles are placed.
func (gs *gridSpot) toGame(gridx, gridy int, units float64) (gamex, gamez float64) {
	return float64(gridx) * units, float64(-gridy) * units
}

// toGrid takes the current game location and translates into a grid location.
// Grid locations are where cores are dropped or fetched.
func (gs *gridSpot) toGrid(gamex, gamey, gamez, units float64) (gridx, gridy int) {
	return int(gamex / float64(units)), int(-gamez / float64(units))
}
