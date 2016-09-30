// Copyright © 2013-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"strconv"

	"github.com/gazed/vu"
	"github.com/gazed/vu/math/lin"
)

// hud is the 2D controller for all parts of the games heads-up-display (HUD).
type hud struct {
	area            // Hud fills up the full screen.
	root *vu.Pov    //
	cam  *vu.Camera // Scene camera.
	pl   *player    // Player model.
	xp   *xpbar     // Show cores collected and current energy.
	mm   *minimap   // Show overhead map centered on player.
	ce   *vu.Pov    // Cloaking effect.
	te   *vu.Pov    // Teleport effect.
	ee   *vu.Pov    // Energy loss effect.
}

// newHud creates all the various parts of the heads up display.
func newHud(eng vu.Eng, sentryCount, wx, wy, ww, wh int) *hud {
	hd := &hud{}
	hd.root = eng.Root().NewPov()
	hd.cam = hd.root.NewCam().SetUI()
	hd.setSize(wx, wy, ww, wh)

	// create the HUD parts.
	hd.pl = newPlayer(hd.root, hd.w, hd.h)
	hd.xp = newXpbar(hd.root, hd.w, hd.h)
	hd.mm = newMinimap(eng.Root().NewPov(), sentryCount)
	hd.ce = hd.cloakingEffect(hd.root)
	hd.te = hd.teleportEffect(hd.root)
	hd.ee = hd.energyLossEffect(hd.root)
	hd.resize(hd.w, hd.h)
	return hd
}

// setSize adjusts the size of the hud to the current screen dimensions.
func (hd *hud) setSize(screenX, screenY, screenWidth, screenHeight int) {
	hd.x, hd.y, hd.w, hd.h = 0, 0, screenWidth, screenHeight
	hd.cam.SetOrthographic(0, float64(hd.w), 0, float64(hd.h), 0, 10)
	hd.cx, hd.cy = hd.center()
}

// resize adapts the overlay to a new window size.
func (hd *hud) resize(screenWidth, screenHeight int) {
	hd.setSize(0, 0, screenWidth, screenHeight)
	hd.xp.resize(screenWidth, screenHeight)
	hd.mm.resize(screenWidth, screenHeight)

	// resize the animation effects.
	hd.ce.SetScale(float64(hd.w), float64(hd.h), 1)
	hd.ce.SetAt(hd.cx, hd.cy, -1)
	hd.te.SetScale(float64(hd.w), float64(hd.h), 1)
	hd.te.SetAt(hd.cx, hd.cy, -1)
	hd.ee.SetScale(float64(hd.w), float64(hd.h), 1)
	hd.ee.SetAt(hd.cx, hd.cy, -1)
}

// setVisible turns the HUD on/off. This is used when transitioning
// between levels.
func (hd *hud) setVisible(isVisible bool) {
	hd.root.Cull = !isVisible
	hd.mm.setVisible(isVisible)
}

// setLevel is called when a level transition happens.
func (hd *hud) setLevel(lvl *level) {
	hd.pl.setLevel(lvl)
	hd.xp.setLevel(lvl)
	hd.mm.setLevel(lvl.cam, lvl)
}

// have the hud wrap the minimap specifics so as to provide a single
// outside interface.
func (hd *hud) addWall(gamex, gamez float64)              { hd.mm.addWall(gamex, gamez) }
func (hd *hud) remCore(gamex, gamez float64)              { hd.mm.remCore(gamex, gamez) }
func (hd *hud) addCore(gamex, gamez float64)              { hd.mm.addCore(gamex, gamez) }
func (hd *hud) resetCores()                               { hd.mm.resetCores() }
func (hd *hud) update(c *vu.Camera, sentries []*sentinel) { hd.mm.update(c, sentries) }

// cloakingEffect creates the model shown when the user cloaks.
func (hd *hud) cloakingEffect(root *vu.Pov) *vu.Pov {
	ce := root.NewPov()
	ce.Cull = true
	ce.NewModel("uv", "msh:icon", "tex:cloakon").SetAlpha(0.5)
	return ce
}
func (hd *hud) cloakingActive(isActive bool) { hd.ce.Cull = !isActive }

// teleportEffect creates the model shown when the user teleports.
func (hd *hud) teleportEffect(root *vu.Pov) *vu.Pov {
	te := root.NewPov()
	te.Cull = true
	m := te.NewModel("uvra", "msh:icon", "tex:smoke")
	m.SetAlpha(0.5)
	m.SetUniform("spin", 10.0)
	m.SetUniform("fd", 1000)
	return te
}
func (hd *hud) teleportActive(isActive bool) { hd.te.Cull = !isActive }
func (hd *hud) teleportFade(alpha float64)   { hd.te.Model().SetAlpha(alpha) }

// energyLossEffect creates the model shown when the player gets hit
// by a sentinel.
func (hd *hud) energyLossEffect(root *vu.Pov) *vu.Pov {
	ee := root.NewPov()
	ee.Cull = true
	m := ee.NewModel("uvra", "msh:icon", "tex:loss")
	m.SetAlpha(0.5)
	m.SetUniform("fd", 1000)
	m.SetUniform("spin", 2.0)
	return ee
}
func (hd *hud) energyLossActive(isActive bool) { hd.ee.Cull = !isActive }
func (hd *hud) energyLossFade(alpha float64)   { hd.ee.Model().SetAlpha(alpha) }

// hud
// ===========================================================================
// player

// player shows the trooper model that corresponds to the player. This allows
// an alternative view, albeit less useful, of the current players health.
//
// Player can ignore resizes since it is in the lower left corner.
type player struct {
	cx, cy float64  // Center location.
	player *trooper // Composite model of the player.
	bg     *vu.Pov  // Health status background.
}

// newPlayer sets the player hud location and creates the white background.
func newPlayer(root *vu.Pov, screenWidth, screenHeight int) *player {
	pl := &player{}
	pl.cx, pl.cy = 100, 100
	pl.bg = root.NewPov().SetScale(110, 110, 1).SetAt(pl.cx, pl.cy, 0)
	pl.bg.NewModel("uv", "msh:icon", "tex:hudbg")
	return pl
}

// setLevel gives the player its tilt. Note that nothing else
// uses the player rotation/location fields.
func (pl *player) setLevel(lvl *level) {
	pl.player = lvl.player

	// twist the player about 15 degrees around X and 15 degrees around Z.
	pl.player.part.SetView(&lin.Q{X: 0.24, Y: 0.16, Z: 0.16, W: 0.95})
	pl.player.part.SetAt(pl.cx, pl.cy, 0)
}

// player
// ===========================================================================
// xpbar

// xpbar reflects the players health and energy statistics using different
// progress bars.
type xpbar struct {
	area
	border int      // Offset from the edge of the screen.
	linew  int      // Line width for the box.
	bh, bw int      // Bar height and width.
	bg     *vu.Pov  // Health background bar.
	fg     *vu.Pov  // Health foreground bar.
	cbg    *vu.Pov  // Cloak energy background bar.
	cfg    *vu.Pov  // Cloak energy foreground bar.
	tbg    *vu.Pov  // Teleport energy background bar.
	tfg    *vu.Pov  // Teleport energy foreground bar.
	hb     *vu.Pov  // Display health amount.
	hbw    int      // Display health width in pixels.
	tk     *vu.Pov  // Display teleport key.
	tkw    int      // Display key width in pixels.
	ck     *vu.Pov  // Display cloak key.
	ckw    int      // Display key width in pixels.
	tr     *trooper // Current player injected with SetStage.
}

// newXpbar creates all three status bars.
func newXpbar(root *vu.Pov, screenWidth, screenHeight int) *xpbar {
	xp := &xpbar{}
	xp.border = 5
	xp.linew = 2
	xp.setSize(screenWidth, screenHeight)

	// add the xp background and foreground bars.
	xp.bg = root.NewPov()
	xp.bg.NewModel("alpha", "msh:square", "mat:tblack")
	xp.fg = root.NewPov()
	xp.fg.NewModel("uv", "msh:icon", "tex:xpcyan", "tex:xpred")

	// add the xp bar text.
	xp.hb = root.NewPov()
	xp.hb.NewLabel("uv", "lucidiaSu22", "lucidiaSu22White")

	// teleport energy background and foreground bars.
	xp.tbg = root.NewPov()
	xp.tbg.NewModel("alpha", "msh:square", "mat:tblack")
	xp.tfg = root.NewPov()
	xp.tfg.NewModel("uv", "msh:icon", "tex:xpblue", "tex:xpred")

	// the teleport bar text.
	xp.tk = root.NewPov()
	xp.tk.NewLabel("uv", "lucidiaSu18", "lucidiaSu18White")

	// cloak energy background and foreground bars.
	xp.cbg = root.NewPov()
	xp.cbg.NewModel("alpha", "msh:square", "mat:tblack")
	xp.cfg = root.NewPov()
	xp.cfg.NewModel("uv", "msh:icon", "tex:xpblue")

	// the cloak bar text.
	xp.ck = root.NewPov()
	xp.ck.NewLabel("uv", "lucidiaSu18", "lucidiaSu18White")
	xp.resize(screenWidth, screenHeight)
	return xp
}

// resize adjusts the graphics to fit the new window dimensions.
func (xp *xpbar) resize(screenWidth, screenHeight int) {
	xp.setSize(screenWidth, screenHeight)
	xp.bg.SetAt(xp.cx+5, xp.cy+5, 1)
	xp.bg.SetScale(float64(xp.bw/2), float64(xp.bh-xp.y), 1)

	// adjust the teleport energy bar.
	xp.tbg.SetAt(xp.cx-float64(xp.w)/10, xp.cy+35, 1)
	xp.tbg.SetScale(float64(xp.bw/10), float64(xp.bh-xp.y)-5, 1)
	bw := xp.tkw
	xp.tk.SetAt(xp.cx-float64(xp.bw)/10-float64(bw/2), xp.cy+26, 0)

	// adjust the cloaking energy bar.
	xp.cbg.SetAt(xp.cx+float64(xp.bw)/10, xp.cy+35, 1)
	xp.cbg.SetScale(float64(xp.bw/10), float64(xp.bh-xp.y)-5, 1)
	bw = xp.ckw
	xp.ck.SetAt(xp.cx+float64(xp.bw)/10-float64(bw/2), xp.cy+26, 0)

	// adjust the energy amounts for the bars.
	if xp.tr != nil {
		xp.healthUpdated(xp.tr.health())
		xp.energyUpdated(xp.tr.energy())
	}
}

// setSize adjusts the xpbars area according to the given screen dimensions.
func (xp *xpbar) setSize(screenWidth, screenHeight int) {
	xp.x, xp.y = 5, 5 // bottom left corner.
	xp.w, xp.h = screenWidth, screenHeight
	xp.bw, xp.bh = screenWidth-2*xp.border, 20
	xp.cx, xp.cy = float64(screenWidth)*0.5-float64(xp.border), float64(xp.bh)*0.5+float64(xp.border)
}

// healthMonitor:healthUpdated. Updates the health banner when it changes.
func (xp *xpbar) healthUpdated(health, warn, high int) {
	maxCores := high / gameCellGain[xp.tr.lvl-1]
	coresNeeded := (high - health) / gameCellGain[xp.tr.lvl-1]
	coreCount := strconv.Itoa(maxCores-coresNeeded) + "/" + strconv.Itoa(maxCores)
	xp.hb.Model().SetStr(coreCount)
	xp.hbw = xp.hb.Model().StrWidth()
	xp.hb.SetAt(xp.cx-float64(xp.hbw/2), xp.cy*0.5, 0)

	// turn on the warning colour if player has less than the starting amount of cores.
	barMax := float64(xp.bw/2 - xp.linew)
	if health >= warn {
		xp.fg.Model().OrderTex("xpcyan", 0)
	} else {
		xp.fg.Model().OrderTex("xpred", 0)
	}
	healthBar := float64(health) / float64(high) * barMax
	zeroSpot := float64(xp.border) + healthBar + float64(xp.linew-xp.border)
	xp.fg.SetAt(zeroSpot+5, xp.cy+5, 0)
	xp.fg.SetScale(healthBar, float64(xp.bh-xp.y-xp.linew)-1, 1)
}

// energyMonitor:energyUpdated. Update the energy banner when it changes.
func (xp *xpbar) energyUpdated(teleportEnergy, tmax, cloakEnergy, cmax int) {
	tratio := float64(teleportEnergy) / float64(tmax)
	if tratio == 1.0 {
		xp.tfg.Model().OrderTex("xpblue", 0)
	} else {
		xp.tfg.Model().OrderTex("xpred", 0)
	}
	xp.tfg.SetAt(xp.cx-float64(xp.w)/10, xp.cy+35, 0)
	xp.tfg.SetScale((float64(xp.bw/10))*tratio, float64(xp.bh-xp.y)-7, 1)
	cratio := float64(cloakEnergy) / float64(cmax)
	xp.cfg.SetAt(xp.cx+float64(xp.w)/10-1, xp.cy+35, 0)
	xp.cfg.SetScale((float64(xp.bw/10))*cratio, float64(xp.bh-xp.y)-7, 1)
}

// setLevel sets the xpbars values and must be called at least once before rendering.
func (xp *xpbar) setLevel(lvl *level) {
	xp.tr = lvl.player
	xp.tr.monitorHealth("xpbar", xp)
	xp.tr.monitorEnergy("xpbar", xp)
	xp.healthUpdated(xp.tr.health())
	xp.energyUpdated(xp.tr.energy())
}

// updateKeys needs to be called on startup and whenever the displayed key
// mappings are changed.
func (xp *xpbar) updateKeys(teleportKey, cloakKey int) {
	if xp.tk != nil && xp.ck != nil {
		if tsym := vu.Keysym(teleportKey); tsym > 0 {
			xp.tk.Model().SetStr(string(tsym))
		}
		if csym := vu.Keysym(cloakKey); csym > 0 {
			xp.ck.Model().SetStr(string(csym))
		}
	}
}

// xpbar
// ===========================================================================
// minimap

// minimap displays a limited portion of the current level from the overhead
// 2D perspective.
type minimap struct {
	area              // Rectangular area.
	root   *vu.Pov    // Root in scene hierarchy.
	cam    *vu.Camera // Xztoxy camera.
	cores  []*vu.Pov  // Keep track of the cores for removal.
	part   *vu.Pov    // Used to transform all the minimap models.
	bg     *vu.Pov    // The white background.
	scale  float64    // Minimap sizing.
	ppm    *vu.Pov    // Player position marker.
	cpm    *vu.Pov    // Center of map position marker.
	spms   []*vu.Pov  // Sentry position markers.
	radius int        // How much of the map is displayed from the center.
}

// newMinimap initializes the minimap. It still needs to be populated.
func newMinimap(root *vu.Pov, numTroops int) *minimap {
	mm := &minimap{root: root}
	mm.radius = 120
	mm.scale = 5.0
	mm.cores = []*vu.Pov{}
	mm.cam = root.NewCam()
	mm.cam.SetUI()
	mm.cam.Cull = vu.NewRadiusCull(float64(mm.radius))
	mm.cam.Vt = vu.XzXy

	// create the parent for all the visible minimap pieces.
	mm.part = root.NewPov().SetAt(float64(mm.x), 0, float64(-mm.y))

	// add the white background.
	mm.bg = mm.part.NewPov().SetScale(110, 1, 110)
	mm.bg.NewModel("uv", "msh:icon_xz", "tex:hudbg")

	// create the sentinel position markers
	mm.spms = []*vu.Pov{}
	for cnt := 0; cnt < numTroops; cnt++ {
		tpm := mm.part.NewPov().SetScale(mm.scale, mm.scale, mm.scale)
		tpm.NewModel("alpha", "msh:square_xz", "mat:tred")
		mm.spms = append(mm.spms, tpm)
	}

	// create the player marker and center map marker.
	mm.cpm = mm.part.NewPov().SetScale(mm.scale, mm.scale, mm.scale)
	mm.cpm.NewModel("alpha", "msh:square_xz", "mat:blue")
	mm.ppm = mm.part.NewPov().SetScale(mm.scale, mm.scale, mm.scale)
	mm.ppm.NewModel("alpha", "msh:tri_xz", "mat:tblack")
	return mm
}

// setVisible (un)hides all the minimap objects.
func (mm *minimap) setVisible(isVisible bool) {
	mm.root.Cull = !isVisible
}

// resize is responsible for keeping the minimap at the bottom
// right corner of the application window.
func (mm *minimap) resize(width, height int) {
	mm.setSize(0, 0, width, height)
	mm.part.SetAt(float64(mm.x), 0, float64(-mm.y))
}

// setSize adjusts the scene perspective to the given window size.
// Generally this is 1 pixel to 1 unit for HUD type scenes.
func (mm *minimap) setSize(x, y, width, height int) {
	mm.x, mm.y, mm.w, mm.h = width-mm.radius-10, 125, width, height
	mm.cam.SetOrthographic(0, float64(mm.w), 0, float64(mm.h), 0, 10)
}

// setLevel is called when a level transition happens.
func (mm *minimap) setLevel(cam *vu.Camera, lvl *level) {
	x, y, z := cam.At()
	mm.cam.SetAt(x*mm.scale, y*mm.scale, z*mm.scale)

	// adjust the center location based on the game maze center.
	mm.cx, mm.cy = float64(lvl.gcx*lvl.units)*mm.scale, float64(-lvl.gcy*lvl.units)*mm.scale
	mm.ppm.SetAt(x, y, z)
	mm.bg.SetAt(x, y, z)
	mm.ppm.SetView(cam.Look)
	mm.setSentryAt(lvl.sentries)
	lvl.player.monitorHealth("mmap", mm)
}

// addWall adds a block representing a wall to the minimap.
func (mm *minimap) addWall(x, y float64) {
	wall := mm.part.NewPov().SetScale(mm.scale, mm.scale, mm.scale)
	wall.SetAt(x*mm.scale, 0, y*mm.scale)
	wall.NewModel("alpha", "msh:square_xz", "mat:gray")
}

// addCore adds a small block representing an energy core to the minimap.
func (mm *minimap) addCore(gamex, gamey float64) {
	scale := mm.scale
	cm := mm.part.NewPov()
	cm.SetAt(gamex*scale, 0, gamey*scale)
	scale *= 0.5
	cm.SetScale(scale, scale, scale)
	cm.NewModel("alpha", "msh:square_xz", "mat:green")
	mm.cores = append(mm.cores, cm)
}

// remCore removes the energy core from the minimap.
func (mm *minimap) remCore(gamex, gamez float64) {
	scale := mm.scale
	gx, gz := lin.Round(gamex, 0)*scale, lin.Round(gamez, 0)*scale
	for index, core := range mm.cores {
		cx, _, cz := core.At()
		cx, cz = lin.Round(cx, 0), lin.Round(cz, 0)
		if cx == gx && cz == gz {
			core.Dispose(vu.PovNode)
			mm.cores = append(mm.cores[:index], mm.cores[index+1:]...)
			return
		}
	}
	logf("hud.mapOverlay.remCore: failed to remove a core.")
}

// resetCores is expected to be called when switching levels so that
// this level is clear of cores the next time it is activated.
func (mm *minimap) resetCores() {
	for _, core := range mm.cores {
		core.Dispose(vu.PovNode)
	}
	mm.cores = []*vu.Pov{}
}

// healthMonitor:healthUpdated. Update the center colour of the maze
// based on the player health.
func (mm *minimap) healthUpdated(health, warn, high int) {
	if health == high {
		mm.cpm.Model().Load("mat:green")
	} else {
		mm.cpm.Model().Load("mat:blue")
	}
}

// update adjusts the minimap according to the players new position.
func (mm *minimap) update(cam *vu.Camera, sentries []*sentinel) {
	scale := mm.scale
	x, y, z := cam.At()
	x, y, z = x*scale, y*scale, z*scale
	mm.cam.SetAt(x, y, z)
	mm.setPlayerAt(x, y, z)
	mm.setPlayerRotation(cam.Look)
	mm.setCenterAt(x, y, z)
	mm.setSentryAt(sentries)
}

// set the position of the player marker by mirroring the game camera.
func (mm *minimap) setPlayerAt(x, y, z float64) {
	mm.ppm.SetAt(x, y, z)
	mm.bg.SetAt(x, y, z)
}
func (mm *minimap) setPlayerRotation(dir *lin.Q) { mm.ppm.SetView(dir) }

// set the position of the maze center marker.
func (mm *minimap) setCenterAt(x, y, z float64) {
	toc := &lin.V3{X: x - mm.cx, Y: y, Z: z - mm.cy} // vector from player to center
	dtoc := toc.Len()                                // distance to center
	mm.cpm.SetAt(mm.cx, 0, mm.cy)                    // set marker at center...
	if dtoc > float64(mm.radius) {                   // ... unless the distance is to great
		toc.Unit().Scale(toc, float64(mm.radius))
		mm.cpm.SetAt(x-toc.X, y, z-toc.Z)
	}
}

// set the position for all the sentry markers.
func (mm *minimap) setSentryAt(sentinels []*sentinel) {
	if len(mm.spms) == len(sentinels) {
		for cnt, sentry := range sentinels {
			tpm := mm.spms[cnt]
			x, y, z := sentry.location()
			x, y, z = x*mm.scale, y*mm.scale, z*mm.scale
			tpm.SetAt(x, y, z)
		}
	} else {
		logf("hud.minimap.setSentryAt: sentry length mismatch")
	}
}
