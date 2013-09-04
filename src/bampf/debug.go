// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

// +build debug

package main

// This file/code is only included in debug builds. Eg:
//     go build -tags 'debug'

import (
	"vu"
)

// debugReactions are extra commands to help debug/test the game.  They are
// not available in the production builds.
func (g *game) debugReactions() map[string]vu.Reaction {
	return map[string]vu.Reaction{
		"Sh-D": vu.NewReactOnce("debug", func() { g.toggleDebug() }),                        // Turn on flying.
		"X":    vu.NewReaction("up", func() { g.lens.up(g.cl.scene, g.eng.Dt, g.run) }),     // Fly up.
		"Z":    vu.NewReaction("down", func() { g.lens.down(g.cl.scene, g.eng.Dt, g.run) }), // Fly down.
		"0":    vu.NewReactOnce("level0", func() { g.setLevel(0) }),                         // Jump to level 0.
		"1":    vu.NewReactOnce("level1", func() { g.setLevel(1) }),                         // Jump to level 1.
		"2":    vu.NewReactOnce("level2", func() { g.setLevel(2) }),                         // Jump to level 2.
		"3":    vu.NewReactOnce("level3", func() { g.setLevel(3) }),                         // Jump to level 3.
		"4":    vu.NewReactOnce("level4", func() { g.setLevel(4) }),                         // Jump to level 4.
		"B":    vu.NewReaction("bang", func() { g.cl.player.detach() }),                     // Lose cores.
		"H":    vu.NewReaction("heal", func() { g.cl.player.attach() }),                     // Gain cores.
		"I":    vu.NewReactOnce("increaseCloak", func() { g.cl.increaseCloak() }),           // Gain longer cloak.
		"O":    vu.NewReactOnce("endGame", func() { g.mp.state(done) }),                     // Jump to the end game animation.
	}
}

// toggleDebug is used to flip into/out-of flying mode.
func (g *game) toggleDebug() {
	g.debug = !g.debug
	if g.debug {
		g.last.lx, g.last.ly, g.last.lz = g.cl.scene.ViewLocation()
		g.last.dx, g.last.dy, g.last.dz, g.last.dw = g.cl.scene.ViewRotation()
		g.last.tilt = g.cl.scene.ViewTilt()
		g.lens = &fly{}
	} else {
		g.cl.scene.SetViewLocation(g.last.lx, g.last.ly, g.last.lz)
		g.cl.scene.SetViewRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
		g.cl.scene.SetViewTilt(g.last.tilt)
		g.lens = &fps{}
	}
}
