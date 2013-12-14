// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

// Package bampf is a 3D arcade collection game with random levels.
// Bampf is the sound made when players teleport to safety.
//
// The subdirectories contain the game resource data.
package main

// The main purpose of bampf is to test the vu (virtual universe) engine.

import (
	"log"
	"runtime/debug"
	"vu"
)

// main initializes the data structures and the game engine.
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

	// SetDirector registers bampf and results in an engine callback to Create().
	mp.eng.SetDirector(mp)
}

// version is set by the build using ld flags. Eg.
//    go build -ldflags "-X main.version `git describe`"
var version string

// main
// ===========================================================================
// bampf

// bampf is the main program and initializes various game parts.
// Its resposibilities are:
//   1. Prepare and share the initial state and data structures.
//   2. Ensure orderly switching between game states.
type bampf struct {
	eng         vu.Engine         // Game engine and user input.
	state       func(int)         // Overall application state.
	screens     map[string]screen // Available screens (states).
	active      screen            // Currently drawn screen (state).
	prior       screen            // Last active screen. Needed for toggling options.
	mute        bool              // Track if the sound is on or off.
	wx, wy      int               // Application window size.
	ani         *animator         // Handles short animations.
	launchLevel int               // Choosen by the user on the launch screen.
}

// Overall application state transitions. These are used as input
// parameters to the bampf.state methods.
const (
	choose = iota // Transition to the choosing state.
	play          // Transition to the playing state.
	done          // Transition to the finished state.
)

// Create is run after engine intialization. The initial game screens are
// created and the main action/update loop is started. All subsequent calls
// from the engine will be to Update().
func (mp *bampf) Create(eng vu.Engine) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic %s: %s Shutting down.", r, debug.Stack())
		}
	}()
	mp.ani = &animator{}
	mp.setMute(mp.mute)
	mp.createScreens()
	mp.state(choose)
	mp.eng.Action() // run the engine until the user decides to quit.
}

// Update is a regular engine callback and is passed onto the currently
// active screen. Update will run many times a second and should return
// promptly.
func (mp *bampf) Update(input *vu.Input) {
	if input.Resized {
		mp.resize()
	}
	if input.Focus {
		mp.ani.animate(input.Dt) // run active animations
		if mp.active != nil {
			mp.active.update(input)
		}
	}
}

// createScreens creates the different application screens and anything else
// needed before the render loop takes over. There is a dependency between
// screens that need key-bindings that means the game screen must be
// created first.
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

// launching state is the first game state with a single transition to the
// initial game screen where the user chooses the starting level.
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

// choosing state is where the user is choosing a starting level. The game
// transitions to the playing state once player has made a choice.
func (mp *bampf) choosing(event int) {
	switch event {
	case play:
		mp.transitionToGameScreen()
		mp.state = mp.playing
	default:
		log.Printf("choosing state: invalid transition %d", event)
	}
}

// playing state is where the user is working through the game levels.
// The user can complete or cancel the game.
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

// finishing state is where the user has finished the final level.
// The end game animation is displayed. The user has the option of going back
// to the choosing screen and starting again.
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
// launch screen.
func (mp *bampf) transitionToGameScreen() {
	fadeOut := mp.screens["launch"].fadeOut()
	fadeIn := mp.screens["game"].fadeIn()
	transition := func() {
		mp.active = mp.screens["game"]
		mp.active.transition(evolve)
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

// toggleOptions shows or hides the options screen.
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

// resize adjusts all the screens to the current game window size.
func (mp *bampf) resize() {
	x, y, w, h := mp.eng.Size()
	mp.eng.Resize(x, y, w, h)
	mp.wx, mp.wy = w, h
	for _, scr := range mp.screens {
		scr.resize(w, h)
	}
	mp.setWindow(x, y, w, h)
}

// prefs recovers the saved game preferences.
// Resonable defaults are returned if no saved information was found.
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

// setWindow saves the window dimensions.
func (mp *bampf) setWindow(x, y, width, height int) {
	saver := newSaver()
	saver.restore()
	saver.persistWindow(x, y, width, height)
}

// setMute turns the game sound off or on and saves the mute setting.
func (mp *bampf) setMute(mute bool) {
	mp.mute = mute
	saver := newSaver()
	saver.persistMute(mp.mute)
	mp.eng.Mute(mp.mute)
}

// bampf
// ===========================================================================
// screen

// screen is used to coordinate which "screen" is currently active. There is
// only ever one active screen receiving user input.
type screen interface {
	fadeIn() animation              // A screen can provide an opening animation.
	fadeOut() animation             // A screen can provide a closing animation.
	resize(width, height int)       // Resize the screen.
	transition(stateTransition int) // Move the screen to a new state.
	update(input *vu.Input)         // Process user input and run any screen specific animations.
}

// Screen states and state transitions.
const (
	deactivate = iota // Transition to the deactive state.
	activate          // Transition to the active state.
	pause             // Transition to the paused state.
	evolve            // Transition to the evolving state.
	query             // Query existing state.
)

// screen
// ===========================================================================
// area

// area describes a 2D part of a screen. It is the base class for sections
// of the HUD and buttons.
type area struct {
	x, y   int     // bottom left corner.
	w, h   int     // width and height.
	cx, cy float64 // area center location.
}

// center calculates the center of the given area.
func (a *area) center() (cx, cy float64) {
	cx = float64(a.x + a.w/2)
	cy = float64(a.y + a.h/2)
	return
}

// area
// ===========================================================================

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
