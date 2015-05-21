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
	paneBgColor   uint32 = 0xffe0ebf5
	statusBgColor uint32 = 0xffe9e9e9

	// colors for use with ttf functions
	bgColorSDL       = sdl.Color{0xff, 0xff, 0xff, 0xff}
	fgColorSDL       = sdl.Color{0x22, 0x22, 0x22, 0xff}
	paneFgColorSDL   = sdl.Color{0x37, 0x5e, 0xab, 0xff}
	paneBgColorSDL   = sdl.Color{0xe0, 0xeb, 0xf5, 0xff}
	statusBgColorSDL = sdl.Color{0xe9, 0xe9, 0xe9, 0xff}
)

var (
	render = make(chan int)    // used to signal a redraw of the screen
	status = make(chan string) // used to update status message

	paneSet = make(chan []Pane) // used to update pane list
)

// Pane is a buffer with associated metadata.
type Pane struct {
	*edit.Buffer
	Title string
}

// createWindow returns a new SDL window of appropriate size given font, and
// titled title.
func createWindow(title string, font *ttf.Font) *sdl.Window {
	fontWidth, _, err := font.SizeUTF8("0")
	if err != nil {
		log.Fatal(err)
	}
	width := fontWidth*80 + padPx*2
	height := font.Height()*27 + padPx*8
	win, err := sdl.CreateWindow(title, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_RESIZABLE)
	if err != nil {
		log.Fatal(err)
	}
	return win
}

// drawPaneHeader draws a pane header displaying the string s.
func drawPaneHeader(dst *sdl.Surface, font *ttf.Font, s string, y int) {
	bgRect := sdl.Rect{
		0,
		int32(y),
		dst.W,
		int32(font.Height()) + padPx*4,
	}
	dst.FillRect(&bgRect, bgColor)
	bgRect = sdl.Rect{
		padPx,
		int32(y) + padPx,
		dst.W - padPx*2,
		int32(font.Height()) + padPx*2,
	}
	dst.FillRect(&bgRect, paneBgColor)
	drawString(font, s, paneFgColorSDL, paneBgColorSDL, dst, padPx*2,
		y+padPx*2)
}

// drawBuffer draws the displayed contents of buf to dst using font.
func drawBuffer(buf *edit.Buffer, font *ttf.Font, dst *sdl.Surface, y int) {
	x := padPx
	for _, line := range buf.DisplayLines() {
		for e := line.Front(); e != nil; e = e.Next() {
			text := e.Value.(edit.Fragment).Text
			x = drawString(font, text, fgColorSDL, bgColorSDL, dst, x, y)
		}
		y += font.Height()
		x = padPx
	}
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
		0,
		dst.H - int32(font.Height()) - padPx*4,
		dst.W,
		int32(font.Height()) + padPx*4,
	}
	dst.FillRect(&bgRect, bgColor)
	bgRect = sdl.Rect{
		padPx,
		dst.H - int32(font.Height()) - padPx*3,
		dst.W - padPx*2,
		int32(font.Height()) + padPx*2,
	}
	dst.FillRect(&bgRect, statusBgColor)
	drawString(font, s, fgColorSDL, statusBgColorSDL, dst, padPx*2,
		int(dst.H)-font.Height()-padPx*2)
}

// paneSpace returns the number of vertical pixels available to each pane,
// sized equally out of n panes.
func paneSpace(height, n int, font *ttf.Font) int {
	return (height - font.Height() - padPx*4) / n
}

// bufSize returns the number of rows and columns available to each pane,
// sized equally out of n panes.
func bufSize(width, height, n int, font *ttf.Font) (cols, rows int) {
	fontWidth, _, err := font.SizeUTF8("0")
	if err != nil {
		log.Fatal(err)
	}
	cols = (width - padPx*2) / fontWidth
	rows = paneSpace(height, n, font) / font.Height()
	return
}

// renderLoop listens on the render channel and draws the screen each time a
// value is received.
func renderLoop(font *ttf.Font, win *sdl.Window) {
	var statusText string
	panes := make([]Pane, 0)
	for {
		select {
		case panes = <-paneSet:
			go func() { render <- 1 }()
		case <-render:
			surf, err := win.GetSurface()
			if err != nil {
				log.Fatal(err)
			}
			surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor)
			ps := paneSpace(int(surf.H), len(panes), font)
			for i, pane := range panes {
				drawPaneHeader(surf, font, pane.Title, ps*i)
				drawBuffer(pane.Buffer, font, surf,
					ps*i+font.Height()+padPx*4)
			}
			drawStatusLine(surf, font, statusText)
			win.UpdateSurface()
		case s := <-status:
			if s != statusText {
				statusText = s
				go func() { render <- 1 }()
			}
		}
	}
}
