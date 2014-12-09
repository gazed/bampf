// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"math"
	"vu"
)

// lens dictates how a camera moves. The lens can be swapped for different
// behaviour, the prime example being switching the game fps for a debug
// fly camera.
type lens interface {
	look(sc vu.Scene, spin, dt, xdiff, ydiff float64)
	lookUpDown(sc vu.Scene, ydiff, spin, dt float64)
	back(bod vu.Part, dt, run float64)
	forward(bod vu.Part, dt, run float64)
	left(bod vu.Part, dt, run float64)
	right(bod vu.Part, dt, run float64)
	up(bod vu.Part, dt, run float64)
	down(bod vu.Part, dt, run float64)
}

// lens
// ===========================================================================
// fps

// fps is a type of lens.
type fps struct{}

// look changes the view left/right for changes in the x direction
// and up/down for changes in the y direction.
func (f *fps) look(sc vu.Scene, spin, dt, xdiff, ydiff float64) {
	if xdiff != 0 {
		switch { // cap movement amount.
		case xdiff > 10:
			xdiff = 10
		case xdiff < -10:
			xdiff = -10
		}
		sc.Cam().Spin(0, dt*float64(-xdiff)*spin, 0)
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

// lookUpDown limits the vertical camera movement to plus/minus 90 degrees.
func (f *fps) lookUpDown(sc vu.Scene, ydiff, spin, dt float64) {
	height := sc.Cam().Tilt()
	height += dt * -ydiff * spin
	if height > 90.0 {
		height = 90.0
	}
	if height < -90.0 {
		height = -90.0
	}
	sc.Cam().SetTilt(height)
}

// implement the rest of the lens interface.
func (f *fps) back(bod vu.Part, dt, run float64)    { f.move(bod, 0, 0, dt*run) }
func (f *fps) forward(bod vu.Part, dt, run float64) { f.move(bod, 0, 0, dt*-run) }
func (f *fps) left(bod vu.Part, dt, run float64)    { f.move(bod, dt*-run, 0, 0) }
func (f *fps) right(bod vu.Part, dt, run float64)   { f.move(bod, dt*run, 0, 0) }
func (f *fps) up(bod vu.Part, dt, run float64)      {} // only works in debug
func (f *fps) down(bod vu.Part, dt, run float64)    {} // only works in debug

// Handle movement assuming there is a physics body associated with the camera.
// This attempts to smooth out movement by adding a higher initial velocity push
// and then capping movement once max accelleration is reached.
func (f *fps) move(bod vu.Part, x, y, z float64) {
	boost := 40.0    // kick into high gear from stop.
	maxAccel := 10.0 // limit accelleration.
	sx, _, sz := bod.Body().Speed()
	if x != 0 {
		switch {
		case sx == 0.0:
			bod.Move(x*boost, 0, 0)
		case math.Abs(sx) < maxAccel && math.Abs(sz) < maxAccel:
			bod.Move(x, 0, 0)
		}
	}
	if z != 0 {
		switch {
		case sz == 0.0:
			bod.Move(0, 0, z*boost)
		case math.Abs(sx) < maxAccel && math.Abs(sz) < maxAccel:
			bod.Move(0, 0, z)
		}
	}
}

// fps
// ===========================================================================
// fly

// fly is a type of lens used in debug builds.
type fly struct{ fps }

// There is no physics body associated with the camera during debug.
func (f *fly) back(bod vu.Part, dt, run float64)    { bod.Move(0, 0, dt*run) }
func (f *fly) forward(bod vu.Part, dt, run float64) { bod.Move(0, 0, dt*-run) }
func (f *fly) left(bod vu.Part, dt, run float64)    { bod.Move(dt*-run, 0, 0) }
func (f *fly) right(bod vu.Part, dt, run float64)   { bod.Move(dt*run, 0, 0) }
func (f *fly) up(bod vu.Part, dt, run float64)      { bod.Move(0, dt*run, 0) }
func (f *fly) down(bod vu.Part, dt, run float64)    { bod.Move(0, dt*-run, 0) }
