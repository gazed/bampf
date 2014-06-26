// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

// react is a means of linking user key sequences to code that updates state.
// Applications can bind reactions to pressed key sequences.

import (
	"log"
	"vu"
)

// ReactionSet provides a mappable set of user key strokes to application
// action handlers. The reactions can be manipulated through their key or
// identifying label.
type ReactionSet interface {
	Add(r Reaction)       // Add or replaces a reaction by id.
	Rem(id string)        // Removes by reaction from the set.
	Key(id string) string // Get the key for the reaction.
	Id(key string) string // Get the id/label for the key.

	// Respond triggers a reaction in the set. Nothing happens if no reaction
	// matches the key parameter. The user input and time down for the given
	// key are passed through to the application action handler.
	Respond(key string, in *vu.Input, down int)

	// Rebind the given reaction to a new key. Rebinding that uses an existing
	// key will cause the keys to flip. Rebinding to non-existing reactions
	// are logged as development errors.
	Rebind(id, key string)
}

// NewReactionSet initializes a ReactionSet. The reaction identfiers and keys
// are expected to be unique and development errors will be logged if they are
// not.
func NewReactionSet(reacts []Reaction) ReactionSet {
	rs := &reactions{}
	rs.idmap = map[string]*reaction{}
	rs.keymap = map[string]*reaction{}
	if reacts != nil {
		for _, r := range reacts {
			rs.Add(r)
		}
	}
	return rs
}

// Reactions
// ===========================================================================
// reactions implementation

// reactions implements Reactions.
type reactions struct {
	idmap  map[string]*reaction
	keymap map[string]*reaction
}

// Add implements reactions.
func (rs *reactions) Add(react Reaction) {
	r := &reaction{react.Id, react.Key, react.Do}
	if _, ok := rs.idmap[r.id]; ok {
		log.Printf("Ignoring duplicate reaction id %s.", r.id)
		return
	}
	if _, ok := rs.keymap[r.key]; ok {
		log.Printf("Ignoring duplicate reaction key %s", r.key)
		return
	}
	rs.idmap[r.id] = r
	rs.keymap[r.key] = r
}

// Add implements reactions.
func (rs *reactions) Rem(id string) {
	if er, ok := rs.idmap[id]; ok {
		delete(rs.idmap, er.id)
		delete(rs.keymap, er.key)
	}
}

// Respond implements reactions.
func (rs *reactions) Respond(key string, in *vu.Input, down int) {
	if r, ok := rs.keymap[key]; ok {
		r.do(in, down)
	}
}

// Key implements reactions.
func (rs *reactions) Key(id string) string {
	if r, ok := rs.idmap[id]; ok {
		return r.key
	}
	return ""
}

// Id implements reactions.
func (rs *reactions) Id(key string) string {
	if r, ok := rs.keymap[key]; ok {
		return r.id
	}
	return ""
}

// Rebind implements reactions.
func (rs *reactions) Rebind(id, key string) {
	if er, ok := rs.idmap[id]; ok {
		delete(rs.keymap, er.key)
		if or, ok := rs.keymap[key]; ok {
			delete(rs.keymap, or.key)
			or.key = er.key
			rs.keymap[er.key] = or
		}
		er.key = key
		rs.keymap[key] = er
	} else {
		log.Printf("Rebind: Reaction %s does not exist", id)
	}
}

// reactions
// ===========================================================================
// Reaction

// Reaction facilities passing information to a ReactionSet. It can also be
// used as the basis for an application reaction implmentation. Note that
// changing the Reaction fields does not affect existing ReactionSet data.
type Reaction struct {
	Id  string          // Unique name.
	Key string          // Trigger sequence.
	Do  vu.InputHandler // Callback.
}

// reaction is an internal representation of Reaction. Primarily used to
// keep a copy of the reaction data that can't be directly manipulated by
// the application.
type reaction struct {
	id  string          // Unique name.
	key string          // Trigger sequence.
	do  vu.InputHandler // Callback.
}
