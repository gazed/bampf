// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"os"
	"testing"
)

func TestSaveRestore(t *testing.T) {
	file := "gob"
	s1 := newSaver()
	s1.File = file
	km := []string{"K", "M"}
	s1.persistBindings(km)
	s1.persistWindow(10, 20, 30, 40)

	// now restore the same file.
	s2 := newSaver()
	s2.File = file
	s2.restore()
	if len(s1.Kbinds) != len(s2.Kbinds) {
		t.Errorf("Expected %d, got %d", len(s1.Kbinds), len(s2.Kbinds))
	}
	if s2.H != 40 {
		t.Errorf("Expected %d, got %d", 40, s2.H)
	}

	// cleanup
	os.Remove(file)
}
