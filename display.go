package main

import (
	"fmt"
	"log"
	"regexp"
	"unicode/utf8"
	"unsafe"

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
	commentColor     = sdl.Color{0x3f, 0x5a, 0x8d, 0xff}
	keywordColor     = sdl.Color{0x3a, 0x63, 0x41, 0xff}
	literalColor     = sdl.Color{0x8e, 0x4a, 0x43, 0xff}
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

// getIcon loads the window icon from memory and returns it as a surface.
func getIcon() *sdl.Surface {
	data, err := Asset("data/icon.bmp")
	if err != nil {
		log.Fatal(err)
	}
	rw := sdl.RWFromMem(unsafe.Pointer(&data[0]), len(data))
	surf, err := sdl.LoadBMP_RW(rw, 1)
	if err != nil {
		log.Fatal(err)
	}
	return surf
}

// createWindow returns a new SDL window of appropriate size given font, and
// titled title.
func createWindow(title string, font *ttf.Font) *sdl.Window {
	width := fontWidth*80 + padPx*2
	height := font.Height()*27 + padPx*6
	win, err := sdl.CreateWindow(title, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_RESIZABLE)
	win.SetIcon(getIcon())
	if err != nil {
		log.Fatal(err)
	}
	return win
}

// drawBuffer draws the displayed contents of pane to dst using font.
func drawBuffer(pane *Pane, font *ttf.Font, dst *sdl.Surface, focused bool) {
	x, y := padPx, padPx
	mark := pane.IndexFromMark(insertMark)
	col, row := pane.CoordsFromIndex(mark)
	sel := pane.IndexFromMark(selMark)
	selStart, selEnd := sel, mark
	if mark.Less(sel) {
		selStart, selEnd = mark, sel
	}
	startCol, startRow := pane.CoordsFromIndex(selStart)
	endCol, endRow := pane.CoordsFromIndex(selEnd)
	for i, line := range pane.DisplayLines() {
		c := 0
		for e := line.Front(); e != nil; e = e.Next() {
			text := e.Value.(edit.Fragment).Text
			fg := fgColorSDL
			switch e.Value.(edit.Fragment).Tag {
			case commentId:
				fg = commentColor
			case keywordId:
				fg = keywordColor
			case literalId:
				fg = literalColor
			}
			if i >= startRow && i <= endRow {
				pre, mid, post := []rune(""), []rune(""), []rune("")
				runes := []rune(text)
				if i == startRow && c < startCol {
					if startCol-c < len(runes) {
						pre = runes[:startCol-c]
					} else {
						pre = runes
					}
				}
				if i == endRow && c+len(runes) > endCol {
					if c < endCol {
						post = runes[endCol-c:]
					} else {
						post = runes
					}
				}
				mid = runes[len(pre) : len(runes)-len(post)]
				x = drawString(font, string(pre), fg, bgColorSDL, dst, x, y)
				x = drawString(font, string(mid), fg, statusBgColorSDL, dst,
					x, y)
				x = drawString(font, string(post), fg, bgColorSDL, dst, x, y)
				c += len(runes)
			} else {
				x = drawString(font, text, fg, bgColorSDL, dst, x, y)
			}
		}
		if focused && i == row {
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
			panic(err)
		}
		defer surf.Free()
		err = surf.Blit(&sdl.Rect{0, 0, surf.W, surf.H}, dst,
			&sdl.Rect{int32(x), int32(y), 0, 0})
		if err != nil {
			log.Fatal(err)
		}

		// check surface size to make sure we're not missing glyphs.
		// this will probably be an issue with zero-width runes--let's hope we
		// don't encounter any of those.
		delta := fontWidth*utf8.RuneCountInString(s) - int(surf.W)
		if delta > fontWidth/2 {
			panic(fmt.Errorf("Rendered surface has incorrect size"))
		}

		x += int(surf.W)
	}
	return x
}

// drawStatusLine draws s at the bottom of dst using font.
func drawStatusLine(dst *sdl.Surface, font *ttf.Font, s string,
	input *edit.Buffer, pane *Pane, focused bool) {
	// draw background
	bgRect := sdl.Rect{
		0,
		dst.H - int32(font.Height()) - padPx*2,
		dst.W,
		int32(font.Height()) + padPx*2,
	}
	dst.FillRect(&bgRect, statusBgColor)

	// draw status text
	x, y := padPx, int(dst.H)-font.Height()-padPx
	x = drawString(font, s, fgColorSDL, statusBgColorSDL, dst, x, y)

	if focused {
		// draw input text and cursor
		drawString(font, input.Get(edit.Index{1, 0}, input.End()), fgColorSDL,
			statusBgColorSDL, dst, x, y)
		index := input.IndexFromMark(insertMark)
		dst.FillRect(&sdl.Rect{int32(x + fontWidth*index.Char), int32(y),
			1, int32(font.Height())}, fgColor)
	} else if s == pane.Title {
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
	Input  *edit.Buffer
	Focus  *edit.Buffer
	Status string
	Font   *ttf.Font
	Window *sdl.Window
	Regexp *regexp.Regexp
}

// render updates the display.
func render(rc *RenderContext) {
	surf, err := rc.Window.GetSurface()
	if err != nil {
		log.Fatal(err)
	}
	surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor)
	paneFocused := rc.Focus == rc.Pane.Buffer
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
			rc.Font = getFont()
			render(rc)
		}
	}()
	drawBuffer(rc.Pane, rc.Font, surf, paneFocused)
	drawStatusLine(surf, rc.Font, rc.Status, rc.Input, rc.Pane, !paneFocused)
	rc.Window.UpdateSurface()
}
