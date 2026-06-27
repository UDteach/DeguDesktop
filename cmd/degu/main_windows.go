//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	appassets "degu-desktop/assets"
	"github.com/lxn/win"
)

const (
	appName                     = "Degu Desktop"
	windowClass                 = "DeguDesktopPetWindow"
	wmTray                      = win.WM_APP + 1
	wmTyping                    = win.WM_APP + 2
	wmMouseClick                = win.WM_APP + 3
	wmUpdateReady               = win.WM_APP + 4
	wmUpdateFailed              = win.WM_APP + 5
	wmUpdateInstallReady        = win.WM_APP + 6
	timerID                     = 42
	timerInterval               = 55
	frameW                      = 96
	frameH                      = 64
	frameCount                  = 62
	motionSets                  = 10
	scale                       = 1
	spriteW                     = frameW * scale
	spriteH                     = frameH * scale
	forageW                     = 32
	forageH                     = 24
	sceneH                      = 92
	wheelSize                   = 72
	maxPetCount                 = 10
	maxForage                   = 5
	defaultOverlayOffsetY       = 10
	minOverlayOffsetY           = -48
	maxOverlayOffsetY           = 96
	overlayOffsetStep           = 4
	defaultWalkRangeStart       = 0
	defaultWalkRangeEnd         = 100
	minWalkRangeSpan            = 25
	walkRangeStep               = 5
	randomWheelChance           = 14
	randomWheelMinTicks         = 120
	randomWheelExtraTicks       = 120
	wheelKeyHold                = 18
	turnTicks                   = 16
	reactionTicks               = 54
	settingsClientW       int32 = 760
	settingsClientH       int32 = 620
	settingsDirName             = "DeguDesktop"
	settingsFileName            = "settings.json"
	defaultUpdateAPIURL         = "https://api.github.com/repos/UDteach/DeguDesktop/releases/latest"
	updaterApplyArg             = "--degudesktop-apply-update"
	updaterCleanupArg           = "--degudesktop-cleanup"
	updateTempPrefix            = "degu-desktop-update-"
	monitorPrimaryFlag          = 1
)

const (
	whKeyboardLL = 13
	whMouseLL    = 14
)

const (
	idleStart      = 0
	idleFrames     = 4
	walkStart      = 4
	walkFrames     = 8
	scurryStart    = 12
	scurryFrames   = 8
	nibbleStart    = 20
	nibbleFrames   = 6
	hopStart       = 26
	hopFrames      = 6
	turnStart      = 32
	turnFrames     = 8
	eatStart       = 40
	eatFrames      = 4
	digStart       = 44
	digFrames      = 4
	standStart     = 48
	standFrames    = 4
	groomStart     = 52
	groomFrames    = 4
	wheelRunStart  = 56
	wheelRunFrames = 6
)

var (
	idleFrameSeq     = []int{idleStart, idleStart + 1, idleStart + 3, idleStart + 1}
	walkFrameSeq     = []int{walkStart, walkStart + 1, walkStart + 3, walkStart + 1}
	nibbleFrameSeq   = []int{nibbleStart, nibbleStart + 1, nibbleStart + 2, nibbleStart + 1}
	hopFrameSeq      = []int{hopStart, hopStart + 1, hopStart + 2, hopStart + 3}
	turnFrameSeq     = []int{turnStart, turnStart + 1, turnStart + 2, turnStart + 3, turnStart + 4, turnStart + 5, turnStart + 6, turnStart + 7}
	eatFrameSeq      = []int{eatStart, eatStart + 1, eatStart + 2, eatStart + 3}
	digFrameSeq      = []int{digStart, digStart + 1, digStart + 2, digStart + 3}
	standFrameSeq    = []int{standStart, standStart + 1, standStart + 2, standStart + 3}
	groomFrameSeq    = []int{groomStart, groomStart + 1, groomStart + 2, groomStart + 3}
	wheelRunFrameSeq = []int{wheelRunStart, wheelRunStart + 1, wheelRunStart + 2, wheelRunStart + 3, wheelRunStart + 4, wheelRunStart + 5}
)

const (
	menuExit          uint16 = 100
	menuModeKeyboard  uint16 = 101
	menuModeRandom    uint16 = 102
	menuSpeedSlow     uint16 = 110
	menuSpeedNormal   uint16 = 111
	menuSpeedFast     uint16 = 112
	menuCountBase     uint16 = 119
	menuWheelToggle   uint16 = 130
	menuCoatFixed     uint16 = 131
	menuCoatSelected  uint16 = 132
	menuCoatRandom    uint16 = 133
	menuSettings      uint16 = 140
	menuToggleHidden  uint16 = 141
	menuCheckUpdate   uint16 = 150
	menuInstallUpdate uint16 = 151
	menuVariantBase   uint16 = 200
)

const (
	ctrlTabHome          int32 = 999
	ctrlTabAnimals       int32 = 1000
	ctrlTabMotion        int32 = 1001
	ctrlTabDisplay       int32 = 1009
	ctrlTabUpdates       int32 = 1010
	ctrlVariantCombo     int32 = 1002
	ctrlPetMinus         int32 = 1003
	ctrlPetPlus          int32 = 1004
	ctrlLanguageCombo    int32 = 1005
	ctrlCoatFixed        int32 = 1006
	ctrlCoatSelected     int32 = 1007
	ctrlCoatRandom       int32 = 1008
	ctrlModeKeyboard     int32 = 1011
	ctrlModeRandom       int32 = 1012
	ctrlSpeedSlow        int32 = 1021
	ctrlSpeedNormal      int32 = 1022
	ctrlSpeedFast        int32 = 1023
	ctrlTypingWheel      int32 = 1031
	ctrlBidirectional    int32 = 1032
	ctrlPositionTaskbar  int32 = 1033
	ctrlPositionBottom   int32 = 1034
	ctrlOffsetUp         int32 = 1035
	ctrlOffsetDown       int32 = 1036
	ctrlLaneStaggered    int32 = 1037
	ctrlLaneAligned      int32 = 1038
	ctrlReset            int32 = 1041
	ctrlClose            int32 = 1042
	ctrlTopClose         int32 = 1043
	ctrlNameLabels       int32 = 1044
	ctrlPetVariantBase   int32 = 1050
	ctrlPetNameBase      int32 = 1070
	ctrlRenameEdit       int32 = 1100
	ctrlRenameOK         int32 = 1101
	ctrlRenameCancel     int32 = 1102
	ctrlDisplayPrev      int32 = 1110
	ctrlDisplayNext      int32 = 1111
	ctrlRangeFull        int32 = 1112
	ctrlRangeNarrow      int32 = 1113
	ctrlRangeWide        int32 = 1114
	ctrlRangeLeft        int32 = 1115
	ctrlRangeRight       int32 = 1116
	ctrlRangeStartScroll int32 = 1117
	ctrlRangeEndScroll   int32 = 1118
	ctrlUpdateCheck      int32 = 1120
	ctrlUpdateInstall    int32 = 1121
	ctrlHomeAnimals      int32 = 1130
	ctrlHomeMotion       int32 = 1131
	ctrlHomeDisplay      int32 = 1132
	ctrlHomeUpdates      int32 = 1133
	ctrlDisplaySingle    int32 = 1134
	ctrlDisplaySpan      int32 = 1135
	ctrlDisplaySpanLess  int32 = 1136
	ctrlDisplaySpanMore  int32 = 1137
)

var appVersion = "dev"
var updateAPIURL = defaultUpdateAPIURL

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
	tabHome settingsTab = iota
	tabAnimals
	tabMotion
	tabDisplay
	tabUpdates
)

type coatMode int

const (
	coatFixed coatMode = iota
	coatSelected
	coatRandom
)

type overlayPositionMode int

const (
	positionTaskbarEdge overlayPositionMode = iota
	positionScreenBottom
)

type displayScope int

const (
	displayScopeSingle displayScope = iota
	displayScopeSpan
)

type petLaneMode int

const (
	laneStaggered petLaneMode = iota
	laneAligned
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

type githubRelease struct {
	TagName    string               `json:"tag_name"`
	HTMLURL    string               `json:"html_url"`
	Draft      bool                 `json:"draft"`
	Prerelease bool                 `json:"prerelease"`
	Assets     []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
}

type updateState struct {
	mu         sync.Mutex
	latest     *githubRelease
	lastError  string
	checking   atomic.Bool
	installing atomic.Bool
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
	petNames           [maxPetCount]string
	nameLabels         bool
	speed              int
	mode               behaviorMode
	petCount           int
	wheelEnabled       bool
	bidirectional      bool
	positionMode       overlayPositionMode
	overlayOffsetY     int
	laneMode           petLaneMode
	displayIndex       int
	displayScope       displayScope
	displaySpanEnd     int
	walkRangeStart     int
	walkRangeEnd       int
	settingsHwnd       win.HWND
	settingsTab        settingsTab
	lang               language
	settingsFont       win.HFONT
	settingsTitleFont  win.HFONT
	settingsSmallFont  win.HFONT
	settingsBrush      win.HBRUSH
	settingsCard       win.HBRUSH
	settingsTooltip    win.HWND
	settingsTipText    [][]uint16
	settingsHoverTip   string
	settingsX          int32
	settingsY          int32
	settingsSaveFailed bool
	update             updateState
	nameHwnd           win.HWND
	nameText           string
	hoverPet           int
	temporarilyHidden  bool
	renameHwnd         win.HWND
	renameEdit         win.HWND
	renameIndex        int
	wheelX             int
	sceneW             int
	tickCount          int
	closing            atomic.Bool
}

type appSettings struct {
	Version        int      `json:"version"`
	Variant        int      `json:"variant"`
	CoatMode       int      `json:"coatMode"`
	SelectedCoats  []int    `json:"selectedCoats"`
	Speed          int      `json:"speed"`
	Mode           int      `json:"mode"`
	PetCount       int      `json:"petCount"`
	WheelEnabled   bool     `json:"wheelEnabled"`
	Bidirectional  bool     `json:"bidirectional"`
	PositionMode   *int     `json:"positionMode,omitempty"`
	VerticalOffset *int     `json:"verticalOffset,omitempty"`
	LaneMode       *int     `json:"laneMode,omitempty"`
	DisplayIndex   *int     `json:"displayIndex,omitempty"`
	DisplayScope   *int     `json:"displayScope,omitempty"`
	DisplaySpanEnd *int     `json:"displaySpanEnd,omitempty"`
	WalkRangeStart *int     `json:"walkRangeStart,omitempty"`
	WalkRangeEnd   *int     `json:"walkRangeEnd,omitempty"`
	Language       int      `json:"language"`
	SettingsX      int32    `json:"settingsX"`
	SettingsY      int32    `json:"settingsY"`
	NameLabels     bool     `json:"nameLabels"`
	PetNames       []string `json:"petNames,omitempty"`
}

var app *petApp

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	ntdll                   = syscall.NewLazyDLL("ntdll.dll")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procGetDlgCtrlID        = user32.NewProc("GetDlgCtrlID")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHook   = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookExProc  = user32.NewProc("CallNextHookEx")
	procUpdateLayeredWin    = user32.NewProc("UpdateLayeredWindow")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procRtlMoveMemory       = ntdll.NewProc("RtlMoveMemory")
)

const (
	acSrcOver       = 0
	ulwAlpha        = 0x00000002
	spiGetWorkArea  = 0x0030
	sbsHorz         = 0x0000
	sbmSetPos       = 0x00E0
	sbmSetRange     = 0x00E2
	sbLineLeft      = 0
	sbLineRight     = 1
	sbPageLeft      = 2
	sbPageRight     = 3
	sbThumbPosition = 4
	sbThumbTrack    = 5
	sbLeft          = 6
	sbRight         = 7
	sbEndScroll     = 8
)

func main() {
	if runUpdaterUtility(os.Args[1:]) {
		return
	}
	cleanupDir := updateCleanupDir(os.Args[1:])

	runtime.LockOSThread()
	rand.Seed(time.Now().UnixNano())

	hinst := win.GetModuleHandle(nil)
	app = &petApp{
		hinst:          hinst,
		frames:         loadSprites(),
		forageSprites:  loadForageSprites(),
		wheel:          loadWheelSprite(),
		variant:        0,
		coatMode:       coatRandom,
		selectedCoats:  [maxPetCount]int{0, 1, 2, 4, 8, 6, 3, 7, 5, 9},
		speed:          3,
		mode:           modeRandom,
		petCount:       2,
		wheelEnabled:   true,
		bidirectional:  true,
		positionMode:   positionTaskbarEdge,
		overlayOffsetY: defaultOverlayOffsetY,
		laneMode:       laneStaggered,
		displayIndex:   0,
		displayScope:   displayScopeSingle,
		displaySpanEnd: 0,
		walkRangeStart: defaultWalkRangeStart,
		walkRangeEnd:   defaultWalkRangeEnd,
		settingsTab:    tabHome,
		lang:           langJapanese,
		settingsX:      120,
		settingsY:      120,
		hoverPet:       -1,
		renameIndex:    -1,
	}
	_ = app.loadSettings()
	if cleanupDir != "" {
		go cleanupUpdateTempDirLater(cleanupDir)
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
	app.showStartupToast()
	app.startUpdateCheck(false)
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
	if app != nil && hwnd == app.nameHwnd {
		return app.nameWndProc(hwnd, msg, wParam, lParam)
	}
	if app != nil && hwnd == app.renameHwnd {
		return app.renameWndProc(hwnd, msg, wParam, lParam)
	}
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
	case wmUpdateReady:
		app.onUpdateReady(wParam != 0)
		return 0
	case wmUpdateFailed:
		app.onUpdateFailed(wParam != 0)
		return 0
	case wmUpdateInstallReady:
		app.onUpdateInstallReady()
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
	overlay := a.overlayRect()
	a.syncScene(overlay)
	a.setPetCount(a.petCount)
	a.arrangePetsForOverlay(overlay)
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
	for i, name := range settings.PetNames {
		if i >= len(a.petNames) {
			break
		}
		a.petNames[i] = sanitizePetName(name)
	}
	a.speed = normalizeSpeed(settings.Speed)
	a.mode = normalizeBehaviorMode(settings.Mode)
	a.petCount = clamp(settings.PetCount, 1, maxPetCount)
	a.wheelEnabled = settings.WheelEnabled
	a.bidirectional = settings.Bidirectional
	if settings.PositionMode != nil {
		a.positionMode = normalizeOverlayPositionMode(*settings.PositionMode)
	}
	if settings.VerticalOffset != nil {
		a.overlayOffsetY = normalizeOverlayOffset(*settings.VerticalOffset)
	}
	if settings.LaneMode != nil {
		a.laneMode = normalizePetLaneMode(*settings.LaneMode)
	}
	if settings.DisplayIndex != nil {
		a.displayIndex = normalizeDisplayIndex(*settings.DisplayIndex, len(monitorAreas()))
	}
	if settings.DisplayScope != nil {
		a.displayScope = normalizeDisplayScope(*settings.DisplayScope)
	}
	if settings.DisplaySpanEnd != nil {
		a.displaySpanEnd = *settings.DisplaySpanEnd
	}
	a.normalizeDisplaySelection(len(displayAreaForScope(a.displayScope)))
	if settings.WalkRangeStart != nil || settings.WalkRangeEnd != nil {
		start := a.walkRangeStart
		end := a.walkRangeEnd
		if settings.WalkRangeStart != nil {
			start = *settings.WalkRangeStart
		}
		if settings.WalkRangeEnd != nil {
			end = *settings.WalkRangeEnd
		}
		a.walkRangeStart, a.walkRangeEnd = normalizeWalkRange(start, end)
	}
	a.nameLabels = settings.NameLabels
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
	names := make([]string, len(a.petNames))
	for i := range a.petNames {
		names[i] = sanitizePetName(a.petNames[i])
	}
	positionMode := int(normalizeOverlayPositionMode(int(a.positionMode)))
	verticalOffset := normalizeOverlayOffset(a.overlayOffsetY)
	laneMode := int(normalizePetLaneMode(int(a.laneMode)))
	displayScope, displayIndex, displaySpanEnd := a.normalizedDisplaySelection(len(displayAreaForScope(a.displayScope)))
	displayScopeValue := int(displayScope)
	walkStart, walkEnd := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	settings := appSettings{
		Version:        1,
		Variant:        clamp(a.variant, 0, len(variants)-1),
		CoatMode:       int(a.coatMode),
		SelectedCoats:  coats,
		Speed:          normalizeSpeed(a.speed),
		Mode:           int(normalizeBehaviorMode(int(a.mode))),
		PetCount:       clamp(a.petCount, 1, maxPetCount),
		WheelEnabled:   a.wheelEnabled,
		Bidirectional:  a.bidirectional,
		PositionMode:   &positionMode,
		VerticalOffset: &verticalOffset,
		LaneMode:       &laneMode,
		DisplayIndex:   &displayIndex,
		DisplayScope:   &displayScopeValue,
		DisplaySpanEnd: &displaySpanEnd,
		WalkRangeStart: &walkStart,
		WalkRangeEnd:   &walkEnd,
		Language:       int(normalizeLanguage(int(a.lang))),
		SettingsX:      a.settingsX,
		SettingsY:      a.settingsY,
		NameLabels:     a.nameLabels,
		PetNames:       names,
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

func normalizeOverlayPositionMode(mode int) overlayPositionMode {
	switch overlayPositionMode(mode) {
	case positionTaskbarEdge, positionScreenBottom:
		return overlayPositionMode(mode)
	default:
		return positionTaskbarEdge
	}
}

func normalizeOverlayOffset(offset int) int {
	return clamp(offset, minOverlayOffsetY, maxOverlayOffsetY)
}

func normalizeDisplayIndex(index int, count int) int {
	if count <= 0 {
		return 0
	}
	return clamp(index, 0, count-1)
}

func normalizeDisplayScope(mode int) displayScope {
	switch displayScope(mode) {
	case displayScopeSingle, displayScopeSpan:
		return displayScope(mode)
	default:
		return displayScopeSingle
	}
}

func normalizeDisplaySpan(start, end, count int) (int, int) {
	if count <= 0 {
		return 0, 0
	}
	start = normalizeDisplayIndex(start, count)
	end = normalizeDisplayIndex(end, count)
	if end < start {
		start, end = end, start
	}
	return start, end
}

func normalizeWalkRange(start, end int) (int, int) {
	start = clamp(start, 0, 100)
	end = clamp(end, 0, 100)
	if end < start {
		start, end = end, start
	}
	if end-start >= minWalkRangeSpan {
		return start, end
	}
	mid := (start + end) / 2
	start = mid - minWalkRangeSpan/2
	end = start + minWalkRangeSpan
	if start < 0 {
		start = 0
		end = minWalkRangeSpan
	}
	if end > 100 {
		end = 100
		start = 100 - minWalkRangeSpan
	}
	return start, end
}

func normalizePetLaneMode(mode int) petLaneMode {
	switch petLaneMode(mode) {
	case laneStaggered, laneAligned:
		return petLaneMode(mode)
	default:
		return laneStaggered
	}
}

func (a *petApp) tick() {
	if a.closing.Load() {
		return
	}
	if a.temporarilyHidden {
		a.hideNameWindow()
		return
	}
	a.syncScene(a.overlayRect())
	a.updateHoverName()
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
	if a.tryStartRandomWheel(p, rand.Intn(100)) {
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
	if a.temporarilyHidden {
		a.hideNameWindow()
		if a.hwnd != 0 {
			win.ShowWindow(a.hwnd, win.SW_HIDE)
		}
		return
	}
	overlay := a.overlayRect()
	a.syncScene(overlay)
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
	updateLayeredWindow(a.hwnd, canvas, int(overlay.Left), int(overlay.Top))
}

func currentFrame(state behaviorState, frame int) int {
	switch state {
	case stateIdle:
		return frameFromSeq(idleFrameSeq, frame, 5)
	case stateWalk, stateForage, stateCarry:
		return frameFromSeq(walkFrameSeq, frame, 2)
	case stateScurry:
		return frameFromSeq(walkFrameSeq, frame, 1)
	case stateWheel:
		return frameFromSeq(wheelRunFrameSeq, frame, 1)
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
	if a.temporarilyHidden {
		return
	}
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
	if a.temporarilyHidden {
		return
	}
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
	overlay := a.overlayRect()
	sceneX := screenX - int(overlay.Left)
	sceneY := screenY - int(overlay.Top)
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

func (a *petApp) updateHoverName() {
	if a.temporarilyHidden {
		a.hoverPet = -1
		a.hideNameWindow()
		return
	}
	if !a.nameLabels {
		a.hoverPet = -1
		a.hideNameWindow()
		return
	}
	var pt win.POINT
	if !win.GetCursorPos(&pt) {
		a.hideNameWindow()
		return
	}
	index := a.petAtScreenPoint(int(pt.X), int(pt.Y))
	if index < 0 || index >= len(a.pets) {
		a.hoverPet = -1
		a.hideNameWindow()
		return
	}
	a.hoverPet = index
	a.showNameWindow(index)
}

func (a *petApp) showNameWindow(index int) {
	if a.temporarilyHidden {
		a.hideNameWindow()
		return
	}
	if index < 0 || index >= len(a.pets) {
		a.hideNameWindow()
		return
	}
	name := a.petDisplayName(index)
	if name == "" {
		a.hideNameWindow()
		return
	}
	display := a.selectedDisplayArea()
	overlay := a.overlayRectFor(display.Work, display.Screen)
	screen := display.Screen
	p := a.pets[index]
	runes := []rune(name)
	w := clamp(34+len(runes)*12, 72, 220)
	h := 30
	baseY := sceneH - spriteH - p.laneOffset
	x := int(overlay.Left) + p.x + spriteW/2 - w/2
	y := int(overlay.Top) + baseY - h - 8
	x = clamp(x, int(overlay.Left), int(overlay.Right)-w)
	y = clamp(y, int(screen.Top), int(screen.Bottom)-h)
	if a.nameHwnd == 0 {
		a.nameHwnd = win.CreateWindowEx(
			win.WS_EX_TOOLWINDOW|win.WS_EX_TOPMOST|win.WS_EX_NOACTIVATE|win.WS_EX_TRANSPARENT,
			syscall.StringToUTF16Ptr(windowClass),
			syscall.StringToUTF16Ptr(name),
			win.WS_POPUP,
			int32(x), int32(y), int32(w), int32(h),
			a.hwnd, 0, a.hinst, nil,
		)
	}
	if a.nameHwnd == 0 {
		return
	}
	if a.nameText != name {
		a.nameText = name
		setWindowText(a.nameHwnd, name)
		win.InvalidateRect(a.nameHwnd, nil, true)
	}
	win.SetWindowPos(a.nameHwnd, win.HWND_TOPMOST, int32(x), int32(y), int32(w), int32(h), win.SWP_NOACTIVATE|win.SWP_SHOWWINDOW)
}

func (a *petApp) hideNameWindow() {
	if a.nameHwnd != 0 {
		win.ShowWindow(a.nameHwnd, win.SW_HIDE)
	}
}

func (a *petApp) nameWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_PAINT:
		a.paintNameWindow(hwnd)
		return 0
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_NCHITTEST:
		return ^uintptr(0)
	case win.WM_DESTROY:
		if hwnd == a.nameHwnd {
			a.nameHwnd = 0
			a.nameText = ""
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (a *petApp) paintNameWindow(hwnd win.HWND) {
	a.ensureSettingsFonts()
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	if hdc == 0 {
		return
	}
	defer win.EndPaint(hwnd, &ps)
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)
	drawRectFill(hdc, rect, rgb(255, 255, 248))
	body := win.RECT{Left: 1, Top: 1, Right: rect.Right - 1, Bottom: rect.Bottom - 1}
	drawRoundFill(hdc, body, rgb(255, 255, 248), 14)
	drawRoundOutline(hdc, body, rgb(64, 91, 72), 14)
	textRect := win.RECT{Left: 10, Top: 1, Right: rect.Right - 10, Bottom: rect.Bottom - 1}
	drawTextLine(hdc, a.nameText, textRect, a.settingsSmallFont, rgb(27, 36, 32), win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
}

func (a *petApp) syncScene(bounds win.RECT) {
	a.sceneW = max(1, int(bounds.Right-bounds.Left))
	nextWheelX := a.sceneW * 2 / 3
	a.wheelX = clamp(nextWheelX, wheelSize/2+24, max(wheelSize/2+24, a.sceneW-wheelSize/2-24))
}

func (a *petApp) overlayRect() win.RECT {
	display := a.selectedDisplayArea()
	return a.overlayRectFor(display.Work, display.Screen)
}

func (a *petApp) normalizedDisplaySelection(count int) (displayScope, int, int) {
	if count <= 0 {
		return displayScopeSingle, 0, 0
	}
	scope := normalizeDisplayScope(int(a.displayScope))
	start := normalizeDisplayIndex(a.displayIndex, count)
	end := normalizeDisplayIndex(a.displaySpanEnd, count)
	if scope == displayScopeSpan && count > 1 {
		start, end = normalizeDisplaySpan(start, end, count)
		if start == end {
			if end < count-1 {
				end++
			} else {
				start--
			}
		}
		return displayScopeSpan, start, end
	}
	return displayScopeSingle, start, start
}

func (a *petApp) normalizeDisplaySelection(count int) {
	scope, start, end := a.normalizedDisplaySelection(count)
	a.displayScope = scope
	a.displayIndex = start
	a.displaySpanEnd = end
}

func (a *petApp) overlayRectFor(work, screen win.RECT) win.RECT {
	mode := normalizeOverlayPositionMode(int(a.positionMode))
	offset := normalizeOverlayOffset(a.overlayOffsetY)
	base := work
	if mode == positionScreenBottom {
		base = screen
	}
	if base.Right <= base.Left {
		base.Left = screen.Left
		base.Right = screen.Right
	}
	if base.Bottom <= base.Top {
		base.Top = screen.Top
		base.Bottom = screen.Bottom
	}
	base = a.applyWalkRange(base)
	y := int(base.Bottom) - sceneH + offset
	minY := int(screen.Top)
	maxY := max(minY, int(screen.Bottom)-sceneH)
	y = clamp(y, minY, maxY)
	return win.RECT{
		Left:   base.Left,
		Top:    int32(y),
		Right:  base.Right,
		Bottom: int32(y + sceneH),
	}
}

func (a *petApp) applyWalkRange(base win.RECT) win.RECT {
	width := int(base.Right - base.Left)
	if width <= 0 {
		return base
	}
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	left := int(base.Left) + width*start/100
	right := int(base.Left) + width*end/100
	if right-left < spriteW {
		mid := (left + right) / 2
		left = mid - spriteW/2
		right = left + spriteW
	}
	left = clamp(left, int(base.Left), max(int(base.Left), int(base.Right)-spriteW))
	right = clamp(right, left+spriteW, int(base.Right))
	base.Left = int32(left)
	base.Right = int32(right)
	return base
}

func (a *petApp) resetOverlayPlacement() {
	a.positionMode = positionTaskbarEdge
	a.overlayOffsetY = defaultOverlayOffsetY
	a.displayScope = displayScopeSingle
	a.displayIndex = 0
	a.displaySpanEnd = 0
	a.walkRangeStart = defaultWalkRangeStart
	a.walkRangeEnd = defaultWalkRangeEnd
}

func (a *petApp) adjustOverlayOffset(delta int) {
	a.overlayOffsetY = normalizeOverlayOffset(a.overlayOffsetY + delta)
}

func (a *petApp) setWalkRange(start, end int) {
	a.walkRangeStart, a.walkRangeEnd = normalizeWalkRange(start, end)
	overlay := a.overlayRect()
	a.syncScene(overlay)
	a.arrangePetsForOverlay(overlay)
}

func (a *petApp) adjustWalkRangeWidth(delta int) {
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	a.setWalkRange(start-delta, end+delta)
}

func (a *petApp) shiftWalkRange(delta int) {
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	span := end - start
	start = clamp(start+delta, 0, 100-span)
	a.setWalkRange(start, start+span)
}

func (a *petApp) clampPetsToScene() {
	limit := max(0, a.sceneW-spriteW)
	for i := range a.pets {
		a.pets[i].x = clamp(a.pets[i].x, 0, limit)
	}
	for i := range a.forage {
		if a.forage[i].active {
			a.forage[i].x = clamp(a.forage[i].x, 24, max(24, a.sceneW-24))
		}
	}
}

func (a *petApp) adjustDisplayIndex(delta int) {
	areas := displayAreaForScope(a.displayScope)
	count := len(areas)
	if count <= 0 {
		a.displayIndex = 0
		a.displaySpanEnd = 0
		return
	}
	scope, start, end := a.normalizedDisplaySelection(count)
	if scope == displayScopeSpan {
		width := end - start
		nextStart := clamp(start+delta, 0, max(0, count-1-width))
		a.displayIndex = nextStart
		a.displaySpanEnd = nextStart + width
	} else {
		a.displayIndex = (start + delta + count) % count
		a.displaySpanEnd = a.displayIndex
	}
	a.resetPosition()
}

func (a *petApp) setDisplayScope(scope displayScope) {
	targetScope := normalizeDisplayScope(int(scope))
	wasSpan := normalizeDisplayScope(int(a.displayScope)) == displayScopeSpan
	if targetScope == displayScopeSpan {
		singleAreas := monitorAreas()
		spanAreas := monitorAreasByPosition()
		count := len(spanAreas)
		if count > 1 {
			start := normalizeDisplayIndex(a.displayIndex, count)
			if normalizeDisplayScope(int(a.displayScope)) != displayScopeSpan && len(singleAreas) > 0 {
				start = findDisplayAreaIndex(spanAreas, singleAreas[normalizeDisplayIndex(a.displayIndex, len(singleAreas))])
			}
			end := a.displaySpanEnd
			if normalizeDisplayScope(int(a.displayScope)) != displayScopeSpan || end == start {
				end = min(start+1, count-1)
				if end == start {
					start = max(0, start-1)
				}
			}
			a.displayScope = displayScopeSpan
			a.displayIndex, a.displaySpanEnd = normalizeDisplaySpan(start, end, count)
			if !wasSpan {
				a.walkRangeStart = defaultWalkRangeStart
				a.walkRangeEnd = defaultWalkRangeEnd
			}
			a.resetPosition()
			return
		}
	}
	singleAreas := monitorAreas()
	spanAreas := monitorAreasByPosition()
	count := len(singleAreas)
	index := normalizeDisplayIndex(a.displayIndex, count)
	if normalizeDisplayScope(int(a.displayScope)) == displayScopeSpan && len(spanAreas) > 0 {
		_, start, _ := a.normalizedDisplaySelection(len(spanAreas))
		index = findDisplayAreaIndex(singleAreas, spanAreas[start])
	}
	a.displayScope = displayScopeSingle
	a.displayIndex = index
	a.displaySpanEnd = a.displayIndex
	a.resetPosition()
}

func (a *petApp) adjustDisplaySpan(delta int) {
	areas := monitorAreasByPosition()
	count := len(areas)
	if count <= 1 {
		a.setDisplayScope(displayScopeSingle)
		return
	}
	if normalizeDisplayScope(int(a.displayScope)) != displayScopeSpan {
		a.setDisplayScope(displayScopeSpan)
	}
	_, start, end := a.normalizedDisplaySelection(count)
	if delta > 0 {
		if end < count-1 {
			end++
		} else if start > 0 {
			start--
		}
	} else if delta < 0 && end-start > 1 {
		end--
	}
	a.displayScope = displayScopeSpan
	a.displayIndex, a.displaySpanEnd = normalizeDisplaySpan(start, end, count)
	a.resetPosition()
}

func (a *petApp) laneOffsetFor(index int) int {
	if normalizePetLaneMode(int(a.laneMode)) == laneAligned {
		return 0
	}
	return (index % 3) * 5
}

func (a *petApp) applyLaneOffsets() {
	for i := range a.pets {
		a.pets[i].laneOffset = a.laneOffsetFor(i)
	}
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
		a.pets[i].laneOffset = a.laneOffsetFor(i)
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
	a.arrangePetsAcrossScene()
}

type sceneSegment struct {
	Left  int
	Right int
}

func (a *petApp) arrangePetsAcrossScene() {
	a.arrangePetsInSegments(nil)
}

func (a *petApp) arrangePetsForOverlay(overlay win.RECT) {
	a.arrangePetsInSegments(a.sceneSegmentsForOverlay(overlay))
}

func (a *petApp) arrangePetsInSegments(segments []sceneSegment) {
	positions := petScenePositions(a.sceneW, len(a.pets), segments)
	for i, x := range positions {
		p := &a.pets[i]
		p.x = x
		dir := 1
		if a.bidirectional && i%2 == 1 {
			dir = -1
		}
		p.dir = dir
		p.nextDir = dir
		if p.state == stateTurn {
			p.state = stateWalk
			p.moveSpeed = max(1, a.speed-1)
			p.stateTicks = 24
		}
	}
}

func (a *petApp) sceneSegmentsForOverlay(overlay win.RECT) []sceneSegment {
	areas := displayAreaForScope(a.displayScope)
	if len(areas) == 0 || overlay.Right <= overlay.Left {
		return nil
	}
	scope, start, end := a.normalizedDisplaySelection(len(areas))
	if scope != displayScopeSpan {
		end = start
	}
	mode := normalizeOverlayPositionMode(int(a.positionMode))
	segments := make([]sceneSegment, 0, end-start+1)
	for _, area := range areas[start : end+1] {
		base := area.Work
		if mode == positionScreenBottom {
			base = area.Screen
		}
		left := max(int(base.Left), int(overlay.Left))
		right := min(int(base.Right), int(overlay.Right))
		if right-left >= spriteW {
			segments = append(segments, sceneSegment{
				Left:  left - int(overlay.Left),
				Right: right - int(overlay.Left),
			})
		}
	}
	return mergeSceneSegments(segments)
}

func mergeSceneSegments(segments []sceneSegment) []sceneSegment {
	if len(segments) <= 1 {
		return segments
	}
	sort.SliceStable(segments, func(i, j int) bool {
		if segments[i].Left != segments[j].Left {
			return segments[i].Left < segments[j].Left
		}
		return segments[i].Right < segments[j].Right
	})
	merged := segments[:0]
	for _, segment := range segments {
		if segment.Right <= segment.Left {
			continue
		}
		if len(merged) == 0 || segment.Left >= merged[len(merged)-1].Right {
			merged = append(merged, segment)
			continue
		}
		if segment.Right > merged[len(merged)-1].Right {
			merged[len(merged)-1].Right = segment.Right
		}
	}
	return merged
}

func petScenePositions(sceneW, count int, segments []sceneSegment) []int {
	if count <= 0 || sceneW <= 0 {
		return nil
	}
	segments = normalizeSceneSegments(sceneW, segments)
	if len(segments) == 0 {
		segments = []sceneSegment{{Left: 0, Right: sceneW}}
	}
	allocations := allocatePetsToSegments(count, segments)
	positions := make([]int, 0, count)
	for i, segment := range segments {
		n := allocations[i]
		if n <= 0 {
			continue
		}
		leftLimit := segment.Left
		rightLimit := segment.Right - spriteW
		if rightLimit < leftLimit {
			rightLimit = leftLimit
		}
		margin := min(24, max(0, (rightLimit-leftLimit)/4))
		left := leftLimit + margin
		right := rightLimit - margin
		if right < left {
			left = leftLimit
			right = rightLimit
		}
		for j := 0; j < n; j++ {
			x := (left + right) / 2
			if n > 1 {
				x = left + (right-left)*j/(n-1)
			}
			positions = append(positions, clamp(x, 0, max(0, sceneW-spriteW)))
		}
	}
	for len(positions) < count {
		positions = append(positions, clamp(sceneW/2-spriteW/2, 0, max(0, sceneW-spriteW)))
	}
	return positions[:count]
}

func normalizeSceneSegments(sceneW int, segments []sceneSegment) []sceneSegment {
	out := make([]sceneSegment, 0, len(segments))
	for _, segment := range segments {
		left := clamp(segment.Left, 0, sceneW)
		right := clamp(segment.Right, 0, sceneW)
		if right-left >= spriteW || (sceneW < spriteW && right > left) {
			out = append(out, sceneSegment{Left: left, Right: right})
		}
	}
	return mergeSceneSegments(out)
}

func allocatePetsToSegments(count int, segments []sceneSegment) []int {
	allocations := make([]int, len(segments))
	if count <= 0 || len(segments) == 0 {
		return allocations
	}
	type remainder struct {
		index int
		value float64
	}
	totalWidth := 0
	for _, segment := range segments {
		totalWidth += max(0, segment.Right-segment.Left)
	}
	if totalWidth <= 0 {
		allocations[0] = count
		return allocations
	}
	remaining := count
	if count >= len(segments) {
		for i := range allocations {
			allocations[i] = 1
		}
		remaining -= len(segments)
	}
	remainders := make([]remainder, 0, len(segments))
	assigned := 0
	for i, segment := range segments {
		share := float64(max(0, segment.Right-segment.Left)) * float64(remaining) / float64(totalWidth)
		whole := int(math.Floor(share))
		allocations[i] += whole
		assigned += whole
		remainders = append(remainders, remainder{index: i, value: share - float64(whole)})
	}
	sort.SliceStable(remainders, func(i, j int) bool {
		if remainders[i].value != remainders[j].value {
			return remainders[i].value > remainders[j].value
		}
		return remainders[i].index < remainders[j].index
	})
	for i := 0; i < remaining-assigned; i++ {
		allocations[remainders[i%len(remainders)].index]++
	}
	return allocations
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
		laneOffset: a.laneOffsetFor(index),
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
	style := uint32(win.WS_POPUP | win.WS_VISIBLE | win.WS_CLIPCHILDREN)
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
	a.settingsTooltip = 0
	a.settingsTipText = nil
	a.settingsHoverTip = ""
	a.ensureSettingsFonts()

	a.createButton(hwnd, ctrlTopClose, "x", 716, 18, 28, 28, 0)
	a.createButton(hwnd, ctrlTabHome, a.txt("tabHome"), 24, 122, 154, 38, win.WS_GROUP)
	a.createButton(hwnd, ctrlTabAnimals, a.txt("tabAnimals"), 24, 170, 154, 38, 0)
	a.createButton(hwnd, ctrlTabMotion, a.txt("tabMotion"), 24, 218, 154, 38, 0)
	a.createButton(hwnd, ctrlTabDisplay, a.txt("tabDisplay"), 24, 266, 154, 38, 0)
	a.createButton(hwnd, ctrlTabUpdates, a.txt("tabUpdates"), 24, 314, 154, 38, 0)

	if a.settingsTab == tabHome {
		a.createButton(hwnd, ctrlHomeAnimals, a.settingsButtonLabel(ctrlHomeAnimals), 600, 166, 88, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlHomeMotion, a.settingsButtonLabel(ctrlHomeMotion), 600, 280, 88, 30, 0)
		a.createButton(hwnd, ctrlHomeDisplay, a.settingsButtonLabel(ctrlHomeDisplay), 600, 394, 88, 30, 0)
		a.createButton(hwnd, ctrlHomeUpdates, a.settingsButtonLabel(ctrlHomeUpdates), 600, 508, 88, 30, 0)
	} else if a.settingsTab == tabAnimals {
		a.createButton(hwnd, ctrlPetMinus, "-", 408, 150, 42, 32, 0)
		a.createButton(hwnd, ctrlPetPlus, "+", 506, 150, 42, 32, 0)

		a.createButton(hwnd, ctrlCoatFixed, a.txt("coatFixed"), 250, 250, 110, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlCoatSelected, a.txt("coatSelected"), 374, 250, 130, 30, 0)
		a.createButton(hwnd, ctrlCoatRandom, a.txt("coatRandom"), 518, 250, 110, 30, 0)

		a.createButton(hwnd, ctrlVariantCombo, "", 360, 298, 318, 34, 0)
		a.createButton(hwnd, ctrlNameLabels, "", 548, 344, 132, 28, 0)
		if a.nameLabels {
			for i := 0; i < a.petCount; i++ {
				_, nameRect := settingsPetNameRects(i)
				a.createButton(hwnd, ctrlPetNameBase+int32(i), "", nameRect.Left, nameRect.Top, nameRect.Right-nameRect.Left, nameRect.Bottom-nameRect.Top, 0)
				if a.coatMode == coatSelected {
					_, buttonRect := settingsPetVariantRects(i)
					a.createButton(hwnd, ctrlPetVariantBase+int32(i), "", buttonRect.Left, buttonRect.Top, buttonRect.Right-buttonRect.Left, buttonRect.Bottom-buttonRect.Top, 0)
				}
			}
		}
	} else if a.settingsTab == tabMotion {
		a.createButton(hwnd, ctrlModeKeyboard, a.txt("modeKeyboard"), 250, 164, 210, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlModeRandom, a.txt("modeRandom"), 478, 164, 210, 32, 0)

		a.createButton(hwnd, ctrlSpeedSlow, a.txt("speedSlow"), 250, 270, 118, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlSpeedNormal, a.txt("speedNormal"), 384, 270, 118, 32, 0)
		a.createButton(hwnd, ctrlSpeedFast, a.txt("speedFast"), 518, 270, 118, 32, 0)

		a.createButton(hwnd, ctrlTypingWheel, a.txt("typingWheel"), 250, 378, 210, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlBidirectional, a.txt("naturalTurns"), 478, 378, 210, 32, 0)
	} else if a.settingsTab == tabDisplay {
		a.createButton(hwnd, ctrlDisplaySingle, a.settingsButtonLabel(ctrlDisplaySingle), 250, 154, 82, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlDisplaySpan, a.settingsButtonLabel(ctrlDisplaySpan), 340, 154, 96, 30, 0)
		a.createButton(hwnd, ctrlDisplaySpanLess, a.settingsButtonLabel(ctrlDisplaySpanLess), 444, 154, 74, 30, 0)
		a.createButton(hwnd, ctrlDisplaySpanMore, a.settingsButtonLabel(ctrlDisplaySpanMore), 526, 154, 74, 30, 0)
		a.createButton(hwnd, ctrlDisplayPrev, a.settingsButtonLabel(ctrlDisplayPrev), 608, 154, 38, 30, 0)
		a.createButton(hwnd, ctrlDisplayNext, a.settingsButtonLabel(ctrlDisplayNext), 650, 154, 38, 30, 0)

		a.createScrollBar(hwnd, ctrlRangeStartScroll, 322, 258, 288, 18)
		a.createScrollBar(hwnd, ctrlRangeEndScroll, 322, 292, 288, 18)
		a.createButton(hwnd, ctrlRangeFull, a.settingsButtonLabel(ctrlRangeFull), 250, 326, 74, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlRangeNarrow, a.settingsButtonLabel(ctrlRangeNarrow), 334, 326, 74, 30, 0)
		a.createButton(hwnd, ctrlRangeWide, a.settingsButtonLabel(ctrlRangeWide), 418, 326, 74, 30, 0)
		a.createButton(hwnd, ctrlRangeLeft, a.settingsButtonLabel(ctrlRangeLeft), 502, 326, 74, 30, 0)
		a.createButton(hwnd, ctrlRangeRight, a.settingsButtonLabel(ctrlRangeRight), 586, 326, 74, 30, 0)

		a.createButton(hwnd, ctrlPositionTaskbar, a.settingsButtonLabel(ctrlPositionTaskbar), 250, 426, 134, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlPositionBottom, a.settingsButtonLabel(ctrlPositionBottom), 392, 426, 160, 30, 0)
		a.createButton(hwnd, ctrlOffsetUp, a.settingsButtonLabel(ctrlOffsetUp), 560, 426, 62, 30, 0)
		a.createButton(hwnd, ctrlOffsetDown, a.settingsButtonLabel(ctrlOffsetDown), 628, 426, 62, 30, 0)

		a.createButton(hwnd, ctrlLaneStaggered, a.settingsButtonLabel(ctrlLaneStaggered), 250, 522, 180, 30, win.WS_GROUP)
		a.createButton(hwnd, ctrlLaneAligned, a.settingsButtonLabel(ctrlLaneAligned), 446, 522, 180, 30, 0)
	} else {
		a.createButton(hwnd, ctrlUpdateCheck, a.settingsButtonLabel(ctrlUpdateCheck), 250, 428, 190, 32, win.WS_GROUP)
		a.createButton(hwnd, ctrlUpdateInstall, a.settingsButtonLabel(ctrlUpdateInstall), 456, 428, 232, 32, 0)
	}

	a.createButton(hwnd, ctrlLanguageCombo, "", 322, 574, 180, 34, 0)
	a.createButton(hwnd, ctrlReset, a.txt("reset"), 534, 576, 100, 32, 0)
	a.createButton(hwnd, ctrlClose, a.txt("close"), 646, 576, 78, 32, 0)
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

	drawRoundFill(hdc, win.RECT{Left: 226, Top: 96, Right: 736, Bottom: 566}, rgb(255, 255, 251), 18)
	drawRoundFill(hdc, win.RECT{Left: 226, Top: 574, Right: 736, Bottom: 612}, rgb(255, 255, 251), 14)

	if a.settingsTab == tabHome {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 118, Right: 708, Bottom: 214}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 232, Right: 708, Bottom: 328}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 346, Right: 708, Bottom: 442}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 460, Right: 708, Bottom: 558}, rgb(238, 242, 237), 14)
	} else if a.settingsTab == tabAnimals {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 142, Right: 708, Bottom: 192}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 238, Right: 708, Bottom: 292}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 338, Right: 708, Bottom: 502}, rgb(238, 242, 237), 14)
	} else if a.settingsTab == tabMotion {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 150, Right: 708, Bottom: 208}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 256, Right: 708, Bottom: 314}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 364, Right: 708, Bottom: 424}, rgb(238, 242, 237), 14)
	} else if a.settingsTab == tabDisplay {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 118, Right: 708, Bottom: 214}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 214, Right: 708, Bottom: 372}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 390, Right: 708, Bottom: 466}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 486, Right: 708, Bottom: 566}, rgb(238, 242, 237), 14)
	} else {
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 118, Right: 708, Bottom: 236}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 254, Right: 708, Bottom: 370}, rgb(238, 242, 237), 14)
		drawRoundFill(hdc, win.RECT{Left: 238, Top: 390, Right: 708, Bottom: 500}, rgb(238, 242, 237), 14)
	}

	drawTextLine(hdc, a.txt("settingsHeader"), win.RECT{Left: 24, Top: 28, Right: 178, Bottom: 56}, a.settingsTitleFont, rgb(245, 250, 244), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
	drawTextLine(hdc, a.txt("settingsLead"), win.RECT{Left: 24, Top: 58, Right: 178, Bottom: 96}, a.settingsSmallFont, rgb(183, 207, 195), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX)
	drawRoundFill(hdc, win.RECT{Left: 24, Top: 474, Right: 178, Bottom: 520}, rgb(35, 62, 52), 14)
	drawTextLine(hdc, a.settingsSidebarStatus(), win.RECT{Left: 38, Top: 486, Right: 164, Bottom: 508}, a.settingsSmallFont, rgb(231, 241, 233), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

	drawTextLine(hdc, a.settingsPageTitle(), win.RECT{Left: 226, Top: 28, Right: 620, Bottom: 56}, a.settingsTitleFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
	leadRect := win.RECT{Left: 226, Top: 56, Right: 700, Bottom: 90}
	leadFlags := uint32(win.DT_LEFT | win.DT_VCENTER | win.DT_SINGLELINE | win.DT_NOPREFIX | win.DT_END_ELLIPSIS)
	if a.settingsHoverTip != "" {
		leadFlags = win.DT_LEFT | win.DT_WORDBREAK | win.DT_NOPREFIX | win.DT_END_ELLIPSIS
	}
	drawTextLine(hdc, a.settingsPageLead(), leadRect, a.settingsSmallFont, rgb(91, 104, 96), leadFlags)

	labelColor := rgb(50, 61, 55)
	if a.settingsTab == tabHome {
		drawTextLine(hdc, a.localText("デグー", "Pets"), win.RECT{Left: 246, Top: 126, Right: 390, Bottom: 150}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.settingsSidebarStatus(), win.RECT{Left: 250, Top: 158, Right: 584, Bottom: 184}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.homePetDetail(), win.RECT{Left: 250, Top: 188, Right: 584, Bottom: 206}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

		drawTextLine(hdc, a.localText("動き", "Motion"), win.RECT{Left: 246, Top: 240, Right: 390, Bottom: 264}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.homeMotionSummary(), win.RECT{Left: 250, Top: 272, Right: 584, Bottom: 298}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.homeMotionDetail(), win.RECT{Left: 250, Top: 302, Right: 584, Bottom: 320}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

		drawTextLine(hdc, a.localText("表示", "Display"), win.RECT{Left: 246, Top: 354, Right: 390, Bottom: 378}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.displaySummary(), win.RECT{Left: 250, Top: 386, Right: 584, Bottom: 412}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.homeDisplayDetail(), win.RECT{Left: 250, Top: 416, Right: 584, Bottom: 434}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

		drawTextLine(hdc, a.localText("更新", "Updates"), win.RECT{Left: 246, Top: 468, Right: 390, Bottom: 492}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.updateStatusSummary(), win.RECT{Left: 250, Top: 500, Right: 584, Bottom: 526}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.updatePackageSummary(), win.RECT{Left: 250, Top: 530, Right: 584, Bottom: 550}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
	} else if a.settingsTab == tabAnimals {
		drawTextLine(hdc, a.txt("animalSection"), win.RECT{Left: 246, Top: 116, Right: 470, Bottom: 140}, a.settingsFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("deguCount"), win.RECT{Left: 250, Top: 154, Right: 390, Bottom: 184}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, fmt.Sprintf("%d", a.petCount), win.RECT{Left: 464, Top: 154, Right: 498, Bottom: 184}, a.settingsFont, rgb(25, 49, 40), win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("coatMode"), win.RECT{Left: 250, Top: 214, Right: 420, Bottom: 238}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("coatColor"), win.RECT{Left: 250, Top: 302, Right: 352, Bottom: 328}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.petNameSectionLabel(), win.RECT{Left: 250, Top: 344, Right: 520, Bottom: 370}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		if a.nameLabels {
			for i := 0; i < a.petCount; i++ {
				numberRect, _ := settingsPetNameRects(i)
				drawTextLine(hdc, fmt.Sprintf("%d", i+1), numberRect, a.settingsSmallFont, rgb(69, 78, 72), win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
			}
		} else {
			drawTextLine(hdc, a.localText("オンにすると、名前の編集とカーソルホバー表示を使えます。", "Turn this on to edit names and show them on hover."), win.RECT{Left: 254, Top: 386, Right: 682, Bottom: 430}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX)
		}
	} else if a.settingsTab == tabMotion {
		drawTextLine(hdc, a.txt("mode"), win.RECT{Left: 246, Top: 126, Right: 400, Bottom: 150}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("speed"), win.RECT{Left: 246, Top: 232, Right: 400, Bottom: 256}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.txt("motion"), win.RECT{Left: 246, Top: 340, Right: 400, Bottom: 364}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
	} else if a.settingsTab == tabDisplay {
		drawTextLine(hdc, a.localText("表示する範囲", "Display scope"), win.RECT{Left: 246, Top: 126, Right: 400, Bottom: 150}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.displaySummary(), win.RECT{Left: 250, Top: 186, Right: 688, Bottom: 208}, a.settingsFont, rgb(27, 36, 32), win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

		drawTextLine(hdc, a.walkRangeSectionLabel(), win.RECT{Left: 246, Top: 222, Right: 390, Bottom: 244}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.walkRangeSummary(), win.RECT{Left: 394, Top: 222, Right: 688, Bottom: 244}, a.settingsSmallFont, rgb(91, 104, 96), win.DT_RIGHT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		a.drawWalkRangePreview(hdc)
		drawTextLine(hdc, a.localText("左端", "Left edge"), win.RECT{Left: 250, Top: 254, Right: 316, Bottom: 280}, a.settingsSmallFont, labelColor, win.DT_RIGHT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.localText("右端", "Right edge"), win.RECT{Left: 250, Top: 288, Right: 316, Bottom: 314}, a.settingsSmallFont, labelColor, win.DT_RIGHT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
		drawTextLine(hdc, fmt.Sprintf("%d%%", start), win.RECT{Left: 616, Top: 254, Right: 680, Bottom: 280}, a.settingsSmallFont, rgb(91, 104, 96), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, fmt.Sprintf("%d%%", end), win.RECT{Left: 616, Top: 288, Right: 680, Bottom: 314}, a.settingsSmallFont, rgb(91, 104, 96), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)

		drawTextLine(hdc, a.localText("表示位置", "Position"), win.RECT{Left: 246, Top: 398, Right: 380, Bottom: 420}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.positionSummary(), win.RECT{Left: 382, Top: 398, Right: 690, Bottom: 420}, a.settingsSmallFont, rgb(91, 104, 96), win.DT_RIGHT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.localText("個体の高さ", "Pet height"), win.RECT{Left: 246, Top: 494, Right: 380, Bottom: 516}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.laneSummary(), win.RECT{Left: 382, Top: 494, Right: 690, Bottom: 516}, a.settingsSmallFont, rgb(91, 104, 96), win.DT_RIGHT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
	} else {
		drawTextLine(hdc, a.localText("更新状態", "Update status"), win.RECT{Left: 246, Top: 126, Right: 400, Bottom: 150}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.updateStatusSummary(), win.RECT{Left: 250, Top: 158, Right: 688, Bottom: 184}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.updateStatusDetail(), win.RECT{Left: 250, Top: 188, Right: 688, Bottom: 226}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

		drawTextLine(hdc, a.localText("配布パッケージ", "Package"), win.RECT{Left: 246, Top: 262, Right: 400, Bottom: 286}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.updatePackageSummary(), win.RECT{Left: 250, Top: 294, Right: 688, Bottom: 324}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
		drawTextLine(hdc, a.updatePackageDetail(), win.RECT{Left: 250, Top: 326, Right: 688, Bottom: 360}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)

		drawTextLine(hdc, a.localText("操作", "Actions"), win.RECT{Left: 246, Top: 398, Right: 400, Bottom: 422}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
		drawTextLine(hdc, a.updateActionNote(), win.RECT{Left: 250, Top: 468, Right: 688, Bottom: 492}, a.settingsSmallFont, rgb(69, 78, 72), win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX|win.DT_END_ELLIPSIS)
	}
	drawTextLine(hdc, a.txt("language"), win.RECT{Left: 226, Top: 582, Right: 314, Bottom: 606}, a.settingsSmallFont, labelColor, win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
}

func (a *petApp) drawWalkRangePreview(hdc win.HDC) {
	track := win.RECT{Left: 250, Top: 246, Right: 688, Bottom: 252}
	drawRoundFill(hdc, track, rgb(215, 221, 213), 6)
	a.drawDisplaySpanMarkers(hdc, track)
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	width := int(track.Right - track.Left)
	selected := win.RECT{
		Left:   track.Left + int32(width*start/100),
		Top:    track.Top - 2,
		Right:  track.Left + int32(width*end/100),
		Bottom: track.Bottom + 2,
	}
	drawRoundFill(hdc, selected, rgb(70, 104, 87), 8)
	drawRoundOutline(hdc, win.RECT{Left: 246, Top: 242, Right: 692, Bottom: 256}, rgb(230, 233, 224), 10)
}

func (a *petApp) drawDisplaySpanMarkers(hdc win.HDC, track win.RECT) {
	areas := displayAreaForScope(a.displayScope)
	if len(areas) <= 1 {
		return
	}
	scope, start, end := a.normalizedDisplaySelection(len(areas))
	if scope != displayScopeSpan || start == end {
		return
	}
	selected := areas[start : end+1]
	combined := combineDisplayAreas(selected)
	totalW := max(1, int(combined.Screen.Right-combined.Screen.Left))
	trackW := max(1, int(track.Right-track.Left))
	for _, area := range selected[1:] {
		x := track.Left + int32(trackW*int(area.Screen.Left-combined.Screen.Left)/totalW)
		drawRectFill(hdc, win.RECT{Left: x - 1, Top: track.Top - 5, Right: x + 1, Bottom: track.Bottom + 5}, rgb(171, 181, 169))
	}
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
	a.addSettingsTooltip(hwnd, id, parent)
	return hwnd
}

func (a *petApp) createScrollBar(parent win.HWND, id int32, x, y, w, h int32) win.HWND {
	hwnd := win.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("SCROLLBAR"),
		nil,
		win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|sbsHorz,
		x, y, w, h,
		parent, win.HMENU(uintptr(id)), a.hinst, nil,
	)
	a.addSettingsTooltip(hwnd, id, parent)
	a.setScrollRangeAndPos(id, a.walkRangeScrollValue(id))
	return hwnd
}

func (a *petApp) ensureSettingsTooltip() win.HWND {
	if a.settingsTooltip != 0 {
		return a.settingsTooltip
	}
	if a.settingsHwnd == 0 {
		return 0
	}
	initControls := win.INITCOMMONCONTROLSEX{
		DwSize: uint32(unsafe.Sizeof(win.INITCOMMONCONTROLSEX{})),
		DwICC:  win.ICC_WIN95_CLASSES,
	}
	win.InitCommonControlsEx(&initControls)
	a.settingsTooltip = win.CreateWindowEx(
		win.WS_EX_TOPMOST,
		syscall.StringToUTF16Ptr("tooltips_class32"),
		nil,
		win.WS_POPUP|win.TTS_ALWAYSTIP|win.TTS_NOPREFIX,
		0, 0, 0, 0,
		a.settingsHwnd, 0, a.hinst, nil,
	)
	if a.settingsTooltip != 0 {
		win.SendMessage(a.settingsTooltip, win.TTM_SETMAXTIPWIDTH, 0, 360)
		win.SendMessage(a.settingsTooltip, win.TTM_ACTIVATE, 1, 0)
	}
	return a.settingsTooltip
}

func (a *petApp) destroySettingsTooltip() {
	if a.settingsTooltip != 0 {
		win.DestroyWindow(a.settingsTooltip)
		a.settingsTooltip = 0
	}
	a.settingsTipText = nil
}

func (a *petApp) addSettingsTooltip(control win.HWND, id int32, parent win.HWND) {
	if control == 0 || parent != a.settingsHwnd {
		return
	}
	text := a.settingsTooltipText(id)
	if text == "" {
		return
	}
	tooltip := a.ensureSettingsTooltip()
	if tooltip == 0 {
		return
	}
	buf := syscall.StringToUTF16(text)
	a.settingsTipText = append(a.settingsTipText, buf)
	info := win.TOOLINFO{
		CbSize:   uint32(unsafe.Sizeof(win.TOOLINFO{})),
		UFlags:   win.TTF_IDISHWND | win.TTF_SUBCLASS,
		Hwnd:     a.settingsHwnd,
		UId:      uintptr(control),
		LpszText: &a.settingsTipText[len(a.settingsTipText)-1][0],
	}
	win.SendMessage(tooltip, win.TTM_ADDTOOL, 0, uintptr(unsafe.Pointer(&info)))
}

func (a *petApp) createEdit(parent win.HWND, id int32, text string, x, y, w, h int32) win.HWND {
	hwnd := win.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("EDIT"),
		syscall.StringToUTF16Ptr(text),
		win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.WS_BORDER|win.ES_LEFT|win.ES_AUTOHSCROLL,
		x, y, w, h,
		parent, win.HMENU(uintptr(id)), a.hinst, nil,
	)
	a.setControlFont(hwnd, a.settingsFont)
	win.SendMessage(hwnd, win.EM_LIMITTEXT, 24, 0)
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
	case ctrlTabHome:
		return a.txt("tabHome")
	case ctrlTabAnimals:
		return a.txt("tabAnimals")
	case ctrlTabMotion:
		return a.txt("tabMotion")
	case ctrlTabDisplay:
		return a.txt("tabDisplay")
	case ctrlTabUpdates:
		return a.txt("tabUpdates")
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
	case ctrlPositionTaskbar:
		return a.localText("タスクバー基準", "Taskbar edge")
	case ctrlPositionBottom:
		return a.localText("画面の一番下", "Screen bottom")
	case ctrlOffsetUp:
		return a.localText("上へ", "Up")
	case ctrlOffsetDown:
		return a.localText("下へ", "Down")
	case ctrlLaneStaggered:
		return a.localText("少しずらす", "Natural stagger")
	case ctrlLaneAligned:
		return a.localText("横一列", "Same baseline")
	case ctrlDisplayPrev:
		return "<"
	case ctrlDisplayNext:
		return ">"
	case ctrlDisplaySingle:
		return a.localText("1画面", "One")
	case ctrlDisplaySpan:
		return a.localText("複数画面", "Multi")
	case ctrlDisplaySpanLess:
		return a.localText("短く", "Less")
	case ctrlDisplaySpanMore:
		return a.localText("広く", "More")
	case ctrlRangeFull:
		if normalizeDisplayScope(int(a.displayScope)) == displayScopeSpan {
			return a.localText("全画面", "All")
		}
		return a.localText("全幅", "Full")
	case ctrlRangeNarrow:
		return a.localText("狭く", "Narrow")
	case ctrlRangeWide:
		return a.localText("広く", "Wider")
	case ctrlRangeLeft:
		return a.localText("左へ", "Left")
	case ctrlRangeRight:
		return a.localText("右へ", "Right")
	case ctrlUpdateCheck:
		return a.localText("更新を確認", "Check for updates")
	case ctrlUpdateInstall:
		return a.localText("更新をインストール", "Install update")
	case ctrlHomeAnimals, ctrlHomeMotion, ctrlHomeDisplay, ctrlHomeUpdates:
		return a.localText("開く", "Open")
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
	case ctrlNameLabels:
		return a.localText("名前を付ける", "Use names")
	case ctrlRenameOK:
		return a.localText("保存", "Save")
	case ctrlRenameCancel:
		return a.localText("キャンセル", "Cancel")
	}
	if id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount {
		return a.variantLabel(a.settingsSelectVariant(id))
	}
	if id >= ctrlPetNameBase && id < ctrlPetNameBase+maxPetCount {
		return a.petDisplayName(int(id - ctrlPetNameBase))
	}
	return ""
}

func (a *petApp) settingsTooltipText(id int32) string {
	switch id {
	case ctrlTabHome:
		return a.localText("よく使う設定と現在の状態をまとめて確認します。", "Review the most-used settings and current status in one place.")
	case ctrlTabAnimals:
		return a.localText("出現数、毛色、名前を設定します。", "Set visible pets, coat colors, and names.")
	case ctrlTabMotion:
		return a.localText("歩き方、回し車、左右ターンを設定します。", "Set motion, typing wheel, and turn behavior.")
	case ctrlTabDisplay:
		return a.localText("1画面または複数画面、歩く画面、高さを設定します。", "Set one-display or multi-display scope, walking displays, and height.")
	case ctrlTabUpdates:
		return a.localText("現在のバージョン、最新リリース、更新の確認とインストールを表示します。", "Show the current version, latest release, update checks, and installation.")
	case ctrlPetMinus:
		return a.localText("出現数を1匹減らします。", "Decrease the visible pet count by one.")
	case ctrlPetPlus:
		return a.localText("出現数を1匹増やします。最大10匹です。", "Increase the visible pet count by one, up to 10.")
	case ctrlCoatFixed:
		return a.localText("全員を同じ毛色にします。", "Use one coat color for every pet.")
	case ctrlCoatSelected:
		return a.localText("1匹ずつ毛色を選びます。複数の色を同時に出せます。", "Choose a coat for each pet so multiple colors can appear together.")
	case ctrlCoatRandom:
		return a.localText("起動時や追加時に毛色をランダムに選びます。", "Pick coats randomly when pets appear.")
	case ctrlVariantCombo:
		return a.localText("固定モードで使う毛色を選びます。", "Choose the coat used in fixed mode.")
	case ctrlNameLabels:
		return a.localText("オンにすると名前を編集でき、デグーにカーソルを合わせると名前が出ます。", "Turn this on to edit names and show a name when the cursor hovers over a pet.")
	case ctrlModeKeyboard:
		return a.localText("キーボード入力に合わせて反応します。入力中だけ回し車も使えます。", "React to keyboard input. The typing wheel can run only while you type.")
	case ctrlModeRandom:
		return a.localText("入力に関係なく、歩く・止まる・食べるなどをランダムに行います。", "Move randomly regardless of typing: walk, pause, eat, and forage.")
	case ctrlSpeedSlow:
		return a.localText("ゆっくり移動します。", "Move slowly.")
	case ctrlSpeedNormal:
		return a.localText("標準の速さで移動します。", "Use the normal movement speed.")
	case ctrlSpeedFast:
		return a.localText("速めに移動します。", "Move faster.")
	case ctrlTypingWheel:
		return a.localText("キーボード入力中だけ、デグーが回し車で走ります。", "While you type, a degu runs in the wheel.")
	case ctrlBidirectional:
		return a.localText("左右どちらにも歩きます。オフにすると右向き中心になります。", "Allow walking both left and right. Turn off for mostly right-facing movement.")
	case ctrlPositionTaskbar:
		return a.localText("Windowsの作業領域を基準にします。通常はタスクバーのすぐ上です。", "Use the Windows work area, normally just above the taskbar.")
	case ctrlPositionBottom:
		return a.localText("画面の一番下を基準にします。左タスクバー環境でも横幅いっぱいに出せます。", "Use the physical screen bottom, useful when the taskbar is on the left.")
	case ctrlOffsetUp:
		return a.localText("表示位置を少し上げます。", "Move the overlay slightly upward.")
	case ctrlOffsetDown:
		return a.localText("表示位置を少し下げます。タスクバーから浮く場合に使います。", "Move the overlay slightly downward when pets float above the taskbar.")
	case ctrlLaneStaggered:
		return a.localText("個体ごとに0/5/10pxだけ高さをずらします。重なった時に見分けやすい自然な並びです。", "Offset pets by 0/5/10 px so overlapping pets are easier to distinguish.")
	case ctrlLaneAligned:
		return a.localText("全員を同じ接地ラインに揃えます。少し上に見える個体をなくします。", "Place every pet on the same baseline so none appears slightly higher.")
	case ctrlDisplayPrev:
		return a.localText("1画面モードでは前のディスプレイへ、複数画面モードでは対象範囲を左へ動かします。", "In one-display mode this switches to the previous display; in multi-display mode it moves the selected span left.")
	case ctrlDisplayNext:
		return a.localText("1画面モードでは次のディスプレイへ、複数画面モードでは対象範囲を右へ動かします。", "In one-display mode this switches to the next display; in multi-display mode it moves the selected span right.")
	case ctrlDisplaySingle:
		return a.localText("選択中の1画面だけにデグーを表示します。", "Show pets on one selected display only.")
	case ctrlDisplaySpan:
		return a.localText("複数のモニタを横断する1つの範囲として使います。切り替え時は歩く範囲を全画面に戻します。", "Use multiple monitors as one wide span. Switching to this resets walking range to all selected displays.")
	case ctrlDisplaySpanLess:
		return a.localText("複数画面の対象範囲を1枚ぶん狭くします。", "Shrink the multi-display span by one monitor.")
	case ctrlDisplaySpanMore:
		return a.localText("複数画面の対象範囲を1枚ぶん広げます。全モニタ表示にもできます。", "Expand the multi-display span by one monitor, up to all monitors.")
	case ctrlRangeFull:
		return a.localText("選択中の画面ぜんぶを歩く範囲に戻します。", "Use all selected displays for walking.")
	case ctrlRangeNarrow:
		return a.localText("歩く範囲を中央寄せで少し狭くします。", "Make the walking range narrower around the center.")
	case ctrlRangeWide:
		return a.localText("歩く範囲を中央寄せで少し広げます。", "Make the walking range wider around the center.")
	case ctrlRangeLeft:
		return a.localText("歩く範囲全体を左へ動かします。", "Move the whole walking range left.")
	case ctrlRangeRight:
		return a.localText("歩く範囲全体を右へ動かします。", "Move the whole walking range right.")
	case ctrlRangeStartScroll:
		return a.localText("タスクバーのここから歩く、という左端の位置を決めます。", "Set where the taskbar walking range starts.")
	case ctrlRangeEndScroll:
		return a.localText("タスクバーのここまで歩く、という右端の位置を決めます。", "Set where the taskbar walking range ends.")
	case ctrlUpdateCheck:
		return a.localText("GitHub Releases から最新バージョンを確認します。", "Check GitHub Releases for the latest version.")
	case ctrlUpdateInstall:
		return a.localText("見つかった更新をダウンロードし、Degu Desktop を再起動して適用します。", "Download the found update and restart Degu Desktop to apply it.")
	case ctrlHomeAnimals:
		return a.localText("デグーの数、毛色、名前の設定へ移動します。", "Open pet count, coat color, and name settings.")
	case ctrlHomeMotion:
		return a.localText("動き方、速度、回し車、左右ターンの設定へ移動します。", "Open motion, speed, wheel, and turn settings.")
	case ctrlHomeDisplay:
		return a.localText("表示画面、歩く範囲、表示高さの設定へ移動します。", "Open display, walking range, and height settings.")
	case ctrlHomeUpdates:
		return a.localText("バージョン確認とアップデート操作へ移動します。", "Open version check and update actions.")
	case ctrlLanguageCombo:
		return a.localText("設定画面の表示言語を切り替えます。", "Switch the settings language.")
	case ctrlReset:
		return a.localText("表示位置を初期値に戻し、デグーを画面内に並べ直します。", "Reset display position and place pets back inside the screen.")
	case ctrlClose, ctrlTopClose:
		return a.localText("設定画面を閉じます。", "Close the settings window.")
	}
	if id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount {
		return a.localText("このデグーの毛色を選びます。", "Choose this pet's coat.")
	}
	if id >= ctrlPetNameBase && id < ctrlPetNameBase+maxPetCount {
		return a.localText("このデグーの名前を編集します。", "Edit this pet's name.")
	}
	return ""
}

func (a *petApp) settingsButtonSelected(id int32) bool {
	switch id {
	case ctrlTabHome:
		return a.settingsTab == tabHome
	case ctrlTabAnimals:
		return a.settingsTab == tabAnimals
	case ctrlTabMotion:
		return a.settingsTab == tabMotion
	case ctrlTabDisplay:
		return a.settingsTab == tabDisplay
	case ctrlTabUpdates:
		return a.settingsTab == tabUpdates
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
	case ctrlPositionTaskbar:
		return normalizeOverlayPositionMode(int(a.positionMode)) == positionTaskbarEdge
	case ctrlPositionBottom:
		return normalizeOverlayPositionMode(int(a.positionMode)) == positionScreenBottom
	case ctrlDisplaySingle:
		return normalizeDisplayScope(int(a.displayScope)) == displayScopeSingle
	case ctrlDisplaySpan:
		return normalizeDisplayScope(int(a.displayScope)) == displayScopeSpan
	case ctrlLaneStaggered:
		return normalizePetLaneMode(int(a.laneMode)) == laneStaggered
	case ctrlLaneAligned:
		return normalizePetLaneMode(int(a.laneMode)) == laneAligned
	case ctrlNameLabels:
		return a.nameLabels
	}
	return false
}

func (a *petApp) settingsSelectButton(id int32) bool {
	return id == ctrlVariantCombo || id == ctrlLanguageCombo || (id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount)
}

func (a *petApp) settingsSidebarButton(id int32) bool {
	return id == ctrlTabHome || id == ctrlTabAnimals || id == ctrlTabMotion || id == ctrlTabDisplay || id == ctrlTabUpdates
}

func (a *petApp) settingsButtonBackplate(id int32) settingsRGB {
	switch id {
	case ctrlTabHome, ctrlTabAnimals, ctrlTabMotion, ctrlTabDisplay, ctrlTabUpdates:
		return rgb(22, 45, 38)
	case ctrlTopClose:
		return rgb(247, 248, 244)
	case ctrlCoatFixed, ctrlCoatSelected, ctrlCoatRandom,
		ctrlModeKeyboard, ctrlModeRandom,
		ctrlSpeedSlow, ctrlSpeedNormal, ctrlSpeedFast,
		ctrlTypingWheel, ctrlBidirectional,
		ctrlPositionTaskbar, ctrlPositionBottom, ctrlOffsetUp, ctrlOffsetDown,
		ctrlDisplaySingle, ctrlDisplaySpan, ctrlDisplaySpanLess, ctrlDisplaySpanMore,
		ctrlLaneStaggered, ctrlLaneAligned,
		ctrlDisplayPrev, ctrlDisplayNext,
		ctrlRangeFull, ctrlRangeNarrow, ctrlRangeWide, ctrlRangeLeft, ctrlRangeRight,
		ctrlUpdateCheck, ctrlUpdateInstall,
		ctrlHomeAnimals, ctrlHomeMotion, ctrlHomeDisplay, ctrlHomeUpdates,
		ctrlNameLabels, ctrlRenameOK, ctrlRenameCancel:
		return rgb(235, 232, 220)
	}
	if (id >= ctrlPetVariantBase && id < ctrlPetVariantBase+maxPetCount) ||
		(id >= ctrlPetNameBase && id < ctrlPetNameBase+maxPetCount) {
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
	if a.settingsTab == tabHome {
		return a.txt("homePageTitle")
	}
	if a.settingsTab == tabUpdates {
		return a.txt("updatesPageTitle")
	}
	if a.settingsTab == tabDisplay {
		return a.txt("displayPageTitle")
	}
	if a.settingsTab == tabMotion {
		return a.txt("motionPageTitle")
	}
	return a.txt("animalPageTitle")
}

func (a *petApp) settingsPageLead() string {
	if a.settingsHoverTip != "" {
		return a.settingsHoverTip
	}
	if a.settingsTab == tabHome {
		return a.txt("homePageLead")
	}
	if a.settingsTab == tabUpdates {
		return a.txt("updatesPageLead")
	}
	if a.settingsTab == tabDisplay {
		return a.txt("displayPageLead")
	}
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
	numberRect := win.RECT{Left: left, Top: top, Right: left + 24, Bottom: top + 26}
	buttonRect := win.RECT{Left: left + 110, Top: top, Right: left + 224, Bottom: top + 26}
	return numberRect, buttonRect
}

func settingsPetNameRects(index int) (win.RECT, win.RECT) {
	col := index / 5
	row := index % 5
	if col > 1 {
		col = 1
	}
	left := int32(250 + col*228)
	top := int32(370 + row*26)
	numberRect := win.RECT{Left: left, Top: top, Right: left + 24, Bottom: top + 26}
	editRect := win.RECT{Left: left + 30, Top: top + 1, Right: left + 104, Bottom: top + 25}
	return numberRect, editRect
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

func (a *petApp) petDisplayName(index int) string {
	if index < 0 || index >= maxPetCount {
		return ""
	}
	if name := sanitizePetName(a.petNames[index]); name != "" {
		return name
	}
	return a.localText(fmt.Sprintf("デグー%d", index+1), fmt.Sprintf("Degu %d", index+1))
}

func (a *petApp) petNameSectionLabel() string {
	if a.coatMode == coatSelected {
		return a.localText("名前 / 個別カラー", "Names / coat colors")
	}
	return a.localText("名前", "Names")
}

func (a *petApp) homePetDetail() string {
	coat := a.localText("固定カラー", "Fixed coat")
	switch a.coatMode {
	case coatSelected:
		coat = a.localText("個別カラー", "Per-pet coats")
	case coatRandom:
		coat = a.localText("ランダムカラー", "Random coats")
	}
	names := a.localText("名前表示オフ", "Names off")
	if a.nameLabels {
		names = a.localText("名前表示オン", "Names on")
	}
	return fmt.Sprintf("%s / %s", coat, names)
}

func (a *petApp) homeMotionSummary() string {
	mode := a.txt("modeKeyboard")
	if normalizeBehaviorMode(int(a.mode)) == modeRandom {
		mode = a.txt("modeRandom")
	}
	return fmt.Sprintf("%s / %s", mode, a.speedLabel())
}

func (a *petApp) homeMotionDetail() string {
	wheel := a.localText("回し車オフ", "Wheel off")
	if a.wheelEnabled {
		wheel = a.localText("回し車オン", "Wheel on")
	}
	turns := a.localText("右向き中心", "Mostly right-facing")
	if a.bidirectional {
		turns = a.localText("左右ターン自然", "Natural left/right turns")
	}
	return fmt.Sprintf("%s / %s", wheel, turns)
}

func (a *petApp) homeDisplayDetail() string {
	return fmt.Sprintf("%s / %s", a.walkRangeSummary(), a.positionSummary())
}

func (a *petApp) speedLabel() string {
	switch normalizeSpeed(a.speed) {
	case 2:
		return a.txt("speedSlow")
	case 5:
		return a.txt("speedFast")
	default:
		return a.txt("speedNormal")
	}
}

func (a *petApp) displaySummary() string {
	areas := displayAreaForScope(a.displayScope)
	if len(areas) == 0 {
		return a.localText("現在の画面", "Current display")
	}
	scope, start, end := a.normalizedDisplaySelection(len(areas))
	if scope == displayScopeSpan {
		selected := areas[start : end+1]
		area := combineDisplayAreas(selected)
		width := max(0, int(area.Screen.Right-area.Screen.Left))
		height := max(0, int(area.Screen.Bottom-area.Screen.Top))
		if len(selected) == len(areas) {
			return fmt.Sprintf("%s / %d%s  %dx%d", a.localText("全モニタ", "All monitors"), len(selected), a.localText("画面", " displays"), width, height)
		}
		return fmt.Sprintf("%s %d-%d/%d  %dx%d", a.localText("複数画面", "Displays"), start+1, end+1, len(areas), width, height)
	}
	area := areas[start]
	width := max(0, int(area.Screen.Right-area.Screen.Left))
	height := max(0, int(area.Screen.Bottom-area.Screen.Top))
	primary := ""
	if area.Primary {
		primary = a.localText(" / メイン", " / primary")
	}
	return fmt.Sprintf("%s %d/%d%s  %dx%d", a.localText("ディスプレイ", "Display"), start+1, len(areas), primary, width, height)
}

func (a *petApp) walkRangeSummary() string {
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	if summary := a.walkRangeSummaryForSegments(start, end, a.displaySegmentsForSummary()); summary != "" {
		return summary
	}
	if start == 0 && end == 100 {
		return a.localText("全幅", "Full width")
	}
	return fmt.Sprintf("%d%% - %d%%", start, end)
}

func (a *petApp) walkRangeSectionLabel() string {
	if normalizeDisplayScope(int(a.displayScope)) == displayScopeSpan {
		return a.localText("歩く画面", "Walking displays")
	}
	return a.localText("歩く範囲", "Walking range")
}

func (a *petApp) displaySegmentsForSummary() []sceneSegment {
	areas := displayAreaForScope(a.displayScope)
	if len(areas) == 0 {
		return nil
	}
	scope, start, end := a.normalizedDisplaySelection(len(areas))
	if scope != displayScopeSpan {
		end = start
	}
	selected := areas[start : end+1]
	combined := combineDisplayAreas(selected)
	base := combined.Work
	if normalizeOverlayPositionMode(int(a.positionMode)) == positionScreenBottom {
		base = combined.Screen
	}
	if base.Right <= base.Left {
		return nil
	}
	segments := make([]sceneSegment, 0, len(selected))
	for _, area := range selected {
		segmentBase := area.Work
		if normalizeOverlayPositionMode(int(a.positionMode)) == positionScreenBottom {
			segmentBase = area.Screen
		}
		left := max(int(segmentBase.Left), int(base.Left))
		right := min(int(segmentBase.Right), int(base.Right))
		if right-left >= spriteW {
			segments = append(segments, sceneSegment{
				Left:  left - int(base.Left),
				Right: right - int(base.Left),
			})
		}
	}
	return mergeSceneSegments(segments)
}

func (a *petApp) walkRangeSummaryForSegments(start, end int, segments []sceneSegment) string {
	start, end = normalizeWalkRange(start, end)
	segments = normalizeSceneSegments(segmentSpanWidth(segments), segments)
	if len(segments) == 0 {
		return ""
	}
	if start == 0 && end == 100 {
		if len(segments) == 1 {
			return a.localText("全幅", "Full width")
		}
		return a.localText("選択した画面ぜんぶ", "All selected displays")
	}
	totalLeft := segments[0].Left
	totalRight := segments[0].Right
	for _, segment := range segments[1:] {
		totalLeft = min(totalLeft, segment.Left)
		totalRight = max(totalRight, segment.Right)
	}
	totalW := max(1, totalRight-totalLeft)
	walkLeft := totalLeft + totalW*start/100
	walkRight := totalLeft + totalW*end/100
	covered := make([]int, 0, len(segments))
	fullCovered := true
	for i, segment := range segments {
		left := max(walkLeft, segment.Left)
		right := min(walkRight, segment.Right)
		if right-left <= 0 {
			continue
		}
		covered = append(covered, i+1)
		if left > segment.Left+2 || right < segment.Right-2 {
			fullCovered = false
		}
	}
	if len(covered) == 0 {
		return fmt.Sprintf("%d%% - %d%%", start, end)
	}
	if len(segments) == 1 {
		return a.localText("画面の一部", "Part of display")
	}
	if len(covered) == len(segments) && fullCovered {
		return a.localText("選択した画面ぜんぶ", "All selected displays")
	}
	first := covered[0]
	last := covered[len(covered)-1]
	if fullCovered {
		if first == last {
			return fmt.Sprintf("%s%d%s", a.localText("画面", "Display "), first, a.localText("だけ", " only"))
		}
		return fmt.Sprintf("%s%d-%d", a.localText("画面", "Displays "), first, last)
	}
	if first == last {
		if a.lang == langEnglish {
			return fmt.Sprintf("Part of display %d", first)
		}
		return fmt.Sprintf("画面%dの一部", first)
	}
	if a.lang == langEnglish {
		return fmt.Sprintf("Part of displays %d-%d", first, last)
	}
	return fmt.Sprintf("画面%d-%dの一部", first, last)
}

func segmentSpanWidth(segments []sceneSegment) int {
	if len(segments) == 0 {
		return 0
	}
	right := segments[0].Right
	for _, segment := range segments[1:] {
		right = max(right, segment.Right)
	}
	return right
}

func (a *petApp) updateSnapshot() (*githubRelease, string, bool, bool) {
	a.update.mu.Lock()
	defer a.update.mu.Unlock()
	var rel *githubRelease
	if a.update.latest != nil {
		cp := *a.update.latest
		cp.Assets = append([]githubReleaseAsset(nil), a.update.latest.Assets...)
		rel = &cp
	}
	return rel, a.update.lastError, a.update.checking.Load(), a.update.installing.Load()
}

func (a *petApp) updateStatusSummary() string {
	rel, lastErr, checking, installing := a.updateSnapshot()
	current := appVersion
	if current == "" {
		current = "dev"
	}
	switch {
	case installing:
		return a.localText("更新を適用中", "Installing update")
	case checking:
		return a.localText("最新バージョンを確認中", "Checking for updates")
	case lastErr != "":
		return a.localText("確認に失敗しました", "Update check failed")
	case rel == nil:
		return a.localText("まだ確認していません", "Not checked yet")
	case !isNewerVersion(rel.TagName, current):
		return a.localText("最新版です", "Up to date")
	case selectUpdateAsset(rel, runtime.GOARCH) == nil:
		return a.localText("更新はありますが、この環境用zipがありません", "Update found, but no package matches this PC")
	default:
		return a.localText(fmt.Sprintf("%s が利用できます", rel.TagName), fmt.Sprintf("%s is available", rel.TagName))
	}
}

func (a *petApp) updateStatusDetail() string {
	rel, lastErr, checking, installing := a.updateSnapshot()
	current := appVersion
	if current == "" {
		current = "dev"
	}
	switch {
	case installing:
		return a.localText("ダウンロード後に本体を終了し、新しいバージョンで再起動します。", "After download, the app will exit and restart with the new version.")
	case checking:
		return a.localText("GitHub Releases から最新リリース情報を取得しています。", "Fetching the latest release from GitHub Releases.")
	case lastErr != "":
		return lastErr
	case rel == nil:
		return a.localText(fmt.Sprintf("現在のバージョン: %s。必要な時に手動で確認できます。", current), fmt.Sprintf("Current version: %s. You can check manually when needed.", current))
	default:
		return a.localText(fmt.Sprintf("現在: %s / 最新: %s", current, rel.TagName), fmt.Sprintf("Current: %s / Latest: %s", current, rel.TagName))
	}
}

func (a *petApp) updatePackageSummary() string {
	rel, _, _, _ := a.updateSnapshot()
	want := updateAssetName(runtime.GOARCH)
	if rel == nil {
		return want
	}
	asset := selectUpdateAsset(rel, runtime.GOARCH)
	if asset == nil {
		return a.localText(want+" は未検出", want+" not found")
	}
	size := formatUpdateSize(asset.Size)
	if size == "" {
		return asset.Name
	}
	return fmt.Sprintf("%s / %s", asset.Name, size)
}

func (a *petApp) updatePackageDetail() string {
	rel, _, _, _ := a.updateSnapshot()
	want := updateAssetName(runtime.GOARCH)
	if rel == nil {
		return a.localText("このPCに合うWindows用zipを確認します。", "The updater checks for the Windows zip that matches this PC.")
	}
	asset := selectUpdateAsset(rel, runtime.GOARCH)
	if asset == nil {
		return a.localText(want+" がReleaseにないため、この画面からは適用できません。", want+" is missing from the release, so it cannot be installed from here.")
	}
	return a.localText("見つかったzipを一時フォルダへダウンロードして適用します。", "The found zip will be downloaded to a temporary folder and applied.")
}

func (a *petApp) updateActionNote() string {
	_, lastErr, checking, installing := a.updateSnapshot()
	switch {
	case installing:
		return a.localText("適用中です。完了するとDegu Desktopを再起動します。", "Installing. Degu Desktop will restart when it is ready.")
	case checking:
		return a.localText("確認中です。完了するとこの画面と通知に結果が出ます。", "Checking. Results will appear here and in a notification.")
	case lastErr != "":
		return a.localText("通信状態を確認して、もう一度「更新を確認」を押してください。", "Check your connection, then press Check for updates again.")
	case a.hasInstallableUpdate():
		return a.localText("インストールを押すと、ダウンロード後にアプリを再起動します。", "Press Install to download and restart the app.")
	default:
		return a.localText("まず「更新を確認」を押すと、利用可能な更新があるか分かります。", "Press Check for updates first to see whether an update is available.")
	}
}

func formatUpdateSize(size int64) string {
	if size <= 0 {
		return ""
	}
	const mb = 1024 * 1024
	if size >= mb {
		return fmt.Sprintf("%.1f MB", float64(size)/mb)
	}
	const kb = 1024
	if size >= kb {
		return fmt.Sprintf("%.0f KB", float64(size)/kb)
	}
	return fmt.Sprintf("%d B", size)
}

func (a *petApp) positionSummary() string {
	mode := a.localText("タスクバー基準", "Taskbar edge")
	if normalizeOverlayPositionMode(int(a.positionMode)) == positionScreenBottom {
		mode = a.localText("画面下基準", "Screen bottom")
	}
	offset := normalizeOverlayOffset(a.overlayOffsetY)
	if offset >= 0 {
		return fmt.Sprintf("%s / +%d px", mode, offset)
	}
	return fmt.Sprintf("%s / %d px", mode, offset)
}

func (a *petApp) laneSummary() string {
	if normalizePetLaneMode(int(a.laneMode)) == laneAligned {
		return a.localText("全員を同じ高さに揃える", "All pets use the same baseline")
	}
	return a.localText("少し高さをずらして重なりを軽減", "Small height offsets reduce overlap")
}

func (a *petApp) localText(ja, en string) string {
	if a.lang == langEnglish {
		return en
	}
	return ja
}

func sanitizePetName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	runes := []rune(name)
	if len(runes) > 24 {
		runes = runes[:24]
	}
	return string(runes)
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

func drawRoundOutline(hdc win.HDC, r win.RECT, c settingsRGB, radius int32) {
	lb := win.LOGBRUSH{LbStyle: win.BS_SOLID, LbColor: win.RGB(c.r, c.g, c.b)}
	pen := win.ExtCreatePen(win.PS_SOLID, 1, &lb, 0, nil)
	if pen == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(pen))
	oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
	oldPen := win.SelectObject(hdc, win.HGDIOBJ(pen))
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
	case win.WM_HSCROLL:
		if a.handleRangeScroll(getDlgCtrlID(win.HWND(lParam)), wParam) {
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
	case win.WM_SETCURSOR:
		a.updateSettingsHoverTip(win.HWND(wParam))
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
			a.destroySettingsTooltip()
			a.settingsHwnd = 0
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (a *petApp) updateSettingsHoverTip(hwnd win.HWND) {
	if a.settingsHwnd == 0 {
		return
	}
	tip := ""
	if hwnd != 0 && hwnd != a.settingsHwnd {
		tip = a.settingsTooltipText(getDlgCtrlID(hwnd))
	}
	if tip == a.settingsHoverTip {
		return
	}
	a.settingsHoverTip = tip
	win.InvalidateRect(a.settingsHwnd, nil, true)
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
	a.setButtonChecked(ctrlTabHome, a.settingsTab == tabHome)
	a.setButtonChecked(ctrlTabAnimals, a.settingsTab == tabAnimals)
	a.setButtonChecked(ctrlTabMotion, a.settingsTab == tabMotion)
	a.setButtonChecked(ctrlTabDisplay, a.settingsTab == tabDisplay)
	a.setButtonChecked(ctrlTabUpdates, a.settingsTab == tabUpdates)
	a.setButtonChecked(ctrlModeKeyboard, a.mode == modeKeyboard)
	a.setButtonChecked(ctrlModeRandom, a.mode == modeRandom)
	a.setButtonChecked(ctrlSpeedSlow, a.speed == 2)
	a.setButtonChecked(ctrlSpeedNormal, a.speed == 3)
	a.setButtonChecked(ctrlSpeedFast, a.speed == 5)
	a.setButtonChecked(ctrlTypingWheel, a.wheelEnabled)
	a.setButtonChecked(ctrlBidirectional, a.bidirectional)
	a.setButtonChecked(ctrlPositionTaskbar, a.positionMode == positionTaskbarEdge)
	a.setButtonChecked(ctrlPositionBottom, a.positionMode == positionScreenBottom)
	a.setButtonChecked(ctrlDisplaySingle, a.displayScope == displayScopeSingle)
	a.setButtonChecked(ctrlDisplaySpan, a.displayScope == displayScopeSpan)
	a.setButtonChecked(ctrlLaneStaggered, a.laneMode == laneStaggered)
	a.setButtonChecked(ctrlLaneAligned, a.laneMode == laneAligned)
	a.setButtonChecked(ctrlNameLabels, a.nameLabels)
	a.setButtonChecked(ctrlCoatFixed, a.coatMode == coatFixed)
	a.setButtonChecked(ctrlCoatSelected, a.coatMode == coatSelected)
	a.setButtonChecked(ctrlCoatRandom, a.coatMode == coatRandom)
	a.syncSelectButton(ctrlVariantCombo)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlVariantCombo), a.coatMode == coatFixed)
	for i := 0; i < a.petCount; i++ {
		a.syncSelectButton(ctrlPetNameBase + int32(i))
		a.syncSelectButton(ctrlPetVariantBase + int32(i))
	}
	a.syncSelectButton(ctrlLanguageCombo)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlPetMinus), a.petCount > 1)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlPetPlus), a.petCount < maxPetCount)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlOffsetUp), a.overlayOffsetY > minOverlayOffsetY)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlOffsetDown), a.overlayOffsetY < maxOverlayOffsetY)
	a.syncRangeScrollBars()
	monitorCount := len(monitorAreas())
	scope, spanStart, spanEnd := a.normalizedDisplaySelection(monitorCount)
	canMovePrev := monitorCount > 1
	canMoveNext := monitorCount > 1
	if scope == displayScopeSpan {
		canMovePrev = spanStart > 0
		canMoveNext = spanEnd < monitorCount-1
	}
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlDisplaySingle), monitorCount > 0)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlDisplaySpan), monitorCount > 1)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlDisplaySpanLess), scope == displayScopeSpan && spanEnd-spanStart > 1)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlDisplaySpanMore), scope == displayScopeSpan && monitorCount > 1 && spanEnd-spanStart+1 < monitorCount)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlDisplayPrev), canMovePrev)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlDisplayNext), canMoveNext)
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlRangeNarrow), end-start > minWalkRangeSpan)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlRangeWide), start > 0 || end < 100)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlRangeLeft), start > 0)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlRangeRight), end < 100)
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlUpdateCheck), !a.updateChecking())
	win.EnableWindow(win.GetDlgItem(a.settingsHwnd, ctrlUpdateInstall), a.hasInstallableUpdate() && !a.updateChecking())
	win.InvalidateRect(a.settingsHwnd, nil, true)
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

func (a *petApp) syncRangeScrollBars() {
	a.setScrollRangeAndPos(ctrlRangeStartScroll, a.walkRangeScrollValue(ctrlRangeStartScroll))
	a.setScrollRangeAndPos(ctrlRangeEndScroll, a.walkRangeScrollValue(ctrlRangeEndScroll))
}

func (a *petApp) setScrollRangeAndPos(id int32, pos int) {
	if a.settingsHwnd == 0 {
		return
	}
	h := win.GetDlgItem(a.settingsHwnd, id)
	if h == 0 {
		return
	}
	pos = clamp(pos, 0, 100)
	win.SendMessage(h, sbmSetRange, 0, 100)
	win.SendMessage(h, sbmSetPos, uintptr(pos), 1)
	win.InvalidateRect(h, nil, true)
}

func (a *petApp) walkRangeScrollValue(id int32) int {
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	if id == ctrlRangeEndScroll {
		return end
	}
	return start
}

func (a *petApp) handleRangeScroll(id int32, wParam uintptr) bool {
	if id != ctrlRangeStartScroll && id != ctrlRangeEndScroll {
		return false
	}
	code := int(uint16(wParam & 0xffff))
	thumb := int(uint16((wParam >> 16) & 0xffff))
	start, end := normalizeWalkRange(a.walkRangeStart, a.walkRangeEnd)
	pos := start
	if id == ctrlRangeEndScroll {
		pos = end
	}
	switch code {
	case sbLineLeft:
		pos--
	case sbLineRight:
		pos++
	case sbPageLeft:
		pos -= walkRangeStep
	case sbPageRight:
		pos += walkRangeStep
	case sbThumbPosition, sbThumbTrack:
		pos = thumb
	case sbLeft:
		pos = 0
	case sbRight:
		pos = 100
	case sbEndScroll:
		return true
	default:
		return true
	}
	if id == ctrlRangeStartScroll {
		start = clamp(pos, 0, end-minWalkRangeSpan)
	} else {
		end = clamp(pos, start+minWalkRangeSpan, 100)
	}
	a.setWalkRange(start, end)
	a.syncSettingsWindow()
	a.persistSettings()
	a.render()
	return true
}

func (a *petApp) handleSettingsCommand(id int32, notify uint16) bool {
	if id == ctrlNameLabels {
		a.nameLabels = !a.nameLabels
		if !a.nameLabels {
			a.hideNameWindow()
		}
		a.recreateSettingsWindow()
		a.persistSettings()
		a.render()
		return true
	}
	if id >= ctrlPetNameBase && id < ctrlPetNameBase+maxPetCount {
		if !a.nameLabels {
			return true
		}
		a.showRenameDialog(int(id - ctrlPetNameBase))
		return true
	}
	switch id {
	case ctrlTabHome:
		a.settingsTab = tabHome
		a.recreateSettingsWindow()
	case ctrlTabAnimals:
		a.settingsTab = tabAnimals
		a.recreateSettingsWindow()
	case ctrlTabMotion:
		a.settingsTab = tabMotion
		a.recreateSettingsWindow()
	case ctrlTabDisplay:
		a.settingsTab = tabDisplay
		a.recreateSettingsWindow()
	case ctrlTabUpdates:
		a.settingsTab = tabUpdates
		a.recreateSettingsWindow()
	case ctrlHomeAnimals:
		a.settingsTab = tabAnimals
		a.recreateSettingsWindow()
	case ctrlHomeMotion:
		a.settingsTab = tabMotion
		a.recreateSettingsWindow()
	case ctrlHomeDisplay:
		a.settingsTab = tabDisplay
		a.recreateSettingsWindow()
	case ctrlHomeUpdates:
		a.settingsTab = tabUpdates
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
		a.resetPosition()
		a.recreateSettingsWindow()
	case ctrlPetPlus:
		a.setPetCount(a.petCount + 1)
		a.resetPosition()
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
	case ctrlPositionTaskbar:
		a.positionMode = positionTaskbarEdge
	case ctrlPositionBottom:
		a.positionMode = positionScreenBottom
	case ctrlOffsetUp:
		a.adjustOverlayOffset(-overlayOffsetStep)
	case ctrlOffsetDown:
		a.adjustOverlayOffset(overlayOffsetStep)
	case ctrlDisplaySingle:
		a.setDisplayScope(displayScopeSingle)
	case ctrlDisplaySpan:
		a.setDisplayScope(displayScopeSpan)
	case ctrlDisplaySpanLess:
		a.adjustDisplaySpan(-1)
	case ctrlDisplaySpanMore:
		a.adjustDisplaySpan(1)
	case ctrlLaneStaggered:
		a.laneMode = laneStaggered
		a.applyLaneOffsets()
	case ctrlLaneAligned:
		a.laneMode = laneAligned
		a.applyLaneOffsets()
	case ctrlDisplayPrev:
		a.adjustDisplayIndex(-1)
	case ctrlDisplayNext:
		a.adjustDisplayIndex(1)
	case ctrlRangeFull:
		a.setWalkRange(defaultWalkRangeStart, defaultWalkRangeEnd)
	case ctrlRangeNarrow:
		a.adjustWalkRangeWidth(-walkRangeStep)
	case ctrlRangeWide:
		a.adjustWalkRangeWidth(walkRangeStep)
	case ctrlRangeLeft:
		a.shiftWalkRange(-walkRangeStep)
	case ctrlRangeRight:
		a.shiftWalkRange(walkRangeStep)
	case ctrlUpdateCheck:
		a.startUpdateCheck(true)
	case ctrlUpdateInstall:
		a.installLatestUpdate()
	case ctrlReset:
		a.resetOverlayPlacement()
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

func (a *petApp) showRenameDialog(index int) {
	if index < 0 || index >= maxPetCount || a.settingsHwnd == 0 {
		return
	}
	if a.renameHwnd != 0 {
		win.DestroyWindow(a.renameHwnd)
		a.renameHwnd = 0
	}
	a.ensureSettingsBrushes()
	a.ensureSettingsFonts()
	var parentRect win.RECT
	win.GetWindowRect(a.settingsHwnd, &parentRect)
	w, h := int32(336), int32(158)
	x := parentRect.Left + (parentRect.Right-parentRect.Left-w)/2
	y := parentRect.Top + (parentRect.Bottom-parentRect.Top-h)/2
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOOLWINDOW,
		syscall.StringToUTF16Ptr(windowClass),
		syscall.StringToUTF16Ptr(a.localText("名前を変更", "Rename pet")),
		win.WS_POPUP|win.WS_CAPTION|win.WS_SYSMENU|win.WS_VISIBLE|win.WS_CLIPCHILDREN,
		x, y, w, h,
		a.settingsHwnd, 0, a.hinst, nil,
	)
	if hwnd == 0 {
		return
	}
	a.renameHwnd = hwnd
	a.renameIndex = index
	a.renameEdit = a.createEdit(hwnd, ctrlRenameEdit, a.petDisplayName(index), 24, 56, 288, 28)
	a.createButton(hwnd, ctrlRenameOK, "", 144, 104, 78, 30, 0)
	a.createButton(hwnd, ctrlRenameCancel, "", 234, 104, 78, 30, 0)
	win.SetForegroundWindow(hwnd)
	win.SetFocus(a.renameEdit)
}

func (a *petApp) renameWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_COMMAND:
		id := int32(uint16(wParam & 0xffff))
		switch id {
		case ctrlRenameOK:
			a.commitRenameDialog()
			return 0
		case ctrlRenameCancel, int32(win.IDCANCEL):
			win.DestroyWindow(hwnd)
			return 0
		}
	case win.WM_PAINT:
		a.paintRenameWindow(hwnd)
		return 0
	case win.WM_DRAWITEM:
		dis := drawItemStruct(lParam)
		if a.drawSettingsButton(&dis) {
			return 1
		}
		return 0
	case win.WM_CTLCOLORSTATIC:
		win.SetBkMode(win.HDC(wParam), win.TRANSPARENT)
		win.SetTextColor(win.HDC(wParam), win.RGB(32, 37, 31))
		return uintptr(win.GetStockObject(win.NULL_BRUSH))
	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_DESTROY:
		if hwnd == a.renameHwnd {
			a.renameHwnd = 0
			a.renameEdit = 0
			a.renameIndex = -1
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (a *petApp) paintRenameWindow(hwnd win.HWND) {
	a.ensureSettingsFonts()
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	if hdc == 0 {
		return
	}
	defer win.EndPaint(hwnd, &ps)
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)
	drawRectFill(hdc, rect, rgb(247, 248, 244))
	label := a.localText(fmt.Sprintf("%d匹目の名前", a.renameIndex+1), fmt.Sprintf("Pet %d name", a.renameIndex+1))
	drawTextLine(hdc, label, win.RECT{Left: 24, Top: 20, Right: rect.Right - 24, Bottom: 46}, a.settingsFont, rgb(27, 36, 32), win.DT_LEFT|win.DT_VCENTER|win.DT_SINGLELINE|win.DT_NOPREFIX)
}

func (a *petApp) commitRenameDialog() {
	if a.renameIndex >= 0 && a.renameIndex < maxPetCount && a.renameEdit != 0 {
		a.petNames[a.renameIndex] = sanitizePetName(getWindowText(a.renameEdit))
		a.syncSettingsWindow()
		a.persistSettings()
		a.updateHoverName()
		a.render()
	}
	if a.renameHwnd != 0 {
		win.DestroyWindow(a.renameHwnd)
	}
}

func (a *petApp) recreateSettingsWindow() {
	if a.settingsHwnd != 0 {
		a.rememberSettingsWindowPosition()
		a.destroySettingsTooltip()
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
		case "homePageTitle":
			return "Overview"
		case "homePageLead":
			return "Review the key settings and jump directly to the section you need."
		case "animalPageTitle":
			return "Pets and coats"
		case "animalPageLead":
			return "Choose how many degus appear and how their coats are assigned."
		case "motionPageTitle":
			return "Motion behavior"
		case "motionPageLead":
			return "Tune keyboard reactions, random strolls, wheel behavior, and turn behavior."
		case "displayPageTitle":
			return "Display and walking displays"
		case "displayPageLead":
			return "Choose one display or a multi-monitor span, then fine-tune the area where degus can roam."
		case "updatesPageTitle":
			return "Updates"
		case "updatesPageLead":
			return "Check the installed version and install a newer GitHub Release when available."
		case "tabAnimals":
			return "Animals"
		case "tabMotion":
			return "Motion"
		case "tabDisplay":
			return "Display"
		case "tabUpdates":
			return "Updates"
		case "tabHome":
			return "Home"
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
			return "Reset layout"
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
	case "homePageTitle":
		return "ホーム"
	case "homePageLead":
		return "よく使う設定と状態をまとめて確認し、必要な場所へ移動できます。"
	case "animalPageTitle":
		return "デグーの数と毛色"
	case "animalPageLead":
		return "デグーの数と、毛色の選び方を調整します。"
	case "motionPageTitle":
		return "動きかた"
	case "motionPageLead":
		return "キーボードへの反応、ランダム散歩、回し車、左右ターンを調整します。"
	case "displayPageTitle":
		return "表示と歩く画面"
	case "displayPageLead":
		return "1つの画面だけ、または複数モニタを横断する範囲を選び、必要な時だけ歩く区間を細かく調整します。"
	case "updatesPageTitle":
		return "更新"
	case "updatesPageLead":
		return "インストール済みのバージョンと、GitHub Release の更新を確認します。"
	case "tabAnimals":
		return "動物"
	case "tabMotion":
		return "動き"
	case "tabDisplay":
		return "表示"
	case "tabUpdates":
		return "更新"
	case "tabHome":
		return "ホーム"
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
		return "配置リセット"
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
	a.enterWheel(p, wheelKeyHold)
}

func (a *petApp) enterWheelFromRandom(p *deguPet) {
	a.enterWheel(p, randomWheelMinTicks+rand.Intn(randomWheelExtraTicks))
}

func (a *petApp) tryStartRandomWheel(p *deguPet, roll int) bool {
	if a.mode != modeRandom || !a.wheelEnabled || p.item != noItem || a.hasWheelRunner() || roll >= randomWheelChance {
		return false
	}
	a.enterWheelFromRandom(p)
	return true
}

func (a *petApp) enterWheel(p *deguPet, ticks int) {
	alreadyRunning := p.state == stateWheel
	p.state = stateWheel
	if !alreadyRunning {
		p.frame = 0
		p.motionSet = rand.Intn(motionSets)
	}
	p.item = noItem
	p.carryKind = noItem
	p.moveSpeed = 0
	p.stateTicks = max(1, ticks)
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

func (a *petApp) showStartupToast() {
	a.showTrayBalloon(
		a.localText("Degu Desktop 起動中", "Degu Desktop is running"),
		a.localText("タスクトレイから設定と終了ができます。", "Use the tray icon for settings and exit."),
	)
}

func (a *petApp) showTrayBalloon(title, body string) {
	if a.hwnd == 0 {
		return
	}
	var nid win.NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	nid.UFlags = win.NIF_INFO
	nid.DwInfoFlags = win.NIIF_INFO | win.NIIF_RESPECT_QUIET_TIME
	copyUTF16Limited(nid.SzInfoTitle[:], title)
	copyUTF16Limited(nid.SzInfo[:], body)
	win.Shell_NotifyIcon(win.NIM_MODIFY, &nid)
}

func copyUTF16Limited(dst []uint16, text string) {
	if len(dst) == 0 {
		return
	}
	src := syscall.StringToUTF16(text)
	n := min(len(dst), len(src))
	copy(dst[:n], src[:n])
	if n == len(dst) {
		dst[len(dst)-1] = 0
	}
}

func (a *petApp) updateChecking() bool {
	return a.update.checking.Load() || a.update.installing.Load()
}

func (a *petApp) hasInstallableUpdate() bool {
	rel := a.currentRelease()
	if rel == nil || !isNewerVersion(rel.TagName, appVersion) {
		return false
	}
	return selectUpdateAsset(rel, runtime.GOARCH) != nil
}

func (a *petApp) updateCheckMenuLabel() string {
	if a.update.installing.Load() {
		return a.localText("アップデート適用中...", "Installing update...")
	}
	if a.update.checking.Load() {
		return a.localText("アップデート確認中...", "Checking for updates...")
	}
	return a.localText("アップデートを確認", "Check for updates")
}

func (a *petApp) updateInstallMenuLabel() string {
	rel := a.currentRelease()
	if rel == nil {
		return a.localText("アップデートをインストール", "Install update")
	}
	return a.localText(
		fmt.Sprintf("%s をインストール", rel.TagName),
		fmt.Sprintf("Install %s", rel.TagName),
	)
}

func (a *petApp) startUpdateCheck(manual bool) {
	if !a.update.checking.CompareAndSwap(false, true) {
		if manual {
			a.showTrayBalloon(
				a.localText("確認中です", "Already checking"),
				a.localText("最新アップデートの確認が進行中です。", "An update check is already in progress."),
			)
		}
		return
	}
	a.syncSettingsWindow()
	go func() {
		rel, err := fetchLatestRelease()
		a.setUpdateResult(rel, err)
		var notify uintptr
		if manual {
			notify = 1
		}
		if err != nil {
			win.PostMessage(a.hwnd, wmUpdateFailed, notify, 0)
			return
		}
		win.PostMessage(a.hwnd, wmUpdateReady, notify, 0)
	}()
}

func (a *petApp) onUpdateReady(manual bool) {
	a.update.checking.Store(false)
	a.syncSettingsWindow()
	rel := a.currentRelease()
	if rel == nil {
		if manual {
			a.showTrayBalloon(
				a.localText("確認できませんでした", "Update check failed"),
				a.localText("最新リリース情報が空でした。", "The latest release response was empty."),
			)
		}
		return
	}
	asset := selectUpdateAsset(rel, runtime.GOARCH)
	if asset == nil {
		if manual {
			a.showTrayBalloon(
				a.localText("配布 zip が見つかりません", "Update package not found"),
				a.localText(updateAssetName(runtime.GOARCH)+" が Release にありません。", updateAssetName(runtime.GOARCH)+" is missing from the release."),
			)
		}
		return
	}
	if !isNewerVersion(rel.TagName, appVersion) {
		if manual {
			a.showTrayBalloon(
				a.localText("最新版です", "Up to date"),
				a.localText("インストール済みの Degu Desktop は最新です。", "The installed Degu Desktop is current."),
			)
		}
		return
	}
	a.showTrayBalloon(
		a.localText("アップデートがあります", "Update available"),
		a.localText(
			fmt.Sprintf("%s をトレイメニューからインストールできます。", rel.TagName),
			fmt.Sprintf("%s can be installed from the tray menu.", rel.TagName),
		),
	)
}

func (a *petApp) onUpdateFailed(notify bool) {
	a.update.checking.Store(false)
	a.update.installing.Store(false)
	a.syncSettingsWindow()
	if !notify {
		return
	}
	a.showTrayBalloon(
		a.localText("アップデートに失敗しました", "Update failed"),
		a.currentUpdateError(),
	)
}

func (a *petApp) installLatestUpdate() {
	rel := a.currentRelease()
	if rel == nil || !isNewerVersion(rel.TagName, appVersion) {
		a.showTrayBalloon(
			a.localText("アップデートなし", "No update available"),
			a.localText("先にアップデートを確認してください。", "Check for updates first."),
		)
		return
	}
	asset := selectUpdateAsset(rel, runtime.GOARCH)
	if asset == nil {
		a.showTrayBalloon(
			a.localText("配布 zip が見つかりません", "Update package not found"),
			a.localText(updateAssetName(runtime.GOARCH)+" が Release にありません。", updateAssetName(runtime.GOARCH)+" is missing from the release."),
		)
		return
	}
	if !a.update.installing.CompareAndSwap(false, true) {
		a.showTrayBalloon(
			a.localText("適用中です", "Install already running"),
			a.localText("アップデートのダウンロードと適用を進めています。", "The update is already being downloaded and installed."),
		)
		return
	}
	a.syncSettingsWindow()
	a.showTrayBalloon(
		a.localText("アップデートを開始", "Starting update"),
		a.localText("ダウンロード後に Degu Desktop を再起動します。", "Degu Desktop will restart after download."),
	)
	go func(asset githubReleaseAsset) {
		err := downloadAndStartUpdater(asset)
		a.setUpdateResult(rel, err)
		if err != nil {
			win.PostMessage(a.hwnd, wmUpdateFailed, 1, 0)
			return
		}
		win.PostMessage(a.hwnd, wmUpdateInstallReady, 0, 0)
	}(*asset)
}

func (a *petApp) onUpdateInstallReady() {
	a.syncSettingsWindow()
	a.showTrayBalloon(
		a.localText("アップデートを適用します", "Applying update"),
		a.localText("本体を終了して、新しいバージョンで再起動します。", "The app will close and restart with the new version."),
	)
	a.closing.Store(true)
	win.DestroyWindow(a.hwnd)
}

func (a *petApp) setUpdateResult(rel *githubRelease, err error) {
	a.update.mu.Lock()
	defer a.update.mu.Unlock()
	if rel != nil {
		cp := *rel
		cp.Assets = append([]githubReleaseAsset(nil), rel.Assets...)
		a.update.latest = &cp
	}
	if err != nil {
		a.update.lastError = err.Error()
	} else {
		a.update.lastError = ""
	}
}

func (a *petApp) currentRelease() *githubRelease {
	a.update.mu.Lock()
	defer a.update.mu.Unlock()
	if a.update.latest == nil {
		return nil
	}
	cp := *a.update.latest
	cp.Assets = append([]githubReleaseAsset(nil), a.update.latest.Assets...)
	return &cp
}

func (a *petApp) currentUpdateError() string {
	a.update.mu.Lock()
	defer a.update.mu.Unlock()
	if a.update.lastError == "" {
		return a.localText("詳細なエラーはありません。", "No detailed error is available.")
	}
	return a.update.lastError
}

func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, updateAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DeguDesktop/"+appVersion)
	client := http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub Releases API returned HTTP %d", resp.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("latest release has no tag name")
	}
	if rel.Draft {
		return nil, fmt.Errorf("latest release is still a draft")
	}
	return &rel, nil
}

func selectUpdateAsset(rel *githubRelease, goarch string) *githubReleaseAsset {
	if rel == nil {
		return nil
	}
	want := updateAssetName(goarch)
	for i := range rel.Assets {
		asset := &rel.Assets[i]
		if strings.EqualFold(asset.Name, want) && asset.BrowserDownloadURL != "" {
			return asset
		}
	}
	return nil
}

func updateAssetName(goarch string) string {
	switch goarch {
	case "386":
		return "DeguDesktop-windows-386.zip"
	default:
		return "DeguDesktop-windows-amd64.zip"
	}
}

func isNewerVersion(latest, current string) bool {
	latest = normalizeVersionTag(latest)
	current = normalizeVersionTag(current)
	if latest == "" || latest == current {
		return false
	}
	latestParts, latestOK := parseVersionParts(latest)
	currentParts, currentOK := parseVersionParts(current)
	if latestOK && currentOK {
		for i := 0; i < max(len(latestParts), len(currentParts)); i++ {
			lv, cv := 0, 0
			if i < len(latestParts) {
				lv = latestParts[i]
			}
			if i < len(currentParts) {
				cv = currentParts[i]
			}
			if lv != cv {
				return lv > cv
			}
		}
		return false
	}
	return current == "" || current == "dev" || strings.HasPrefix(current, "pages-")
}

func normalizeVersionTag(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

func parseVersionParts(version string) ([]int, bool) {
	if version == "" {
		return nil, false
	}
	if cut := strings.IndexAny(version, "-+"); cut >= 0 {
		version = version[:cut]
	}
	parts := strings.Split(version, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
		value := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				return nil, false
			}
			value = value*10 + int(r-'0')
		}
		out = append(out, value)
	}
	return out, true
}

func downloadAndStartUpdater(asset githubReleaseAsset) error {
	if asset.BrowserDownloadURL == "" {
		return fmt.Errorf("update asset has no download URL")
	}
	tmpDir, err := os.MkdirTemp("", updateTempPrefix+"*")
	if err != nil {
		return err
	}
	zipPath := filepath.Join(tmpDir, "update.zip")
	if err := downloadFile(asset.BrowserDownloadURL, zipPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if err := verifyDownloadedAsset(zipPath, asset); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	exePath, err := extractUpdateExe(zipPath, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	currentExe, err := os.Executable()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	helperPath := filepath.Join(tmpDir, "helper", "DeguDesktop.exe")
	if err := os.MkdirAll(filepath.Dir(helperPath), 0o755); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if err := copyFile(currentExe, helperPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return startUpdaterHelper(helperPath, tmpDir, exePath, currentExe, os.Getpid())
}

func downloadFile(url, path string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DeguDesktop/"+appVersion)
	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func verifyDownloadedAsset(path string, asset githubReleaseAsset) error {
	if asset.Size > 0 {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() != asset.Size {
			return fmt.Errorf("downloaded update size mismatch: got %d bytes, want %d", info.Size(), asset.Size)
		}
	}
	if asset.Digest == "" {
		return nil
	}
	algorithm, want, ok := strings.Cut(asset.Digest, ":")
	if !ok || !strings.EqualFold(algorithm, "sha256") || len(want) != 64 {
		return fmt.Errorf("unsupported update digest %q", asset.Digest)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := fmt.Sprintf("%x", sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("downloaded update digest mismatch")
	}
	return nil
}

func extractUpdateExe(zipPath, tmpDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if !strings.EqualFold(filepath.Base(file.Name), "DeguDesktop.exe") {
			continue
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		defer src.Close()
		exePath := filepath.Join(tmpDir, "payload", "DeguDesktop.exe")
		if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
			return "", err
		}
		dst, err := os.Create(exePath)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			return "", err
		}
		if err := dst.Close(); err != nil {
			return "", err
		}
		return exePath, os.Chmod(exePath, 0o755)
	}
	return "", fmt.Errorf("DeguDesktop.exe was not found in update zip")
}

func startUpdaterHelper(helperPath, tmpDir, sourceExe, targetExe string, pid int) error {
	return newUpdaterHelperCommand(helperPath, tmpDir, sourceExe, targetExe, pid).Start()
}

func newUpdaterHelperCommand(helperPath, tmpDir, sourceExe, targetExe string, pid int) *exec.Cmd {
	cmd := exec.Command(
		helperPath,
		updaterApplyArg,
		"--source", sourceExe,
		"--target", targetExe,
		"--parent-pid", strconv.Itoa(pid),
		"--cleanup-dir", tmpDir,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

type updateApplyOptions struct {
	Source     string
	Target     string
	ParentPID  int
	CleanupDir string
}

func runUpdaterUtility(args []string) bool {
	if len(args) == 0 || args[0] != updaterApplyArg {
		return false
	}
	opts, err := parseUpdateApplyArgs(args[1:])
	if err != nil {
		os.Exit(2)
	}
	if err := applyUpdate(opts); err != nil {
		os.Exit(1)
	}
	return true
}

func parseUpdateApplyArgs(args []string) (updateApplyOptions, error) {
	var opts updateApplyOptions
	for i := 0; i < len(args); i++ {
		if i+1 >= len(args) {
			return opts, fmt.Errorf("%s is missing a value", args[i])
		}
		value := args[i+1]
		switch args[i] {
		case "--source":
			opts.Source = value
		case "--target":
			opts.Target = value
		case "--parent-pid":
			pid, err := strconv.Atoi(value)
			if err != nil || pid < 0 {
				return opts, fmt.Errorf("invalid parent pid %q", value)
			}
			opts.ParentPID = pid
		case "--cleanup-dir":
			opts.CleanupDir = value
		default:
			return opts, fmt.Errorf("unknown updater argument %q", args[i])
		}
		i++
	}
	if opts.Source == "" || opts.Target == "" || opts.CleanupDir == "" {
		return opts, fmt.Errorf("updater source, target, and cleanup-dir are required")
	}
	if !isUpdateTempDir(opts.CleanupDir) {
		return opts, fmt.Errorf("refusing cleanup outside update temp dir")
	}
	if !isPathInsideDir(opts.Source, opts.CleanupDir) || !strings.EqualFold(filepath.Base(opts.Source), "DeguDesktop.exe") {
		return opts, fmt.Errorf("updater source must be DeguDesktop.exe inside the update temp dir")
	}
	if !strings.EqualFold(filepath.Base(opts.Target), "DeguDesktop.exe") || isPathInsideDir(opts.Target, opts.CleanupDir) {
		return opts, fmt.Errorf("updater target must be an installed DeguDesktop.exe outside the update temp dir")
	}
	return opts, nil
}

func applyUpdate(opts updateApplyOptions) error {
	if opts.ParentPID > 0 {
		if err := waitForProcessExit(opts.ParentPID, 120*time.Second); err != nil {
			return err
		}
		time.Sleep(300 * time.Millisecond)
	}
	if err := copyFile(opts.Source, opts.Target); err != nil {
		return err
	}
	return newUpdaterCleanupCommand(opts.Target, opts.CleanupDir).Start()
}

func newUpdaterCleanupCommand(targetExe, cleanupDir string) *exec.Cmd {
	cmd := exec.Command(targetExe, updaterCleanupArg, cleanupDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(handle)
	waitMS := uint32(timeout / time.Millisecond)
	if timeout <= 0 {
		waitMS = syscall.INFINITE
	}
	result, err := syscall.WaitForSingleObject(handle, waitMS)
	if err != nil {
		return err
	}
	if result == syscall.WAIT_TIMEOUT {
		return fmt.Errorf("timed out waiting for process %d to exit", pid)
	}
	if result == syscall.WAIT_FAILED {
		return fmt.Errorf("failed waiting for process %d", pid)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}

func updateCleanupDir(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == updaterCleanupArg && isUpdateTempDir(args[i+1]) {
			return args[i+1]
		}
	}
	return ""
}

func cleanupUpdateTempDirLater(dir string) {
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if err := os.RemoveAll(dir); err == nil {
			return
		}
	}
}

func isUpdateTempDir(path string) bool {
	if path == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absTemp, err := filepath.Abs(os.TempDir())
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absTemp, absPath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	return strings.HasPrefix(filepath.Base(absPath), updateTempPrefix)
}

func isPathInsideDir(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
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

func (a *petApp) temporaryVisibilityMenuLabel() string {
	if a.temporarilyHidden {
		return a.localText("再表示", "Show pets")
	}
	return a.localText("一時的に非表示", "Temporarily hide")
}

func (a *petApp) setTemporarilyHidden(hidden bool) {
	if a.temporarilyHidden == hidden {
		return
	}
	a.temporarilyHidden = hidden
	a.hoverPet = -1
	a.hideNameWindow()
	if a.hwnd == 0 {
		return
	}
	if hidden {
		win.ShowWindow(a.hwnd, win.SW_HIDE)
		return
	}
	win.ShowWindow(a.hwnd, win.SW_SHOWNOACTIVATE)
	a.render()
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
	for count := 1; count <= maxPetCount; count++ {
		appendChecked(countMenu, menuIDForPetCount(count), fmt.Sprintf("%d", count), a.petCount == count)
	}
	appendMenu(menu, win.MF_POPUP|win.MF_STRING, uintptr(countMenu), syscall.StringToUTF16Ptr(a.txt("deguCount")))

	appendChecked(menu, menuWheelToggle, a.txt("typingWheel"), a.wheelEnabled)
	appendMenu(menu, win.MF_SEPARATOR, 0, nil)
	updateFlags := uint32(win.MF_STRING)
	if a.updateChecking() {
		updateFlags |= win.MF_GRAYED
	}
	appendMenu(menu, updateFlags, uintptr(menuCheckUpdate), syscall.StringToUTF16Ptr(a.updateCheckMenuLabel()))
	if a.hasInstallableUpdate() {
		appendMenu(menu, win.MF_STRING, uintptr(menuInstallUpdate), syscall.StringToUTF16Ptr(a.updateInstallMenuLabel()))
	}
	appendMenu(menu, win.MF_SEPARATOR, 0, nil)
	appendMenu(menu, win.MF_STRING, uintptr(menuToggleHidden), syscall.StringToUTF16Ptr(a.temporaryVisibilityMenuLabel()))
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

func menuIDForPetCount(count int) uint16 {
	return menuCountBase + uint16(clamp(count, 1, maxPetCount))
}

func petCountFromMenuID(id uint16) (int, bool) {
	if id <= menuCountBase {
		return 0, false
	}
	count := int(id - menuCountBase)
	if count < 1 || count > maxPetCount {
		return 0, false
	}
	return count, true
}

func (a *petApp) handleMenu(id uint16) {
	if !a.handleMenuCommand(id) {
		return
	}
	a.syncSettingsWindow()
	if id == menuToggleHidden {
		return
	}
	a.persistSettings()
}

func (a *petApp) handleMenuCommand(id uint16) bool {
	if count, ok := petCountFromMenuID(id); ok {
		a.setPetCount(count)
		a.resetPosition()
		return true
	}
	switch {
	case id == menuExit:
		a.closing.Store(true)
		win.DestroyWindow(a.hwnd)
	case id == menuSettings:
		a.showSettings()
	case id == menuToggleHidden:
		a.setTemporarilyHidden(!a.temporarilyHidden)
	case id == menuCheckUpdate:
		a.startUpdateCheck(true)
	case id == menuInstallUpdate:
		a.installLatestUpdate()
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
		a.destroySettingsTooltip()
		win.DestroyWindow(a.settingsHwnd)
		a.settingsHwnd = 0
	}
	if a.renameHwnd != 0 {
		win.DestroyWindow(a.renameHwnd)
		a.renameHwnd = 0
	}
	if a.nameHwnd != 0 {
		win.DestroyWindow(a.nameHwnd)
		a.nameHwnd = 0
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

func getDlgCtrlID(hwnd win.HWND) int32 {
	if hwnd == 0 {
		return 0
	}
	ret, _, _ := procGetDlgCtrlID.Call(uintptr(hwnd))
	return int32(ret)
}

func getWindowText(hwnd win.HWND) string {
	if hwnd == 0 {
		return ""
	}
	length := int(win.SendMessage(hwnd, win.WM_GETTEXTLENGTH, 0, 0))
	buf := make([]uint16, max(1, length+1))
	win.SendMessage(hwnd, win.WM_GETTEXT, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
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

type displayArea struct {
	Work    win.RECT
	Screen  win.RECT
	Primary bool
}

var (
	monitorEnumCallback = syscall.NewCallback(monitorEnumProc)
	monitorEnumMu       sync.Mutex
	monitorEnumAreas    *[]displayArea
)

func monitorEnumProc(hMonitor win.HMONITOR, _ win.HDC, _ *win.RECT, lParam uintptr) uintptr {
	if monitorEnumAreas == nil {
		return 0
	}
	info := win.MONITORINFO{CbSize: uint32(unsafe.Sizeof(win.MONITORINFO{}))}
	if win.GetMonitorInfo(hMonitor, &info) {
		*monitorEnumAreas = append(*monitorEnumAreas, displayArea{
			Work:    info.RcWork,
			Screen:  info.RcMonitor,
			Primary: info.DwFlags&monitorPrimaryFlag != 0,
		})
	}
	return 1
}

func (a *petApp) selectedDisplayArea() displayArea {
	areas := displayAreaForScope(a.displayScope)
	if len(areas) == 0 {
		return displayArea{Work: workArea(), Screen: screenArea(), Primary: true}
	}
	scope, start, end := a.normalizedDisplaySelection(len(areas))
	a.displayScope = scope
	a.displayIndex = start
	a.displaySpanEnd = end
	if scope == displayScopeSpan {
		return combineDisplayAreas(areas[start : end+1])
	}
	return areas[start]
}

func monitorAreas() []displayArea {
	areas := make([]displayArea, 0, 4)
	monitorEnumMu.Lock()
	monitorEnumAreas = &areas
	ret, _, _ := procEnumDisplayMonitors.Call(0, 0, monitorEnumCallback, 0)
	monitorEnumAreas = nil
	monitorEnumMu.Unlock()
	if ret == 0 || len(areas) == 0 {
		return nil
	}
	sort.SliceStable(areas, func(i, j int) bool {
		if areas[i].Primary != areas[j].Primary {
			return areas[i].Primary
		}
		if areas[i].Screen.Left != areas[j].Screen.Left {
			return areas[i].Screen.Left < areas[j].Screen.Left
		}
		return areas[i].Screen.Top < areas[j].Screen.Top
	})
	return areas
}

func monitorAreasByPosition() []displayArea {
	areas := append([]displayArea(nil), monitorAreas()...)
	sort.SliceStable(areas, func(i, j int) bool {
		if areas[i].Screen.Left != areas[j].Screen.Left {
			return areas[i].Screen.Left < areas[j].Screen.Left
		}
		if areas[i].Screen.Top != areas[j].Screen.Top {
			return areas[i].Screen.Top < areas[j].Screen.Top
		}
		if areas[i].Primary != areas[j].Primary {
			return areas[i].Primary
		}
		return false
	})
	return areas
}

func displayAreaForScope(scope displayScope) []displayArea {
	if normalizeDisplayScope(int(scope)) == displayScopeSpan {
		return monitorAreasByPosition()
	}
	return monitorAreas()
}

func sameDisplayArea(a, b displayArea) bool {
	return a.Screen.Left == b.Screen.Left &&
		a.Screen.Top == b.Screen.Top &&
		a.Screen.Right == b.Screen.Right &&
		a.Screen.Bottom == b.Screen.Bottom
}

func findDisplayAreaIndex(areas []displayArea, target displayArea) int {
	for i, area := range areas {
		if sameDisplayArea(area, target) {
			return i
		}
	}
	return 0
}

func combineDisplayAreas(areas []displayArea) displayArea {
	if len(areas) == 0 {
		return displayArea{}
	}
	screen := areas[0].Screen
	workLeft := areas[0].Work.Left
	workRight := areas[0].Work.Right
	workTop := areas[0].Work.Top
	workBottom := areas[0].Work.Bottom
	primary := areas[0].Primary
	for _, area := range areas[1:] {
		screen.Left = min32(screen.Left, area.Screen.Left)
		screen.Top = min32(screen.Top, area.Screen.Top)
		screen.Right = max32(screen.Right, area.Screen.Right)
		screen.Bottom = max32(screen.Bottom, area.Screen.Bottom)
		workLeft = min32(workLeft, area.Work.Left)
		workRight = max32(workRight, area.Work.Right)
		workTop = max32(workTop, area.Work.Top)
		workBottom = min32(workBottom, area.Work.Bottom)
		primary = primary || area.Primary
	}
	work := win.RECT{Left: workLeft, Top: workTop, Right: workRight, Bottom: workBottom}
	if work.Right <= work.Left || work.Bottom <= work.Top {
		work = screen
	}
	return displayArea{Work: work, Screen: screen, Primary: primary}
}

func workArea() win.RECT {
	var rect win.RECT
	if !win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&rect), 0) {
		rect = win.RECT{Left: 0, Top: 0, Right: 1280, Bottom: 720}
	}
	return rect
}

func screenArea() win.RECT {
	left := int32(win.GetSystemMetrics(win.SM_XVIRTUALSCREEN))
	top := int32(win.GetSystemMetrics(win.SM_YVIRTUALSCREEN))
	width := win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
	height := win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	if width <= 0 || height <= 0 {
		left = 0
		top = 0
		width = win.GetSystemMetrics(win.SM_CXSCREEN)
		height = win.GetSystemMetrics(win.SM_CYSCREEN)
	}
	if width <= 0 || height <= 0 {
		return workArea()
	}
	return win.RECT{Left: left, Top: top, Right: left + width, Bottom: top + height}
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
	runnerW := 56
	runnerH := 38
	scaled := fitVisibleImageTo(src, runnerW, runnerH)
	bob := int(math.Sin(float64(frame)/2.0) * 2)
	dstX := x + (wheelSize-runnerW)/2
	dstY := y + wheelSize/2 - runnerH/2 + 2 + bob
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

func max32(a, b int32) int32 {
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

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
