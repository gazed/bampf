// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"math"
	"vu"
	"vu/data"
	"vu/grid"
	"vu/physics"
)

// level groups everything needed for a single level.
// This includes the players, the AI's, and the level map.
type level struct {
	mp        *bampf       // Main program
	eng       *vu.Eng      // Engine: used to create new props.
	scene     vu.Scene     // Camera/player position for the stage.
	hd        *hud         // 2D information display for the stage.
	num       int          // Level number.
	gcx, gcy  int          // Grid level center.
	center    vu.Part      // Center tile model.
	walls     []vu.Part    // Walls.
	floor     vu.Part      // Large invisible floor for balls to bounce on.
	body      vu.Part      // Physics body for the player.
	player    *trooper     // Player size/shape for this stage
	sentries  []*sentinel  // Sentinels: player enemy AI's
	cc        *coreControl // Controls dropping cores on a stage.
	plan      grid.Grid    // Stage floorplan
	coreLimit int          // Limits the number of cores per level.
	units     int          // Reference base size for all game elements.
	colour    float32      // Current background shade-of-gray colour.
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
	lvl.units = 2
	lvl.colour = 1.0
	lvl.scene = g.eng.AddScene(vu.VF)
	lvl.scene.SetPerspective(75, float32(g.w)/float32(g.h), 0.1, 50)
	lvl.scene.SetLightLocation(50, 50, 50)
	lvl.scene.SetLightColour(0.2, 0.5, 0.8)
	lvl.scene.SetVisibleRadius(g.vr * 0.70)
	lvl.scene.SetVisibleDirection(true)
	lvl.scene.SetSorted(true)

	// save everything as one game stage.
	lvl.eng = g.eng
	lvl.mp = g.mp
	lvl.num = levelNum
	lvl.loadLevelResources(lvl.num)
	lvl.hd = newHud(g.eng, gameMuster[lvl.num])                 // create before player since...
	lvl.player = lvl.makePlayer(g.eng, lvl.hd.scene, lvl.num+1) // ... player is drawn within hd.scene.
	lvl.makeSentries(g.eng, lvl.scene, lvl.num)

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

	// add a physics body for the camera.  The center of the box is at 0, 0, 0.
	lvl.body = lvl.scene.AddPart()
	lvl.body.SetBody(10, 100)
	lvl.body.SetLocation(4, 0.5, 10)
	lvl.body.SetShape(physics.Abox(-0.25, -0.25, -0.25, 0.25, 0.25, 0.25))
	lvl.body.SetResolver(lvl.resolver)

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
	ratio := float32(width) / float32(height)
	lvl.scene.SetPerspective(75, ratio, 0.1, 50)
	lvl.hd.resize(width, height)
}

// update is called from game update. This keeps level related state update
// code together.
func (lvl *level) update() {
	lvl.setMist()
	lvl.fetchCores()
	lvl.moveSentinals()
	lvl.collideSentinals()
	lvl.createCore()
	lvl.hd.update(lvl.scene, lvl.sentries)
	lvl.player.updateEnergy()
	lvl.hd.cloakingActive(lvl.player.cloaked)
}

// The background colour becomes darker the deeper into the maze
// and the greater the level.
func (lvl *level) setMist() {
	px, _, pz := lvl.scene.ViewLocation()
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

// playerWorthy returns true if the player is able to ascend to the next level.
func (lvl *level) playerWorthy() bool { return lvl.player.fullHealth() && !lvl.player.cloaked }

// deactivate means this level is being taken out of action.  Tidy it up by ensuring all
// of its parts are out of the physics simulation.
func (lvl *level) deactivate() {
	lvl.setVisible(false)

	// remove the walls and floor from physics.
	for _, wall := range lvl.walls {
		wall.RemBody()
	}
	lvl.floor.RemBody()

	// reset the cores.
	lvl.cc.reset(lvl.scene)
	lvl.hd.resetCores()
}

// activate the current level. Bring its physics parts into the physics simulation.
func (lvl *level) activate(hm healthMonitor) {
	lvl.setVisible(true)
	lvl.player.monitorHealth("game", hm)
	lvl.hd.setLevel(lvl)

	// reset the camera each time, so it is in a known position.
	lvl.scene.SetViewRotation(0, 0, 0, 1)
	lvl.scene.SetViewLocation(4, 0.5, 10)
	lvl.player.resetEnergy()

	// ensure the walls and floor are added to the physics simulation.
	for _, wall := range lvl.walls {
		// set the walls collision shape based on (hand copied from) the .obj file.
		wall.SetBody(10, 100)
		wall.SetShape(physics.Abox(-1.0, -1.0, -1.0, 1.0, 1.0, 1.0))
	}
	lvl.floor.SetBody(0, 0)
	lvl.floor.SetShape(physics.Plane(0, 1, 0, 0, 0, 0))
}

// ensure the displayed action keys and labels are correct.
func (lvl *level) updateKeys(reacts map[string]vu.Reaction) {
	tk, ck := "t", "c" // defaults are intentionally off.
	for str, react := range reacts {
		if react.Name() == "teleport" {
			tk = str
		}
		if react.Name() == "cloak" {
			ck = str
		}
	}
	lvl.hd.xp.updateKeys(tk, ck)
}

// checkCollision is called each time the user moves the camera.
// The current camera position is used to update the related physics body location.
// Any collision callback results in a callback to Resolver.
func (lvl *level) checkCollision() {
	lvl.body.SetLocation(lvl.scene.ViewLocation())
	lvl.body.Collide()
}

// resolver is responsible for keeping the player out of walls and for allowing
// a sliding motion.
func (lvl *level) resolver(contacts []*physics.Contact) {
	if len(contacts) <= 0 {
		return
	}

	// the velocity vector of the camera is the difference of where it
	// was and where it is now.
	lx, ly, lz := lvl.scene.ViewLocation()

	// get out of a collision situation, by reversing the position along the
	// contact normal (plus some small fraction). Sliding occurs naturally because
	// the user is moved a portion in a direction that does not cause collision
	safetyMargin := float32(0.01)
	nx, ny, nz := lx, ly, lz
	for _, c := range contacts {
		nx += -c.Normal.X * (c.Depth + safetyMargin)
		// ny: up down collisions are ignored.
		nz += -c.Normal.Z * (c.Depth + safetyMargin)
	}

	// Note that the player previous location is not set when the view location
	// is set, but the prior-previous location is still valid.
	lvl.scene.SetViewLocation(nx, ny, nz)
	lvl.body.SetLocation(nx, ny, nz)
}

// loadLevelResources checks that the level specific models are available and loads
// whatever is needed.
//     level  0 == first playable level.  Bands 0, 1
//     level  1 == second playable level. Bands 0, 1, 2
//     level  n == last playable level.   Bands 0, 1, ..., n+1
func (lvl *level) loadLevelResources(level int) {
	if level < 0 || level > 4 {
		log.Printf("stage:loadBands invalid level.")
		return
	}

	// load floor tile textures.
	t := lvl.tileLabel(level)
	lvl.eng.LoadTexture(t)

	// load walls
	for band := 0; band <= level+1; band++ {

		// load wall model
		wm := lvl.wallMeshLabel(band)
		if !lvl.eng.Loaded(wm, &data.Mesh{}) {
			lvl.eng.LoadMesh(wm)
		}

		// load wall textures.
		wt := lvl.wallTextureLabel(band)
		lvl.eng.LoadTexture(wt)
	}
}
func (lvl *level) wallMeshLabel(band int) string    { return fmt.Sprintf("%dwall", band) }
func (lvl *level) wallTextureLabel(band int) string { return fmt.Sprintf("wall%d0", band) }
func (lvl *level) tileLabel(band int) string        { return fmt.Sprintf("tile%d0", band) }

// buildFloorPlan creates the level layout.
func (lvl *level) buildFloorPlan(scene vu.Scene, hd *hud, plan grid.Grid) {
	width, height := plan.Size()
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			xc := float32(x * lvl.units)
			yc := float32(-y * lvl.units)
			band := plan.Band(x, y) / 3
			if x == width/2 && y == height/2 {
				lvl.gcx, lvl.gcy = x, y // remember the maze center location
				lvl.center = scene.AddPart()
				lvl.center.SetFacade("tile", "uvra", "alpha")
				lvl.center.SetTexture("drop1", 1)
				lvl.center.SetLocation(xc, 0, yc)
			} else if plan.IsWall(x, y) {

				// draw flat on the y plane with the maze extending into the screen.
				wm := lvl.wallMeshLabel(band)
				wt := lvl.wallTextureLabel(band)
				wall := scene.AddPart()
				wall.SetFacade(wm, "uva", "green")
				wall.SetTexture(wt, 0)
				wall.SetLocation(xc, 0, yc)
				lvl.walls = append(lvl.walls, wall)

				// add the wall to the minimap
				hd.addWall(xc, yc)
			} else {
				// the floor tiles.
				tileLabel := lvl.tileLabel(band)
				tile := scene.AddPart()
				tile.SetFacade("tile", "uva", "alpha")
				tile.SetTexture(tileLabel, 0)
				tile.SetLocation(xc, 0, yc)

				// remember the tile locations for drop spots.
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
func (lvl *level) makePlayer(eng *vu.Eng, scene vu.Scene, levelNum int) *trooper {
	player := newTrooper(eng, scene.AddPart(), levelNum)
	player.part.RotateX(15)
	player.part.RotateY(15)
	player.setScale(100)
	player.noises["teleport"] = eng.UseSound("bampf")
	player.noises["fetch"] = eng.UseSound("fetch")
	player.noises["cloak"] = eng.UseSound("cloak")
	player.noises["decloak"] = eng.UseSound("decloak")
	player.noises["collide"] = eng.UseSound("collide")
	return player
}

// makeSentries creates some AI sentinels.
func (lvl *level) makeSentries(eng *vu.Eng, scene vu.Scene, levelNum int) {
	sentinels := []*sentinel{}
	numSentinels := gameMuster[levelNum]
	for cnt := 0; cnt < numSentinels; cnt++ {
		sentry := newSentinel(eng, scene.AddPart(), levelNum, lvl.units)
		sentry.setScale(0.25)
		sentinels = append(sentinels, sentry)
	}
	lvl.sentries = sentinels
}

// moveSentinels updates the sentintal locations by moving them a bit
// forward along their paths.
func (lvl *level) moveSentinals() {
	for _, sentry := range lvl.sentries {
		sentry.move(lvl.plan)
	}
}

// collideSentinals checks if the player collided with a sentinal.
func (lvl *level) collideSentinals() {
	if lvl.player.cloaked {
		return // player is immume from sentries.
	}
	pgx, pgy := lvl.cc.playerToGrid(lvl.scene.ViewLocation())
	for _, sentry := range lvl.sentries {
		sgx, sgy := lvl.cc.playerToGrid(sentry.Location())
		if pgx == sgx && pgy == sgy {
			lvl.eng.AuditorLocation(lvl.player.loc())
			noise := lvl.player.noises["collide"]
			noise.SetLocation(lvl.player.loc())
			noise.Play()

			// teleport the sentinal to the outside of the maze so that the
			// collision doesn't happen again.  Use the corner opposite the
			// starting location.
			sentry.setGridLocation(lvl.plan.Size())

			// remove health from the player and play the energy loss animation.
			lvl.player.detachCores(gameCellLoss[lvl.num])
			lvl.mp.ani.addAnimation(lvl.newEnergyLossAnimation())
		}
	}
}

// fetchCores picks up any nearby free cores. Don't check for collision...
// if the core is in the same grid element as the player, then it is
// automatically picked up.
func (lvl *level) fetchCores() {
	px, _, pz := lvl.scene.ViewLocation()
	coreIndex := lvl.cc.hitCore(px, pz)

	// attach the core to the player.
	health, _, max := lvl.player.health()
	if coreIndex >= 0 && health != max && !lvl.player.cloaked {
		lvl.eng.AuditorLocation(lvl.player.loc())
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

// createCore creates a core for the player to collect.  They are dropped onto
// floor tiles until enough have been generated.  A different, empty, floor
// tile is used each time.
func (lvl *level) createCore() {
	if !lvl.cc.timeToDrop() {
		return
	}
	health, _, max := lvl.player.health()
	energyNeeded := max - health
	coresNeeded := energyNeeded / gameCellGain[lvl.num]
	if lvl.cc.canDrop(coresNeeded) {
		gridx, gridy := lvl.cc.dropSpot()
		gamex, gamez := lvl.cc.dropCore(lvl.scene, gridx, gridy)
		lvl.hd.addCore(gamex, gamez)
	}
}

// teleport puts the player back to the starting location, safe from
// any sentinels.  The up/down and view direction are also reset to
// their original values in case the player has lost the maze.
func (lvl *level) teleport() {
	if lvl.player.teleport() {
		lvl.scene.SetViewLocation(0, 0.5, 10)
		lvl.scene.SetViewRotation(0, 0, 0, 1)
		lvl.scene.SetViewTilt(0)
		lvl.mp.ani.addAnimation(lvl.newTeleportAnimation())
	}
}

// cloak acts as a toggle if there is sufficient cloaking energy.
func (lvl *level) cloak() { lvl.player.cloak(!lvl.player.cloaked) }

// increaseCloak is a debug only method that greatly expands the cloaking time.
func (lvl *level) increaseCloak() { lvl.player.cloakEnergy += lvl.player.cemax * 10 }

// level
// ===========================================================================
// teleportAnimation

func (lvl *level) newTeleportAnimation() Animation { return &teleportAnimation{hd: lvl.hd, ticks: 25} }

// teleportAnimation shows a brief teleport after-effects which is supposed to
// look like smoke clearing.
type teleportAnimation struct {
	hd    *hud    // Needed to access teleport effect.
	fade  float32 // Quick fade the teleport effect.
	ticks int     // Animation run rate - number of animation steps.
	tkcnt int     // Current step.
	state int     // Track progress 0:start, 1:run, 2:done.
}

// Animate is called each game loop while the animation is active.
func (ta *teleportAnimation) Animate(gt, dt float32) bool {
	switch ta.state {
	case 0:
		ta.hd.teleportActive(true)
		ta.fade = 1
		ta.hd.teleportFade(ta.fade)
		ta.state = 1
		return true
	case 1:
		ta.fade -= 1 / float32(ta.ticks)
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
func (lvl *level) newEnergyLossAnimation() Animation {
	return &energyLossAnimation{hd: lvl.hd, ticks: 25}
}

// energyLossAnimation shows a brief flash to indicate a player has been hit
// by a sentry and has lost some energy.
type energyLossAnimation struct {
	hd    *hud    // needed to access energy loss effect.
	fade  float32 // quick fade the teleport effect.
	ticks int     // animation run rate - number of animation steps.
	tkcnt int     // current step
	state int     // track progress 0:start, 1:run, 2:done.
}

// Animate is called each game loop while the animation is active.
func (ea *energyLossAnimation) Animate(gt, dt float32) bool {
	switch ea.state {
	case 0:
		ea.hd.energyLossActive(true)
		ea.fade = 1
		ea.hd.energyLossFade(ea.fade)
		ea.state = 1
		return true
	case 1:
		ea.fade -= 1 / float32(ea.ticks)
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
