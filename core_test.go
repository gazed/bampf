// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"testing"
)

func TestToGrid(t *testing.T) {
	units := 2.0
	gridx, gridy := toGrid(-0.9, 0, -0.9, units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(0.9, 0, 0.9, units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(-0.9, 0, 0.9, units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(0.9, 0, -0.9, units)
	if gridx != 0 || gridy != 0 {
		t.Errorf("Expected 0,0 got %d,%d", gridx, gridy)
	}

	gridx, gridy = toGrid(1.01, 0, -1.01, units)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(2.99, 0, -2.99, units)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(1.01, 0, -2.99, units)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(2.99, 0, -1.01, units)
	if gridx != 1 || gridy != 1 {
		t.Errorf("Expected 1,1 got %d,%d", gridx, gridy)
	}

	gridx, gridy = toGrid(-1.01, 0, 1.01, units)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(-2.99, 0, 2.99, units)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(-1.01, 0, 2.99, units)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
	gridx, gridy = toGrid(-2.99, 0, 1.01, units)
	if gridx != -1 || gridy != -1 {
		t.Errorf("Expected -1,-1 got %d,%d", gridx, gridy)
	}
}
