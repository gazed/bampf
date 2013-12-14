// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"math"
	"runtime/debug"
	"vu"
	"vu/math/lin"
)

// game keeps track of the game play screen. This includes all game levels
// and the heads up display (hud).
type game struct {
	mp     *bampf            // Main program.
	eng    vu.Engine         // Game engine.
	levels map[int]*level    // Game levels.
	cl     *level            // Current level.
	acts   map[string]string // Reactions.
	lens   lens              // Dictates how the camera moves.
	w, h   int               // Window size.
	state  func(int) int     // Current screen state.

	// User input handlers for this screen.
	reacts   map[string]vu.Reaction // User action map
	moveDt   float64                // Delta amount for moves.
	mx, my   int                    // Mouse locations.
	mxp, myp int                    // Previous mouse locations.
	dt       float64                // Update delta time.

	// Debug variables
	fly  bool     // Debug flying ability switch, see game_debug.go
	last lastSpot // Keeps the last valid player position when debugging.

	// Static state.
	run  float64 // Run speed.
	spin float64 // Spin speed.
	vr   float64 // Visible radius.
}

// Implement the screen interface.
func (g *game) fadeIn() animation        { return g.newStartGameAnimation() }
func (g *game) fadeOut() animation       { return g.newEndGameAnimation() }
func (g *game) resize(width, height int) { g.handleResize(width, height) }
func (g *game) update(input *vu.Input)   { g.handleUpdate(input) }
func (g *game) transition(event int)     { g.state(event) }

// newGameScreen initializes the gameplay screen.
func newGameScreen(mp *bampf) (scr screen, reacts map[string]vu.Reaction) {
	g := &game{}
	g.state = g.deactive
	g.mp = mp
	g.eng = mp.eng
	g.lens = &fps{}
	g.w, g.h = mp.wx, mp.wy
	g.run = 10  // shared constant
	g.spin = 25 // shared constant
	g.vr = 25   // shared constant
	g.levels = make(map[int]*level)

	// user input handlers.
	g.reacts = g.reactions()
	g.addDebugReactions(g)
	return g, g.reacts
}

// deactive state waits for the evolve event.
func (g *game) deactive(event int) int {
	switch event {
	case query:
		return deactivate
	case evolve:
		g.setLevel(g.mp.launchLevel)
		g.cl.setHudVisible(false)
		g.cl.scene.SetViewTilt(75)
		g.cl.scene.SetViewLocation(4, g.vr, 10)
		g.cl.setBackgroundColour(g.cl.colour)
		g.enableKeys()
		g.eng.ShowCursor(false)
		g.state = g.evolving
	default:
		log.Printf("game: deactive state: invalid transition %d", event)
	}
	return deactivate
}

// active state waits for the evolve, pause, or deactivate events.
func (g *game) active(event int) int {
	switch event {
	case query:
		return activate
	case evolve:
		g.disableKeys()
		g.state = g.evolving
	case pause:
		g.disableKeys()
		g.eng.ShowCursor(true)
		g.state = g.paused
	case deactivate:
		g.disableKeys()
		g.eng.ShowCursor(true)
		g.cl.setVisible(false)
		g.state = g.deactive
	case activate:
		// ignored. Possible when the animation was skipped before
		// it put the game into evolve state.
	default:
		log.Printf("game: active state: invalid transition %d", event)
		debug.PrintStack()
	}
	return activate
}

// paused state waits for the activate or deactivate events.
func (g *game) paused(event int) int {
	switch event {
	case query:
		return pause
	case activate:
		g.enableKeys()
		g.eng.ShowCursor(false)
		g.state = g.active
	case deactivate:
		g.cl.setVisible(false)
		g.state = g.deactive
	default:
		log.Printf("game: paused state: invalid transition %d", event)
	}
	return pause
}

// evolving state waits for deactivate or activate events.
func (g *game) evolving(event int) int {
	switch event {
	case query:
		return evolve
	case deactivate:
		g.eng.ShowCursor(true)
		g.cl.setVisible(false)
		g.state = g.deactive
	case activate:
		g.enableKeys()
		g.state = g.active
	case evolve:
		// ignored. Happens when evolving from one level to the next
		// as both animations try to enter the evolve state.
	default:
		log.Printf("game: evolving state: invalid transition %d", event)
	}
	return evolve
}

// disableKeys disallows certain keys when the screen is not active.
func (g *game) disableKeys() { delete(g.reacts, "Esc") }

// enableKeys reenables deactivated keys.
func (g *game) enableKeys() {
	g.reacts["Esc"] = vu.NewReactOnce("options", func() { g.mp.toggleOptions() })
	if lm, ok := g.reacts["Lm"]; ok {
		lm.SetTime()
	}
	g.cl.updateKeys(g.reacts)
}

// handleResize affects all levels, not just the current one.
func (g *game) handleResize(width, height int) {
	g.w, g.h = width, height
	for _, stage := range g.levels {
		stage.resize(width, height)
	}
}

// handleUpdate processes the user input.
func (g *game) handleUpdate(input *vu.Input) {
	g.mxp, g.myp = g.mx, g.my
	g.dt, g.mx, g.my = input.Dt, input.Mx, input.My
	if g.cl == nil { // no current level just yet... still starting.
		return
	}

	// pre-process user input. Ensure that each simultaneous move request
	// gets a portion of the total move amount.
	g.moveDt = input.Dt * 2
	for key, _ := range input.Down {
		if reaction, ok := g.reacts[key]; ok {
			rn := reaction.Name()
			if rn == "mForward" || rn == "mBack" || rn == "mLeft" || rn == "mRight" {
				g.moveDt *= 0.5
			}
		}
	}

	// react to all other user input.
	for key, release := range input.Down {
		if input.Shift {
			key = "Sh-" + key
		}
		if reaction, ok := g.reacts[key]; ok {
			reaction.Do()
			rn := reaction.Name()

			// limit how far away from the center a player can get.
			if rn == "mForward" || rn == "mBack" || rn == "mLeft" || rn == "mRight" {
				g.limitWandering(g.cl.scene)
				if release < 0 {
					g.cl.body.Stop()
				}
			}
		}
	}

	// update the camera based on the mouse movements each time through the game loop.
	xdiff, ydiff := float64(g.mx-g.mxp), float64(g.my-g.myp)
	g.lens.look(g.cl.scene, g.spin, g.dt, xdiff, ydiff)

	// update game state if the game is active and not transitioning betwen levels.
	if g.state(query) == activate {
		if g.state(query) != evolve {
			g.cl.update()   // level specific updates.
			g.evolveCheck() // check if the player is ready to evolve.
		}
		g.centerMouse() // keep centering the mouse.
	}
}

// centerMouse pops the mouse back to the center of the window, but only
// when the mouse starts to stray too far away.
func (g *game) centerMouse() {
	cx, cy := g.w/2, g.h/2
	if math.Abs(float64(cx-g.mx)) > 200 || math.Abs(float64(cy-g.my)) > 200 {
		g.eng.SetCursorAt(g.w/2, g.h/2)
	}
}

// limitWandering puts a limit on how far the player can get from the center
// of the level. This allows the player to feel like they are traveling away
// forever, but they can then return to the center in very little time.
func (g *game) limitWandering(scene vu.Scene) {
	maxd := g.vr * 3                    // max allowed distance from center
	cx, _, cz := g.cl.center.Location() // center location
	x, y, z := g.cl.body.Location()     // player location
	toc := &lin.V3{x - cx, y, z - cz}   // vector to center
	dtoc := toc.Len()                   // distance to center
	if dtoc > maxd {

		// stop moving forward and move a bit back to center.
		g.cl.body.Stop()
		g.cl.body.Push(-toc.X/100, 0, -toc.Z/100)
	}
}

// reactions are the user input handlers. These are the default mappings and
// will be used unless overridden by the user in this session or from a
// previous sessions saved key mappings.
func (g *game) reactions() map[string]vu.Reaction {
	reactions := map[string]vu.Reaction{
		"W":   vu.NewReaction("mForward", func() { g.lens.forward(g.cl.body, g.dt, g.run) }),
		"S":   vu.NewReaction("mBack", func() { g.lens.back(g.cl.body, g.dt, g.run) }),
		"A":   vu.NewReaction("mLeft", func() { g.lens.left(g.cl.body, g.dt, g.run) }),
		"D":   vu.NewReaction("mRight", func() { g.lens.right(g.cl.body, g.dt, g.run) }),
		"C":   vu.NewReactOnce("cloak", func() { g.cl.cloak() }),
		"T":   vu.NewReactOnce("teleport", func() { g.cl.teleport() }),
		"Esc": vu.NewReactOnce("options", func() { g.mp.toggleOptions() }),
		"Sp":  vu.NewReactOnce("skip", func() { g.mp.ani.skip() }),
	}
	return g.restoreBindings(reactions)
}

// restoreBindings overwrites the default bindings with saved bindings.
func (g *game) restoreBindings(original map[string]vu.Reaction) map[string]vu.Reaction {
	saver := newSaver()
	fromDisk := saver.restore()
	if len(fromDisk.Kmap) > 0 {
		restored := map[string]vu.Reaction{}
		for oKey, reaction := range original {
			wasRestored := false
			for boundName, mKey := range fromDisk.Kmap {
				if boundName == reaction.Name() {
					wasRestored = true
					restored[mKey] = reaction
				}
			}
			if !wasRestored {
				restored[oKey] = reaction
			}
		}
		return restored
	}
	return original
}

// addDebugReactions checks if the optional debugReactions method is present
// in the build, and adds the extra debug reactions if it is.
func (g *game) addDebugReactions(gi interface{}) {
	if gd, ok := gi.(interface {
		debugReactions() map[string]vu.Reaction
	}); ok {
		for key, val := range gd.debugReactions() {
			g.reacts[key] = val
		}
	}
}

// evolveCheck looks for a player at full health that is at the center of
// the level. This is the trigger to complete the level.
func (g *game) evolveCheck() {
	if g.cl.isPlayerWorthy() {
		gridx, gridy := g.cl.cc.playerToGrid(g.cl.scene.ViewLocation())
		if gridx == g.cl.gcx && gridy == g.cl.gcy {
			if g.cl.num < 4 {
				g.cl.center.SetTexture("drop1", 1)
				g.cl.center.SetScale(1, 1, 1)
				g.mp.ani.addAnimation(g.newEvolveAnimation(1))
			} else if g.cl.num == 4 {
				g.mp.state(done) // let bampf know that the game is over.
			}
		}
	}
}

// healthUpdated is a callback whenever player health changes.
// Players that have full health are worthy to descend to the next level
// (they just have to reach the center first).
func (g *game) healthUpdated(health, warn, high int) {
	if health <= 0 {
		if g.cl.num > 0 {
			g.mp.ani.addAnimation(g.newEvolveAnimation(-1))
		}
	}
	if g.cl.isPlayerWorthy() {
		g.cl.center.SetTexture("drop2", 1)
		g.cl.center.SetScale(1, 50, 1)
	} else {
		g.cl.center.SetTexture("drop1", 1)
		g.cl.center.SetScale(1, 1, 1)
	}
}

// setLevel updates to the requested level, generating a new level if necessary.
func (g *game) setLevel(lvl int) {
	if g.cl != nil {
		g.cl.deactivate()
	}
	if _, ok := g.levels[lvl]; !ok {
		g.levels[lvl] = newLevel(g, lvl)
	}
	g.cl = g.levels[lvl]
	g.cl.activate(g)
	g.cl.updateKeys(g.reacts)
}

// create the various game transition animations.
func (g *game) newStartGameAnimation() animation {
	return &fadeLevelAnimation{g: g, gstate: activate, dir: 1, ticks: 100, start: g.vr, stop: 0.5}
}
func (g *game) newEndGameAnimation() animation {
	return &fadeLevelAnimation{g: g, gstate: deactivate, dir: 1, ticks: 100, start: 0.5, stop: -g.vr}
}
func (g *game) newEvolveAnimation(dir int) animation {
	g.cl.scene.SetViewTilt(0)
	fadeOut := &fadeLevelAnimation{g: g, gstate: -1, dir: dir, ticks: 100, start: 0.5, stop: g.vr * float64(-dir)}
	transition := func() {
		g.setLevel(g.cl.num + dir) // switch to the new level.
		g.cl.setHudVisible(false)
		g.cl.scene.SetViewTilt(75 * float64(dir))
		g.cl.scene.SetViewLocation(4, g.vr*float64(dir), 10)
	}
	fadeIn := &fadeLevelAnimation{g: g, gstate: activate, dir: dir, ticks: 100, start: g.vr * float64(dir), stop: 0.5}
	return newTransitionAnimation(fadeOut, fadeIn, transition)
}

// game
// ===========================================================================
// fadeLevelAnimation animates the transition between levels.

// Animation to fade a level.  This does both up and down evovle directions
// and does fade ins and fade outs.
type fadeLevelAnimation struct {
	g      *game   // All the state needed to do the fade.
	gstate int     // Will be set for the second of two animations.
	dir    int     // Which way the level is fading (up or down).
	ticks  int     // Animation run rate - number of animation steps.
	tkcnt  int     // Current step.
	start  float64 // Animation start height.
	stop   float64 // The height where the animation stops.
	state  int     // Track progress 0:start, 1:run, 2:done.
	colr   float32 // Amount needed to change colour.
}

// fade in/out the level.
func (f *fadeLevelAnimation) Animate(dt float64) bool {
	switch f.state {
	case 0:
		g, cl := f.g, f.g.cl
		g.state(evolve)
		g.lens = &fly{}
		cl.setHudVisible(false)
		cl.player.reset()
		cl.body.RemBody()
		f.colr = (float32(1) - cl.colour) / float32(f.ticks)
		f.state = 1
		return true
	case 1:
		g, cl := f.g, f.g.cl
		cl.colour += f.colr
		cl.setBackgroundColour(cl.colour)
		cl.scene.MoveView(0, -g.vr*float64(f.dir)/float64(f.ticks), 0)
		g.lens.lookUpDown(cl.scene, float64(f.dir)*(75/float64(f.ticks))*2, g.spin, g.dt)
		if f.tkcnt >= f.ticks {
			f.Wrap()
			return false // animation done.
		}
		f.tkcnt += 1
		return true
	default:
		return false // animation done.
	}
}

// Wrap finishes the fade level animation and sets the player position to a safe
// and stable location.
func (f *fadeLevelAnimation) Wrap() {
	g := f.g
	x, _, z := g.cl.scene.ViewLocation()
	g.cl.scene.SetViewLocation(x, 0.5, z)
	g.cl.scene.SetViewTilt(0)
	g.lens = &fps{}
	g.cl.setHudVisible(true)
	g.cl.body.SetLocation(x, 0.5, z)
	g.cl.body.SetRotation(0, 0, 0, 1)
	g.cl.body.SetBody(vu.Sphere(0.25), 1, 0)
	f.state = 2

	// set the new game state if appropriate.
	if f.gstate == deactivate || f.gstate == activate {
		g.state(f.gstate)
	}
}

// fadeLevelAnimation
// ===========================================================================
// Various game algorithms

// gameMapSize gives the grid size for a given level.
func gameMapSize(lvl int) int { return lvl*6 + 9 }

// gameCcol is the inverse background colour for the center of the given level.
func gameCcol(lvl int) float64 { return float64(lvl+1) * 0.15 }

// gameMuster is the number of sentinels generated for a given level.
var gameMuster = []int{1, 5, 25, 50, 100}

// gameCellGain gives the per-level number of cells gained for each core
// collected.
var gameCellGain = []int{1, 2, 4, 8, 8}

// gameCellLoss gives the per-level number of cells lost for each collision
// with a sentinel. These are multiples of the corresponding cell gains.
var gameCellLoss = []int{1, 12, 24, 48, 64}

// lastSpot is used during debug to return the player to their previous position
// when fly mode is turned off.
type lastSpot struct {
	lx, ly, lz     float64 // location
	dx, dy, dz, dw float64 // direction
	tilt           float64 // up/down.
}
