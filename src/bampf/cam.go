// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"math"

	"github.com/gazed/vu"
	"github.com/gazed/vu/math/lin"
)

// cam controls the main game level camera.
type cam struct {
	pitch float64 // used to smooth camera.
	yaw   float64 // used to smooth camera.
}

// implement the rest of the lens interface.
func (c *cam) back(bod vu.Pov, dt, run float64, q *lin.Q)    { c.move(bod, 0, 0, dt*run, q) }
func (c *cam) forward(bod vu.Pov, dt, run float64, q *lin.Q) { c.move(bod, 0, 0, dt*-run, q) }
func (c *cam) left(bod vu.Pov, dt, run float64, q *lin.Q)    { c.move(bod, dt*-run, 0, 0, q) }
func (c *cam) right(bod vu.Pov, dt, run float64, q *lin.Q)   { c.move(bod, dt*run, 0, 0, q) }

// Handle movement assuming there is a physics body associated with the camera.
// This attempts to smooth out movement by adding a higher initial velocity push
// and then capping movement once max accelleration is reached.
func (c *cam) move(bod vu.Pov, x, y, z float64, dir *lin.Q) {
	if body := bod.Body(); body != nil {
		boost := 40.0    // kick into high gear from stop.
		maxAccel := 10.0 // limit accelleration.
		sx, _, sz := body.Speed()
		if x != 0 {
			switch {
			case sx == 0.0:
				// apply push in the current direction.
				dx, dy, dz := lin.MultSQ(x*boost, 0, 0, dir)
				body.Push(dx, dy, dz)
			case math.Abs(sx) < maxAccel && math.Abs(sz) < maxAccel:
				dx, dy, dz := lin.MultSQ(x, 0, 0, dir)
				body.Push(dx, dy, dz)
			}
		}
		if z != 0 {
			switch {
			case sz == 0.0:
				dx, dy, dz := lin.MultSQ(0, 0, z*boost, dir)
				body.Push(dx, dy, dz)
			case math.Abs(sx) < maxAccel && math.Abs(sz) < maxAccel:
				dx, dy, dz := lin.MultSQ(0, 0, z, dir)
				body.Push(dx, dy, dz)
			}
		}
	} else {
		bod.Move(x, y, z, dir)
	}
}

// look changes the view left/right for changes in the x direction
// and up/down for changes in the y direction.
func (c *cam) look(spin, dt, xdiff, ydiff float64) {
	limit := 20.0 // pixels
	if xdiff != 0 {
		switch { // cap movement amount.
		case xdiff > limit:
			xdiff = limit
		case xdiff < -limit:
			xdiff = -limit
		}
		c.yaw += dt * float64(-xdiff) * spin
	}
	if ydiff != 0 {
		switch { // cap movement amount.
		case ydiff > limit:
			ydiff = limit
		case ydiff < -limit:
			ydiff = -limit
		}
		c.pitch = c.updatePitch(c.pitch, ydiff, spin, dt)
	}
}

// updatePitch limits the vertical camera movement to plus/minus 90 degrees.
func (c *cam) updatePitch(pitch, ydiff, spin, dt float64) float64 {
	limit := 90.0 // degrees
	pitch += dt * ydiff * spin
	if pitch > limit {
		pitch = limit
	}
	if pitch < -limit {
		pitch = -limit
	}
	return pitch
}

// reset puts the target pitch and yaw back to zero.
func (c *cam) reset(camera vu.Camera) {
	c.pitch, c.yaw = 0, 0
	camera.SetPitch(0)
	camera.SetYaw(0)
}

func (c *cam) update(camera vu.Camera) {
	fraction := 0.25
	pitch := camera.Pitch()
	if !lin.Aeq(pitch, c.pitch) {
		pitch = (c.pitch-pitch)*fraction + pitch
		camera.SetPitch(pitch)
	}
	yaw := camera.Yaw()
	if !lin.Aeq(yaw, c.yaw) {
		yaw = (c.yaw-yaw)*fraction + yaw
		camera.SetYaw(yaw)
	}
}
