package main

import (
	"log"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

var (
	// Colors for use with FillRect.
	BgColor uint32 = 0xffe0e0e0

	// Colors for use with ttf functions.
	BgColorSDL = sdl.Color{0xe0, 0xe0, 0xe0, 0xff}
	FgColorSDL = sdl.Color{0x20, 0x20, 0x20, 0xff}
)

// Render is a channel used to communicate with RenderLoop.
var Render = make(chan int)

// DrawString draws s to dst at (x, y) using font, and returns x plus the width
// of the text in pixels.
func DrawString(font *ttf.Font, s string, fg, bg sdl.Color, dst *sdl.Surface,
	x, y int) int {
	if s != "" {
		surf, err := font.RenderUTF8_Shaded(s, fg, bg)
		if err != nil {
			log.Fatal(err, s)
		}
		defer surf.Free()
		err = surf.Blit(&sdl.Rect{0, 0, surf.W, surf.H}, dst,
			&sdl.Rect{int32(x), int32(y), 0, 0})
		if err != nil {
			log.Fatal(err)
		}
		x += int(surf.W)
	}
	return x
}

// RenderLoop listens on the Render channel and draws the screen each time a
// value is received.
func RenderLoop(buf *edit.Buffer, font *ttf.Font, win *sdl.Window) {
	for range Render {
		surf, err := win.GetSurface()
		if err != nil {
			log.Fatal(err)
		}
		surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, BgColor)
		x, y := 0, 0
		for _, line := range buf.DisplayLines() {
			for e := line.Front(); e != nil; e = e.Next() {
				text := e.Value.(edit.Fragment).Text
				x = DrawString(font, text, FgColorSDL, BgColorSDL, surf, x, y)
			}
			y += font.Height()
			x = 0
		}
		win.UpdateSurface()
	}
}
