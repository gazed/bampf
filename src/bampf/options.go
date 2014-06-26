// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"vu"
)

// options is an overlay screen that presents the game options while pausing
// the previous screen. Options can be made active when any of the screens
// are active:
//     start screen : allows the user to map keys.
//     game screen  : allows the user to map keys or quit the level.
//     end screen   : allows the user to map keys or return to the start screen.
type options struct {
	area                    // Options fills up the full screen.
	scene       vu.Scene    // Scene created at init.
	mp          *bampf      // Main program.
	eng         vu.Engine   // 3D engine.
	bg          vu.Part     // Gray out the screen when options are up.
	buttons     []*button   // Option buttons.
	buttonSize  int         // Width and height of each button.
	buttonGroup vu.Part     // Part to group buttons.
	quit        *button     // Quit level button.
	back        *button     // Back to game button.
	info        *button     // Info/credits button.
	mute        *button     // Mute toggle.
	creditList  []vu.Part   // The info model.
	reacts      ReactionSet // User input handlers for this screen.
	greacts     ReactionSet // User input handlers for the game screen.
	state       func(int)   // Tracks screen state.
	mx, my      int         // Current mouse locations.
}

// mappable button ids
const (
	mForward   = "mForward"
	mBack      = "mBack"
	mLeft      = "mLeft"
	mRight     = "mRight"
	cloak      = "cloak"
	teleport   = "teleport"
	mForwardId = 0
	mBackId    = 1
	mLeftId    = 2
	mRightId   = 3
	cloakId    = 4
	teleportId = 5
)

//	blocs       map[string]int // Button index.

// options implements the screen interface.
func (o *options) fadeIn() animation        { return nil }
func (o *options) fadeOut() animation       { return nil }
func (o *options) resize(width, height int) { o.handleResize(width, height) }
func (o *options) update(in *vu.Input)      { o.handleUpdate(in) }
func (o *options) transition(event int)     { o.state(event) }

// newOptionsScreen creates the options screen. It needs the map of user actions
// before the game screen becomes active.
func newOptionsScreen(mp *bampf, gameReactions ReactionSet) *options {
	o := &options{}
	o.state = o.deactive
	o.mp = mp
	o.eng = mp.eng
	o.buttonSize = 64
	o.scene = o.eng.AddScene(vu.VO)
	o.scene.Set2D()
	_, _, w, h := o.eng.Size()
	o.handleResize(w, h)
	o.bg = o.scene.AddPart().SetLocation(float64(o.cx), float64(o.cy), 0)
	o.bg.SetScale(float64(o.w), float64(o.h), 1)
	o.bg.SetRole("flat").SetMesh("square").SetMaterial("tblack")

	// the options screen reacts mostly to mouse clicks.
	o.reacts = NewReactionSet([]Reaction{
		{"click", "Lm", func(i *vu.Input, down int) { o.click(i, down) }},
	})
	o.greacts = gameReactions

	// ensure that the game buttons always appear in the same location
	// by mapping reaction ids to button positions.
	o.buttons = make([]*button, teleportId+1)
	o.buttonGroup = o.scene.AddPart()
	o.createButtons()

	// create the non-mappable buttons.
	sz := o.buttonSize
	o.info = newButton(o.buttonGroup, sz/2, "info", o.rollCredits)
	o.info.position(30, 20) // bottom left corner
	o.mute = newButton(o.buttonGroup, sz/2, "muteoff", o.toggleMute)
	o.mute.position(70, 20) // bottom left corner
	if o.mp.mute {
		o.mute.setIcon("muteon")
	}
	o.back = newButton(o.buttonGroup, sz/2, "back", o.mp.toggleOptions)
	o.back.position(float64(o.w-20-o.back.w/2), 20) // bottom right corner
	o.quit = newButton(o.buttonGroup, sz/2, "quit", o.mp.stopGame)
	o.quit.position(float64(o.cx), 20) // bottom center of screen.
	mp.eng.SetLastScene(o.scene)
	o.scene.SetVisible(false)
	return o
}

// deactive state waits for the activate event.
func (o *options) deactive(event int) {
	switch event {
	case activate:
		o.reacts.Add(Reaction{"options", "Esc", o.mp.toggleOptions})
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
		o.reacts.Rem("options")
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
func (o *options) handleUpdate(in *vu.Input) {
	o.mx, o.my = in.Mx, in.My
	o.hover()
	for key, down := range in.Down {
		if down == 1 { // ignore key repeats.

			// don't allow mapping of reserved keys.
			if key != "Esc" && key != "Sp" {
				for _, btn := range o.buttons {
					if btn.hover(o.mx, o.my) {
						o.rebind(o.greacts.Key(btn.id), key)
						o.labelButtons()
					}
				}
			}
			o.reacts.Respond(key, in, down)
		}
	}
}

// createButtons makes the options buttons for mappable actions.
func (o *options) createButtons() {
	sz := o.buttonSize
	o.buttons[mForwardId] = newButton(o.buttonGroup, sz, mForward, func(in *vu.Input, down int) {})
	o.buttons[mBackId] = newButton(o.buttonGroup, sz, mBack, func(in *vu.Input, down int) {})
	o.buttons[mLeftId] = newButton(o.buttonGroup, sz, mLeft, func(in *vu.Input, down int) {})
	o.buttons[mRightId] = newButton(o.buttonGroup, sz, mRight, func(in *vu.Input, down int) {})
	o.buttons[cloakId] = newButton(o.buttonGroup, sz, cloak, func(in *vu.Input, down int) {})
	o.buttons[teleportId] = newButton(o.buttonGroup, sz, teleport, func(in *vu.Input, down int) {})
	o.labelButtons()
	o.layout()
}

func (o *options) labelButtons() {
	o.buttons[mForwardId].label(o.buttonGroup, o.greacts.Key(mForward))
	o.buttons[mBackId].label(o.buttonGroup, o.greacts.Key(mBack))
	o.buttons[mLeftId].label(o.buttonGroup, o.greacts.Key(mLeft))
	o.buttons[mRightId].label(o.buttonGroup, o.greacts.Key(mRight))
	o.buttons[cloakId].label(o.buttonGroup, o.greacts.Key(cloak))
	o.buttons[teleportId].label(o.buttonGroup, o.greacts.Key(teleport))
}

// click is called when the user presses a left mouse button.
func (o *options) click(i *vu.Input, down int) {
	if down == 1 {
		for _, btn := range o.buttons {
			if btn.clicked(i, down) {
				return // clicking a button results in call to Bind(key)
			}
		}
		if o.mute.clicked(i, down) || o.info.clicked(i, down) ||
			o.quit.clicked(i, down) || o.back.clicked(i, down) {
			return
		}
	}
}

// layout positions the option screen buttons.
func (o *options) layout() {
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
		o.back.position(float64(o.w-10-o.back.w/2), 20) // bottom right corner
	}
}

// rebind changes the key for a given reaction. If the newKey is already used,
// then it's reaction is bound to the oldKey. Otherwise the oldKey is dropped.
func (o *options) rebind(oldKey, newKey string) {
	if id := o.greacts.Id(oldKey); id != "" {
		o.greacts.Rebind(id, newKey)
		o.persistBindings()
	} else {
		println("rebind could not find id for key", oldKey)
	}
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
	mappedKeys[mForward] = o.greacts.Key(mForward)
	mappedKeys[mBack] = o.greacts.Key(mBack)
	mappedKeys[mLeft] = o.greacts.Key(mLeft)
	mappedKeys[mRight] = o.greacts.Key(mRight)
	mappedKeys[cloak] = o.greacts.Key(cloak)
	mappedKeys[teleport] = o.greacts.Key(teleport)
	saver := newSaver()
	saver.persistBindings(mappedKeys)
}

// hide or display game credits.
func (o *options) rollCredits(in *vu.Input, down int) {
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
		tex := "weblySleek16White"
		height := float64(45)
		for _, credit := range credits {
			banner := o.scene.AddPart().SetLocation(20, height, 0)
			banner.SetVisible(true)
			banner.SetRole("uv").AddTex(tex).SetFont("weblySleek16").SetPhrase(credit)
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
func (o *options) toggleMute(in *vu.Input, down int) {
	o.mp.setMute(!o.mp.mute)
	if o.mp.mute {
		o.mute.setIcon("muteon")
	} else {
		o.mute.setIcon("muteoff")
	}
}
