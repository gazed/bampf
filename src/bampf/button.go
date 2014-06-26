// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"vu"
)

// button has both an image/icon and an action. The action can be linked to a key.
// Both a button image (and corresponding action key, if applicable) are shown on
// the button.
type button struct {
	area                   // Button is rectangular.
	id     string          // Button unique name.
	action vu.InputHandler // Click handler.
	icon   vu.Part         // Button image.
	hilite vu.Part         // Hover overlay.
	banner vu.Part         // Label for the action associated with the button.
	cx, cy float64         // Button center location.
	model  vu.Part         // Holds button 3D model. Used for transforms.
}

// newButton creates a button. Buttons are initialized with a size and repositioned later.
//   part   is the parent model.
//   size   is both the width and height.
//   icon   is the (already loaded) texture image.
//   action is the action to perform when the button is pressed.
func newButton(parent vu.Part, size int, icon string, action vu.InputHandler) *button {
	btn := &button{}
	btn.model = parent.AddPart()
	btn.action = action
	btn.w, btn.h = size, size

	// create the button icon.
	btn.id = icon
	btn.icon = btn.model.AddPart().SetScale(float64(btn.w/2), float64(btn.h/2), 1)
	btn.icon.SetRole("uv").SetMesh("icon").AddTex(icon).SetMaterial("half")

	// create a hilite that is only shown on mouse over.
	btn.hilite = btn.model.AddPart().SetScale(float64(btn.w/2.0), float64(btn.h/2.0), 1)
	btn.hilite.SetVisible(false)
	btn.hilite.SetRole("flat").SetMesh("square").SetMaterial("tblue")
	return btn
}

// setVisible hides and disables the button.
func (b *button) setVisible(visible bool) {
	b.model.SetVisible(visible)
}

// setIcon changes the buttons icon.
func (b *button) setIcon(icon string) {
	b.icon.Role().UseTex(icon, 0)
}

// clicked returns true if the button was clicked. The associated action
// is triggered.
func (b *button) clicked(in *vu.Input, down int) bool {
	if b.model.Visible() {
		mx, my := in.Mx, in.My
		if mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h {
			b.action(in, down)
			return true
		}
	}
	return false
}

// label adds a banner to a button or updates the banner if there is a
// existing banner.
func (b *button) label(part vu.Part, text string) {
	texture := "weblySleek22Black"
	if b.banner == nil {
		if text == "" {
			text = "Sp"
		}
		b.banner = part.AddPart().SetLocation(float64(b.x), float64(b.y), 0)
		b.banner.SetRole("uv").AddTex(texture).SetFont("weblySleek22")
	}
	b.banner.Role().SetPhrase(text)
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
