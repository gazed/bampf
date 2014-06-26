// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

// Animations, matching the animation interface, are added to the animator.
// The animator ensures regular callbacks to Animate() ending with a call
// to Wrap().

// Animation provides regular callbacks to motion updaters.
// An animation is expected to run for a bit and then finish as opposed
// to the continuous animations run from the application loop.
type animation interface {

	// Animate is called regularly to control motion. Animate returns
	// true as long as it is running. By convention the first call
	// to Animate is for initialization purposes and deltaTime is 0.
	Animate(deltaTime float64) bool

	// Wrap is called to stop an animation and skip to the end state.
	// Generally expected to be used so the user can skip longer or repeated
	// animations.
	Wrap()
}

// animation
// ===========================================================================
// animator

// animator runs animations.  It keeps track of animations, runs the active
// ones, and discards completed animations.
type animator struct {
	animations []animation
}

// addAnimation adds a new animation to the list active of animations.
// It also calls Animate(0) as the initialization convention.
func (a *animator) addAnimation(ani animation) {
	if a.animations == nil {
		a.animations = []animation{}
	}
	a.animations = append(a.animations, ani)

	// initialize with the first call, don't wait for the update loop.
	ani.Animate(0)
}

// animate runs each of the active animations one step. It is expected to be
// called each update loop.
func (a *animator) animate(deltaTime float64) {
	active := []animation{}
	startA := len(a.animations)
	for _, animation := range a.animations {
		if animation.Animate(deltaTime) {
			active = append(active, animation)
		}
	}

	// Only reset the list if animations have not been added during
	// the Animate() callbacks.
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
	a.animations = []animation{}
}

// animator
// ===========================================================================
// transitionAnimation

// transitionAnimation runs an action in-between two animations. Generally
// used for transitioning between two screens. It is a composite animation
// that acts like a single Animation.
type transitionAnimation struct {
	firstA  animation // First animation.
	transit func()    // The function to run between the animations.
	lastA   animation // Second animation.
	state   int       // Track which animation is running.
}

// state constants for transitionAnimation
const (
	runFirst = iota // Running the first animation.
	runLast         // Running the last animation.
)

// newTransitionAnimation creates a composite animation using two animations
// and an action that is run between the two animations.
func newTransitionAnimation(firstA, lastA animation, action func()) animation {
	return &transitionAnimation{firstA, action, lastA, runFirst}
}

// Animate runs the animations and the transition action in sequence.
func (ta *transitionAnimation) Animate(dt float64) bool {
	switch ta.state {
	case runFirst:
		if ta.firstA == nil || !ta.firstA.Animate(dt) {
			if ta.transit != nil {
				ta.transit()
			}
			ta.state = runLast
		}
	case runLast:
		if ta.lastA != nil {
			return ta.lastA.Animate(dt)
		}
		return false // finished animaiton.
	}
	return true // keep running.
}

// Wrap forces the animation to the end. This ensures that both animations
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
