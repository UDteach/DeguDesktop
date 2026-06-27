package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	_ "image/gif"
	_ "image/jpeg"

	"golang.org/x/image/draw"
)

func main() {
	srcPath := flag.String("src", "", "source image path")
	outPath := flag.String("out", "", "output PNG path")
	size := flag.Int("size", 256, "square output size")
	flag.Parse()

	if *srcPath == "" || *outPath == "" {
		fatalf("-src and -out are required")
	}
	if *size <= 0 || *size > 1024 {
		fatalf("-size must be between 1 and 1024")
	}
	if err := makeIconPNG(*srcPath, *outPath, *size); err != nil {
		fatalf("%v", err)
	}
}

func makeIconPNG(srcPath, outPath string, size int) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	src, _, err := image.Decode(srcFile)
	if err != nil {
		return err
	}
	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return fmt.Errorf("source image has empty bounds")
	}

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	fitW, fitH := fitInside(bounds.Dx(), bounds.Dy(), size)
	offsetX := (size - fitW) / 2
	offsetY := (size - fitH) / 2
	draw.CatmullRom.Scale(dst, image.Rect(offsetX, offsetY, offsetX+fitW, offsetY+fitH), src, bounds, draw.Over, nil)

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, dst)
}

func fitInside(width, height, maxSize int) (int, int) {
	if width >= height {
		return maxSize, max(1, height*maxSize/width)
	}
	return max(1, width*maxSize/height), maxSize
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
