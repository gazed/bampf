// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"container/list"
	"math"

	"github.com/gazed/vu"
)

// end is the screen that shows the end of game animation. This is a model of
// a silicon atom. No one is expected to get here based on the current game
// difficulty settings.
type end struct {
	scene    *vu.Ent     // 3D scene.
	bg       *vu.Ent     // Background.
	atom     *vu.Ent     // Group the animated atom.
	e1       *vu.Ent     // Up/down electron group.
	e2       *vu.Ent     // Left/right electron group.
	e3       *vu.Ent     // Slash electron group.
	e4       *vu.Ent     // Backslash electron group.
	eles     []*electron // All electrons.
	scale    float64     // Used for the fade in animation.
	fov      float64     // Field of view.
	evolving bool        // Used to disable keys during screen transitions.
}

// Implement the screen interface.
func (e *end) fadeIn() animation        { return e.createFadeIn() }
func (e *end) fadeOut() animation       { return nil }
func (e *end) resize(width, height int) {}
func (e *end) activate(state int) {
	switch state {
	case screenActive:
		e.scene.Cull(false)
		e.evolving = false
	case screenDeactive:
		e.scene.Cull(true)
		e.evolving = false
	case screenEvolving:
		e.scene.Cull(false)
		e.evolving = true
	default:
		logf("end state error")
	}
}

// User input to game events. Implements screen interface.
func (e *end) processInput(in *vu.Input, eventq *list.List) {
	for press, down := range in.Down {
		switch {
		case press == vu.KEsc && down == 1 && !e.evolving:
			publish(eventq, toggleOptions, nil)
		}
	}
}

// Process game events. Implements screen interface.
func (e *end) processEvents(eventq *list.List) (transition int) {
	for ev := eventq.Front(); ev != nil; ev = ev.Next() {
		eventq.Remove(ev)
		event := ev.Value.(*event)
		switch event.id {
		case toggleOptions:
			return configGame
		}
	}
	return finishGame
}

// newEndScreen creates the end game screen.
// Expected to be called once on game startup.
func newEndScreen(mp *bampf, ww, wh int) *end {
	e := &end{}
	e.scale = 0.01
	e.fov = 75
	e.scene = mp.eng.AddScene()
	e.scene.Cam().SetClip(0.1, 50).SetFov(e.fov).SetAt(0, 0, 10)
	e.scene.Cull(true)

	// use a filter effect for the background.
	e.bg = e.scene.AddPart().SetScale(100, 100, 1).SetAt(0, 0, -10)
	m := e.bg.MakeModel("wave", "msh:square", "mat:solid")
	m.SetUniform("screen", 500, 500)

	// create the atom and its electrons.
	e.newAtom()
	return e
}

// createFadeIn returns a new fade-in animation. The initial setup is necessary for
// cases where the user finishes the game and then plays again and finishes again
// all in one application session.
func (e *end) createFadeIn() animation {
	e.scale = 0.01
	e.atom.SetScale(e.scale, e.scale, e.scale)
	return e.newFadeAnimation()
}

// create the silicon atom.
func (e *end) newAtom() {
	e.atom = e.scene.AddPart().SetScale(e.scale, e.scale, e.scale).SetAt(0, 0, 0)

	// rotating image.
	cimg := e.atom.AddPart().SetScale(2, 2, 2)
	model := cimg.MakeModel("spinball", "msh:billboard", "tex:ele", "tex:halo")
	model.Clamp("ele").Clamp("halo")
	model.SetAlpha(0.6)

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
		f.e.bg.SetAlpha(0)
		f.e.scale = 0.01
		f.state = 1
		return true
	case 1:
		f.e.scale += 0.99 / float64(f.ticks)
		f.e.atom.SetScale(f.e.scale, f.e.scale, f.e.scale)
		alpha := f.e.bg.Alpha() + float64(1)/float64(f.ticks)
		f.e.bg.SetAlpha(alpha)
		if f.tkcnt >= f.ticks {
			f.Wrap()
			return false // animation done.
		}
		f.tkcnt++
		return true
	default:
		return false // animation done.
	}
}

// Wrap is called to immediately finish up the animation.
func (f *fadeEndAnimation) Wrap() {
	f.e.bg.SetAlpha(1.0)
	f.e.scale = 1.0
	f.e.atom.SetScale(f.e.scale, f.e.scale, f.e.scale)
	f.e.activate(screenActive)
	f.state = 2
}

// fadeEndAnimation
// ===========================================================================
// electron

// electron is used for the atom electron model instances.
type electron struct {
	core *vu.Ent // 3D model.
	band int     // Electron band.
}

// newElectron creates a new electron model.
func newElectron(root *vu.Ent, band int, angle float64) *electron {
	ele := &electron{}
	ele.band = band
	x, y := ele.initialLocation(angle)
	ele.core = root.AddPart().SetAt(x, y, 0)

	// rotating image.
	cimg := ele.core.AddPart().SetScale(0.25, 0.25, 0.25)
	model := cimg.MakeModel("spinball", "msh:billboard", "tex:ele", "tex:halo")
	model.SetAlpha(0.6)
	return ele
}

// initialLocation positions each electron in the given band and angle.
func (ele *electron) initialLocation(angle float64) (dx, dy float64) {
	dx = float64(float64(ele.band) * math.Cos(angle*math.Pi/180))
	dy = float64(float64(ele.band) * math.Sin(angle*math.Pi/180))
	return
}
