package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"unicode/utf8"
	"unsafe"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const padPx = 2 // number of pixels used to pad UI elements

var (
	lightBgColor      = sdl.Color{0xff, 0xff, 0xff, 0xff}
	lightFgColor      = sdl.Color{0x2f, 0x2f, 0x2f, 0xff}
	lightStatusColor  = sdl.Color{0xe8, 0xe8, 0xe8, 0xff}
	lightCommentColor = sdl.Color{0x3f, 0x5a, 0x8d, 0xff}
	lightKeywordColor = sdl.Color{0x3a, 0x63, 0x41, 0xff}
	lightLiteralColor = sdl.Color{0x8e, 0x4a, 0x43, 0xff}

	darkBgColor      = sdl.Color{0x25, 0x25, 0x25, 0xff}
	darkFgColor      = sdl.Color{0xe2, 0xe2, 0xe2, 0xff}
	darkStatusColor  = sdl.Color{0x40, 0x40, 0x40, 0xff}
	darkCommentColor = sdl.Color{0xa0, 0xb6, 0xdf, 0xff}
	darkKeywordColor = sdl.Color{0x99, 0xbe, 0x9f, 0xff}
	darkLiteralColor = sdl.Color{0xda, 0xaa, 0xa5, 0xff}

	bgColor, fgColor, statusColor            sdl.Color
	commentColor, keywordColor, literalColor sdl.Color
)

func setColorScheme() {
	if darkFlag {
		bgColor = darkBgColor
		fgColor = darkFgColor
		statusColor = darkStatusColor
		commentColor = darkCommentColor
		keywordColor = darkKeywordColor
		literalColor = darkLiteralColor
	} else {
		bgColor = lightBgColor
		fgColor = lightFgColor
		statusColor = lightStatusColor
		commentColor = lightCommentColor
		keywordColor = lightKeywordColor
		literalColor = lightLiteralColor
	}
}

var fontHeight, fontWidth int

// Pane is a buffer with associated metadata.
type Pane struct {
	*edit.Buffer
	Title      string
	TabWidth   int
	Cols, Rows int
	LineEnding string
}

// getFont loads the default TTF from memory and returns it.
func getFont() *ttf.Font {
	var font *ttf.Font
	var err error
	// if font flag is specified, try loading that font
	if fontFlag != "" {
		font, err = ttf.OpenFont(fontFlag, ptsizeFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if !font.FaceIsFixedWidth() {
			fmt.Fprintf(os.Stderr, "%s is not fixed-width\n", fontFlag)
			font = nil
		}
	}
	if font == nil {
		// fall back to loading built-in font
		data, err := Asset("data/font.ttf")
		if err != nil {
			log.Fatal(err)
		}
		rw := sdl.RWFromMem(unsafe.Pointer(&data[0]), len(data))
		font, err = ttf.OpenFontRW(rw, 1, ptsizeFlag)
		if err != nil {
			log.Fatal(err)
		}
	}

	// set globally accessible font dimensions
	if fontWidth, _, err = font.SizeUTF8("0"); err != nil {
		log.Fatal(err)
	}
	fontHeight = font.Height()

	return font
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
	width, height := fontWidth*80+padPx*2, fontHeight*25+padPx*6
	win, err := sdl.CreateWindow(title, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_RESIZABLE)
	if err != nil {
		log.Fatal(err)
	}
	win.SetIcon(getIcon())
	return win
}

// drawBuffer draws the displayed contents of b to dst using font.
func drawBuffer(b *edit.Buffer, font *ttf.Font, dst *sdl.Surface,
	focused bool) {
	x, y := padPx, padPx

	// get cursor position
	ins := b.IndexFromMark(insMark)
	col, row := b.CoordsFromIndex(ins)

	// get selection start and end positions
	sel := b.IndexFromMark(selMark)
	selStart, selEnd := order(sel, ins)
	startCol, startRow := b.CoordsFromIndex(selStart)
	endCol, endRow := b.CoordsFromIndex(selEnd)

	// draw each line in display
	for i, line := range b.DisplayLines() {
		c := 0

		// draw each syntax-highlighted fragment
		for e := line.Front(); e != nil; e = e.Next() {
			text := e.Value.(edit.Fragment).Text
			fg := fgColor
			switch e.Value.(edit.Fragment).Tag {
			case commentId:
				fg = commentColor
			case keywordId:
				fg = keywordColor
			case literalId:
				fg = literalColor
			}

			if sel != ins && i >= startRow && i <= endRow {
				// text might be in selection range, so it needs to be split
				// and drawn piecewise
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

				drawString(font, string(pre), fg, bgColor, dst, x, y)
				x += len(pre) * fontWidth
				drawString(font, string(mid), fg, statusColor, dst, x, y)
				x += len(mid) * fontWidth
				drawString(font, string(post), fg, bgColor, dst, x, y)
				x += len(post) * fontWidth

				c += len(runes)
			} else {
				drawString(font, text, fg, bgColor, dst, x, y)
				x += utf8.RuneCountInString(text) * fontWidth
			}
		}

		if focused && i == row {
			// draw cursor
			dst.FillRect(&sdl.Rect{int32(padPx + fontWidth*col), int32(y),
				1 + int32(ptsizeFlag)/18, int32(fontHeight)}, fgColor.Uint32())
		}

		y += fontHeight
		x = padPx
	}
}

// drawString draws s to dst at (x, y) using font.
func drawString(font *ttf.Font, s string, fg, bg sdl.Color, dst *sdl.Surface,
	x, y int) {
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
	}
}

// drawStatusLine draws s at the bottom of dst using font.
func drawStatusLine(dst *sdl.Surface, font *ttf.Font, s string,
	input *edit.Buffer, pane *Pane, focused bool) {
	// draw background
	bgRect := sdl.Rect{
		0,
		dst.H - int32(fontHeight) - padPx*2,
		dst.W,
		int32(fontHeight) + padPx*2,
	}
	dst.FillRect(&bgRect, statusColor.Uint32())

	// draw status text
	x, y := padPx, int(dst.H)-fontHeight-padPx
	drawString(font, s, fgColor, statusColor, dst, x, y)
	x += utf8.RuneCountInString(s) * fontWidth

	if focused {
		// draw input text and cursor
		drawString(font, input.Get(edit.Index{1, 0}, input.End()), fgColor,
			statusColor, dst, x, y)
		index := input.IndexFromMark(insMark)
		dst.FillRect(&sdl.Rect{int32(x + fontWidth*index.Char), int32(y),
			1 + int32(ptsizeFlag)/18, int32(fontHeight)}, fgColor.Uint32())
	} else if s == pane.Title {
		// draw cursor pos
		index := pane.IndexFromMark(insMark)
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
		drawString(font, cursorPos, fgColor, statusColor, dst,
			int(dst.W)-padPx-fontWidth*17, int(dst.H)-fontHeight-padPx)

		// draw scroll percent
		f := pane.ScrollFraction()
		scrollStr := fmt.Sprintf("%d%%", int(f*100))
		if f < 0 {
			scrollStr = "All"
		}
		drawString(font, scrollStr, fgColor, statusColor, dst,
			int(dst.W)-padPx-fontWidth*4, int(dst.H)-fontHeight-padPx)
	}
}

// paneSpace returns the number of vertical pixels available to a pane.
func paneSpace(height int) int {
	return height - fontHeight - padPx*2
}

// bufSize returns the number of rows and columns available to a pane.
func bufSize(width, height int) (cols, rows int) {
	cols = (width - padPx*2) / fontWidth
	rows = paneSpace(height) / fontHeight
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

// render redraws and updates the display.
func render(rc *RenderContext) {
	surf, err := rc.Window.GetSurface()
	if err != nil {
		log.Fatal(err)
	}
	surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor.Uint32())
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
			rc.Font = getFont()
			render(rc)
		}
	}()
	paneFocused := rc.Focus == rc.Pane.Buffer
	drawBuffer(rc.Pane.Buffer, rc.Font, surf, paneFocused)
	drawStatusLine(surf, rc.Font, rc.Status, rc.Input, rc.Pane, !paneFocused)
	rc.Window.UpdateSurface()
}
