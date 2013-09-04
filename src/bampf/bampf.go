// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

// Package bampf is a 3D arcade collection game with random levels.
// Bampf is the sound made when players teleport to safety.
//
// Note that the subdirectories contain resource data for the game.
package main

import (
	"log"
	"vu"
)

// bampf is the main program and initializes various game parts.
// Its resposibilities are:
//   1. start everything up by preparing and sharing the initial state
//      and data structures.
//   2. ensure orderly startup and switching between game states.
type bampf struct {
	eng         *vu.Eng           // Game engine and user input.
	screens     map[string]screen // Available screens (states): start, game, end, options.
	active      screen            // Currently drawn screen (state).
	prior       screen            // Last active screen.  Needed for toggling options.
	focus       bool              // Track if the window has focus
	mute        bool              // Track if the sound is on or off.
	wx, wy      int               // Application window size.
	ani         *animator         // Handles short animations.
	launchLevel int               // Choosen by the user on the launch screen.
	state       func(int)         // Overall application state.
}

// version is set by the build using ld flags. Eg.
//    go build -ldflags "-X main.version `git describe`"
var version string

// main initializes the data structures and starts the game loop.
func main() {
	mp := &bampf{}
	mp.state = mp.launching

	// recover the saved preferences and initialize the engine.
	var err error
	var x, y int
	x, y, mp.wx, mp.wy, mp.mute = mp.prefs()
	if mp.eng, err = vu.New("Bampf", x, y, mp.wx, mp.wy); err != nil {
		log.Printf("Failed to initialize engine %s", err)
		return
	}
	defer mp.eng.Shutdown()
	mp.eng.SetDirector(mp)
	mp.setMute(mp.mute)
	mp.ani = &animator{}
	mp.createScreens()
	mp.state(choose)

	// run the engine until the user decides to quit.
	mp.eng.Action()
}

// createScreens creates the different application screens and anything else
// needed before the render loop takes over.
func (mp *bampf) createScreens() *bampf {
	gameScreen, gameReactions := newGameScreen(mp)
	mp.screens = map[string]screen{
		"launch":  newLaunchScreen(mp),
		"game":    gameScreen,
		"end":     newEndScreen(mp),
		"options": newOptionsScreen(mp, gameReactions),
	}
	mp.eng.Enable(vu.BLEND, true)
	mp.eng.Enable(vu.CULL, true)
	mp.eng.Enable(vu.DEPTH, true)
	mp.eng.Color(1.0, 1.0, 1.0, 1)
	return mp
}

// Overall application state transitions.
const (
	choose = iota // Transition to the choosing state.
	play          // Transition to the playing state.
	done          // Transition to the finished state.
)

// Launching state.
func (mp *bampf) launching(event int) {
	switch event {
	case choose:
		mp.active = mp.screens["launch"]
		mp.active.transition(activate)
		mp.state = mp.choosing
	default:
		log.Printf("launching state: invalid transition %d", event)
	}
}

// Choosing state.
func (mp *bampf) choosing(event int) {
	switch event {
	case play:
		mp.transitionToGameScreen()
		mp.state = mp.playing
	default:
		log.Printf("choosing state: invalid transition %d", event)
	}
}

// Playing state.
func (mp *bampf) playing(event int) {
	switch event {
	case done:
		mp.transitionToEndScreen()
		mp.state = mp.finishing
	case choose:
		mp.returnToMenu()
		mp.state = mp.choosing
	default:
		log.Printf("playing state: invalid transition %d", event)
	}
}

// Finishing state.
func (mp *bampf) finishing(event int) {
	switch event {
	case choose:
		mp.returnToMenu()
		mp.state = mp.choosing
	default:
		log.Printf("finishing state: invalid transition %d", event)
	}
}

// transitionToGameScreen happens when the player chooses play from the
// launchscreen.
func (mp *bampf) transitionToGameScreen() {
	fadeOut := mp.screens["launch"].fadeOut()
	fadeIn := mp.screens["game"].fadeIn()
	transition := func() {
		mp.active = mp.screens["game"]
		mp.active.transition(activate)
	}
	mp.ani.addAnimation(newTransitionAnimation(fadeOut, fadeIn, transition))
}

// transitionToEndScreen happens when the player manages to get through the
// final level.
func (mp *bampf) transitionToEndScreen() {
	fadeOut := mp.screens["game"].fadeOut()
	fadeIn := mp.screens["end"].fadeIn()
	transition := func() {
		mp.active = mp.screens["end"]
		mp.active.transition(activate)
	}
	mp.ani.addAnimation(newTransitionAnimation(fadeOut, fadeIn, transition))
}

// returnToMenu cancels the current game and returns the player
// to the start menu in order to choose a new game.
func (mp *bampf) returnToMenu() {
	mp.prior.transition(deactivate)
	mp.active.transition(deactivate)
	mp.active = mp.screens["launch"]
	mp.active.transition(activate)
}

// toggleOptions shows or hides the options screen.  It can be triggered by
// a start screen button or the "Esc" key.
func (mp *bampf) toggleOptions() {
	if mp.active == mp.screens["options"] {
		mp.active.transition(deactivate)
		mp.active = mp.prior
		mp.active.transition(activate)
	} else {
		mp.active.transition(pause)
		mp.prior = mp.active
		mp.active = mp.screens["options"]
		mp.active.transition(activate)
	}
}

// gameStarted returns true if there is or was a game in progress.
func (mp *bampf) gameStarted() bool {
	return mp.prior == mp.screens["game"] || mp.active == mp.screens["game"] ||
		mp.prior == mp.screens["end"] || mp.active == mp.screens["end"]
}

// Resize is an engine callback and is passed onto all screens.
func (mp *bampf) Resize(x, y, width, height int) {
	mp.eng.ResizeViewport(x, y, width, height)
	mp.wx, mp.wy = width, height
	for _, scr := range mp.screens {
		scr.resize(width, height)
	}
	mp.setWindow(x, y, width, height)
}

// Focus is an engine callback. Updates will only be processed if the window
// has focus.
func (mp *bampf) Focus(focus bool) { mp.focus = focus }

// React is an engine callback and is passed onto the currently active screen.
func (mp *bampf) Update(urges []string, gameTime, deltaTime float32) {
	if mp.focus {
		mp.ani.animate(gameTime, deltaTime) // run active animations
		if mp.active != nil {
			mp.active.update(urges, gameTime, deltaTime)
		}
	}
}

// prefs recovers the saved game preferences.
// Resonable defaults are returned if nothing was persisted.
func (mp *bampf) prefs() (x, y, w, h int, mute bool) {
	x, y, w, h = 400, 100, 800, 600
	saver := newSaver()
	saver.restore()
	mute = saver.Mute
	if saver.X > 0 {
		x = saver.X
	}
	if saver.Y > 0 {
		y = saver.Y
	}
	if saver.W > 0 {
		w = saver.W
	}
	if saver.H > 0 {
		h = saver.H
	}
	return
}

// setWindow persists the window dimensions.
func (mp *bampf) setWindow(x, y, width, height int) {
	saver := newSaver()
	saver.restore()
	saver.persistWindow(x, y, width, height)
}

// setMute turns the game sound off or on and persists the mute setting.
func (mp *bampf) setMute(mute bool) {
	mp.mute = mute
	saver := newSaver()
	saver.persistMute(mp.mute)
	mp.eng.Mute(mp.mute)
}

// bampf
// ===========================================================================
// screen

// screen is used to coordinate which "screen" is currently active. There
// should be one active screen at a time that receives user input.
type screen interface {
	fadeIn() Animation                     // A screen can provide an opening animation.
	fadeOut() Animation                    // A screen can provide a closing animation.
	resize(width, height int)              // Resize the screen.
	transition(stateTransition int)        // Move the screen to a new state.
	update(urges []string, gt, dt float32) // Process user input and run any screen specific animations.
}

// Screen states and state transitions.
const (
	deactivate = iota // Transition to the deactive state.
	activate          // Transition to the active state.
	pause             // Transition to the paused state.
	evolve            // Transition to the evolving state.
)

// CPU or MEM profiling can be turned on by adding a few lines
// in main(). See http://blog.golang.org/profiling-go-programs
//
// Here is a simplistic way to check that memory does not leak by dumping mem stats in the Update loop.
//	  if time.Since(lastDump).Seconds() > 1 {
//	    lastDump = time.Now()
//	    runtime.GC()
//	    runtime.ReadMemStats(mem)
//	    fmt.Printf("Alloc: %10d Heap: %10d Objects: %10d\n", mem.Alloc, mem.HeapAlloc, mem.HeapObjects)
//    }
