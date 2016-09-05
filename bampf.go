// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

// Package bampf is a 3D arcade collection game with random levels.
// Bampf is the sound made when players teleport to safety.
//
// The subdirectories contain the game resource data.
package main

// Dev Notes:
// The main purpose of bampf is to test the vu (virtual universe) engine.

import (
	"container/list"
	"math/rand"
	"runtime/debug"
	"time"

	"github.com/gazed/vu"
)

// main recovers saved preferences and initializes the game.
func main() {
	mp := &bampf{}
	var err error
	var x, y int
	x, y, mp.ww, mp.wh, mp.mute = mp.prefs()
	mp.setLogger(mp)
	if err = vu.New(mp, "Bampf", x, y, mp.ww, mp.wh); err != nil {
		logf("Failed to initialize engine %s", err)
		return
	}
	defer catchErrors()
}

// version is set by the build using ld flags. Eg.
//    go build -ldflags "-X main.version `git describe`"
var version string

// catchErrors is for debugging developer loads.
func catchErrors() {
	if r := recover(); r != nil {
		logf("Panic %s: %s Shutting down.", r, debug.Stack())
	}
}

// main
// ===========================================================================
// bampf

// bampf is the main program and initializes various game parts.
// Its resposibilities are:
//   1. Prepare and share the initial state and data structures.
//   2. Ensure orderly switching between game states.
type bampf struct {
	eng         vu.Eng     // Engine.
	state       gameState  // Which main screen is active.
	launch      *launch    // Initial choosing screen.
	game        *game      // Main game play screen.
	end         *end       // Final "you won" screen.
	config      *config    // Options screen.
	active      screen     // Currently drawn screen (state).
	eventq      *list.List // Game event queue.
	mute        bool       // Track if the sound is on or off.
	ww, wh      int        // Application window size.
	ani         *animator  // Handles short animations.
	launchLevel int        // Choosen by the user on the launch screen.
	keys        []int      // Restored key bindings.
}

// Game state transition constants are passed to game state methods which
// result in new game state.
const (
	chooseGame = iota // Transition to the choosing state.
	configGame        // Transition to the options and preferences.
	playGame          // Transition to the playing state.
	finishGame        // Transition to the finished state.
)

// Game state is realized through functions that process game state transitions
type gameState func(int) gameState

// create the game screens before the main action/update loop is started.
func (mp *bampf) Create(eng vu.Eng, s *vu.State) {
	rand.Seed(time.Now().UnixNano())
	mp.eng = eng
	mp.ani = &animator{}
	mp.setMute(mp.mute)
	mp.eventq = list.New()
	mp.createScreens(s.W, s.H)
	mp.state = mp.choosing
	mp.active = mp.launch
	mp.active.activate(screenActive)
	eng.Set(vu.Color(1, 1, 1, 1)) // White as default background.
}

// Update is a regular engine callback and is passed onto the currently
// active screen. Update will run many times a second and should return
// promptly.
func (mp *bampf) Update(eng vu.Eng, in *vu.Input, s *vu.State) {
	if in.Resized {
		mp.resize(s.X, s.Y, s.W, s.H)
	}
	if in.Focus {
		mp.ani.animate(in.Dt)                 // run active animations
		mp.active.processInput(in, mp.eventq) // user input to game events.
		for mp.eventq.Len() > 0 {
			transition := mp.active.processEvents(mp.eventq)
			mp.state = mp.state(transition)
		}
	}
}

// createScreens creates the different application screens and anything
// else needed before the render loop takes over.
func (mp *bampf) createScreens(ww, wh int) *bampf {
	mp.launch = newLaunchScreen(mp)
	mp.game = newGameScreen(mp)
	mp.end = newEndScreen(mp, ww, wh)
	mp.config = newConfigScreen(mp, mp.keys, ww, wh)

	// ensure game has a intial set of keys.
	mp.game.setKeys(mp.keys)
	if len(mp.keys) != len(mp.config.keys) {
		mp.game.setKeys(mp.config.keys)
	}
	return mp
}

// choosing state is where the user is choosing a starting level. The game
// transitions to the playing state once player has made a choice.
func (mp *bampf) choosing(event int) gameState {
	switch event {
	case configGame:
		mp.config.setExitTransition(chooseGame)
		mp.active = mp.config
		mp.active.activate(screenActive)
		return mp.configuring
	case playGame:
		mp.transitionToGameScreen()
		return mp.playing
	case chooseGame:
	default:
		logf("choosing: invalid transition %d", event)
	}
	return mp.choosing
}

// configuring state is where the user is rebinding keys or changing
// game options.
func (mp *bampf) configuring(event int) gameState {
	switch event {
	case chooseGame:
		mp.active = mp.launch
		mp.active.activate(screenActive)
		return mp.choosing
	case playGame:
		mp.active = mp.game
		mp.active.activate(screenActive)
		return mp.playing
	case finishGame:
		mp.active = mp.end
		mp.active.activate(screenActive)
		return mp.finishing
	case configGame:
	default:
		logf("configuring: invalid transition %d", event)
	}
	return mp.configuring
}

// playing state is where the user is working through the game levels.
// The user can complete or cancel the game.
func (mp *bampf) playing(event int) gameState {
	switch event {
	case configGame:
		mp.active.activate(screenPaused)
		mp.config.setExitTransition(playGame)
		mp.active = mp.config
		mp.active.activate(screenActive)
		return mp.configuring
	case finishGame:
		mp.transitionToEndScreen()
		return mp.finishing
	case chooseGame:
	case playGame:
	default:
		logf("playing: invalid transition %d", event)
	}
	return mp.playing
}

// finishing state is where the user has finished the final level.
// The end game animation is displayed. The user has the option of going back
// to the choosing screen and starting again.
func (mp *bampf) finishing(event int) gameState {
	switch event {
	case chooseGame:
		mp.returnToMenu()
		return mp.choosing
	case configGame:
		mp.config.setExitTransition(finishGame)
		mp.active = mp.config
		mp.active.activate(screenActive)
		return mp.configuring
	case finishGame:
	default:
		logf("finishing: invalid transition %d", event)
	}
	return mp.finishing
}

// transitionToGameScreen happens when the player chooses play from the
// launch screen.
func (mp *bampf) transitionToGameScreen() {
	mp.active.activate(screenEvolving)
	fadeOut := mp.launch.fadeOut()
	fadeIn := mp.game.fadeIn()
	mid := func() {
		mp.active = mp.game
		mp.game.setLevel(mp.launchLevel)
		mp.active.activate(screenEvolving)
	}
	mp.ani.addAnimation(newTransitionAnimation(fadeOut, fadeIn, mid))
}

// transitionToEndScreen happens when the player manages to get through the
// final level.
func (mp *bampf) transitionToEndScreen() {
	fadeOut := mp.game.fadeOut()
	fadeIn := mp.end.fadeIn()
	mid := func() {
		mp.active = mp.end
		mp.active.activate(screenEvolving)
	}
	mp.ani.addAnimation(newTransitionAnimation(fadeOut, fadeIn, mid))
}

// returnToMenu cancels the current game and returns the player
// to the start menu in order to choose a new game.
// This is triggered from the game screen.
func (mp *bampf) returnToMenu() {
	mp.config.activate(screenDeactive)
	mp.game.activate(screenDeactive)
	mp.end.activate(screenDeactive)
	mp.active = mp.launch
	mp.active.activate(screenActive)
}

// stopGame transitions to the launching screen.
func (mp *bampf) stopGame(in *vu.Input, down int) { mp.state(chooseGame) }

// skipAnimation is used to short circuit the initial level transition.
func (mp *bampf) skipAnimation() {
	mp.ani.skip()
}

// resize adjusts all the screens to the current game window size.
func (mp *bampf) resize(wx, wy, ww, wh int) {
	mp.ww, mp.wh = ww, wh
	mp.launch.resize(ww, wh)
	mp.game.resize(ww, wh)
	mp.end.resize(ww, wh)
	mp.config.resize(ww, wh)
	mp.setWindow(wx, wy, ww, wh)
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
	mp.keys = append(mp.keys, saver.Kbinds...)
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
	mp.eng.Set(vu.Mute(mp.mute))
}

// bampf
// ===========================================================================
// screen

// screen is used to coordinate which "screen" is currently active. There is
// only ever one active screen receiving user input.
type screen interface {
	resize(width, height int) // Resize the screen.
	fadeIn() animation        // A screen can provide an opening animation.
	fadeOut() animation       // A screen can provide a closing animation.
	activate(state int)       // Move the screen to a new state.

	// Turns user input into game events. Calling it on a screen implies the
	// screen is the active, so inactive screens become active when called.
	processInput(in *vu.Input, eventq *list.List) // User input to game events.

	// Handle the list of game events. The list should be cleared, or the
	// screen should be returning a transition to another screen that will
	// process the events.
	processEvents(eventq *list.List) (transition int) // Process game events.
}

// screen states.
const (
	screenActive   = iota //
	screenDeactive        //
	screenEvolving        //
	screenPaused          // Screen is visible behind config screen.
)

// screen
// ===========================================================================
// utilities

// logf does nothing by default so that log messages are discarded
// during production builds.
var logf = func(format string, v ...interface{}) {}

// setLogger turns logging on in debug loads.
func (mp *bampf) setLogger(gi interface{}) {
	if fn, ok := gi.(interface {
		logger(string, ...interface{})
	}); ok {
		logf = fn.logger
	}
}

// utilities
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
// game events

// Game events.
const (
	_             = iota // start at 1.
	goForward            // Move the player forward.
	goBack               // Move the player back.
	goLeft               // Move the player left.
	goRight              // Move the player right.
	cloak                // Toggle cloaking.
	teleport             // Trigger teleport.
	skipAnim             // Skip any playing animation.
	rollCredits          // Toggle the game developer list.
	toggleMute           // Toggle sound.
	toggleOptions        // Toggle the config screen.
	pickLevel            // expects int data.
	rebindKey            // expects rebindKeyEvent data.
	keysRebound          // expects []string data.
	startGame            // Transition to the game level.
	wonGame              // Transition to the end screen.
	quitLevel            // Transition to the launch screen.
)

// event is the standard structure for all game events.
type event struct {
	id   int         // unique event id.
	data interface{} // nil, value, or struct; depends on the event.
}

// rebindKeyEvent is the data for rebindKey events.
type rebindKeyEvent struct {
	index int
	key   int
}

// publish adds the event to the end of the game event queue.
func publish(eventq *list.List, eventID int, eventData interface{}) {
	eventq.PushBack(&event{id: eventID, data: eventData})
}

// game events
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
