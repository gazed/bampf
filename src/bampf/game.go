// Copyright © 2013-2014 Galvanized Logic Inc.
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
	reacts   ReactionSet // User action map
	mxp, myp int         // Previous mouse locations.

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
func (g *game) update(in *vu.Input)      { g.handleUpdate(in) }
func (g *game) transition(event int)     { g.state(event) }

// newGameScreen initializes the gameplay screen.
func newGameScreen(mp *bampf) (scr *game, reacts ReactionSet) {
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
	g.reacts = NewReactionSet([]Reaction{
		{mForward, "W", g.goForward},
		{mBack, "S", g.goBack},
		{mLeft, "A", g.goLeft},
		{mRight, "D", g.goRight},
		{cloak, "C", g.cloak},
		{teleport, "T", g.teleport},
		{"skip", "Sp", g.skipAnimation},
	})
	g.restoreBindings(g.reacts)
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
		g.cl.scene.SetTilt(75)
		g.cl.scene.SetLocation(4, g.vr, 10)
		g.cl.setBackgroundColour(g.cl.colour)
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
func (g *game) disableKeys() {
	g.reacts.Rem("opts")
}

// enableKeys reenables deactivated keys.
func (g *game) enableKeys() {
	g.reacts.Add(Reaction{"opts", "Esc", g.mp.toggleOptions})
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
func (g *game) handleUpdate(in *vu.Input) {
	if g.cl == nil { // no current level just yet... still starting.
		return
	}

	// react to user input.
	shift := ""
	if _, ok := in.Down["Sh"]; ok {
		shift = "Sh-"
	}
	for key, down := range in.Down {
		g.reacts.Respond(shift+key, in, down)
	}

	// update the camera based on the mouse movements each time through the game loop.
	xdiff, ydiff := float64(in.Mx-g.mxp), float64(in.My-g.myp)
	g.lens.look(g.cl.scene, g.spin, in.Dt, xdiff, ydiff)
	g.mxp, g.myp = in.Mx, in.My

	// update game state if the game is active and not transitioning betwen levels.
	if g.state(query) == activate {
		if g.state(query) != evolve {
			g.cl.update()   // level specific updates.
			g.evolveCheck() // check if the player is ready to evolve.
		}
		g.centerMouse(in.Mx, in.My) // keep centering the mouse.
	}
}

// centerMouse pops the mouse back to the center of the window, but only
// when the mouse starts to stray too far away.
func (g *game) centerMouse(mx, my int) {
	cx, cy := g.w/2, g.h/2
	if math.Abs(float64(cx-mx)) > 200 || math.Abs(float64(cy-my)) > 200 {
		g.eng.SetCursorAt(g.w/2, g.h/2)
	}
}

// limitWandering puts a limit on how far the player can get from the center
// of the level. This allows the player to feel like they are traveling away
// forever, but they can then return to the center in very little time.
func (g *game) limitWandering(scene vu.Scene, down int) {
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
	if down < 0 {
		g.cl.body.Stop()
	}
}

// The game handlers.
func (g *game) goForward(in *vu.Input, down int) {
	g.lens.forward(g.cl.body, in.Dt, g.run)
	g.limitWandering(g.cl.scene, down)
}
func (g *game) goBack(in *vu.Input, down int) {
	g.lens.back(g.cl.body, in.Dt, g.run)
	g.limitWandering(g.cl.scene, down)
}
func (g *game) goLeft(in *vu.Input, down int) {
	g.lens.left(g.cl.body, in.Dt, g.run)
	g.limitWandering(g.cl.scene, down)
}
func (g *game) goRight(in *vu.Input, down int) {
	g.lens.right(g.cl.body, in.Dt, g.run)
	g.limitWandering(g.cl.scene, down)
}
func (g *game) cloak(in *vu.Input, down int)    { g.cl.cloak(down) }
func (g *game) teleport(in *vu.Input, down int) { g.cl.teleport(down) }
func (g *game) skipAnimation(in *vu.Input, down int) {
	if down == 1 {
		g.mp.ani.skip()
	}
}

// restoreBindings overwrites the default bindings with saved bindings.
func (g *game) restoreBindings(original ReactionSet) ReactionSet {
	saver := newSaver()
	fromDisk := saver.restore()
	if len(fromDisk.Kmap) > 0 {
		for boundName, mKey := range fromDisk.Kmap {
			original.Rebind(boundName, mKey)
		}
	}
	return original
}

// addDebugReactions checks if the optional debugReactions method is present
// in the build, and adds the extra debug reactions if it is.
func (g *game) addDebugReactions(gi interface{}) {
	if gd, ok := gi.(interface {
		debugReactions() []Reaction
	}); ok {
		for _, val := range gd.debugReactions() {
			g.reacts.Add(val)
		}
	}
}

// evolveCheck looks for a player at full health that is at the center of
// the level. This is the trigger to complete the level.
func (g *game) evolveCheck() {
	if g.cl.isPlayerWorthy() {
		gridx, gridy := g.cl.cc.playerToGrid(g.cl.scene.Location())
		if gridx == g.cl.gcx && gridy == g.cl.gcy {
			if g.cl.num < 4 {
				g.cl.center.SetScale(1, 1, 1)
				g.cl.center.Role().UseTex("drop1", 0)
				g.cl.center.Role().SetUniform("spin", 1.0)
				g.mp.ani.addAnimation(g.newEvolveAnimation(1))
			} else if g.cl.num == 4 {
				g.mp.state(doneGame) // let bampf know that the game is over.
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
		g.cl.center.SetScale(1, 50, 1)
	} else {
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
	g.cl.scene.SetTilt(0)
	fadeOut := &fadeLevelAnimation{g: g, gstate: -1, dir: dir, ticks: 100, start: 0.5, stop: g.vr * float64(-dir)}
	transition := func() {
		g.setLevel(g.cl.num + dir) // switch to the new level.
		g.cl.setHudVisible(false)
		g.cl.scene.SetTilt(75 * float64(dir))
		g.cl.scene.SetLocation(4, g.vr*float64(dir), 10)
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
		cl.scene.Move(0, -g.vr*float64(f.dir)/float64(f.ticks), 0)
		g.lens.lookUpDown(cl.scene, float64(f.dir)*(75/float64(f.ticks))*2, g.spin, dt)
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
	x, _, z := g.cl.scene.Location()
	g.cl.scene.SetLocation(x, 0.5, z)
	g.cl.scene.SetTilt(0)
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
