// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

// area describes a 2D part of a screen. It is the base class for sections
// of the HUD and buttons.
type area struct {
	x, y   int     // bottom left corner.
	w, h   int     // width and height.
	cx, cy float32 // area center location.
}

// center calculates the center of the given area.
func (a *area) center() (cx, cy float32) {
	cx = float32(a.x + a.w/2)
	cy = float32(a.y + a.h/2)
	return
}
