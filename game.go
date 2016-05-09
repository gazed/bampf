// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"container/list"
	"math"

	"github.com/gazed/vu"
	"github.com/gazed/vu/math/lin"
)

// game keeps track of the game play screen. This includes all game levels
// and the heads up display (hud).
type game struct {
	mp        *bampf          // Main program.
	levels    map[int]*level  // Game levels.
	cl        *level          // Current level.
	dt        float64         // Delta time updated per game tick.
	keys      []int           // Key bindings.
	lens      *cam            // Dictates how the camera moves.
	ww, wh    int             // Window size.
	mxp, myp  int             // Previous mouse locations.
	procDebug func(*vu.Input) // Debugging commands in debug loads.
	evolving  bool            // True when player is moving between levels.
	dir       *lin.Q          // Movement direction.

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
func (g *game) activate(state int) {
	switch state {
	case screenActive:
		g.mp.eng.ShowCursor(false)
		g.cl.setVisible(true)
		g.setKeys(g.keys)
		g.evolving = false
	case screenDeactive:
		g.mp.eng.ShowCursor(true)
		g.cl.setVisible(false)
		g.evolving = false
	case screenPaused:
		g.mp.eng.ShowCursor(true)
	case screenEvolving:
		g.evolving = true
	}
}

// User input to game events. Implements screen interface.
func (g *game) processInput(in *vu.Input, eventq *list.List) {
	if g.cl == nil { // no current level just yet... still starting.
		return
	}

	// update game state if the game is active and not transitioning between levels.
	// Do the evolve check before processing any other input.
	g.spinView(in.Mx, in.My, g.dt)
	if !g.evolving {
		g.lens.update(g.cl.cam) // smooth camera.
		g.cl.update()           // level per-tick updates.
		g.evolveCheck(eventq)   // kick off any necessary level transitions.
	}
	g.centerMouse(in.Mx, in.My) // keep centering the mouse.

	// process any new input.
	g.dt = in.Dt
	for press, down := range in.Down {
		switch {
		case press == vu.K_Esc && down == 1 && !g.evolving:
			publish(eventq, toggleOptions, nil)
		case press == vu.K_Space && down == 1:
			publish(eventq, skipAnim, nil)
		case press == g.keys[0] && !g.evolving: // rebindable keys from here on.
			publish(eventq, goForward, down)
		case press == g.keys[1] && !g.evolving:
			publish(eventq, goBack, down)
		case press == g.keys[2] && !g.evolving:
			publish(eventq, goLeft, down)
		case press == g.keys[3] && !g.evolving:
			publish(eventq, goRight, down)
		case press == g.keys[4] && down == 1 && !g.evolving:
			publish(eventq, cloak, nil)
		case press == g.keys[5] && down == 1 && !g.evolving:
			publish(eventq, teleport, nil)
		}
	}
	g.procDebug(in) // noop method call in production loads.
}

// Process game events. Implements screen interface.
func (g *game) processEvents(eventq *list.List) (transition int) {
	for e := eventq.Front(); e != nil; e = e.Next() {
		eventq.Remove(e)
		event := e.Value.(*event)
		switch event.id {
		case toggleOptions:
			return configGame
		case goForward:
			if dwn, ok := event.data.(int); ok {
				g.goForward(g.dt, dwn)
			} else {
				logf("game.processEvents: did not receive goForward down")
			}
		case goBack:
			if dwn, ok := event.data.(int); ok {
				g.goBack(g.dt, dwn)
			} else {
				logf("game.processEvents: did not receive goBack down")
			}
		case goLeft:
			if dwn, ok := event.data.(int); ok {
				g.goLeft(g.dt, dwn)
			} else {
				logf("game.processEvents: did not receive goLeft down")
			}
		case goRight:
			if dwn, ok := event.data.(int); ok {
				g.goRight(g.dt, dwn)
			} else {
				logf("game.processEvents: did not receive goRight down")
			}
		case cloak:
			g.cl.cloak()
		case teleport:
			g.lens.reset(g.cl.cam)
			g.cl.teleport()
		case keysRebound:
			if keys, ok := event.data.([]int); ok {
				g.setKeys(keys)
			} else {
				logf("game.processEvents: did not receive keysRebound keys")
			}
		case skipAnim:
			g.mp.ani.skip()
		case wonGame:
			g.activate(screenDeactive)
			return finishGame
		}
	}
	return playGame
}

// newGameScreen initializes the gameplay screen.
func newGameScreen(mp *bampf) (scr *game) {
	g := &game{}
	g.mp = mp
	g.lens = &cam{}
	g.ww, g.wh = mp.ww, mp.wh
	g.run = 10  // shared constant
	g.spin = 25 // shared constant
	g.vr = 25   // shared constant
	g.levels = make(map[int]*level)
	g.procDebug = g.setDebugProcessor(g)
	return g
}

// setDebugProcessor checks if the optional processDebugInput method
// is present in the build.
func (g *game) setDebugProcessor(gi interface{}) func(*vu.Input) {
	if fn, ok := gi.(interface {
		processDebugInput(*vu.Input)
	}); ok {
		return fn.processDebugInput // debuggin is on.
	}
	return func(in *vu.Input) {} // debugging is off.
}

// handleResize affects all levels, not just the current one.
func (g *game) handleResize(width, height int) {
	g.ww, g.wh = width, height
	for _, stage := range g.levels {
		stage.resize(width, height)
	}
}

// spinView updates the camera look direction based on amount of mouse movement
// from the previous call.
func (g *game) spinView(mx, my int, dt float64) {
	xdiff, ydiff := float64(mx-g.mxp), float64(my-g.myp)
	g.lens.look(g.spin, dt, xdiff, ydiff)
	g.mxp, g.myp = mx, my
}

// centerMouse pops the mouse back to the center of the window, but only
// when the mouse starts to stray too far away.
func (g *game) centerMouse(mx, my int) {
	cx, cy := g.ww/2, g.wh/2
	if math.Abs(float64(cx-mx)) > 200 || math.Abs(float64(cy-my)) > 200 {
		g.mp.eng.SetCursorAt(g.ww/2, g.wh/2)
		g.mxp, g.myp = cx, cy
	}
}

// limitWandering puts a limit on how far the player can get from the center
// of the level. This allows the player to feel like they are traveling away
// forever, but they can then return to the center in very little time.
func (g *game) limitWandering(down int) {
	maxd := g.vr * 3                           // max allowed distance from center
	cx, _, cz := g.cl.center.Location()        // center location
	x, y, z := g.cl.body.Location()            // player location
	toc := &lin.V3{X: x - cx, Y: y, Z: z - cz} // vector to center
	dtoc := toc.Len()                          // distance to center
	if dtoc > maxd {

		// stop moving forward and move a bit back to center.
		if body := g.cl.body.Body(); body != nil {
			body.Stop()
			body.Rest()
			body.Push(-toc.X/100, 0, -toc.Z/100)
		}
	}
	if down < 0 {
		if body := g.cl.body.Body(); body != nil {
			body.Stop()
			body.Rest()
		}
	}
}

// Player movement handlers.
func (g *game) goForward(dt float64, down int) {
	g.lens.forward(g.cl.body, dt, g.run, g.dir)
	g.limitWandering(down)
}
func (g *game) goBack(dt float64, down int) {
	g.lens.back(g.cl.body, dt, g.run, g.dir)
	g.limitWandering(down)
}
func (g *game) goLeft(dt float64, down int) {
	g.lens.left(g.cl.body, dt, g.run, g.dir)
	g.limitWandering(down)
}
func (g *game) goRight(dt float64, down int) {
	g.lens.right(g.cl.body, dt, g.run, g.dir)
	g.limitWandering(down)
}

// evolveCheck looks for a player at full health that is at the center
// of the level. This is the trigger to complete the level.
func (g *game) evolveCheck(eventq *list.List) {
	if g.cl.isPlayerWorthy() {
		x, y, z := g.cl.cam.Location()
		gridx, gridy := toGrid(x, y, z, float64(g.cl.units))
		if gridx == g.cl.gcx && gridy == g.cl.gcy {
			if g.cl.num < 4 {
				g.mp.ani.addAnimation(g.newEvolveAnimation(1))
			} else if g.cl.num == 4 {
				publish(eventq, wonGame, nil)
			}
		}
	}
}

// healthUpdated is a callback whenever player health changes.
// Players that have full health are worthy to descend to the
// next level, they just have to reach the center first.
func (g *game) healthUpdated(health, warn, high int) {
	if health <= 0 {
		if g.cl.num > 0 {
			g.mp.ani.addAnimation(g.newEvolveAnimation(-1))
		}
	}

	// increase the center block scale when player is ready to evolve.
	if g.cl.isPlayerWorthy() {
		g.cl.center.SetScale(1, 50, 1)
	} else {
		g.cl.center.SetScale(1, 1, 1)
	}
}

// setKeys sets the rebindable keys.
func (g *game) setKeys(keys []int) {
	g.keys = keys
	if g.cl != nil {
		g.cl.updateKeys(g.keys)
	}
}

// setLevel updates to the requested level,
// generating a new level if necessary.
func (g *game) setLevel(lvl int) {
	if g.cl != nil {
		g.cl.deactivate()
	}
	if _, ok := g.levels[lvl]; !ok {
		g.levels[lvl] = newLevel(g, lvl)
	} else {
		g.levels[lvl].player.reset()
	}
	g.cl = g.levels[lvl]
	g.lens.reset(g.cl.cam)
	g.cl.activate(g)
	g.cl.updateKeys(g.keys)
	g.dir = g.cl.cam.Lookxz()
}

// newStartGameAnimation descends to the initial level from
// the launch screen.
func (g *game) newStartGameAnimation() animation {
	return &fadeLevelAnimation{g: g, gameState: screenActive, dir: 1, out: false, ticks: 100}
}

// newEndGameAnimation descends from the final level to the end screen.
func (g *game) newEndGameAnimation() animation {
	return &fadeLevelAnimation{g: g, gameState: screenDeactive, dir: 1, out: true, ticks: 100}
}

// newEvolveAnimation descends or ascends from one game level to another.
func (g *game) newEvolveAnimation(dir int) animation {
	g.activate(screenEvolving)
	fadeOut := &fadeLevelAnimation{g: g, gameState: screenDeactive, dir: dir, out: true, ticks: 100}
	fadeIn := &fadeLevelAnimation{g: g, gameState: screenActive, dir: dir, out: false, ticks: 100}
	transition := func() { g.switchLevel(fadeOut, fadeIn) }
	return newTransitionAnimation(fadeOut, fadeIn, transition)
}

// switchLevel resets any changes to the center of the current level
// and then switches to the next level.
func (g *game) switchLevel(fo, fi *fadeLevelAnimation) {
	g.cl.setBackgroundColour(1)
	g.cl.center.SetScale(1, 1, 1)
	m := g.cl.center.Model()
	m.SetTex(0, "drop1")
	m.SetUniform("spin", 1.0)

	// switch to the new level.
	g.setLevel(g.cl.num + fo.dir)
}

// game
// ===========================================================================
// fadeLevelAnimation animates the transition between levels.

// Animation to fade a level.  This does both up and down evovle directions
// and does fade ins and fade outs.
type fadeLevelAnimation struct {
	g         *game   // All the state needed to do the fade.
	gameState int     // Will be set after finishing the second of two animations.
	dir       int     // Which way the level is fading (up or down).
	out       bool    // true if the level is fading out, false otherwise.
	ticks     int     // Animation run rate - number of animation steps.
	tickCnt   int     // Current step.
	distA     float64 // Animation start height.
	distB     float64 // The height where the animation stops.
	tiltA     float64 // Animation start tilt.
	tiltB     float64 // Animation end tilt.
	state     int     // Track animation progress 0:start, 1:run, 2:done.
	colr      float32 // Amount needed to change colour.
}

// fade in/out the level.
func (f *fadeLevelAnimation) Animate(dt float64) bool {
	switch f.state {
	case 0:
		g := f.g
		g.evolving = true
		g.cl.body.Dispose(vu.BODY)
		x, z := 4.0, 10.0 // standard starting spot.
		if f.out {

			// fading out:
			// start level drop below if dir == 1
			//   cam tilt from 0 to 75
			//   location goes down from 0 to -g.vr.
			// start level and rise if dir == -1
			//   cam tilt from 0 to -75
			//   location goes up from 0 to g.vr.
			f.tiltA, f.tiltB = 0.0, float64(75*f.dir)
			f.distA, f.distB = 0.0, float64(f.dir)*-g.vr
			x, _, z = g.cl.cam.Location() // start from player location.
		} else {

			// fading in:
			// start high and drop to level if dir == 1
			//   cam tilt from -75 to 0
			//   location goes from g.vr down to 0.
			// start low and rise to level if dir == -1
			//   cam tilt from 75 to 0
			//   location goes from -g.vr down to 0.
			f.tiltA, f.tiltB = float64(-75*f.dir), 0.0
			f.distA, f.distB = float64(f.dir)*g.vr, 0.0
		}

		g.lens.pitch = f.tiltA
		g.cl.cam.SetLocation(x, f.distA, z)
		f.colr = (float32(1) - g.cl.colour) / float32(f.ticks)
		g.cl.setVisible(true)
		g.cl.setHudVisible(false)
		f.state = 1
		return true
	case 1:
		g := f.g
		g.cl.colour += f.colr
		g.cl.setBackgroundColour(g.cl.colour)
		move := (f.distB - f.distA) / float64(f.ticks)
		g.cl.cam.Move(0, move, 0, lin.QI)
		tilt := (f.tiltB - f.tiltA) / float64(f.ticks) * 2
		g.lens.pitch = g.lens.updatePitch(g.lens.pitch, tilt, g.spin, g.dt)
		g.cl.cam.SetPitch(g.lens.pitch)
		if f.tickCnt >= f.ticks {
			f.Wrap()
			return false // animation done.
		}
		f.tickCnt += 1
		return true
	default:
		return false // animation done.
	}
}

// Wrap finishes the fade level animation and sets the player position to
// a safe and stable location.
func (f *fadeLevelAnimation) Wrap() {
	g := f.g
	g.lens = &cam{}
	g.cl.setHudVisible(true)
	g.cl.body.NewBody(vu.NewSphere(0.25))
	g.cl.body.SetSolid(1, 0)
	x, _, z := g.cl.cam.Location()
	g.cl.cam.SetLocation(x, 0.5, z)
	g.lens.pitch = 0
	g.cl.cam.SetPitch(g.lens.pitch)
	g.cl.body.SetLocation(x, 0.5, z)
	g.cl.body.SetRotation(lin.QI)

	// set the new game state if appropriate.
	if f.gameState == screenDeactive || f.gameState == screenActive {
		g.activate(f.gameState)
	}
	f.state = 2
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

// lastSpot is used during debug to return the player to their previous
// position when debug fly mode is turned off.
type lastSpot struct {
	lx, ly, lz float64 // location
	// dx, dy, dz, dw float64 // direction
	pitch float64 // up/down.
	yaw   float64 // spin.
}

// calculate a unique id for an x, y coordinate.
func id(x, y, size int) int { return x*size + y }

// get the x, y coordinate for a unique identifier.
func at(id, size int) (x, y int) { return id % size, id / size }
