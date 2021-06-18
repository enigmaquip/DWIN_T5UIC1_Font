package main

import (
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func main() {
	filepath.WalkDir(".", ProcessFont)
}

func ProcessFont(path string, d fs.DirEntry, err error) error {
	widths := []int{6, 8, 10, 12, 14, 16, 20, 24, 28, 32}

	if err != nil {
		fmt.Printf("Failure accessing %q: %v\n", path, err)
		return err
	}
	if d.IsDir() {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".ttf" && ext != ".otf" {
		// Not a font
		return err
	}

	original, err2 := os.Open("0T5UIC1.HZK")
	if err2 != nil {
		fmt.Printf("Error opening original font file")
		return err
	}

	filename := d.Name()
	filename = fmt.Sprintf("0T5UIC1_%s.HZK", filename[:len(filename)-len(ext)])
	outfile, err2 := os.Create(filename)
	if err2 != nil {
		fmt.Printf("Unable to create file: %s\n", filename)
	}
	fmt.Printf("Processing: %s\n", path)
	fontfile, err2 := os.Open(path)
	if err2 != nil {
		fmt.Printf("Error opening font: %s\n", path)
		return err
	}
	ttffont, err2 := opentype.ParseReaderAt(fontfile)
	if err2 != nil {
		fmt.Printf("Error Parsing Font: %v\n", err2)
		return err
	}
	var filepos int64
	for size, width := range widths {
		height := 2 * width
		face, err2 := opentype.NewFace(ttffont, &opentype.FaceOptions{Size: float64(height), DPI: 72, Hinting: font.HintingNone})
		if err2 != nil {
			fmt.Printf("Error Creating TypeFace")
			return err
		}
		var numBytes int
		if width%8 > 0 {
			numBytes = ((width / 8) + 1)
		} else {
			numBytes = (width / 8)
		}
		// The first 32 characters in the ascii table are command characters and not used, we'll copy these instead of replacing them
		for i := 0; i < 32; i++ {
			written, err2 := io.CopyN(outfile, original, int64(numBytes*height))
			filepos += written
			if err2 != nil {
				return err
			}
		}
		for i := 32; i < 127; i++ {
			bounds, _ := font.BoundString(face, string(i))
			//fmt.Println(bounds)
			charWidth := bounds.Max.X.Round() + 2
			DotX := 2 - bounds.Min.X.Round()
			DotY := height - (width * 1 / 3)
			resize := false
			picWidth := width
			if charWidth > width {
				picWidth = charWidth
				resize = true
			}
			dst := image.NewGray(image.Rect(0, 0, picWidth, height))
			d := font.Drawer{
				Dst:  dst,
				Src:  image.White,
				Face: face,
				Dot:  fixed.P(DotX, DotY),
			}
			d.DrawString(string(i))

			if resize {
				newdest := image.NewGray(image.Rect(0, 0, width, height))
				draw.ApproxBiLinear.Scale(newdest, newdest.Rect, dst, dst.Rect, draw.Over, nil)
				dst = newdest
			}
			/*
				//command line ascii visualization test code
				const asciiArt = ".++8"
				buft := make([]byte, 0, height*(width+1))
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						c := asciiArt[dst.GrayAt(x, y).Y>>6]
						if c != '.' {
							// No-op.
						} else if x == DotX-1 {
							c = ']'
						} else if y == DotY-1 {
							c = '_'
						}
						buft = append(buft, c)
					}
					buft = append(buft, '\n')
				}
				os.Stdout.Write(buft)
				fmt.Println("")
			*/
			var buf []byte
			for y := 0; y < height; y++ {
				x := 0
				for j := 0; j < numBytes; j++ {
					xByte := byte(0)
					for k := 0; k < 8; k++ {
						if x < width {
							if dst.GrayAt(x, y).Y>>6 > 0 {
								xByte = xByte | 1
							}
						}
						if k < 7 {
							xByte = xByte << 1
						}
						x++
					}
					buf = append(buf, xByte)
				}
			}
			written, err2 := outfile.Write(buf)
			if err2 != nil {
				fmt.Println("Erorr writing to font file")
				return err
			}
			filepos += int64(written)
		}
		// Handle ascii 127 which is the del key (this is skipped on the last size)
		original.Seek(filepos, 0)
		if size < 9 {
			written, err2 := io.CopyN(outfile, original, int64(numBytes*height))
			filepos += written
			if err2 != nil {
				return err
			}
		}

	}
	// Copy the rest of the font file
	_, err2 = io.Copy(outfile, original)
	if err2 != nil {
		fmt.Printf("Error copying the rest of the font file: %v\n", err2)
		return err
	}
	outfile.Close()
	fontfile.Close()
	original.Close()
	return err
}
