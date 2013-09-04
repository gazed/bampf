// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"math"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"vu"
	"vu/math/lin"
)

// game keeps track of the game play screen. This includes all game levels
// and the heads up overlay.
type game struct {
	mp     *bampf            // Main program.
	eng    *vu.Eng           // Game engine.
	levels map[int]*level    // Game levels.
	cl     *level            // Current level.
	acts   map[string]string // Reactions.
	lens   lens              // Dictates how the camera moves.
	w, h   int               // Window size.
	state  func(int)         // Current screen state.

	// user input handlers for this screen.
	reacts map[string]vu.Reaction // urge to action map
	moveDt float32                // delta amount for moves.

	// debug variables
	debug bool     // game debug switch, see game_debug.go
	last  lastSpot // keeps the last valid player position when debugging.

	// static state.
	run  float32 // run speed.
	spin float32 // spin speed.
	vr   float32 // visible radius.
}

// implement the screen interface.
func (g *game) fadeIn() Animation                     { return g.newStartGameAnimation() }
func (g *game) fadeOut() Animation                    { return g.newEndGameAnimation() }
func (g *game) resize(width, height int)              { g.handleResize(width, height) }
func (g *game) update(urges []string, gt, dt float32) { g.handleUpdate(urges, gt, gt) }
func (g *game) transition(event int)                  { g.state(event) }

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

// Deactive state.
func (g *game) deactive(event int) {
	switch event {
	case activate:
		g.setLevel(g.mp.launchLevel)
		g.cl.setHudVisible(false)
		g.cl.scene.SetViewTilt(75)
		g.cl.scene.SetViewLocation(4, g.vr, 10)
		g.cl.setBackgroundColour(g.cl.colour)
		g.enableKeys()
		g.eng.ShowCursor(false)
		g.state = g.active
	default:
		log.Printf("game: deactive state: invalid transition %d", event)
	}
}

// Active State.
func (g *game) active(event int) {
	switch event {
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
		// ignored. Possible when skipping animations where the animation
		// was skipped before it put the game into evolve state.
	default:
		log.Printf("game: active state: invalid transition %d", event)
		debug.PrintStack()
	}
}

// Paused state.
func (g *game) paused(event int) {
	switch event {
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
}

// Evolving state.
func (g *game) evolving(event int) {
	switch event {
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
}

// disableKeys disallows certain keys when the screen is not active.
func (g *game) disableKeys() { delete(g.reacts, "Esc") }

// enableKeys puts back the keys that were disabled when the screen
// was deactivated.
func (g *game) enableKeys() {
	g.reacts["Esc"] = vu.NewReactOnce("options", func() { g.mp.toggleOptions() })
	if lm, ok := g.reacts["Lm"]; ok {
		lm.SetTime()
	}
}

// handleResize affects all levels, not just the current one.
func (g *game) handleResize(width, height int) {
	g.w, g.h = width, height
	for _, stage := range g.levels {
		stage.resize(width, height)
	}
}

// handleUpdate process the user input.
func (g *game) handleUpdate(urges []string, gt, dt float32) {
	if g.cl == nil { // no current level just yet... still starting.
		return
	}

	// Pre-process user urges. Ensure that simultaneous move requests each only
	// get a portion of the total move amount.
	g.moveDt = g.eng.Dt * 2
	for _, urge := range urges {
		if reaction, ok := g.reacts[urge]; ok {
			rn := reaction.Name()
			if rn == "mForward" || rn == "mBack" || rn == "mLeft" || rn == "mRight" {
				g.moveDt *= 0.5
			}
		}
	}

	// React to all other user urges.
	for _, urge := range urges {
		if reaction, ok := g.reacts[urge]; ok {
			reaction.Do()
			rn := reaction.Name()

			// any player movement may cause a collision so check collision
			// each time. Also limit how far away from the center a player can get.
			if rn == "mForward" || rn == "mBack" || rn == "mLeft" || rn == "mRight" {
				g.cl.checkCollision()
				g.limitWandering(g.cl.scene)
			}
		}
	}

	// update the camera based on the mouse movements each time through the game loop.
	xdiff, ydiff := float32(g.eng.Xm-g.eng.Xp), float32(g.eng.Ym-g.eng.Yp)
	g.lens.look(g.cl.scene, g.spin, g.eng.Dt, xdiff, ydiff)

	// only update game state if the game is active.
	stateName := runtime.FuncForPC(reflect.ValueOf(g.state).Pointer()).Name()
	if strings.Contains(stateName, "active") {
		g.cl.update()   // let the level run methods.
		g.evolveCheck() // check if the player is ready to evolve.
		g.centerMouse() // keep centering the mouse.
	}
}

// centerMouse pops the mouse back to the center of the window, but only
// when the mouse starts to stray to far away.
func (g *game) centerMouse() {
	cx, cy := g.w/2, g.h/2
	if math.Abs(float64(cx-g.eng.Xm)) > 200 || math.Abs(float64(cy-g.eng.Ym)) > 200 {
		g.eng.SetCursorAt(g.w/2, g.h/2)
	}
}

// limitWandering puts a limit on how far the player can get from the center
// of the maze.  This allows the player to feel like they are traveling away
// forever, but they can then return to the maze in very little time.
func (g *game) limitWandering(scene vu.Scene) {
	maxd := g.vr * 3                    // max allowed distance from center.
	cx, _, cz := g.cl.center.Location() // center location
	x, y, z := scene.ViewLocation()     // player location.
	toc := &lin.V3{x - cx, y, z - cz}   // vector to center
	dtoc := toc.Len()                   // distance to center
	if dtoc > maxd {
		toc.Unit()
		toc.Scale(g.vr*3 - 10)
		scene.SetViewLocation(cx+toc.X, y, cz+toc.Z)
	}
}

// reactions are the user input handlers.  These are the default mappings and will
// be used unless overridden by the user in this session or from a persisted key
// mappings from a previous session.
func (g *game) reactions() map[string]vu.Reaction {
	reactions := map[string]vu.Reaction{
		"W":   vu.NewReaction("mForward", func() { g.lens.forward(g.cl.scene, g.eng.Dt, g.run) }),
		"S":   vu.NewReaction("mBack", func() { g.lens.back(g.cl.scene, g.eng.Dt, g.run) }),
		"A":   vu.NewReaction("mLeft", func() { g.lens.left(g.cl.scene, g.eng.Dt, g.run) }),
		"D":   vu.NewReaction("mRight", func() { g.lens.right(g.cl.scene, g.eng.Dt, g.run) }),
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
// the maze. This is the trigger to complete the level.
func (g *game) evolveCheck() {
	if g.cl.playerWorthy() {
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

// healthUpdated is a callback from the trooper whenever its health changes.
// Players that have full health are worthy to descend to the next level
// (they just have to reach the center first).
func (g *game) healthUpdated(health, warn, high int) {
	if health <= 0 {
		if g.cl.num > 0 {
			g.mp.ani.addAnimation(g.newEvolveAnimation(-1))
		}
	}
	if g.cl.playerWorthy() {
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

// game
// ===========================================================================
// fadeLevelAnimation animates the transition between levels.

func (g *game) newStartGameAnimation() Animation {
	return &fadeLevelAnimation{g: g, gstate: activate, dir: 1, ticks: 100, start: g.vr, stop: 0.5}
}
func (g *game) newEndGameAnimation() Animation {
	return &fadeLevelAnimation{g: g, gstate: deactivate, dir: 1, ticks: 100, start: 0.5, stop: -g.vr}
}
func (g *game) newEvolveAnimation(dir int) Animation {
	g.cl.scene.SetViewTilt(0)
	fadeOut := &fadeLevelAnimation{g: g, gstate: -1, dir: dir, ticks: 100, start: 0.5, stop: g.vr * float32(-dir)}
	transition := func() {
		g.setLevel(g.cl.num + dir) // switch to the new level.
		g.cl.setHudVisible(false)
		g.cl.scene.SetViewTilt(75 * float32(dir))
		g.cl.scene.SetViewLocation(4, g.vr*float32(dir), 10)
	}
	fadeIn := &fadeLevelAnimation{g: g, gstate: activate, dir: dir, ticks: 100, start: g.vr * float32(dir), stop: 0.5}
	return newTransitionAnimation(fadeOut, fadeIn, transition)
}

// Animation to fade a level.  This goes in both up and down directions
// and does fade ins and fade outs.
type fadeLevelAnimation struct {
	g      *game   // All the state needed to do the fade.
	gstate int     // Will be set for the second of two animations.
	dir    int     // Which way the level is fading (up or down).
	ticks  int     // Animation run rate - number of animation steps.
	tkcnt  int     // Current step.
	start  float32 // Animation start height.
	stop   float32 // The height where the animation stops.
	state  int     // Track progress 0:start, 1:run, 2:done.
	colr   float32 // Amount needed to change colour.
}

// fade in the level.
func (f *fadeLevelAnimation) Animate(gt, dt float32) bool {
	switch f.state {
	case 0:
		g, cl := f.g, f.g.cl
		g.state(evolve)
		g.lens = &fly{}
		cl.setHudVisible(false)
		cl.player.reset()
		f.colr = (float32(1) - cl.colour) / float32(f.ticks)
		f.state = 1
		return true
	case 1:
		g, cl := f.g, f.g.cl
		cl.colour += f.colr
		cl.setBackgroundColour(cl.colour)
		cl.scene.MoveView(0, -g.vr*float32(f.dir)/float32(f.ticks), 0)
		g.lens.lookUpDown(cl.scene, float32(f.dir)*(75/float32(f.ticks))*2, g.spin, g.eng.Dt)
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
	f.state = 2

	// set the new game state if appropriate.
	if f.gstate == deactivate || f.gstate == activate {
		g.state(f.gstate)
	}
}

// fadeLevelAnimation
// ===========================================================================
// Various game algorithms

//    muster   calculates the number of
//    mapSize  gives the grid size for a given level.
//    ccol     is the inverse background colour for the center of the given level.
func gameMapSize(lvl int) int  { return lvl*6 + 9 }
func gameCcol(lvl int) float64 { return float64(lvl+1) * 0.15 }

// gameMuster is the number of troops generated for a given level.
var gameMuster = []int{1, 5, 25, 50, 100}

// gameCellGain gives the per-level number of cells to gain as each core is
// collected.
var gameCellGain = []int{1, 2, 4, 8, 8}

// gameCellLoss gives the per-level number of cells to lose for each collision
// with a sentinel.  These are multiples of the corresponding cell gains.
var gameCellLoss = []int{1, 12, 24, 48, 64}

// lastSpot is used for debug to return the player to their previous position
// when fly mode is turned off.
type lastSpot struct {
	lx, ly, lz     float32 // location
	dx, dy, dz, dw float32 // direction
	tilt           float32 // up/down.
}
