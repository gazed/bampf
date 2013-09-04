// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"os"
	"path"
)

// directoryLocation gives the save file location for Windows.
//    win  : C:\Users\[USER]\AppData\Local\Bampf\bampf.save
func (s *Saver) directoryLocation() string {
	return path.Join(os.Getenv("LOCALAPPDATA"), "Bampf/")
}
