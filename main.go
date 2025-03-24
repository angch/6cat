package main

import (
	"bufio"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/BurntSushi/graphics-go/graphics"
	"github.com/dolmen-go/kittyimg"
	"github.com/mattn/go-sixel"
)

// Reasonable max size, to reduce output
var maxWidth = 1280
var maxHeight = 720

func dump(f *os.File) error {
	buf := bufio.NewReader(f)
	_, _ = io.Copy(os.Stdout, buf)
	return nil
}

func render(filename string) error {
	var f *os.File
	var err error
	if filename != "-" {
		f, err = os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		f = os.Stdin
	}

	hSixel, hKitty := hasSixel(), hasKitty()

	if !hSixel && !hKitty {
		return dump(f)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		if err == image.ErrFormat {
			f.Close()
			f, err = os.Open(filename)
			if err != nil {
				return err
			}
			defer f.Close()
			return dump(f)
		}
		return err
	}

	h, w := img.Bounds().Dy(), img.Bounds().Dx()
	if h > maxHeight || w > maxWidth {
		rx := float64(w) / float64(maxWidth)
		ry := float64(h) / float64(maxHeight)
		if rx < ry {
			w = int(float64(w) / ry)
			h = maxHeight
		} else {
			h = int(float64(h) / rx)
			w = maxWidth
		}
		tmp := image.NewNRGBA64(image.Rect(0, 0, int(w), int(h)))
		err = graphics.Scale(tmp, img)
		if err != nil {
			return err
		}
		img = tmp
	}

	buf := bufio.NewWriter(os.Stdout)
	defer buf.Flush()

	if hasSixel() {
		enc := sixel.NewEncoder(buf)
		enc.Dither = true
		err = enc.Encode(img)
	} else if hasKitty() {
		kittyimg.Fprint(buf, img)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}

// FIXME better detection

func hasSixel() bool {
	termenv := os.Getenv("TERM")
	switch termenv {
	case "xterm-kitty":
		return false
	case "xterm-ghostty":
		return false
	}
	return true
}

func hasKitty() bool {
	termenv := os.Getenv("TERM")
	switch termenv {
	case "xterm-kitty":
		return true
	case "xterm-ghostty":
		return true
	}
	// if term.IsTerminal(int(os.Stdin.Fd())) {
	// 	return false
	// }

	return false
}

// FIXME: Support `cat` options like `-vET`, and scratches your arm up

func main() {
	if len(os.Args) < 2 {
		// Make it cat compatible
		_ = render("-")
	}
	for _, arg := range os.Args[1:] {
		err := render(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
