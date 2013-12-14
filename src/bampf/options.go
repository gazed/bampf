// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"time"
	"vu"
)

// options is an overlay screen that presents the game options while pausing
// the previous screen. Options can be made active when any of the screens
// are active:
//     start screen : allows the user to map keys.
//     game screen  : allows the user to map keys or quit the level.
//     end screen   : allows the user to map keys or return to the start screen.
type options struct {
	area                               // Options fills up the full screen.
	scene       vu.Scene               // Scene created at init.
	mp          *bampf                 // Main program.
	eng         vu.Engine              // 3D engine.
	bg          vu.Part                // Gray out the screen when options are up.
	buttons     []*button              // Option buttons.
	buttonSize  int                    // Width and height of each button.
	blocs       map[string]int         // Button index.
	buttonGroup vu.Part                // Part to group buttons.
	quit        *button                // Quit level button.
	back        *button                // Back to game button.
	info        *button                // Info/credits button.
	mute        *button                // Mute toggle.
	creditList  []vu.Part              // The info model.
	reacts      map[string]vu.Reaction // User input handlers for this screen.
	greacts     map[string]vu.Reaction // User input handlers for the game screen.
	state       func(int)              // Tracks screen state.
	mx, my      int                    // Current mouse locations.

	// slow down user input handling for the options screen.
	last time.Time     // last time a user request was processed.
	hold time.Duration // delay between processing user requests.
}

// options implements the screen interface.
func (o *options) fadeIn() animation        { return nil }
func (o *options) fadeOut() animation       { return nil }
func (o *options) resize(width, height int) { o.handleResize(width, height) }
func (o *options) update(input *vu.Input)   { o.handleUpdate(input) }
func (o *options) transition(event int)     { o.state(event) }

// newOptionsScreen creates the options screen. It needs the map of user actions
// before the game screen becomes active.
func newOptionsScreen(mp *bampf, gameReactions map[string]vu.Reaction) screen {
	o := &options{}
	o.state = o.deactive
	o.mp = mp
	o.eng = mp.eng
	o.buttonSize = 64
	o.last = time.Now()
	o.hold, _ = time.ParseDuration("500ms")
	o.scene = o.eng.AddScene(vu.VO)
	o.scene.Set2D()
	o.eng.SetOverlay(o.scene)
	_, _, w, h := o.eng.Size()
	o.handleResize(w, h)
	o.bg = o.scene.AddPart()
	o.bg.SetFacade("square", "flat").SetMaterial("tblack")
	o.bg.SetScale(float64(o.w), float64(o.h), 1)
	o.bg.SetLocation(float64(o.cx), float64(o.cy), 0)

	// the options screen reacts mostly to mouse clicks.
	o.reacts = map[string]vu.Reaction{
		"Lm":  vu.NewReactOnce("click", func() { o.click(o.mx, o.my) }),
		"Esc": vu.NewReactOnce("options", func() { o.mp.toggleOptions() }),
	}
	o.greacts = gameReactions

	// ensure that the game buttons always appear in the same location
	// by mapping reaction ids to button positions.
	o.blocs = map[string]int{
		"mForward": 0,
		"mBack":    1,
		"mLeft":    2,
		"mRight":   3,
		"cloak":    4,
		"teleport": 5,
	}
	o.buttons = make([]*button, len(o.blocs))
	o.buttonGroup = o.scene.AddPart()
	o.createButtons(o.greacts)

	// create the non-mappable buttons.
	sz := o.buttonSize
	o.info = newButton(o.eng, o.buttonGroup, sz/2, "info", vu.NewReaction("info", func() { o.rollCredits() }))
	o.info.position(20, 20) // bottom left corner
	o.mute = newButton(o.eng, o.buttonGroup, sz/2, "muteoff", vu.NewReaction("mute", func() { o.toggleMute() }))
	o.mute.position(60, 20) // bottom left corner
	if o.mp.mute {
		o.mute.setIcon("muteon")
	}
	o.back = newButton(o.eng, o.buttonGroup, sz/2, "back", vu.NewReaction("back", func() { o.mp.toggleOptions() }))
	o.back.position(float64(o.w-20-o.back.w/2), 20) // bottom right corner
	o.quit = newButton(o.eng, o.buttonGroup, sz/2, "quit", vu.NewReaction("quit", func() { o.mp.state(choose) }))
	o.quit.position(float64(o.cx), 20) // bottom center of screen.
	o.scene.SetVisible(false)
	return o
}

// deactive state waits for the activate event.
func (o *options) deactive(event int) {
	switch event {
	case activate:
		o.reacts["Esc"] = vu.NewReactOnce("options", func() { o.mp.toggleOptions() })
		o.scene.SetVisible(true)
		o.quit.setVisible(o.mp.gameStarted())
		o.state = o.active
	default:
		log.Printf("options: clean state: invalid transition %d", event)
	}
}

// active state waits for the deactivate event.
func (o *options) active(event int) {
	switch event {
	case evolve:
	case deactivate:
		delete(o.reacts, "Esc")
		o.scene.SetVisible(false)
		o.state = o.deactive
	default:
		log.Printf("options: active state: invalid transition %d", event)
	}
}

// handleResize repositions the visible elements when the user resizes the screen.
func (o *options) handleResize(width, height int) {
	o.x, o.y, o.w, o.h = 0, 0, width, height
	o.scene.SetOrthographic(0, float64(o.w), 0, float64(o.h), 0, 10)
	o.cx, o.cy = o.center()
	if o.bg != nil {
		o.bg.SetScale(float64(o.w), float64(o.h), 1)
		o.bg.SetLocation(float64(o.cx), float64(o.cy), 0)
	}
	o.layout()
}

// handleUpdate processes user input.
func (o *options) handleUpdate(input *vu.Input) {
	o.mx, o.my = input.Mx, input.My
	o.hover()
	if len(input.Down) > 0 && o.holdoff() {
		return
	}
	for key, _ := range input.Down {
		// don't allow mapping of reserved keys.
		if key != "Esc" && key != "Sp" {
			for _, btn := range o.buttons {
				if btn.hover(o.mx, o.my) {
					o.rebind(btn.action.Name(), key)
					o.createButtons(o.greacts)
				}
			}
		}
		if reaction, ok := o.reacts[key]; ok {
			reaction.Do()
		}
	}
}

// holdoff prevents user action spamming. Returning true means proceed with
// action. True is only returned if enough time has passed since the last
// true was returned.
func (o *options) holdoff() bool {
	if time.Now().After(o.last.Add(o.hold)) {
		o.last = time.Now()
		return false
	}
	return true
}

// createButtons makes the options buttons for mappable actions.
func (o *options) createButtons(gameReactions map[string]vu.Reaction) {
	sz := o.buttonSize
	for key, reaction := range gameReactions {
		id := reaction.Name()
		if index, ok := o.blocs[id]; ok {
			var b *button
			if b = o.buttons[index]; b == nil {
				b = newButton(o.eng, o.buttonGroup, sz, id, vu.NewReaction(key, func() {}))
				o.buttons[index] = b
			} else {
				b = o.buttons[index]
				b.action = vu.NewReaction(key, func() {})
			}
			b.label(o.eng, o.buttonGroup, key)
		}
	}
	o.layout()
}

// click is called when the user presses a left mouse button.
func (o *options) click(mx, my int) {
	for _, btn := range o.buttons {
		if btn.clicked(mx, my) {
			return // clicking a button results in call to Bind(key)
		}
	}
	if o.mute.clicked(mx, my) || o.info.clicked(mx, my) ||
		o.quit.clicked(mx, my) || o.back.clicked(mx, my) {
		return
	}
}

// layout positions the option screen buttons.
func (o *options) layout() {
	if len(o.buttons) != len(o.blocs) {
		log.Printf("options.layout: forgot to adjust the button layout")
		return
	}
	cx1 := o.cx
	cy := o.cy + float64(2*o.buttonSize)
	dy := 1.5 * float64(o.buttonSize)

	// don't panic in the case of programmer error.
	if o.buttons != nil && o.buttons[0] != nil {
		o.buttons[0].position(cx1, cy)         // forward
		o.buttons[2].position(cx1-dy, cy-dy)   // left
		o.buttons[1].position(cx1, cy-dy)      // back
		o.buttons[3].position(cx1+dy, cy-dy)   // right
		o.buttons[4].position(cx1-dy, cy-2*dy) // cloak
		o.buttons[5].position(cx1+dy, cy-2*dy) // teleport
	}
	if o.quit != nil {
		o.quit.position(float64(o.cx), 20) // bottom center of screen.
	}
	if o.back != nil {
		o.back.position(float64(o.w-20-o.back.w/2), 20) // bottom right corner
	}
}

// rebind changes the key for a given reaction. If the newKey is already used,
// then it's reaction is bound to the oldKey. Otherwise the oldKey is dropped.
func (o *options) rebind(oldKey, newKey string) {
	reactions := o.greacts
	if oldAction, ok := reactions[oldKey]; ok {
		delete(reactions, oldKey)

		// check if the new key was used in the mapping.  If it was then
		// swap reactions with the new key.
		if otherReaction, ok := reactions[newKey]; ok {
			reactions[oldKey] = otherReaction
		}
		reactions[newKey] = oldAction
	}
	o.persistBindings()
	return
}

// hover hilites any button the mouse is over.
func (o *options) hover() {
	for _, btn := range o.buttons {
		btn.hover(o.mx, o.my)
	}
}

// persistBindings ensures that the current key bindings are saved
// across game restarts.
func (o *options) persistBindings() {
	mappedKeys := map[string]string{}
	for boundName, _ := range o.blocs {
		for key, val := range o.greacts {
			if val.Name() == boundName {
				mappedKeys[boundName] = key
			}
		}
	}
	saver := newSaver()
	saver.persistBindings(mappedKeys)
}

// hide or display game credits.
func (o *options) rollCredits() {
	credits := []string{
		"@galvanizedlogic.com",
		"jewl",
		"soap",
		"rust",
	}
	info := "Bampf " + version
	credits = append(credits, info)
	if o.creditList == nil {
		o.creditList = []vu.Part{}
		colour := "weblySleek16White"
		height := float64(45)
		for _, credit := range credits {
			banner := o.scene.AddPart()
			banner.SetBanner(credit, "uv", "weblySleek16", colour)
			banner.SetLocation(5, height, 0)
			banner.SetVisible(true)
			height += 18
			o.creditList = append(o.creditList, banner)
		}
	} else {
		for _, banner := range o.creditList {
			banner.SetVisible(!banner.Visible())
		}
	}
}

// toggleMute turns the game sound off or on.
func (o *options) toggleMute() {
	o.mp.setMute(!o.mp.mute)
	if o.mp.mute {
		o.mute.setIcon("muteon")
	} else {
		o.mute.setIcon("muteoff")
	}
}
