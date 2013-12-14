// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// Saver persists any game state that needs to be remembered between one game
// session and the next. Saver needs to be public and visible for the encoding package.
type Saver struct {
	File       string            // Save file name.
	Kmap       map[string]string // Key bindings.
	X, Y, W, H int               // Window location.
	Mute       bool              // True if the game is muted.
}

// newSaver creates default persistent application state. The directory is
// platform specific and specified by:
//    osx  : see saver_darwin.go
//    win  : see saver_windows.go
//    lin  : FUTURE
func newSaver() *Saver {
	s := &Saver{}
	s.Kmap = map[string]string{}
	dir := s.directoryLocation()
	if err := os.MkdirAll(dir, 0755); err != nil {
		dir = ""
	}
	s.File = path.Join(dir, "bampf.save")
	return s
}

// persistBindings saves the new keybindings, while preserving the other information.
func (s *Saver) persistBindings(keymap map[string]string) {
	s.restore()
	s.Kmap = keymap
	s.persist()
}

// persistWindow saves the new window location and size, while preserving
// the other information.
func (s *Saver) persistWindow(x, y, w, h int) {
	s.restore()
	s.X, s.Y, s.W, s.H = x, y, w, h
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
			log.Printf("Failed to save game state: %s", err)
		}
	} else {
		log.Printf("Failed to encode game state: %s", err)
	}
}

// restore reads persisted information from disk. It handles the case where
// a previous restore file doesn't exist.
func (s *Saver) restore() *Saver {
	if bites, err := ioutil.ReadFile(s.File); err == nil {
		data := bytes.NewBuffer(bites)
		dec := gob.NewDecoder(data)
		if err := dec.Decode(s); err != nil {
			log.Printf("Failed to restore game state. %s", err)
		}
	}
	return s
}

// reset clears the saved file.
func (s *Saver) reset() {
	os.Remove(s.File)
}
