//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lxn/win"
)

func TestHorizontalMotionFramesUseStableRightFacingSequence(t *testing.T) {
	allowed := map[int]bool{
		walkStart:     true,
		walkStart + 1: true,
		walkStart + 3: true,
	}
	states := []behaviorState{
		stateWalk,
		stateScurry,
		stateForage,
		stateCarry,
	}

	for _, state := range states {
		for frame := 0; frame < 64; frame++ {
			got := currentFrame(state, frame)
			if !allowed[got] {
				t.Fatalf("currentFrame(%v, %d) = %d, want stable right-facing walk frame", state, frame, got)
			}
		}
	}
}

func TestWheelUsesDedicatedRunFrames(t *testing.T) {
	for frame := 0; frame < wheelRunFrames*2; frame++ {
		got := currentFrame(stateWheel, frame)
		if got < wheelRunStart || got >= wheelRunStart+wheelRunFrames {
			t.Fatalf("currentFrame(stateWheel, %d) = %d, want dedicated wheelrun frame", frame, got)
		}
	}
	if got := currentFrame(stateWheel, wheelRunFrames); got != wheelRunStart {
		t.Fatalf("wheelrun loop frame = %d, want %d", got, wheelRunStart)
	}
}

func TestWheelRunnerFitsInsideRimAcrossCoatSets(t *testing.T) {
	spriteSets := loadSprites()
	center := float64(wheelSize) / 2
	allowedRadius := float64(wheelSize/2 - 5)

	for _, variant := range variants {
		sets := spriteSets[variant.ID]
		if len(sets) != motionSets {
			t.Fatalf("%s sets = %d, want %d", variant.ID, len(sets), motionSets)
		}
		for setIndex, frames := range sets {
			for frame := wheelRunStart; frame < wheelRunStart+wheelRunFrames; frame++ {
				canvas := image.NewRGBA(image.Rect(0, 0, wheelSize, wheelSize))
				drawWheelRunner(canvas, 0, 0, frames[frame], frame)
				outside, total := wheelRunnerOutsidePixels(canvas, center, allowedRadius)
				if total == 0 {
					t.Fatalf("%s set %d frame %d produced no runner pixels", variant.ID, setIndex, frame)
				}
				if outside != 0 {
					t.Fatalf("%s set %d frame %d outside wheel rim = %d/%d pixels", variant.ID, setIndex, frame, outside, total)
				}
			}
		}
	}
}

func wheelRunnerOutsidePixels(img *image.RGBA, center, allowedRadius float64) (int, int) {
	outside := 0
	total := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A == 0 {
				continue
			}
			total++
			dx := float64(x) + 0.5 - center
			dy := float64(y) + 0.5 - center
			if math.Sqrt(dx*dx+dy*dy) > allowedRadius {
				outside++
			}
		}
	}
	return outside, total
}

func TestFrameFromSeqHandlesEmptyAndBadDivisor(t *testing.T) {
	if got := frameFromSeq(nil, 12, 2); got != idleStart {
		t.Fatalf("frameFromSeq(nil) = %d, want %d", got, idleStart)
	}
	seq := []int{7, 9}
	if got := frameFromSeq(seq, 3, 0); got != 9 {
		t.Fatalf("frameFromSeq with zero divisor = %d, want 9", got)
	}
}

func TestFrameFromSeqClampedHoldsFinalFrame(t *testing.T) {
	seq := []int{7, 9, 11}
	if got := frameFromSeqClamped(seq, 999, 2); got != 11 {
		t.Fatalf("frameFromSeqClamped past end = %d, want 11", got)
	}
	if got := frameFromSeqClamped(seq, 3, 0); got != 11 {
		t.Fatalf("frameFromSeqClamped with zero divisor = %d, want 11", got)
	}
}

func TestTypingStartsAndExtendsWheelOnlyInKeyboardMode(t *testing.T) {
	a := &petApp{
		mode:         modeKeyboard,
		wheelEnabled: true,
		wheelX:       400,
		sceneW:       1200,
		speed:        3,
		pets: []deguPet{
			{state: stateWalk, stateTicks: 12, item: noItem},
			{state: stateWalk, stateTicks: 12, item: noItem},
		},
	}

	a.onTyping()
	if got := a.pets[0].state; got != stateWheel {
		t.Fatalf("first pet state = %v, want stateWheel", got)
	}
	if got := a.pets[0].stateTicks; got != wheelKeyHold {
		t.Fatalf("wheel hold ticks = %d, want %d", got, wheelKeyHold)
	}
	if got := a.pets[0].moveSpeed; got != 0 {
		t.Fatalf("wheel pet moveSpeed = %d, want 0", got)
	}
	wantX := clamp(a.wheelX-wheelSize/2, 0, max(0, a.sceneW-spriteW))
	if got := a.pets[0].x; got != wantX {
		t.Fatalf("wheel pet x = %d, want %d", got, wantX)
	}
	if got := a.pets[1].state; got != stateScurry {
		t.Fatalf("second pet state = %v, want stateScurry", got)
	}

	a.pets[0].frame = 7
	a.pets[0].stateTicks = 3
	a.onTyping()
	if got := a.pets[0].frame; got != 7 {
		t.Fatalf("wheel frame reset while extending: got %d, want 7", got)
	}
	if got := a.pets[0].stateTicks; got != wheelKeyHold {
		t.Fatalf("extended wheel hold ticks = %d, want %d", got, wheelKeyHold)
	}
}

func TestTypingDoesNotStartWheelInRandomMode(t *testing.T) {
	a := &petApp{
		mode:         modeRandom,
		wheelEnabled: true,
		wheelX:       400,
		sceneW:       1200,
		pets: []deguPet{
			{state: stateWalk, stateTicks: 12, item: noItem},
		},
	}

	a.onTyping()
	if got := a.pets[0].state; got == stateWheel {
		t.Fatalf("typing in random mode started wheel state")
	}
}

func TestRandomStrollCanStartWheelWithoutTyping(t *testing.T) {
	a := &petApp{
		mode:         modeRandom,
		wheelEnabled: true,
		wheelX:       400,
		sceneW:       1200,
		pets: []deguPet{
			{state: stateWalk, item: noItem, moveSpeed: 3},
		},
	}

	if !a.tryStartRandomWheel(&a.pets[0], 0) {
		t.Fatalf("tryStartRandomWheel returned false, want true")
	}
	if got := a.pets[0].state; got != stateWheel {
		t.Fatalf("random wheel state = %v, want stateWheel", got)
	}
	if got := a.pets[0].moveSpeed; got != 0 {
		t.Fatalf("random wheel moveSpeed = %d, want 0", got)
	}
	if a.pets[0].stateTicks < randomWheelMinTicks || a.pets[0].stateTicks >= randomWheelMinTicks+randomWheelExtraTicks {
		t.Fatalf("random wheel ticks = %d, want [%d, %d)", a.pets[0].stateTicks, randomWheelMinTicks, randomWheelMinTicks+randomWheelExtraTicks)
	}
	wantX := clamp(a.wheelX-wheelSize/2, 0, max(0, a.sceneW-spriteW))
	if got := a.pets[0].x; got != wantX {
		t.Fatalf("random wheel x = %d, want %d", got, wantX)
	}
}

func TestRandomWheelRequiresRandomModeEnabledAndNoRunner(t *testing.T) {
	tests := []struct {
		name    string
		makeApp func() *petApp
		pet     deguPet
	}{
		{
			name:    "keyboard mode",
			makeApp: func() *petApp { return &petApp{mode: modeKeyboard, wheelEnabled: true, sceneW: 1000} },
			pet:     deguPet{state: stateWalk, item: noItem},
		},
		{
			name:    "wheel disabled",
			makeApp: func() *petApp { return &petApp{mode: modeRandom, wheelEnabled: false, sceneW: 1000} },
			pet:     deguPet{state: stateWalk, item: noItem},
		},
		{
			name:    "roll outside chance",
			makeApp: func() *petApp { return &petApp{mode: modeRandom, wheelEnabled: true, sceneW: 1000} },
			pet:     deguPet{state: stateWalk, item: noItem},
		},
		{
			name: "existing runner",
			makeApp: func() *petApp {
				return &petApp{
					mode:         modeRandom,
					wheelEnabled: true,
					sceneW:       1000,
					pets:         []deguPet{{state: stateWheel}},
				}
			},
			pet: deguPet{state: stateWalk, item: noItem},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.makeApp()
			roll := 0
			if tt.name == "roll outside chance" {
				roll = randomWheelChance
			}
			pet := tt.pet
			if got := a.tryStartRandomWheel(&pet, roll); got {
				t.Fatalf("tryStartRandomWheel() = true, want false")
			}
		})
	}
}

func TestSettingsRoundTripPersistsCoreOptions(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("APPDATA", configRoot)

	a := &petApp{
		variant:        4,
		coatMode:       coatSelected,
		selectedCoats:  [maxPetCount]int{1, 3, 5, 7, 9, 0, 2, 4, 6, 8},
		petNames:       [maxPetCount]string{"モカ", "Sora", "  Nagi  ", "", "", "", "", "", "", ""},
		nameLabels:     true,
		speed:          5,
		mode:           modeKeyboard,
		petCount:       10,
		wheelEnabled:   false,
		bidirectional:  false,
		positionMode:   positionScreenBottom,
		overlayOffsetY: 28,
		laneMode:       laneAligned,
		displayIndex:   1,
		walkRangeStart: 15,
		walkRangeEnd:   85,
		lang:           langEnglish,
		settingsX:      220,
		settingsY:      180,
	}
	if err := a.saveSettings(); err != nil {
		t.Fatalf("saveSettings() error = %v", err)
	}

	path := filepath.Join(configRoot, settingsDirName, settingsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings file was not written: %v", err)
	}
	var saved appSettings
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("settings json is invalid: %v", err)
	}
	if saved.Version != 1 || saved.PetCount != 10 || saved.Mode != int(modeKeyboard) {
		t.Fatalf("saved settings = %+v, want version 1 petCount 10 keyboard mode", saved)
	}
	if saved.PositionMode == nil || *saved.PositionMode != int(positionScreenBottom) {
		t.Fatalf("saved PositionMode = %v, want screen bottom", saved.PositionMode)
	}
	if saved.VerticalOffset == nil || *saved.VerticalOffset != 28 {
		t.Fatalf("saved VerticalOffset = %v, want 28", saved.VerticalOffset)
	}
	if saved.LaneMode == nil || *saved.LaneMode != int(laneAligned) {
		t.Fatalf("saved LaneMode = %v, want aligned", saved.LaneMode)
	}
	wantDisplayIndex := normalizeDisplayIndex(a.displayIndex, len(monitorAreas()))
	if saved.DisplayIndex == nil || *saved.DisplayIndex != wantDisplayIndex {
		t.Fatalf("saved DisplayIndex = %v, want %d", saved.DisplayIndex, wantDisplayIndex)
	}
	if saved.WalkRangeStart == nil || *saved.WalkRangeStart != 15 {
		t.Fatalf("saved WalkRangeStart = %v, want 15", saved.WalkRangeStart)
	}
	if saved.WalkRangeEnd == nil || *saved.WalkRangeEnd != 85 {
		t.Fatalf("saved WalkRangeEnd = %v, want 85", saved.WalkRangeEnd)
	}
	if !saved.NameLabels {
		t.Fatalf("saved NameLabels = false, want true")
	}
	if got := saved.PetNames[0]; got != "モカ" {
		t.Fatalf("saved pet name 0 = %q, want モカ", got)
	}
	if got := saved.PetNames[2]; got != "Nagi" {
		t.Fatalf("saved pet name 2 = %q, want sanitized Nagi", got)
	}

	b := &petApp{
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
		lang:           langJapanese,
		settingsX:      120,
		settingsY:      120,
	}
	if err := b.loadSettings(); err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if b.variant != a.variant || b.coatMode != a.coatMode || b.speed != a.speed || b.mode != a.mode || b.petCount != a.petCount {
		t.Fatalf("loaded scalar settings = variant:%d coat:%d speed:%d mode:%d count:%d", b.variant, b.coatMode, b.speed, b.mode, b.petCount)
	}
	if b.wheelEnabled != a.wheelEnabled || b.bidirectional != a.bidirectional || b.lang != a.lang {
		t.Fatalf("loaded flags = wheel:%v bidirectional:%v lang:%d", b.wheelEnabled, b.bidirectional, b.lang)
	}
	if b.positionMode != a.positionMode || b.overlayOffsetY != a.overlayOffsetY {
		t.Fatalf("loaded position = mode:%d offset:%d, want mode:%d offset:%d", b.positionMode, b.overlayOffsetY, a.positionMode, a.overlayOffsetY)
	}
	if b.laneMode != a.laneMode {
		t.Fatalf("loaded laneMode = %d, want %d", b.laneMode, a.laneMode)
	}
	if b.displayIndex != wantDisplayIndex {
		t.Fatalf("loaded displayIndex = %d, want %d", b.displayIndex, wantDisplayIndex)
	}
	if b.walkRangeStart != 15 || b.walkRangeEnd != 85 {
		t.Fatalf("loaded walk range = %d-%d, want 15-85", b.walkRangeStart, b.walkRangeEnd)
	}
	if b.nameLabels != a.nameLabels {
		t.Fatalf("loaded nameLabels = %v, want %v", b.nameLabels, a.nameLabels)
	}
	for i := 0; i < maxPetCount; i++ {
		if b.selectedCoats[i] != a.selectedCoats[i] {
			t.Fatalf("selectedCoats[%d] = %d, want %d", i, b.selectedCoats[i], a.selectedCoats[i])
		}
	}
	if b.petNames[0] != "モカ" || b.petNames[1] != "Sora" || b.petNames[2] != "Nagi" {
		t.Fatalf("loaded pet names = %#v", b.petNames[:3])
	}
}

func TestOverlayRectSupportsTaskbarAndScreenBottomModes(t *testing.T) {
	work := winRect(80, 0, 1920, 1040)
	screen := winRect(0, 0, 1920, 1080)
	a := &petApp{positionMode: positionTaskbarEdge, overlayOffsetY: 20, walkRangeEnd: 100}

	got := a.overlayRectFor(work, screen)
	if got.Left != work.Left || got.Right != work.Right {
		t.Fatalf("taskbar overlay x bounds = %+v, want work area x bounds %+v", got, work)
	}
	if wantTop := int32(1040 - sceneH + 20); got.Top != wantTop {
		t.Fatalf("taskbar overlay top = %d, want %d", got.Top, wantTop)
	}

	a.positionMode = positionScreenBottom
	a.overlayOffsetY = 0
	got = a.overlayRectFor(work, screen)
	if got.Left != screen.Left || got.Right != screen.Right {
		t.Fatalf("screen-bottom overlay x bounds = %+v, want screen x bounds %+v", got, screen)
	}
	if wantTop := int32(1080 - sceneH); got.Top != wantTop {
		t.Fatalf("screen-bottom overlay top = %d, want %d", got.Top, wantTop)
	}

	a.positionMode = positionTaskbarEdge
	a.overlayOffsetY = maxOverlayOffsetY
	got = a.overlayRectFor(work, screen)
	if got.Bottom != screen.Bottom {
		t.Fatalf("large downward offset bottom = %d, want clamped to screen bottom %d", got.Bottom, screen.Bottom)
	}
}

func TestOverlayRectAppliesWalkRange(t *testing.T) {
	work := winRect(100, 0, 1100, 1040)
	screen := winRect(100, 0, 1100, 1080)
	a := &petApp{
		positionMode:   positionTaskbarEdge,
		overlayOffsetY: 0,
		walkRangeStart: 20,
		walkRangeEnd:   80,
	}

	got := a.overlayRectFor(work, screen)
	if got.Left != 300 || got.Right != 900 {
		t.Fatalf("overlay walk range = left:%d right:%d, want 300-900", got.Left, got.Right)
	}
}

func TestDisplaySelectionSupportsMultiMonitorSpans(t *testing.T) {
	a := &petApp{displayScope: displayScopeSpan, displayIndex: 1, displaySpanEnd: 2}
	scope, start, end := a.normalizedDisplaySelection(3)
	if scope != displayScopeSpan || start != 1 || end != 2 {
		t.Fatalf("span selection = scope:%d start:%d end:%d, want span 1-2", scope, start, end)
	}

	a = &petApp{displayScope: displayScopeSpan, displayIndex: 2, displaySpanEnd: 2}
	scope, start, end = a.normalizedDisplaySelection(3)
	if scope != displayScopeSpan || start != 1 || end != 2 {
		t.Fatalf("single-point span = scope:%d start:%d end:%d, want adjacent span 1-2", scope, start, end)
	}

	a = &petApp{displayScope: displayScopeSingle, displayIndex: 2, displaySpanEnd: 0}
	scope, start, end = a.normalizedDisplaySelection(3)
	if scope != displayScopeSingle || start != 2 || end != 2 {
		t.Fatalf("single selection = scope:%d start:%d end:%d, want single 2", scope, start, end)
	}
}

func TestCombinedDisplayAreaAppliesWalkRangeAcrossSelectedMonitors(t *testing.T) {
	areas := []displayArea{
		{Work: winRect(0, 0, 1920, 1040), Screen: winRect(0, 0, 1920, 1080), Primary: true},
		{Work: winRect(1920, 0, 3840, 1080), Screen: winRect(1920, 0, 3840, 1080)},
		{Work: winRect(3840, 0, 5760, 1040), Screen: winRect(3840, 0, 5760, 1080)},
	}
	combined := combineDisplayAreas(areas[1:3])
	if combined.Screen.Left != 1920 || combined.Screen.Right != 5760 {
		t.Fatalf("combined screen = %+v, want 1920-5760", combined.Screen)
	}
	if combined.Work.Left != 1920 || combined.Work.Right != 5760 || combined.Work.Bottom != 1040 {
		t.Fatalf("combined work = %+v, want horizontal span with shared work bottom 1040", combined.Work)
	}

	a := &petApp{
		positionMode:   positionTaskbarEdge,
		overlayOffsetY: 0,
		walkRangeStart: 25,
		walkRangeEnd:   75,
	}
	got := a.overlayRectFor(combined.Work, combined.Screen)
	if got.Left != 2880 || got.Right != 4800 {
		t.Fatalf("multi-monitor walk range = left:%d right:%d, want 2880-4800", got.Left, got.Right)
	}
	if got.Top != 1040-sceneH {
		t.Fatalf("multi-monitor overlay top = %d, want %d", got.Top, 1040-sceneH)
	}
}

func TestOverlayRectHandlesNegativeMonitorCoordinates(t *testing.T) {
	work := winRect(-1920, 0, 0, 1040)
	screen := winRect(-1920, 0, 0, 1080)
	a := &petApp{
		positionMode:   positionScreenBottom,
		overlayOffsetY: 0,
		walkRangeStart: 50,
		walkRangeEnd:   100,
	}

	got := a.overlayRectFor(work, screen)
	if got.Left != -960 || got.Right != 0 {
		t.Fatalf("negative-coordinate overlay = left:%d right:%d, want -960-0", got.Left, got.Right)
	}
	if got.Top != 1080-sceneH {
		t.Fatalf("negative-coordinate overlay top = %d, want %d", got.Top, 1080-sceneH)
	}
}

func TestNormalizeWalkRangeKeepsMinimumSpan(t *testing.T) {
	start, end := normalizeWalkRange(52, 53)
	if end-start != minWalkRangeSpan {
		t.Fatalf("normalized span = %d, want %d", end-start, minWalkRangeSpan)
	}
	if start < 0 || end > 100 {
		t.Fatalf("normalized range escaped bounds: %d-%d", start, end)
	}
	start, end = normalizeWalkRange(95, 10)
	if start != 10 || end != 95 {
		t.Fatalf("reordered range = %d-%d, want 10-95", start, end)
	}
}

func TestMonitorAreasCanBeCalledRepeatedly(t *testing.T) {
	for i := 0; i < 2500; i++ {
		_ = monitorAreas()
	}
}

func TestDefaultOverlayOffsetSurvivesLegacySettings(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("APPDATA", configRoot)
	path := filepath.Join(configRoot, settingsDirName, settingsFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"petCount":2,"speed":3,"mode":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &petApp{positionMode: positionTaskbarEdge, overlayOffsetY: defaultOverlayOffsetY, laneMode: laneStaggered}
	if err := a.loadSettings(); err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if a.positionMode != positionTaskbarEdge || a.overlayOffsetY != defaultOverlayOffsetY {
		t.Fatalf("legacy position defaults = mode:%d offset:%d", a.positionMode, a.overlayOffsetY)
	}
	if a.laneMode != laneStaggered {
		t.Fatalf("legacy laneMode = %d, want staggered", a.laneMode)
	}
}

func winRect(left, top, right, bottom int32) win.RECT {
	return win.RECT{Left: left, Top: top, Right: right, Bottom: bottom}
}

func TestTrayCountCommandsSupportEveryVisibleCount(t *testing.T) {
	a := &petApp{
		sceneW:   900,
		speed:    3,
		coatMode: coatFixed,
		laneMode: laneAligned,
		petCount: 1,
	}

	for count := 1; count <= maxPetCount; count++ {
		if !a.handleMenuCommand(menuIDForPetCount(count)) {
			t.Fatalf("handleMenuCommand count %d returned false", count)
		}
		if a.petCount != count || len(a.pets) != count {
			t.Fatalf("count command %d left petCount=%d len=%d", count, a.petCount, len(a.pets))
		}
	}
}

func TestTemporaryVisibilityMenuLabelUsesCurrentStateAndLanguage(t *testing.T) {
	a := &petApp{lang: langJapanese}
	if got := a.temporaryVisibilityMenuLabel(); got != "一時的に非表示" {
		t.Fatalf("visible Japanese label = %q, want 一時的に非表示", got)
	}
	a.temporarilyHidden = true
	if got := a.temporaryVisibilityMenuLabel(); got != "再表示" {
		t.Fatalf("hidden Japanese label = %q, want 再表示", got)
	}

	a.lang = langEnglish
	if got := a.temporaryVisibilityMenuLabel(); got != "Show pets" {
		t.Fatalf("hidden English label = %q, want Show pets", got)
	}
	a.temporarilyHidden = false
	if got := a.temporaryVisibilityMenuLabel(); got != "Temporarily hide" {
		t.Fatalf("visible English label = %q, want Temporarily hide", got)
	}
}

func TestTemporaryHideMenuTogglesAndBlocksTyping(t *testing.T) {
	a := &petApp{
		sceneW:       1200,
		speed:        3,
		mode:         modeKeyboard,
		wheelEnabled: true,
		wheelX:       420,
		pets: []deguPet{{
			state:      stateIdle,
			stateTicks: 1,
			item:       noItem,
			carryKind:  noItem,
			dir:        1,
		}},
	}

	if !a.handleMenuCommand(menuToggleHidden) {
		t.Fatal("hide command returned false")
	}
	if !a.temporarilyHidden {
		t.Fatal("hide command did not set temporarilyHidden")
	}
	a.onTyping()
	if got := a.pets[0].state; got != stateIdle {
		t.Fatalf("hidden typing changed state to %v, want idle", got)
	}

	if !a.handleMenuCommand(menuToggleHidden) {
		t.Fatal("show command returned false")
	}
	if a.temporarilyHidden {
		t.Fatal("show command left temporarilyHidden set")
	}
	a.onTyping()
	if got := a.pets[0].state; got != stateWheel {
		t.Fatalf("visible typing changed state to %v, want wheel", got)
	}
}

func TestPetLaneModeControlsVerticalOffsets(t *testing.T) {
	a := &petApp{
		sceneW:   900,
		speed:    3,
		coatMode: coatFixed,
		laneMode: laneStaggered,
		petCount: 6,
	}
	a.setPetCount(6)
	for i, want := range []int{0, 5, 10, 0, 5, 10} {
		if got := a.pets[i].laneOffset; got != want {
			t.Fatalf("staggered laneOffset[%d] = %d, want %d", i, got, want)
		}
	}

	a.laneMode = laneAligned
	a.applyLaneOffsets()
	for i, pet := range a.pets {
		if pet.laneOffset != 0 {
			t.Fatalf("aligned laneOffset[%d] = %d, want 0", i, pet.laneOffset)
		}
	}
}

func TestSettingsTooltipsExplainLayoutControls(t *testing.T) {
	a := &petApp{}
	if got := a.settingsTooltipText(ctrlTabHome); got == "" || !strings.Contains(got, "まとめ") {
		t.Fatalf("home tab tooltip = %q, want overview explanation", got)
	}
	if got := a.settingsTooltipText(ctrlTabDisplay); got == "" || !strings.Contains(got, "タスクバー") {
		t.Fatalf("display tab tooltip = %q, want taskbar explanation", got)
	}
	if got := a.settingsTooltipText(ctrlDisplaySpan); got == "" || !strings.Contains(got, "複数") {
		t.Fatalf("display span tooltip = %q, want multi-display explanation", got)
	}
	if got := a.settingsTooltipText(ctrlRangeStartScroll); got == "" || !strings.Contains(got, "ここから") {
		t.Fatalf("range start tooltip = %q, want here-from explanation", got)
	}
	if got := a.settingsTooltipText(ctrlRangeEndScroll); got == "" || !strings.Contains(got, "ここまで") {
		t.Fatalf("range end tooltip = %q, want here-to explanation", got)
	}
	if got := a.settingsTooltipText(ctrlTabUpdates); got == "" || !strings.Contains(got, "バージョン") {
		t.Fatalf("updates tab tooltip = %q, want version explanation", got)
	}
	if got := a.settingsTooltipText(ctrlUpdateInstall); got == "" || !strings.Contains(got, "再起動") {
		t.Fatalf("update install tooltip = %q, want restart explanation", got)
	}
	if got := a.settingsTooltipText(ctrlLaneStaggered); got == "" || !strings.Contains(got, "0/5/10") {
		t.Fatalf("staggered tooltip = %q, want 0/5/10 explanation", got)
	}
	if got := a.settingsTooltipText(ctrlLaneAligned); got == "" || !strings.Contains(got, "同じ") {
		t.Fatalf("aligned tooltip = %q, want same-baseline explanation", got)
	}
	if got := a.settingsTooltipText(ctrlReset); got == "" || !strings.Contains(got, "初期値") {
		t.Fatalf("reset tooltip = %q, want initial-position explanation", got)
	}
}

func TestHomeSettingsSummariesShowUsefulState(t *testing.T) {
	a := &petApp{
		petCount:       5,
		coatMode:       coatSelected,
		nameLabels:     true,
		speed:          5,
		mode:           modeRandom,
		wheelEnabled:   true,
		bidirectional:  true,
		positionMode:   positionScreenBottom,
		overlayOffsetY: -4,
		walkRangeStart: 10,
		walkRangeEnd:   90,
	}
	if got := a.homePetDetail(); !strings.Contains(got, "個別") || !strings.Contains(got, "名前") {
		t.Fatalf("homePetDetail() = %q, want coat and name state", got)
	}
	if got := a.homeMotionSummary(); !strings.Contains(got, "ランダム") || !strings.Contains(got, "はやい") {
		t.Fatalf("homeMotionSummary() = %q, want mode and speed", got)
	}
	if got := a.homeMotionDetail(); !strings.Contains(got, "回し車") || !strings.Contains(got, "左右") {
		t.Fatalf("homeMotionDetail() = %q, want wheel and turn state", got)
	}
	if got := a.homeDisplayDetail(); !strings.Contains(got, "10%") || !strings.Contains(got, "-4 px") {
		t.Fatalf("homeDisplayDetail() = %q, want range and offset", got)
	}
}

func TestPetVariantRectsFitTenPetsInSettingsWindow(t *testing.T) {
	seen := map[[4]int]bool{}
	for i := 0; i < maxPetCount; i++ {
		numberRect, buttonRect := settingsPetVariantRects(i)
		if buttonRect.Right > 708 || buttonRect.Bottom > 502 {
			t.Fatalf("pet variant button %d rect %+v overflows selected-coats panel", i, buttonRect)
		}
		if numberRect.Left < 238 || buttonRect.Left <= numberRect.Right {
			t.Fatalf("pet variant %d number/button rects overlap or escape: number=%+v button=%+v", i, numberRect, buttonRect)
		}
		key := [4]int{int(buttonRect.Left), int(buttonRect.Top), int(buttonRect.Right), int(buttonRect.Bottom)}
		if seen[key] {
			t.Fatalf("pet variant button %d duplicates another rect: %+v", i, buttonRect)
		}
		seen[key] = true
	}
}

func TestPetNameRectsFitTenPetsWithCoatPicker(t *testing.T) {
	for i := 0; i < maxPetCount; i++ {
		numberRect, nameRect := settingsPetNameRects(i)
		_, coatRect := settingsPetVariantRects(i)
		if nameRect.Right >= coatRect.Left {
			t.Fatalf("pet %d name rect overlaps coat rect: name=%+v coat=%+v", i, nameRect, coatRect)
		}
		if numberRect.Left < 238 || nameRect.Left <= numberRect.Right || coatRect.Right > 708 || nameRect.Bottom > 502 {
			t.Fatalf("pet %d name/coat row escapes panel: number=%+v name=%+v coat=%+v", i, numberRect, nameRect, coatRect)
		}
	}
}

func TestUpdateVersionComparison(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v1.2.0", "v1.1.9", true},
		{"v1.2.0", "1.2.0", false},
		{"v1.2.0", "v1.3.0", false},
		{"v2.0.0", "dev", true},
		{"v2.0.0", "pages-abc123", true},
		{"not-semver", "v1.0.0", false},
	}
	for _, tt := range tests {
		if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
			t.Fatalf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestSelectUpdateAssetFindsWindowsZip(t *testing.T) {
	rel := &githubRelease{Assets: []githubReleaseAsset{
		{Name: "notes.txt", BrowserDownloadURL: "https://example.test/notes.txt"},
		{Name: "DeguDesktop-windows-amd64.zip", BrowserDownloadURL: "https://example.test/app.zip"},
		{Name: "DeguDesktop-windows-386.zip", BrowserDownloadURL: "https://example.test/app-x86.zip"},
	}}
	asset := selectUpdateAsset(rel, "amd64")
	if asset == nil || asset.BrowserDownloadURL != "https://example.test/app.zip" {
		t.Fatalf("selectUpdateAsset(amd64) = %+v", asset)
	}
	asset = selectUpdateAsset(rel, "386")
	if asset == nil || asset.BrowserDownloadURL != "https://example.test/app-x86.zip" {
		t.Fatalf("selectUpdateAsset(386) = %+v", asset)
	}
}

func TestUpdateSettingsSummariesExplainStates(t *testing.T) {
	oldVersion := appVersion
	appVersion = "v1.0.0"
	t.Cleanup(func() { appVersion = oldVersion })

	a := &petApp{}
	if got := a.updateStatusSummary(); got == "" || !strings.Contains(got, "まだ") {
		t.Fatalf("initial update summary = %q, want not-checked guidance", got)
	}
	if got := a.updatePackageSummary(); got != updateAssetName(runtime.GOARCH) {
		t.Fatalf("initial package summary = %q, want asset name", got)
	}

	a.update.checking.Store(true)
	if got := a.updateStatusSummary(); got == "" || !strings.Contains(got, "確認中") {
		t.Fatalf("checking update summary = %q, want checking state", got)
	}
	a.update.checking.Store(false)

	a.setUpdateResult(&githubRelease{
		TagName: "v1.2.0",
		Assets: []githubReleaseAsset{{
			Name:               updateAssetName(runtime.GOARCH),
			BrowserDownloadURL: "https://example.test/app.zip",
			Size:               2 * 1024 * 1024,
		}},
	}, nil)
	if !a.hasInstallableUpdate() {
		t.Fatalf("hasInstallableUpdate() = false, want true")
	}
	if got := a.updateStatusSummary(); got == "" || !strings.Contains(got, "v1.2.0") {
		t.Fatalf("available update summary = %q, want release tag", got)
	}
	if got := a.updatePackageSummary(); got == "" || !strings.Contains(got, "2.0 MB") {
		t.Fatalf("package summary = %q, want formatted size", got)
	}

	a.setUpdateResult(nil, fmt.Errorf("network down"))
	if got := a.updateStatusSummary(); got == "" || !strings.Contains(got, "失敗") {
		t.Fatalf("error update summary = %q, want failure state", got)
	}
	if got := a.updateStatusDetail(); got != "network down" {
		t.Fatalf("error detail = %q, want raw error", got)
	}
}

func TestFetchLatestReleaseFromUpdateAPI(t *testing.T) {
	var sawUserAgent bool
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		sawUserAgent = strings.HasPrefix(r.Header.Get("User-Agent"), "DeguDesktop/")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v9.9.9",
			"html_url": "https://example.test/release",
			"draft": false,
			"prerelease": false,
			"assets": [
				{"name": "DeguDesktop-windows-amd64.zip", "browser_download_url": "https://example.test/app.zip", "size": 123}
			]
		}`))
	}))
	defer server.Close()
	oldURL := updateAPIURL
	updateAPIURL = server.URL + "/latest"
	t.Cleanup(func() { updateAPIURL = oldURL })

	rel, err := fetchLatestRelease()
	if err != nil {
		t.Fatalf("fetchLatestRelease() error = %v", err)
	}
	if gotPath != "/latest" {
		t.Fatalf("path = %s, want /latest", gotPath)
	}
	if !sawUserAgent {
		t.Fatalf("fetchLatestRelease did not send the DeguDesktop user agent")
	}
	if rel.TagName != "v9.9.9" || len(rel.Assets) != 1 {
		t.Fatalf("release = %+v, want tag v9.9.9 with one asset", rel)
	}
}

func TestFetchLatestReleaseRejectsHTTPErrorAndDraft(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "http error", status: http.StatusInternalServerError, body: `server error`},
		{name: "draft", status: http.StatusOK, body: `{"tag_name":"v9.9.9","draft":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			oldURL := updateAPIURL
			updateAPIURL = server.URL
			t.Cleanup(func() { updateAPIURL = oldURL })

			if rel, err := fetchLatestRelease(); err == nil || rel != nil {
				t.Fatalf("fetchLatestRelease() = (%+v, %v), want error", rel, err)
			}
		})
	}
}

func TestDownloadFileAndExtractUpdateExe(t *testing.T) {
	var zipBytes bytes.Buffer
	zw := zip.NewWriter(&zipBytes)
	exe, err := zw.Create("DeguDesktop.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exe.Write([]byte("fake exe payload")); err != nil {
		t.Fatal(err)
	}
	readme, err := zw.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readme.Write([]byte("readme")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipBytes.Bytes())
	}))
	defer server.Close()

	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "update.zip")
	if err := downloadFile(server.URL, zipPath); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	exePath, err := extractUpdateExe(zipPath, tmp)
	if err != nil {
		t.Fatalf("extractUpdateExe() error = %v", err)
	}
	data, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read extracted exe: %v", err)
	}
	if string(data) != "fake exe payload" {
		t.Fatalf("extracted exe payload = %q", data)
	}
}

func TestTurnStateUsesGeneratedTurnFrames(t *testing.T) {
	if got := currentFrame(stateTurn, 0); got != turnStart {
		t.Fatalf("turn frame 0 = %d, want %d", got, turnStart)
	}
	if got := currentFrame(stateTurn, turnTicks-1); got != turnStart+turnFrames-1 {
		t.Fatalf("turn final active frame = %d, want %d", got, turnStart+turnFrames-1)
	}
	if got := currentFrame(stateTurn, turnTicks+10); got != turnStart+turnFrames-1 {
		t.Fatalf("turn frame after duration = %d, want held final frame %d", got, turnStart+turnFrames-1)
	}
}

func TestTurnDrawDirectionMirrorsOnlyLeftToRightTurns(t *testing.T) {
	if got := turnDrawDirection(1, -1); got != 1 {
		t.Fatalf("right-to-left turn draw direction = %d, want 1", got)
	}
	if got := turnDrawDirection(-1, 1); got != -1 {
		t.Fatalf("left-to-right turn draw direction = %d, want -1", got)
	}
}

func TestSetBidirectionalOffNormalizesPets(t *testing.T) {
	a := &petApp{
		bidirectional: true,
		speed:         3,
		pets: []deguPet{
			{state: stateTurn, dir: -1, nextDir: -1, item: noItem},
			{state: stateWalk, dir: -1, nextDir: -1, item: noItem},
		},
	}

	a.setBidirectional(false)
	if a.bidirectional {
		t.Fatalf("bidirectional stayed enabled")
	}
	for i, pet := range a.pets {
		if pet.dir != 1 || pet.nextDir != 1 {
			t.Fatalf("pet %d direction = (%d,%d), want (1,1)", i, pet.dir, pet.nextDir)
		}
		if pet.state == stateTurn {
			t.Fatalf("pet %d remained in stateTurn", i)
		}
	}
}

func TestFixedCoatModeRefreshesAllPets(t *testing.T) {
	a := &petApp{
		variant: 2,
		pets: []deguPet{
			{variant: 0},
			{variant: 1},
			{variant: 3},
		},
	}

	a.setCoatMode(coatFixed)

	for i, pet := range a.pets {
		if pet.variant != 2 {
			t.Fatalf("pet %d variant = %d, want fixed variant 2", i, pet.variant)
		}
	}
}

func TestSelectedCoatModeUsesPerPetChoices(t *testing.T) {
	a := &petApp{
		selectedCoats: [maxPetCount]int{0, 3, 5, 7, 9},
		pets: []deguPet{
			{variant: 0},
			{variant: 0},
			{variant: 0},
		},
	}

	a.setCoatMode(coatSelected)

	for i, want := range []int{0, 3, 5} {
		if got := a.pets[i].variant; got != want {
			t.Fatalf("pet %d variant = %d, want %d", i, got, want)
		}
	}
	a.setSelectedVariant(1, 8)
	if got := a.pets[1].variant; got != 8 {
		t.Fatalf("selected variant update = %d, want 8", got)
	}
}

func TestRandomCoatModeAssignsValidVariants(t *testing.T) {
	a := &petApp{coatMode: coatRandom}
	for i := 0; i < 100; i++ {
		got := a.variantForIndex(i)
		if got < 0 || got >= len(variants) {
			t.Fatalf("random variant = %d, want 0..%d", got, len(variants)-1)
		}
	}
}

func TestPetAtScenePointFindsTopmostPet(t *testing.T) {
	a := &petApp{
		sceneW: 800,
		pets: []deguPet{
			{x: 100, laneOffset: 0, state: stateWalk},
			{x: 110, laneOffset: 0, state: stateIdle},
		},
	}

	got := a.petAtScenePoint(132, sceneH-spriteH+24)
	if got != 1 {
		t.Fatalf("petAtScenePoint overlap = %d, want topmost pet 1", got)
	}
	if got := a.petAtScenePoint(4, 4); got != -1 {
		t.Fatalf("petAtScenePoint outside = %d, want -1", got)
	}
}

func TestShowPetReactionRefreshesExistingBubble(t *testing.T) {
	a := &petApp{
		pets: []deguPet{{state: stateWalk}},
		reactions: []petReaction{
			{pet: 0, kind: 1, ticks: 3},
		},
	}

	a.showPetReaction(0)
	if len(a.reactions) != 1 {
		t.Fatalf("reaction count = %d, want 1 refreshed reaction", len(a.reactions))
	}
	if a.reactions[0].ticks != reactionTicks {
		t.Fatalf("reaction ticks = %d, want %d", a.reactions[0].ticks, reactionTicks)
	}
}

func TestTickReactionsDropsExpiredAndInvalid(t *testing.T) {
	a := &petApp{
		pets: []deguPet{{state: stateWalk}},
		reactions: []petReaction{
			{pet: 0, ticks: 1},
			{pet: 3, ticks: 5},
		},
	}

	a.tickReactions()
	if len(a.reactions) != 0 {
		t.Fatalf("remaining reactions = %d, want 0", len(a.reactions))
	}
}
