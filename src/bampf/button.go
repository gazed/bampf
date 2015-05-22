// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"github.com/gazed/vu"
)

// button has both an image/icon and an action. The action can be linked to a key.
// Both a button image, and corresponding action key, if applicable, are shown on
// the button.
type button struct {
	area                  // Button is rectangular.
	id        string      // Button unique name.
	eventId   int         // game event identifier.
	eventData interface{} // game event data.
	icon      vu.Pov      // Button image.
	hilite    vu.Pov      // Hover overlay.
	banner    vu.Pov      // Label for the action associated with the button.
	cx, cy    float64     // Button center location.
	model     vu.Pov      // Holds button 3D model. Used for transforms.
}

// newButton creates a button. Buttons are initialized with a size and repositioned later.
//   root   is the parent transform.
//   size   is both the width and height.
//   icon   is the (already loaded) texture image.
//   action is the action to perform when the button is pressed.
func newButton(root vu.Pov, size int, icon string, eventId int, eventData interface{}) *button {
	btn := &button{}
	btn.model = root.NewPov()
	btn.eventId = eventId
	btn.eventData = eventData
	btn.w, btn.h = size, size

	// create the button icon.
	btn.id = icon
	btn.icon = btn.model.NewPov().SetScale(float64(btn.w/2), float64(btn.h/2), 1)
	btn.icon.NewModel("uv").LoadMesh("icon").AddTex(icon).SetAlpha(0.5)

	// create a hilite that is only shown on mouse over.
	btn.hilite = btn.model.NewPov().SetScale(float64(btn.w/2.0), float64(btn.h/2.0), 1)
	btn.hilite.SetVisible(false)
	btn.hilite.NewModel("alpha").LoadMesh("square").LoadMat("tblue")
	return btn
}

// setVisible hides and disables the button.
func (b *button) setVisible(visible bool) {
	b.model.SetVisible(visible)
}

// setIcon changes the buttons icon.
func (b *button) setIcon(icon string) {
	b.icon.Model().SetTex(0, icon)
}

// clicked returns true if the button was clicked.
func (b *button) clicked(mx, my int) bool {
	return b.model.Visible() && mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h
}

// label adds a banner to a button or updates the banner if there is
// an existing banner.
func (b *button) label(part vu.Pov, text string) {
	texture := "weblySleek22Black"
	if b.banner == nil {
		if text == "" {
			text = "Sp"
		}
		b.banner = part.NewPov().SetLocation(float64(b.x), float64(b.y), 0)
		b.banner.NewModel("uv").AddTex(texture).LoadFont("weblySleek22")
	}
	b.banner.Model().SetPhrase(text)
}

// position specifies the new center location for the button. This ensures the
// button remains properly located after a screen resize.
func (b *button) position(cx, cy float64) {
	b.cx = cx
	b.cy = cy
	b.x = int(cx) - b.w/2
	b.y = int(cy) - b.h/2
	b.model.SetLocation(b.cx, b.cy, 0)
	if b.banner != nil {
		b.banner.SetLocation(float64(b.x), float64(b.y), 0)
	}
}

// hover hilights the button when the mouse is over it.
func (b *button) hover(mx, my int) bool {
	b.hilite.SetVisible(false)
	if mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h {
		b.hilite.SetVisible(true)
		return true
	}
	return false
}
