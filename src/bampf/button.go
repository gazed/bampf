// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"vu"
)

// Button has both an image and an action.  The action can be linked to a key.
// Both a button image (and corresponding action key, if applicable) are shown on
// the button.
type button struct {
	area               // Button is rectangular.
	action vu.Reaction // Click handler.
	icon   vu.Part     // Button image.
	hilite vu.Part     // Hover overlay.
	banner vu.Part     // Label for the urge associated with the button.
	cx, cy float32     // Button center location.
	model  vu.Part     // Holds button 3D model. Used for transforms.
}

// newButton creates a button. Buttons are given a size to start and are positioned later.
//   part is the parent model.
//   size is both the width and height.
//   icon is the (already loaded) texture image.
//   action is the action to perform when the button is pressed.
func newButton(eng *vu.Eng, parent vu.Part, size int, icon string, action vu.Reaction) *button {
	btn := &button{}
	btn.model = parent.AddPart()
	btn.action = action
	btn.w, btn.h = size, size

	// create the button icon.
	btn.icon = btn.model.AddPart()
	btn.icon.SetFacade("icon", "uv", "half")
	btn.icon.SetTexture(icon, 0)
	btn.icon.SetScale(float32(btn.w/2), float32(btn.h/2), 1)

	// create a hilite that is only shown on mouse over.
	btn.hilite = btn.model.AddPart()
	btn.hilite.SetFacade("square", "flat", "tblue")
	btn.hilite.SetScale(float32(btn.w/2), float32(btn.h/2), 1)
	btn.hilite.SetVisible(false)
	return btn
}

// setVisible hides and disables the button if visible is false.
func (b *button) setVisible(visible bool) {
	b.model.SetVisible(visible)
}

// setIcon changes the buttons icon.
func (b *button) setIcon(icon string) {
	b.icon.SetTexture(icon, 0)
}

// click returns true if the button was clicked and the associated action
// was triggered.
func (b *button) click(mx, my int) bool {
	if b.model.Visible() {
		if mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h {
			b.action.Do()
			return true
		}
	}
	return false
}

// label adds a banner to a button or updates the banner if there is a
// existing banner.
func (b *button) label(eng *vu.Eng, part vu.Part, text string) {
	colour := "weblySleek22Black"
	if b.banner == nil {
		b.banner = part.AddPart()
		b.banner.SetBanner(text, "uv", "weblySleek22", colour)
		b.banner.SetLocation(float32(b.x), float32(b.y), 0)
	} else {
		b.banner.UpdateBanner(text)
	}
}

// position specifies the new center location for the button. This ensures the
// button remains properly located after a screen resize.
func (b *button) position(cx, cy float32) {
	b.cx = cx
	b.cy = cy
	b.x = int(cx) - b.w/2
	b.y = int(cy) - b.h/2
	b.model.SetLocation(b.cx, b.cy, 0)
	if b.banner != nil {
		b.banner.SetLocation(float32(b.x), float32(b.y), 0)
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
