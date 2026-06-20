//go:build windows

package main

import (
	"bytes"
	"encoding/json"
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
	appName                = "Degu Desktop"
	windowClass            = "DeguDesktopPetWindow"
	wmTray                 = win.WM_APP + 1
	wmTyping               = win.WM_APP + 2
	wmMouseClick           = win.WM_APP + 3
	timerID                = 42
	timerInterval          = 55
	frameW                 = 96
	frameH                 = 64
	frameCount             = 56
	motionSets             = 10
	scale                  = 1
	spriteW                = frameW * scale
	spriteH                = frameH * scale
	forageW                = 32
	forageH                = 24
	sceneH                 = 92
	wheelSize              = 72
	maxPetCount            = 10
	maxForage              = 5
	wheelKeyHold           = 18
	turnTicks              = 16
	reactionTicks          = 54
	settingsClientW  int32 = 760
	settingsClientH  int32 = 560
	settingsDirName        = "DeguDesktop"
	settingsFileName       = "settings.json"
)

const (
	whKeyboardLL = 13
	whMouseLL    = 14
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
	turnStart    = 32
	turnFrames   = 8
	eatStart     = 40
	eatFrames    = 4
	digStart     = 44
	digFrames    = 4
	standStart   = 48
	standFrames  = 4
	groomStart   = 52
	groomFrames  = 4
)

var (
	idleFrameSeq   = []int{idleStart, idleStart + 1, idleStart + 3, idleStart + 1}
	walkFrameSeq   = []int{walkStart, walkStart + 1, walkStart + 3, walkStart + 1}
	nibbleFrameSeq = []int{nibbleStart, nibbleStart + 1, nibbleStart + 2, nibbleStart + 1}
	hopFrameSeq    = []int{hopStart, hopStart + 1, hopStart + 2, hopStart + 3}
	turnFrameSeq   = []int{turnStart, turnStart + 1, turnStart + 2, turnStart + 3, turnStart + 4, turnStart + 5, turnStart + 6, turnStart + 7}
	eatFrameSeq    = []int{eatStart, eatStart + 1, eatStart + 2, eatStart + 3}
	digFrameSeq    = []int{digStart, digStart + 1, digStart + 2, digStart + 3}
	standFrameSeq  = []int{standStart, standStart + 1, standStart + 2, standStart + 3}
	groomFrameSeq  = []int{groomStart, groomStart + 1, groomStart + 2, groomStart + 3}
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
	menuCount10      uint16 = 124
	menuWheelToggle  uint16 = 130
	menuCoatFixed    uint16 = 131
	menuCoatSelected uint16 = 132
	menuCoatRandom   uint16 = 133
	menuSettings     uint16 = 140
	menuVariantBase  uint16 = 200
)

const (
	ctrlTabAnimals     int32 = 1000
	ctrlTabMotion      int32 = 1001
	ctrlVariantCombo   int32 = 1002
	ctrlPetMinus       int32 = 1003
	ctrlPetPlus        int32 = 1004
	ctrlLanguageCombo  int32 = 1005
	ctrlCoatFixed      int32 = 1006
	ctrlCoatSelected   int32 = 1007
	ctrlCoatRandom     int32 = 1008
	ctrlModeKeyboard   int32 = 1011
	ctrlModeRandom     int32 = 1012
	ctrlSpeedSlow      int32 = 1021
	ctrlSpeedNormal    int32 = 1022
	ctrlSpeedFast      int32 = 1023
	ctrlTypingWheel    int32 = 1031
	ctrlBidirectional  int32 = 1032
	ctrlReset          int32 = 1041
	ctrlClose          int32 = 1042
	ctrlTopClose       int32 = 1043
	ctrlPetVariantBase int32 = 1050
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
	stateTurn
	stateEat
	stateDig
	stateStand
	stateFaceGroom
)

const (
	reservedItem = -2
	noItem       = -1
)

type coatVariant struct {
	ID      string
	LabelEN string
	LabelJA string
}

var variants = []coatVariant{
	{ID: "wild_agouti", LabelEN: "Wild agouti", LabelJA: "野生色 / アグーチ"},
	{ID: "black", LabelEN: "Black", LabelJA: "ブラック"},
	{ID: "blue", LabelEN: "Blue (slate gray)", LabelJA: "ブルー（青みグレー）"},
	{ID: "gray", LabelEN: "Gray", LabelJA: "グレー"},
	{ID: "white_cream", LabelEN: "White / cream", LabelJA: "ホワイト / クリーム"},
	{ID: "sand_champagne", LabelEN: "Sand / champagne", LabelJA: "サンド / シャンパン"},
	{ID: "chocolate", LabelEN: "Chocolate", LabelJA: "チョコレート"},
	{ID: "black_pied", LabelEN: "Black pied", LabelJA: "ブラックパイド"},
	{ID: "agouti_pied", LabelEN: "Agouti pied", LabelJA: "アグーチパイド"},
	{ID: "blue_pied", LabelEN: "Blue pied (slate gray)", LabelJA: "ブルーパイド（青みグレー）"},
	{ID: "cream_pied", LabelEN: "Cream pied", LabelJA: "クリームパイド"},
}

type language int

const (
	langJapanese language = iota
	langEnglish
)

type settingsTab int

const (
	tabAnimals settingsTab = iota
	tabMotion
)

type coatMode int

const (
	coatFixed coatMode = iota
	coatSelected
	coatRandom
)

type deguPet struct {
	motionSet  int
	variant    int
	frame      int
	x          int
	laneOffset int
	item       int
	carryKind  int
	state      behaviorState
	prevState  behaviorState
	stateTicks int
	moveSpeed  int
	dir        int
	nextDir    int
}

type forageItem struct {
	x      int
	kind   int
	owner  int
	active bool
}

type petReaction struct {
	pet   int
	kind  int
	ticks int
}

type mouseHookStruct struct {
	pt          win.POINT
	mouseData   uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type petApp struct {
	hwnd               win.HWND
	hinst              win.HINSTANCE
	trayIcon           win.HICON
	keyHook            uintptr
	mouseHook          uintptr
	keyHookFailed      bool
	mouseHookFailed    bool
	frames             map[string][][]*image.RGBA
	forageSprites      []*image.RGBA
	wheel              *image.RGBA
	pets               []deguPet
	forage             []forageItem
	reactions          []petReaction
	variant            int
	coatMode           coatMode
	selectedCoats      [maxPetCount]int
	speed              int
	mode               behaviorMode
	petCount           int
	wheelEnabled       bool
	bidirectional      bool
	settingsHwnd       win.HWND
	settingsTab        settingsTab
	lang               language
	settingsFont       win.HFONT
	settingsTitleFont  win.HFONT
	settingsSmallFont  win.HFONT
	settingsBrush      win.HBRUSH
	settingsCard       win.HBRUSH
	settingsX          int32
	settingsY          int32
	settingsSaveFailed bool
	wheelX             int
	sceneW             int
	tickCount          int
	closing            atomic.Bool
}

type appSettings struct {
	Version       int   `json:"version"`
	Variant       int   `json:"variant"`
	CoatMode      int   `json:"coatMode"`
	SelectedCoats []int `json:"selectedCoats"`
	Speed         int   `json:"speed"`
	Mode          int   `json:"mode"`
	PetCount      int   `json:"petCount"`
	WheelEnabled  bool  `json:"wheelEnabled"`
	Bidirectional bool  `json:"bidirectional"`
	Language      int   `json:"language"`
	SettingsX     int32 `json:"settingsX"`
	SettingsY     int32 `json:"settingsY"`
}

var app *petApp

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	ntdll                  = syscall.NewLazyDLL("ntdll.dll")
	procAppendMenuW        = user32.NewProc("AppendMenuW")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procSetWindowsHookExW  = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHook  = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookExProc = user32.NewProc("CallNextHookEx")
	procUpdateLayeredWin   = user32.NewProc("UpdateLayeredWindow")
	procRtlMoveMemory      = ntdll.NewProc("RtlMoveMemory")
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
		hinst:         hinst,
		frames:        loadSprites(),
		forageSprites: loadForageSprites(),
		wheel:         loadWheelSprite(),
		variant:       0,
		coatMode:      coatRandom,
		selectedCoats: [maxPetCount]int{0, 1, 2, 4, 8, 6, 3, 7, 5, 9},
		speed:         3,
		mode:          modeRandom,
		petCount:      2,
		wheelEnabled:  true,
		bidirectional: true,
		settingsTab:   tabAnimals,
		lang:          langJapanese,
		settingsX:     120,
		settingsY:     120,
	}
	_ = app.loadSettings()

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
	app.installMouseHook()
	win.ShowWindow(app.hwnd, win.SW_SHOWNOACTIVATE)
	win.SetTimer(app.hwnd, timerID, timerInterval, 0)
	app.render()

	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) > 0 {
		if app.settingsHwnd != 0 && win.IsWindowVisible(app.settingsHwnd) && win.IsDialogMessage(app.settingsHwnd, &msg) {
			continue
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
	app.cleanup()
}

func wndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if app != nil && hwnd == app.settingsHwnd {
		return app.settingsWndProc(hwnd, msg, wParam, lParam)
	}
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
	case wmTyping:
		app.onTyping()
		return 0
	case wmMouseClick:
		x := int(int32(uint32(wParam)))
		y := int(int32(uint32(lParam)))
		app.onMouseClick(x, y)
		return 0
	case win.WM_COMMAND:
		id := uint16(wParam & 0xffff)
		notify := uint16((wParam >> 16) & 0xffff)
		if app != nil && app.handleSettingsCommand(int32(id), notify) {
			return 0
		}
		if app != nil && app.handleMenuCommand(id) {
			return 0
		}
	case win.WM_DESTROY:
		if hwnd == app.hwnd {
			win.PostQuitMessage(0)
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (a *petApp) resetPosition() {
	work := workArea()
	a.syncScene(work)
	a.setPetCount(a.petCount)
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsDirName, settingsFileName), nil
}

func (a *petApp) loadSettings() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var settings appSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	if settings.Version != 1 {
		return nil
	}
	a.variant = clamp(settings.Variant, 0, len(variants)-1)
	a.coatMode = normalizeCoatMode(settings.CoatMode)
	for i, variant := range settings.SelectedCoats {
		if i >= len(a.selectedCoats) {
			break
		}
		a.selectedCoats[i] = clamp(variant, 0, len(variants)-1)
	}
	a.speed = normalizeSpeed(settings.Speed)
	a.mode = normalizeBehaviorMode(settings.Mode)
	a.petCount = clamp(settings.PetCount, 1, maxPetCount)
	a.wheelEnabled = settings.WheelEnabled
	a.bidirectional = settings.Bidirectional
	a.lang = normalizeLanguage(settings.Language)
	if settings.SettingsX != 0 || settings.SettingsY != 0 {
		a.settingsX = settings.SettingsX
		a.settingsY = settings.SettingsY
		a.clampSettingsWindowPosition()
	}
	return nil
}

func (a *petApp) saveSettings() error {
	if a.settingsHwnd != 0 {
		a.rememberSettingsWindowPosition()
	}
	path, err := settingsPath()
	if err != nil {
		return err
	}
	coats := make([]int, len(a.selectedCoats))
	copy(coats, a.selectedCoats[:])
	settings := appSettings{
		Version:       1,
		Variant:       clamp(a.variant, 0, len(variants)-1),
		CoatMode:      int(a.coatMode),
		SelectedCoats: coats,
		Speed:         normalizeSpeed(a.speed),
		Mode:          int(normalizeBehaviorMode(int(a.mode))),
		PetCount:      clamp(a.petCount, 1, maxPetCount),
		WheelEnabled:  a.wheelEnabled,
		Bidirectional: a.bidirectional,
		Language:      int(normalizeLanguage(int(a.lang))),
		SettingsX:     a.settingsX,
		SettingsY:     a.settingsY,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomically(path, data)
}

func writeFileAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, settingsFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (a *petApp) persistSettings() {
	a.settingsSaveFailed = a.saveSettings() != nil
}

func normalizeCoatMode(mode int) coatMode {
	switch coatMode(mode) {
	case coatFixed, coatSelected, coatRandom:
		return coatMode(mode)
	default:
		return coatRandom
	}
}

func normalizeBehaviorMode(mode int) behaviorMode {
	switch behaviorMode(mode) {
	case modeKeyboard, modeRandom:
		return behaviorMode(mode)
	default:
		return modeRandom
	}
}

func normalizeLanguage(lang int) language {
	switch language(lang) {
	case langJapanese, langEnglish:
		return language(lang)
	default:
		return langJapanese
	}
}

func normalizeSpeed(speed int) int {
	switch speed {
	case 2, 3, 5:
		return speed
	default:
		return 3
	}
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
	a.tickReactions()
	a.syncNearbyWalkers()
	a.maybeStartSocial()
	a.tickCount++
}

func (a *petApp) tickReactions() {
	out := a.reactions[:0]
	for _, reaction := range a.reactions {
		reaction.ticks--
		if reaction.ticks > 0 && reaction.pet >= 0 && reaction.pet < len(a.pets) {
			out = append(out, reaction)
		}
	}
	a.reactions = out
}

func (a *petApp) tickPet(index int, p *deguPet) {
	if p.stateTicks <= 0 {
		switch p.state {
		case stateWheel:
			a.leaveWheel(p)
		case stateTurn:
			a.finishTurn(p)
		case stateEat:
			if p.item >= 0 && p.item < len(a.forage) {
				a.finishEating(index, p)
			} else {
				a.chooseRandomAction(p)
			}
		case stateDig, stateStand, stateFaceGroom:
			if a.mode == modeRandom {
				a.chooseRandomAction(p)
			} else {
				p.state = stateIdle
				p.moveSpeed = 0
				p.stateTicks = 12
			}
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
		p.x += speed * p.dir
		if p.state == stateForage {
			a.maybeStartGnawing(index, p)
		}
	}

	p.stateTicks--
	if p.x > a.sceneW+8 {
		a.resetPetAtEdge(index, p, 1)
	} else if p.x < -spriteW-8 {
		a.resetPetAtEdge(index, p, -1)
	}
	p.frame++
}

func (a *petApp) chooseRandomAction(p *deguPet) {
	roll := rand.Intn(100)
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.item = noItem
	p.carryKind = noItem
	if p.dir == 0 {
		p.dir = 1
	}
	if a.bidirectional && p.state != stateTurn && rand.Intn(100) < 16 {
		a.startTurn(p, -p.dir, stateWalk)
		return
	}
	if roll < 18 && a.maybeAssignForageTarget(p) {
		return
	}
	switch {
	case roll < 30:
		p.state = stateIdle
		p.moveSpeed = 0
		p.stateTicks = 24 + rand.Intn(58)
		return
	case roll < 70:
		p.state = stateWalk
		p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
		p.stateTicks = 34 + rand.Intn(92)
	case roll < 84:
		p.state = stateScurry
		p.moveSpeed = a.speed + 1 + rand.Intn(2)
		p.stateTicks = 10 + rand.Intn(18)
	case roll < 90:
		p.state = stateNibble
		p.moveSpeed = 0
		p.stateTicks = 26 + rand.Intn(32)
	case roll < 94:
		p.state = stateStand
		p.moveSpeed = 0
		p.stateTicks = 24 + rand.Intn(28)
	case roll < 98:
		p.state = stateFaceGroom
		p.moveSpeed = 0
		p.stateTicks = 28 + rand.Intn(30)
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
		src := a.frames[variants[a.petVariant(p)].ID][p.motionSet][frame]
		y := sceneH - spriteH - p.laneOffset
		drawPetSprite(canvas, src, p, p.x, y)
		if p.state == stateCarry && p.carryKind != noItem {
			propX := p.x + spriteW - 18
			if p.dir < 0 {
				propX = p.x + 18
			}
			a.drawForageProp(canvas, propX, y+35, p.carryKind)
		} else if (p.state == stateEat || p.state == stateDig) && p.carryKind != noItem {
			propX := p.x + spriteW - 20
			if p.dir < 0 {
				propX = p.x + 20
			}
			a.drawForageProp(canvas, propX, y+44, p.carryKind)
		}
	}

	if wheelActive {
		for i := range a.pets {
			p := &a.pets[i]
			if p.state != stateWheel {
				continue
			}
			frame := currentFrame(p.state, p.frame)
			src := a.frames[variants[a.petVariant(p)].ID][p.motionSet][frame]
			drawWheelRunner(canvas, wheelX, wheelY, src, p.frame)
		}
		drawWheelFront(canvas, wheelX, wheelY, a.tickCount)
	}
	a.drawReactions(canvas)
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
	case stateTurn:
		return frameFromSeqClamped(turnFrameSeq, frame, 2)
	case stateEat:
		return frameFromSeq(eatFrameSeq, frame, 3)
	case stateDig:
		return frameFromSeq(digFrameSeq, frame, 2)
	case stateStand:
		return frameFromSeq(standFrameSeq, frame, 4)
	case stateFaceGroom:
		return frameFromSeq(groomFrameSeq, frame, 3)
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

func frameFromSeqClamped(seq []int, frame, divisor int) int {
	if len(seq) == 0 {
		return idleStart
	}
	if divisor < 1 {
		divisor = 1
	}
	index := frame / divisor
	if index >= len(seq) {
		index = len(seq) - 1
	}
	return seq[index]
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

func (a *petApp) onMouseClick(screenX, screenY int) {
	index := a.petAtScreenPoint(screenX, screenY)
	if index < 0 {
		return
	}
	a.showPetReaction(index)
	a.render()
}

func (a *petApp) showPetReaction(index int) {
	if index < 0 || index >= len(a.pets) {
		return
	}
	kind := rand.Intn(3)
	for i := range a.reactions {
		if a.reactions[i].pet == index {
			a.reactions[i].kind = kind
			a.reactions[i].ticks = reactionTicks
			return
		}
	}
	a.reactions = append(a.reactions, petReaction{pet: index, kind: kind, ticks: reactionTicks})
}

func (a *petApp) petAtScreenPoint(screenX, screenY int) int {
	work := workArea()
	sceneX := screenX - int(work.Left)
	sceneY := screenY - (int(work.Bottom) - sceneH)
	return a.petAtScenePoint(sceneX, sceneY)
}

func (a *petApp) petAtScenePoint(sceneX, sceneY int) int {
	if sceneX < 0 || sceneX >= a.sceneW || sceneY < 0 || sceneY >= sceneH {
		return -1
	}
	for i := len(a.pets) - 1; i >= 0; i-- {
		if scenePointInPet(a.pets[i], sceneX, sceneY) {
			return i
		}
	}
	return -1
}

func scenePointInPet(p deguPet, sceneX, sceneY int) bool {
	if p.state == stateWheel {
		return false
	}
	y := sceneH - spriteH - p.laneOffset
	return sceneX >= p.x+6 && sceneX <= p.x+spriteW-6 && sceneY >= y+8 && sceneY <= y+spriteH-4
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
		if a.coatMode != coatRandom {
			a.pets[i].variant = a.variantForIndex(i)
		} else if a.pets[i].variant < 0 || a.pets[i].variant >= len(variants) {
			a.pets[i].variant = a.variantForIndex(i)
		}
		if a.pets[i].dir == 0 {
			a.pets[i].dir = 1
			a.pets[i].nextDir = 1
		}
	}
}

func (a *petApp) setCoatMode(mode coatMode) {
	a.coatMode = mode
	a.refreshPetVariants()
}

func (a *petApp) setFixedVariant(variant int) {
	a.variant = clamp(variant, 0, len(variants)-1)
	if a.coatMode == coatFixed {
		a.refreshPetVariants()
	}
}

func (a *petApp) setSelectedVariant(index int, variant int) {
	if index < 0 || index >= maxPetCount {
		return
	}
	a.selectedCoats[index] = clamp(variant, 0, len(variants)-1)
	if a.coatMode == coatSelected && index < len(a.pets) {
		a.pets[index].variant = a.selectedCoats[index]
	}
}

func (a *petApp) refreshPetVariants() {
	for i := range a.pets {
		a.pets[i].variant = a.variantForIndex(i)
	}
}

func (a *petApp) variantForIndex(index int) int {
	if len(variants) == 0 {
		return 0
	}
	switch a.coatMode {
	case coatRandom:
		return rand.Intn(len(variants))
	case coatSelected:
		if index >= 0 && index < len(a.selectedCoats) {
			return clamp(a.selectedCoats[index], 0, len(variants)-1)
		}
	}
	return clamp(a.variant, 0, len(variants)-1)
}

func (a *petApp) petVariant(p *deguPet) int {
	if len(variants) == 0 {
		return 0
	}
	return clamp(p.variant, 0, len(variants)-1)
}

func (a *petApp) newPet(index int) deguPet {
	spread := max(spriteW+24, a.sceneW/max(1, a.petCount+1))
	dir := 1
	if a.bidirectional && index%2 == 1 {
		dir = -1
	}
	x := -spriteW - index*spread - rand.Intn(80)
	if dir < 0 {
		x = a.sceneW + index*spread + rand.Intn(80)
	}
	p := deguPet{
		x:          x,
		laneOffset: (index % 3) * 5,
		variant:    a.variantForIndex(index),
		item:       noItem,
		carryKind:  noItem,
		motionSet:  rand.Intn(motionSets),
		state:      stateWalk,
		moveSpeed:  max(1, a.speed-1+rand.Intn(2)),
		stateTicks: 30 + rand.Intn(80),
		dir:        dir,
		nextDir:    dir,
	}
	if index == 0 {
		p.x = rand.Intn(max(1, a.sceneW-spriteW))
	}
	a.chooseRandomAction(&p)
	return p
}

func (a *petApp) resetPetAtLeft(index int, p *deguPet) {
	a.resetPetAtEdge(index, p, 1)
}

func (a *petApp) resetPetAtEdge(index int, p *deguPet, dir int) {
	a.releaseForage(index, p)
	if dir < 0 {
		p.x = a.sceneW + rand.Intn(120)
	} else {
		p.x = -spriteW - rand.Intn(120)
	}
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.variant = a.variantForIndex(index)
	p.item = noItem
	p.carryKind = noItem
	p.state = stateWalk
	p.prevState = stateWalk
	p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
	p.stateTicks = 40 + rand.Intn(90)
	p.dir = normalizeDir(dir)
	p.nextDir = p.dir
}

func (a *petApp) startTurn(p *deguPet, nextDir int, after behaviorState) {
	nextDir = normalizeDir(nextDir)
	if p.dir == 0 {
		p.dir = 1
	}
	if p.dir == nextDir {
		return
	}
	p.prevState = after
	p.state = stateTurn
	p.nextDir = nextDir
	p.moveSpeed = 0
	p.stateTicks = turnTicks
	p.frame = 0
	p.item = noItem
	p.carryKind = noItem
}

func (a *petApp) finishTurn(p *deguPet) {
	p.dir = normalizeDir(p.nextDir)
	p.nextDir = p.dir
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.state = p.prevState
	if p.state == stateTurn || p.state == stateWheel || p.state == stateGroom || p.state == stateForage || p.state == stateCarry {
		p.state = stateWalk
	}
	p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
	p.stateTicks = 32 + rand.Intn(70)
}

func (a *petApp) setBidirectional(enabled bool) {
	a.bidirectional = enabled
	if enabled {
		return
	}
	for i := range a.pets {
		p := &a.pets[i]
		p.dir = 1
		p.nextDir = 1
		if p.state == stateTurn {
			p.state = stateWalk
			p.moveSpeed = max(1, a.speed-1)
			p.stateTicks = 24 + rand.Intn(40)
		}
	}
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
	dir := normalizeDir(p.dir)
	for i, item := range a.forage {
		if !item.active || item.owner != noItem {
			continue
		}
		mouthX := p.x + spriteW - 22
		distance := item.x - mouthX
		if dir < 0 {
			mouthX = p.x + 22
			distance = mouthX - item.x
		}
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
	p.dir = dir
	p.nextDir = dir
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
	if p.dir < 0 {
		mouthX = p.x + 22
		if mouthX > item.x {
			return
		}
		p.x = clamp(item.x-22, 0, max(0, a.sceneW-spriteW))
	} else {
		if mouthX < item.x {
			return
		}
		p.x = clamp(item.x-spriteW+22, 0, max(0, a.sceneW-spriteW))
	}
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
	if kind == 0 || rand.Intn(100) < 54 {
		p.state = stateEat
		p.frame = 0
		p.carryKind = kind
		p.moveSpeed = 0
		p.stateTicks = 28 + rand.Intn(24)
		return
	}
	if kind == 2 && rand.Intn(100) < 58 {
		p.state = stateDig
		p.frame = 0
		p.carryKind = kind
		p.moveSpeed = 0
		p.stateTicks = 30 + rand.Intn(26)
		return
	}
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

func (a *petApp) finishEating(index int, p *deguPet) {
	a.releaseForage(index, p)
	p.carryKind = noItem
	if a.mode == modeRandom {
		a.chooseRandomAction(p)
		return
	}
	p.state = stateIdle
	p.moveSpeed = 0
	p.stateTicks = 12
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
			if normalizeDir(pj.dir) != normalizeDir(pi.dir) {
				a.startTurn(pj, pi.dir, stateWalk)
			}
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
			if normalizeDir(pi.dir) != 1 || normalizeDir(pj.dir) != -1 {
				if normalizeDir(pi.dir) != 1 {
					a.startTurn(pi, 1, stateWalk)
				}
				if normalizeDir(pj.dir) != -1 {
					a.startTurn(pj, -1, stateWalk)
				}
				return
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
		a.drawForageProp(dst, item.x, y, item.kind)
	}
}

func (a *petApp) drawReactions(dst *image.RGBA) {
	for _, reaction := range a.reactions {
		if reaction.pet < 0 || reaction.pet >= len(a.pets) {
			continue
		}
		p := a.pets[reaction.pet]
		if p.state == stateWheel {
			continue
		}
		baseY := sceneH - spriteH - p.laneOffset
		x := clamp(p.x+spriteW/2-18, 2, max(2, a.sceneW-42))
		y := clamp(baseY-26-(reactionTicks-reaction.ticks)/8, 0, sceneH-32)
		drawReactionBubble(dst, x, y, reaction.kind, reaction.ticks)
	}
}

func (a *petApp) showSettings() {
	a.clampSettingsWindowPosition()
	if a.settingsHwnd == 0 {
		a.createSettingsWindow()
	}
	a.syncSettingsWindow()
	win.ShowWindow(a.settingsHwnd, win.SW_SHOW)
	win.SetForegroundWindow(a.settingsHwnd)
}

func (a *petApp) createSettingsWindow() {
	title := a.txt("settingsTitle")
	a.ensureSettingsBrushes()
	style := uint32(win.WS_POPUP | win.WS_VISIBLE)
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOOLWINDOW,
		syscall.StringToUTF16Ptr(windowClass),
		syscall.StringToUTF16Ptr(title),
		style,
		a.settingsX, a.settingsY, settingsClientW, settingsClientH,
		a.hwnd, 0, a.hinst, nil,
	)
	if hwnd == 0 {
		return
	}
	a.settingsHwnd = hwnd
	a.ensureSettingsFonts()

	a.createButton(hwnd, ctrlTopClose, "x", 716, 18, 28, 28, 0)
	a.createButton(hwnd, ctrlTabAnimals, a.txt("tabAnimals"), 24, 122, 154, 38, win.WS_GROUP)
	a.createButton(hwnd, ctrlTabMotion, a.txt("tabMotion"), 24, 170, 154, 38, 0)

	if a.settingsTab == tabAnimals {
		a.createButton(hwnd, ctrlPetMinus, "-", 408, 150, 42, 32, 0)
		a.createButton(hwnd, ctrlPetPlus, "+", 506, 150, 42, 32, 0)

		a.createButton(hwnd, ctrlCoatFixed, a.txt("coatFixed"), 250, 250, 110, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlCoatSelected, a.txt("coatSelected"), 374, 250, 130, 30, 0)
		a.createButton(hwnd, ctrlCoatRandom, a.txt("coatRandom"), 518, 250, 110, 30, 0)

		a.createButton(hwnd, ctrlVariantCombo, "", 360, 298, 318, 34, 0)
		if a.coatMode == coatSelected {
			for i := 0; i < a.petCount; i++ {
				_, buttonRect := settingsPetVariantRects(i)
				a.createButton(hwnd, ctrlPetVariantBase+int32(i), "", buttonRect.Left, buttonRect.Top, buttonRect.Right-buttonRect.Left, buttonRect.Bottom-buttonRect.Top, 0)
			}
		}
	} else {
		a.createButton(hwnd, ctrlModeKeyboard, a.txt("modeKeyboard"), 250, 164, 210, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlModeRandom, a.txt("modeRandom"), 478, 164, 210, 32, 0)

		a.createButton(hwnd, ctrlSpeedSlow, a.txt("speedSlow"), 250, 270, 118, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlSpeedNormal, a.txt("speedNormal"), 384, 270, 118, 32, 0)
		a.createButton(hwnd, ctrlSpeedFast, a.txt("speedFast"), 518, 270, 118, 32, 0)

		a.createButton(hwnd, ctrlTypingWheel, a.txt("typingWheel"), 250, 378, 210, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlBidirectional, a.txt("naturalTurns"), 478, 378, 210, 32, 0)
	}

	a.createButton(hwnd, ctrlLanguageCombo, "", 322, 514, 180, 34, 0)
	a.createButton(hwnd, ctrlReset, a.txt("reset"), 556, 516, 78, 32, 0)
	a.createButton(hwnd, ctrlClose, a.txt("close"), 646, 516, 78, 32, 0)
}

func (a *petApp) ensureSettingsBrushes() {
	if a.settingsBrush == 0 {
		a.settingsBrush = solidBrush(244, 242, 235)
	}
	if a.settingsCard == 0 {
		a.settingsCard = solidBrush(255, 255, 251)
	}
}

func (a *petApp) ensureSettingsFonts() {
	if a.settingsFont == 0 {
		a.settingsFont = makeSettingsFont("Yu Gothic UI", -13, win.FW_NORMAL)
	}
	if a.settingsTitleFont == 0 {
		a.settingsTitleFont = makeSettingsFont("Yu Gothic UI", -19, win.FW_SEMIBOLD)
	}
	if a.settingsSmallFont == 0 {
		a.settingsSmallFont = makeSettingsFont("Yu Gothic UI", -12, win.FW_NORMAL)
	}
}

func makeSettingsFont(face string, height int32, weight int32) win.HFONT {
	var lf win.LOGFONT
	lf.LfHeight = height
	lf.LfWeight = weight
	lf.LfQuality = 5
	name := syscall.StringToUTF16(face)
	copy(lf.LfFaceName[:], name)
	return win.CreateFontIndirect(&lf)
}

func solidBrush(r, g, b byte) win.HBRUSH {
	return win.CreateBrushIndirect(&win.LOGBRUSH{
		LbStyle: win.BS_SOLID,
		LbColor: win.RGB(r, g, b),
	})
}

func (a *petApp) paintSettingsWindow(hwnd win.HWND) {
	a.ensureSettingsBrushes()
	a.ensureSettingsFonts()
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	if hdc == 0 {
		return
	}
	defer win.EndPaint(hwnd, &ps)

	drawRectFill(hdc, win.RECT{Left: 0, Top: 0, Right: settingsClientW, Bottom: settingsClientH}, rgb(246, 248, 244))
	drawRectFill(hdc, win.RECT{Left: 0, Top: 0, Right: 204, Bottom: settingsClientH}, rgb(22, 45, 38))
	drawRectFill(hdc, win.RECT{Left: 204, Top: 0, Right: settingsClientW, Bottom: settingsClientH}, rgb(247, 248, 244))

	drawRoundFill(hdc, win.RECT{Left: 226, Top: 96, Right: 736, Bottom: 506}, rgb(255, 255, 251), 18)
	drawRoundFill(hdc, win.RECT{Left: 226, Top: 514, Right: 736, Bottom: 552}, rgb(255, 255, 251), 14)

	if a.settingsTab == tabAnimals {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 142, Right: 708, Bottom: 192}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 238, Right: 708, Bottom: 292}, rgb(238, 242, 237), 14)
		if a.coatMode == coatSelected {
			drawRoundFill(hdc, win.RECT{Left: 238, Top: 338, Right: 708, Bottom: 502}, rgb(238, 242, 237), 14)
		} else {
			drawRoundFill(hdc, win.RECT{Left: 238, Top: 344, Right: 708, Bottom: 420}, rgb(238, 242, 237), 14)
		}
	} else {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 150, Right: 708, Bottom: 208}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 256, Right: 708, Bottom: 314}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 364, Right: 708, Bottom: 424}, rgb(238, 242, 237), 14)
	}

	drawTextLine(hdc, a.txt("settingsHeader"), win.RECT{Left: 24, Top: 28, Right: 178, Bottom: 56}, a.settingsTitleFont, rgb(245, 250, 244), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
	drawTextLine(hdc, a.txt("settingsLead"), win.RECT{Left: 24, Top: 58, Right: 178, Bottom: 96}, a.settingsSmallFont, rgb(183, 207, 195), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX)
	drawRoundFill(hdc, win.RECT{Left: 24, Top: 474, Right: 178, Bottom: 520}, rgb(35, 62, 52), 14)
	drawTextLine(hdc, a.settingsSidebarStatus(), win.RECT{Left: 38, Top: 486, Right: 164, Bottom: 508}, a.settingsSmallFont, rgb(231, 241, 233), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

	drawTextLine(hdc, a.settingsPageTitle(), win.RECT{Left: 226, Top: 28, Right: 620, Bottom: 56}, a.settingsTitleFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
	drawTextLine(hdc, a.settingsPageLead(), win.RECT{Left: 226, Top: 56, Right: 660, Bottom: 82}, a.settingsSmallFont, rgb(91, 104, 96), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

	labelColor := rgb(50, 61, 55)
	if a.settingsTab == tabAnimals {
		drawTextLine(hdc, a.txt("animalSection"), win.RECT{Left: 246, Top: 116, Right: 470, Bottom: 140}, a.settingsFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("deguCount"), win.RECT{Left: 250, Top: 154, Right: 390, Bottom: 184}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, fmt.Sprintf("%d", a.petCount), win.RECT{Left: 464, Top: 154, Right: 498, Bottom: 184}, a.settingsFont, rgb(25, 49, 40), win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("coatMode"), win.RECT{Left: 250, Top: 214, Right: 420, Bottom: 238}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("coatColor"), win.RECT{Left: 250, Top: 302, Right: 352, Bottom: 328}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		if a.coatMode == coatSelected {
			drawTextLine(hdc, a.txt("selectedCoats"), win.RECT{Left: 250, Top: 344, Right: 430, Bottom: 370}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
			for i := 0; i < a.petCount; i++ {
				numberRect, _ := settingsPetVariantRects(i)
				drawTextLine(hdc, fmt.Sprintf("%d", i+1), numberRect, a.settingsSmallFont, rgb(69, 78, 72), win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
			}
		} else {
			drawTextLine(hdc, a.txt("coatNote"), win.RECT{Left: 254, Top: 360, Right: 682, Bottom: 410}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX)
		}
	} else {
		drawTextLine(hdc, a.txt("mode"), win.RECT{Left: 246, Top: 126, Right: 400, Bottom: 150}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("speed"), win.RECT{Left: 246, Top: 232, Right: 400, Bottom: 256}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("motion"), win.RECT{Left: 246, Top: 340, Right: 400, Bottom: 364}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
	}
	drawTextLine(hdc, a.txt("language"), win.RECT{Left: 226, Top: 522, Right: 314, Bottom: 546}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
}

func (a *petApp) createStatic(parent win.HWND, text string, x, y, w, h int32) win.HWND {
	return a.createStaticWithFont(parent, text, x, y, w, h, a.settingsFont)
}

func (a *petApp) createStaticWithFont(parent win.HWND, text string, x, y, w, h int32, font win.HFONT) win.HWND {
	hwnd := win.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("STATIC"),
		syscall.StringToUTF16Ptr(text),
		win.WS_CHILD|win.WS_VISIBLE|win.SS_LEFT,
		x, y, w, h,
		parent, 0, a.hinst, nil,
	)
	a.setControlFont(hwnd, font)
	return hwnd
}

func (a *petApp) createButton(parent win.HWND, id int32, text string, x, y, w, h int32, style uint32) win.HWND {
	buttonStyle := uint32(win.WS_CHILD | win.WS_VISIBLE | win.WS_TABSTOP | win.BS_OWNERDRAW)
	if style&win.WS_GROUP != 0 {
		buttonStyle |= win.WS_GROUP
	}
	hwnd := win.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("BUTTON"),
		syscall.StringToUTF16Ptr(text),
		buttonStyle,
		x, y, w, h,
		parent, win.HMENU(uintptr(id)), a.hinst, nil,
	)
	a.setControlFont(hwnd, a.settingsFont)
	return hwnd
}

func (a *petApp) createCombo(parent win.HWND, id int32, x, y, w, h int32) win.HWND {
	hwnd := win.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("COMBOBOX"),
		nil,
		win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.WS_VSCROLL|win.CBS_DROPDOWNLIST,
		x, y, w, h,
		parent, win.HMENU(uintptr(id)), a.hinst, nil,
	)
	a.setControlFont(hwnd, a.settingsFont)
	return hwnd
}

func (a *petApp) drawSettingsButton(dis *win.DRAWITEMSTRUCT) bool {
	if dis == nil {
		return false
	}
	id := int32(dis.CtlID)
	label := a.settingsButtonLabel(id)
	if label == "" {
		return false
	}

	r := dis.RcItem
	enabled := dis.ItemState&win.ODS_DISABLED == 0 && win.IsWindowEnabled(dis.HwndItem)
	pressed := dis.ItemState&win.ODS_SELECTED != 0
	selected := a.settingsButtonSelected(id)
	selectField := a.settingsSelectButton(id)
	coatSelect := a.settingsCoatSelectButton(id)

	fill := rgb(252, 250, 244)
	text := rgb(35, 42, 35)
	if a.settingsSidebarButton(id) {
		fill = rgb(35, 62, 52)
		text = rgb(224, 238, 229)
		if selected {
			fill = rgb(232, 242, 231)
			text = rgb(22, 45, 38)
		} else if pressed {
			fill = rgb(48, 82, 68)
		}
	} else if id == ctrlTopClose {
		fill = rgb(239, 232, 228)
		text = rgb(122, 47, 38)
		if pressed {
			fill = rgb(224, 210, 204)
		}
	} else if selectField {
		fill = rgb(255, 255, 251)
	} else if selected {
		fill = rgb(48, 97, 73)
		text = rgb(255, 255, 251)
	} else if pressed {
		fill = rgb(226, 232, 220)
	}
	if !enabled {
		fill = rgb(230, 227, 216)
		text = rgb(126, 124, 112)
	}

	drawRectFill(dis.HDC, r, a.settingsButtonBackplate(id))
	inset := win.RECT{Left: r.Left + 1, Top: r.Top + 1, Right: r.Right - 1, Bottom: r.Bottom - 1}
	drawRoundFill(dis.HDC, inset, fill, 10)
	if enabled && dis.ItemState&win.ODS_FOCUS != 0 {
		focusRect := win.RECT{Left: r.Left + 4, Top: r.Top + 4, Right: r.Right - 4, Bottom: r.Bottom - 4}
		win.DrawFocusRect(dis.HDC, &focusRect)
	}

	textRect := win.RECT{Left: r.Left + 14, Top: r.Top, Right: r.Right - 14, Bottom: r.Bottom}
	if id == ctrlTopClose {
		textRect = win.RECT{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
	}
	if coatSelect {
		idx := a.settingsSelectVariant(id)
		drawVariantSwatch(dis.HDC, r.Left+13, r.Top+(r.Bottom-r.Top-16)/2, idx, enabled)
		textRect.Left += 26
	}
	if selectField {
		arrowRect := win.RECT{Left: r.Right - 24, Top: r.Top, Right: r.Right - 8, Bottom: r.Bottom}
		drawTextLine(dis.HDC, "v", arrowRect, a.settingsFont, text, win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		textRect.Right -= 24
	}

	flags := uint32(win.DT_VCENTER | win.DT_SINGLELINE | win.DT_NOPREFIX | win.DT_END_ELLIPSIS)
	if !selectField {
		flags |= win.DT_CENTER
	}
	drawTextLine(dis.HDC, label, textRect, a.settingsFont, text, flags)
	return true
}

func (a *petApp) settingsButtonLabel(id int32) string {
	switch id {
	case ctrlTabAnimals:
		return a.txt("tabAnimals")
	case ctrlTabMotion:
		return a.txt("tabMotion")
	case ctrlPetMinus:
		return "-"
	case ctrlPetPlus:
		return "+"
	case ctrlCoatFixed:
		return a.txt("coatFixed")
	case ctrlCoatSelected:
		return a.txt("coatSelected")
	case ctrlCoatRandom:
		return a.txt("coatRandom")
	case ctrlVariantCombo:
		return a.variantLabel(a.variant)
	case ctrlModeKeyboard:
		return a.txt("modeKeyboard")
	case ctrlModeRandom:
		return a.txt("modeRandom")
	case ctrlSpeedSlow:
		return a.txt("speedSlow")
	case ctrlSpeedNormal:
		return a.txt("speedNormal")
	case ctrlSpeedFast:
		return a.txt("speedFast")
	case ctrlTypingWheel:
		return a.txt("typingWheel")
	case ctrlBidirectional:
		return a.txt("naturalTurns")
	case ctrlLanguageCombo:
		if a.lang == langEnglish {
			return "English"
		}
		return "日本語"
	case ctrlTopClose:
		return "X"
	case ctrlReset:
		return a.txt("reset")
	case ctrlClose:
		return a.txt("close")
	}
	if id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount {
		return a.variantLabel(a.settingsSelectVariant(id))
	}
	return ""
}

func (a *petApp) settingsButtonSelected(id int32) bool {
	switch id {
	case ctrlTabAnimals:
		return a.settingsTab == tabAnimals
	case ctrlTabMotion:
		return a.settingsTab == tabMotion
	case ctrlCoatFixed:
		return a.coatMode == coatFixed
	case ctrlCoatSelected:
		return a.coatMode == coatSelected
	case ctrlCoatRandom:
		return a.coatMode == coatRandom
	case ctrlModeKeyboard:
		return a.mode == modeKeyboard
	case ctrlModeRandom:
		return a.mode == modeRandom
	case ctrlSpeedSlow:
		return a.speed == 2
	case ctrlSpeedNormal:
		return a.speed == 3
	case ctrlSpeedFast:
		return a.speed == 5
	case ctrlTypingWheel:
		return a.wheelEnabled
	case ctrlBidirectional:
		return a.bidirectional
	}
	return false
}

func (a *petApp) settingsSelectButton(id int32) bool {
	return id == ctrlVariantCombo || id == ctrlLanguageCombo || (id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount)
}

func (a *petApp) settingsSidebarButton(id int32) bool {
	return id == ctrlTabAnimals || id == ctrlTabMotion
}

func (a *petApp) settingsButtonBackplate(id int32) settingsRGB {
	switch id {
	case ctrlTabAnimals, ctrlTabMotion:
		return rgb(22, 45, 38)
	case ctrlTopClose:
		return rgb(247, 248, 244)
	case ctrlCoatFixed, ctrlCoatSelected, ctrlCoatRandom,
		ctrlModeKeyboard, ctrlModeRandom,
		ctrlSpeedSlow, ctrlSpeedNormal, ctrlSpeedFast,
		ctrlTypingWheel, ctrlBidirectional:
		return rgb(235, 232, 220)
	}
	if id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount {
		return rgb(235, 232, 220)
	}
	return rgb(255, 255, 251)
}

func (a *petApp) settingsSidebarStatus() string {
	if a.settingsSaveFailed {
		if a.lang == langEnglish {
			return "Settings save warning"
		}
		return "設定保存に失敗"
	}
	if a.keyHookFailed || a.mouseHookFailed {
		if a.lang == langEnglish {
			return "Input hook warning"
		}
		return "入力反応に問題"
	}
	if a.lang == langEnglish {
		if a.coatMode == coatRandom {
			return fmt.Sprintf("%d pets / random", a.petCount)
		}
		if a.coatMode == coatSelected {
			return fmt.Sprintf("%d pets / custom", a.petCount)
		}
		return fmt.Sprintf("%d pets / fixed", a.petCount)
	}
	if a.coatMode == coatRandom {
		return fmt.Sprintf("%d匹 / ランダム", a.petCount)
	}
	if a.coatMode == coatSelected {
		return fmt.Sprintf("%d匹 / 個別指定", a.petCount)
	}
	return fmt.Sprintf("%d匹 / 固定", a.petCount)
}

func (a *petApp) settingsPageTitle() string {
	if a.settingsTab == tabMotion {
		return a.txt("motionPageTitle")
	}
	return a.txt("animalPageTitle")
}

func (a *petApp) settingsPageLead() string {
	if a.settingsTab == tabMotion {
		return a.txt("motionPageLead")
	}
	return a.txt("animalPageLead")
}

func (a *petApp) settingsCoatSelectButton(id int32) bool {
	return id == ctrlVariantCombo || (id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount)
}

func settingsPetVariantRects(index int) (win.RECT, win.RECT) {
	col := index / 5
	row := index % 5
	if col > 1 {
		col = 1
	}
	left := int32(250 + col*228)
	top := int32(370 + row*26)
	numberRect := win.RECT{Left: left, Top: top, Right: left + 30, Bottom: top + 26}
	buttonRect := win.RECT{Left: left + 38, Top: top, Right: left + 214, Bottom: top + 26}
	return numberRect, buttonRect
}

func (a *petApp) settingsSelectVariant(id int32) int {
	if id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount {
		index := int(id - ctrlPetVariantBase)
		if index >= 0 && index < len(a.selectedCoats) {
			return clamp(a.selectedCoats[index], 0, len(variants)-1)
		}
	}
	return clamp(a.variant, 0, len(variants)-1)
}

type settingsRGB struct {
	r byte
	g byte
	b byte
}

func rgb(r, g, b byte) settingsRGB {
	return settingsRGB{r: r, g: g, b: b}
}

func drawRoundFill(hdc win.HDC, r win.RECT, c settingsRGB, radius int32) {
	brush := solidBrush(c.r, c.g, c.b)
	defer win.DeleteObject(win.HGDIOBJ(brush))
	oldBrush := win.SelectObject(hdc, win.HGDIOBJ(brush))
	oldPen := win.SelectObject(hdc, win.GetStockObject(win.NULL_PEN))
	win.RoundRect(hdc, r.Left, r.Top, r.Right, r.Bottom, radius, radius)
	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
}

func drawRectFill(hdc win.HDC, r win.RECT, c settingsRGB) {
	brush := solidBrush(c.r, c.g, c.b)
	defer win.DeleteObject(win.HGDIOBJ(brush))
	oldBrush := win.SelectObject(hdc, win.HGDIOBJ(brush))
	oldPen := win.SelectObject(hdc, win.GetStockObject(win.NULL_PEN))
	win.Rectangle_(hdc, r.Left, r.Top, r.Right, r.Bottom)
	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
}

func drawTextLine(hdc win.HDC, text string, r win.RECT, font win.HFONT, c settingsRGB, flags uint32) {
	win.SetBkMode(hdc, win.TRANSPARENT)
	win.SetTextColor(hdc, win.RGB(c.r, c.g, c.b))
	oldFont := win.SelectObject(hdc, win.HGDIOBJ(font))
	chars := syscall.StringToUTF16(text)
	if len(chars) > 0 {
		win.DrawTextEx(hdc, &chars[0], int32(len(chars)-1), &r, flags, nil)
	}
	win.SelectObject(hdc, oldFont)
}

func drawVariantSwatch(hdc win.HDC, x, y int32, variant int, enabled bool) {
	base, patch, pied := variantSwatch(variant)
	if !enabled {
		base = rgb(168, 164, 151)
		patch = rgb(224, 220, 207)
	}
	brush := solidBrush(base.r, base.g, base.b)
	oldBrush := win.SelectObject(hdc, win.HGDIOBJ(brush))
	oldPen := win.SelectObject(hdc, win.GetStockObject(win.NULL_PEN))
	win.Ellipse(hdc, x, y, x+16, y+16)
	win.SelectObject(hdc, oldBrush)
	win.DeleteObject(win.HGDIOBJ(brush))
	if pied {
		patchBrush := solidBrush(patch.r, patch.g, patch.b)
		oldBrush = win.SelectObject(hdc, win.HGDIOBJ(patchBrush))
		win.Ellipse(hdc, x+7, y+3, x+15, y+12)
		win.SelectObject(hdc, oldBrush)
		win.DeleteObject(win.HGDIOBJ(patchBrush))
	}
	win.SelectObject(hdc, oldPen)
}

func variantSwatch(index int) (settingsRGB, settingsRGB, bool) {
	if index < 0 || index >= len(variants) {
		return rgb(128, 120, 105), rgb(240, 235, 220), false
	}
	switch variants[index].ID {
	case "black":
		return rgb(42, 37, 33), rgb(240, 235, 220), false
	case "blue":
		return rgb(111, 116, 111), rgb(240, 235, 220), false
	case "gray":
		return rgb(128, 128, 120), rgb(240, 235, 220), false
	case "white_cream":
		return rgb(228, 218, 190), rgb(240, 235, 220), false
	case "sand_champagne":
		return rgb(185, 158, 115), rgb(240, 235, 220), false
	case "chocolate":
		return rgb(102, 69, 48), rgb(240, 235, 220), false
	case "black_pied":
		return rgb(42, 37, 33), rgb(236, 228, 204), true
	case "agouti_pied":
		return rgb(118, 96, 67), rgb(236, 228, 204), true
	case "blue_pied":
		return rgb(111, 116, 111), rgb(236, 228, 204), true
	case "cream_pied":
		return rgb(202, 175, 126), rgb(244, 238, 218), true
	default:
		return rgb(118, 96, 67), rgb(240, 235, 220), false
	}
}

func (a *petApp) setControlFont(hwnd win.HWND, font win.HFONT) {
	if hwnd == 0 || font == 0 {
		return
	}
	win.SendMessage(hwnd, win.WM_SETFONT, uintptr(font), 1)
}

func (a *petApp) settingsWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_COMMAND:
		id := int32(uint16(wParam & 0xffff))
		notify := uint16((wParam >> 16) & 0xffff)
		if a.handleSettingsCommand(id, notify) {
			return 0
		}
	case win.WM_PAINT:
		a.paintSettingsWindow(hwnd)
		return 0
	case win.WM_DRAWITEM:
		dis := drawItemStruct(lParam)
		if a.drawSettingsButton(&dis) {
			return 1
		}
		return 0
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_CTLCOLORSTATIC:
		win.SetBkMode(win.HDC(wParam), win.TRANSPARENT)
		win.SetTextColor(win.HDC(wParam), win.RGB(32, 37, 31))
		return uintptr(win.GetStockObject(win.NULL_BRUSH))
	case win.WM_NCHITTEST:
		if a.settingsDragHit(lParam) {
			return uintptr(win.HTCAPTION)
		}
		return uintptr(win.HTCLIENT)
	case win.WM_CLOSE:
		a.rememberSettingsWindowPosition()
		a.persistSettings()
		win.ShowWindow(hwnd, win.SW_HIDE)
		return 0
	case win.WM_DESTROY:
		if hwnd == a.settingsHwnd {
			a.settingsHwnd = 0
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (a *petApp) settingsDragHit(lParam uintptr) bool {
	var rect win.RECT
	if a.settingsHwnd == 0 || !win.GetWindowRect(a.settingsHwnd, &rect) {
		return false
	}
	x := int32(int16(lParam & 0xffff))
	y := int32(int16((lParam >> 16) & 0xffff))
	if y < rect.Top || y > rect.Top+88 {
		return false
	}
	if x >= rect.Right-64 && y <= rect.Top+58 {
		return false
	}
	return true
}

func drawItemStruct(lParam uintptr) win.DRAWITEMSTRUCT {
	var dis win.DRAWITEMSTRUCT
	if lParam != 0 {
		procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&dis)), lParam, unsafe.Sizeof(dis))
	}
	return dis
}

func (a *petApp) syncSettingsWindow() {
	if a.settingsHwnd == 0 {
		return
	}
	setWindowText(a.settingsHwnd, a.txt("settingsTitle"))
	a.setButtonChecked(ctrlTabAnimals, a.settingsTab == tabAnimals)
	a.setButtonChecked(ctrlTabMotion, a.settingsTab == tabMotion)
	a.setButtonChecked(ctrlModeKeyboard, a.mode == modeKeyboard)
	a.setButtonChecked(ctrlModeRandom, a.mode == modeRandom)
	a.setButtonChecked(ctrlSpeedSlow, a.speed == 2)
	a.setButtonChecked(ctrlSpeedNormal, a.speed == 3)
	a.setButtonChecked(ctrlSpeedFast, a.speed == 5)
	a.setButtonChecked(ctrlTypingWheel, a.wheelEnabled)
	a.setButtonChecked(ctrlBidirectional, a.bidirectional)
	a.setButtonChecked(ctrlCoatFixed, a.coatMode == coatFixed)
	a.setButtonChecked(ctrlCoatSelected, a.coatMode == coatSelected)
	a.setButtonChecked(ctrlCoatRandom, a.coatMode == coatRandom)
	a.syncSelectButton(ctrlVariantCombo)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlVariantCombo), a.coatMode == coatFixed)
	for i := 0; i < a.petCount; i++ {
		a.syncSelectButton(ctrlPetVariantBase + int32(i))
	}
	a.syncSelectButton(ctrlLanguageCombo)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlPetMinus), a.petCount > 1)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlPetPlus), a.petCount < maxPetCount)
}

func (a *petApp) setButtonChecked(id int32, checked bool) {
	h := win.GetDlgItem(a.settingsHwnd, id)
	if h == 0 {
		return
	}
	value := uintptr(win.BST_UNCHECKED)
	if checked {
		value = uintptr(win.BST_CHECKED)
	}
	win.SendMessage(h, win.BM_SETCHECK, value, 0)
	win.InvalidateRect(h, nil, true)
}

func (a *petApp) syncSelectButton(id int32) {
	h := win.GetDlgItem(a.settingsHwnd, id)
	if h == 0 {
		return
	}
	setWindowText(h, a.settingsButtonLabel(id))
	win.InvalidateRect(h, nil, true)
}

func (a *petApp) handleSettingsCommand(id int32, notify uint16) bool {
	switch id {
	case ctrlTabAnimals:
		a.settingsTab = tabAnimals
		a.recreateSettingsWindow()
	case ctrlTabMotion:
		a.settingsTab = tabMotion
		a.recreateSettingsWindow()
	case ctrlVariantCombo:
		sel, ok := a.pickVariantFromMenu(id, a.variant)
		if ok {
			a.setFixedVariant(sel)
		}
	case ctrlCoatFixed:
		a.setCoatMode(coatFixed)
		a.recreateSettingsWindow()
	case ctrlCoatSelected:
		a.setCoatMode(coatSelected)
		a.recreateSettingsWindow()
	case ctrlCoatRandom:
		a.setCoatMode(coatRandom)
		a.recreateSettingsWindow()
	case ctrlLanguageCombo:
		if lang, ok := a.pickLanguageFromMenu(id); ok {
			a.lang = lang
			a.recreateSettingsWindow()
		}
	case ctrlPetMinus:
		a.setPetCount(a.petCount - 1)
		a.recreateSettingsWindow()
	case ctrlPetPlus:
		a.setPetCount(a.petCount + 1)
		a.recreateSettingsWindow()
	case ctrlModeKeyboard:
		a.handleMenu(menuModeKeyboard)
	case ctrlModeRandom:
		a.handleMenu(menuModeRandom)
	case ctrlSpeedSlow:
		a.handleMenu(menuSpeedSlow)
	case ctrlSpeedNormal:
		a.handleMenu(menuSpeedNormal)
	case ctrlSpeedFast:
		a.handleMenu(menuSpeedFast)
	case ctrlTypingWheel:
		a.handleMenu(menuWheelToggle)
	case ctrlBidirectional:
		a.setBidirectional(!a.bidirectional)
	case ctrlReset:
		a.resetPosition()
		a.render()
	case ctrlClose, ctrlTopClose, int32(win.IDCANCEL):
		if a.settingsHwnd != 0 {
			a.rememberSettingsWindowPosition()
			win.ShowWindow(a.settingsHwnd, win.SW_HIDE)
		}
	default:
		if id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount {
			sel, ok := a.pickVariantFromMenu(id, a.settingsSelectVariant(id))
			if ok {
				a.setSelectedVariant(int(id-ctrlPetVariantBase), sel)
			}
			break
		}
		return false
	}
	a.syncSettingsWindow()
	a.persistSettings()
	a.render()
	return true
}

func (a *petApp) pickVariantFromMenu(id int32, selected int) (int, bool) {
	menu := win.CreatePopupMenu()
	for i := range variants {
		flags := uint32(win.MF_STRING)
		if i == selected {
			flags |= win.MF_CHECKED
		}
		appendMenu(menu, flags, uintptr(i+1), syscall.StringToUTF16Ptr(a.variantLabel(i)))
	}
	cmd := a.trackControlMenu(id, menu)
	win.DestroyMenu(menu)
	if cmd == 0 {
		return 0, false
	}
	choice := int(cmd) - 1
	if choice < 0 || choice >= len(variants) {
		return 0, false
	}
	return choice, true
}

func (a *petApp) pickLanguageFromMenu(id int32) (language, bool) {
	menu := win.CreatePopupMenu()
	appendMenu(menu, win.MF_STRING|checkedFlag(a.lang == langJapanese), 1, syscall.StringToUTF16Ptr("日本語"))
	appendMenu(menu, win.MF_STRING|checkedFlag(a.lang == langEnglish), 2, syscall.StringToUTF16Ptr("English"))
	cmd := a.trackControlMenu(id, menu)
	win.DestroyMenu(menu)
	switch cmd {
	case 1:
		return langJapanese, true
	case 2:
		return langEnglish, true
	}
	return a.lang, false
}

func checkedFlag(checked bool) uint32 {
	if checked {
		return win.MF_CHECKED
	}
	return 0
}

func (a *petApp) trackControlMenu(id int32, menu win.HMENU) uint32 {
	h := win.GetDlgItem(a.settingsHwnd, id)
	if h == 0 {
		return 0
	}
	var rect win.RECT
	if !win.GetWindowRect(h, &rect) {
		return 0
	}
	win.SetForegroundWindow(a.settingsHwnd)
	return win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_LEFTALIGN|win.TPM_TOPALIGN|win.TPM_RIGHTBUTTON, rect.Left, rect.Bottom+4, 0, a.settingsHwnd, nil)
}

func (a *petApp) recreateSettingsWindow() {
	if a.settingsHwnd != 0 {
		a.rememberSettingsWindowPosition()
		win.DestroyWindow(a.settingsHwnd)
		a.settingsHwnd = 0
	}
	a.showSettings()
}

func (a *petApp) rememberSettingsWindowPosition() {
	var rect win.RECT
	if a.settingsHwnd == 0 || !win.GetWindowRect(a.settingsHwnd, &rect) {
		return
	}
	a.settingsX = rect.Left
	a.settingsY = rect.Top
	a.clampSettingsWindowPosition()
}

func (a *petApp) clampSettingsWindowPosition() {
	work := workArea()
	maxX := work.Right - settingsClientW
	maxY := work.Bottom - settingsClientH
	if maxX < work.Left {
		maxX = work.Left
	}
	if maxY < work.Top {
		maxY = work.Top
	}
	a.settingsX = int32(clamp(int(a.settingsX), int(work.Left), int(maxX)))
	a.settingsY = int32(clamp(int(a.settingsY), int(work.Top), int(maxY)))
}

func (a *petApp) txt(key string) string {
	if a.lang == langEnglish {
		switch key {
		case "settingsTitle":
			return "Degu Desktop Settings"
		case "settingsHeader":
			return "Degu Desktop"
		case "settingsLead":
			return "Taskbar companion controls"
		case "animalPageTitle":
			return "Pets and coats"
		case "animalPageLead":
			return "Choose how many degus appear and how their coats are assigned."
		case "motionPageTitle":
			return "Motion behavior"
		case "motionPageLead":
			return "Tune keyboard reactions, random strolls, and turn behavior."
		case "tabAnimals":
			return "Animals"
		case "tabMotion":
			return "Motion"
		case "animalSection":
			return "Degu"
		case "deguCount":
			return "Visible pets"
		case "coatColor":
			return "Fixed coat"
		case "coatMode":
			return "Color appearance"
		case "coatFixed":
			return "Fixed"
		case "coatSelected":
			return "Choose each"
		case "coatRandom":
			return "Random"
		case "selectedCoats":
			return "Per-pet colors"
		case "coatNote":
			return "Random gives each degu its own coat. Pied coats have natural irregular patches."
		case "language":
			return "Language"
		case "mode":
			return "Mode"
		case "modeKeyboard":
			return "Keyboard reaction"
		case "modeRandom":
			return "Random stroll"
		case "speed":
			return "Speed"
		case "speedSlow":
			return "Slow"
		case "speedNormal":
			return "Normal"
		case "speedFast":
			return "Fast"
		case "motion":
			return "Motion"
		case "typingWheel":
			return "Typing wheel"
		case "naturalTurns":
			return "Natural left/right turns"
		case "reset":
			return "Reset"
		case "close":
			return "Close"
		case "exit":
			return "Exit"
		}
	}
	switch key {
	case "settingsTitle":
		return "デグーデスクトップ設定"
	case "settingsHeader":
		return "デグーデスクトップ"
	case "settingsLead":
		return "タスクバーで遊ぶペットの設定"
	case "animalPageTitle":
		return "デグーの数と毛色"
	case "animalPageLead":
		return "デグーの数と、毛色の選び方を調整します。"
	case "motionPageTitle":
		return "動きかた"
	case "motionPageLead":
		return "キーボードへの反応、ランダム散歩、左右ターンを調整します。"
	case "tabAnimals":
		return "動物"
	case "tabMotion":
		return "動き"
	case "animalSection":
		return "デグー"
	case "deguCount":
		return "出現数"
	case "coatColor":
		return "決まった毛色"
	case "coatMode":
		return "色の決め方"
	case "coatFixed":
		return "固定"
	case "coatSelected":
		return "1匹ずつ選ぶ"
	case "coatRandom":
		return "ランダム"
	case "selectedCoats":
		return "それぞれの毛色"
	case "coatNote":
		return "「ランダム」では、1匹ごとに異なる毛色で現れます。パイドは自然なぶち模様です。"
	case "language":
		return "表示言語"
	case "mode":
		return "モード"
	case "modeKeyboard":
		return "キーボード反応"
	case "modeRandom":
		return "ランダム散歩"
	case "speed":
		return "速度"
	case "speedSlow":
		return "ゆっくり"
	case "speedNormal":
		return "ふつう"
	case "speedFast":
		return "はやい"
	case "motion":
		return "動作"
	case "typingWheel":
		return "入力中だけ回し車"
	case "naturalTurns":
		return "自然な左右ターン"
	case "reset":
		return "整列"
	case "close":
		return "閉じる"
	case "exit":
		return "終了"
	}
	return key
}

func (a *petApp) variantLabel(i int) string {
	if i < 0 || i >= len(variants) {
		return ""
	}
	if a.lang == langEnglish {
		return variants[i].LabelEN
	}
	return variants[i].LabelJA
}

func (a *petApp) drawForageProp(dst *image.RGBA, x, y, kind int) {
	if kind >= 0 && kind < len(a.forageSprites) && a.forageSprites[kind] != nil {
		src := a.forageSprites[kind]
		drawFacingImage(dst, src, image.Rect(x-forageW/2, y-forageH, x+forageW/2, y), 1)
		return
	}
	fillCircle(dst, x, y-2, 3, rgba(170, 150, 94, 220))
}

func drawPetSprite(dst *image.RGBA, src *image.RGBA, p *deguPet, x, y int) {
	dir := normalizeDir(p.dir)
	if p.state == stateTurn {
		dir = turnDrawDirection(p.dir, p.nextDir)
	}
	drawFacingImage(dst, src, image.Rect(x, y, x+spriteW, y+spriteH), dir)
}

func turnDrawDirection(dir, nextDir int) int {
	if normalizeDir(dir) < 0 && normalizeDir(nextDir) > 0 {
		return -1
	}
	return 1
}

func drawFacingImage(dst *image.RGBA, src *image.RGBA, r image.Rectangle, dir int) {
	if r.Empty() {
		return
	}
	sb := src.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y < dst.Bounds().Min.Y || y >= dst.Bounds().Max.Y {
			continue
		}
		sy := sb.Min.Y + (y-r.Min.Y)*sb.Dy()/max(1, r.Dy())
		for x := r.Min.X; x < r.Max.X; x++ {
			if x < dst.Bounds().Min.X || x >= dst.Bounds().Max.X {
				continue
			}
			dx := x - r.Min.X
			sx := sb.Min.X + dx*sb.Dx()/max(1, r.Dx())
			if dir < 0 {
				sx = sb.Max.X - 1 - dx*sb.Dx()/max(1, r.Dx())
			}
			c := src.RGBAAt(sx, sy)
			if c.A == 0 {
				continue
			}
			dst.SetRGBA(x, y, overRGBA(dst.RGBAAt(x, y), c))
		}
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
	p.dir = 1
	p.nextDir = 1
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

func loadForageSprites() []*image.RGBA {
	names := []string{"forage_hay", "forage_twig", "forage_seed"}
	out := make([]*image.RGBA, len(names))
	for i, name := range names {
		data, err := fs.ReadFile(appassets.FS, "sprites/"+name+".png")
		if err != nil {
			continue
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		dst := image.NewRGBA(image.Rect(0, 0, forageW, forageH))
		draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)
		out[i] = dst
	}
	return out
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
	for i := range variants {
		flags := uint32(win.MF_STRING)
		if i == a.variant {
			flags |= win.MF_CHECKED
		}
		appendMenu(coatMenu, flags, uintptr(menuVariantBase+uint16(i)), syscall.StringToUTF16Ptr(a.variantLabel(i)))
	}
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(coatMenu), syscall.StringToUTF16Ptr(a.txt("coatColor")))
	coatModeMenu := win.CreatePopupMenu()
	appendChecked(coatModeMenu, menuCoatFixed, a.txt("coatFixed"), a.coatMode == coatFixed)
	appendChecked(coatModeMenu, menuCoatSelected, a.txt("coatSelected"), a.coatMode == coatSelected)
	appendChecked(coatModeMenu, menuCoatRandom, a.txt("coatRandom"), a.coatMode == coatRandom)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(coatModeMenu), syscall.StringToUTF16Ptr(a.txt("coatMode")))
	appendMenu(menu, win.MF_SEPARATOR, 0, nil)

	speedMenu := win.CreatePopupMenu()
	appendChecked(speedMenu, menuSpeedSlow, a.txt("speedSlow"), a.speed == 2)
	appendChecked(speedMenu, menuSpeedNormal, a.txt("speedNormal"), a.speed == 3)
	appendChecked(speedMenu, menuSpeedFast, a.txt("speedFast"), a.speed == 5)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(speedMenu), syscall.StringToUTF16Ptr(a.txt("speed")))

	modeMenu := win.CreatePopupMenu()
	appendChecked(modeMenu, menuModeKeyboard, a.txt("modeKeyboard"), a.mode == modeKeyboard)
	appendChecked(modeMenu, menuModeRandom, a.txt("modeRandom"), a.mode == modeRandom)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(modeMenu), syscall.StringToUTF16Ptr(a.txt("mode")))

	countMenu := win.CreatePopupMenu()
	appendChecked(countMenu, menuCount1, "1", a.petCount == 1)
	appendChecked(countMenu, menuCount2, "2", a.petCount == 2)
	appendChecked(countMenu, menuCount3, "3", a.petCount == 3)
	appendChecked(countMenu, menuCount5, "5", a.petCount == 5)
	appendChecked(countMenu, menuCount10, "10", a.petCount == 10)
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(countMenu), syscall.StringToUTF16Ptr(a.txt("deguCount")))

	appendChecked(menu, menuWheelToggle, a.txt("typingWheel"), a.wheelEnabled)
	appendMenu(menu, win.MF_SEPARATOR, 0, nil)
	appendMenu(menu, win.MF_STRING, uintptr(menuSettings), syscall.StringToUTF16Ptr(a.txt("settingsTitle")))
	appendMenu(menu, win.MF_STRING, uintptr(menuExit), syscall.StringToUTF16Ptr(a.txt("exit")))

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
	if !a.handleMenuCommand(id) {
		return
	}
	a.syncSettingsWindow()
	a.persistSettings()
}

func (a *petApp) handleMenuCommand(id uint16) bool {
	switch {
	case id == menuExit:
		a.closing.Store(true)
		win.DestroyWindow(a.hwnd)
	case id == menuSettings:
		a.showSettings()
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
	case id == menuCount10:
		a.setPetCount(10)
	case id == menuWheelToggle:
		a.wheelEnabled = !a.wheelEnabled
		for i := range a.pets {
			if a.pets[i].state == stateWheel {
				a.leaveWheel(&a.pets[i])
			}
		}
	case id == menuCoatFixed:
		a.setCoatMode(coatFixed)
	case id == menuCoatSelected:
		a.setCoatMode(coatSelected)
	case id == menuCoatRandom:
		a.setCoatMode(coatRandom)
	case id >= menuVariantBase && int(id-menuVariantBase) < len(variants):
		a.setFixedVariant(int(id - menuVariantBase))
		a.setCoatMode(coatFixed)
		a.render()
	default:
		return false
	}
	return true
}

func (a *petApp) cleanup() {
	win.KillTimer(a.hwnd, timerID)
	a.persistSettings()
	if a.settingsHwnd != 0 {
		win.DestroyWindow(a.settingsHwnd)
		a.settingsHwnd = 0
	}
	if a.keyHook != 0 {
		unhookWindowsHookEx(a.keyHook)
	}
	if a.mouseHook != 0 {
		unhookWindowsHookEx(a.mouseHook)
	}
	var nid win.NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	win.Shell_NotifyIcon(win.NIM_DELETE, &nid)
	if a.trayIcon != 0 {
		win.DestroyIcon(a.trayIcon)
	}
	if a.settingsBrush != 0 {
		win.DeleteObject(win.HGDIOBJ(a.settingsBrush))
		a.settingsBrush = 0
	}
	if a.settingsCard != 0 {
		win.DeleteObject(win.HGDIOBJ(a.settingsCard))
		a.settingsCard = 0
	}
	if a.settingsFont != 0 {
		win.DeleteObject(win.HGDIOBJ(a.settingsFont))
		a.settingsFont = 0
	}
	if a.settingsTitleFont != 0 {
		win.DeleteObject(win.HGDIOBJ(a.settingsTitleFont))
		a.settingsTitleFont = 0
	}
	if a.settingsSmallFont != 0 {
		win.DeleteObject(win.HGDIOBJ(a.settingsSmallFont))
		a.settingsSmallFont = 0
	}
}

func (a *petApp) installKeyboardHook() {
	cb := syscall.NewCallback(func(code int, wParam uintptr, lParam uintptr) uintptr {
		if code >= 0 && (wParam == win.WM_KEYDOWN || wParam == win.WM_SYSKEYDOWN) {
			a.postTypingFromHook()
		}
		return callNextHookEx(0, code, wParam, lParam)
	})
	a.keyHook = setWindowsHookEx(whKeyboardLL, cb, a.hinst, 0)
	a.keyHookFailed = a.keyHook == 0
}

func (a *petApp) installMouseHook() {
	cb := syscall.NewCallback(func(code int, wParam uintptr, lParam uintptr) uintptr {
		if code >= 0 && wParam == win.WM_LBUTTONDOWN {
			a.postMouseClickFromHook(lParam)
		}
		return callNextHookEx(0, code, wParam, lParam)
	})
	a.mouseHook = setWindowsHookEx(whMouseLL, cb, a.hinst, 0)
	a.mouseHookFailed = a.mouseHook == 0
}

func (a *petApp) postTypingFromHook() {
	defer recoverHookCallback()
	win.PostMessage(a.hwnd, wmTyping, 0, 0)
}

func (a *petApp) postMouseClickFromHook(lParam uintptr) {
	defer recoverHookCallback()
	pt := mouseHookPoint(lParam)
	win.PostMessage(a.hwnd, wmMouseClick, uintptr(uint32(pt.X)), uintptr(uint32(pt.Y)))
}

func recoverHookCallback() {
	_ = recover()
}

func mouseHookPoint(lParam uintptr) win.POINT {
	var hook mouseHookStruct
	if lParam != 0 {
		procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&hook)), lParam, unsafe.Sizeof(hook))
	}
	return hook.pt
}

func appendMenu(menu win.HMENU, flags uint32, item uintptr, text *uint16) bool {
	var textPtr uintptr
	if text != nil {
		textPtr = uintptr(unsafe.Pointer(text))
	}
	ret, _, _ := procAppendMenuW.Call(uintptr(menu), uintptr(flags), item, textPtr)
	return ret != 0
}

func setWindowText(hwnd win.HWND, text string) bool {
	ret, _, _ := procSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
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

func drawReactionBubble(dst *image.RGBA, x, y, kind, ticks int) {
	alpha := uint8(235)
	if ticks < 12 {
		alpha = uint8(max(0, ticks) * 235 / 12)
	}
	bg := rgba(255, 255, 248, alpha)
	edge := rgba(54, 89, 70, uint8(min(180, int(alpha))))
	shadow := rgba(22, 28, 24, uint8(min(70, int(alpha)/2)))
	drawRoundedRect(dst, x+2, y+3, 38, 24, 7, shadow)
	drawRoundedRect(dst, x, y, 38, 24, 7, bg)
	drawRoundedRectOutline(dst, x, y, 38, 24, 7, edge)
	drawBubbleTail(dst, x+17, y+23, alpha)
	switch kind % 3 {
	case 0:
		drawHeartIcon(dst, x+13, y+6, rgba(219, 72, 92, alpha))
	case 1:
		drawSmileIcon(dst, x+12, y+5, rgba(239, 184, 68, alpha), rgba(67, 62, 45, alpha))
	default:
		drawSparkleIcon(dst, x+11, y+5, rgba(70, 128, 104, alpha), rgba(239, 184, 68, alpha))
	}
}

func drawBubbleTail(dst *image.RGBA, x, y int, alpha uint8) {
	c := rgba(255, 255, 248, alpha)
	for row := 0; row < 6; row++ {
		for col := -row; col <= row; col++ {
			if image.Pt(x+col, y+row).In(dst.Bounds()) {
				dst.SetRGBA(x+col, y+row, overRGBA(dst.RGBAAt(x+col, y+row), c))
			}
		}
	}
}

func drawHeartIcon(dst *image.RGBA, x, y int, c color.RGBA) {
	fillCircle(dst, x+5, y+4, 4, c)
	fillCircle(dst, x+12, y+4, 4, c)
	for row := 4; row < 16; row++ {
		half := max(0, 9-row/2)
		for px := x + 8 - half; px <= x+8+half; px++ {
			if image.Pt(px, y+row).In(dst.Bounds()) {
				dst.SetRGBA(px, y+row, overRGBA(dst.RGBAAt(px, y+row), c))
			}
		}
	}
}

func drawSmileIcon(dst *image.RGBA, x, y int, face, ink color.RGBA) {
	fillCircle(dst, x+8, y+8, 8, face)
	fillCircle(dst, x+5, y+6, 1, ink)
	fillCircle(dst, x+11, y+6, 1, ink)
	for px := x + 4; px <= x+12; px++ {
		dx := px - (x + 8)
		py := y + 9 + abs(dx)/3
		for sy := 0; sy < 2; sy++ {
			if image.Pt(px, py+sy).In(dst.Bounds()) {
				dst.SetRGBA(px, py+sy, overRGBA(dst.RGBAAt(px, py+sy), ink))
			}
		}
	}
}

func drawSparkleIcon(dst *image.RGBA, x, y int, main, accent color.RGBA) {
	drawDiamond(dst, x+8, y+8, 8, main)
	drawPixelLine(dst, x+8, y, x+8, y+16, main)
	drawPixelLine(dst, x, y+8, x+16, y+8, main)
	drawDiamond(dst, x+22, y+5, 4, accent)
	drawDiamond(dst, x+20, y+15, 3, accent)
}

func drawDiamond(dst *image.RGBA, cx, cy, r int, c color.RGBA) {
	for py := cy - r; py <= cy+r; py++ {
		half := r - abs(py-cy)
		for px := cx - half; px <= cx+half; px++ {
			if image.Pt(px, py).In(dst.Bounds()) {
				dst.SetRGBA(px, py, overRGBA(dst.RGBAAt(px, py), c))
			}
		}
	}
}

func drawRoundedRect(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if roundedRectContains(px-x, py-y, w, h, radius) && image.Pt(px, py).In(dst.Bounds()) {
				dst.SetRGBA(px, py, overRGBA(dst.RGBAAt(px, py), c))
			}
		}
	}
}

func drawRoundedRectOutline(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if !image.Pt(px, py).In(dst.Bounds()) || !roundedRectContains(px-x, py-y, w, h, radius) {
				continue
			}
			if px == x || px == x+w-1 || py == y || py == y+h-1 ||
				!roundedRectContains(px-x-1, py-y, w, h, radius) ||
				!roundedRectContains(px-x+1, py-y, w, h, radius) ||
				!roundedRectContains(px-x, py-y-1, w, h, radius) ||
				!roundedRectContains(px-x, py-y+1, w, h, radius) {
				dst.SetRGBA(px, py, overRGBA(dst.RGBAAt(px, py), c))
			}
		}
	}
}

func roundedRectContains(x, y, w, h, radius int) bool {
	if x < 0 || y < 0 || x >= w || y >= h {
		return false
	}
	if (x >= radius && x < w-radius) || (y >= radius && y < h-radius) {
		return true
	}
	cx := radius
	if x >= w-radius {
		cx = w - radius - 1
	}
	cy := radius
	if y >= h-radius {
		cy = h - radius - 1
	}
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= radius*radius
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

func overRGBA(dst, src color.RGBA) color.RGBA {
	a := int(src.A)
	inv := 255 - a
	return color.RGBA{
		R: uint8((int(src.R)*a + int(dst.R)*inv) / 255),
		G: uint8((int(src.G)*a + int(dst.G)*inv) / 255),
		B: uint8((int(src.B)*a + int(dst.B)*inv) / 255),
		A: uint8(a + int(dst.A)*inv/255),
	}
}

func normalizeDir(dir int) int {
	if dir < 0 {
		return -1
	}
	return 1
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
