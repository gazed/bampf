// Copyright © 2013-2016 Galvanized Logic Inc.
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
	eventID   int         // game event identifier.
	eventData interface{} // game event data.
	icon      *vu.Ent     // Button image.
	hilite    *vu.Ent     // Hover overlay.
	banner    *vu.Ent     // Label for the action associated with the button.
	cx, cy    float64     // Button center location.
	model     *vu.Ent     // Holds button 3D model. Used for transforms.
}

// newButton creates a button. Buttons are initialized with a size and repositioned later.
//   root   is the parent transform.
//   size   is both the width and height.
//   icon   is the (already loaded) texture image.
//   action is the action to perform when the button is pressed.
func newButton(root *vu.Ent, size int, icon string, eventID int, eventData interface{}) *button {
	btn := &button{}
	btn.model = root.AddPart()
	btn.eventID = eventID
	btn.eventData = eventData
	btn.w, btn.h = size, size

	// create the button icon.
	btn.id = icon
	btn.icon = btn.model.AddPart().SetScale(float64(btn.w/2), float64(btn.h/2), 1)
	btn.icon.MakeModel("textured", "msh:icon", "tex:"+icon)
	btn.icon.SetAlpha(0.5)

	// create a hilite that is only shown on mouse over.
	btn.hilite = btn.model.AddPart().SetScale(float64(btn.w/2.0), float64(btn.h/2.0), 1)
	btn.hilite.Cull(true)
	btn.hilite.MakeModel("colored", "msh:square", "mat:tblue")
	return btn
}

// setVisible hides and disables the button.
func (b *button) setVisible(visible bool) { b.model.Cull(!visible) }

// setIcon changes the buttons icon.
func (b *button) setIcon(icon string) { b.icon.SetFirst(icon) }

// clicked returns true if the button was clicked.
func (b *button) clicked(mx, my int) bool {
	return !b.model.Culled() && mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h
}

// label adds a banner to a button or updates the banner if there is
// an existing banner.
func (b *button) label(part *vu.Ent, keyCode int) {
	if keysym := vu.Symbol(keyCode); keysym > 0 {
		if b.banner == nil {
			b.banner = part.AddPart().SetAt(float64(b.x), float64(b.y), 0)
			b.banner.MakeLabel("labeled", "lucidiaSu22")
			b.banner.SetColor(0, 0, 0)
		}
		if keyCode == 0 {
			keyCode = vu.KSpace
		}
		b.banner.SetStr(string(keysym))
	}
}

// position specifies the new center location for the button. This ensures the
// button remains properly located after a screen resize.
func (b *button) position(cx, cy float64) {
	b.cx = cx
	b.cy = cy
	b.x = int(cx) - b.w/2
	b.y = int(cy) - b.h/2
	b.model.SetAt(b.cx, b.cy, 0)
	if b.banner != nil {
		b.banner.SetAt(float64(b.x), float64(b.y), 0)
	}
}

// hover hilights the button when the mouse is over it.
func (b *button) hover(mx, my int) bool {
	b.hilite.Cull(true)
	if mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h {
		b.hilite.Cull(false)
		return true
	}
	return false
}
