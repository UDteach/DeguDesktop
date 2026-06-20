//go:build darwin

package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinHorizontalMotionFramesUseStableSequence(t *testing.T) {
	allowed := map[int]bool{
		walkStart:     true,
		walkStart + 1: true,
		walkStart + 3: true,
	}

	for _, delay := range []int{1, 2} {
		for tick := 0; tick < 64; tick++ {
			got := seqFrameFrom(walkFrameSeq, tick, delay)
			if !allowed[got] {
				t.Fatalf("seqFrameFrom(walkFrameSeq, %d, %d) = %d, want stable horizontal walk frame", tick, delay, got)
			}
		}
	}
}

func TestDarwinTickWalksInDirectionWithStableFrame(t *testing.T) {
	allowed := map[int]bool{
		walkStart:     true,
		walkStart + 1: true,
		walkStart + 3: true,
	}
	a := &darwinPetApp{
		sceneW: 500,
		mode:   darwinModeRandom,
		pets: []darwinPet{
			{x: 20, dir: -1, speed: 2, nextPause: 50},
			{x: 80, dir: 1, speed: 2, nextPause: 50},
		},
	}

	a.tickPets()
	if got := a.pets[0].x; got != 18 {
		t.Fatalf("left-moving pet x = %d, want 18", got)
	}
	if got := a.pets[1].x; got != 82 {
		t.Fatalf("right-moving pet x = %d, want 82", got)
	}
	for i, pet := range a.pets {
		if !allowed[pet.frame] {
			t.Fatalf("pet %d frame = %d, want stable horizontal walk frame", i, pet.frame)
		}
	}
}

func TestDarwinSeqFrameFromHandlesEmptyAndBadDivisor(t *testing.T) {
	if got := seqFrameFrom(nil, 12, 2); got != idleStart {
		t.Fatalf("seqFrameFrom(nil) = %d, want %d", got, idleStart)
	}
	seq := []int{7, 9}
	if got := seqFrameFrom(seq, 3, 0); got != 9 {
		t.Fatalf("seqFrameFrom with zero divisor = %d, want 9", got)
	}
}

func TestDarwinSettingsUpdateRuntimeStateAndPersist(t *testing.T) {
	oldSettingsPath := darwinSettingsPath
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	darwinSettingsPath = func() (string, error) {
		return settingsPath, nil
	}
	t.Cleanup(func() {
		darwinSettingsPath = oldSettingsPath
	})

	a := &darwinPetApp{
		sceneW:        640,
		speed:         darwinSpeedNormal,
		petCount:      5,
		mode:          darwinModeRandom,
		coatMode:      darwinCoatRandom,
		selectedCoats: defaultDarwinSelectedCoats(),
		wheelEnabled:  true,
	}
	a.resetPets()

	a.setSpeed(darwinSpeedFast)
	if got := a.speed; got != darwinSpeedFast {
		t.Fatalf("speed = %d, want %d", got, darwinSpeedFast)
	}
	if got := a.pets[0].speed; got != 4 {
		t.Fatalf("first pet speed = %d, want 4", got)
	}
	if got := a.pets[1].speed; got != 5 {
		t.Fatalf("second pet speed = %d, want 5", got)
	}

	a.setPetCount(3)
	if got := len(a.pets); got != 3 {
		t.Fatalf("pet count = %d, want 3", got)
	}
	a.setPetCount(9)
	if got := len(a.pets); got != 9 {
		t.Fatalf("pet count = %d, want 9", got)
	}
	a.setPetCount(3)
	a.setMode(int(darwinModeKeyboard))
	if got := a.mode; got != darwinModeKeyboard {
		t.Fatalf("mode = %d, want %d", got, darwinModeKeyboard)
	}
	a.setCoatMode(int(darwinCoatSelected))
	a.setSelectedVariant(1, 7)
	if a.coatMode != darwinCoatSelected || a.selectedCoats[1] != 7 || a.pets[1].variant != 7 {
		t.Fatalf("selected coat state = mode:%d selected:%d pet:%d, want selected mode and variant 7", a.coatMode, a.selectedCoats[1], a.pets[1].variant)
	}

	a.keyHold = wheelKeyHold
	a.setWheelEnabled(false)
	if a.wheelEnabled {
		t.Fatal("wheelEnabled = true, want false")
	}
	if got := a.keyHold; got != 0 {
		t.Fatalf("keyHold = %d, want 0 after disabling keyboard reaction", got)
	}
	a.nameLabels = true
	a.setPetName(0, "  モカ  ")
	a.setPetName(2, "abcdefghijklmnopqrstuvwxyz")

	a.saveSettings()
	b := &darwinPetApp{
		speed:         darwinSpeedNormal,
		petCount:      5,
		mode:          darwinModeRandom,
		coatMode:      darwinCoatRandom,
		selectedCoats: defaultDarwinSelectedCoats(),
		wheelEnabled:  true,
	}
	b.loadSettings()
	if b.speed != darwinSpeedFast || b.petCount != 3 || b.mode != darwinModeKeyboard || b.coatMode != darwinCoatSelected || b.selectedCoats[1] != 7 || b.wheelEnabled {
		t.Fatalf("loaded settings = speed:%d count:%d mode:%d coat:%d selected:%d wheel:%v, want speed:%d count:3 keyboard selected variant 7 wheel:false", b.speed, b.petCount, b.mode, b.coatMode, b.selectedCoats[1], b.wheelEnabled, darwinSpeedFast)
	}
	if !b.nameLabels || b.petNames[0] != "モカ" || b.petNames[2] != "abcdefghijklmnopqrstuvwx" {
		t.Fatalf("loaded names = labels:%v names:%#v", b.nameLabels, b.petNames[:3])
	}
	if got := b.petDisplayName(1); got != "デグー2" {
		t.Fatalf("default display name = %q, want デグー2", got)
	}
}

func TestDarwinPetCountSupportsEveryVisibleCount(t *testing.T) {
	a := &darwinPetApp{
		sceneW:        900,
		speed:         darwinSpeedNormal,
		petCount:      5,
		mode:          darwinModeRandom,
		coatMode:      darwinCoatRandom,
		selectedCoats: defaultDarwinSelectedCoats(),
		wheelEnabled:  true,
	}
	for count := 1; count <= maxPetCount; count++ {
		a.setPetCount(count)
		if a.petCount != count || len(a.pets) != count {
			t.Fatalf("setPetCount(%d) = state:%d pets:%d", count, a.petCount, len(a.pets))
		}
	}
}

func TestDarwinClickReactionHitTestsPet(t *testing.T) {
	a := &darwinPetApp{
		sceneW:       400,
		wheelEnabled: true,
		pets: []darwinPet{
			{x: 50, lane: 0},
			{x: 70, lane: 7},
		},
	}

	if index := a.petAtScenePoint(90, 40); index != 1 {
		t.Fatalf("petAtScenePoint overlapping pets = %d, want topmost pet 1", index)
	}
	if !a.addClickReaction(90, 40) {
		t.Fatal("addClickReaction inside pet = false, want true")
	}
	if len(a.reactions) != 1 || a.reactions[0].pet != 1 || a.reactions[0].ticks != reactionTicks {
		t.Fatalf("reaction = %#v", a.reactions)
	}
	a.reactions[0].ticks = 10
	if !a.addClickReaction(90, 40) {
		t.Fatal("second addClickReaction inside pet = false, want true")
	}
	if len(a.reactions) != 1 || a.reactions[0].ticks != reactionTicks {
		t.Fatalf("updated reaction = %#v", a.reactions)
	}
	if a.addClickReaction(3, 3) {
		t.Fatal("addClickReaction outside pet = true, want false")
	}

	a.keyHold = 1
	a.pets = []darwinPet{{x: 50, lane: 0}}
	if index := a.petAtScenePoint(90, 40); index != -1 {
		t.Fatalf("wheel runner hit = %d, want ignored", index)
	}
}

func TestDarwinPartialSettingsKeepRuntimeDefaults(t *testing.T) {
	oldSettingsPath := darwinSettingsPath
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	darwinSettingsPath = func() (string, error) {
		return settingsPath, nil
	}
	t.Cleanup(func() {
		darwinSettingsPath = oldSettingsPath
	})
	if err := os.WriteFile(settingsPath, []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &darwinPetApp{
		speed:        darwinSpeedNormal,
		petCount:     5,
		wheelEnabled: true,
	}
	a.loadSettings()
	if a.speed != darwinSpeedNormal || a.petCount != 5 || !a.wheelEnabled {
		t.Fatalf("settings defaults = speed:%d count:%d wheel:%v, want speed:%d count:5 wheel:true", a.speed, a.petCount, a.wheelEnabled, darwinSpeedNormal)
	}
}

func TestDarwinDrawFacingImageMirrorsNegativeDirection(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	src.SetRGBA(0, 0, red)
	src.SetRGBA(1, 0, blue)

	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	drawFacingImage(dst, src, dst.Bounds(), 1)
	if got := dst.RGBAAt(0, 0); got != red {
		t.Fatalf("drawFacingImage positive left pixel = %#v, want %#v", got, red)
	}
	if got := dst.RGBAAt(1, 0); got != blue {
		t.Fatalf("drawFacingImage positive right pixel = %#v, want %#v", got, blue)
	}

	dst = image.NewRGBA(image.Rect(0, 0, 2, 1))
	drawFacingImage(dst, src, dst.Bounds(), -1)
	if got := dst.RGBAAt(0, 0); got != blue {
		t.Fatalf("drawFacingImage negative left pixel = %#v, want %#v", got, blue)
	}
	if got := dst.RGBAAt(1, 0); got != red {
		t.Fatalf("drawFacingImage negative right pixel = %#v, want %#v", got, red)
	}
}
