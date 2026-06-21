package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedSpritesHaveFixedCanvas(t *testing.T) {
	for _, path := range spriteSheetPaths(t) {
		img := openTestPNG(t, path)
		if got, want := img.Bounds().Dx(), frameW*totalFrames; got != want {
			t.Fatalf("%s width = %d, want %d", path, got, want)
		}
		if got, want := img.Bounds().Dy(), frameH; got != want {
			t.Fatalf("%s height = %d, want %d", path, got, want)
		}

		for frame := 0; frame < totalFrames; frame++ {
			rect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
			content := alphaBoundsInImage(img, rect)
			if content.Empty() {
				t.Fatalf("%s frame %d is empty", path, frame)
			}
			if content.Min.X < rect.Min.X || content.Max.X > rect.Max.X || content.Min.Y < rect.Min.Y || content.Max.Y > rect.Max.Y {
				t.Fatalf("%s frame %d content escapes frame bounds: %v in %v", path, frame, content, rect)
			}
		}
	}
}

func TestGeneratedSpritesContainPoseMotion(t *testing.T) {
	for _, path := range spriteSheetPaths(t) {
		img := openTestPNG(t, path)
		assertPoseMotion(t, img, path, actionSpec("walk"), 80)
		assertPoseMotion(t, img, path, actionSpec("scurry"), 120)
		assertPoseMotion(t, img, path, actionSpec("nibble"), 50)
		assertPoseMotion(t, img, path, actionSpec("hop"), 120)
		assertPoseMotion(t, img, path, actionSpec("wheelrun"), 120)
	}
}

func TestGeneratedSpritesHaveNoDetachedFragments(t *testing.T) {
	for _, path := range spriteSheetPaths(t) {
		img := openTestPNG(t, path)
		for frame := 0; frame < totalFrames; frame++ {
			rect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
			components := connectedComponents(img, rect)
			if len(components) == 0 {
				t.Fatalf("%s frame %d has no components", path, frame)
			}
			largest := largestComponent(components)
			minArea := maxInt(24, largest.area/20)
			for _, component := range components {
				if component == largest || component.area < minArea {
					continue
				}
				if verticalGap(component.bounds, largest.bounds) > 6 {
					t.Fatalf("%s frame %d has detached fragment area=%d bounds=%v main=%v", path, frame, component.area, component.bounds, largest.bounds)
				}
			}
		}
	}
}

func TestGeneratedSpritesMoveSmoothly(t *testing.T) {
	for _, path := range spriteSheetPaths(t) {
		img := openTestPNG(t, path)
		for _, spec := range rows {
			var lastCenter image.Point
			minArea, maxArea := 1<<30, 0
			minW, maxW := 1<<30, 0
			for i := 0; i < spec.Cols; i++ {
				frame := spec.Offset + i
				rect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
				content := alphaBoundsInImage(img, rect)
				center := image.Pt(centerX(content)-rect.Min.X, centerY(content)-rect.Min.Y)
				area := opaqueCountInImage(img, content)
				minArea = minInt(minArea, area)
				maxArea = maxInt(maxArea, area)
				minW = minInt(minW, content.Dx())
				maxW = maxInt(maxW, content.Dx())
				if i > 0 {
					dx := abs(center.X - lastCenter.X)
					dy := abs(center.Y - lastCenter.Y)
					if dx > 14 || dy > 10 {
						t.Fatalf("%s %s frame %d center jump = %d,%d", path, spec.Name, frame, dx, dy)
					}
				}
				lastCenter = center
			}
			if maxArea-minArea > 1200 {
				t.Fatalf("%s %s area delta = %d", path, spec.Name, maxArea-minArea)
			}
			if maxW-minW > 22 {
				t.Fatalf("%s %s width delta = %d", path, spec.Name, maxW-minW)
			}
		}
	}
}

func TestGeneratedSpriteSetCount(t *testing.T) {
	paths := spriteSheetPaths(t)
	if got, want := len(paths), len(variants)*motionSets; got != want {
		t.Fatalf("sprite set count = %d, want %d", got, want)
	}
}

func TestCleanWheelArtworkRemovesEnclosedBakedChecker(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			if (x/2+y/2)%2 == 0 {
				src.SetRGBA(x, y, color.RGBA{R: 238, G: 238, B: 238, A: 255})
			} else {
				src.SetRGBA(x, y, color.RGBA{R: 252, G: 252, B: 252, A: 255})
			}
		}
	}
	wood := color.RGBA{R: 120, G: 80, B: 42, A: 255}
	for y := 5; y <= 18; y++ {
		for x := 5; x <= 18; x++ {
			if x <= 7 || x >= 16 || y <= 7 || y >= 16 {
				src.SetRGBA(x, y, wood)
			}
		}
	}

	cleaned := cleanWheelArtwork(src)
	if got := cleaned.RGBAAt(12, 12).A; got != 0 {
		t.Fatalf("enclosed checker alpha = %d, want 0", got)
	}
	if got := cleaned.RGBAAt(6, 12).A; got == 0 {
		t.Fatalf("wheel rim was removed")
	}
}

func TestGeneratedWheelSpriteHasTransparentCenter(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "sprites", "wheel.png")
	img := openTestPNG(t, path)
	if got, want := img.Bounds().Dx(), wheelW; got != want {
		t.Fatalf("wheel width = %d, want %d", got, want)
	}
	if got, want := img.Bounds().Dy(), wheelH; got != want {
		t.Fatalf("wheel height = %d, want %d", got, want)
	}
	_, _, _, alpha := img.At(wheelW/2, wheelH/2).RGBA()
	if alpha != 0 {
		t.Fatalf("wheel center alpha = %#x, want transparent", alpha)
	}
}

func spriteSheetPaths(t *testing.T) []string {
	t.Helper()
	spriteDir := filepath.Join("..", "..", "assets", "sprites")
	paths := make([]string, 0, len(variants)*motionSets)
	for _, id := range variants {
		for set := 0; set < motionSets; set++ {
			path := filepath.Join(spriteDir, "degu_"+id+"_set"+twoDigits(set)+".png")
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing sprite set %s: %v", path, err)
			}
			paths = append(paths, path)
		}
	}
	return paths
}

func twoDigits(v int) string {
	return fmt.Sprintf("%02d", v)
}

func assertPoseMotion(t *testing.T, img image.Image, path string, spec rowSpec, minChanged int) {
	t.Helper()
	base := normalizedPose(img, spec.Offset)
	maxChanged := 0
	for i := 1; i < spec.Cols; i++ {
		changed := changedPixels(base, normalizedPose(img, spec.Offset+i))
		if changed > maxChanged {
			maxChanged = changed
		}
	}
	if maxChanged < minChanged {
		t.Fatalf("%s %s normalized pose motion changed %d pixels, want at least %d", path, spec.Name, maxChanged, minChanged)
	}
}

func actionSpec(name string) rowSpec {
	for _, spec := range rows {
		if spec.Name == name {
			return spec
		}
	}
	panic("unknown action spec: " + name)
}

type testComponent struct {
	area   int
	bounds image.Rectangle
}

func connectedComponents(img image.Image, rect image.Rectangle) []testComponent {
	rect = rect.Intersect(img.Bounds())
	w, h := rect.Dx(), rect.Dy()
	seen := make([]bool, w*h)
	components := []testComponent{}
	index := func(x, y int) int {
		return (y-rect.Min.Y)*w + (x - rect.Min.X)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if seen[index(x, y)] || !opaqueAt(img, x, y) {
				continue
			}
			queue := []image.Point{image.Pt(x, y)}
			seen[index(x, y)] = true
			area := 0
			minX, minY, maxX, maxY := x, y, x+1, y+1
			for len(queue) > 0 {
				p := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				area++
				if p.X < minX {
					minX = p.X
				}
				if p.Y < minY {
					minY = p.Y
				}
				if p.X+1 > maxX {
					maxX = p.X + 1
				}
				if p.Y+1 > maxY {
					maxY = p.Y + 1
				}
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := p.X+dx, p.Y+dy
						if nx < rect.Min.X || nx >= rect.Max.X || ny < rect.Min.Y || ny >= rect.Max.Y {
							continue
						}
						i := index(nx, ny)
						if seen[i] || !opaqueAt(img, nx, ny) {
							continue
						}
						seen[i] = true
						queue = append(queue, image.Pt(nx, ny))
					}
				}
			}
			components = append(components, testComponent{area: area, bounds: image.Rect(minX, minY, maxX, maxY)})
		}
	}
	return components
}

func opaqueAt(img image.Image, x int, y int) bool {
	_, _, _, a := img.At(x, y).RGBA()
	return a > 0x0800
}

func largestComponent(components []testComponent) testComponent {
	largest := components[0]
	for _, component := range components[1:] {
		if component.area > largest.area {
			largest = component
		}
	}
	return largest
}

func verticalGap(a, b image.Rectangle) int {
	if a.Max.Y < b.Min.Y {
		return b.Min.Y - a.Max.Y
	}
	if b.Max.Y < a.Min.Y {
		return a.Min.Y - b.Max.Y
	}
	return 0
}

func openTestPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func normalizedPose(img image.Image, frame int) *image.RGBA {
	rect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
	content := alphaBoundsInImage(img, rect)
	out := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	if content.Empty() {
		return out
	}
	offset := image.Pt((frameW-content.Dx())/2, baselineY-content.Dy())
	drawRect := image.Rect(offset.X, offset.Y, offset.X+content.Dx(), offset.Y+content.Dy())
	draw.Draw(out, drawRect, img, content.Min, draw.Src)
	return out
}

func changedPixels(a, b image.Image) int {
	bounds := a.Bounds().Intersect(b.Bounds())
	changed := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if abs32(ar, br)+abs32(ag, bg)+abs32(ab, bb)+abs32(aa, ba) > 0x1800 {
				changed++
			}
		}
	}
	return changed
}

func opaqueCountInImage(img image.Image, rect image.Rectangle) int {
	rect = rect.Intersect(img.Bounds())
	count := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if opaqueAt(img, x, y) {
				count++
			}
		}
	}
	return count
}

func centerX(r image.Rectangle) int {
	return (r.Min.X + r.Max.X) / 2
}

func centerY(r image.Rectangle) int {
	return (r.Min.Y + r.Max.Y) / 2
}

func alphaBoundsInImage(img image.Image, rect image.Rectangle) image.Rectangle {
	rect = rect.Intersect(img.Bounds())
	minX, minY := rect.Max.X, rect.Max.Y
	maxX, maxY := rect.Min.X, rect.Min.Y
	found := false
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
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

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func abs32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
