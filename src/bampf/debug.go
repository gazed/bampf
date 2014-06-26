// Copyright © 2013-2014 Galvanized Logic Inc.
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
func (g *game) debugReactions() []Reaction {
	return []Reaction{
		{"fly", "Sh-F", func(i *vu.Input, down int) { g.toggleFly(down) }},                 // Turn flying on or off.
		{"up", "X", func(i *vu.Input, down int) { g.lens.up(g.cl.body, i.Dt, g.run) }},     // Fly up.
		{"down", "Z", func(i *vu.Input, down int) { g.lens.down(g.cl.body, i.Dt, g.run) }}, // Fly down.
		{"level0", "1", func(i *vu.Input, down int) { g.setLevel(0) }},                     // Jump to level 1.
		{"level1", "2", func(i *vu.Input, down int) { g.setLevel(1) }},                     // Jump to level 2.
		{"level2", "3", func(i *vu.Input, down int) { g.setLevel(2) }},                     // Jump to level 3.
		{"level3", "4", func(i *vu.Input, down int) { g.setLevel(3) }},                     // Jump to level 4.
		{"level4", "5", func(i *vu.Input, down int) { g.setLevel(4) }},                     // Jump to level 5.
		{"bang", "B", func(i *vu.Input, down int) { g.cl.player.detach() }},                // Lose cores.
		{"heal", "H", func(i *vu.Input, down int) { g.cl.player.attach() }},                // Gain cores.
		{"incCloak", "I", func(i *vu.Input, down int) { g.cl.increaseCloak() }},            // Gain longer cloak.
		{"endGame", "O", func(i *vu.Input, down int) { g.mp.state(doneGame) }},             // Jump to the end game animation.
		{"wide", "KP+", func(i *vu.Input, down int) { g.cl.alterFov(+1) }},                 // Increase fov
		{"narrow", "KP-", func(i *vu.Input, down int) { g.cl.alterFov(-1) }},               // Decrease fov
	}
}

// toggleFly is used to flip into and out of flying mode.
func (g *game) toggleFly(down int) {
	if down == 1 {
		g.fly = !g.fly
		if g.fly {
			g.last.lx, g.last.ly, g.last.lz = g.cl.scene.Location()
			g.last.dx, g.last.dy, g.last.dz, g.last.dw = g.cl.scene.Rotation()
			g.last.tilt = g.cl.scene.Tilt()
			g.cl.body.RemBody()
			g.lens = &fly{}
		} else {
			g.cl.scene.SetLocation(g.last.lx, g.last.ly, g.last.lz)
			g.cl.scene.SetRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
			g.cl.scene.SetTilt(g.last.tilt)
			g.cl.body.SetLocation(g.last.lx, g.last.ly, g.last.lz)
			g.cl.body.SetRotation(g.last.dx, g.last.dy, g.last.dz, g.last.dw)
			g.cl.body.SetBody(vu.Sphere(0.25), 1, 0)
			g.lens = &fps{}
		}
	}
}
