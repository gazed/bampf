// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"container/list"

	"github.com/gazed/vu"
)

// launch is the application menu/start screen.  It is the first screen after the
// application launches. The start screen allows the user to change options and
// to choose the game difficulty before starting to play.
type launch struct {
	ui         *vu.Ent         // Group of 2D model objects.
	area                       // The launch screen fills up the game window.
	anim       *startAnimation // The start button animation.
	buttons    []*button       // The game select and option screen buttons.
	bg1        *vu.Ent         // Background rotating one way.
	bg2        *vu.Ent         // Background rotating the other way.
	buttonSize int             // Width and height of each button.
	mp         *bampf          // Needed for toggling the option screen.
	evolving   bool            // True when player is moving between levels.
}

// launch implements the screen interface.
func (l *launch) fadeIn() animation        { return nil }
func (l *launch) fadeOut() animation       { return l.newFadeAnimation() }
func (l *launch) resize(width, height int) { l.handleResize(width, height) }
func (l *launch) activate(state int) {
	switch state {
	case screenActive:
		l.anim.scale = 200
		l.ui.Cull(false)
		l.evolving = false
	case screenDeactive:
		l.ui.Cull(true)
		l.evolving = false
	case screenEvolving:
		l.evolving = true
	}
}

// User input to game events. Implements screen interface.
func (l *launch) processInput(in *vu.Input, eventq *list.List) {
	for press, down := range in.Down {
		switch {
		case press == vu.KEsc && down == 1 && !l.evolving:
			publish(eventq, toggleOptions, nil)
		case press == vu.KSpace && down == 1:
			publish(eventq, skipAnim, nil)
		case press == vu.KLm && down == 1:
			for _, btn := range l.buttons {
				if btn.clicked(in.Mx, in.My) {
					publish(eventq, btn.eventID, btn.eventData)
				}
			}
			if l.anim.clicked(in.Mx, in.My) {
				publish(eventq, startGame, nil)
			}
		}
	}

	// handle once per game tick processing.
	l.hover(in)
	l.rotateBackdrop()
	l.anim.rotate(in.Ut, in.Dt)
}

// Process game events. Implements screen interface.
func (l *launch) processEvents(eventq *list.List) (transition int) {
	for e := eventq.Front(); e != nil; e = e.Next() {
		eventq.Remove(e)
		event := e.Value.(*event)
		switch event.id {
		case skipAnim:
			l.mp.skipAnimation()
		case toggleOptions:
			return configGame
		case pickLevel:
			if level, ok := event.data.(int); ok {
				l.mp.launchLevel = level
				l.anim.showLevel(level)
			} else {
				logf("launch.processEvents: did not receive startGame level")
			}
		case startGame:
			return playGame
		}
	}
	return chooseGame
}

// newLaunchScreen creates the start screen. Measurements are 1 pixel == 1 unit
// because the launch screen is done as an overlay.
func newLaunchScreen(mp *bampf) *launch {
	l := &launch{}
	l.mp = mp
	l.ui = mp.eng.AddScene().SetUI()
	l.ui.Cam().SetClip(0, 10)
	l.setSize(mp.eng.State().Screen())
	l.buttonSize = 64

	// create the background.
	l.bg1 = l.ui.AddPart()
	m := l.bg1.MakeModel("textured", "msh:icon", "tex:backdrop")
	m.SetAlpha(0.5).SetUniform("spin", 10.0)
	l.bg2 = l.ui.AddPart()
	m = l.bg2.MakeModel("textured", "msh:icon", "tex:backdrop")
	m.SetAlpha(0.5).SetUniform("spin", -10.0)

	// add the animated start button to the scene.
	l.anim = newStartAnimation(mp, l.ui.AddPart(), l.w, l.h)

	// create the other buttons. Note that the names, eg. "lvl0",
	// are the icon image names.
	buttonPart := l.ui.AddPart()
	sz := int(l.buttonSize)
	l.buttons = []*button{
		newButton(buttonPart, sz, "lvl0", pickLevel, 0),
		newButton(buttonPart, sz, "lvl1", pickLevel, 1),
		newButton(buttonPart, sz, "lvl2", pickLevel, 2),
		newButton(buttonPart, sz, "lvl3", pickLevel, 3),
		newButton(buttonPart, sz, "lvl4", pickLevel, 4),
		newButton(buttonPart, sz, "options", toggleOptions, nil),
	}
	for _, btn := range l.buttons {
		btn.icon.SetScale(1, 1, 0)
	}
	l.layout(0)
	l.handleResize(l.w, l.h)

	// start the button animation.
	l.mp.ani.addAnimation(l.newButtonAnimation())
	l.ui.Cull(true)
	return l
}

// handleResize adjusts the screen to the current window size.
func (l *launch) handleResize(width, height int) {
	l.setSize(0, 0, width, height)
	l.anim.resize(width, height)

	// resize the background to match.
	if l.bg1 != nil {
		size := l.w
		if l.h > size {
			size = l.h
		}
		l.bg1.SetScale(float64(size), float64(size), 1)
		l.bg1.SetAt(float64(l.w/2)-5, float64(l.h/2)-5, 1)
		l.bg2.SetScale(float64(size), float64(size), 1)
		l.bg2.SetAt(float64(l.w/2)-5, float64(l.h/2)-5, 1)
	}
	l.layout(1)
}

// setSize adjusts the start screen dimensions.
func (l *launch) setSize(x, y, width, height int) {
	l.x, l.y, l.w, l.h = 0, 0, width, height
	l.cx, l.cy = l.center()
}

// hover hilites any button the mouse is over.
func (l *launch) hover(i *vu.Input) {
	l.anim.hover(i.Mx, i.My)
	for _, btn := range l.buttons {
		btn.hover(i.Mx, i.My)
	}
}

// layout positions the buttons to the lower-middle part of the screen.
func (l *launch) layout(buttonIndex float64) {
	if len(l.buttons) != 6 {
		logf("start.layout: buttons changed without updating layout.")
		return
	}
	cy := (l.cy - float64(l.h/2) + float64(2*l.buttonSize))
	dx := buttonIndex * 1.15 * float64(l.buttonSize)
	cx := l.cx
	l.buttons[0].position(cx-dx*2, cy)
	l.buttons[1].position(cx-dx, cy)
	l.buttons[2].position(cx, cy)
	l.buttons[3].position(cx+dx, cy)
	l.buttons[4].position(cx+dx*2, cy)
	l.buttons[5].position(cx, cy-float64(l.buttonSize)-10)
}

// rotateBackdrop rotates the start screen backgrounds in opposite
// directions and different speeds.
func (l *launch) rotateBackdrop() {
	l.bg1.Spin(0, 0, 0.2)
	l.bg2.Spin(0, 0, -0.166)
}

// launch
// ===========================================================================
// fadeStartAnimation fades out the start screen.

// newFadeAnimation creates the launch screen fade out animation.
func (l *launch) newFadeAnimation() animation {
	return &fadeStartAnimation{l: l, ticks: 75}
}

// fadeStartAnimation fades out the launch screen when the user starts a game.
type fadeStartAnimation struct {
	l     *launch // Main state needed by the animation.
	ticks int     // Animation run rate - number of animation steps.
	tkcnt int     // Current step.
	state int     // Track progress 0:start, 1:run, 2:done.
}

// Animate fades out the launch screen before transitioning to the first level.
// Note that this changes the transparency on the global "grey" material
// and in the related shader (so set it back when done).
func (f *fadeStartAnimation) Animate(dt float64) bool {
	switch f.state {
	case 0:
		for _, btn := range f.l.buttons {
			btn.setVisible(false)
		}
		f.l.activate(screenEvolving)
		f.l.anim.hilite.SetAlpha(0.0)
		f.state = 1
		return true
	case 1:
		f.l.anim.scale -= 200 / float64(f.ticks)
		alpha := f.l.bg1.Alpha() - float64(0.5)/float64(f.ticks)
		f.l.bg1.SetAlpha(alpha)
		f.l.bg2.SetAlpha(alpha)
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

// Wrap stops the animation and puts the alpha values for the material
// back to what they were (so that others using the same material aren't
// affected).
func (f *fadeStartAnimation) Wrap() {
	f.l.anim.hilite.SetAlpha(0.3)
	f.l.bg1.SetAlpha(0.5)
	f.l.bg2.SetAlpha(0.5)
	f.state = 2
	f.l.activate(screenDeactive)
	for _, btn := range f.l.buttons {
		btn.setVisible(true)
	}
}

// fadeStartAnimation
// ===========================================================================
// buttonAnimation

// buttonAnimation flips the buttons open on the launch screen as the game begins.
type buttonAnimation struct {
	l        *launch // main state needed by the animation.
	state    int     // track progress 0:start, 1:run, 2:done.
	buttonA  float64 // button position animation.
	buttonSc float64 // button original scale animation.
	buttonSx float64 // button scale animation.
	buttonSy float64 // button scale animation.
}

// newButtonAnimation sets the initial conditions for the button animation.
func (l *launch) newButtonAnimation() animation { return &buttonAnimation{l: l} }

// Animate get regular calls to run the start screen animation.
// Float the buttons into position.
func (ba *buttonAnimation) Animate(dt float64) bool {
	switch ba.state {
	case 0:
		ba.buttonSx = 0.1
		ba.buttonSy = 0.1
		ba.buttonSc = float64(ba.l.buttonSize) * 0.5
		ba.l.layout(0)
		ba.state = 1
		return true
	case 1:
		speed := float64(4)
		if ba.buttonSy < 1.0 {
			ba.buttonSy += speed * dt
			for _, btn := range ba.l.buttons {
				sx, _, sz := btn.icon.Scale()
				btn.icon.SetScale(sx, ba.buttonSc*ba.buttonSy, sz)
			}
		} else if ba.buttonA < 1.0 {
			ba.buttonA += speed * dt
			ba.l.layout(ba.buttonA)
		} else if ba.buttonSx < 1.0 {
			ba.buttonSx += speed * dt
			for _, btn := range ba.l.buttons {
				_, sy, sz := btn.icon.Scale()
				btn.icon.SetScale(ba.buttonSc*ba.buttonSx, sy, sz)
			}
		} else {
			ba.Wrap()
			return false // animation done.
		}
		return true
	default:
		return false // animation done.
	}
}

// Wrap stops the button animation and ensures the button scale is exact.
func (ba *buttonAnimation) Wrap() {
	ba.state = 2
	for _, btn := range ba.l.buttons {
		btn.icon.SetScale(ba.buttonSc, ba.buttonSc, 0)
	}
}

// buttonAnimation
// ===========================================================================
// startAnimation - the start-the-game button animation.

// startAnimation shows a rotating cube that is regenerating cells. This is not a
// normal animation as it is also used as the game start button.
type startAnimation struct {
	area            // Start animation acts like a button.
	parent *vu.Ent  // Parent part of the player.
	cx, cy float64  // Center of the area.
	player *trooper // Player can be new or saved.
	hilite *vu.Ent  // Hover overlay.
	scale  float64  // Controls the animation size.
}

// newStartAnimation creates the start screen animation.
func newStartAnimation(mp *bampf, parent *vu.Ent, screenWidth, screenHeight int) *startAnimation {
	sa := &startAnimation{}
	sa.parent = parent
	sa.scale = 200
	sa.hilite = parent.AddPart()
	sa.hilite.MakeModel("colored", "msh:square", "mat:white")
	sa.hilite.Cull(true)
	sa.resize(screenWidth, screenHeight)
	sa.showLevel(0)
	return sa
}

// showLevel changes the animation to match the given user level choice.
func (sa *startAnimation) showLevel(level int) {
	if sa.player != nil {
		sa.player.trash()
	}
	sa.player = newTrooper(sa.parent.AddPart(), level)
	sa.player.part.Spin(15, 0, 0)
	sa.player.part.Spin(0, 0, 15)
	sa.player.setScale(sa.scale)
	sa.player.setLoc(sa.cx, sa.cy, 0)
}

// resize ensures that animation only takes up most of the available area.
func (sa *startAnimation) resize(screenWidth, screenHeight int) {
	sa.x, sa.y = 0, 50
	sa.w, sa.h = screenWidth, screenHeight
	sa.cx, sa.cy = sa.center()
	size := screenWidth
	if size > screenHeight {
		size = screenHeight
	}
	size = 175 // take up most of the available area.
	sa.w, sa.h = size*2, size*2
	sa.x, sa.y = int(sa.cx)-size, int(sa.cy)-size

	// reposition the hover hilite.
	sa.hilite.SetAt(sa.cx, sa.cy, 0)
	sa.hilite.SetScale(float64(size), float64(size), 1)

	// reposition the trooper.
	if sa.player != nil {
		sa.player.setLoc(sa.cx, sa.cy, 0)
	}
}

// clicked is called to see if the start animation was clicked.
func (sa *startAnimation) clicked(mx, my int) bool {
	return mx >= sa.x && mx <= sa.x+sa.w && my >= sa.y && my <= sa.y+sa.h
}

// hover shows the hover part when the mouse is over the start button.
func (sa *startAnimation) hover(mx, my int) {
	sa.hilite.Cull(true)
	if mx >= sa.x && mx <= sa.x+sa.w && my >= sa.y && my <= sa.y+sa.h {
		sa.hilite.Cull(false)
	}
}

// rotate is called each game loop to update the player rotation.
func (sa *startAnimation) rotate(updateTicks uint64, deltaTime float64) {
	spinSpeed := float64(25) // degrees per second.
	sa.player.part.Spin(0, deltaTime*spinSpeed, 0)
	sa.player.setScale(sa.scale)
	sa.player.setLoc(sa.player.loc())

	// regenerate cubes faster as the player gets bigger.
	rate := (sa.player.lvl + 1) * (sa.player.lvl + 1) * 2
	if int(updateTicks)%(100/rate) == 0 {
		sa.player.attach()
	}
}
