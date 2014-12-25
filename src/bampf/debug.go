// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

// +build debug

package main

// This file/code is only included in debug builds. Eg:
//     go build -tags debug

import (
	"log"

	"github.com/gazed/vu"
)

// logger enables logging in debug loads.
func (b *bampf) logger(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// processDebugInput are extra commands to help debug/test the game.
// They are not available in the production builds.
// Don't bother with game events, immediately process the debug request.
func (g *game) processDebugInput(in *vu.Input) {
	for press, down := range in.Down {
		switch {
		case press == "F" && down == 1:
			g.toggleFly() // Turn flying on or off.
		case press == "X":
			g.lens.up(g.cl.body, in.Dt, g.run) // Fly up.
		case press == "Z":
			g.lens.down(g.cl.body, in.Dt, g.run) // Fly down.
		case press == "B":
			g.cl.player.detach() // Lose cores.
		case press == "H":
			g.cl.player.attach() // Gain cores.
		case press == "I":
			g.cl.debugCloak() // Gain longer cloak.
		case press == "O" && down == 1:
			g.mp.state(finishGame) // Jump to the end game animation.
		case press == "KP+":
			g.cl.alterFov(+1) // Increase fov
		case press == "KP-":
			g.cl.alterFov(-1) // Decrease fov
		}
	}
}

// toggleFly is used to flip into and out of flying mode.
func (g *game) toggleFly() {
	g.fly = !g.fly
	if g.fly {
		g.last.lx, g.last.ly, g.last.lz = g.cl.cam.Location()
		g.last.dx, g.last.dy, g.last.dz, g.last.dw = g.cl.cam.Rotation()
		g.last.tilt = g.cl.cam.Tilt()
		g.cl.body.RemBody()
		g.lens = &fly{}
	} else {
		g.cl.cam.SetLocation(g.last.lx, g.last.ly, g.last.lz)
		g.cl.cam.SetRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
		g.cl.cam.SetTilt(g.last.tilt)
		g.cl.body.SetLocation(g.last.lx, g.last.ly, g.last.lz)
		g.cl.body.SetRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
		g.cl.body.SetBody(vu.NewSphere(0.25), 1, 0)
		g.lens = &fps{}
	}
}
