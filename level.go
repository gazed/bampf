// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"fmt"
	"math"

	"github.com/gazed/vu"
	"github.com/gazed/vu/grid"
	"github.com/gazed/vu/math/lin"
)

// level groups everything needed for a single level.
// This includes the player, the sentinels, and the level map.
type level struct {
	mp        *bampf       // Main program.
	root      *vu.Pov      // Top of local transform hierarchy.
	cam       *vu.Camera   // Quick access to the scene camera.
	hd        *hud         // 2D information display for the stage.
	num       int          // Level number.
	gcx, gcy  int          // Grid level center.
	center    *vu.Pov      // Center tile model.
	walls     []*vu.Pov    // Walls.
	floor     *vu.Pov      // Large invisible floor.
	body      *vu.Pov      // Physics body for the player.
	player    *trooper     // Player size/shape for this stage.
	sentries  []*sentinel  // Sentinels: player enemy AI's.
	cc        *coreControl // Controls dropping cores on a stage.
	plan      grid.Grid    // Stage floorplan.
	coreLimit int          // Max cores for this level.
	units     int          // Reference base size for all game elements.
	fade      float64      // distance to fade out.
	colour    float32      // Current background shade-of-gray colour.
	fov       float64      // Field of view.
}

// newLevel creates the indicated game level.
func newLevel(g *game, levelNum int) *level {
	var levelType = map[int]grid.Grid{
		0: grid.New(grid.DenseSkirmish),
		1: grid.New(grid.DenseSkirmish),
		2: grid.New(grid.SparseSkirmish),
		3: grid.New(grid.RoomSkirmish),
		4: grid.New(grid.RoomSkirmish),
	}

	// initialize the scenes.
	lvl := &level{}
	lvl.fade = g.vr * 0.7
	lvl.units = 2
	lvl.colour = 1.0
	lvl.fov = 75
	lvl.root = g.mp.eng.Root().NewPov()
	lvl.cam = lvl.root.NewCam()
	lvl.cam.Cull = vu.NewFrontCull(g.vr)
	lvl.cam.SetPerspective(lvl.fov, float64(g.ww)/float64(g.wh), 0.1, 50)

	// save everything as one game stage.
	lvl.mp = g.mp
	lvl.num = levelNum

	// create hud before player since player is drawn within hd.scene.
	s := g.mp.eng.State()
	lvl.hd = newHud(g.mp.eng, gameMuster[lvl.num], s.X, s.Y, s.W, s.H)
	lvl.player = lvl.makePlayer(lvl.hd.root, lvl.num+1)
	lvl.makeSentries(lvl.root, lvl.num)

	// create one large floor.
	lvl.floor = lvl.root.NewPov().SetAt(0, 0.2, 0)

	// create a new layout for the stage.
	plan := levelType[lvl.num]
	levelSize := gameMapSize(lvl.num)
	plan.Generate(levelSize, levelSize)

	// build and populate the floorplan
	lvl.walls = []*vu.Pov{}
	lvl.cc = newCoreControl(lvl.units, g.mp.ani)
	lvl.buildFloorPlan(lvl.root, lvl.hd, plan)
	lvl.plan = plan

	// set the intial player location.
	lvl.body = lvl.root.NewPov().SetAt(4, 0.5, 10)

	// start sentinels at the center of the stage.
	for _, sentry := range lvl.sentries {
		sentry.setGridAt(lvl.gcx, lvl.gcy)
	}
	lvl.player.resetEnergy()
	lvl.setVisible(false)
	return lvl
}

// setHudVisible turns the heads-up-display on or off.
func (lvl *level) setHudVisible(isVisible bool) {
	lvl.hd.setVisible(isVisible)
}

// setVisible toggles the visibility of the entire level.
func (lvl *level) setVisible(isVisible bool) {
	lvl.root.Cull = !isVisible
	lvl.hd.setVisible(isVisible)
}

// resize adjusts the level to the new window dimensions.
func (lvl *level) resize(width, height int) {
	ratio := float64(width) / float64(height)
	lvl.cam.SetPerspective(lvl.fov, ratio, 0.1, 50)
	lvl.hd.resize(width, height)
}

// update is called from game update.
// Note that update is not called during evolve transitions.
func (lvl *level) update() {

	// use the camera's orientation and the physics bodies location.
	lvl.body.SetView(lvl.cam.Look)
	lvl.cam.SetAt(lvl.body.At())

	// run animations and other regular checks.
	lvl.setMist()
	lvl.fetchCores()
	lvl.moveSentinels()
	lvl.collideSentinels()
	lvl.createCore()
	lvl.hd.update(lvl.cam, lvl.sentries)
	lvl.player.updateEnergy()
	lvl.hd.cloakingActive(lvl.player.cloaked)
}

// updateKeys ensures the displayed action keys and labels are correct.
func (lvl *level) updateKeys(keys []int) {
	if len(keys) > 5 {
		cloakKey, teleportKey := keys[4], keys[5]
		lvl.hd.xp.updateKeys(teleportKey, cloakKey)
	}
}

// The background colour becomes darker the deeper into the maze
// and the greater the level.
func (lvl *level) setMist() {
	px, _, pz := lvl.cam.At()
	cx, _, cz := lvl.center.At()
	dx, dz := float64(px-cx), float64(pz-cz)
	dist := math.Sqrt(dx*dx + dz*dz)
	dx, dz = float64(cx), float64(cz)
	edge := math.Sqrt(dx*dx + dz*dz)

	// darken the colour approaching the center of the maze.
	colour := float32(1.0) // full white
	if dist < edge {
		ratio := (edge - dist) / edge
		colour -= float32(ratio * gameCcol(lvl.num))
	}
	lvl.colour = colour // remember for level transitions.
	lvl.setBackgroundColour(colour)
}

// setBackgroundColour uses colour to form a gray based background.
func (lvl *level) setBackgroundColour(colour float32) {
	lvl.mp.eng.Set(vu.Color(colour, colour, colour, 1))
}

// isPlayerWorthy returns true if the player is able to ascend
// to the next level.
func (lvl *level) isPlayerWorthy() bool {
	return lvl.player.fullHealth() && !lvl.player.cloaked
}

// deactivate means this level is being taken out of action.
// Tidy it up by ensuring all of its parts are out of the
// physics simulation.
func (lvl *level) deactivate() {

	// remove the walls and floor from physics.
	for _, wall := range lvl.walls {
		wall.Dispose(vu.PovBody)
	}
	lvl.floor.Dispose(vu.PovBody)
	lvl.body.Dispose(vu.PovBody)

	// remove the cores.
	lvl.cc.reset()
	lvl.hd.resetCores()
}

// activate the current level. Add physics parts to the physics simulation.
func (lvl *level) activate(hm healthMonitor) {
	lvl.player.monitorHealth("game", hm)
	lvl.player.resetEnergy()
	lvl.hd.setLevel(lvl)

	// reset the camera each time, so it is in a known position.
	lvl.cam.SetAt(4, 0.5, 10)
	lvl.player.resetEnergy()

	// ensure the walls and floor are added to the physics simulation.
	for _, wall := range lvl.walls {
		// set the walls collision shape based on (hand copied from) the .obj file.
		wall.NewBody(vu.NewBox(1, 1, 1))
		wall.SetSolid(0, 0)
	}
	lvl.floor.NewBody(vu.NewBox(100, 25, 100))
	lvl.floor.SetSolid(0, 0.4)
	lvl.floor.SetAt(0, -25, 0)

	// add a physics body for the camera.
	lvl.body.NewBody(vu.NewSphere(0.25))
	lvl.body.SetSolid(1, 0)
}

// Generate the specific resource filenames for a particular level.
func (lvl *level) wallMeshLabel(band int) string    { return fmt.Sprintf("%dwall", band) }
func (lvl *level) wallTextureLabel(band int) string { return fmt.Sprintf("wall%d0", band) }
func (lvl *level) tileLabel(band int) string        { return fmt.Sprintf("tile%d0", band) }

// buildFloorPlan creates the level layout.
func (lvl *level) buildFloorPlan(root *vu.Pov, hd *hud, plan grid.Grid) {
	width, height := plan.Size()
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			xc := float64(x * lvl.units)
			yc := float64(-y * lvl.units)
			band := plan.Band(x, y) / 3
			if x == width/2 && y == height/2 {
				lvl.gcx, lvl.gcy = x, y // remember the maze center location
				lvl.center = root.NewPov().SetAt(xc, 0, yc)
				m := lvl.center.NewModel("uvra", "msh:tile", "tex:drop1")
				m.SetAlpha(0.7)
				m.SetUniform("spin", 1.0)
				m.SetUniform("fd", lvl.fade)
			} else if plan.IsOpen(x, y) {

				// the floor tiles.
				tileLabel := lvl.tileLabel(band)
				tile := root.NewPov().SetAt(xc, 0, yc)
				m := tile.NewModel("uva", "msh:tile", "tex:"+tileLabel)
				m.SetAlpha(0.7)
				m.SetUniform("fd", lvl.fade)

				// remember the tile locations for drop spots inside the maze.
				lvl.cc.addDropAt(x, y)
			} else {

				// draw flat on the y plane with the maze extending into the screen.
				wm := lvl.wallMeshLabel(band)
				wt := lvl.wallTextureLabel(band)
				wall := root.NewPov().SetAt(xc, 0, yc)
				m := wall.NewModel("uva", "msh:"+wm, "tex:"+wt)
				m.SetUniform("fd", lvl.fade)
				lvl.walls = append(lvl.walls, wall)

				// add the wall to the minimap
				hd.addWall(xc, yc)
			}
		}
	}

	// add core drop locations around the outside of the maze.
	for x := -1; x < width+1; x++ {
		lvl.cc.addDropAt(x, -1)
		lvl.cc.addDropAt(x, height)
	}
	for y := 0; y < height; y++ {
		lvl.cc.addDropAt(-1, y)
		lvl.cc.addDropAt(width, y)
	}
}

// makePlayer: the player is the camera... the player-trooper is used by the hud
// to show player status and as such this trooper is part of the hud scene.
func (lvl *level) makePlayer(root *vu.Pov, levelNum int) *trooper {
	player := newTrooper(root.NewPov(), levelNum)
	player.part.Spin(15, 0, 0)
	player.part.Spin(0, 15, 0)
	player.setScale(100)
	player.part.SetListener()
	return player
}

// makeSentries creates some AI sentinels.
func (lvl *level) makeSentries(root *vu.Pov, levelNum int) {
	sentinels := []*sentinel{}
	numSentinels := gameMuster[levelNum]
	for cnt := 0; cnt < numSentinels; cnt++ {
		sentry := newSentinel(root.NewPov(), levelNum, lvl.units, lvl.fade)
		sentry.setScale(0.25)
		sentinels = append(sentinels, sentry)
	}
	lvl.sentries = sentinels
}

// moveSentinels updates the sentinels locations by moving them a bit
// forward along their paths.
func (lvl *level) moveSentinels() {
	for _, sentry := range lvl.sentries {
		sentry.move(lvl.plan)
	}
}

// collideSentinels checks if the player collided with a sentinel.
// The check is grid based, not physics based.
func (lvl *level) collideSentinels() {
	if lvl.player.cloaked {
		return // player is immume from sentries.
	}
	x, y, z := lvl.cam.At()
	pgx, pgy := toGrid(x, y, z, float64(lvl.units))
	for _, sentry := range lvl.sentries {
		sx, sy, sz := sentry.location()
		sgx, sgy := toGrid(sx, sy, sz, float64(lvl.units))
		if pgx == sgx && pgy == sgy {
			lvl.player.play(collideSound)

			// teleport the sentinel to the outside of the maze so that the
			// collision doesn't happen again.
			safex, safey := lvl.plan.Size() // top right corner.
			sentry.setGridAt(safex, safey)
			if pgx == safex && pgy == safey {
				sentry.setGridAt(-1, -1) // bottom left corner.
			}

			// remove health from the player and show the energy loss animation.
			lvl.player.detachCores(gameCellLoss[lvl.num])
			lvl.mp.ani.addAnimation(lvl.newEnergyLossAnimation())
		}
	}
}

// fetchCores picks up any nearby free cores if the core is in the
// same grid element as the player. No need to check for actual collision.
func (lvl *level) fetchCores() {
	px, _, pz := lvl.cam.At()
	coreIndex := lvl.cc.hitCore(px, pz)

	// attach the core to the player.
	health, _, max := lvl.player.health()
	if coreIndex >= 0 && health != max && !lvl.player.cloaked {
		lvl.player.play(fetchSound)
		gamex, gamez := lvl.cc.remCore(coreIndex)
		lvl.hd.remCore(gamex, gamez)
		for cnt := 0; cnt < gameCellGain[lvl.num]; cnt++ {
			lvl.player.attach()
		}

		// add more cloaking energy each time a core is picked up.
		lvl.player.addCloakEnergy()
	}
}

// createCore creates a core if necessary. The core is dropped onto
// an empty floor tile.
func (lvl *level) createCore() {
	if !lvl.cc.timeToDrop() {
		return
	}
	health, _, max := lvl.player.health()
	energyNeeded := max - health
	coresNeeded := energyNeeded / gameCellGain[lvl.num]
	if lvl.cc.canDrop(coresNeeded) {
		gridx, gridy := lvl.cc.dropSpot()
		gamex, gamez := lvl.cc.dropCore(lvl.root, lvl.fade, gridx, gridy)
		lvl.hd.addCore(gamex, gamez)
	}
}

// teleport puts the player back to the starting location, safe from
// any sentinels. The up/down and view direction are also reset to
// their original values in case the player has lost sight of the maze.
func (lvl *level) teleport() {
	if lvl.player.teleport() {
		lvl.body.Dispose(vu.PovBody)
		lvl.body.SetAt(0, 0.5, 10)
		lvl.body.SetView(lin.QI)
		lvl.cam.SetAt(0, 0.5, 10)
		lvl.body.NewBody(vu.NewSphere(0.25))
		lvl.body.SetSolid(1, 0)
		lvl.mp.ani.addAnimation(lvl.newTeleportAnimation())
	}
}

// cloak toggles player cloaking. Cloaking only enables if there is
// sufficient cloaking energy.
func (lvl *level) cloak() {
	lvl.player.cloak(!lvl.player.cloaked)
}

// debugCloak is a debug only method that greatly expands the cloaking time.
func (lvl *level) debugCloak() {
	lvl.player.cloakEnergy += lvl.player.cemax * 10
}

// level
// ===========================================================================
// teleportAnimation

func (lvl *level) newTeleportAnimation() animation {
	return &teleportAnimation{hd: lvl.hd, ticks: 25}
}

// teleportAnimation shows a brief teleport after-effect which is supposed to
// look like smoke clearing.
type teleportAnimation struct {
	hd    *hud    // Needed to access teleport effect.
	fade  float64 // Quick fade the teleport effect.
	ticks int     // Animation run rate - number of animation steps.
	tkcnt int     // Current step.
	state int     // Track progress 0:start, 1:run, 2:done.
}

// Animate is called each game loop while the animation is active.
func (ta *teleportAnimation) Animate(dt float64) bool {
	switch ta.state {
	case 0:
		ta.hd.teleportActive(true)
		ta.fade = 1
		ta.hd.teleportFade(ta.fade)
		ta.state = 1
		return true
	case 1:
		ta.fade -= 1 / float64(ta.ticks)
		ta.hd.teleportFade(ta.fade)
		if ta.tkcnt >= ta.ticks {
			ta.Wrap()
			return false // animation done.
		}
		ta.tkcnt++
		return true
	default:
		return false // animation done.
	}
}

// Wrap cleans up and closes down the animation.
func (ta *teleportAnimation) Wrap() {
	ta.fade = 0.5
	ta.hd.teleportFade(ta.fade)
	ta.hd.teleportActive(false)
	ta.state = 2
}

// teleportAnimation
// ===========================================================================
// energyLossAnimation

func (lvl *level) newEnergyLossAnimation() animation {
	return &energyLossAnimation{hd: lvl.hd, ticks: 25}
}

// energyLossAnimation shows a brief flash to indicate a player has been hit
// by a sentry and has lost some energy.
type energyLossAnimation struct {
	hd    *hud    // needed to access energy loss effect.
	fade  float64 // quick fade the teleport effect.
	ticks int     // animation run rate - number of animation steps.
	tkcnt int     // current step
	state int     // track progress 0:start, 1:run, 2:done.
}

// Animate is called each game loop while the animation is active.
func (ea *energyLossAnimation) Animate(dt float64) bool {
	switch ea.state {
	case 0:
		ea.hd.energyLossActive(true)
		ea.fade = 1
		ea.hd.energyLossFade(ea.fade)
		ea.state = 1
		return true
	case 1:
		ea.fade -= 1 / float64(ea.ticks)
		ea.hd.energyLossFade(ea.fade)
		if ea.tkcnt >= ea.ticks {
			ea.Wrap()
			return false // animation done.
		}
		ea.tkcnt++
		return true
	default:
		return false // animation done.
	}
}

// Wrap cleans up and closes down the animation.
func (ea *energyLossAnimation) Wrap() {
	ea.fade = 0.5
	ea.hd.energyLossFade(ea.fade)
	ea.hd.energyLossActive(false)
	ea.state = 2
}
