//go:build windows

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	appassets "degu-desktop/assets"
	"github.com/lxn/win"
)

const (
	appName       = "Degu Desktop"
	windowClass   = "DeguDesktopPetWindow"
	wmTray        = win.WM_APP + 1
	timerID       = 42
	timerInterval = 55
	frameW        = 96
	frameH        = 64
	frameCount    = 32
	motionSets    = 10
	scale         = 1
	spriteW       = frameW * scale
	spriteH       = frameH * scale
	sceneH        = 92
	wheelSize     = 72
	maxPetCount   = 5
	maxForage     = 5
	wheelKeyHold  = 18
)

const (
	idleStart    = 0
	idleFrames   = 4
	walkStart    = 4
	walkFrames   = 8
	scurryStart  = 12
	scurryFrames = 8
	nibbleStart  = 20
	nibbleFrames = 6
	hopStart     = 26
	hopFrames    = 6
)

var (
	idleFrameSeq   = []int{idleStart, idleStart + 1, idleStart + 3, idleStart + 1}
	walkFrameSeq   = []int{walkStart, walkStart + 1, walkStart + 3, walkStart + 1}
	nibbleFrameSeq = []int{nibbleStart, nibbleStart + 1, nibbleStart + 2, nibbleStart + 1}
	hopFrameSeq    = []int{hopStart, hopStart + 1, hopStart + 2, hopStart + 3}
)

const (
	menuExit         uint16 = 100
	menuModeKeyboard uint16 = 101
	menuModeRandom   uint16 = 102
	menuSpeedSlow    uint16 = 110
	menuSpeedNormal  uint16 = 111
	menuSpeedFast    uint16 = 112
	menuCount1       uint16 = 120
	menuCount2       uint16 = 121
	menuCount3       uint16 = 122
	menuCount5       uint16 = 123
	menuWheelToggle  uint16 = 130
	menuVariantBase  uint16 = 200
)

type behaviorMode int

const (
	modeKeyboard behaviorMode = iota
	modeRandom
)

type behaviorState int

const (
	stateIdle behaviorState = iota
	stateWalk
	stateScurry
	stateNibble
	stateHop
	stateWheel
	stateGroom
	stateForage
	stateCarry
)

const (
	reservedItem = -2
	noItem       = -1
)

type coatVariant struct {
	ID    string
	Label string
}

var variants = []coatVariant{
	{ID: "wild_agouti", Label: "Wild agouti"},
	{ID: "black", Label: "Black"},
	{ID: "blue", Label: "Blue (slate gray)"},
	{ID: "gray", Label: "Gray"},
	{ID: "white_cream", Label: "White / cream"},
	{ID: "sand_champagne", Label: "Sand / champagne"},
	{ID: "chocolate", Label: "Chocolate"},
	{ID: "black_pied", Label: "Black pied"},
	{ID: "agouti_pied", Label: "Agouti pied"},
	{ID: "blue_pied", Label: "Blue pied (slate gray)"},
	{ID: "cream_pied", Label: "Cream pied"},
}

type deguPet struct {
	motionSet  int
	frame      int
	x          int
	laneOffset int
	item       int
	carryKind  int
	state      behaviorState
	stateTicks int
	moveSpeed  int
}

type forageItem struct {
	x      int
	kind   int
	owner  int
	active bool
}

type petApp struct {
	hwnd         win.HWND
	hinst        win.HINSTANCE
	trayIcon     win.HICON
	keyHook      uintptr
	frames       map[string][][]*image.RGBA
	wheel        *image.RGBA
	pets         []deguPet
	forage       []forageItem
	variant      int
	speed        int
	mode         behaviorMode
	petCount     int
	wheelEnabled bool
	wheelX       int
	sceneW       int
	tickCount    int
	closing      atomic.Bool
}

var app *petApp

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	procAppendMenuW        = user32.NewProc("AppendMenuW")
	procSetWindowsHookExW  = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHook  = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookExProc = user32.NewProc("CallNextHookEx")
	procUpdateLayeredWin   = user32.NewProc("UpdateLayeredWindow")
)

const (
	acSrcOver      = 0
	ulwAlpha       = 0x00000002
	spiGetWorkArea = 0x0030
)

func main() {
	runtime.LockOSThread()
	rand.Seed(time.Now().UnixNano())

	hinst := win.GetModuleHandle(nil)
	app = &petApp{
		hinst:        hinst,
		frames:       loadSprites(),
		wheel:        loadWheelSprite(),
		variant:      0,
		speed:        3,
		mode:         modeRandom,
		petCount:     2,
		wheelEnabled: true,
	}

	className := syscall.StringToUTF16Ptr(windowClass)
	wc := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hinst,
		HIcon:         win.LoadIcon(0, win.MAKEINTRESOURCE(win.IDI_APPLICATION)),
		HCursor:       win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW)),
		HbrBackground: 0,
		LpszClassName: className,
	}
	if win.RegisterClassEx(&wc) == 0 {
		panic(fmt.Sprintf("RegisterClassEx failed: %v", syscall.GetLastError()))
	}

	app.hwnd = win.CreateWindowEx(
		win.WS_EX_LAYERED|win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|win.WS_EX_TRANSPARENT,
		className,
		syscall.StringToUTF16Ptr(appName),
		win.WS_POPUP,
		0, 0, 1, 1,
		0, 0, hinst, nil,
	)
	if app.hwnd == 0 {
		panic(fmt.Sprintf("CreateWindowEx failed: %v", syscall.GetLastError()))
	}

	app.resetPosition()
	app.installTray()
	app.installKeyboardHook()
	win.ShowWindow(app.hwnd, win.SW_SHOWNOACTIVATE)
	win.SetTimer(app.hwnd, timerID, timerInterval, 0)
	app.render()

	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) > 0 {
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
	app.cleanup()
}

func wndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_TIMER:
		if wParam == timerID {
			app.tick()
			app.render()
			return 0
		}
	case wmTray:
		if lParam == win.WM_RBUTTONUP || lParam == win.WM_LBUTTONUP {
			app.showTrayMenu()
			return 0
		}
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (a *petApp) resetPosition() {
	work := workArea()
	a.syncScene(work)
	a.setPetCount(a.petCount)
}

func (a *petApp) tick() {
	if a.closing.Load() {
		return
	}
	work := workArea()
	a.syncScene(work)
	a.ensureForageItems()
	for i := range a.pets {
		a.tickPet(i, &a.pets[i])
	}
	a.syncNearbyWalkers()
	a.maybeStartSocial()
	a.tickCount++
}

func (a *petApp) tickPet(index int, p *deguPet) {
	if p.stateTicks <= 0 {
		switch p.state {
		case stateWheel:
			a.leaveWheel(p)
		case stateNibble:
			if p.item >= 0 && p.item < len(a.forage) {
				a.finishGnawing(index, p)
			} else if a.mode == modeRandom {
				a.chooseRandomAction(p)
			} else {
				p.state = stateIdle
				p.moveSpeed = 0
				p.stateTicks = 12
			}
		case stateGroom, stateCarry:
			a.releaseForage(index, p)
			if a.mode == modeRandom {
				a.chooseRandomAction(p)
			} else {
				p.state = stateIdle
				p.moveSpeed = 0
				p.stateTicks = 12
			}
		default:
			if a.mode != modeRandom {
				p.state = stateIdle
				p.moveSpeed = 0
				p.stateTicks = 12
				break
			}
			a.chooseRandomAction(p)
		}
	}

	speed := 0
	switch p.state {
	case stateWalk, stateScurry, stateHop, stateForage, stateCarry:
		speed = p.moveSpeed
	case stateWheel:
		p.x = clamp(a.wheelX-wheelSize/2, 0, max(0, a.sceneW-spriteW))
	}

	if speed > 0 {
		p.x += speed
		if p.state == stateForage {
			a.maybeStartGnawing(index, p)
		}
	}

	p.stateTicks--
	if p.x > a.sceneW+8 {
		a.resetPetAtLeft(index, p)
	}
	p.frame++
}

func (a *petApp) chooseRandomAction(p *deguPet) {
	roll := rand.Intn(100)
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.item = noItem
	p.carryKind = noItem
	if roll < 18 && a.maybeAssignForageTarget(p) {
		return
	}
	switch {
	case roll < 34:
		p.state = stateIdle
		p.moveSpeed = 0
		p.stateTicks = 24 + rand.Intn(58)
		return
	case roll < 78:
		p.state = stateWalk
		p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
		p.stateTicks = 34 + rand.Intn(92)
	case roll < 92:
		p.state = stateScurry
		p.moveSpeed = a.speed + 1 + rand.Intn(2)
		p.stateTicks = 10 + rand.Intn(18)
	case roll < 98:
		p.state = stateNibble
		p.moveSpeed = 0
		p.stateTicks = 26 + rand.Intn(32)
	default:
		p.state = stateHop
		p.moveSpeed = max(1, a.speed-1)
		p.stateTicks = 14 + rand.Intn(16)
	}
}

func (a *petApp) render() {
	work := workArea()
	a.syncScene(work)
	canvas := image.NewRGBA(image.Rect(0, 0, a.sceneW, sceneH))
	draw.Draw(canvas, canvas.Bounds(), image.Transparent, image.Point{}, draw.Src)

	wheelActive := a.wheelEnabled && a.hasWheelRunner()
	wheelX := a.wheelX - wheelSize/2
	wheelY := sceneH - wheelSize - 2
	if wheelActive {
		drawWheelBack(canvas, wheelX, wheelY, a.wheel)
	}

	a.drawForageItems(canvas)

	for i := range a.pets {
		p := &a.pets[i]
		if p.state == stateWheel {
			continue
		}
		frame := currentFrame(p.state, p.frame)
		src := a.frames[variants[a.variant].ID][p.motionSet][frame]
		scaled := scaleImage(src, scale)
		y := sceneH - spriteH - p.laneOffset
		draw.Draw(canvas, image.Rect(p.x, y, p.x+spriteW, y+spriteH), scaled, image.Point{}, draw.Over)
		if p.state == stateCarry && p.carryKind != noItem {
			drawForageProp(canvas, p.x+spriteW-18, y+35, p.carryKind)
		}
	}

	if wheelActive {
		for i := range a.pets {
			p := &a.pets[i]
			if p.state != stateWheel {
				continue
			}
			frame := currentFrame(p.state, p.frame)
			src := a.frames[variants[a.variant].ID][p.motionSet][frame]
			drawWheelRunner(canvas, wheelX, wheelY, src, p.frame)
		}
		drawWheelFront(canvas, wheelX, wheelY, a.tickCount)
	}
	updateLayeredWindow(a.hwnd, canvas, int(work.Left), int(work.Bottom)-sceneH)
}

func currentFrame(state behaviorState, frame int) int {
	switch state {
	case stateIdle:
		return frameFromSeq(idleFrameSeq, frame, 5)
	case stateWalk, stateForage, stateCarry:
		return frameFromSeq(walkFrameSeq, frame, 2)
	case stateScurry, stateWheel:
		return frameFromSeq(walkFrameSeq, frame, 1)
	case stateNibble:
		return frameFromSeq(nibbleFrameSeq, frame, 3)
	case stateHop:
		return frameFromSeq(hopFrameSeq, frame, 2)
	case stateGroom:
		return frameFromSeq(nibbleFrameSeq, frame, 4)
	}
	return idleStart
}

func frameFromSeq(seq []int, frame, divisor int) int {
	if len(seq) == 0 {
		return idleStart
	}
	if divisor < 1 {
		divisor = 1
	}
	return seq[(frame/divisor)%len(seq)]
}

func (a *petApp) onTyping() {
	if a.mode != modeKeyboard {
		return
	}
	for i := range a.pets {
		p := &a.pets[i]
		if a.wheelEnabled && i == 0 && p.item == noItem {
			a.enterWheelFromTyping(p)
			continue
		}
		p.state = stateScurry
		p.frame = rand.Intn(scurryFrames)
		p.motionSet = rand.Intn(motionSets)
		p.stateTicks = 18 + rand.Intn(16)
		p.moveSpeed = a.speed + 2 + rand.Intn(2)
	}
}

func (a *petApp) syncScene(work win.RECT) {
	a.sceneW = max(1, int(work.Right-work.Left))
	nextWheelX := a.sceneW * 2 / 3
	a.wheelX = clamp(nextWheelX, wheelSize/2+24, max(wheelSize/2+24, a.sceneW-wheelSize/2-24))
}

func (a *petApp) setPetCount(count int) {
	count = clamp(count, 1, maxPetCount)
	a.petCount = count
	for len(a.pets) < count {
		a.pets = append(a.pets, a.newPet(len(a.pets)))
	}
	if len(a.pets) > count {
		a.pets = a.pets[:count]
	}
	for i := range a.forage {
		if a.forage[i].owner >= count || a.forage[i].owner == reservedItem {
			a.forage[i].owner = noItem
			a.forage[i].active = false
		}
	}
	for i := range a.pets {
		a.pets[i].laneOffset = (i % 3) * 5
	}
}

func (a *petApp) newPet(index int) deguPet {
	spread := max(spriteW+24, a.sceneW/max(1, a.petCount+1))
	p := deguPet{
		x:          -spriteW - index*spread - rand.Intn(80),
		laneOffset: (index % 3) * 5,
		item:       noItem,
		carryKind:  noItem,
		motionSet:  rand.Intn(motionSets),
		state:      stateWalk,
		moveSpeed:  max(1, a.speed-1+rand.Intn(2)),
		stateTicks: 30 + rand.Intn(80),
	}
	if index == 0 {
		p.x = rand.Intn(max(1, a.sceneW-spriteW))
	}
	a.chooseRandomAction(&p)
	return p
}

func (a *petApp) resetPetAtLeft(index int, p *deguPet) {
	a.releaseForage(index, p)
	p.x = -spriteW - rand.Intn(120)
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.item = noItem
	p.carryKind = noItem
	p.state = stateWalk
	p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
	p.stateTicks = 40 + rand.Intn(90)
}

func (a *petApp) ensureForageItems() {
	for len(a.forage) < maxForage {
		a.forage = append(a.forage, forageItem{owner: noItem})
	}
	if a.tickCount%90 != 0 && a.tickCount != 0 {
		return
	}
	for i := range a.forage {
		if a.forage[i].active || a.forage[i].owner != noItem {
			continue
		}
		if rand.Intn(100) > 45 {
			continue
		}
		x := 28 + rand.Intn(max(1, a.sceneW-56))
		if abs(x-a.wheelX) < wheelSize {
			x = clamp(x+wheelSize+24, 24, max(24, a.sceneW-24))
		}
		a.forage[i] = forageItem{
			x:      x,
			kind:   rand.Intn(3),
			owner:  noItem,
			active: true,
		}
	}
}

func (a *petApp) maybeAssignForageTarget(p *deguPet) bool {
	if p.item != noItem || p.state == stateWheel {
		return false
	}
	best := noItem
	bestDistance := a.sceneW + spriteW
	for i, item := range a.forage {
		if !item.active || item.owner != noItem {
			continue
		}
		distance := item.x - (p.x + spriteW - 22)
		if distance < 12 || distance > bestDistance {
			continue
		}
		best = i
		bestDistance = distance
	}
	if best == noItem {
		return false
	}
	a.forage[best].owner = reservedItem
	p.item = best
	p.carryKind = noItem
	p.state = stateForage
	p.moveSpeed = max(1, a.speed-1)
	p.stateTicks = max(45, bestDistance/max(1, p.moveSpeed)+36)
	return true
}

func (a *petApp) maybeStartGnawing(index int, p *deguPet) {
	if p.item < 0 || p.item >= len(a.forage) {
		a.releaseForage(index, p)
		a.chooseRandomAction(p)
		return
	}
	item := &a.forage[p.item]
	item.owner = index
	if !item.active {
		a.releaseForage(index, p)
		a.chooseRandomAction(p)
		return
	}
	mouthX := p.x + spriteW - 22
	if mouthX < item.x {
		return
	}
	p.x = clamp(item.x-spriteW+22, 0, max(0, a.sceneW-spriteW))
	p.state = stateNibble
	p.frame = 0
	p.moveSpeed = 0
	p.stateTicks = 28 + rand.Intn(34)
}

func (a *petApp) finishGnawing(index int, p *deguPet) {
	item := &a.forage[p.item]
	kind := item.kind
	item.active = false
	item.owner = index
	if rand.Intn(100) < 58 {
		p.state = stateCarry
		p.frame = 0
		p.carryKind = kind
		p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
		p.stateTicks = 26 + rand.Intn(44)
		return
	}
	a.releaseForage(index, p)
	a.chooseRandomAction(p)
}

func (a *petApp) releaseForage(index int, p *deguPet) {
	if p.item >= 0 && p.item < len(a.forage) && (a.forage[p.item].owner == index || a.forage[p.item].owner == reservedItem) {
		a.forage[p.item].owner = noItem
		a.forage[p.item].active = false
	}
	p.item = noItem
	p.carryKind = noItem
}

func (a *petApp) syncNearbyWalkers() {
	for i := 0; i < len(a.pets); i++ {
		for j := i + 1; j < len(a.pets); j++ {
			pi := &a.pets[i]
			pj := &a.pets[j]
			if pi.state != stateWalk || pj.state != stateWalk {
				continue
			}
			if abs(pi.x-pj.x) > 72 {
				continue
			}
			speed := max(1, min(pi.moveSpeed, pj.moveSpeed))
			pi.moveSpeed = speed
			pj.moveSpeed = speed
		}
	}
}

func (a *petApp) maybeStartSocial() {
	if len(a.pets) < 2 || a.tickCount%24 != 0 || rand.Intn(100) > 28 {
		return
	}
	for i := 0; i < len(a.pets); i++ {
		for j := i + 1; j < len(a.pets); j++ {
			pi := &a.pets[i]
			pj := &a.pets[j]
			if !canSocialize(pi) || !canSocialize(pj) {
				continue
			}
			if abs(pi.x-pj.x) > 84 {
				continue
			}
			anchor := min(pi.x, pj.x)
			pi.x = clamp(anchor, 0, max(0, a.sceneW-spriteW-36))
			pj.x = clamp(pi.x+34+rand.Intn(14), 0, max(0, a.sceneW-spriteW))
			pj.laneOffset = pi.laneOffset
			ticks := 44 + rand.Intn(70)
			pi.state = stateGroom
			pj.state = stateGroom
			pi.moveSpeed = 0
			pj.moveSpeed = 0
			pi.frame = 0
			pj.frame = 3
			pi.stateTicks = ticks
			pj.stateTicks = ticks + rand.Intn(16)
			return
		}
	}
}

func canSocialize(p *deguPet) bool {
	return p.item == noItem && (p.state == stateIdle || p.state == stateWalk || p.state == stateNibble)
}

func (a *petApp) hasWheelRunner() bool {
	for i := range a.pets {
		if a.pets[i].state == stateWheel {
			return true
		}
	}
	return false
}

func (a *petApp) drawForageItems(dst *image.RGBA) {
	y := sceneH - 9
	for _, item := range a.forage {
		if !item.active {
			continue
		}
		drawForageProp(dst, item.x, y, item.kind)
	}
}

func drawForageProp(dst *image.RGBA, x, y, kind int) {
	switch kind {
	case 0:
		hay := rgba(155, 177, 91, 235)
		shadow := rgba(83, 101, 55, 180)
		drawPixelLine(dst, x-8, y+2, x+8, y-3, shadow)
		drawPixelLine(dst, x-6, y, x+10, y-4, hay)
		drawPixelLine(dst, x-4, y+3, x+7, y-2, hay)
	case 1:
		twig := rgba(111, 78, 47, 240)
		tip := rgba(164, 119, 72, 230)
		drawPixelLine(dst, x-9, y+2, x+9, y-2, twig)
		drawPixelLine(dst, x+1, y-1, x+7, y-7, tip)
	case 2:
		fillCircle(dst, x, y-2, 4, rgba(184, 148, 84, 240))
	default:
		fillCircle(dst, x, y-2, 3, rgba(170, 150, 94, 220))
	}
}

func (a *petApp) enterWheelFromTyping(p *deguPet) {
	alreadyRunning := p.state == stateWheel
	p.state = stateWheel
	if !alreadyRunning {
		p.frame = 0
		p.motionSet = rand.Intn(motionSets)
	}
	p.item = noItem
	p.carryKind = noItem
	p.moveSpeed = 0
	p.stateTicks = wheelKeyHold
	p.x = clamp(a.wheelX-wheelSize/2, 0, max(0, a.sceneW-spriteW))
}

func (a *petApp) leaveWheel(p *deguPet) {
	p.state = stateScurry
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.moveSpeed = a.speed + 1 + rand.Intn(2)
	p.stateTicks = 16 + rand.Intn(20)
	p.x = clamp(a.wheelX+wheelSize/2-20, 0, max(0, a.sceneW-spriteW))
}

func loadSprites() map[string][][]*image.RGBA {
	out := make(map[string][][]*image.RGBA)
	for _, v := range variants {
		sets := make([][]*image.RGBA, 0, motionSets)
		for set := 0; set < motionSets; set++ {
			name := fmt.Sprintf("sprites/degu_%s_set%02d.png", v.ID, set)
			data, err := fs.ReadFile(appassets.FS, name)
			if err != nil {
				panic(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				panic(err)
			}
			if img.Bounds().Dx() != frameW*frameCount || img.Bounds().Dy() != frameH {
				panic(fmt.Sprintf("%s must be %dx%d; run cmd/importsheet", name, frameW*frameCount, frameH))
			}
			frames := make([]*image.RGBA, 0, frameCount)
			for i := 0; i < frameCount; i++ {
				r := image.Rect(i*frameW, 0, (i+1)*frameW, frameH)
				frame := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
				draw.Draw(frame, frame.Bounds(), img, r.Min, draw.Src)
				frames = append(frames, frame)
			}
			sets = append(sets, frames)
		}
		out[v.ID] = sets
	}
	return out
}

func loadWheelSprite() *image.RGBA {
	data, err := fs.ReadFile(appassets.FS, "sprites/wheel.png")
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, wheelSize, wheelSize))
	if img.Bounds().Dx() == wheelSize && img.Bounds().Dy() == wheelSize {
		draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)
		return dst
	}
	src := image.NewRGBA(img.Bounds())
	draw.Draw(src, src.Bounds(), img, img.Bounds().Min, draw.Src)
	scaled := scaleImageTo(src, wheelSize, wheelSize)
	draw.Draw(dst, dst.Bounds(), scaled, image.Point{}, draw.Src)
	return dst
}

func (a *petApp) installTray() {
	iconPath := filepath.Join(os.TempDir(), "degu-desktop-tray.ico")
	if data, err := fs.ReadFile(appassets.FS, "tray.ico"); err == nil {
		_ = os.WriteFile(iconPath, data, 0o644)
	}
	a.trayIcon = win.HICON(win.LoadImage(0, syscall.StringToUTF16Ptr(iconPath), win.IMAGE_ICON, 0, 0, win.LR_LOADFROMFILE|win.LR_DEFAULTSIZE))
	if a.trayIcon == 0 {
		a.trayIcon = win.LoadIcon(0, win.MAKEINTRESOURCE(win.IDI_APPLICATION))
	}
	var nid win.NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	nid.UFlags = win.NIF_MESSAGE | win.NIF_ICON | win.NIF_TIP
	nid.UCallbackMessage = wmTray
	nid.HIcon = a.trayIcon
	copy(nid.SzTip[:], syscall.StringToUTF16(appName))
	win.Shell_NotifyIcon(win.NIM_ADD, &nid)
}

func (a *petApp) showTrayMenu() {
	menu := win.CreatePopupMenu()
	coatMenu := win.CreatePopupMenu()
	for i, v := range variants {
		flags := uint32(win.MF_STRING)
		if i == a.variant {
			flags |= win.MF_CHECKED
		}
		appendMenu(coatMenu, flags, uintptr(menuVariantBase+uint16(i)), syscall.StringToUTF16Ptr(v.Label))
	}
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(coatMenu), syscall.StringToUTF16Ptr("Coat"))
	appendMenu(menu, win.MF_SEPARATOR, 0, nil)

	speedMenu := win.CreatePopupMenu()
	appendChecked(speedMenu, menuSpeedSlow, "Slow", a.speed == 2)
	appendChecked(speedMenu, menuSpeedNormal, "Normal", a.speed == 3)
	appendChecked(speedMenu, menuSpeedFast, "Fast", a.speed == 5)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(speedMenu), syscall.StringToUTF16Ptr("Speed"))

	modeMenu := win.CreatePopupMenu()
	appendChecked(modeMenu, menuModeKeyboard, "Keyboard reaction", a.mode == modeKeyboard)
	appendChecked(modeMenu, menuModeRandom, "Random stroll", a.mode == modeRandom)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(modeMenu), syscall.StringToUTF16Ptr("Mode"))

	countMenu := win.CreatePopupMenu()
	appendChecked(countMenu, menuCount1, "1 degu", a.petCount == 1)
	appendChecked(countMenu, menuCount2, "2 degus", a.petCount == 2)
	appendChecked(countMenu, menuCount3, "3 degus", a.petCount == 3)
	appendChecked(countMenu, menuCount5, "5 degus", a.petCount == 5)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(countMenu), syscall.StringToUTF16Ptr("Degu count"))

	appendChecked(menu, menuWheelToggle, "Typing wheel", a.wheelEnabled)
	appendMenu(menu, win.MF_SEPARATOR, 0, nil)
	appendMenu(menu, win.MF_STRING, uintptr(menuExit), syscall.StringToUTF16Ptr("Exit"))

	var pt win.POINT
	win.GetCursorPos(&pt)
	win.SetForegroundWindow(a.hwnd)
	cmd := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_RIGHTBUTTON, pt.X, pt.Y, 0, a.hwnd, nil)
	win.DestroyMenu(menu)
	if cmd == 0 {
		return
	}
	a.handleMenu(uint16(cmd))
}

func appendChecked(menu win.HMENU, id uint16, label string, checked bool) {
	flags := uint32(win.MF_STRING)
	if checked {
		flags |= win.MF_CHECKED
	}
	appendMenu(menu, flags, uintptr(id), syscall.StringToUTF16Ptr(label))
}

func (a *petApp) handleMenu(id uint16) {
	switch {
	case id == menuExit:
		a.closing.Store(true)
		win.DestroyWindow(a.hwnd)
	case id == menuModeKeyboard:
		a.mode = modeKeyboard
		for i := range a.pets {
			a.pets[i].state = stateIdle
			a.pets[i].stateTicks = 1
			a.pets[i].moveSpeed = 0
		}
	case id == menuModeRandom:
		a.mode = modeRandom
		for i := range a.pets {
			a.chooseRandomAction(&a.pets[i])
		}
	case id == menuSpeedSlow:
		a.speed = 2
	case id == menuSpeedNormal:
		a.speed = 3
	case id == menuSpeedFast:
		a.speed = 5
	case id == menuCount1:
		a.setPetCount(1)
	case id == menuCount2:
		a.setPetCount(2)
	case id == menuCount3:
		a.setPetCount(3)
	case id == menuCount5:
		a.setPetCount(5)
	case id == menuWheelToggle:
		a.wheelEnabled = !a.wheelEnabled
		for i := range a.pets {
			if a.pets[i].state == stateWheel {
				a.leaveWheel(&a.pets[i])
			}
		}
	case id >= menuVariantBase && int(id-menuVariantBase) < len(variants):
		a.variant = int(id - menuVariantBase)
		a.render()
	}
}

func (a *petApp) cleanup() {
	win.KillTimer(a.hwnd, timerID)
	if a.keyHook != 0 {
		unhookWindowsHookEx(a.keyHook)
	}
	var nid win.NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	win.Shell_NotifyIcon(win.NIM_DELETE, &nid)
	if a.trayIcon != 0 {
		win.DestroyIcon(a.trayIcon)
	}
}

func (a *petApp) installKeyboardHook() {
	cb := syscall.NewCallback(func(code int, wParam uintptr, lParam uintptr) uintptr {
		if code >= 0 && (wParam == win.WM_KEYDOWN || wParam == win.WM_SYSKEYDOWN) {
			a.onTyping()
		}
		return callNextHookEx(0, code, wParam, lParam)
	})
	a.keyHook = setWindowsHookEx(13, cb, 0, 0)
}

func appendMenu(menu win.HMENU, flags uint32, item uintptr, text *uint16) bool {
	var textPtr uintptr
	if text != nil {
		textPtr = uintptr(unsafe.Pointer(text))
	}
	ret, _, _ := procAppendMenuW.Call(uintptr(menu), uintptr(flags), item, textPtr)
	return ret != 0
}

func setWindowsHookEx(idHook int, callback uintptr, module win.HINSTANCE, threadID uint32) uintptr {
	ret, _, _ := procSetWindowsHookExW.Call(uintptr(idHook), callback, uintptr(module), uintptr(threadID))
	return ret
}

func unhookWindowsHookEx(hook uintptr) bool {
	ret, _, _ := procUnhookWindowsHook.Call(hook)
	return ret != 0
}

func callNextHookEx(hook uintptr, code int, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procCallNextHookExProc.Call(hook, uintptr(code), wParam, lParam)
	return ret
}

func updateLayeredWindowNative(hwnd win.HWND, dstDC win.HDC, dst *win.POINT, size *win.SIZE, srcDC win.HDC, src *win.POINT, key uint32, blend *win.BLENDFUNCTION, flags uint32) bool {
	ret, _, _ := procUpdateLayeredWin.Call(
		uintptr(hwnd),
		uintptr(dstDC),
		uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(size)),
		uintptr(srcDC),
		uintptr(unsafe.Pointer(src)),
		uintptr(key),
		uintptr(unsafe.Pointer(blend)),
		uintptr(flags),
	)
	return ret != 0
}

func updateLayeredWindow(hwnd win.HWND, img *image.RGBA, x, y int) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	screenDC := win.GetDC(0)
	memDC := win.CreateCompatibleDC(screenDC)
	defer win.ReleaseDC(0, screenDC)
	defer win.DeleteDC(memDC)

	var bmi win.BITMAPINFOHEADER
	bmi.BiSize = uint32(unsafe.Sizeof(bmi))
	bmi.BiWidth = int32(w)
	bmi.BiHeight = -int32(h)
	bmi.BiPlanes = 1
	bmi.BiBitCount = 32
	bmi.BiCompression = win.BI_RGB

	var bits unsafe.Pointer
	bitmap := win.CreateDIBSection(memDC, &bmi, win.DIB_RGB_COLORS, &bits, 0, 0)
	if bitmap == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(bitmap))
	dst := unsafe.Slice((*byte)(bits), w*h*4)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			c := img.RGBAAt(px, py)
			i := (py*w + px) * 4
			a := uint16(c.A)
			dst[i+0] = byte(uint16(c.B) * a / 255)
			dst[i+1] = byte(uint16(c.G) * a / 255)
			dst[i+2] = byte(uint16(c.R) * a / 255)
			dst[i+3] = c.A
		}
	}
	old := win.SelectObject(memDC, win.HGDIOBJ(bitmap))
	defer win.SelectObject(memDC, old)

	ptDst := win.POINT{X: int32(x), Y: int32(y)}
	size := win.SIZE{CX: int32(w), CY: int32(h)}
	ptSrc := win.POINT{X: 0, Y: 0}
	blend := win.BLENDFUNCTION{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: win.AC_SRC_ALPHA}
	updateLayeredWindowNative(hwnd, screenDC, &ptDst, &size, memDC, &ptSrc, 0, &blend, ulwAlpha)
}

func workArea() win.RECT {
	var rect win.RECT
	if !win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&rect), 0) {
		rect = win.RECT{Left: 0, Top: 0, Right: 1280, Bottom: 720}
	}
	return rect
}

func scaleImage(src *image.RGBA, factor int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx()*factor, src.Bounds().Dy()*factor))
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			c := src.RGBAAt(x, y)
			for sy := 0; sy < factor; sy++ {
				for sx := 0; sx < factor; sx++ {
					dst.SetRGBA(x*factor+sx, y*factor+sy, c)
				}
			}
		}
	}
	return dst
}

func scaleImageTo(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sw := max(1, src.Bounds().Dx())
	sh := max(1, src.Bounds().Dy())
	for y := 0; y < h; y++ {
		sy := src.Bounds().Min.Y + y*sh/h
		for x := 0; x < w; x++ {
			sx := src.Bounds().Min.X + x*sw/w
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func fitVisibleImageTo(src *image.RGBA, w, h int) *image.RGBA {
	content := alphaBounds(src)
	if content.Empty() {
		return scaleImageTo(src, w, h)
	}
	cropped := image.NewRGBA(image.Rect(0, 0, content.Dx(), content.Dy()))
	draw.Draw(cropped, cropped.Bounds(), src, content.Min, draw.Src)
	scale := math.Min(float64(w)/float64(content.Dx()), float64(h)/float64(content.Dy()))
	outW := max(1, int(math.Round(float64(content.Dx())*scale)))
	outH := max(1, int(math.Round(float64(content.Dy())*scale)))
	scaled := scaleImageTo(cropped, outW, outH)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	offX := (w - outW) / 2
	offY := h - outH
	draw.Draw(dst, image.Rect(offX, offY, offX+outW, offY+outH), scaled, image.Point{}, draw.Over)
	return dst
}

func alphaBounds(img *image.RGBA) image.Rectangle {
	b := img.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A == 0 {
				continue
			}
			found = true
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+1 > maxX {
				maxX = x + 1
			}
			if y+1 > maxY {
				maxY = y + 1
			}
		}
	}
	if !found {
		return image.Rect(0, 0, 0, 0)
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func drawWheelBack(dst *image.RGBA, x, y int, wheel *image.RGBA) {
	cx := x + wheelSize/2
	cy := y + wheelSize/2
	outer := float64(wheelSize/2 - 2)
	inner := outer - 5
	rim := rgba(92, 86, 76, 210)
	shadow := rgba(44, 41, 38, 120)
	base := rgba(74, 67, 58, 210)

	if wheel != nil {
		draw.Draw(dst, image.Rect(x, y, x+wheelSize, y+wheelSize), wheel, image.Point{}, draw.Over)
		return
	}
	for py := y; py < y+wheelSize; py++ {
		for px := x; px < x+wheelSize; px++ {
			dx := float64(px - cx)
			dy := float64(py - cy)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= outer+1 && dist >= inner {
				if px < cx && py > cy {
					dst.SetRGBA(px, py, shadow)
				} else {
					dst.SetRGBA(px, py, rim)
				}
			}
		}
	}
	drawPixelLine(dst, cx-20, y+wheelSize-2, cx-30, sceneH-2, base)
	drawPixelLine(dst, cx+20, y+wheelSize-2, cx+30, sceneH-2, base)
	for px := cx - 38; px <= cx+38; px++ {
		for py := sceneH - 4; py <= sceneH-2; py++ {
			if image.Pt(px, py).In(dst.Bounds()) {
				dst.SetRGBA(px, py, base)
			}
		}
	}
}

func drawWheelRunner(dst *image.RGBA, x, y int, src *image.RGBA, frame int) {
	runnerW := 68
	runnerH := 46
	scaled := fitVisibleImageTo(src, runnerW, runnerH)
	bob := int(math.Sin(float64(frame)/2.0) * 2)
	dstX := x + (wheelSize-runnerW)/2
	dstY := y + wheelSize/2 - runnerH/2 + 6 + bob
	draw.Draw(dst, image.Rect(dstX, dstY, dstX+runnerW, dstY+runnerH), scaled, image.Point{}, draw.Over)
}

func drawWheelFront(dst *image.RGBA, x, y, tick int) {
	cx := x + wheelSize/2
	cy := y + wheelSize/2
	inner := float64(wheelSize/2 - 7)
	spoke := rgba(132, 123, 106, 115)
	hub := rgba(86, 78, 68, 230)
	rim := rgba(92, 86, 76, 160)

	angle := float64(tick) * 0.34
	for i := 0; i < 8; i++ {
		a := angle + float64(i)*math.Pi/4
		x2 := cx + int(math.Cos(a)*(inner-2))
		y2 := cy + int(math.Sin(a)*(inner-2))
		drawThinLine(dst, cx, cy, x2, y2, spoke)
	}

	fillCircle(dst, cx, cy, 4, hub)
	for py := y; py < y+wheelSize; py++ {
		for px := x; px < x+wheelSize; px++ {
			dx := float64(px - cx)
			dy := float64(py - cy)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= float64(wheelSize/2-5) && dist <= float64(wheelSize/2-1) {
				dst.SetRGBA(px, py, rim)
			}
		}
	}
}

func drawPixelLine(dst *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		drawBlock(dst, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawThinLine(dst *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(dst.Bounds()) {
			dst.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func fillCircle(dst *image.RGBA, cx, cy, r int, c color.RGBA) {
	for py := cy - r; py <= cy+r; py++ {
		for px := cx - r; px <= cx+r; px++ {
			dx := px - cx
			dy := py - cy
			if dx*dx+dy*dy <= r*r {
				drawBlock(dst, px, py, c)
			}
		}
	}
}

func rgba(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func drawBlock(dst *image.RGBA, x, y int, c color.RGBA) {
	for py := y - 1; py <= y+1; py++ {
		for px := x - 1; px <= x+1; px++ {
			if image.Pt(px, py).In(dst.Bounds()) {
				dst.SetRGBA(px, py, c)
			}
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
