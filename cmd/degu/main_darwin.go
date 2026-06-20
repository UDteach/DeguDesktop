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
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"
	"math/rand"
	"runtime"
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

var appVersion = "dev"

var darwinApp *darwinPetApp

type darwinPetApp struct {
	mu       sync.Mutex
	sceneW   int
	tick     int
	keyHold  int
	frames   map[string][]*image.RGBA
	wheel    *image.RGBA
	pets     []darwinPet
	variants []string
}

type darwinPet struct {
	x         int
	lane      int
	dir       int
	speed     int
	frame     int
	variant   string
	nextPause int
	pause     int
}

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
	variants := []string{"wild_agouti", "blue", "sand_champagne", "black_pied", "cream_pied"}
	return &darwinPetApp{
		sceneW:   900,
		frames:   loadDarwinSprites(variants),
		wheel:    loadDarwinWheel(),
		variants: variants,
	}
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
	darwinApp.keyHold = 18
	darwinApp.mu.Unlock()
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
	count := 5
	if a.sceneW < 640 {
		count = 3
	}
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
			speed:     1 + i%2,
			variant:   a.variants[i%len(a.variants)],
			nextPause: 90 + rand.Intn(90),
		}
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
			p.frame = seqFrame(scurryStart, scurryFrames, a.tick, 2)
			continue
		}
		if p.pause > 0 {
			p.pause--
			p.frame = seqFrame(idleStart, idleFrames, a.tick, 12)
			continue
		}
		p.nextPause--
		if p.nextPause <= 0 {
			p.pause = 30 + rand.Intn(70)
			p.nextPause = 120 + rand.Intn(180)
			switch rand.Intn(3) {
			case 0:
				p.frame = seqFrame(nibbleStart, nibbleFrames, a.tick, 5)
			case 1:
				p.frame = seqFrame(hopStart, hopFrames, a.tick, 4)
			default:
				p.frame = seqFrame(idleStart, idleFrames, a.tick, 8)
			}
			continue
		}
		p.x += p.dir * p.speed
		if p.x <= 0 {
			p.x = 0
			p.dir = 1
		}
		if p.x >= max(0, a.sceneW-spriteW) {
			p.x = max(0, a.sceneW-spriteW)
			p.dir = -1
		}
		p.frame = seqFrame(walkStart, walkFrames, a.tick, 3)
	}
}

func (a *darwinPetApp) render() *image.RGBA {
	w := max(320, a.sceneW)
	canvas := image.NewRGBA(image.Rect(0, 0, w, sceneH))
	draw.Draw(canvas, canvas.Bounds(), image.Transparent, image.Point{}, draw.Src)

	wheelActive := a.keyHold > 0 && len(a.pets) > 0
	if wheelActive {
		wheelX := clamp(w-116, 8, max(8, w-wheelSize-8))
		wheelY := sceneH - wheelSize - 4
		if a.wheel != nil {
			draw.Draw(canvas, image.Rect(wheelX, wheelY, wheelX+wheelSize, wheelY+wheelSize), a.wheel, image.Point{}, draw.Over)
		}
		if frames := a.frames[a.pets[0].variant]; len(frames) > a.pets[0].frame {
			runner := scaleNearest(frames[a.pets[0].frame], 66, 44)
			drawFacingImage(canvas, runner, image.Rect(wheelX+3, wheelY+22, wheelX+69, wheelY+66), 1)
		}
	}

	for i := range a.pets {
		if wheelActive && i == 0 {
			continue
		}
		p := &a.pets[i]
		frames := a.frames[p.variant]
		if len(frames) <= p.frame {
			continue
		}
		y := sceneH - spriteH - p.lane
		drawFacingImage(canvas, frames[p.frame], image.Rect(p.x, y, p.x+spriteW, y+spriteH), p.dir)
	}
	return canvas
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

func seqFrame(start, frames, tick, delay int) int {
	if frames <= 0 {
		return start
	}
	if delay <= 0 {
		delay = 1
	}
	return start + (tick/delay)%frames
}

func loadDarwinSprites(ids []string) map[string][]*image.RGBA {
	out := make(map[string][]*image.RGBA, len(ids))
	for _, id := range ids {
		name := fmt.Sprintf("sprites/degu_%s_set00.png", id)
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
		out[id] = frames
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
