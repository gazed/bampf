// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

// Animation provides regular callbacks to a specific piece of code.
// An animation is expected to run for a bit and then finish as
// opposed to continuous animations run from the application loop.
type Animation interface {

	// Animate runs animation code for the given animation state. Animate
	// returns true as long as it is running.  By convention the first call
	// to Animate is for initialization purposes and 0 is passed in for both
	// gameTime and deltaTime. The first initialization call is done immediately
	// upon adding the animation, it does not wait for the update loop.
	Animate(gameTime, deltaTime float32) bool

	// Wrap is called to stop an animation and skip to the end state.
	// Generally expected to be used so the user can skip longer or repeated
	// animations.
	Wrap()
}

// animator runs animations.  It keeps track of animations, runs the active
// ones, and discard animations that are finished.
type animator struct {
	animations []Animation
}

// addAnimation adds a new animation to the list active of animations.
func (a *animator) addAnimation(animation Animation) {
	if a.animations == nil {
		a.animations = []Animation{}
	}
	a.animations = append(a.animations, animation)

	// initialize with the first call, don't wait for the update loop.
	animation.Animate(0, 0)
}

// animate runs each of the active animations one step. It is expected to be
// called each update loop.
func (a *animator) animate(gameTime, deltaTime float32) {
	active := []Animation{}
	startA := len(a.animations)
	for _, animation := range a.animations {
		if animation.Animate(gameTime, deltaTime) {
			active = append(active, animation)
		}
	}

	// An animation may have been added during an animation. In this case reset
	// the list on the next pass so as to not lose the added animation.  Any
	// animations that are finished should still be finished.
	if startA == len(a.animations) {
		a.animations = active
	}
}

// skip wraps up any current animations and discards the list of active
// animations.
func (a *animator) skip() {
	for _, animation := range a.animations {
		animation.Wrap()
	}
	a.animations = []Animation{}
}

// ===========================================================================

// transitionAnimation runs an action in-between two animations.  Generally
// used for transitioning between two screens. It is a composite animation
// that acts like a single Animation.
type transitionAnimation struct {
	firstA  Animation // First animation.
	transit func()    // The function to run between the animations.
	lastA   Animation // Second animation.
	state   int       // Track which animation is running.
}

// state constants for transitionAnimation
const (
	runFirst = iota // Running the first animation.
	runLast         // Running the last animation.
)

// newTransitionAnimation creates a composite animation using two animations
// and an action that is run between the two animations.
func newTransitionAnimation(firstA, lastA Animation, action func()) Animation {
	return &transitionAnimation{firstA, action, lastA, runFirst}
}

// Animate runs the animations and the transition action in sequence.
func (ta *transitionAnimation) Animate(gt, dt float32) bool {
	switch ta.state {
	case runFirst:
		if ta.firstA == nil || !ta.firstA.Animate(gt, dt) {
			if ta.transit != nil {
				ta.transit()
			}
			ta.state = runLast
		}
	case runLast:
		if ta.lastA != nil {
			return ta.lastA.Animate(gt, dt)
		}
		return false // finished animaiton.
	}
	return true // keep running.
}

// Wrap forces the animation to the end.  This ensures that both animations
// are wrapped and that the action has been run.
func (ta *transitionAnimation) Wrap() {
	if ta.state == runFirst {
		if ta.firstA != nil {
			ta.firstA.Wrap()
		}
		if ta.transit != nil {
			ta.transit()
		}
		ta.state = runLast
	}
	if ta.state == runLast {
		if ta.lastA != nil {
			ta.lastA.Wrap()
		}
	}
}
