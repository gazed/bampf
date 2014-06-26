// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"math"
	"math/rand"
	"time"
	"vu"
)

// end is the screen that shows the end of game animation. This is a model of
// a silicon atom. No one is expected to get here based on the current game
// difficulty settings.
type end struct {
	mp     *bampf      // The main program (bampf).
	eng    vu.Engine   // The 3D engine.
	scene  vu.Scene    // Group of model objects for the start screen.
	bg     vu.Part     // Background.
	atom   vu.Part     // Group the animated atom.
	eles   []*electron // All electrons.
	e1     vu.Part     // Up/down electron group.
	e2     vu.Part     // Left/right electron group.
	e3     vu.Part     // Slash electron group.
	e4     vu.Part     // Backslash electron group.
	reacts ReactionSet // User input handlers for this screen.
	scale  float64     // Used for the fade in animation.
	state  func(int)   // Current screen state.
	fov    float64     // Field of view.
}

// Implement the screen interface.
func (e *end) fadeIn() animation        { return e.createFadeIn() }
func (e *end) fadeOut() animation       { return nil }
func (e *end) resize(width, height int) { e.handleResize(width, height) }
func (e *end) update(in *vu.Input)      { e.handleUpdate(in) }
func (e *end) transition(event int)     { e.state(event) }

// newEndScreen creates the end game screen. Expected to be called once
// on game startup.
func newEndScreen(mp *bampf) *end {
	e := &end{}
	e.state = e.deactive
	e.mp = mp
	e.eng = mp.eng
	e.scale = 0.01
	e.fov = 75
	e.reacts = NewReactionSet([]Reaction{})
	_, _, w, h := e.eng.Size()
	e.scene = e.eng.AddScene(vu.VP)
	e.scene.SetPerspective(e.fov, float64(w)/float64(h), 0.1, 50)
	e.scene.SetLocation(0, 0, 10)
	e.scene.SetVisible(false)

	// use a filter effect for the background.
	e.bg = e.scene.AddPart().SetLocation(0, 0, -10).SetScale(100, 100, 1)
	e.bg.SetRole("wave").SetMesh("square").SetMaterial("solid")
	e.bg.Role().SetUniform("screen", []float64{500, 500})

	// create the atom and its electrons.
	e.newAtom()
	return e
}

// deactive state waits for activate events.
func (e *end) deactive(event int) {
	switch event {
	case activate:
		e.reacts.Add(Reaction{"end", "Esc", e.mp.toggleOptions})
		e.scene.SetVisible(true)
		e.state = e.active
	default:
		log.Printf("end: deactive state: invalid transition %d", event)
	}
}

// active state waits for pause or deactivate events.
func (e *end) active(event int) {
	switch event {
	case evolve:
	case pause:
		e.reacts.Rem("end")
		e.state = e.paused
	case deactivate:
		e.reacts.Rem("end")
		e.scene.SetVisible(false)
		e.state = e.deactive
	default:
		log.Printf("end: active state: invalid transition %d", event)
	}
}

// paused state waits for activate or deactivate events.
func (e *end) paused(event int) {
	switch event {
	case activate:
		e.reacts.Add(Reaction{"end", "Esc", e.mp.toggleOptions})
		e.state = e.active
	case deactivate:
		e.scene.SetVisible(false)
		e.state = e.deactive
	default:
		log.Printf("end: paused state: invalid transition %d", event)
	}
}

// createFadeIn returns a new fade-in animation. The initial setup is necessary for
// cases where the user finishes the game and then plays again and finishes again
// all in one application session.
func (e *end) createFadeIn() animation {
	e.bg.SetVisible(false)
	e.scale = 0.01
	e.atom.SetScale(e.scale, e.scale, e.scale)
	return e.newFadeAnimation()
}

// handleResize adapts the screen to the new window dimensions.
func (e *end) handleResize(width, height int) {
	ratio := float64(width) / float64(height)
	e.scene.SetPerspective(e.fov, ratio, 0.1, 50)
}

// handleUpdate processes user events.
func (e *end) handleUpdate(in *vu.Input) {
	for press, down := range in.Down {
		e.reacts.Respond(press, in, down)
	}
}

// create the silicon atom.
func (e *end) newAtom() {
	e.atom = e.scene.AddPart()
	e.atom.SetLocation(0, 0, 0)
	e.atom.SetScale(e.scale, e.scale, e.scale)

	cimg := e.atom.AddPart().SetScale(2, 2, 2)
	cimg.SetRole("bbr").SetMesh("billboard").AddTex("atom").SetMaterial("alpha")
	cimg.Role().SetUniform("spin", 1.93)
	cimg.Role().Set2D()

	// same billboard rotating the other way.
	cimg = e.atom.AddPart().SetScale(2, 2, 2)
	cimg.SetRole("bbr").SetMesh("billboard").AddTex("atom").SetMaterial("alpha")
	cimg.Role().SetUniform("spin", -0.7)
	cimg.Role().Set2D()

	// create the electrons.
	e.e1 = e.atom.AddPart()
	e.eles = []*electron{}
	e.eles = append(e.eles, newElectron(e.e1, 2, 90))
	e.eles = append(e.eles, newElectron(e.e1, 3, 90))
	e.eles = append(e.eles, newElectron(e.e1, 4, 90))
	e.eles = append(e.eles, newElectron(e.e1, 2, -90))
	e.eles = append(e.eles, newElectron(e.e1, 3, -90))
	e.eles = append(e.eles, newElectron(e.e1, 4, -90))
	e.e2 = e.atom.AddPart()
	e.eles = append(e.eles, newElectron(e.e2, 3, 0))
	e.eles = append(e.eles, newElectron(e.e2, 4, 0))
	e.eles = append(e.eles, newElectron(e.e2, 3, 180))
	e.eles = append(e.eles, newElectron(e.e2, 4, 180))
	e.e3 = e.atom.AddPart()
	e.eles = append(e.eles, newElectron(e.e3, 3, 45))
	e.eles = append(e.eles, newElectron(e.e3, 3, -135))
	e.e4 = e.atom.AddPart()
	e.eles = append(e.eles, newElectron(e.e4, 3, -45))
	e.eles = append(e.eles, newElectron(e.e4, 3, 135))
}

// newFadeAnimation creates the fade-in to the end screen animation.
func (e *end) newFadeAnimation() animation { return &fadeEndAnimation{e: e, ticks: 75} }

// end
// ===========================================================================
// fadeEndAnimation fades in the end screen.

// fadeEndAnimation fades in the end screen upon game completion.
type fadeEndAnimation struct {
	e     *end // Main state needed by the animation.
	ticks int  // Animation run rate - number of animation steps.
	tkcnt int  // Current step.
	state int  // Track progress 0:start, 1:run, 2:done.
}

// Animate is called each engine update while the animation is running.
func (f *fadeEndAnimation) Animate(dt float64) bool {
	switch f.state {
	case 0:
		f.tkcnt = 0
		f.e.bg.SetVisible(true)
		f.e.bg.Role().SetAlpha(0.0)
		f.e.scale = 0.01
		f.state = 1
		return true
	case 1:
		f.e.scale += 0.99 / float64(f.ticks)
		f.e.atom.SetScale(f.e.scale, f.e.scale, f.e.scale)
		f.e.bg.Role().SetAlpha(f.e.bg.Role().Alpha() + float64(1)/float64(f.ticks))
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
	f.e.bg.Role().SetAlpha(1.0)
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
func newElectron(part vu.Part, band int, angle float64) *electron {
	// combine billboards to get an effect with some movement.
	ele := &electron{}
	ele.band = band
	x, y := ele.initialLocation(angle)
	ele.core = part.AddPart().SetLocation(x, y, 0)

	// get the animations spinning at different speeds.
	random := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	offset := float64(random.Intn(100)) / 75

	// a rotating billboard.
	cimg := ele.core.AddPart().SetScale(0.5, 0.5, 0.5)
	cimg.SetRole("bbr").SetMesh("billboard").AddTex("ele").SetMaterial("alpha")
	cimg.Role().SetUniform("spin", 1.90+offset)
	cimg.Role().Set2D()

	// same billboard rotating the other way.
	cimg = ele.core.AddPart().SetScale(0.5, 0.5, 0.5)
	cimg.SetRole("bbr").SetMesh("billboard").AddTex("ele").SetMaterial("alpha")
	cimg.Role().SetUniform("spin", -0.4-offset)
	cimg.Role().Set2D()
	return ele
}

// initialLocation positions each electron in the given band and angle.
func (ele *electron) initialLocation(angle float64) (dx, dy float64) {
	dx = float64(float64(ele.band) * math.Cos(angle*math.Pi/180))
	dy = float64(float64(ele.band) * math.Sin(angle*math.Pi/180))
	return
}
