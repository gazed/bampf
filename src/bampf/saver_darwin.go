// Copyright © 2013-2014 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"os"
	"path"
)

// directoryLocation gives the save file location for OSX.
//    osx   : /Users/[USER]/Library/Application\ Support/Bampf/
func (s *Saver) directoryLocation() string {
	return path.Join(os.Getenv("HOME"), "/Library/Application Support/Bampf/")
}
