// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

// Sentinel is the player enemy AI's. A *very* simple AI based on random
// movements that don't backtrack.
//
// FUTURE: AI improvers can refer to sophisticated algorithms from:
//    http://aigamedev.com
//    http://www.aiwisdom.com

import (
	"math"
	"math/rand"

	"github.com/gazed/vu"
	"github.com/gazed/vu/grid"
)

// sentinel tracks and moves one player enemy. The maze position information
// is kept as x,y grid spots.
type sentinel struct {
	part   vu.Pov    // Top level for model transforms.
	model  vu.Pov    // Simple model for initial levels.
	center vu.Pov    // Add some difference for later levels.
	prev   *gridSpot // Sentinels previous location.
	next   *gridSpot // Sentinels next location.
	units  float64   // Maze scale factor
}

// newSentinel creates a player enemy.
func newSentinel(part vu.Pov, level, units int, fade float64) *sentinel {
	s := &sentinel{}
	s.part = part
	s.units = float64(units)
	s.part.SetLocation(0, 0.5, 0)
	if level > 0 {
		s.center = s.part.NewPov().SetScale(0.125, 0.125, 0.125)
		m := s.center.NewModel("flata").LoadMesh("cube").LoadMat("tred")
		m.SetUniform("fd", fade)
	}
	s.model = part.NewPov()
	m := s.model.NewModel("flata").LoadMesh("cube").LoadMat("tblue")
	m.SetUniform("fd", fade)
	return s
}

// move adjusts the sentinels current position according to the movement algorithm.
// The sentry gets moved a little closer to its next spot. If its at the next spot,
// then it gets a new spot to move to.
func (s *sentinel) move(plan grid.Grid) {
	speed := float64(25) // higher is slower
	gamex, gamey, gamez := s.part.Location()
	inv := float64(1) / float64(s.units)
	gridfx, gridfy := gamex*inv, -gamez*inv
	atx := math.Abs(float64(gridfx-float64(s.next.x))) < 0.001
	atz := math.Abs(float64(gridfy-float64(s.next.y))) < 0.001
	if atx && atz {

		// arrived at next spot... get a new one.
		s.prev, s.next = s.next, s.nextSpot(plan)
	} else {

		// move a bit closer to the next spot.
		if !atx {
			gridfx += float64(s.next.x-s.prev.x) / speed
		}
		if !atz {
			gridfy += float64(s.next.y-s.prev.y) / speed
		}
	}
	s.part.SetLocation(gridfx*float64(s.units), gamey, -gridfy*float64(s.units))
}

// setGridLocation puts the sentinel down at the given grid location.
func (s *sentinel) setGridLocation(gridx, gridy int) {
	s.prev = &gridSpot{gridx, gridy}
	s.next = &gridSpot{gridx, gridy}
	_, gamey, _ := s.part.Location()
	gamex, gamez := toGame(gridx, gridy, s.units)
	s.part.SetLocation(gamex, gamey, gamez)
}

// location gets the sentinels current location.
func (s *sentinel) location() (x, y, z float64) { return s.part.Location() }

// setScale changes the sentinels size.
func (s *sentinel) setScale(scale float64) { s.model.SetScale(scale, scale, scale) }

// nextSpot picks where the sentinel will be going to by considering
// all the surrounding spaces and picking from the valid ones.
func (s *sentinel) nextSpot(plan grid.Grid) *gridSpot {
	at := s.next
	was := s.prev
	w, h := plan.Size()

	// using knowledge that the grid starts at 0, 0 and goes to size, -size.
	// and that the outside border is also valid.
	choices := []*gridSpot{}
	if at.x >= -1 && at.y >= -1 && at.x <= w && at.y <= h {
		if s.isValidSpot(plan, w, h, was, at.x+1, at.y) {
			choices = append(choices, &gridSpot{at.x + 1, at.y})
		}
		if s.isValidSpot(plan, w, h, was, at.x-1, at.y) {
			choices = append(choices, &gridSpot{at.x - 1, at.y})
		}
		if s.isValidSpot(plan, w, h, was, at.x, at.y+1) {
			choices = append(choices, &gridSpot{at.x, at.y + 1})
		}
		if s.isValidSpot(plan, w, h, was, at.x, at.y-1) {
			choices = append(choices, &gridSpot{at.x, at.y - 1})
		}
	}
	if len(choices) > 0 {
		way := 0
		if len(choices) > 1 {
			way = rand.Intn(len(choices))
		}
		return choices[way]
	}
	return was // backtrack should never happen.
}

// isValidSpot checks that a spot is valid for a sentinel, i.e. not a wall or the
// previous location.
func (s *sentinel) isValidSpot(plan grid.Grid, w, h int, old *gridSpot, x, y int) bool {
	if x == old.x && y == old.y { // can't use previous position.
		return false
	}
	if x >= 0 && y >= 0 && x < w && y < h { // exclude walls.
		return plan.IsOpen(x, y)
	}
	if x >= -1 && y >= -1 && x <= w && y <= h { // outside edge ok.
		return true
	}
	return false // anywhere else is a no-go zone.
}
