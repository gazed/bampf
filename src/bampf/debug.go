// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

// +build debug

package main

// This file/code is only included in debug builds. Eg:
//     go build -tags debug

import (
	"vu"
)

// debugReactions are extra commands to help debug/test the game. They are
// not available in the production builds.
func (g *game) debugReactions() map[string]vu.Reaction {
	return map[string]vu.Reaction{
		"Sh-F": vu.NewReactOnce("fly", func() { g.toggleFly() }),                       // Turn flying on or off.
		"X":    vu.NewReaction("up", func() { g.lens.up(g.cl.body, g.dt, g.run) }),     // Fly up.
		"Z":    vu.NewReaction("down", func() { g.lens.down(g.cl.body, g.dt, g.run) }), // Fly down.
		"0":    vu.NewReactOnce("level0", func() { g.setLevel(0) }),                    // Jump to level 0.
		"1":    vu.NewReactOnce("level1", func() { g.setLevel(1) }),                    // Jump to level 1.
		"2":    vu.NewReactOnce("level2", func() { g.setLevel(2) }),                    // Jump to level 2.
		"3":    vu.NewReactOnce("level3", func() { g.setLevel(3) }),                    // Jump to level 3.
		"4":    vu.NewReactOnce("level4", func() { g.setLevel(4) }),                    // Jump to level 4.
		"B":    vu.NewReaction("bang", func() { g.cl.player.detach() }),                // Lose cores.
		"H":    vu.NewReaction("heal", func() { g.cl.player.attach() }),                // Gain cores.
		"I":    vu.NewReactOnce("increaseCloak", func() { g.cl.increaseCloak() }),      // Gain longer cloak.
		"O":    vu.NewReactOnce("endGame", func() { g.mp.state(done) }),                // Jump to the end game animation.
	}
}

// toggleFly is used to flip into and out of flying mode.
func (g *game) toggleFly() {
	g.fly = !g.fly
	if g.fly {
		g.last.lx, g.last.ly, g.last.lz = g.cl.scene.ViewLocation()
		g.last.dx, g.last.dy, g.last.dz, g.last.dw = g.cl.scene.ViewRotation()
		g.last.tilt = g.cl.scene.ViewTilt()
		g.cl.body.RemBody()
		g.lens = &fly{}
	} else {
		g.cl.scene.SetViewLocation(g.last.lx, g.last.ly, g.last.lz)
		g.cl.scene.SetViewRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
		g.cl.scene.SetViewTilt(g.last.tilt)
		g.cl.body.SetLocation(g.last.lx, g.last.ly, g.last.lz)
		g.cl.body.SetRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
		g.cl.body.SetBody(vu.Sphere(0.25), 1, 0)
		g.lens = &fps{}
	}
}
