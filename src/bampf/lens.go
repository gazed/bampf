// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"vu"
)

// lens dictates how a camera moves.  The lens can be swapped for different
// behaviour, the prime example being switching the game fps for a debug
// fly camera.
type lens interface {
	look(sc vu.Scene, spin, dt, xdiff, ydiff float32)
	lookUpDown(sc vu.Scene, ydiff, spin, dt float32)
	back(sc vu.Scene, dt, run float32)
	forward(sc vu.Scene, dt, run float32)
	left(sc vu.Scene, dt, run float32)
	right(sc vu.Scene, dt, run float32)
	up(sc vu.Scene, dt, run float32)
	down(sc vu.Scene, dt, run float32)
}

// lens
// ===========================================================================
// fps

// fps is a type of lens.
type fps struct{}

// look changes the view left/right for changes in the x direction
// and up/down for changes in the y direction.
func (f *fps) look(sc vu.Scene, spin, dt, xdiff, ydiff float32) {
	if xdiff != 0 {
		switch { // cap movement amount.
		case xdiff > 10:
			xdiff = 10
		case xdiff < -10:
			xdiff = -10
		}
		sc.PanView(vu.YAxis, dt*float32(-xdiff)*spin)
	}
	if ydiff != 0 {
		switch { // cap movement amount.
		case ydiff > 10:
			ydiff = 10
		case ydiff < -10:
			ydiff = -10
		}
		f.lookUpDown(sc, ydiff, spin, dt)
	}
}

func (f *fps) lookUpDown(sc vu.Scene, ydiff, spin, dt float32) {
	height := sc.ViewTilt()
	height += dt * -ydiff * spin
	if height > 90.0 {
		height = 90.0
	}
	if height < -90.0 {
		height = -90.0
	}
	sc.SetViewTilt(height)
}

// implement the rest of the lens interface.
func (f *fps) back(sc vu.Scene, dt, run float32)    { sc.MoveView(0, 0, dt*run) }
func (f *fps) forward(sc vu.Scene, dt, run float32) { sc.MoveView(0, 0, dt*-run) }
func (f *fps) left(sc vu.Scene, dt, run float32)    { sc.MoveView(dt*-run, 0, 0) }
func (f *fps) right(sc vu.Scene, dt, run float32)   { sc.MoveView(dt*run, 0, 0) }
func (f *fps) up(sc vu.Scene, dt, run float32)      {} // only works in debug
func (f *fps) down(sc vu.Scene, dt, run float32)    {} // only works in debug

// fps
// ===========================================================================
// fly

// fly is a type of lens used in debug builds.
type fly struct{ fps } // debug camera movement

func (f *fly) up(sc vu.Scene, dt, run float32)   { sc.MoveView(0, dt*run, 0) }
func (f *fly) down(sc vu.Scene, dt, run float32) { sc.MoveView(0, dt*-run, 0) }
