// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"math"
	"math/rand"
	"time"
	"vu"
)

// end is the screen that shows the end of game animation.  This is a model of
// a silicon atom. No one is expected to get here based on the current game
// difficulty settings.
type end struct {
	mp     *bampf                 // The main program (bampf).
	eng    *vu.Eng                // The 3D engine.
	scene  vu.Scene               // Group of model objects for the start screen.
	bg     vu.Part                // Background.
	atom   vu.Part                // Group the animated atom.
	eles   []*electron            // All electrons.
	e1     vu.Part                // Up/down electron group.
	e2     vu.Part                // Left/right electron group.
	e3     vu.Part                // Slash electron group.
	e4     vu.Part                // Backslash electron group.
	reacts map[string]vu.Reaction // User input handlers for this screen.
	scale  float32                // Used for the fade in animation.
	state  func(int)              // Current screen state.
}

// Implement the screen interface.
func (e *end) fadeIn() Animation                     { return e.createFadeIn() }
func (e *end) fadeOut() Animation                    { return nil }
func (e *end) resize(width, height int)              { e.handleResize(width, height) }
func (e *end) update(urges []string, gt, dt float32) { e.handleUpdate(urges, gt, dt) }
func (e *end) transition(event int)                  { e.state(event) }

// newEndScreen creates the end game screen.
func newEndScreen(mp *bampf) screen {
	e := &end{}
	e.state = e.deactive
	e.mp = mp
	e.eng = mp.eng
	e.scale = 0.01
	e.reacts = map[string]vu.Reaction{}
	_, _, w, h := e.eng.Size()
	e.scene = e.eng.AddScene(vu.VP)
	e.scene.SetPerspective(75, float32(w)/float32(h), 0.1, 50)
	e.scene.SetViewLocation(0, 0, 10)
	e.scene.SetVisibleRadius(250)
	e.scene.SetVisible(false)

	// use a filter effect for the background.
	e.bg = e.scene.AddPart()
	e.bg.SetFacade("square", "wave", "solid")
	e.bg.SetScale(100, 100, 1)
	e.bg.SetLocation(0, 0, -10)

	// create the atom and its electrons.
	e.newAtom()
	return e
}

// Deactive state.
func (e *end) deactive(event int) {
	switch event {
	case activate:
		e.reacts["Esc"] = vu.NewReactOnce("end", func() { e.mp.toggleOptions() })
		e.scene.SetVisible(true)
		e.state = e.active
	default:
		log.Printf("end: deactive state: invalid transition %d", event)
	}
}

// Active state.
func (e *end) active(event int) {
	switch event {
	case evolve:
	case pause:
		delete(e.reacts, "Esc")
		e.state = e.paused
	case deactivate:
		delete(e.reacts, "Esc")
		e.scene.SetVisible(false)
		e.state = e.deactive
	default:
		log.Printf("end: active state: invalid transition %d", event)
	}
}

// Paused state.
func (e *end) paused(event int) {
	switch event {
	case activate:
		e.reacts["Esc"] = vu.NewReactOnce("end", func() { e.mp.toggleOptions() })
		e.state = e.active
	case deactivate:
		e.scene.SetVisible(false)
		e.state = e.deactive
	default:
		log.Printf("end: paused state: invalid transition %d", event)
	}
}

// createFadeIn makes the fade in animation. The initial setup is necessary for
// cases where the user finishes the game and then plays again and finishes again
// all in one application session.
func (e *end) createFadeIn() Animation {
	e.bg.SetVisible(false)
	e.scale = 0.01
	e.atom.SetScale(e.scale, e.scale, e.scale)
	return e.newFadeAnimation()
}

// handleResize adapts the screen to the new window dimensions.
func (e *end) handleResize(width, height int) {
	ratio := float32(width) / float32(height)
	e.scene.SetPerspective(75, ratio, 0.1, 50)
}

// handleUpdate processes user events.
func (e *end) handleUpdate(urges []string, gt, dt float32) {
	for _, urge := range urges {
		if reaction, ok := e.reacts[urge]; ok {
			reaction.Do()
		}
	}
}

// create the atom.
func (e *end) newAtom() {
	e.atom = e.scene.AddPart()
	e.atom.SetCullable(false)
	e.atom.SetLocation(0, 0, 0)
	e.atom.SetScale(e.scale, e.scale, e.scale)

	cimg := e.atom.AddPart()
	cimg.SetFacade("billboard", "bbra", "alpha")
	cimg.SetScale(2, 2, 2)
	cimg.SetTexture("atom", 1.93)
	cimg.SetCullable(false)

	// same billboard rotating the other way.
	cimg = e.atom.AddPart()
	cimg.SetFacade("billboard", "bbra", "alpha")
	cimg.SetScale(2, 2, 2)
	cimg.SetTexture("atom", -1.3)
	cimg.SetCullable(false)

	// create the electrons.
	e.e1 = e.atom.AddPart()
	e.eles = []*electron{}
	e.eles = append(e.eles, newElectron(e.eng, e.e1, 2, 90))
	e.eles = append(e.eles, newElectron(e.eng, e.e1, 3, 90))
	e.eles = append(e.eles, newElectron(e.eng, e.e1, 4, 90))
	e.eles = append(e.eles, newElectron(e.eng, e.e1, 2, -90))
	e.eles = append(e.eles, newElectron(e.eng, e.e1, 3, -90))
	e.eles = append(e.eles, newElectron(e.eng, e.e1, 4, -90))
	e.e2 = e.atom.AddPart()
	e.eles = append(e.eles, newElectron(e.eng, e.e2, 3, 0))
	e.eles = append(e.eles, newElectron(e.eng, e.e2, 4, 0))
	e.eles = append(e.eles, newElectron(e.eng, e.e2, 3, 180))
	e.eles = append(e.eles, newElectron(e.eng, e.e2, 4, 180))
	e.e3 = e.atom.AddPart()
	e.eles = append(e.eles, newElectron(e.eng, e.e3, 3, 45))
	e.eles = append(e.eles, newElectron(e.eng, e.e3, 3, -135))
	e.e4 = e.atom.AddPart()
	e.eles = append(e.eles, newElectron(e.eng, e.e4, 3, -45))
	e.eles = append(e.eles, newElectron(e.eng, e.e4, 3, 135))
}

// end
// ===========================================================================
// fadeEndAnimation fades in the end screen.

// newFadeAnimation creates the fade-in to the end screen animation.
func (e *end) newFadeAnimation() Animation { return &fadeEndAnimation{e: e, ticks: 75} }

// fadeEndAnimation fades in the end screen when the game has been completed.
type fadeEndAnimation struct {
	e     *end // Main state needed by the animation.
	ticks int  // Animation run rate - number of animation steps.
	tkcnt int  // Current step.
	state int  // Track progress 0:start, 1:run, 2:done.
}

// Animate is called each update while the animation is running.
func (f *fadeEndAnimation) Animate(gt, dt float32) bool {
	switch f.state {
	case 0:
		f.tkcnt = 0
		f.e.bg.SetVisible(true)
		f.e.bg.SetAlpha(0)
		f.e.scale = 0.01
		f.state = 1
		return true
	case 1:
		f.e.scale += 0.99 / float32(f.ticks)
		f.e.atom.SetScale(f.e.scale, f.e.scale, f.e.scale)
		f.e.bg.SetAlpha(f.e.bg.Alpha() + float32(1)/float32(f.ticks))
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

// Wrap is called to immediately finish up the animation.
func (f *fadeEndAnimation) Wrap() {
	f.e.bg.SetVisible(true)
	f.e.bg.SetAlpha(1)
	f.e.scale = 1.0
	f.e.atom.SetScale(f.e.scale, f.e.scale, f.e.scale)
	f.state = 2
}

// fadeEndAnimation
// ===========================================================================
// electron

// electron is used for the atom electron model instances.
type electron struct {
	core vu.Part // 3D model.
	band int     // Electron band.
}

// newElectron creates a new electron model.
func newElectron(eng *vu.Eng, part vu.Part, band int, angle float64) *electron {
	// combine billboards to get an effect with some movement.
	ele := &electron{}
	ele.band = band
	ele.core = part.AddPart()
	ele.core.SetCullable(false)
	x, y := ele.initialLocation(angle)
	ele.core.SetLocation(x, y, 0)

	// get the animations spinning at different speeds.
	random := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	offset := float32(random.Intn(100)) / 75

	// a rotating billboard.
	cimg := ele.core.AddPart()
	cimg.SetFacade("billboard", "bbra", "alpha")
	cimg.SetScale(0.5, 0.5, 0.5)
	cimg.SetTexture("ele", 1.90+offset)
	cimg.SetCullable(false)

	// same billboard rotating the other way.
	cimg = ele.core.AddPart()
	cimg.SetFacade("billboard", "bbra", "alpha")
	cimg.SetScale(0.5, 0.5, 0.5)
	cimg.SetTexture("ele", -1.1-offset)
	cimg.SetCullable(false)
	return ele
}

// initialLocation positions each electron in the given band and angle.
func (ele *electron) initialLocation(angle float64) (dx, dy float32) {
	dx = float32(float64(ele.band) * math.Cos(angle*math.Pi/180))
	dy = float32(float64(ele.band) * math.Sin(angle*math.Pi/180))
	return
}
