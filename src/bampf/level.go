// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"fmt"
	"math"
	"vu"
	"vu/grid"
)

// level groups everything needed for a single level.
// This includes the player, the sentinels, and the level map.
type level struct {
	mp        *bampf       // Main program.
	eng       vu.Engine    // Engine: used to create new props.
	scene     vu.Scene     // Camera/player position for the stage.
	hd        *hud         // 2D information display for the stage.
	num       int          // Level number.
	gcx, gcy  int          // Grid level center.
	center    vu.Part      // Center tile model.
	walls     []vu.Part    // Walls.
	floor     vu.Part      // Large invisible floor for balls to bounce on.
	body      vu.Part      // Physics body for the player.
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
		0: grid.New(grid.DENSE_SKIRMISH),
		1: grid.New(grid.DENSE_SKIRMISH),
		2: grid.New(grid.SPARSE_SKIRMISH),
		3: grid.New(grid.ROOMS_SKIRMISH),
		4: grid.New(grid.ROOMS_SKIRMISH),
	}

	// initialize the scenes.
	lvl := &level{}
	lvl.fade = g.vr * 0.7
	lvl.units = 2
	lvl.colour = 1.0
	lvl.fov = 75
	lvl.scene = g.eng.AddScene(vu.VF)
	lvl.scene.SetPerspective(lvl.fov, float64(g.w)/float64(g.h), 0.1, 50)
	lvl.scene.SetLightLocation(50, 50, 50)
	lvl.scene.SetLightColour(0.2, 0.5, 0.8)
	lvl.scene.SetCuller(vu.NewFacingCuller(lvl.fade))
	lvl.scene.SetSorted(true)

	// save everything as one game stage.
	lvl.eng = g.eng
	lvl.mp = g.mp
	lvl.num = levelNum

	// create hud before player since player is drawn within hd.scene.
	lvl.hd = newHud(g.eng, gameMuster[lvl.num])
	lvl.player = lvl.makePlayer(g.eng, lvl.hd.scene, lvl.num+1)
	lvl.makeSentries(lvl.scene, lvl.num)

	// create one large physics floor.
	lvl.floor = lvl.scene.AddPart()
	lvl.floor.SetLocation(0, 0.2, 0)

	// create a new layout for the stage.
	plan := levelType[lvl.num]
	levelSize := gameMapSize(lvl.num)
	plan.Generate(levelSize, levelSize)

	// build and populate the floorplan
	lvl.walls = []vu.Part{}
	lvl.cc = newCoreControl(lvl.units)
	lvl.buildFloorPlan(lvl.scene, lvl.hd, plan)
	lvl.plan = plan

	// set the intial player location.
	lvl.body = lvl.scene.AddPart()
	lvl.body.SetLocation(4, 0.5, 10)

	// start sentinels at the center of the stage.
	for _, sentry := range lvl.sentries {
		sentry.setGridLocation(lvl.gcx, lvl.gcy)
	}
	lvl.player.resetEnergy()
	return lvl
}

// setHudVisible turns the heads-up-display on or off.
func (lvl *level) setHudVisible(isVisible bool) {
	lvl.hd.setVisible(isVisible)
}

// setVisible toggles the visibility of the entire level.
func (lvl *level) setVisible(isVisible bool) {
	lvl.scene.SetVisible(isVisible)
	lvl.hd.setVisible(isVisible)
}

// resize adjusts the level to the new window dimensions.
func (lvl *level) resize(width, height int) {
	ratio := float64(width) / float64(height)
	lvl.scene.SetPerspective(lvl.fov, ratio, 0.1, 50)
	lvl.hd.resize(width, height)
}

// alterFov is for debugging and testing. It helps to find a reasonable
// vertical fov value.
func (lvl *level) alterFov(change float64) {
	lvl.fov += change
	_, _, w, h := lvl.eng.Size()
	lvl.resize(w, h)

	// Uncomment to see the actual vertical and horizontal field of view values.
	// vrad := lin.Rad(lvl.fov)
	// hrad := 2 * math.Atan(math.Tan(vrad/2)*float64(w)/float64(h))
	// fmt.Printf("vfov %f hfov %f\n", lvl.fov, lin.Deg(hrad))
}

// update is called from game update.
// Note that update is not called during evolve transitions.
func (lvl *level) update() {

	// use the camera's orientation and the physics bodies location.
	lvl.body.SetRotation(lvl.scene.Rotation())
	lvl.scene.SetLocation(lvl.body.Location())

	// run animations and other regular checks.
	lvl.setMist()
	lvl.fetchCores()
	lvl.moveSentinels()
	lvl.collideSentinels()
	lvl.createCore()
	lvl.hd.update(lvl.scene, lvl.sentries)
	lvl.player.updateEnergy()
	lvl.hd.cloakingActive(lvl.player.cloaked)
}

// The background colour becomes darker the deeper into the maze
// and the greater the level.
func (lvl *level) setMist() {
	px, _, pz := lvl.scene.Location()
	cx, _, cz := lvl.center.Location()
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
	lvl.eng.Color(colour, colour, colour, 1)
}

// isPlayerWorthy returns true if the player is able to ascend to the next level.
func (lvl *level) isPlayerWorthy() bool { return lvl.player.fullHealth() && !lvl.player.cloaked }

// deactivate means this level is being taken out of action. Tidy it up by ensuring all
// of its parts are out of the physics simulation.
func (lvl *level) deactivate() {
	lvl.setVisible(false)

	// remove the walls and floor from physics.
	for _, wall := range lvl.walls {
		wall.RemBody()
	}
	lvl.floor.RemBody()
	lvl.body.RemBody()

	// remove the cores.
	lvl.cc.reset(lvl.scene)
	lvl.hd.resetCores()
}

// activate the current level. Add physics parts to the physics simulation.
func (lvl *level) activate(hm healthMonitor) {
	lvl.setVisible(true)
	lvl.player.monitorHealth("game", hm)
	lvl.hd.setLevel(lvl)

	// reset the camera each time, so it is in a known position.
	lvl.scene.SetRotation(0, 0, 0, 1)
	lvl.scene.SetLocation(4, 0.5, 10)
	lvl.player.resetEnergy()

	// ensure the walls and floor are added to the physics simulation.
	for _, wall := range lvl.walls {
		// set the walls collision shape based on (hand copied from) the .obj file.
		wall.SetBody(vu.Box(1, 1, 1), 0, 0)
	}
	lvl.floor.SetBody(vu.Box(100, 25, 100), 0, 0.4)
	lvl.floor.SetLocation(0, -25, 0)

	// add a physics body for the camera.
	lvl.body.SetBody(vu.Sphere(0.25), 1, 0)
}

// ensure the displayed action keys and labels are correct.
func (lvl *level) updateKeys(reacts ReactionSet) {
	tk, ck := reacts.Key("teleport"), reacts.Key("cloak")
	lvl.hd.xp.updateKeys(tk, ck)
}

// Generate the specific resource filenames for a particular level.
func (lvl *level) wallMeshLabel(band int) string    { return fmt.Sprintf("%dwall", band) }
func (lvl *level) wallTextureLabel(band int) string { return fmt.Sprintf("wall%d0", band) }
func (lvl *level) tileLabel(band int) string        { return fmt.Sprintf("tile%d0", band) }

// buildFloorPlan creates the level layout.
func (lvl *level) buildFloorPlan(scene vu.Scene, hd *hud, plan grid.Grid) {
	width, height := plan.Size()
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			xc := float64(x * lvl.units)
			yc := float64(-y * lvl.units)
			band := plan.Band(x, y) / 3
			if x == width/2 && y == height/2 {
				lvl.gcx, lvl.gcy = x, y // remember the maze center location
				lvl.center = scene.AddPart().SetLocation(xc, 0, yc)
				lvl.center.SetRole("uvra").SetMesh("tile").AddTex("drop1").SetMaterial("alpha")
				lvl.center.Role().SetUniform("spin", 1.0)
				lvl.center.Role().SetUniform("fd", lvl.fade)
			} else if plan.IsWall(x, y) {

				// draw flat on the y plane with the maze extending into the screen.
				wm := lvl.wallMeshLabel(band)
				wt := lvl.wallTextureLabel(band)
				wall := scene.AddPart().SetLocation(xc, 0, yc)
				wall.SetRole("uva").SetMesh(wm).AddTex(wt).SetMaterial("green")
				wall.Role().SetUniform("fd", lvl.fade)
				lvl.walls = append(lvl.walls, wall)

				// add the wall to the minimap
				hd.addWall(xc, yc)
			} else {
				// the floor tiles.
				tileLabel := lvl.tileLabel(band)
				tile := scene.AddPart().SetLocation(xc, 0, yc)
				tile.SetRole("uva").SetMesh("tile").AddTex(tileLabel).SetMaterial("alpha")
				tile.Role().SetUniform("fd", lvl.fade)

				// remember the tile locations for drop spots inside the maze.
				lvl.cc.addDropLocation(x, y)
			}
		}
	}

	// add core drop locations around the outside of the maze.
	for x := -1; x < width+1; x++ {
		lvl.cc.addDropLocation(x, -1)
		lvl.cc.addDropLocation(x, height)
	}
	for y := 0; y < height; y++ {
		lvl.cc.addDropLocation(-1, y)
		lvl.cc.addDropLocation(width, y)
	}
}

// makePlayer: the player is the camera... the player-trooper is used by the hud
// to show player status and as such this trooper is part of the hud scene.
func (lvl *level) makePlayer(eng vu.Engine, scene vu.Scene, levelNum int) *trooper {
	player := newTrooper(eng, scene.AddPart(), levelNum)
	player.part.Spin(15, 0, 0)
	player.part.Spin(0, 15, 0)
	player.setScale(100)
	player.noises["teleport"] = eng.UseSound("bampf")
	player.noises["fetch"] = eng.UseSound("fetch")
	player.noises["cloak"] = eng.UseSound("cloak")
	player.noises["decloak"] = eng.UseSound("decloak")
	player.noises["collide"] = eng.UseSound("collide")
	return player
}

// makeSentries creates some AI sentinels.
func (lvl *level) makeSentries(scene vu.Scene, levelNum int) {
	sentinels := []*sentinel{}
	numSentinels := gameMuster[levelNum]
	for cnt := 0; cnt < numSentinels; cnt++ {
		sentry := newSentinel(scene.AddPart(), levelNum, lvl.units, lvl.fade)
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
func (lvl *level) collideSentinels() {
	if lvl.player.cloaked {
		return // player is immume from sentries.
	}
	pgx, pgy := lvl.cc.playerToGrid(lvl.scene.Location())
	for _, sentry := range lvl.sentries {
		sgx, sgy := lvl.cc.playerToGrid(sentry.location())
		if pgx == sgx && pgy == sgy {
			lvl.eng.PlaceSoundListener(lvl.player.loc())
			noise := lvl.player.noises["collide"]
			noise.SetLocation(lvl.player.loc())
			noise.Play()

			// teleport the sentinel to the outside of the maze so that the
			// collision doesn't happen again. Use the corner opposite the
			// starting location.
			sentry.setGridLocation(lvl.plan.Size())

			// remove health from the player and show the energy loss animation.
			lvl.player.detachCores(gameCellLoss[lvl.num])
			lvl.mp.ani.addAnimation(lvl.newEnergyLossAnimation())
		}
	}
}

// fetchCores picks up any nearby free cores if the core is in the
// same grid element as the player. No need to check for actual collision.
func (lvl *level) fetchCores() {
	px, _, pz := lvl.scene.Location()
	coreIndex := lvl.cc.hitCore(px, pz)

	// attach the core to the player.
	health, _, max := lvl.player.health()
	if coreIndex >= 0 && health != max && !lvl.player.cloaked {
		lvl.eng.PlaceSoundListener(lvl.player.loc())
		fetchNoise := lvl.player.noises["fetch"]
		fetchNoise.SetLocation(lvl.player.loc())
		fetchNoise.Play()
		gamex, gamez := lvl.cc.remCore(lvl.scene, coreIndex)
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
		gamex, gamez := lvl.cc.dropCore(lvl.scene, lvl.fade, gridx, gridy)
		lvl.hd.addCore(gamex, gamez)
	}
}

// teleport puts the player back to the starting location, safe from
// any sentinels. The up/down and view direction are also reset to
// their original values in case the player has lost sight of the maze.
func (lvl *level) teleport(down int) {
	if down == 1 {
		if lvl.player.teleport() {
			lvl.body.RemBody()
			lvl.body.SetLocation(0, 0.5, 10)
			lvl.body.SetRotation(0, 0, 0, 1)
			lvl.scene.SetLocation(0, 0.5, 10)
			lvl.scene.SetRotation(0, 0, 0, 1)
			lvl.scene.SetTilt(0)
			lvl.body.SetBody(vu.Sphere(0.25), 1, 0)
			lvl.mp.ani.addAnimation(lvl.newTeleportAnimation())
		}
	}
}

// cloak toggles player cloaking. Cloaking only enables if there is
// sufficient cloaking energy.
func (lvl *level) cloak(down int) {
	if down == 1 {
		lvl.player.cloak(!lvl.player.cloaked)
	}
}

// increaseCloak is a debug only method that greatly expands the cloaking time.
func (lvl *level) increaseCloak() { lvl.player.cloakEnergy += lvl.player.cemax * 10 }

// level
// ===========================================================================
// teleportAnimation

func (lvl *level) newTeleportAnimation() animation { return &teleportAnimation{hd: lvl.hd, ticks: 25} }

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
		ta.tkcnt += 1
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

// newEnergyLossAnimation
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
		ea.tkcnt += 1
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
