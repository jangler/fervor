package main

import (
	"fmt"
	"log"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const padPx = 2 // number of pixels used to pad UI elements

var (
	// colors for use with FillRect
	bgColor       uint32 = 0xffffffff
	fgColor       uint32 = 0xff2f2f2f
	statusBgColor uint32 = 0xffe2e2e2

	// colors for use with ttf functions
	bgColorSDL       = sdl.Color{0xff, 0xff, 0xff, 0xff}
	fgColorSDL       = sdl.Color{0x2f, 0x2f, 0x2f, 0xff}
	statusBgColorSDL = sdl.Color{0xe2, 0xe2, 0xe2, 0xff}
	commentColor     = sdl.Color{0x42, 0x6c, 0xbb, 0xff}
	keywordColor     = sdl.Color{0x37, 0x79, 0x49, 0xff}
	literalColor     = sdl.Color{0xbe, 0x53, 0x4b, 0xff}
)

var fontWidth int

// Pane is a buffer with associated metadata.
type Pane struct {
	*edit.Buffer
	Title      string
	TabWidth   int
	Cols, Rows int
}

// See ensurees that the mark with ID id is visible on the pane's screen.
func (p Pane) See(id int) {
	index := p.IndexFromMark(id)
	_, row := p.CoordsFromIndex(index)
	if row < -p.Rows {
		p.Scroll(row - p.Rows/2)
	} else if row < 0 {
		p.Scroll(row)
	} else if row >= p.Rows*2 {
		p.Scroll(row + 1 - p.Rows/2)
	} else if row >= p.Rows {
		p.Scroll(row + 1 - p.Rows)
	}
}

// createWindow returns a new SDL window of appropriate size given font, and
// titled title.
func createWindow(title string, font *ttf.Font) *sdl.Window {
	width := fontWidth*80 + padPx*2
	height := font.Height()*27 + padPx*6
	win, err := sdl.CreateWindow(title, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_RESIZABLE)
	if err != nil {
		log.Fatal(err)
	}
	return win
}

// drawBuffer draws the displayed contents of pane to dst using font.
func drawBuffer(pane *Pane, font *ttf.Font, dst *sdl.Surface) {
	x, y := padPx, padPx
	mark := pane.IndexFromMark(insertMark)
	col, row := pane.CoordsFromIndex(mark)
	for i, line := range pane.DisplayLines() {
		for e := line.Front(); e != nil; e = e.Next() {
			text := e.Value.(edit.Fragment).Text
			var fg sdl.Color
			switch e.Value.(edit.Fragment).Tag {
			case commentId:
				fg = commentColor
			case keywordId:
				fg = keywordColor
			case literalId:
				fg = literalColor
			default:
				fg = fgColorSDL
			}
			x = drawString(font, text, fg, bgColorSDL, dst, x, y)
		}
		if i == row {
			dst.FillRect(&sdl.Rect{int32(padPx + fontWidth*col), int32(y),
				1, int32(font.Height())}, fgColor)
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
func drawStatusLine(dst *sdl.Surface, font *ttf.Font, s string, pane *Pane) {
	// draw background
	bgRect := sdl.Rect{
		0,
		dst.H - int32(font.Height()) - padPx*2,
		dst.W,
		int32(font.Height()) + padPx*2,
	}
	dst.FillRect(&bgRect, statusBgColor)

	// draw status text
	drawString(font, s, fgColorSDL, statusBgColorSDL, dst, padPx,
		int(dst.H)-font.Height()-padPx)

	// draw cursor pos
	index := pane.IndexFromMark(insertMark)
	line := pane.Get(edit.Index{index.Line, 0}, index)
	col := 0
	for _, ch := range line {
		if ch == '\t' {
			col += pane.TabWidth - col%pane.TabWidth
		} else {
			col++
		}
	}
	cursorPos := fmt.Sprintf("%d,%d", index.Line, index.Char)
	if col != index.Char {
		cursorPos += fmt.Sprintf("-%d", col)
	}
	drawString(font, cursorPos, fgColorSDL, statusBgColorSDL, dst,
		int(dst.W)-padPx-fontWidth*17, int(dst.H)-font.Height()-padPx)

	// draw scroll percent
	f := pane.ScrollFraction()
	scrollStr := fmt.Sprintf("%d%%", int(f*100))
	if f < 0 {
		scrollStr = "All"
	}
	drawString(font, scrollStr, fgColorSDL, statusBgColorSDL, dst,
		int(dst.W)-padPx-fontWidth*4, int(dst.H)-font.Height()-padPx)
}

// paneSpace returns the number of vertical pixels available to each pane,
// sized equally out of n panes.
func paneSpace(height, n int, font *ttf.Font) int {
	return (height - font.Height() - padPx*2) / n
}

// bufSize returns the number of rows and columns available to each pane,
// sized equally out of n panes.
func bufSize(width, height, n int, font *ttf.Font) (cols, rows int) {
	cols = (width - padPx*2) / fontWidth
	rows = paneSpace(height, n, font) / font.Height()
	return
}

// RenderContext contains information needed to update the display.
type RenderContext struct {
	Pane   *Pane
	Status string
	Font   *ttf.Font
	Window *sdl.Window
}

// render updates the display.
func render(rc *RenderContext) {
	surf, err := rc.Window.GetSurface()
	if err != nil {
		log.Fatal(err)
	}
	surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor)
	drawBuffer(rc.Pane, rc.Font, surf)
	drawStatusLine(surf, rc.Font, rc.Status, rc.Pane)
	rc.Window.UpdateSurface()
}
