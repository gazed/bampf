// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"path"
)

// Saver persists any game state that needs to be remembered between one
// game session and the next. Saver needs to be public and visible for
// the encoding package.
type Saver struct {
	File       string // Save file name.
	Kbinds     []int  // Key bindings.
	X, Y, W, H int    // Window location.
	Mute       bool   // True if the game is muted.
	Full       bool   // True if the game is fullscreen.
}

// newSaver creates default persistent application state. The directory
// is platform specific and specified by:
//    osx  : see saver_darwin.go
//    win  : see saver_windows.go
//    lin  : FUTURE
func newSaver() *Saver {
	s := &Saver{}
	s.Kbinds = []int{}
	dir := s.directoryLocation()
	if err := os.MkdirAll(dir, 0755); err != nil {
		dir = ""
	}
	s.File = path.Join(dir, "bampf.save")
	return s
}

// persistBindings saves the new keybindings, while preserving the other
// information.
func (s *Saver) persistBindings(keys []int) {
	s.restore()
	s.Kbinds = keys
	s.persist()
}

// persistWindow saves the new window location and size, while preserving
// the other information.
func (s *Saver) persistWindow(x, y, w, h int, fullScreen bool) {
	s.restore()
	s.Full = fullScreen
	if !s.Full {
		// only save dimensions when not full screen.
		s.X, s.Y, s.W, s.H = x, y, w, h
	}
	s.persist()
}

// persistMute saves the mute preference while preserving
// the other information.
func (s *Saver) persistMute(isMuted bool) {
	s.restore()
	s.Mute = isMuted
	s.persist()
}

// persist is called to record any user preferences. This is expected
// to be called when a user preference changes.
func (s *Saver) persist() {
	data := &bytes.Buffer{}
	enc := gob.NewEncoder(data) // saves
	if err := enc.Encode(s); err == nil {
		if err = ioutil.WriteFile(s.File, data.Bytes(), 0644); err != nil {
			logf("Failed to save game state: %s", err)
		}
	} else {
		logf("Failed to encode game state: %s", err)
	}
}

// restore reads persisted information from disk. It handles the case where
// a previous restore file doesn't exist.
func (s *Saver) restore() {
	if bites, err := ioutil.ReadFile(s.File); err == nil {
		data := bytes.NewBuffer(bites)
		dec := gob.NewDecoder(data)
		if err := dec.Decode(s); err != nil {
			logf("Failed to restore game state. %s", err)
		}
	}
}

// reset clears the saved file.
func (s *Saver) reset() {
	os.Remove(s.File)
}
