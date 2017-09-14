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
	ui   *vu.Ent  // 2D scene.
	area          // Hud fills up the full screen.
	pl   *player  // Player model.
	xp   *xpbar   // Show cores collected and current energy.
	mm   *minimap // Show overhead map centered on player.
	ce   *vu.Ent  // Cloaking effect.
	te   *vu.Ent  // Teleport effect.
	ee   *vu.Ent  // Energy loss effect.
}

// newHud creates all the various parts of the heads up display.
func newHud(eng vu.Eng, sentryCount, wx, wy, ww, wh int) *hud {
	hd := &hud{}
	hd.ui = eng.AddScene().SetUI()
	hd.ui.Cam().SetClip(0, 10)
	hd.setSize(wx, wy, ww, wh)

	// create the HUD parts.
	hd.pl = newPlayer(hd.ui.AddPart(), hd.w, hd.h)
	hd.xp = newXpbar(hd.ui, hd.w, hd.h)
	hd.mm = newMinimap(eng, sentryCount)
	hd.ce = hd.cloakingEffect(hd.ui.AddPart())
	hd.te = hd.teleportEffect(hd.ui.AddPart())
	hd.ee = hd.energyLossEffect(hd.ui.AddPart())
	hd.resize(hd.w, hd.h)
	return hd
}

// setSize adjusts the size of the hud to the current screen dimensions.
func (hd *hud) setSize(screenX, screenY, screenWidth, screenHeight int) {
	hd.x, hd.y, hd.w, hd.h = 0, 0, screenWidth, screenHeight
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
	hd.ui.Cull(!isVisible)
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
func (hd *hud) cloakingEffect(ce *vu.Ent) *vu.Ent {
	ce.Cull(true)
	ce.MakeModel("uv", "msh:icon", "tex:cloakon")
	ce.SetAlpha(0.5)
	return ce
}
func (hd *hud) cloakingActive(isActive bool) { hd.ce.Cull(!isActive) }

// teleportEffect creates the model shown when the user teleports.
func (hd *hud) teleportEffect(te *vu.Ent) *vu.Ent {
	te.Cull(true)
	m := te.MakeModel("uvra", "msh:icon", "tex:smoke")
	m.SetAlpha(0.5).SetUniform("spin", 10.0).SetUniform("fd", 1000)
	return te
}
func (hd *hud) teleportActive(isActive bool) { hd.te.Cull(!isActive) }
func (hd *hud) teleportFade(alpha float64) {
	hd.te.SetAlpha(lin.Clamp(alpha, 0, 1))
}

// energyLossEffect creates the model shown when the player gets hit
// by a sentinel.
func (hd *hud) energyLossEffect(ee *vu.Ent) *vu.Ent {
	ee.Cull(true)
	m := ee.MakeModel("uvra", "msh:icon", "tex:loss")
	m.SetAlpha(0.5).SetUniform("fd", 1000).SetUniform("spin", 2.0)
	return ee
}
func (hd *hud) energyLossActive(isActive bool) { hd.ee.Cull(!isActive) }
func (hd *hud) energyLossFade(alpha float64) {
	hd.ee.SetAlpha(lin.Clamp(alpha, 0, 1))
}

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
	bg     *vu.Ent  // Health status background.
}

// newPlayer sets the player hud location and creates the white background.
func newPlayer(pov *vu.Ent, screenWidth, screenHeight int) *player {
	pl := &player{}
	pl.cx, pl.cy = 100, 100
	pl.bg = pov.SetScale(110, 110, 1).SetAt(pl.cx, pl.cy, 0)
	pl.bg.MakeModel("uv", "msh:icon", "tex:hudbg")
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
	bg     *vu.Ent  // Health background bar.
	fg     *vu.Ent  // Health foreground bar.
	cbg    *vu.Ent  // Cloak energy background bar.
	cfg    *vu.Ent  // Cloak energy foreground bar.
	tbg    *vu.Ent  // Teleport energy background bar.
	tfg    *vu.Ent  // Teleport energy foreground bar.
	hb     *vu.Ent  // Display health amount.
	hbw    int      // Display health width in pixels.
	tk     *vu.Ent  // Display teleport key.
	tkw    int      // Display key width in pixels.
	ck     *vu.Ent  // Display cloak key.
	ckw    int      // Display key width in pixels.
	tr     *trooper // Current player injected with SetStage.
}

// newXpbar creates all three status bars.
func newXpbar(scene *vu.Ent, screenWidth, screenHeight int) *xpbar {
	xp := &xpbar{}
	xp.border = 5
	xp.linew = 2
	xp.setSize(screenWidth, screenHeight)

	// add the xp background and foreground bars.
	xp.bg = scene.AddPart()
	xp.bg.MakeModel("alpha", "msh:square", "mat:tgray")
	xp.fg = scene.AddPart()
	xp.fg.MakeModel("uv", "msh:icon", "tex:xpcyan", "tex:xpred")

	// add the xp bar text.
	xp.hb = scene.AddPart()
	xp.hb.MakeLabel("uv", "lucidiaSu22")

	// teleport energy background and foreground bars.
	xp.tbg = scene.AddPart()
	xp.tbg.MakeModel("alpha", "msh:square", "mat:tgray")
	xp.tfg = scene.AddPart()
	xp.tfg.MakeModel("uv", "msh:icon", "tex:xpblue", "tex:xpred")

	// the teleport bar text.
	xp.tk = scene.AddPart().MakeLabel("uv", "lucidiaSu18")

	// cloak energy background and foreground bars.
	xp.cbg = scene.AddPart()
	xp.cbg.MakeModel("alpha", "msh:square", "mat:tgray")
	xp.cfg = scene.AddPart()
	xp.cfg.MakeModel("uv", "msh:icon", "tex:xpblue")

	// the cloak bar text.
	xp.ck = scene.AddPart().MakeLabel("uv", "lucidiaSu18")
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
	xp.hb.Typeset(coreCount)
	xp.hbw, _ = xp.hb.Size()
	xp.hb.SetAt(xp.cx-float64(xp.hbw/2), xp.cy*0.5, 0)

	// turn on the warning colour if player has less than the starting amount of cores.
	barMax := float64(xp.bw/2 - xp.linew)
	if health >= warn {
		xp.fg.SetFirst("xpcyan")
	} else {
		xp.fg.SetFirst("xpred")
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
		xp.tfg.SetFirst("xpblue")
	} else {
		xp.tfg.SetFirst("xpred")
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
		if tsym := vu.Symbol(teleportKey); tsym > 0 {
			xp.tk.Typeset(string(tsym))
		}
		if csym := vu.Symbol(cloakKey); csym > 0 {
			xp.ck.Typeset(string(csym))
		}
	}
}

// xpbar
// ===========================================================================
// minimap

// minimap displays a limited portion of the current level from the overhead
// 2D perspective.
type minimap struct {
	ui     *vu.Ent   // 2D overlay scene.
	area             // Rectangular area.
	cores  []*vu.Ent // Keep track of the cores for removal.
	top    *vu.Ent   // Map scale and position on screen.
	root   *vu.Ent   // Reposition map as player move.s
	bg     *vu.Ent   // The white background.
	scale  float64   // Minimap sizing.
	ppm    *vu.Ent   // Player position marker.
	cpm    *vu.Ent   // Center of map position marker.
	spms   []*vu.Ent // Sentry position markers.
	radius int       // Limits map visibility. Distance squared in pixels.
}

// newMinimap initializes the minimap. It still needs to be populated.
func newMinimap(eng vu.Eng, numTroops int) *minimap {
	mm := &minimap{}
	mm.radius = 120
	mm.scale = 5.0
	mm.cores = []*vu.Ent{}
	mm.ui = eng.AddScene().SetUI()
	mm.ui.Cam().SetClip(0, 10)
	mm.ui.SetCuller(mm) // mm implements Culler

	// parent for all the visible minimap pieces.
	mm.top = mm.ui.AddPart().SetScale(mm.scale, mm.scale, 1)
	mm.root = mm.top.AddPart()

	// add the white background to highlight player marker.
	mm.bg = mm.root.AddPart().SetScale(110, 110, 1)
	mm.bg = mm.root.AddPart().SetScale(110, 110, 1)
	mm.bg.MakeModel("uv", "msh:icon", "tex:hudbg")

	// create the sentinel position markers
	mm.spms = []*vu.Ent{}
	for cnt := 0; cnt < numTroops; cnt++ {
		tpm := mm.root.AddPart()
		tpm.MakeModel("alpha", "msh:square", "mat:tred")
		mm.spms = append(mm.spms, tpm)
	}

	// create the player marker and center map marker.
	mm.cpm = mm.root.AddPart()
	mm.cpm.MakeModel("alpha", "msh:square", "mat:blue")
	mm.ppm = mm.root.AddPart()
	mm.ppm.MakeModel("alpha", "msh:tri", "mat:tblack")
	return mm
}

// setVisible (un)hides all the minimap objects.
func (mm *minimap) setVisible(isVisible bool) {
	mm.ui.Cull(!isVisible)
}

// Culled returns true if the given Pov is to far away from the player.
// Used to limit the minimap view to map elements close to the player.
func (mm *minimap) Culled(cam *vu.Camera, wx, wy, wz float64) bool {
	px, py, _ := mm.ppm.World()
	dx := px - wx
	dy := py - wy
	return (dx*dx + dy*dy) > float64(mm.radius*mm.radius)
}

// resize is responsible for keeping the minimap at the bottom
// right corner of the application window.
func (mm *minimap) resize(width, height int) {
	mm.x, mm.y, mm.w, mm.h = width-mm.radius-10, 125, width, height
	mm.top.SetAt(float64(mm.x), float64(mm.y), 0)
}

// setLevel is called when a level transition happens.
func (mm *minimap) setLevel(cam *vu.Camera, lvl *level) {
	x, _, z := cam.At()

	// adjust the center location based on the game maze center.
	mm.cx, mm.cy = float64(lvl.gcx*lvl.units), float64(lvl.gcy*lvl.units)
	mm.ppm.SetAt(x, -z, 0)
	mm.bg.SetAt(x, -z, 0)
	mm.ppm.View().SetAa(0, 0, 1, lin.Rad(cam.Yaw))
	mm.setSentryAt(lvl.sentries)
	lvl.player.monitorHealth("mmap", mm)
}

// addWall adds a block representing a wall to the minimap.
func (mm *minimap) addWall(x, y float64) {
	wall := mm.root.AddPart().SetAt(x, -y, 0)
	wall.MakeModel("alpha", "msh:square", "mat:gray")
}

// addCore adds a small block representing an energy core to the minimap.
func (mm *minimap) addCore(gamex, gamez float64) {
	cm := mm.root.AddPart().SetAt(gamex, -gamez, 0).SetScale(0.5, 0.5, 1)
	cm.MakeModel("alpha", "msh:square", "mat:green")
	mm.cores = append(mm.cores, cm)
}

// remCore removes a collected energy core from the minimap.
func (mm *minimap) remCore(gamex, gamez float64) {
	gx, gy := lin.Round(gamex, 0), lin.Round(-gamez, 0)
	for index, core := range mm.cores {
		cx, cy, _ := core.At()
		cx, cy = lin.Round(cx, 0), lin.Round(cy, 0)
		if cx == gx && cy == gy {
			core.Dispose()
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
		core.Dispose()
	}
	mm.cores = []*vu.Ent{}
}

// healthMonitor:healthUpdated. Update the center colour of the maze
// based on the player health.
func (mm *minimap) healthUpdated(health, warn, high int) {
	if health == high {
		mm.cpm.SetColor(0, 0.62, 0.6)
	} else {
		mm.cpm.SetColor(0.4, 0.5, 0.8)
	}
}

// update adjusts the minimap according to the players new position.
func (mm *minimap) update(cam *vu.Camera, sentries []*sentinel) {
	x, _, z := cam.At()
	mm.root.SetAt(-x, z, 0)
	mm.setCenterAt(x, -z)
	mm.bg.SetAt(x, -z, 0)
	mm.ppm.SetAt(x, -z, 0)
	mm.ppm.View().SetAa(0, 0, 1, lin.Rad(cam.Yaw))
	mm.setSentryAt(sentries)
}

// set the position of the maze center marker. Ensure the center marker
// is always visible to the player knows where the maze is if they wander
// to far away.
func (mm *minimap) setCenterAt(x, y float64) {
	radius := float64(mm.radius) / 5.1
	toc := &lin.V3{X: x - mm.cx, Y: y - mm.cy, Z: 0} // vector from player to center
	dtoc := toc.Len()                                // distance to center
	mm.cpm.SetAt(mm.cx, mm.cy, 0)                    // set marker at center...
	if dtoc > radius {                               // ... unless the distance is to great
		toc.Unit().Scale(toc, radius)
		mm.cpm.SetAt(x-toc.X, y-toc.Y, 0)
	}
}

// set the position for all the sentry markers.
func (mm *minimap) setSentryAt(sentinels []*sentinel) {
	if len(mm.spms) != len(sentinels) {
		logf("hud.minimap.setSentryAt: sentry length mismatch")
		return
	}
	for cnt, sentry := range sentinels {
		tpm := mm.spms[cnt]
		x, _, z := sentry.location()
		tpm.SetAt(x, -z, 0)
	}
}
