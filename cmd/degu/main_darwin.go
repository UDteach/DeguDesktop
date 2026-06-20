//go:build darwin

package main

/*
#cgo darwin CFLAGS: -fblocks
#cgo darwin LDFLAGS: -framework Cocoa
#include "darwin_cocoa.h"
*/
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	appassets "degu-desktop/assets"
)

const (
	frameW        = 96
	frameH        = 64
	frameCount    = 56
	spriteW       = frameW
	spriteH       = frameH
	sceneH        = 92
	wheelSize     = 72
	timerInterval = 55
	maxPetCount   = 10
	wheelKeyHold  = 18
	reactionTicks = 54
	maxPetNameLen = 24
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
	settingsDirName  = "DeguDesktop"
	settingsFileName = "settings.json"

	darwinSpeedSlow   = 2
	darwinSpeedNormal = 3
	darwinSpeedFast   = 5
)

type darwinBehaviorMode int

const (
	darwinModeKeyboard darwinBehaviorMode = iota
	darwinModeRandom
)

type darwinCoatMode int

const (
	darwinCoatFixed darwinCoatMode = iota
	darwinCoatSelected
	darwinCoatRandom
)

var (
	idleFrameSeq   = []int{idleStart, idleStart + 1, idleStart + 3, idleStart + 1}
	walkFrameSeq   = []int{walkStart, walkStart + 1, walkStart + 3, walkStart + 1}
	nibbleFrameSeq = []int{nibbleStart, nibbleStart + 1, nibbleStart + 2, nibbleStart + 1}
	hopFrameSeq    = []int{hopStart, hopStart + 1, hopStart + 2, hopStart + 3}
)

type darwinCoatVariant struct {
	ID      string
	LabelJA string
}

var darwinVariants = []darwinCoatVariant{
	{ID: "wild_agouti", LabelJA: "野生色 / アグーチ"},
	{ID: "black", LabelJA: "ブラック"},
	{ID: "blue", LabelJA: "ブルー（青みグレー）"},
	{ID: "gray", LabelJA: "グレー"},
	{ID: "white_cream", LabelJA: "ホワイト / クリーム"},
	{ID: "sand_champagne", LabelJA: "サンド / シャンパン"},
	{ID: "chocolate", LabelJA: "チョコレート"},
	{ID: "black_pied", LabelJA: "ブラックパイド"},
	{ID: "agouti_pied", LabelJA: "アグーチパイド"},
	{ID: "blue_pied", LabelJA: "ブルーパイド（青みグレー）"},
	{ID: "cream_pied", LabelJA: "クリームパイド"},
}

var appVersion = "dev"

var darwinApp *darwinPetApp

type darwinPetApp struct {
	mu            sync.Mutex
	sceneW        int
	tick          int
	keyHold       int
	speed         int
	petCount      int
	mode          darwinBehaviorMode
	coatMode      darwinCoatMode
	variant       int
	selectedCoats [maxPetCount]int
	petNames      [maxPetCount]string
	nameLabels    bool
	wheelEnabled  bool
	reactions     []darwinReaction
	frames        map[string][]*image.RGBA
	wheel         *image.RGBA
	pets          []darwinPet
}

type darwinPet struct {
	x         int
	lane      int
	dir       int
	speed     int
	frame     int
	variant   int
	nextPause int
	pause     int
}

type darwinReaction struct {
	pet   int
	kind  int
	ticks int
}

type darwinSettings struct {
	Version       int      `json:"version"`
	Variant       *int     `json:"variant,omitempty"`
	CoatMode      *int     `json:"coatMode,omitempty"`
	SelectedCoats []int    `json:"selectedCoats,omitempty"`
	Speed         int      `json:"speed"`
	Mode          *int     `json:"mode,omitempty"`
	PetCount      int      `json:"petCount"`
	WheelEnabled  *bool    `json:"wheelEnabled,omitempty"`
	NameLabels    bool     `json:"nameLabels"`
	PetNames      []string `json:"petNames,omitempty"`
}

var darwinSettingsPath = defaultDarwinSettingsPath

func main() {
	runtime.LockOSThread()
	rand.Seed(time.Now().UnixNano())
	darwinApp = newDarwinPetApp()
	icon := darwinApp.statusIconPNG()
	if len(icon) > 0 {
		C.startDeguApp(C.int(sceneH), (*C.uchar)(unsafe.Pointer(&icon[0])), C.int(len(icon)))
		runtime.KeepAlive(icon)
		return
	}
	C.startDeguApp(C.int(sceneH), nil, 0)
}

func newDarwinPetApp() *darwinPetApp {
	app := &darwinPetApp{
		sceneW:        900,
		speed:         darwinSpeedNormal,
		petCount:      5,
		mode:          darwinModeRandom,
		coatMode:      darwinCoatRandom,
		selectedCoats: defaultDarwinSelectedCoats(),
		wheelEnabled:  true,
		frames:        loadDarwinSprites(),
		wheel:         loadDarwinWheel(),
	}
	app.loadSettings()
	return app
}

//export goDeguSetSceneWidth
func goDeguSetSceneWidth(width C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	darwinApp.setSceneWidth(int(width))
}

//export goDeguKeyDown
func goDeguKeyDown() {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.keyHold = wheelKeyHold
	darwinApp.mu.Unlock()
}

//export goDeguSetSpeed
func goDeguSetSpeed(speed C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setSpeed(int(speed))
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetPetCount
func goDeguSetPetCount(count C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setPetCount(int(count))
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetWheelEnabled
func goDeguSetWheelEnabled(enabled C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setWheelEnabled(enabled != 0)
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetMode
func goDeguSetMode(mode C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setMode(int(mode))
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetCoatMode
func goDeguSetCoatMode(mode C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setCoatMode(int(mode))
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetVariant
func goDeguSetVariant(variant C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setFixedVariant(int(variant))
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetSelectedCoat
func goDeguSetSelectedCoat(index C.int, variant C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.setSelectedVariant(int(index), int(variant))
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetNameLabels
func goDeguSetNameLabels(enabled C.int) {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.nameLabels = enabled != 0
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguSetPetName
func goDeguSetPetName(index C.int, value *C.char) {
	if darwinApp == nil {
		return
	}
	name := ""
	if value != nil {
		name = C.GoString(value)
	}
	darwinApp.mu.Lock()
	darwinApp.setPetName(int(index), name)
	darwinApp.saveSettings()
	darwinApp.mu.Unlock()
}

//export goDeguClick
func goDeguClick(x C.int, y C.int) C.int {
	if darwinApp == nil {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	if darwinApp.addClickReaction(int(x), int(y)) {
		return C.int(1)
	}
	return C.int(0)
}

//export goDeguPetAt
func goDeguPetAt(x C.int, y C.int) C.int {
	if darwinApp == nil {
		return C.int(-1)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	return C.int(darwinApp.petAtScenePoint(int(x), int(y)))
}

//export goDeguGetSpeed
func goDeguGetSpeed() C.int {
	if darwinApp == nil {
		return C.int(darwinSpeedNormal)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	return C.int(darwinApp.speed)
}

//export goDeguGetPetCount
func goDeguGetPetCount() C.int {
	if darwinApp == nil {
		return C.int(5)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	return C.int(darwinApp.petCount)
}

//export goDeguGetWheelEnabled
func goDeguGetWheelEnabled() C.int {
	if darwinApp == nil {
		return C.int(1)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	if darwinApp.wheelEnabled {
		return C.int(1)
	}
	return C.int(0)
}

//export goDeguGetMode
func goDeguGetMode() C.int {
	if darwinApp == nil {
		return C.int(darwinModeRandom)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	return C.int(darwinApp.mode)
}

//export goDeguGetCoatMode
func goDeguGetCoatMode() C.int {
	if darwinApp == nil {
		return C.int(darwinCoatRandom)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	return C.int(darwinApp.coatMode)
}

//export goDeguGetVariant
func goDeguGetVariant() C.int {
	if darwinApp == nil {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	return C.int(darwinApp.variant)
}

//export goDeguGetSelectedCoat
func goDeguGetSelectedCoat(index C.int) C.int {
	if darwinApp == nil {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	i := int(index)
	if i < 0 || i >= maxPetCount {
		return C.int(0)
	}
	return C.int(darwinApp.selectedCoats[i])
}

//export goDeguGetVariantCount
func goDeguGetVariantCount() C.int {
	return C.int(len(darwinVariants))
}

//export goDeguGetNameLabels
func goDeguGetNameLabels() C.int {
	if darwinApp == nil {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	if darwinApp.nameLabels {
		return C.int(1)
	}
	return C.int(0)
}

//export goDeguCopyPetName
func goDeguCopyPetName(index C.int, buffer *C.char, length C.int) C.int {
	if darwinApp == nil || buffer == nil || length <= 0 {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	i := int(index)
	if i < 0 || i >= maxPetCount {
		return C.int(0)
	}
	name := sanitizeDarwinPetName(darwinApp.petNames[i])
	data := []byte(name)
	maxLen := int(length) - 1
	if maxLen < 0 {
		return C.int(0)
	}
	if len(data) > maxLen {
		data = data[:maxLen]
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(buffer)), int(length))
	copy(dst, data)
	dst[len(data)] = 0
	return C.int(len(data))
}

//export goDeguGetPetDrawX
func goDeguGetPetDrawX(index C.int) C.int {
	if darwinApp == nil {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	i := int(index)
	if i < 0 || i >= len(darwinApp.pets) {
		return C.int(0)
	}
	return C.int(darwinApp.pets[i].x)
}

//export goDeguGetPetDrawY
func goDeguGetPetDrawY(index C.int) C.int {
	if darwinApp == nil {
		return C.int(0)
	}
	darwinApp.mu.Lock()
	defer darwinApp.mu.Unlock()
	i := int(index)
	if i < 0 || i >= len(darwinApp.pets) {
		return C.int(0)
	}
	return C.int(sceneH - spriteH - darwinApp.pets[i].lane)
}

//export goDeguTick
func goDeguTick() {
	if darwinApp == nil {
		return
	}
	darwinApp.mu.Lock()
	darwinApp.tickPets()
	img := darwinApp.render()
	darwinApp.mu.Unlock()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}
	data := buf.Bytes()
	if len(data) == 0 {
		return
	}
	C.updateDeguImage((*C.uchar)(unsafe.Pointer(&data[0])), C.int(len(data)), C.int(img.Bounds().Dx()), C.int(img.Bounds().Dy()))
	runtime.KeepAlive(data)
}

func defaultDarwinSettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsDirName, settingsFileName), nil
}

func (a *darwinPetApp) loadSettings() {
	path, err := darwinSettingsPath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var settings darwinSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}
	if settings.Speed != 0 {
		a.speed = normalizeDarwinSpeed(settings.Speed)
	}
	if settings.PetCount != 0 {
		a.petCount = normalizeDarwinPetCount(settings.PetCount)
	}
	if settings.Variant != nil {
		a.variant = normalizeDarwinVariant(*settings.Variant)
	}
	if settings.CoatMode != nil {
		a.coatMode = normalizeDarwinCoatMode(*settings.CoatMode)
	}
	if len(settings.SelectedCoats) > 0 {
		for i, variant := range settings.SelectedCoats {
			if i >= maxPetCount {
				break
			}
			a.selectedCoats[i] = normalizeDarwinVariant(variant)
		}
	}
	if settings.Mode != nil {
		a.mode = normalizeDarwinMode(*settings.Mode)
	}
	if settings.WheelEnabled != nil {
		a.wheelEnabled = *settings.WheelEnabled
	}
	a.nameLabels = settings.NameLabels
	for i, name := range settings.PetNames {
		if i >= maxPetCount {
			break
		}
		a.petNames[i] = sanitizeDarwinPetName(name)
	}
}

func (a *darwinPetApp) saveSettings() {
	path, err := darwinSettingsPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	wheelEnabled := a.wheelEnabled
	selectedCoats := make([]int, maxPetCount)
	copy(selectedCoats, a.selectedCoats[:])
	petNames := make([]string, maxPetCount)
	for i := range a.petNames {
		petNames[i] = sanitizeDarwinPetName(a.petNames[i])
	}
	variant := normalizeDarwinVariant(a.variant)
	coatMode := int(normalizeDarwinCoatMode(int(a.coatMode)))
	mode := int(normalizeDarwinMode(int(a.mode)))
	settings := darwinSettings{
		Version:       1,
		Variant:       &variant,
		CoatMode:      &coatMode,
		SelectedCoats: selectedCoats,
		Speed:         normalizeDarwinSpeed(a.speed),
		Mode:          &mode,
		PetCount:      normalizeDarwinPetCount(a.petCount),
		WheelEnabled:  &wheelEnabled,
		NameLabels:    a.nameLabels,
		PetNames:      petNames,
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0o644)
}

func (a *darwinPetApp) setSceneWidth(width int) {
	if width < 320 {
		width = 320
	}
	a.sceneW = width
	if len(a.pets) == 0 {
		a.resetPets()
		return
	}
	for i := range a.pets {
		a.pets[i].x = clamp(a.pets[i].x, 0, max(0, a.sceneW-spriteW))
	}
}

func (a *darwinPetApp) resetPets() {
	count := normalizeDarwinPetCount(a.petCount)
	spacing := max(spriteW+28, a.sceneW/(count+1))
	a.pets = make([]darwinPet, count)
	for i := range a.pets {
		dir := 1
		if i%2 == 1 {
			dir = -1
		}
		a.pets[i] = darwinPet{
			x:         clamp(spacing*(i+1)-spriteW/2, 0, max(0, a.sceneW-spriteW)),
			lane:      (i % 3) * 7,
			dir:       dir,
			speed:     a.petSpeed(i),
			variant:   a.variantForIndex(i),
			nextPause: 90 + rand.Intn(90),
		}
	}
}

func (a *darwinPetApp) setSpeed(speed int) {
	a.speed = normalizeDarwinSpeed(speed)
	for i := range a.pets {
		a.pets[i].speed = a.petSpeed(i)
	}
}

func (a *darwinPetApp) setPetCount(count int) {
	a.petCount = normalizeDarwinPetCount(count)
	a.resetPets()
	a.tickReactions()
}

func (a *darwinPetApp) setMode(mode int) {
	a.mode = normalizeDarwinMode(mode)
	if a.mode == darwinModeKeyboard {
		for i := range a.pets {
			a.pets[i].pause = 24
			a.pets[i].nextPause = 80 + rand.Intn(90)
		}
	}
}

func (a *darwinPetApp) setCoatMode(mode int) {
	a.coatMode = normalizeDarwinCoatMode(mode)
	a.refreshPetVariants()
}

func (a *darwinPetApp) setFixedVariant(variant int) {
	a.variant = normalizeDarwinVariant(variant)
	if a.coatMode == darwinCoatFixed {
		a.refreshPetVariants()
	}
}

func (a *darwinPetApp) setSelectedVariant(index int, variant int) {
	if index < 0 || index >= maxPetCount {
		return
	}
	a.selectedCoats[index] = normalizeDarwinVariant(variant)
	if a.coatMode == darwinCoatSelected && index < len(a.pets) {
		a.pets[index].variant = a.selectedCoats[index]
	}
}

func (a *darwinPetApp) setWheelEnabled(enabled bool) {
	a.wheelEnabled = enabled
	if !enabled {
		a.keyHold = 0
	}
}

func (a *darwinPetApp) setPetName(index int, name string) {
	if index < 0 || index >= maxPetCount {
		return
	}
	a.petNames[index] = sanitizeDarwinPetName(name)
}

func (a *darwinPetApp) petDisplayName(index int) string {
	if index < 0 || index >= maxPetCount {
		return ""
	}
	if name := sanitizeDarwinPetName(a.petNames[index]); name != "" {
		return name
	}
	return fmt.Sprintf("デグー%d", index+1)
}

func (a *darwinPetApp) petSpeed(index int) int {
	return max(1, a.speed-1+index%2)
}

func (a *darwinPetApp) refreshPetVariants() {
	for i := range a.pets {
		a.pets[i].variant = a.variantForIndex(i)
	}
}

func (a *darwinPetApp) variantForIndex(index int) int {
	if len(darwinVariants) == 0 {
		return 0
	}
	switch a.coatMode {
	case darwinCoatRandom:
		return rand.Intn(len(darwinVariants))
	case darwinCoatSelected:
		if index >= 0 && index < maxPetCount {
			return normalizeDarwinVariant(a.selectedCoats[index])
		}
	}
	return normalizeDarwinVariant(a.variant)
}

func (a *darwinPetApp) variantID(index int) string {
	index = normalizeDarwinVariant(index)
	return darwinVariants[index].ID
}

func defaultDarwinSelectedCoats() [maxPetCount]int {
	return [maxPetCount]int{0, 1, 2, 4, 8, 6, 3, 7, 5, 9}
}

func normalizeDarwinSpeed(speed int) int {
	switch speed {
	case darwinSpeedSlow, darwinSpeedNormal, darwinSpeedFast:
		return speed
	default:
		return darwinSpeedNormal
	}
}

func normalizeDarwinPetCount(count int) int {
	return clamp(count, 1, maxPetCount)
}

func normalizeDarwinMode(mode int) darwinBehaviorMode {
	switch darwinBehaviorMode(mode) {
	case darwinModeKeyboard, darwinModeRandom:
		return darwinBehaviorMode(mode)
	default:
		return darwinModeRandom
	}
}

func normalizeDarwinCoatMode(mode int) darwinCoatMode {
	switch darwinCoatMode(mode) {
	case darwinCoatFixed, darwinCoatSelected, darwinCoatRandom:
		return darwinCoatMode(mode)
	default:
		return darwinCoatRandom
	}
}

func normalizeDarwinVariant(variant int) int {
	return clamp(variant, 0, max(0, len(darwinVariants)-1))
}

func sanitizeDarwinPetName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	runes := []rune(name)
	if len(runes) > maxPetNameLen {
		runes = runes[:maxPetNameLen]
	}
	return string(runes)
}

func (a *darwinPetApp) movePet(p *darwinPet, speed int) {
	p.x += p.dir * max(1, speed)
	if p.x <= 0 {
		p.x = 0
		p.dir = 1
	}
	if p.x >= max(0, a.sceneW-spriteW) {
		p.x = max(0, a.sceneW-spriteW)
		p.dir = -1
	}
}

func (a *darwinPetApp) tickPets() {
	a.tick++
	if a.keyHold > 0 {
		a.keyHold--
	}

	for i := range a.pets {
		p := &a.pets[i]
		if a.keyHold > 0 && i == 0 {
			p.frame = seqFrameFrom(walkFrameSeq, a.tick, 1)
			if !a.wheelEnabled {
				a.movePet(p, p.speed+1)
			}
			continue
		}
		if a.mode == darwinModeKeyboard {
			p.frame = seqFrameFrom(idleFrameSeq, a.tick, 5)
			continue
		}
		if p.pause > 0 {
			p.pause--
			p.frame = seqFrameFrom(idleFrameSeq, a.tick, 5)
			continue
		}
		p.nextPause--
		if p.nextPause <= 0 {
			p.pause = 30 + rand.Intn(70)
			p.nextPause = 120 + rand.Intn(180)
			switch rand.Intn(3) {
			case 0:
				p.frame = seqFrameFrom(nibbleFrameSeq, a.tick, 3)
			case 1:
				p.frame = seqFrameFrom(hopFrameSeq, a.tick, 2)
			default:
				p.frame = seqFrameFrom(idleFrameSeq, a.tick, 5)
			}
			continue
		}
		a.movePet(p, p.speed)
		p.frame = seqFrameFrom(walkFrameSeq, a.tick, 2)
	}
	a.tickReactions()
}

func (a *darwinPetApp) tickReactions() {
	out := a.reactions[:0]
	for _, reaction := range a.reactions {
		reaction.ticks--
		if reaction.ticks > 0 && reaction.pet >= 0 && reaction.pet < len(a.pets) {
			out = append(out, reaction)
		}
	}
	a.reactions = out
}

func (a *darwinPetApp) addClickReaction(sceneX, sceneY int) bool {
	index := a.petAtScenePoint(sceneX, sceneY)
	if index < 0 {
		return false
	}
	kind := rand.Intn(3)
	for i := range a.reactions {
		if a.reactions[i].pet == index {
			a.reactions[i].kind = kind
			a.reactions[i].ticks = reactionTicks
			return true
		}
	}
	a.reactions = append(a.reactions, darwinReaction{pet: index, kind: kind, ticks: reactionTicks})
	return true
}

func (a *darwinPetApp) petAtScenePoint(sceneX, sceneY int) int {
	if sceneX < 0 || sceneX >= a.sceneW || sceneY < 0 || sceneY >= sceneH {
		return -1
	}
	wheelActive := a.wheelEnabled && a.keyHold > 0 && len(a.pets) > 0
	for i := len(a.pets) - 1; i >= 0; i-- {
		if wheelActive && i == 0 {
			continue
		}
		p := a.pets[i]
		y := sceneH - spriteH - p.lane
		if sceneX >= p.x+6 && sceneX <= p.x+spriteW-6 && sceneY >= y+8 && sceneY <= y+spriteH-4 {
			return i
		}
	}
	return -1
}

func (a *darwinPetApp) render() *image.RGBA {
	w := max(320, a.sceneW)
	canvas := image.NewRGBA(image.Rect(0, 0, w, sceneH))
	draw.Draw(canvas, canvas.Bounds(), image.Transparent, image.Point{}, draw.Src)

	wheelActive := a.wheelEnabled && a.keyHold > 0 && len(a.pets) > 0
	if wheelActive {
		wheelX := clamp(w-116, 8, max(8, w-wheelSize-8))
		wheelY := sceneH - wheelSize - 4
		if a.wheel != nil {
			draw.Draw(canvas, image.Rect(wheelX, wheelY, wheelX+wheelSize, wheelY+wheelSize), a.wheel, image.Point{}, draw.Over)
		}
		if frames := a.frames[a.variantID(a.pets[0].variant)]; len(frames) > a.pets[0].frame {
			runner := scaleNearest(frames[a.pets[0].frame], 66, 44)
			drawFacingImage(canvas, runner, image.Rect(wheelX+3, wheelY+22, wheelX+69, wheelY+66), 1)
		}
	}

	for i := range a.pets {
		if wheelActive && i == 0 {
			continue
		}
		p := &a.pets[i]
		frames := a.frames[a.variantID(p.variant)]
		if len(frames) <= p.frame {
			continue
		}
		y := sceneH - spriteH - p.lane
		drawFacingImage(canvas, frames[p.frame], image.Rect(p.x, y, p.x+spriteW, y+spriteH), p.dir)
	}
	a.drawReactions(canvas)
	return canvas
}

func (a *darwinPetApp) drawReactions(dst *image.RGBA) {
	for _, reaction := range a.reactions {
		if reaction.pet < 0 || reaction.pet >= len(a.pets) {
			continue
		}
		wheelActive := a.wheelEnabled && a.keyHold > 0 && reaction.pet == 0
		if wheelActive {
			continue
		}
		p := a.pets[reaction.pet]
		baseY := sceneH - spriteH - p.lane
		x := clamp(p.x+spriteW/2-18, 2, max(2, a.sceneW-42))
		y := clamp(baseY-26-(reactionTicks-reaction.ticks)/8, 0, sceneH-32)
		drawReactionBubble(dst, x, y, reaction.kind, reaction.ticks)
	}
}

func (a *darwinPetApp) statusIconPNG() []byte {
	frames := a.frames["wild_agouti"]
	if len(frames) == 0 {
		return nil
	}
	visible := cropVisible(frames[idleStart])
	if visible.Bounds().Empty() {
		return nil
	}

	const iconW = 22
	const iconH = 18
	vb := visible.Bounds()
	targetW := iconW
	targetH := max(1, vb.Dy()*targetW/vb.Dx())
	if targetH > iconH {
		targetH = iconH
		targetW = max(1, vb.Dx()*targetH/vb.Dy())
	}
	scaled := scaleNearest(visible, targetW, targetH)
	icon := image.NewRGBA(image.Rect(0, 0, iconW, iconH))
	draw.Draw(icon, icon.Bounds(), image.Transparent, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect((iconW-targetW)/2, (iconH-targetH)/2, (iconW+targetW)/2, (iconH+targetH)/2), scaled, image.Point{}, draw.Over)

	var buf bytes.Buffer
	if err := png.Encode(&buf, icon); err != nil {
		return nil
	}
	return buf.Bytes()
}

func seqFrameFrom(seq []int, tick, delay int) int {
	if len(seq) == 0 {
		return idleStart
	}
	if delay <= 0 {
		delay = 1
	}
	return seq[(tick/delay)%len(seq)]
}

func loadDarwinSprites() map[string][]*image.RGBA {
	out := make(map[string][]*image.RGBA, len(darwinVariants))
	for _, variant := range darwinVariants {
		name := fmt.Sprintf("sprites/degu_%s_set00.png", variant.ID)
		data, err := fs.ReadFile(appassets.FS, name)
		if err != nil {
			panic(err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			panic(err)
		}
		if img.Bounds().Dx() != frameW*frameCount || img.Bounds().Dy() != frameH {
			panic(fmt.Sprintf("%s must be %dx%d", name, frameW*frameCount, frameH))
		}
		frames := make([]*image.RGBA, frameCount)
		for i := 0; i < frameCount; i++ {
			frame := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
			srcRect := image.Rect(i*frameW, 0, (i+1)*frameW, frameH)
			draw.Draw(frame, frame.Bounds(), img, srcRect.Min, draw.Src)
			frames[i] = frame
		}
		out[variant.ID] = frames
	}
	return out
}

func loadDarwinWheel() *image.RGBA {
	data, err := fs.ReadFile(appassets.FS, "sprites/wheel.png")
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, wheelSize, wheelSize))
	draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)
	return dst
}

func drawFacingImage(dst *image.RGBA, src *image.RGBA, r image.Rectangle, dir int) {
	if src == nil {
		return
	}
	if dir >= 0 {
		draw.Draw(dst, r, src, image.Point{}, draw.Over)
		return
	}
	flipped := image.NewRGBA(src.Bounds())
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			flipped.Set(x, y, src.At(b.Min.X+b.Dx()-1-x, b.Min.Y+y))
		}
	}
	draw.Draw(dst, r, flipped, image.Point{}, draw.Over)
}

func scaleNearest(src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	if src == nil || width <= 0 || height <= 0 {
		return dst
	}
	sb := src.Bounds()
	for y := 0; y < height; y++ {
		sy := sb.Min.Y + y*sb.Dy()/height
		for x := 0; x < width; x++ {
			sx := sb.Min.X + x*sb.Dx()/width
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func cropVisible(src *image.RGBA) *image.RGBA {
	if src == nil {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.RGBAAt(x, y).A <= 8 {
				continue
			}
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
	if minX >= maxX || minY >= maxY {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	content := image.Rect(max(b.Min.X, minX-1), max(b.Min.Y, minY-1), min(b.Max.X, maxX+1), min(b.Max.Y, maxY+1))
	dst := image.NewRGBA(image.Rect(0, 0, content.Dx(), content.Dy()))
	draw.Draw(dst, dst.Bounds(), src, content.Min, draw.Src)
	return dst
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
			pt := image.Pt(x+col, y+row)
			if pt.In(dst.Bounds()) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), c))
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
			pt := image.Pt(px, y+row)
			if pt.In(dst.Bounds()) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), c))
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
			pt := image.Pt(px, py+sy)
			if pt.In(dst.Bounds()) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), ink))
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
			pt := image.Pt(px, py)
			if pt.In(dst.Bounds()) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), c))
			}
		}
	}
}

func drawRoundedRect(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			pt := image.Pt(px, py)
			if roundedRectContains(px-x, py-y, w, h, radius) && pt.In(dst.Bounds()) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), c))
			}
		}
	}
}

func drawRoundedRectOutline(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			pt := image.Pt(px, py)
			if !pt.In(dst.Bounds()) || !roundedRectContains(px-x, py-y, w, h, radius) {
				continue
			}
			if px == x || px == x+w-1 || py == y || py == y+h-1 ||
				!roundedRectContains(px-x-1, py-y, w, h, radius) ||
				!roundedRectContains(px-x+1, py-y, w, h, radius) ||
				!roundedRectContains(px-x, py-y-1, w, h, radius) ||
				!roundedRectContains(px-x, py-y+1, w, h, radius) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), c))
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

func drawBlock(dst *image.RGBA, x, y int, c color.RGBA) {
	for py := y - 1; py <= y+1; py++ {
		for px := x - 1; px <= x+1; px++ {
			pt := image.Pt(px, py)
			if pt.In(dst.Bounds()) {
				dst.SetRGBA(pt.X, pt.Y, overRGBA(dst.RGBAAt(pt.X, pt.Y), c))
			}
		}
	}
}

func rgba(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
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

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
