// Copyright © 2013-2016 Galvanized Logic Inc.
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
		case press == "B":
			g.cl.player.detach() // Lose cores.
		case press == "H":
			g.cl.player.attach() // Gain cores.
		case press == "I":
			g.cl.debugCloak() // Gain longer cloak.
		case press == "O" && down == 1:
			g.mp.state(finishGame) // Jump to the end game animation.
		}
	}
}

// toggleFly is used to flip into and out of flying mode.
func (g *game) toggleFly() {
	g.fly = !g.fly
	if g.fly {
		g.last.lx, g.last.ly, g.last.lz = g.cl.cam.At()
		g.last.pitch = g.cl.cam.Pitch()
		g.last.yaw = g.cl.cam.Yaw()
		g.cl.body.Dispose(vu.BODY)
		g.dir = g.cl.cam.Lookat()
	} else {
		g.lens.pitch = g.last.pitch
		g.lens.yaw = g.last.yaw
		g.cl.cam.SetPitch(g.last.pitch)
		g.cl.cam.SetYaw(g.last.yaw)
		g.cl.cam.SetAt(g.last.lx, g.last.ly, g.last.lz)
		g.cl.body.SetAt(g.last.lx, g.last.ly, g.last.lz)
		g.cl.body.SetRotation(g.cl.cam.Lookat())
		g.cl.body.NewBody(vu.NewSphere(0.25))
		g.cl.body.SetSolid(1, 0)
		g.dir = g.cl.cam.Lookxz()
	}
}
