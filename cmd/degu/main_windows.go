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
	state      behaviorState
	stateTicks int
	moveSpeed  int
}

type petApp struct {
	hwnd         win.HWND
	hinst        win.HINSTANCE
	trayIcon     win.HICON
	keyHook      uintptr
	frames       map[string][][]*image.RGBA
	wheel        *image.RGBA
	pets         []deguPet
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
	for i := range a.pets {
		a.tickPet(&a.pets[i])
	}
	a.tickCount++
}

func (a *petApp) tickPet(p *deguPet) {
	if p.stateTicks <= 0 {
		if p.state == stateWheel {
			a.leaveWheel(p)
		} else if a.mode == modeRandom {
			a.chooseRandomAction(p)
		} else {
			p.state = stateIdle
			p.moveSpeed = 0
			p.stateTicks = 12
		}
	}

	speed := 0
	switch p.state {
	case stateWalk, stateScurry, stateHop:
		speed = p.moveSpeed
	case stateWheel:
		p.x = clamp(a.wheelX-spriteW/2+int(math.Sin(float64(p.frame)/3.0)*2), 0, max(0, a.sceneW-spriteW))
	}

	if speed > 0 {
		p.x += speed
		a.maybeEnterWheel(p)
	}

	p.stateTicks--
	if p.x > a.sceneW+8 {
		a.resetPetAtLeft(p)
	}
	p.frame++
}

func (a *petApp) chooseRandomAction(p *deguPet) {
	roll := rand.Intn(100)
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	switch {
	case roll < 28:
		p.state = stateIdle
		p.moveSpeed = 0
		p.stateTicks = 24 + rand.Intn(58)
		return
	case roll < 80:
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

	if a.wheelEnabled {
		drawWheel(canvas, a.wheelX-wheelSize/2, sceneH-wheelSize-2, a.tickCount, a.wheel)
	}

	for i := range a.pets {
		p := &a.pets[i]
		frame := currentFrame(p.state, p.frame)
		src := a.frames[variants[a.variant].ID][p.motionSet][frame]
		scaled := scaleImage(src, scale)
		y := sceneH - spriteH - p.laneOffset
		draw.Draw(canvas, image.Rect(p.x, y, p.x+spriteW, y+spriteH), scaled, image.Point{}, draw.Over)
	}
	updateLayeredWindow(a.hwnd, canvas, int(work.Left), int(work.Bottom)-sceneH)
}

func currentFrame(state behaviorState, frame int) int {
	switch state {
	case stateIdle:
		return idleStart + (frame/5)%idleFrames
	case stateWalk:
		return walkStart + (frame/2)%walkFrames
	case stateScurry:
		return scurryStart + frame%scurryFrames
	case stateNibble:
		return nibbleStart + (frame/3)%nibbleFrames
	case stateHop:
		return hopStart + (frame/2)%hopFrames
	case stateWheel:
		return scurryStart + frame%scurryFrames
	}
	return idleStart
}

func (a *petApp) onTyping() {
	if a.mode != modeKeyboard {
		return
	}
	for i := range a.pets {
		p := &a.pets[i]
		if a.wheelEnabled && i == 0 && abs((p.x+spriteW/2)-a.wheelX) < 180 {
			a.enterWheel(p)
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
	for i := range a.pets {
		a.pets[i].laneOffset = (i % 3) * 5
	}
}

func (a *petApp) newPet(index int) deguPet {
	spread := max(spriteW+24, a.sceneW/max(1, a.petCount+1))
	p := deguPet{
		x:          -spriteW - index*spread - rand.Intn(80),
		laneOffset: (index % 3) * 5,
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

func (a *petApp) resetPetAtLeft(p *deguPet) {
	p.x = -spriteW - rand.Intn(120)
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.state = stateWalk
	p.moveSpeed = max(1, a.speed-1+rand.Intn(2))
	p.stateTicks = 40 + rand.Intn(90)
}

func (a *petApp) maybeEnterWheel(p *deguPet) {
	if !a.wheelEnabled || p.state == stateWheel {
		return
	}
	center := p.x + spriteW/2
	if abs(center-a.wheelX) > 8 {
		return
	}
	if rand.Intn(100) < 38 {
		a.enterWheel(p)
	}
}

func (a *petApp) enterWheel(p *deguPet) {
	p.state = stateWheel
	p.frame = 0
	p.motionSet = rand.Intn(motionSets)
	p.moveSpeed = 0
	p.stateTicks = 52 + rand.Intn(96)
	p.x = clamp(a.wheelX-spriteW/2, 0, max(0, a.sceneW-spriteW))
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

	appendChecked(menu, menuWheelToggle, "Wheel motion", a.wheelEnabled)
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

func drawWheel(dst *image.RGBA, x, y, tick int, wheel *image.RGBA) {
	cx := x + wheelSize/2
	cy := y + wheelSize/2
	outer := float64(wheelSize/2 - 2)
	inner := outer - 5
	rim := rgba(92, 86, 76, 210)
	shadow := rgba(44, 41, 38, 120)
	spoke := rgba(128, 119, 102, 170)
	hub := rgba(86, 78, 68, 220)
	base := rgba(74, 67, 58, 210)

	if wheel != nil {
		draw.Draw(dst, image.Rect(x, y, x+wheelSize, y+wheelSize), wheel, image.Point{}, draw.Over)
	} else {
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
	}

	angle := float64(tick) * 0.22
	for i := 0; i < 8; i++ {
		a := angle + float64(i)*math.Pi/4
		x2 := cx + int(math.Cos(a)*(inner-2))
		y2 := cy + int(math.Sin(a)*(inner-2))
		drawPixelLine(dst, cx, cy, x2, y2, spoke)
	}

	fillCircle(dst, cx, cy, 4, hub)
	if wheel == nil {
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
