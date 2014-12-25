// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"container/list"

	"github.com/gazed/vu"
)

// config is an overlay screen that presents the game options while pausing
// the previous screen. Options can be made active when any of the screens
// are active:
//     start screen : allows the user to map keys.
//     game screen  : allows the user to map keys or quit the level.
//     end screen   : allows the user to map keys or return to the start screen.
type config struct {
	area                     // Options fills up the full screen.
	keys           []string  // Rebindable keys.
	keysRebound    bool      // True if keys were changed.
	scene          vu.Scene  // Scene created at init.
	mp             *bampf    // Main program.
	eng            vu.Engine // 3D engine.
	bg             vu.Part   // Gray out the screen when options are up.
	buttons        []*button // Option buttons.
	buttonSize     int       // Width and height of each button.
	buttonGroup    vu.Part   // Part to group buttons.
	restart        *button   // Quit level button.
	back           *button   // Back to game button.
	info           *button   // Info/credits button.
	mute           *button   // Mute toggle.
	creditList     []vu.Part // The info model.
	exitTransition int       // Transition to use when exiting config.
}

// options implements the screen interface.
func (c *config) fadeIn() animation        { return nil }
func (c *config) fadeOut() animation       { return nil }
func (c *config) resize(width, height int) { c.handleResize(width, height) }
func (c *config) activate(state int) {
	switch state {
	case screenActive:
		c.keysRebound = false
		c.scene.SetVisible(true)
		c.mp.eng.SetLastScene(c.scene)
	case screenDeactive:
		c.scene.SetVisible(false)
	default:
		logf("config state error")
	}
}

// User input to game events. Implements screen interface.
func (c *config) processInput(in *vu.Input, eventq *list.List) {
	overIndex := c.hover(in.Mx, in.My) // per tick processing.
	for press, down := range in.Down {
		switch {
		case press == "Esc" && down == 1:
			publish(eventq, toggleOptions, nil)
		case overIndex >= 0 && down == 1:
			publish(eventq, rebindKey, rebindKeyEvent{index: overIndex, key: press})
		case press == "Lm" && down == 1:
			for _, btn := range c.buttons {
				if btn.clicked(in.Mx, in.My) {
					publish(eventq, btn.eventId, btn.eventData)
				}
			}
			switch {
			case c.mute.clicked(in.Mx, in.My):
				publish(eventq, c.mute.eventId, c.mute.eventData)
			case c.info.clicked(in.Mx, in.My):
				publish(eventq, c.info.eventId, c.info.eventData)
			case c.restart.clicked(in.Mx, in.My):
				publish(eventq, c.restart.eventId, c.restart.eventData)
			case c.back.clicked(in.Mx, in.My):
				publish(eventq, c.back.eventId, c.back.eventData)
			}
		}
	}
}

// Process game events. Implements screen interface.
func (c *config) processEvents(eventq *list.List) (transition int) {
	for e := eventq.Front(); e != nil; e = e.Next() {
		eventq.Remove(e)
		event := e.Value.(*event)
		switch event.id {
		case toggleOptions:
			c.activate(screenDeactive)
			if c.keysRebound {
				saver := newSaver()
				saver.persistBindings(c.keys)
				publish(eventq, keysRebound, c.keys)
			}
			return c.exitTransition
		case rebindKey:
			if rke, ok := event.data.(rebindKeyEvent); ok {
				c.rebindKey(rke.index, rke.key)
			} else {
				logf("options.processEvents: did not receive rebindKeyEvent")
			}
		case quitLevel:
			c.mp.returnToMenu()
			return chooseGame
		case rollCredits:
			c.rollCredits()
		case toggleMute:
			c.toggleMute()
		}

	}
	return configGame
}

// newConfigScreen creates the options screen. It needs the key bindings
// for user actions.
func newConfigScreen(mp *bampf, keys []string) *config {
	c := &config{}
	c.mp = mp
	c.eng = mp.eng
	c.buttonSize = 64
	c.scene = c.eng.AddScene(vu.VO)
	c.scene.Set2D()
	_, _, w, h := c.eng.Size()
	c.handleResize(w, h)
	c.bg = c.scene.AddPart()
	c.bg.SetLocation(float64(c.cx), float64(c.cy), 0)
	c.bg.SetScale(float64(c.w), float64(c.h), 1)
	c.bg.SetRole("flat").SetMesh("square").SetMaterial("tblack")
	c.keys = []string{ // rebindable key defaults.
		"W", // forwards
		"S", // backwards
		"A", // left
		"D", // right
		"C", // cloak
		"T", // teleport
	}
	if len(keys) == len(c.keys) { // override with saved keys.
		c.keys = keys
	}

	// ensure that the game buttons always appear in the same location
	// by mapping reaction ids to button positions.
	c.buttons = make([]*button, len(c.keys))
	c.buttonGroup = c.scene.AddPart()
	c.createButtons()

	// create the non-mappable buttons.
	sz := c.buttonSize
	c.info = newButton(c.buttonGroup, sz/2, "info", rollCredits, nil)
	c.info.position(30, 20) // bottom left corner
	c.mute = newButton(c.buttonGroup, sz/2, "muteoff", toggleMute, nil)
	c.mute.position(70, 20) // bottom left corner
	if c.mp.mute {
		c.mute.setIcon("muteon")
	}
	c.back = newButton(c.buttonGroup, sz/2, "back", toggleOptions, nil)
	c.back.position(float64(c.w-20-c.back.w/2), 20) // bottom right corner
	c.restart = newButton(c.buttonGroup, sz/2, "quit", quitLevel, nil)
	c.restart.position(float64(c.cx), 20) // bottom center of screen.
	c.scene.SetVisible(false)
	return c
}

// handleResize repositions the visible elements when the user resizes the screen.
func (c *config) handleResize(width, height int) {
	c.x, c.y, c.w, c.h = 0, 0, width, height
	c.scene.Cam().SetOrthographic(0, float64(c.w), 0, float64(c.h), 0, 10)
	c.cx, c.cy = c.center()
	if c.bg != nil {
		c.bg.SetScale(float64(c.w), float64(c.h), 1)
		c.bg.SetLocation(float64(c.cx), float64(c.cy), 0)
	}
	c.layout()
}

// createButtons makes the options buttons for mappable actions.
func (c *config) createButtons() {
	sz := c.buttonSize
	c.buttons[0] = newButton(c.buttonGroup, sz, "mForward", 0, nil)
	c.buttons[1] = newButton(c.buttonGroup, sz, "mBack", 0, nil)
	c.buttons[2] = newButton(c.buttonGroup, sz, "mLeft", 0, nil)
	c.buttons[3] = newButton(c.buttonGroup, sz, "mRight", 0, nil)
	c.buttons[4] = newButton(c.buttonGroup, sz, "cloak", 0, nil)
	c.buttons[5] = newButton(c.buttonGroup, sz, "teleport", 0, nil)
	c.labelButtons()
	c.layout()
}

// labelButtons displays the rebindable key associated with the button.
func (c *config) labelButtons() {
	c.buttons[0].label(c.buttonGroup, c.keys[0])
	c.buttons[1].label(c.buttonGroup, c.keys[1])
	c.buttons[2].label(c.buttonGroup, c.keys[2])
	c.buttons[3].label(c.buttonGroup, c.keys[3])
	c.buttons[4].label(c.buttonGroup, c.keys[4])
	c.buttons[5].label(c.buttonGroup, c.keys[5])
}

// layout positions the option screen buttons.
func (c *config) layout() {
	cx1 := c.cx
	cy := c.cy + float64(2*c.buttonSize)
	dy := 1.5 * float64(c.buttonSize)

	// don't panic in the case of programmer error.
	if c.buttons != nil && c.buttons[0] != nil {
		c.buttons[0].position(cx1, cy)         // forward
		c.buttons[2].position(cx1-dy, cy-dy)   // left
		c.buttons[1].position(cx1, cy-dy)      // back
		c.buttons[3].position(cx1+dy, cy-dy)   // right
		c.buttons[4].position(cx1-dy, cy-2*dy) // cloak
		c.buttons[5].position(cx1+dy, cy-2*dy) // teleport
	}
	if c.restart != nil {
		c.restart.position(float64(c.cx), 20) // bottom center of screen.
	}
	if c.back != nil {
		c.back.position(float64(c.w-10-c.back.w/2), 20) // bottom right corner
	}
}

// setExitTransition is called by lost so that closing the options
// screen returns to the screen that it opened from.
func (c *config) setExitTransition(transition int) {
	c.exitTransition = transition
	c.restart.setVisible(c.exitTransition != chooseGame)
}

// rebindKey changes the key for a given reaction. If the newKey is already used,
// then it's reaction is bound to the oldKey. Otherwise the oldKey is dropped.
func (c *config) rebindKey(index int, key string) {
	if key != "Esc" && key != "Sp" && key != "Cmd" && key != "Ctl" &&
		key != "Fn" && key != "Sh" && key != "Alt" {

		// check if the key is already used and swap if necessary.
		swap := -1
		for kcnt, existingKey := range c.keys {
			if key == existingKey {
				swap = kcnt
			}
		}
		if swap >= 0 {
			c.keys[swap] = c.keys[index]
			c.keys[index] = key
			c.buttons[swap].label(c.buttonGroup, c.keys[swap])
		} else {
			c.keys[index] = key
		}
		c.buttons[index].label(c.buttonGroup, c.keys[index])
		c.keysRebound = true
	}
}

// hover hilites any button the mouse is over.
func (c *config) hover(mx, my int) int {
	for cnt, btn := range c.buttons {
		if btn.hover(mx, my) {
			return cnt
		}
	}
	return -1
}

// hide or display game credits.
func (c *config) rollCredits() {
	credits := []string{
		"@galvanizedlogic.com",
		"rust",
		"hymn",
		"jewl",
		"soap",
	}
	info := "Bampf " + version
	credits = append(credits, info)
	if c.creditList == nil {
		c.creditList = []vu.Part{}
		tex := "weblySleek16White"
		height := float64(45)
		for _, credit := range credits {
			banner := c.scene.AddPart()
			banner.SetLocation(20, height, 0)
			banner.SetVisible(true)
			banner.SetRole("uv").AddTex(tex).SetFont("weblySleek16").SetPhrase(credit)
			height += 18
			c.creditList = append(c.creditList, banner)
		}
	} else {
		for _, banner := range c.creditList {
			banner.SetVisible(!banner.Visible())
		}
	}
}

// toggleMute turns the game sound off or on.
func (c *config) toggleMute() {
	c.mp.setMute(!c.mp.mute)
	if c.mp.mute {
		c.mute.setIcon("muteon")
	} else {
		c.mute.setIcon("muteoff")
	}
}
