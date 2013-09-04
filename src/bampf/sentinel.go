// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

// Sentinel is the player enemy AI's.  A *very* simple AI based on random
// movements that don't backtrack. AI improvers can refer to (much) more sophisticated
// algorithms from:
//    http://aigamedev.com
//    http://www.aiwisdom.com

import (
	"math"
	"math/rand"
	"time"
	"vu"
	"vu/grid"
)

// sentinel tracks and moves one player enemy. The maze position information
// is kept as x,y grid spots.
type sentinel struct {
	part   vu.Part   // Top level for model transforms.
	model  vu.Part   // Simple model for initial levels.
	center vu.Part   // Add some difference for later levels.
	prev   *gridSpot // Sentinels previous location.
	next   *gridSpot // Sentinels next location.
	units  float32   // Maze scale factor
}

// newSentinel creates a player enemy.
func newSentinel(eng *vu.Eng, part vu.Part, level, units int) *sentinel {
	s := &sentinel{}
	s.part = part
	s.units = float32(units)
	s.part.SetLocation(0, 0.5, 0)
	s.model = part.AddPart()
	s.model.SetCullable(false)
	s.model.SetFacade("cube", "flata", "tblue")
	if level > 0 {
		s.center = s.part.AddPart()
		s.center.SetCullable(false)
		s.center.SetFacade("cube", "flata", "tred")
		s.center.SetScale(0.125, 0.125, 0.125)
	}
	return s
}

// move adjusts the sentinels current position according to the movement algorithm.
// The sentry gets either gets moved a little closer to its next spot or it gets
// a new spot to move to.
func (s *sentinel) move(plan grid.Grid) {
	speed := float32(25) // higher is slower
	gamex, gamey, gamez := s.part.Location()
	inv := float32(1) / float32(s.units)
	gridfx, gridfy := gamex*inv, -gamez*inv
	atx := math.Abs(float64(gridfx-float32(s.next.x))) < 0.001
	atz := math.Abs(float64(gridfy-float32(s.next.y))) < 0.001
	if atx && atz {

		// arrived at next spot... get a new one.
		s.prev, s.next = s.next, s.nextSpot(plan)
	} else {

		// move a bit closer to the next spot.
		if !atx {
			gridfx += float32(s.next.x-s.prev.x) / speed
		}
		if !atz {
			gridfy += float32(s.next.y-s.prev.y) / speed
		}
	}
	s.part.SetLocation(gridfx*float32(s.units), gamey, -gridfy*float32(s.units))
}

// setGridLocation puts the sentinal down at the given grid location.
func (s *sentinel) setGridLocation(gridx, gridy int) {
	s.prev = &gridSpot{gridx, gridy}
	s.next = &gridSpot{gridx, gridy}
	_, gamey, _ := s.part.Location()
	gamex, gamez := s.next.toGame(gridx, gridy, s.units)
	s.part.SetLocation(gamex, gamey, gamez)
}

// Location gets the sentinels current location.
func (s *sentinel) Location() (x, y, z float32) { return s.part.Location() }

// setScale changes the troopers size.
func (s *sentinel) setScale(scale float32) { s.model.SetScale(scale, scale, scale) }

// nextSpot picks where the sentinel will be going towards by considering
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
			chooser := rand.New(rand.NewSource(time.Now().UnixNano() + int64(s.part.Id())))
			way = chooser.Intn(len(choices))
		}
		return choices[way]
	}
	return was // just walk back to where they where.
}

// isValidSpot checks that a spot is valid for a sentinel, i.e. not a wall or the
// previous location.
func (s *sentinel) isValidSpot(plan grid.Grid, w, h int, old *gridSpot, x, y int) bool {
	if x == old.x && y == old.y { // can't use previous position.
		return false
	}
	if x >= 0 && y >= 0 && x < w && y < h { // exclude walls.
		return !plan.IsWall(x, y)
	}
	if x >= -1 && y >= -1 && x <= w && y <= h { // outside edge ok.
		return true
	}
	return false // anywhere else is a no-go zone.
}
