package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
)

const (
	totalFrames = 56
	motionSets  = 10
	atlasRows   = 11
	atlasCols   = 56
	coatRows    = 5
	coatCols    = 7
	frameW      = 96
	frameH      = 64
	forageW     = 32
	forageH     = 24
	wheelW      = 72
	wheelH      = 72
	padding     = 5
	targetW     = 88
	targetH     = 52
	baselineY   = 59
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

type rowSpec struct {
	Name   string
	Row    int
	Cols   int
	Offset int
}

var variants = []string{
	"wild_agouti",
	"black",
	"blue",
	"gray",
	"white_cream",
	"sand_champagne",
	"chocolate",
	"black_pied",
	"agouti_pied",
	"blue_pied",
	"cream_pied",
}

var rows = []rowSpec{
	{Name: "idle", Row: 0, Cols: 4, Offset: 0},
	{Name: "walk", Row: 1, Cols: 8, Offset: 4},
	{Name: "scurry", Row: 2, Cols: 8, Offset: 12},
	{Name: "nibble", Row: 3, Cols: 6, Offset: 20},
	{Name: "hop", Row: 4, Cols: 6, Offset: 26},
	{Name: "turn", Row: 5, Cols: 8, Offset: 32},
	{Name: "eat", Row: 6, Cols: 4, Offset: 40},
	{Name: "dig", Row: 7, Cols: 4, Offset: 44},
	{Name: "stand", Row: 8, Cols: 4, Offset: 48},
	{Name: "groomface", Row: 9, Cols: 4, Offset: 52},
}

type report struct {
	Source       string       `json:"source"`
	SourceWidth  int          `json:"source_width"`
	SourceHeight int          `json:"source_height"`
	Columns      int          `json:"columns"`
	Rows         int          `json:"rows"`
	FrameWidth   int          `json:"frame_width"`
	FrameHeight  int          `json:"frame_height"`
	Cells        []cellReport `json:"cells"`
	Warnings     []string     `json:"warnings,omitempty"`
}

type cellReport struct {
	Variant  string   `json:"variant"`
	Frame    int      `json:"frame"`
	Sheet    string   `json:"sheet"`
	Source   rectJSON `json:"source_rect"`
	Content  rectJSON `json:"content_rect"`
	Output   string   `json:"output"`
	Warnings []string `json:"warnings,omitempty"`
}

type rectJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type rawFrame struct {
	img      *image.RGBA
	content  image.Rectangle
	valid    bool
	warnings []string
}

type coatGuide struct {
	img     *image.RGBA
	content image.Rectangle
}

type frameSpec struct {
	Index  int
	Action string
	Step   int
}

func main() {
	source := flag.String("source", filepath.FromSlash("assets/source/imagegen-sheet-clean.png"), "single 56x11 ImageGen atlas")
	poseDir := flag.String("pose-dir", filepath.FromSlash("assets/source/poses"), "directory containing one clean ImageGen pose per coat")
	frameDir := flag.String("frame-dir", filepath.FromSlash("assets/source/frames"), "directory containing one ImageGen PNG per runtime frame")
	sourceDir := flag.String("source-dir", filepath.FromSlash("assets/source/coats"), "directory containing one 8x5 ImageGen sheet per coat")
	outDir := flag.String("out", filepath.FromSlash("assets/sprites"), "output sprite directory")
	preview := flag.String("preview", filepath.FromSlash("docs/assets/degu-preview.png"), "preview PNG path")
	reportPath := flag.String("report", filepath.FromSlash("assets/source/import-report.json"), "validation report path")
	wheelSource := flag.String("wheel-source", filepath.FromSlash("assets/source/imagegen-wheel.png"), "single ImageGen wheel PNG")
	forageDir := flag.String("forage-dir", filepath.FromSlash("assets/source/forage"), "directory containing ImageGen forage prop PNGs")
	coatGuideDir := flag.String("coat-guide-dir", filepath.FromSlash("assets/source/coat-guides"), "directory containing ImageGen coat guide PNGs")
	flag.Parse()

	must(os.MkdirAll(*outDir, 0o755))
	must(os.MkdirAll(filepath.Dir(*preview), 0o755))
	must(os.MkdirAll(filepath.Dir(*reportPath), 0o755))

	rep := report{
		Source:      filepath.Clean(*source),
		Columns:     totalFrames,
		Rows:        atlasRows,
		FrameWidth:  frameW,
		FrameHeight: frameH,
	}
	sheets := make(map[string]*image.RGBA, len(variants))
	if frameFilesReady(*frameDir) {
		rep.Source = filepath.Clean(*frameDir)
		rep.Rows = len(variants)
		importFrameFiles(*frameDir, *coatGuideDir, sheets, &rep, *outDir)
	} else if actionSheetsReady(filepath.Dir(*source)) {
		rep.Source = filepath.Clean(filepath.Dir(*source))
		rep.Rows = len(variants)
		importActionSheets(filepath.Dir(*source), sheets, &rep, *outDir)
	} else if posesReady(*poseDir) {
		rep.Source = filepath.Clean(*poseDir)
		rep.Rows = 1
		importPoses(*poseDir, sheets, &rep, *outDir)
	} else if *source != "" {
		src, err := openPNG(*source)
		if err != nil {
			log.Fatalf("open %s: %v", *source, err)
		}
		rep.SourceWidth = src.Bounds().Dx()
		rep.SourceHeight = src.Bounds().Dy()
		importFullAtlas(src, filepath.Base(*source), sheets, &rep, *outDir)
	} else {
		rep.Source = filepath.Clean(*sourceDir)
		rep.Rows = len(rows)
		for _, id := range variants {
			path := filepath.Join(*sourceDir, id+".png")
			src, err := openPNG(path)
			if err != nil {
				log.Fatalf("open %s: %v", path, err)
			}
			if rep.SourceWidth == 0 {
				rep.SourceWidth = src.Bounds().Dx()
				rep.SourceHeight = src.Bounds().Dy()
			}
			sheet := image.NewRGBA(image.Rect(0, 0, frameW*totalFrames, frameH))
			importCoatSheet(src, id, id+".png", sheet, &rep, *outDir)
			sheets[id] = sheet
		}
	}

	for _, id := range variants {
		writeSpriteSets(*outDir, id, sheets[id])
	}

	writePreview(*preview, sheets)
	if wild := sheets["wild_agouti"]; wild != nil {
		writeICO(filepath.FromSlash("assets/tray.ico"), firstFrame(wild))
	}
	writeWheelSprite(*wheelSource, filepath.Join(*outDir, "wheel.png"))
	writeForageSprites(*forageDir, *outDir)
	writeJSON(*reportPath, rep)
	if len(rep.Warnings) > 0 {
		for _, warning := range rep.Warnings {
			fmt.Println("warning:", warning)
		}
	}
	fmt.Printf("imported %d variants into %d-frame sheets\n", len(variants), totalFrames)
}

func posesReady(dir string) bool {
	for _, id := range variants {
		if _, err := os.Stat(filepath.Join(dir, id+".png")); err != nil {
			return false
		}
	}
	return true
}

func actionSheetsReady(dir string) bool {
	for _, spec := range rows {
		if _, err := os.Stat(filepath.Join(dir, "imagegen-"+spec.Name+".png")); err != nil {
			return false
		}
	}
	return true
}

func frameFilesReady(dir string) bool {
	for _, spec := range expectedFrameSpecs() {
		if _, err := os.Stat(filepath.Join(dir, "wild_agouti", frameFileName(spec))); err != nil {
			return false
		}
	}
	return true
}

func expectedFrameSpecs() []frameSpec {
	specs := make([]frameSpec, 0, totalFrames)
	for _, row := range rows {
		for i := 0; i < row.Cols; i++ {
			specs = append(specs, frameSpec{
				Index:  row.Offset + i,
				Action: row.Name,
				Step:   i,
			})
		}
	}
	return specs
}

func frameFileName(spec frameSpec) string {
	return fmt.Sprintf("%02d_%s_%02d.png", spec.Index, spec.Action, spec.Step)
}

func importFrameFiles(dir string, coatGuideDir string, sheets map[string]*image.RGBA, rep *report, outDir string) {
	specs := expectedFrameSpecs()
	canonical := loadFrameFileSet(dir, "wild_agouti", specs, rep, outDir)
	repairFrames(canonical)
	canonical = stabilizeCanonicalMotion(canonical)
	coatGuides := loadCoatGuides(coatGuideDir)
	for _, id := range variants {
		rawFrames := cloneAndTintFrames(canonical, id, coatGuides[id])
		appendFrameFileReports(id, specs, rawFrames, rep, outDir)
		sheet := image.NewRGBA(image.Rect(0, 0, frameW*totalFrames, frameH))
		for _, row := range rows {
			scale := commonActionScale(rawFrames, row)
			for i := 0; i < row.Cols; i++ {
				frameIndex := row.Offset + i
				frame := rawFrames[frameIndex]
				fitted := fitToFrameWithScale(frame.img, frame.content, scale)
				draw.Draw(sheet, image.Rect(frameIndex*frameW, 0, (frameIndex+1)*frameW, frameH), fitted, image.Point{}, draw.Over)
			}
		}
		sheets[id] = sheet
	}
}

func loadFrameFileSet(dir string, id string, specs []frameSpec, rep *report, outDir string) []rawFrame {
	rawFrames := make([]rawFrame, totalFrames)
	for _, spec := range specs {
		path := filepath.Join(dir, id, frameFileName(spec))
		src, err := openPNG(path)
		if err != nil {
			log.Fatalf("open %s: %v", path, err)
		}
		b := src.Bounds()
		rep.SourceWidth = max(rep.SourceWidth, b.Dx())
		rep.SourceHeight = max(rep.SourceHeight, b.Dy())
		cell := imageToCleanCell(src)
		content := alphaBounds(cell)
		cellWarnings := validateCell(content, cell.Bounds())
		valid := isValidContent(content, cell)
		if !valid {
			cellWarnings = append(cellWarnings, "invalid canonical single-frame asset; nearest valid frame will replace it")
		}
		rawFrames[spec.Index] = rawFrame{
			img:      cell,
			content:  content,
			valid:    valid,
			warnings: cellWarnings,
		}
	}
	return rawFrames
}

func stabilizeCanonicalMotion(frames []rawFrame) []rawFrame {
	out := cloneRawFrames(frames)
	walk := make([]rawFrame, walkFrames)
	for i := 0; i < walkFrames; i++ {
		walk[i] = cloneRawFrame(frames[walkStart+i])
	}

	idlePattern := []int{0, 0, 1, 0}
	for i, source := range idlePattern {
		out[idleStart+i] = cloneRawFrame(walk[source])
	}
	for i := 0; i < scurryFrames; i++ {
		out[scurryStart+i] = cloneRawFrame(walk[i%len(walk)])
	}
	nibblePattern := []int{0, 1, 2, 1, 0, 7}
	for i, source := range nibblePattern {
		out[nibbleStart+i] = cloneRawFrame(walk[source])
	}
	hopPattern := []int{0, 1, 2, 3, 4, 5}
	for i, source := range hopPattern {
		out[hopStart+i] = cloneRawFrame(walk[source])
	}
	return out
}

func cloneRawFrames(frames []rawFrame) []rawFrame {
	out := make([]rawFrame, len(frames))
	for i, frame := range frames {
		out[i] = cloneRawFrame(frame)
	}
	return out
}

func cloneRawFrame(frame rawFrame) rawFrame {
	return rawFrame{
		img:      cloneRGBA(frame.img),
		content:  frame.content,
		valid:    frame.valid,
		warnings: append([]string{}, frame.warnings...),
	}
}

func appendFrameFileReports(id string, specs []frameSpec, rawFrames []rawFrame, rep *report, outDir string) {
	for _, spec := range specs {
		frame := rawFrames[spec.Index]
		rep.Cells = append(rep.Cells, cellReport{
			Variant:  id,
			Frame:    spec.Index,
			Sheet:    filepath.ToSlash(filepath.Join("wild_agouti", frameFileName(spec))),
			Source:   toRectJSON(frame.img.Bounds()),
			Content:  toRectJSON(frame.content),
			Output:   filepath.ToSlash(filepath.Join(outDir, "degu_"+id+".png")),
			Warnings: frame.warnings,
		})
	}
}

func loadCoatGuides(dir string) map[string]coatGuide {
	out := map[string]coatGuide{}
	for _, id := range variants {
		path := filepath.Join(dir, id+".png")
		src, err := openPNG(path)
		if err != nil {
			continue
		}
		img := imageToCleanFrame(src)
		out[id] = coatGuide{img: img, content: alphaBounds(img)}
	}
	return out
}

func cloneAndTintFrames(frames []rawFrame, id string, guide coatGuide) []rawFrame {
	out := make([]rawFrame, len(frames))
	for i, frame := range frames {
		img := cloneRGBA(frame.img)
		if id != "wild_agouti" {
			img = tintFrame(img, id, guide)
		}
		content := alphaBounds(img)
		out[i] = rawFrame{
			img:      img,
			content:  content,
			valid:    !content.Empty(),
			warnings: append([]string{}, frame.warnings...),
		}
	}
	return out
}

func importActionSheets(dir string, sheets map[string]*image.RGBA, rep *report, outDir string) {
	rawByVariant := make(map[string][]rawFrame, len(variants))
	for _, id := range variants {
		rawByVariant[id] = make([]rawFrame, totalFrames)
	}

	for _, spec := range rows {
		path := filepath.Join(dir, "imagegen-"+spec.Name+".png")
		src, err := openPNG(path)
		if err != nil {
			log.Fatalf("open %s: %v", path, err)
		}
		b := src.Bounds()
		rep.SourceWidth = max(rep.SourceWidth, b.Dx())
		rep.SourceHeight = max(rep.SourceHeight, b.Dy())
		for r, id := range variants {
			rowRect := proportionalRowRect(b, r, len(variants))
			rowImage := copyCell(src, rowRect)
			rowClean := removeEdgeBackground(rowImage)
			segments := proportionalSegments(rowClean.Bounds(), spec.Cols)

			rawFrames := make([]rawFrame, spec.Cols)
			for c := 0; c < spec.Cols; c++ {
				segment := segments[c]
				cell := copyCell(rowClean, segment)
				cell = removeEdgeBackground(cell)
				cell = keepPrimaryArtwork(cell)
				content := alphaBounds(cell)
				cellWarnings := validateCell(content, cell.Bounds())
				valid := isValidContent(content, cell)
				if !valid {
					cellWarnings = append(cellWarnings, "invalid action frame; nearest valid frame will replace it")
				}
				rawFrames[c] = rawFrame{
					img:      cell,
					content:  content,
					valid:    valid,
					warnings: cellWarnings,
				}
				rep.Cells = append(rep.Cells, cellReport{
					Variant:  id,
					Frame:    spec.Offset + c,
					Sheet:    filepath.Base(path) + ":" + spec.Name,
					Source:   toRectJSON(segment.Add(rowRect.Min)),
					Content:  toRectJSON(content),
					Output:   filepath.ToSlash(filepath.Join(outDir, "degu_"+id+".png")),
					Warnings: cellWarnings,
				})
			}
			repairFrames(rawFrames)
			for c := 0; c < spec.Cols; c++ {
				rawByVariant[id][spec.Offset+c] = rawFrames[c]
			}
		}
	}

	for _, id := range variants {
		rawFrames := rawByVariant[id]
		repairFrames(rawFrames)
		sheet := image.NewRGBA(image.Rect(0, 0, frameW*totalFrames, frameH))
		for _, row := range rows {
			scale := commonActionScale(rawFrames, row)
			for i := 0; i < row.Cols; i++ {
				frameIndex := row.Offset + i
				frame := rawFrames[frameIndex]
				fitted := fitToFrameWithScale(frame.img, frame.content, scale)
				draw.Draw(sheet, image.Rect(frameIndex*frameW, 0, (frameIndex+1)*frameW, frameH), fitted, image.Point{}, draw.Over)
			}
		}
		sheets[id] = sheet
	}
}

func importPoses(dir string, sheets map[string]*image.RGBA, rep *report, outDir string) {
	for _, id := range variants {
		path := filepath.Join(dir, id+".png")
		src, err := openPNG(path)
		if err != nil {
			log.Fatalf("open %s: %v", path, err)
		}
		if rep.SourceWidth == 0 {
			rep.SourceWidth = src.Bounds().Dx()
			rep.SourceHeight = src.Bounds().Dy()
		}
		base := imageToCleanFrame(src)
		rawFrames := animatePoseFrames(base)
		sheet := image.NewRGBA(image.Rect(0, 0, frameW*totalFrames, frameH))
		for c, frame := range rawFrames {
			draw.Draw(sheet, image.Rect(c*frameW, 0, (c+1)*frameW, frameH), frame, image.Point{}, draw.Over)
			rep.Cells = append(rep.Cells, cellReport{
				Variant: id,
				Frame:   c,
				Sheet:   filepath.Base(path),
				Source:  toRectJSON(src.Bounds()),
				Content: toRectJSON(alphaBounds(frame)),
				Output:  filepath.ToSlash(filepath.Join(outDir, "degu_"+id+".png")),
			})
		}
		sheets[id] = sheet
	}
}

func imageToCleanFrame(src image.Image) *image.RGBA {
	b := src.Bounds()
	cell := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(cell, cell.Bounds(), src, b.Min, draw.Src)
	cell = cleanArtwork(cell)
	content := alphaBounds(cell)
	return fitToFrameSmooth(cell, content)
}

func imageToCleanCell(src image.Image) *image.RGBA {
	b := src.Bounds()
	cell := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(cell, cell.Bounds(), src, b.Min, draw.Src)
	return cleanArtwork(cell)
}

func cleanArtwork(src *image.RGBA) *image.RGBA {
	src = removeEdgeBackground(src)
	src = keepPrimaryArtwork(src)
	return src
}

type coatPalette struct {
	Dark   color.RGBA
	Base   color.RGBA
	Light  color.RGBA
	Patch  color.RGBA
	Pied   bool
	Creamy bool
}

func tintFrame(src *image.RGBA, id string, guide coatGuide) *image.RGBA {
	palette := paletteFor(id)
	b := src.Bounds()
	content := alphaBounds(src)
	dst := image.NewRGBA(b)
	if content.Empty() {
		return dst
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := src.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			lum := luminance(c)
			if lum < 30 {
				dst.SetRGBA(x, y, color.RGBA{R: uint8(max(4, int(c.R)/2)), G: uint8(max(4, int(c.G)/2)), B: uint8(max(4, int(c.B)/2)), A: c.A})
				continue
			}
			if isPinkPixel(c) {
				dst.SetRGBA(x, y, c)
				continue
			}
			xn := float64(x-content.Min.X) / float64(max(1, content.Dx()))
			yn := float64(y-content.Min.Y) / float64(max(1, content.Dy()))
			bodyPalette := palette
			if palette.Pied && (guidePiedPatch(guide, xn, yn) || (guide.img == nil && piedPatch(id, xn, yn))) {
				if palette.Creamy {
					bodyPalette = coatPalette{
						Dark:  color.RGBA{R: 205, G: 190, B: 162, A: 255},
						Base:  color.RGBA{R: 240, G: 232, B: 210, A: 255},
						Light: color.RGBA{R: 255, G: 252, B: 240, A: 255},
					}
				} else {
					bodyPalette = coatPalette{
						Dark:  color.RGBA{R: 162, G: 154, B: 140, A: 255},
						Base:  color.RGBA{R: 228, G: 222, B: 206, A: 255},
						Light: color.RGBA{R: 255, G: 250, B: 235, A: 255},
					}
				}
			}
			out := shadeColor(bodyPalette, lum)
			out.A = c.A
			dst.SetRGBA(x, y, out)
		}
	}
	return dst
}

func guidePiedPatch(guide coatGuide, xn, yn float64) bool {
	if guide.img == nil || guide.content.Empty() {
		return false
	}
	content := guide.content
	x := content.Min.X + clampInt(int(math.Round(xn*float64(content.Dx()-1))), 0, max(0, content.Dx()-1))
	y := content.Min.Y + clampInt(int(math.Round(yn*float64(content.Dy()-1))), 0, max(0, content.Dy()-1))
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			px := clampInt(x+dx, content.Min.X, content.Max.X-1)
			py := clampInt(y+dy, content.Min.Y, content.Max.Y-1)
			c := guide.img.RGBAAt(px, py)
			if isWhitePatchPixel(c) {
				return true
			}
		}
	}
	return false
}

func isWhitePatchPixel(c color.RGBA) bool {
	if c.A < 64 {
		return false
	}
	maxc := max3(c.R, c.G, c.B)
	minc := min3(c.R, c.G, c.B)
	return maxc > 170 && maxc-minc < 55
}

func paletteFor(id string) coatPalette {
	switch id {
	case "black":
		return coatPalette{
			Dark:  color.RGBA{R: 8, G: 9, B: 10, A: 255},
			Base:  color.RGBA{R: 34, G: 36, B: 38, A: 255},
			Light: color.RGBA{R: 86, G: 88, B: 90, A: 255},
		}
	case "blue":
		return coatPalette{
			Dark:  color.RGBA{R: 58, G: 61, B: 61, A: 255},
			Base:  color.RGBA{R: 108, G: 111, B: 106, A: 255},
			Light: color.RGBA{R: 169, G: 168, B: 158, A: 255},
		}
	case "gray":
		return coatPalette{
			Dark:  color.RGBA{R: 69, G: 70, B: 69, A: 255},
			Base:  color.RGBA{R: 124, G: 123, B: 118, A: 255},
			Light: color.RGBA{R: 190, G: 186, B: 174, A: 255},
		}
	case "white_cream":
		return coatPalette{
			Dark:  color.RGBA{R: 170, G: 150, B: 116, A: 255},
			Base:  color.RGBA{R: 234, G: 222, B: 196, A: 255},
			Light: color.RGBA{R: 255, G: 249, B: 230, A: 255},
		}
	case "sand_champagne":
		return coatPalette{
			Dark:  color.RGBA{R: 114, G: 82, B: 48, A: 255},
			Base:  color.RGBA{R: 194, G: 157, B: 101, A: 255},
			Light: color.RGBA{R: 239, G: 211, B: 154, A: 255},
		}
	case "chocolate":
		return coatPalette{
			Dark:  color.RGBA{R: 43, G: 25, B: 16, A: 255},
			Base:  color.RGBA{R: 92, G: 57, B: 34, A: 255},
			Light: color.RGBA{R: 146, G: 98, B: 60, A: 255},
		}
	case "black_pied":
		return coatPalette{
			Dark:  color.RGBA{R: 8, G: 9, B: 10, A: 255},
			Base:  color.RGBA{R: 34, G: 36, B: 38, A: 255},
			Light: color.RGBA{R: 86, G: 88, B: 90, A: 255},
			Pied:  true,
		}
	case "agouti_pied":
		return coatPalette{
			Dark:  color.RGBA{R: 76, G: 48, B: 28, A: 255},
			Base:  color.RGBA{R: 138, G: 94, B: 52, A: 255},
			Light: color.RGBA{R: 214, G: 166, B: 94, A: 255},
			Pied:  true,
		}
	case "blue_pied":
		return coatPalette{
			Dark:  color.RGBA{R: 58, G: 61, B: 61, A: 255},
			Base:  color.RGBA{R: 108, G: 111, B: 106, A: 255},
			Light: color.RGBA{R: 169, G: 168, B: 158, A: 255},
			Pied:  true,
		}
	case "cream_pied":
		return coatPalette{
			Dark:   color.RGBA{R: 170, G: 150, B: 116, A: 255},
			Base:   color.RGBA{R: 234, G: 222, B: 196, A: 255},
			Light:  color.RGBA{R: 255, G: 249, B: 230, A: 255},
			Pied:   true,
			Creamy: true,
		}
	default:
		return coatPalette{
			Dark:  color.RGBA{R: 76, G: 48, B: 28, A: 255},
			Base:  color.RGBA{R: 138, G: 94, B: 52, A: 255},
			Light: color.RGBA{R: 214, G: 166, B: 94, A: 255},
		}
	}
}

func shadeColor(palette coatPalette, lum int) color.RGBA {
	t := clamp01((float64(lum) - 35) / 185)
	if t < 0.55 {
		return mixColor(palette.Dark, palette.Base, t/0.55)
	}
	return mixColor(palette.Base, palette.Light, (t-0.55)/0.45)
}

func mixColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp01(t)
	return color.RGBA{
		R: uint8(math.Round(float64(a.R)*(1-t) + float64(b.R)*t)),
		G: uint8(math.Round(float64(a.G)*(1-t) + float64(b.G)*t)),
		B: uint8(math.Round(float64(a.B)*(1-t) + float64(b.B)*t)),
		A: uint8(math.Round(float64(a.A)*(1-t) + float64(b.A)*t)),
	}
}

func piedPatch(id string, xn, yn float64) bool {
	if yn < 0.22 || yn > 0.96 || xn < 0.10 {
		return false
	}
	switch id {
	case "black_pied":
		return ellipsePatch(xn, yn, 0.35, 0.62, 0.16, 0.22) ||
			ellipsePatch(xn, yn, 0.68, 0.42, 0.18, 0.20) ||
			ellipsePatch(xn, yn, 0.18, 0.72, 0.10, 0.14)
	case "agouti_pied":
		return ellipsePatch(xn, yn, 0.42, 0.50, 0.18, 0.24) ||
			ellipsePatch(xn, yn, 0.72, 0.68, 0.18, 0.18) ||
			ellipsePatch(xn, yn, 0.23, 0.38, 0.09, 0.13)
	case "blue_pied":
		return ellipsePatch(xn, yn, 0.30, 0.44, 0.14, 0.18) ||
			ellipsePatch(xn, yn, 0.58, 0.64, 0.22, 0.22) ||
			ellipsePatch(xn, yn, 0.82, 0.46, 0.10, 0.16)
	case "cream_pied":
		return ellipsePatch(xn, yn, 0.48, 0.58, 0.20, 0.24) ||
			ellipsePatch(xn, yn, 0.74, 0.38, 0.16, 0.18)
	default:
		return ellipsePatch(xn, yn, 0.55, 0.55, 0.22, 0.24)
	}
}

func ellipsePatch(x, y, cx, cy, rx, ry float64) bool {
	dx := (x - cx) / rx
	dy := (y - cy) / ry
	return dx*dx+dy*dy <= 1
}

func luminance(c color.RGBA) int {
	return int(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B))
}

func isPinkPixel(c color.RGBA) bool {
	return c.R > 145 && c.G > 80 && c.B > 70 && c.R > c.G+18 && c.R > c.B+12
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func animatePoseFrames(base *image.RGBA) []*image.RGBA {
	frames := make([]*image.RGBA, totalFrames)
	for i := range frames {
		frames[i] = shiftFrame(base, motionSetShift(0, i))
	}
	return frames
}

func shiftFrame(src *image.RGBA, shift image.Point) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	draw.Draw(dst, dst.Bounds().Add(shift), src, image.Point{}, draw.Over)
	return dst
}

func importFullAtlas(src image.Image, sheetName string, sheets map[string]*image.RGBA, rep *report, outDir string) {
	b := src.Bounds()
	if b.Dx()%atlasCols != 0 {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s width is not divisible by %d; proportional grid rounding may be used", sheetName, atlasCols))
	}
	if b.Dy()%atlasRows != 0 {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s height is not divisible by %d; proportional row rounding may be used", sheetName, atlasRows))
	}
	for r, id := range variants {
		sheet := image.NewRGBA(image.Rect(0, 0, frameW*totalFrames, frameH))
		rawFrames := make([]rawFrame, totalFrames)
		rowRect := proportionalRowRect(b, r, atlasRows)
		rowImage := copyCell(src, rowRect)
		rowClean := removeEdgeBackground(rowImage)
		segments := horizontalSegments(rowClean, totalFrames)
		if len(segments) != totalFrames {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s:%s detected %d segments; proportional fallback was used", sheetName, id, len(segments)))
			segments = proportionalSegments(rowClean.Bounds(), totalFrames)
		}
		for c := 0; c < totalFrames; c++ {
			cellRect := expandRect(segments[c], rowClean.Bounds(), 10)
			cell := copyCell(rowClean, cellRect)
			cell = keepPrimaryArtwork(cell)
			content := alphaBounds(cell)
			cellWarnings := validateCell(content, cell.Bounds())
			rawFrames[c] = rawFrame{
				img:      cell,
				content:  content,
				valid:    isValidContent(content, cell),
				warnings: cellWarnings,
			}
			rep.Cells = append(rep.Cells, cellReport{
				Variant:  id,
				Frame:    c,
				Sheet:    sheetName,
				Source:   toRectJSON(cellRect.Add(rowRect.Min)),
				Content:  toRectJSON(content),
				Output:   filepath.ToSlash(filepath.Join(outDir, "degu_"+id+".png")),
				Warnings: cellWarnings,
			})
		}
		repairFrames(rawFrames)
		for c := 0; c < totalFrames; c++ {
			fitted := fitToFrameFixed(rawFrames[c].img, rawFrames[c].content)
			draw.Draw(sheet, image.Rect(c*frameW, 0, (c+1)*frameW, frameH), fitted, image.Point{}, draw.Over)
		}
		sheets[id] = sheet
	}
}

func importCoatSheet(src image.Image, variant string, sheetName string, sheet *image.RGBA, rep *report, outDir string) {
	b := src.Bounds()
	if b.Dx()%coatCols != 0 {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s width is not divisible by %d; proportional grid rounding may be used", sheetName, coatCols))
	}
	if b.Dy()%coatRows != 0 {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s height is not divisible by %d; proportional row rounding may be used", sheetName, coatRows))
	}
	for _, spec := range rows {
		rowRect := proportionalRowRect(b, spec.Row, coatRows)
		rowImage := copyCell(src, rowRect)
		rowClean := removeEdgeBackground(rowImage)
		segments := proportionalSegments(rowClean.Bounds(), coatCols)
		rawFrames := make([]rawFrame, spec.Cols)
		for c := 0; c < spec.Cols; c++ {
			sourceIndex := resampleIndex(c, spec.Cols, len(segments))
			segment := expandRect(segments[sourceIndex], rowClean.Bounds(), 10)
			cell := copyCell(rowClean, segment)
			cell = keepPrimaryArtwork(cell)
			content := alphaBounds(cell)
			cellWarnings := validateCell(content, cell.Bounds())
			rawFrames[c] = rawFrame{
				img:      cell,
				content:  content,
				valid:    isValidContent(content, cell),
				warnings: cellWarnings,
			}
			rep.Cells = append(rep.Cells, cellReport{
				Variant:  variant,
				Frame:    spec.Offset + c,
				Sheet:    sheetName + ":" + spec.Name,
				Source:   toRectJSON(segment.Add(rowRect.Min)),
				Content:  toRectJSON(content),
				Output:   filepath.ToSlash(filepath.Join(outDir, "degu_"+variant+".png")),
				Warnings: cellWarnings,
			})
		}
		repairFrames(rawFrames)
		for c := 0; c < spec.Cols; c++ {
			frameIndex := spec.Offset + c
			fitted := fitToFrameFixed(rawFrames[c].img, rawFrames[c].content)
			draw.Draw(sheet, image.Rect(frameIndex*frameW, 0, (frameIndex+1)*frameW, frameH), fitted, image.Point{}, draw.Over)
		}
	}
}

func opaquePixels(img *image.RGBA) int {
	b := img.Bounds()
	count := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A > 0 {
				count++
			}
		}
	}
	return count
}

func isValidContent(content image.Rectangle, img *image.RGBA) bool {
	if content.Empty() {
		return false
	}
	w, h := content.Dx(), content.Dy()
	pixels := opaquePixels(img)
	if pixels < 320 {
		return false
	}
	if w < 20 || h < 16 {
		return false
	}
	if float64(w)/float64(h) < 0.55 || float64(w)/float64(h) > 3.8 {
		return false
	}
	return true
}

func repairFrames(frames []rawFrame) {
	lastValid := -1
	for i := range frames {
		if frames[i].valid {
			lastValid = i
			continue
		}
		replacement := nearestValidFrame(frames, i, lastValid)
		if replacement >= 0 {
			frames[i].img = cloneRGBA(frames[replacement].img)
			frames[i].content = frames[replacement].content
			frames[i].valid = true
			frames[i].warnings = append(frames[i].warnings, "invalid frame replaced with nearest valid frame")
		}
	}
}

func nearestValidFrame(frames []rawFrame, index int, lastValid int) int {
	if lastValid >= 0 {
		return lastValid
	}
	for i := index + 1; i < len(frames); i++ {
		if frames[i].valid {
			return i
		}
	}
	return -1
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

func toRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return cloneRGBA(rgba)
	}
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

func openPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func proportionalRowRect(bounds image.Rectangle, r int, rowCount int) image.Rectangle {
	h := bounds.Dy()
	y0 := bounds.Min.Y + int(math.Round(float64(r*h)/float64(rowCount)))
	y1 := bounds.Min.Y + int(math.Round(float64((r+1)*h)/float64(rowCount)))
	return image.Rect(bounds.Min.X, y0, bounds.Max.X, y1)
}

func proportionalSegments(bounds image.Rectangle, count int) []image.Rectangle {
	segs := make([]image.Rectangle, 0, count)
	w := bounds.Dx()
	for c := 0; c < count; c++ {
		x0 := bounds.Min.X + int(math.Round(float64(c*w)/float64(count)))
		x1 := bounds.Min.X + int(math.Round(float64((c+1)*w)/float64(count)))
		segs = append(segs, image.Rect(x0, bounds.Min.Y, x1, bounds.Max.Y))
	}
	return segs
}

func horizontalSegments(img *image.RGBA, expectedCount int) []image.Rectangle {
	b := img.Bounds()
	has := make([]bool, b.Dx())
	for x := b.Min.X; x < b.Max.X; x++ {
		count := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if img.RGBAAt(x, y).A > 0 {
				count++
			}
		}
		has[x-b.Min.X] = count >= 2
	}
	runs := []image.Rectangle{}
	inRun := false
	start := 0
	for i, ok := range has {
		if ok && !inRun {
			inRun = true
			start = i
		}
		if (!ok || i == len(has)-1) && inRun {
			end := i
			if ok && i == len(has)-1 {
				end = i + 1
			}
			if end-start >= 2 {
				runs = append(runs, image.Rect(b.Min.X+start, b.Min.Y, b.Min.X+end, b.Max.Y))
			}
			inRun = false
		}
	}
	runs = closeHorizontalGaps(runs, max(8, b.Dx()/(expectedCount*10)))
	runs = mergeToCount(runs, expectedCount)
	out := make([]image.Rectangle, 0, len(runs))
	for _, run := range runs {
		if content := alphaBoundsInRect(img, run); !content.Empty() {
			out = append(out, content)
		}
	}
	return out
}

func resampleIndex(targetIndex int, targetCount int, sourceCount int) int {
	if sourceCount <= 1 || targetCount <= 1 {
		return 0
	}
	idx := int(math.Round(float64(targetIndex) * float64(sourceCount-1) / float64(targetCount-1)))
	if idx < 0 {
		return 0
	}
	if idx >= sourceCount {
		return sourceCount - 1
	}
	return idx
}

func closeHorizontalGaps(runs []image.Rectangle, maxGap int) []image.Rectangle {
	if len(runs) == 0 {
		return runs
	}
	out := []image.Rectangle{runs[0]}
	for _, run := range runs[1:] {
		last := out[len(out)-1]
		if run.Min.X-last.Max.X <= maxGap {
			out[len(out)-1] = image.Rect(last.Min.X, min(last.Min.Y, run.Min.Y), run.Max.X, max(last.Max.Y, run.Max.Y))
			continue
		}
		out = append(out, run)
	}
	return out
}

func mergeToCount(runs []image.Rectangle, target int) []image.Rectangle {
	for len(runs) > target {
		best := 0
		bestGap := runs[1].Min.X - runs[0].Max.X
		for i := 1; i < len(runs)-1; i++ {
			gap := runs[i+1].Min.X - runs[i].Max.X
			if gap < bestGap {
				best = i
				bestGap = gap
			}
		}
		merged := image.Rect(runs[best].Min.X, min(runs[best].Min.Y, runs[best+1].Min.Y), runs[best+1].Max.X, max(runs[best].Max.Y, runs[best+1].Max.Y))
		runs = append(runs[:best], append([]image.Rectangle{merged}, runs[best+2:]...)...)
	}
	return runs
}

func alphaBoundsInRect(img *image.RGBA, rect image.Rectangle) image.Rectangle {
	rect = rect.Intersect(img.Bounds())
	minX, minY := rect.Max.X, rect.Max.Y
	maxX, maxY := rect.Min.X, rect.Min.Y
	found := false
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
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

func expandRect(rect, bounds image.Rectangle, amount int) image.Rectangle {
	return image.Rect(rect.Min.X-amount, rect.Min.Y-amount, rect.Max.X+amount, rect.Max.Y+amount).Intersect(bounds)
}

func copyCell(src image.Image, rect image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
	return dst
}

func removeEdgeBackground(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, image.Point{}, draw.Src)
	seen := make([]bool, b.Dx()*b.Dy())
	queue := make([]image.Point, 0, b.Dx()*2+b.Dy()*2)
	push := func(p image.Point) {
		if p.X < b.Min.X || p.X >= b.Max.X || p.Y < b.Min.Y || p.Y >= b.Max.Y {
			return
		}
		i := (p.Y-b.Min.Y)*b.Dx() + (p.X - b.Min.X)
		if seen[i] {
			return
		}
		seen[i] = true
		if isBackground(dst.RGBAAt(p.X, p.Y)) {
			queue = append(queue, p)
		}
	}
	for x := b.Min.X; x < b.Max.X; x++ {
		push(image.Pt(x, b.Min.Y))
		push(image.Pt(x, b.Max.Y-1))
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		push(image.Pt(b.Min.X, y))
		push(image.Pt(b.Max.X-1, y))
	}
	for len(queue) > 0 {
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		dst.SetRGBA(p.X, p.Y, color.RGBA{})
		push(image.Pt(p.X+1, p.Y))
		push(image.Pt(p.X-1, p.Y))
		push(image.Pt(p.X, p.Y+1))
		push(image.Pt(p.X, p.Y-1))
	}
	return dst
}

func keepPrimaryArtwork(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	visited := make([]bool, b.Dx()*b.Dy())
	type component struct {
		points []image.Point
		bounds image.Rectangle
	}
	components := []component{}
	index := func(x, y int) int {
		return (y-b.Min.Y)*b.Dx() + (x - b.Min.X)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if visited[index(x, y)] || src.RGBAAt(x, y).A == 0 {
				continue
			}
			queue := []image.Point{image.Pt(x, y)}
			visited[index(x, y)] = true
			points := []image.Point{}
			minX, minY, maxX, maxY := x, y, x+1, y+1
			for len(queue) > 0 {
				p := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				points = append(points, p)
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
						if nx < b.Min.X || nx >= b.Max.X || ny < b.Min.Y || ny >= b.Max.Y {
							continue
						}
						i := index(nx, ny)
						if visited[i] || src.RGBAAt(nx, ny).A == 0 {
							continue
						}
						visited[i] = true
						queue = append(queue, image.Pt(nx, ny))
					}
				}
			}
			components = append(components, component{points: points, bounds: image.Rect(minX, minY, maxX, maxY)})
		}
	}
	if len(components) == 0 {
		return src
	}
	largestIndex := 0
	for i, c := range components {
		if len(c.points) > len(components[largestIndex].points) {
			largestIndex = i
		}
	}
	largest := len(components[largestIndex].points)
	mainBounds := components[largestIndex].bounds
	dst := image.NewRGBA(b)
	for i, c := range components {
		if i == largestIndex || keepComponent(c.bounds, len(c.points), largest, mainBounds) {
			for _, p := range c.points {
				dst.SetRGBA(p.X, p.Y, src.RGBAAt(p.X, p.Y))
			}
		}
	}
	return dst
}

func keepComponent(bounds image.Rectangle, area int, largest int, mainBounds image.Rectangle) bool {
	distance := rectDistance(bounds, mainBounds)
	close := distance <= max(24, mainBounds.Dx()/4)
	if close && area >= 12 && area >= largest/100 {
		return true
	}
	if distance <= 18 && bounds.Dx() >= 5 && bounds.Dy() >= 1 && area >= 5 {
		return true
	}
	return false
}

func rectDistance(a, b image.Rectangle) int {
	dx := 0
	if a.Max.X < b.Min.X {
		dx = b.Min.X - a.Max.X
	} else if b.Max.X < a.Min.X {
		dx = a.Min.X - b.Max.X
	}
	dy := 0
	if a.Max.Y < b.Min.Y {
		dy = b.Min.Y - a.Max.Y
	} else if b.Max.Y < a.Min.Y {
		dy = a.Min.Y - b.Max.Y
	}
	return int(math.Round(math.Hypot(float64(dx), float64(dy))))
}

func isBackground(c color.RGBA) bool {
	if c.A < 230 {
		return true
	}
	maxc := max3(c.R, c.G, c.B)
	minc := min3(c.R, c.G, c.B)
	return maxc > 205 && maxc-minc < 18
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

func validateCell(content, cell image.Rectangle) []string {
	if content.Empty() {
		return []string{"no non-transparent content found after background removal"}
	}
	warnings := []string{}
	if content.Min.X-cell.Min.X < 3 {
		warnings = append(warnings, "content touches left edge; tail or whiskers may be cropped")
	}
	if content.Min.Y-cell.Min.Y < 3 {
		warnings = append(warnings, "content touches top edge; ears may be cropped")
	}
	if cell.Max.X-content.Max.X < 3 {
		warnings = append(warnings, "content touches right edge; whiskers or nose may be cropped")
	}
	if cell.Max.Y-content.Max.Y < 3 {
		warnings = append(warnings, "content touches bottom edge; feet may be cropped")
	}
	return warnings
}

func fitToFrame(src *image.RGBA, content image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	if content.Empty() {
		return dst
	}
	cw, ch := content.Dx(), content.Dy()
	scale := math.Min(float64(frameW-padding*2)/float64(cw), float64(frameH-padding*2)/float64(ch))
	if scale <= 0 {
		return dst
	}
	outW := max(1, int(math.Round(float64(cw)*scale)))
	outH := max(1, int(math.Round(float64(ch)*scale)))
	offX := (frameW - outW) / 2
	offY := (frameH - outH) / 2
	for y := 0; y < outH; y++ {
		sy := content.Min.Y + min(ch-1, int(float64(y)/scale))
		for x := 0; x < outW; x++ {
			sx := content.Min.X + min(cw-1, int(float64(x)/scale))
			dst.SetRGBA(offX+x, offY+y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func fitToFrameFixed(src *image.RGBA, content image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	if content.Empty() {
		return dst
	}
	cw, ch := content.Dx(), content.Dy()
	scale := float64(targetH) / float64(ch)
	if int(math.Round(float64(cw)*scale)) > targetW {
		scale = float64(targetW) / float64(cw)
	}
	if scale <= 0 {
		return dst
	}
	outW := max(1, int(math.Round(float64(cw)*scale)))
	outH := max(1, int(math.Round(float64(ch)*scale)))
	offX := (frameW - outW) / 2
	offY := baselineY - outH
	if offY < padding {
		offY = padding
	}
	for y := 0; y < outH; y++ {
		sy := content.Min.Y + min(ch-1, int(float64(y)/scale))
		for x := 0; x < outW; x++ {
			sx := content.Min.X + min(cw-1, int(float64(x)/scale))
			dst.SetRGBA(offX+x, offY+y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func fitToFrameSmooth(src *image.RGBA, content image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	if content.Empty() {
		return dst
	}
	cw, ch := content.Dx(), content.Dy()
	scale := float64(targetH) / float64(ch)
	if int(math.Round(float64(cw)*scale)) > targetW {
		scale = float64(targetW) / float64(cw)
	}
	if scale <= 0 {
		return dst
	}
	outW := max(1, int(math.Round(float64(cw)*scale)))
	outH := max(1, int(math.Round(float64(ch)*scale)))
	offX := (frameW - outW) / 2
	offY := baselineY - outH
	if offY < padding {
		offY = padding
	}
	srcContent := image.NewRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(srcContent, srcContent.Bounds(), src, content.Min, draw.Src)
	target := image.Rect(offX, offY, offX+outW, offY+outH)
	xdraw.CatmullRom.Scale(dst, target, srcContent, srcContent.Bounds(), draw.Over, nil)
	cleanFittedFrame(dst)
	return dst
}

func commonActionScale(frames []rawFrame, row rowSpec) float64 {
	maxW, maxH := 0, 0
	for i := 0; i < row.Cols; i++ {
		frame := frames[row.Offset+i]
		if frame.content.Empty() {
			continue
		}
		maxW = max(maxW, frame.content.Dx())
		maxH = max(maxH, frame.content.Dy())
	}
	if maxW == 0 || maxH == 0 {
		return 1
	}
	scale := float64(targetH) / float64(maxH)
	if int(math.Round(float64(maxW)*scale)) > targetW {
		scale = float64(targetW) / float64(maxW)
	}
	if scale <= 0 {
		return 1
	}
	return scale
}

func fitToFrameWithScale(src *image.RGBA, content image.Rectangle, scale float64) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	if content.Empty() || scale <= 0 {
		return dst
	}
	cw, ch := content.Dx(), content.Dy()
	outW := max(1, int(math.Round(float64(cw)*scale)))
	outH := max(1, int(math.Round(float64(ch)*scale)))
	if outW > frameW-padding*2 || outH > frameH-padding*2 {
		limit := math.Min(float64(frameW-padding*2)/float64(outW), float64(frameH-padding*2)/float64(outH))
		outW = max(1, int(math.Round(float64(outW)*limit)))
		outH = max(1, int(math.Round(float64(outH)*limit)))
	}
	offX := (frameW - outW) / 2
	offY := baselineY - outH
	if offY < padding {
		offY = padding
	}
	srcContent := image.NewRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(srcContent, srcContent.Bounds(), src, content.Min, draw.Src)
	target := image.Rect(offX, offY, offX+outW, offY+outH)
	xdraw.CatmullRom.Scale(dst, target, srcContent, srcContent.Bounds(), draw.Over, nil)
	cleanFittedFrame(dst)
	return dst
}

func cleanFittedFrame(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			if c.A < 16 || isCoolColorFringe(c) {
				img.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
}

func isCoolColorFringe(c color.RGBA) bool {
	if c.A >= 96 {
		return false
	}
	return (c.B > 130 && c.R < 110) || (c.G > 150 && c.R < 110 && c.B > 100)
}

func writePreview(path string, sheets map[string]*image.RGBA) {
	const (
		previewW  = 960
		previewH  = 360
		taskbarH  = 62
		petScale  = 2
		petBaseY  = previewH - taskbarH + 2
		layerBase = previewH - taskbarH - 45
	)
	preview := image.NewRGBA(image.Rect(0, 0, previewW, previewH))
	fillRect(preview, preview.Bounds(), color.RGBA{R: 253, G: 253, B: 250, A: 255})
	fillRect(preview, image.Rect(0, layerBase, previewW, previewH-taskbarH), color.RGBA{R: 246, G: 248, B: 244, A: 255})
	fillRect(preview, image.Rect(0, previewH-taskbarH, previewW, previewH), color.RGBA{R: 239, G: 242, B: 245, A: 255})
	fillRect(preview, image.Rect(0, previewH-taskbarH, previewW, previewH-taskbarH+2), color.RGBA{R: 216, G: 222, B: 226, A: 255})
	drawTaskbarPreview(preview, previewH-taskbarH)

	pets := []struct {
		id    string
		frame int
		x     int
		lane  int
		flip  bool
	}{
		{"wild_agouti", 8, 28, 0, false},
		{"blue", 12, 170, 8, false},
		{"gray", 4, 318, 3, false},
		{"black_pied", 20, 474, 11, false},
		{"sand_champagne", 6, 630, 0, true},
		{"agouti_pied", 22, 782, 7, true},
	}
	for _, pet := range pets {
		sheet := sheets[pet.id]
		if sheet == nil {
			continue
		}
		sprite := previewFrame(sheet, pet.frame, petScale, pet.flip)
		y := petBaseY - sprite.Bounds().Dy() - pet.lane
		drawPreviewShadow(preview, pet.x+sprite.Bounds().Dx()/2, petBaseY-pet.lane-5, 46, 8)
		draw.Draw(preview, image.Rect(pet.x, y, pet.x+sprite.Bounds().Dx(), y+sprite.Bounds().Dy()), sprite, image.Point{}, draw.Over)
	}
	drawPreviewBubble(preview, 410, 126)
	writePNG(path, preview)
}

func fillRect(dst *image.RGBA, rect image.Rectangle, c color.RGBA) {
	draw.Draw(dst, rect.Intersect(dst.Bounds()), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func previewFrame(sheet *image.RGBA, frame int, scale int, flip bool) *image.RGBA {
	frame = clampInt(frame, 0, totalFrames-1)
	srcRect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
	frameImg := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	if !flip {
		draw.Draw(frameImg, frameImg.Bounds(), sheet, srcRect.Min, draw.Src)
	} else {
		for y := 0; y < frameH; y++ {
			for x := 0; x < frameW; x++ {
				frameImg.SetRGBA(frameW-1-x, y, sheet.RGBAAt(srcRect.Min.X+x, srcRect.Min.Y+y))
			}
		}
	}
	return scaleNearest(frameImg, scale)
}

func drawTaskbarPreview(dst *image.RGBA, top int) {
	startX := 24
	for i := 0; i < 5; i++ {
		x := startX + i*42
		fillRect(dst, image.Rect(x, top+18, x+26, top+44), color.RGBA{R: 255, G: 255, B: 255, A: 255})
		fillRect(dst, image.Rect(x+7, top+25, x+19, top+37), []color.RGBA{
			{R: 51, G: 91, B: 128, A: 255},
			{R: 50, G: 112, B: 80, A: 255},
			{R: 194, G: 139, B: 55, A: 255},
			{R: 92, G: 102, B: 114, A: 255},
			{R: 126, G: 82, B: 52, A: 255},
		}[i])
	}
	fillRect(dst, image.Rect(820, top+21, 934, top+25), color.RGBA{R: 210, G: 216, B: 221, A: 255})
	fillRect(dst, image.Rect(820, top+35, 900, top+39), color.RGBA{R: 210, G: 216, B: 221, A: 255})
}

func drawPreviewShadow(dst *image.RGBA, cx, cy, rx, ry int) {
	c := color.RGBA{R: 223, G: 228, B: 221, A: 255}
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1 && image.Pt(x, y).In(dst.Bounds()) {
				dst.SetRGBA(x, y, c)
			}
		}
	}
}

func drawPreviewBubble(dst *image.RGBA, x, y int) {
	fillRect(dst, image.Rect(x+3, y+3, x+57, y+39), color.RGBA{R: 224, G: 229, B: 224, A: 255})
	fillRect(dst, image.Rect(x, y, x+54, y+34), color.RGBA{R: 255, G: 255, B: 252, A: 255})
	fillRect(dst, image.Rect(x+1, y+1, x+53, y+3), color.RGBA{R: 63, G: 97, B: 78, A: 230})
	fillRect(dst, image.Rect(x+1, y+31, x+53, y+33), color.RGBA{R: 63, G: 97, B: 78, A: 230})
	fillRect(dst, image.Rect(x+1, y+1, x+3, y+33), color.RGBA{R: 63, G: 97, B: 78, A: 230})
	fillRect(dst, image.Rect(x+51, y+1, x+53, y+33), color.RGBA{R: 63, G: 97, B: 78, A: 230})
	fillRect(dst, image.Rect(x+24, y+33, x+31, y+40), color.RGBA{R: 255, G: 255, B: 252, A: 255})
	fillRect(dst, image.Rect(x+18, y+11, x+24, y+18), color.RGBA{R: 221, G: 77, B: 96, A: 255})
	fillRect(dst, image.Rect(x+29, y+11, x+35, y+18), color.RGBA{R: 221, G: 77, B: 96, A: 255})
	fillRect(dst, image.Rect(x+21, y+18, x+33, y+23), color.RGBA{R: 221, G: 77, B: 96, A: 255})
	fillRect(dst, image.Rect(x+24, y+23, x+30, y+27), color.RGBA{R: 221, G: 77, B: 96, A: 255})
}

func writeSpriteSets(outDir string, id string, base *image.RGBA) {
	for set := 0; set < motionSets; set++ {
		sheet := variantMotionSheet(base, set)
		name := fmt.Sprintf("degu_%s_set%02d.png", id, set)
		writePNG(filepath.Join(outDir, name), sheet)
		if set == 0 {
			writePNG(filepath.Join(outDir, "degu_"+id+".png"), sheet)
		}
	}
}

func writeWheelSprite(sourcePath, outPath string) {
	src, err := openPNG(sourcePath)
	if err != nil {
		fmt.Println("warning: wheel source not found:", err)
		return
	}
	cleaned := cleanArtwork(toRGBA(src))
	content := alphaBounds(cleaned)
	if content.Empty() {
		fmt.Println("warning: wheel source has no visible content")
		return
	}
	dst := image.NewRGBA(image.Rect(0, 0, wheelW, wheelH))
	scale := math.Min(float64(wheelW-4)/float64(content.Dx()), float64(wheelH-4)/float64(content.Dy()))
	outW := max(1, int(math.Round(float64(content.Dx())*scale)))
	outH := max(1, int(math.Round(float64(content.Dy())*scale)))
	offX := (wheelW - outW) / 2
	offY := (wheelH - outH) / 2
	srcContent := image.NewRGBA(image.Rect(0, 0, content.Dx(), content.Dy()))
	draw.Draw(srcContent, srcContent.Bounds(), cleaned, content.Min, draw.Src)
	xdraw.CatmullRom.Scale(dst, image.Rect(offX, offY, offX+outW, offY+outH), srcContent, srcContent.Bounds(), draw.Over, nil)
	writePNG(outPath, dst)
}

func writeForageSprites(sourceDir, outDir string) {
	names := []string{"forage_hay", "forage_twig", "forage_seed"}
	for _, name := range names {
		path := filepath.Join(sourceDir, name+".png")
		src, err := openPNG(path)
		if err != nil {
			fmt.Println("warning: forage source not found:", err)
			continue
		}
		cleaned := cleanArtwork(toRGBA(src))
		content := alphaBounds(cleaned)
		if content.Empty() {
			fmt.Println("warning: forage source has no visible content:", path)
			continue
		}
		dst := image.NewRGBA(image.Rect(0, 0, forageW, forageH))
		scale := math.Min(float64(forageW-4)/float64(content.Dx()), float64(forageH-4)/float64(content.Dy()))
		outW := max(1, int(math.Round(float64(content.Dx())*scale)))
		outH := max(1, int(math.Round(float64(content.Dy())*scale)))
		offX := (forageW - outW) / 2
		offY := forageH - outH - 2
		srcContent := image.NewRGBA(image.Rect(0, 0, content.Dx(), content.Dy()))
		draw.Draw(srcContent, srcContent.Bounds(), cleaned, content.Min, draw.Src)
		xdraw.CatmullRom.Scale(dst, image.Rect(offX, offY, offX+outW, offY+outH), srcContent, srcContent.Bounds(), draw.Over, nil)
		cleanFittedFrame(dst)
		writePNG(filepath.Join(outDir, name+".png"), dst)
	}
}

func variantMotionSheet(base *image.RGBA, set int) *image.RGBA {
	if set == 0 {
		return cloneRGBA(base)
	}
	sheet := image.NewRGBA(base.Bounds())
	for frame := 0; frame < totalFrames; frame++ {
		srcRect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
		src := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
		draw.Draw(src, src.Bounds(), base, srcRect.Min, draw.Src)
		shift := motionSetShift(set, frame)
		shifted := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
		draw.Draw(shifted, shifted.Bounds().Add(shift), src, image.Point{}, draw.Over)
		dstRect := image.Rect(frame*frameW, 0, (frame+1)*frameW, frameH)
		draw.Draw(sheet, dstRect, shifted, image.Point{}, draw.Over)
	}
	return sheet
}

func motionSetShift(set int, frame int) image.Point {
	actionOffset := frame
	switch {
	case frame >= groomStart:
		actionOffset = frame - groomStart
	case frame >= standStart:
		actionOffset = frame - standStart
	case frame >= digStart:
		actionOffset = frame - digStart
	case frame >= eatStart:
		actionOffset = frame - eatStart
	case frame >= turnStart:
		actionOffset = frame - turnStart
	case frame >= hopStart:
		actionOffset = frame - hopStart
	case frame >= nibbleStart:
		actionOffset = frame - nibbleStart
	case frame >= scurryStart:
		actionOffset = frame - scurryStart
	case frame >= walkStart:
		actionOffset = frame - walkStart
	}
	phase := (set + actionOffset) % 4
	switch phase {
	case 0:
		return image.Point{}
	case 1:
		return image.Pt(0, -1)
	case 2:
		return image.Pt(1, 0)
	default:
		return image.Pt(-1, 0)
	}
}

func firstFrame(sheet *image.RGBA) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	draw.Draw(frame, frame.Bounds(), sheet, image.Point{}, draw.Src)
	return scaleNearest(frame, 2)
}

func scaleNearest(src *image.RGBA, scale int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.RGBAAt(x, y)
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.SetRGBA(x*scale+sx, y*scale+sy, c)
				}
			}
		}
	}
	return dst
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
}

func writeJSON(path string, value any) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	must(enc.Encode(value))
}

func writeICO(path string, src *image.RGBA) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	headerSize := 6 + 16
	imageSize := 40 + w*h*4 + ((w + 31) / 32 * 4 * h)
	writeU16(f, 0)
	writeU16(f, 1)
	writeU16(f, 1)
	f.Write([]byte{byte(w), byte(h), 0, 0})
	writeU16(f, 1)
	writeU16(f, 32)
	writeU32(f, uint32(imageSize))
	writeU32(f, uint32(headerSize))
	writeU32(f, 40)
	writeI32(f, int32(w))
	writeI32(f, int32(h*2))
	writeU16(f, 1)
	writeU16(f, 32)
	writeU32(f, 0)
	writeU32(f, uint32(w*h*4))
	writeI32(f, 0)
	writeI32(f, 0)
	writeU32(f, 0)
	writeU32(f, 0)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := src.RGBAAt(x, y)
			f.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	maskStride := (w + 31) / 32 * 4
	f.Write(make([]byte, maskStride*h))
}

func writeU16(f *os.File, v uint16) {
	f.Write([]byte{byte(v), byte(v >> 8)})
}

func writeU32(f *os.File, v uint32) {
	f.Write([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
}

func writeI32(f *os.File, v int32) {
	writeU32(f, uint32(v))
}

func toRectJSON(r image.Rectangle) rectJSON {
	return rectJSON{X: r.Min.X, Y: r.Min.Y, W: r.Dx(), H: r.Dy()}
}

func max3(a, b, c uint8) uint8 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func min3(a, b, c uint8) uint8 {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
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

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
