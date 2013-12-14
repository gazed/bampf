// Copyright © 2013 Galvanized Logic Inc.
// Use is governed by a FreeBSD license found in the LICENSE file.

package main

import (
	"log"
	"strconv"
	"vu"
	"vu/math/lin"
)

// hud is the 2D controller for all parts of the games heads-up-display (HUD).
type hud struct {
	area           // Hud fills up the full screen.
	scene vu.Scene // Scene graph plus camera and lighting.
	pl    *player  // Player model.
	xp    *xpbar   // Show cores collected and current energy.
	mm    *minimap // Show overhead map centered on player.
	ce    vu.Part  // Cloaking effect.
	te    vu.Part  // Teleport effect.
	ee    vu.Part  // Energy loss effect.
}

// newHud creates all the various parts of the heads up display.
func newHud(eng vu.Engine, sentryCount int) *hud {
	hd := &hud{}
	hd.scene = eng.AddScene(vu.VO)
	hd.scene.Set2D()
	hd.setSize(eng.Size())

	// create the HUD parts.
	hd.pl = newPlayer(eng, hd.scene, hd.w, hd.h)
	hd.xp = newXpbar(eng, hd.scene, hd.w, hd.h)
	hd.mm = newMinimap(eng, sentryCount)
	hd.ce = hd.cloakingEffect(hd.scene)
	hd.te = hd.teleportEffect(hd.scene)
	hd.ee = hd.energyLossEffect(hd.scene)
	hd.resize(hd.w, hd.h)
	return hd
}

// setSize adjusts the size of the hud to the current screen dimensions.
func (hd *hud) setSize(screenX, screenY, screenWidth, screenHeight int) {
	hd.x, hd.y, hd.w, hd.h = 0, 0, screenWidth, screenHeight
	hd.scene.SetOrthographic(0, float64(hd.w), 0, float64(hd.h), 0, 10)
	hd.cx, hd.cy = hd.center()
}

// resize adapts the overlay to a new window size.
func (hd *hud) resize(screenWidth, screenHeight int) {
	hd.setSize(0, 0, screenWidth, screenHeight)
	hd.xp.resize(screenWidth, screenHeight)
	hd.mm.resize(screenWidth, screenHeight)

	// resize the animation effects.
	hd.ce.SetScale(float64(hd.w), float64(hd.h), 1)
	hd.ce.SetLocation(hd.cx, hd.cy, -1)
	hd.te.SetScale(float64(hd.w), float64(hd.h), 1)
	hd.te.SetLocation(hd.cx, hd.cy, -1)
	hd.ee.SetScale(float64(hd.w), float64(hd.h), 1)
	hd.ee.SetLocation(hd.cx, hd.cy, -1)
}

// setVisible turns the HUD on/off. This is used when transitioning between
// levels.
func (hd *hud) setVisible(isVisible bool) {
	hd.scene.SetVisible(isVisible)
	hd.mm.setVisible(isVisible)
}

// setLevel is called when a level transition happens.
func (hd *hud) setLevel(lvl *level) {
	hd.pl.setLevel(lvl)
	hd.xp.setLevel(lvl)
	hd.mm.setLevel(lvl.scene, lvl)
}

// have the hud wrap the minimap specifics so as to provide a single
// outside interface.
func (hd *hud) addWall(gamex, gamez float64)             { hd.mm.addWall(gamex, gamez) }
func (hd *hud) remCore(gamex, gamez float64)             { hd.mm.remCore(gamex, gamez) }
func (hd *hud) addCore(gamex, gamez float64)             { hd.mm.addCore(gamex, gamez) }
func (hd *hud) resetCores()                              { hd.mm.resetCores() }
func (hd *hud) update(sc vu.Scene, sentries []*sentinel) { hd.mm.update(sc, sentries) }

// cloakingEffect creates the model shown when the user cloaks.
func (hd *hud) cloakingEffect(scene vu.Scene) vu.Part {
	ce := scene.AddPart()
	ce.SetFacade("icon", "uv").SetMaterial("half")
	ce.SetTexture("cloakon", 0)
	ce.SetVisible(false)
	return ce
}
func (hd *hud) cloakingActive(isActive bool) { hd.ce.SetVisible(isActive) }

// teleportEffect creates the model shown when the user teleports.
func (hd *hud) teleportEffect(scene vu.Scene) vu.Part {
	te := scene.AddPart()
	te.SetFacade("icon", "uvra").SetMaterial("half")
	te.SetTexture("smoke", 10)
	te.SetVisible(false)
	return te
}
func (hd *hud) teleportActive(isActive bool) { hd.te.SetVisible(isActive) }
func (hd *hud) teleportFade(alpha float64)   { hd.te.SetAlpha(alpha) }

// energyLossEffect creates the model shown when the player gets hit
// by a sentinel.
func (hd *hud) energyLossEffect(scene vu.Scene) vu.Part {
	ee := scene.AddPart()
	ee.SetFacade("icon", "uvra").SetMaterial("half")
	ee.SetTexture("loss", 0)
	ee.SetVisible(false)
	return ee
}
func (hd *hud) energyLossActive(isActive bool) { hd.ee.SetVisible(isActive) }
func (hd *hud) energyLossFade(alpha float64)   { hd.ee.SetAlpha(alpha) }

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
	bg     vu.Part  // Health status background.
}

// newPlayer sets the player hud location and creates the white background.
func newPlayer(eng vu.Engine, scene vu.Scene, screenWidth, screenHeight int) *player {
	pl := &player{}
	pl.cx, pl.cy = 100, 100
	pl.bg = scene.AddPart()
	pl.bg.SetFacade("icon", "uvm").SetMaterial("green")
	pl.bg.SetTexture("hudbg", 0)
	pl.bg.SetScale(110, 110, 1)
	pl.bg.SetLocation(pl.cx, pl.cy, 0)
	return pl
}

// setLevel gives the player its tilt. Note that nothing else
// uses the player rotation/location fields.
func (pl *player) setLevel(lvl *level) {
	pl.player = lvl.player

	// twist the player about 15 degrees around X and 15 degrees around Z.
	pl.player.part.SetRotation(0.24, 0.16, 0.16, 0.95)
	pl.player.part.SetLocation(pl.cx, pl.cy, 0)
}

// player
// ===========================================================================
// xpbar

// xpbar reflects the players health and energy statistics using different
// progress bars.
type xpbar struct {
	area
	eng    vu.Engine
	border int      // Offset from the edge of the screen.
	linew  int      // Line width for the box.
	bh, bw int      // Bar height and width.
	bg     vu.Part  // Health background bar.
	fg     vu.Part  // Health foreground bar.
	cbg    vu.Part  // Cloak energy background bar.
	cfg    vu.Part  // Cloak energy foreground bar.
	tbg    vu.Part  // Teleport energy background bar.
	tfg    vu.Part  // Teleport energy foreground bar.
	hb     vu.Part  // Display health amount.
	tk     vu.Part  // Display teleport key.
	ck     vu.Part  // Display cloak key.
	tr     *trooper // Current player injected with SetStage.
}

// newXpbar creates all three status bars.
func newXpbar(e vu.Engine, scene vu.Scene, screenWidth, screenHeight int) *xpbar {
	xp := &xpbar{}
	xp.eng = e
	xp.border = 5
	xp.linew = 2
	xp.setSize(screenWidth, screenHeight)

	// add the xp background and foreground bars.
	xp.bg = scene.AddPart()
	xp.bg.SetFacade("square", "flat").SetMaterial("tblack")
	xp.fg = scene.AddPart()
	xp.fg.SetFacade("icon", "uv").SetMaterial("green")
	xp.fg.SetTexture("xpgreen", 0)

	// add the xp bar text.
	xp.hb = scene.AddPart()
	xp.hb.SetBanner("0", "uv", "weblySleek22", "weblySleek22White")
	bw := xp.hb.BannerWidth()
	xp.hb.SetLocation(xp.cx-float64(bw/2), xp.cy*0.5-5, 0)

	// teleport energy background and foreground bars.
	xp.tbg = scene.AddPart()
	xp.tbg.SetFacade("square", "flat").SetMaterial("tblack")
	xp.tfg = scene.AddPart()
	xp.tfg.SetFacade("icon", "uv").SetMaterial("green")
	xp.tfg.SetTexture("xpblue", 0)

	// the teleport bar text.
	xp.tk = scene.AddPart()
	xp.tk.SetBanner("0", "uv", "weblySleek16", "weblySleek16White")

	// cloak energy background and foreground bars.
	xp.cbg = scene.AddPart()
	xp.cbg.SetFacade("square", "flat").SetMaterial("tblack")
	xp.cfg = scene.AddPart()
	xp.cfg.SetFacade("icon", "uv").SetMaterial("green")
	xp.cfg.SetTexture("xpblue", 0)

	// the cloak bar text.
	xp.ck = scene.AddPart()
	xp.ck.SetBanner("0", "uv", "weblySleek16", "weblySleek16White")
	xp.resize(screenWidth, screenHeight)
	return xp
}

// resize adjusts the graphics to fit the new window dimensions.
func (xp *xpbar) resize(screenWidth, screenHeight int) {
	xp.setSize(screenWidth, screenHeight)
	xp.bg.SetLocation(xp.cx, xp.cy, 1)
	xp.bg.SetScale(float64(xp.bw/2), float64(xp.bh-xp.y), 1)

	// adjust the teleport energy bar.
	xp.tbg.SetLocation(xp.cx-float64(xp.w)/10-2, xp.cy+30, 1)
	xp.tbg.SetScale(float64(xp.bw/10), float64(xp.bh-xp.y)-5, 1)
	bw := xp.tk.BannerWidth()
	xp.tk.SetLocation(xp.cx-float64(xp.bw)/10-2-float64(bw/2), xp.cy+21, 0)

	// adjust the cloaking energy bar.
	xp.cbg.SetLocation(xp.cx+float64(xp.bw)/10+2, xp.cy+30, 1)
	xp.cbg.SetScale(float64(xp.bw/10), float64(xp.bh-xp.y)-5, 1)
	bw = xp.ck.BannerWidth()
	xp.ck.SetLocation(xp.cx+float64(xp.bw)/10+2-float64(bw/2), xp.cy+21, 0)

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
	xp.hb.UpdateBanner(coreCount)
	bw := xp.hb.BannerWidth()
	xp.hb.SetLocation(xp.cx-float64(bw/2), xp.cy*0.5-5, 0)

	// turn on the warning colour if player has less than the starting amount of cores.
	barMax := float64(xp.bw/2 - xp.linew)
	if health >= warn {
		xp.fg.SetTexture("xpcyan", 0)
	} else {
		xp.fg.SetTexture("xpred", 0)
	}
	healthBar := float64(health) / float64(high) * barMax
	zeroSpot := float64(xp.border) + healthBar + float64(xp.linew-xp.border)
	xp.fg.SetLocation(zeroSpot, xp.cy, 0)
	xp.fg.SetScale(healthBar, float64(xp.bh-xp.y-xp.linew)-1, 1)
}

// energyMonitor:energyUpdated. Update the energy banner when it changes.
func (xp *xpbar) energyUpdated(teleportEnergy, tmax, cloakEnergy, cmax int) {
	tratio := float64(teleportEnergy) / float64(tmax)
	if tratio == 1.0 {
		xp.tfg.SetTexture("xpblue", 0)
	} else {
		xp.tfg.SetTexture("xpred", 0)
	}
	xp.tfg.SetLocation(xp.cx-float64(xp.w)/10-2, xp.cy+30, 0)
	xp.tfg.SetScale((float64(xp.bw/10)-2)*tratio, float64(xp.bh-xp.y)-7, 1)
	cratio := float64(cloakEnergy) / float64(cmax)
	xp.cfg.SetLocation(xp.cx+float64(xp.w)/10+2, xp.cy+30, 0)
	xp.cfg.SetScale((float64(xp.bw/10)-2)*cratio, float64(xp.bh-xp.y)-7, 1)
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
func (xp *xpbar) updateKeys(teleportKey, cloakKey string) {
	if xp.tk != nil && xp.ck != nil {
		xp.tk.UpdateBanner(teleportKey)
		xp.ck.UpdateBanner(cloakKey)
	}
}

// xpbar
// ===========================================================================
// minimap

// minimap displays a limited portion of the current level from the overhead
// 2D perspective.
type minimap struct {
	area             // Rectangular area.
	scene  vu.Scene  // Xztoxy scene.
	eng    vu.Engine // Engine.
	cores  []vu.Part // Keep track of the cores for removal.
	part   vu.Part   // Used to transform all the minimap models.
	bg     vu.Part   // The white background.
	scale  float64   // Minimap sizing.
	ppm    vu.Part   // Player position marker.
	cpm    vu.Part   // Center of map position marker.
	spms   []vu.Part // Sentry position markers.
	radius int       // How much of the map is displayed from the center.
}

// newMinimap initializes the minimap. It still needs to be populated.
func newMinimap(eng vu.Engine, numTroops int) *minimap {
	mm := &minimap{}
	mm.eng = eng
	mm.radius = 120
	mm.scale = float64(5)
	mm.cores = []vu.Part{}
	mm.scene = eng.AddScene(vu.XZ_XY)
	mm.scene.SetVisibleRadius(float64(mm.radius))
	mm.scene.Set2D()
	mm.setSize(eng.Size())

	// create the parent for all the visible minimap pieces.
	mm.part = mm.scene.AddPart()
	mm.part.SetCullable(false)
	mm.part.SetLocation(float64(mm.x), 0, float64(-mm.y))

	// add the white background.
	mm.bg = mm.part.AddPart()
	mm.bg.SetFacade("icon_xz", "uvm").SetMaterial("green")
	mm.bg.SetTexture("hudbg", 0)
	mm.bg.SetScale(110, 1, 110)

	// create the sentinel position markers
	mm.spms = []vu.Part{}
	for cnt := 0; cnt < numTroops; cnt++ {
		tpm := mm.part.AddPart()
		tpm.SetFacade("square_xz", "flat").SetMaterial("tred")
		tpm.SetScale(mm.scale, mm.scale, mm.scale)
		mm.spms = append(mm.spms, tpm)
	}

	// create the player marker and center map marker.
	mm.cpm = mm.part.AddPart()
	mm.cpm.SetFacade("square_xz", "flat").SetMaterial("blue")
	mm.cpm.SetScale(mm.scale, mm.scale, mm.scale)
	mm.ppm = mm.part.AddPart()
	mm.ppm.SetFacade("tri_xz", "flat").SetMaterial("tblack")
	mm.ppm.SetScale(mm.scale, mm.scale, mm.scale)
	return mm
}

// setVisible (un)hides all the minimap objects.
func (mm *minimap) setVisible(isVisible bool) {
	mm.scene.SetVisible(isVisible)
}

// resize is responsible for keeping the minimap at the bottom
// right corner of the application window.
func (mm *minimap) resize(width, height int) {
	mm.setSize(0, 0, width, height)
	mm.part.SetLocation(float64(mm.x), 0, float64(-mm.y))
}

// setSize adjusts the scene perspective to the given window size.
// Generally this is 1 pixel to 1 unit for HUD type scenes.
func (mm *minimap) setSize(x, y, width, height int) {
	mm.x, mm.y, mm.w, mm.h = width-mm.radius-10, 120, width, height
	mm.scene.SetOrthographic(0, float64(mm.w), 0, float64(mm.h), 0, 10)
}

// setLevel is called when a level transition happens.
func (mm *minimap) setLevel(sc vu.Scene, lvl *level) {
	x, y, z := sc.ViewLocation()
	mm.scene.SetViewLocation(x*mm.scale, y*mm.scale, z*mm.scale)

	// adjust the center location based on the game maze center.
	mm.cx, mm.cy = float64(lvl.gcx*lvl.units)*mm.scale, float64(-lvl.gcy*lvl.units)*mm.scale
	mm.ppm.SetLocation(x, y, z)
	mm.bg.SetLocation(x, y, z)
	mm.ppm.SetRotation(sc.ViewRotation())
	mm.setSentryLocations(lvl.sentries)
	lvl.player.monitorHealth("mmap", mm)
}

// addWall adds a block representing a wall to the minimap.
func (mm *minimap) addWall(x, y float64) {
	wall := mm.part.AddPart()
	wall.SetFacade("square_xz", "flat").SetMaterial("gray")
	wall.SetLocation(x*mm.scale, 0, y*mm.scale)
	wall.SetScale(mm.scale, mm.scale, mm.scale)
}

// addCore adds a small block representing an energy core to the minimap.
func (mm *minimap) addCore(x, y float64) {
	scale := mm.scale
	cm := mm.part.AddPart()
	cm.SetFacade("square_xz", "flat").SetMaterial("green")
	cm.SetLocation(x*scale, 0, y*scale)
	scale *= 0.5
	cm.SetScale(scale, scale, scale)
	mm.cores = append(mm.cores, cm)
}

// remCore removes the energy core from the minimap.
func (mm *minimap) remCore(gamex, gamez float64) {
	scale := mm.scale
	gx, gz := lin.Round(gamex, 0)*scale, lin.Round(gamez, 0)*scale
	for index, core := range mm.cores {
		cx, _, cz := core.Location()
		cx, cz = lin.Round(cx, 0), lin.Round(cz, 0)
		if cx == gx && cz == gz {
			mm.part.RemPart(core)
			mm.cores = append(mm.cores[:index], mm.cores[index+1:]...)
			return
		}
	}
	log.Printf("hud.mapOverlay.remCore: failed to remove a core.")
}

// resetCores is expected to be called when switching levels so that this level
// is clear of cores the next time it is activated.
func (mm *minimap) resetCores() {
	for _, core := range mm.cores {
		mm.part.RemPart(core)
	}
	mm.cores = []vu.Part{}
}

// healthMonitor:healthUpdated. Update the center colour of the maze
// based on the player health.
func (mm *minimap) healthUpdated(health, warn, high int) {
	if health == high {
		mm.cpm.SetMaterial("green")
	} else {
		mm.cpm.SetMaterial("blue")
	}
}

// update adjusts the minimap according to the players new position.
func (mm *minimap) update(sc vu.Scene, sentries []*sentinel) {
	scale := mm.scale
	x, y, z := sc.ViewLocation()
	x, y, z = x*scale, y*scale, z*scale
	mm.scene.SetViewLocation(x, y, z)
	mm.setPlayerLocation(x, y, z)
	mm.setPlayerRotation(sc.ViewRotation())
	mm.setCenterLocation(x, y, z)
	mm.setSentryLocations(sentries)
}

// set the position of the player marker by mirroring the game camera.
func (mm *minimap) setPlayerLocation(x, y, z float64) {
	mm.ppm.SetLocation(x, y, z)
	mm.bg.SetLocation(x, y, z)
}
func (mm *minimap) setPlayerRotation(x, y, z, w float64) { mm.ppm.SetRotation(x, y, z, w) }

// set the position of the maze center marker.
func (mm *minimap) setCenterLocation(x, y, z float64) {
	toc := &lin.V3{x - mm.cx, y, z - mm.cy} // vector from player to center
	dtoc := toc.Len()                       // distance to center
	mm.cpm.SetLocation(mm.cx, 0, mm.cy)     // set marker at center...
	if dtoc > float64(mm.radius) {          // ... unless the distance is to great
		toc.Unit().Scale(toc, float64(mm.radius))
		mm.cpm.SetLocation(x-toc.X, y, z-toc.Z)
	}
}

// set the position for all the sentry markers.
func (mm *minimap) setSentryLocations(sentinels []*sentinel) {
	if len(mm.spms) == len(sentinels) {
		for cnt, sentry := range sentinels {
			tpm := mm.spms[cnt]
			x, y, z := sentry.location()
			x, y, z = x*mm.scale, y*mm.scale, z*mm.scale
			tpm.SetLocation(x, y, z)
		}
	} else {
		log.Printf("hud.minimap.setSentryLocations: sentry length mismatch")
	}
}
