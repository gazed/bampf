// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"testing"
)

func TestGameToGrid(t *testing.T) {
	cc := &coreControl{}
	cc.units = 2
	cc.spot = &gridSpot{}
	gridx, gridy := cc.spot.toGrid(1.0, 0, -1.0, cc.units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.spot.toGrid(1.0, 0, 1.0, cc.units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.spot.toGrid(1.5, 0, 1.5, cc.units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.spot.toGrid(2.0, 0, 2.0, cc.units)
	if gridx != 1 || gridy != -1 {
		t.Errorf("Expected 1,-1 got %d,%d", gridx, gridy)
	}
}

func TestPlayerToGrid(t *testing.T) {
	cc := &coreControl{}
	cc.units = 2
	gridx, gridy := cc.playerToGrid(-0.9, 0, -0.9)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(0.9, 0, 0.9)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(-0.9, 0, 0.9)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(0.9, 0, -0.9)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}

	gridx, gridy = cc.playerToGrid(1.01, 0, -1.01)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(2.99, 0, -2.99)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(1.01, 0, -2.99)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(2.99, 0, -1.01)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}

	gridx, gridy = cc.playerToGrid(-1.01, 0, 1.01)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(-2.99, 0, 2.99)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(-1.01, 0, 2.99)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = cc.playerToGrid(-2.99, 0, 1.01)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
}
