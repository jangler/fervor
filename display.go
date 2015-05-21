package main

import (
	"log"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const padPx = 2 // number of pixels used to pad UI elements

var (
	// colors for use with FillRect
	bgColor       uint32 = 0xffffffff
	statusBgColor uint32 = 0xffe9e9e9

	// colors for use with ttf functions
	bgColorSDL       = sdl.Color{0xff, 0xff, 0xff, 0xff}
	fgColorSDL       = sdl.Color{0x22, 0x22, 0x22, 0xff}
	statusBgColorSDL = sdl.Color{0xe9, 0xe9, 0xe9, 0xff}
)

var (
	render = make(chan int)    // used to signal a redraw of the screen
	status = make(chan string) // used to update status message
)

// createWindow returns a new SDL window of appropriate size given font, and
// titled title.
func createWindow(title string, font *ttf.Font) *sdl.Window {
	fontWidth, _, err := font.SizeUTF8("0")
	if err != nil {
		log.Fatal(err)
	}
	width := fontWidth*80 + padPx*2
	height := fontWidth*26 + padPx*4
	win, err := sdl.CreateWindow(title, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_RESIZABLE)
	if err != nil {
		log.Fatal(err)
	}
	return win
}

// drawString draws s to dst at (x, y) using font, and returns x plus the width
// of the text in pixels.
func drawString(font *ttf.Font, s string, fg, bg sdl.Color, dst *sdl.Surface,
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

// drawStatusLine draws s at the bottom of dst using font.
func drawStatusLine(dst *sdl.Surface, font *ttf.Font, s string) {
	bgRect := sdl.Rect{
		padPx,
		dst.H - int32(font.Height()) - padPx*3,
		dst.W - padPx*2,
		int32(font.Height()) + padPx*2,
	}
	dst.FillRect(&bgRect, statusBgColor)
	drawString(font, s, fgColorSDL, statusBgColorSDL, dst, padPx*2,
		int(dst.H)-font.Height()-padPx*2)
}

// renderLoop listens on the render channel and draws the screen each time a
// value is received.
func renderLoop(buf *edit.Buffer, font *ttf.Font, win *sdl.Window) {
	var statusText string
	for {
		select {
		case s := <-status:
			if s != statusText {
				statusText = s
				go func() { render <- 1 }()
			}
		case <-render:
			surf, err := win.GetSurface()
			if err != nil {
				log.Fatal(err)
			}
			surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor)
			x, y := padPx, padPx
			for _, line := range buf.DisplayLines() {
				for e := line.Front(); e != nil; e = e.Next() {
					text := e.Value.(edit.Fragment).Text
					x = drawString(font, text, fgColorSDL, bgColorSDL, surf,
						x, y)
				}
				y += font.Height()
				x = padPx
			}
			drawStatusLine(surf, font, statusText)
			win.UpdateSurface()
		}
	}
}
