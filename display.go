package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"

	"code.google.com/p/jamslam-freetype-go/freetype/truetype"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/jangler/edit"
)

const fontSize = 12
const padPx = 2 // number of pixels used to pad UI elements

var (
	bgColor       = color.RGBA{0xff, 0xff, 0xff, 0xff}
	fgColor       = color.RGBA{0x2f, 0x2f, 0x2f, 0xff}
	statusBgColor = color.RGBA{0xe2, 0xe2, 0xe2, 0xff}
	commentColor  = color.RGBA{0x3f, 0x5a, 0x8d, 0xff}
	keywordColor  = color.RGBA{0x3a, 0x63, 0x41, 0xff}
	literalColor  = color.RGBA{0x8e, 0x4a, 0x43, 0xff}
)

var fontHeight, fontWidth int

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

// createWindow returns a new window of appropriate size given font.
func createWindow(xu *xgbutil.XUtil, font *truetype.Font) *xwindow.Window {
	width := fontWidth*80 + padPx*2
	height := fontHeight*27 + padPx*6
	win, err := xwindow.Generate(xu)
	if err != nil {
		log.Fatal(err)
	}
	win.Create(xu.RootWin(), 0, 0, width, height, 0, 0)
	win.Map()
	return win
}

// drawBuffer draws the displayed contents of pane to img using font.
func drawBuffer(pane *Pane, font *truetype.Font, img *xgraphics.Image,
	focused bool) {
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
			var fg color.Color
			switch e.Value.(edit.Fragment).Tag {
			case commentId:
				fg = commentColor
			case keywordId:
				fg = keywordColor
			case literalId:
				fg = literalColor
			default:
				fg = fgColor
			}
			if i >= startRow && i <= endRow {
				var pre, mid, post string
				if i == startRow && c < startCol {
					if startCol-c < len(text) {
						pre = text[:startCol-c]
					} else {
						pre = text
					}
				}
				if i == endRow && c+len(text) > endCol {
					if c < endCol {
						post = text[endCol-c:]
					} else {
						post = text
					}
				}
				mid = text[len(pre) : len(text)-len(post)]
				x = drawString(font, pre, fg, bgColor, img, x, y)
				x = drawString(font, mid, fg, statusBgColor, img, x, y)
				x = drawString(font, post, fg, bgColor, img, x, y)
				c += len(text)
			} else {
				x = drawString(font, text, fg, bgColor, img, x, y)
			}
		}
		if focused && i == row {
			rect := image.Rect(padPx+fontWidth*col, y,
				padPx+fontWidth*col+1, y+fontHeight)
			draw.Draw(img, rect, &image.Uniform{fgColor}, image.ZP, draw.Src)
		}
		y += fontHeight
		x = padPx
	}
}

// drawString draws s to img at (x, y) using font, and returns x plus the width
// of the text in pixels.
func drawString(font *truetype.Font, s string, fg, bg color.Color,
	img *xgraphics.Image, x, y int) int {
	x, y, err := img.Text(x, y, fg, fontSize, font, s)
	if err != nil {
		log.Fatal(err)
		return x
	}
	return x
}

// drawStatusLine draws s at the bottom of dst using font.
func drawStatusLine(img *xgraphics.Image, font *truetype.Font, s string,
	input *edit.Buffer, pane *Pane, focused bool) {
	// draw background
	imgSize := img.Bounds().Size()
	rect := image.Rect(0, imgSize.Y-fontHeight-padPx*2, imgSize.X,
		imgSize.Y)
	draw.Draw(img, rect, &image.Uniform{statusBgColor}, image.ZP, draw.Src)

	// draw status text
	x, y := padPx, imgSize.Y-fontHeight-padPx
	x = drawString(font, s, fgColor, statusBgColor, img, x, y)

	if focused {
		// draw input text and cursor
		drawString(font, input.Get(edit.Index{1, 0}, input.End()), fgColor,
			statusBgColor, img, x, y)
		index := input.IndexFromMark(insertMark)
		rect = image.Rect(x+fontWidth*index.Char, y,
			x+fontWidth*index.Char+1, y+fontHeight)
		draw.Draw(img, rect, &image.Uniform{fgColor}, image.ZP, draw.Src)
	} else {
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
		drawString(font, cursorPos, fgColor, statusBgColor, img,
			imgSize.X-padPx-fontWidth*17, imgSize.Y-fontHeight-padPx)

		// draw scroll percent
		f := pane.ScrollFraction()
		scrollStr := fmt.Sprintf("%d%%", int(f*100))
		if f < 0 {
			scrollStr = "All"
		}
		drawString(font, scrollStr, fgColor, statusBgColor, img,
			imgSize.X-padPx-fontWidth*4, imgSize.Y-fontHeight-padPx)
	}
}

// paneSpace returns the number of vertical pixels available to each pane,
// sized equally out of n panes.
func paneSpace(height, n int, font *truetype.Font) int {
	return (height - fontHeight - padPx*2) / n
}

// bufSize returns the number of rows and columns available to each pane,
// sized equally out of n panes.
func bufSize(width, height, n int, font *truetype.Font) (cols, rows int) {
	cols = (width - padPx*2) / fontWidth
	rows = paneSpace(height, n, font) / fontHeight
	return
}

// RenderContext contains information needed to update the display.
type RenderContext struct {
	Pane   *Pane
	Input  *edit.Buffer
	Focus  *edit.Buffer
	Status string
	Font   *truetype.Font
	Window *xwindow.Window
}

var savedImg *xgraphics.Image

// getImg gets an image to be used for drawing the window contents. A
// previously allocated image is reused if possible.
func getImg(rc *RenderContext) *xgraphics.Image {
	if savedImg == nil || savedImg.Bounds().Dx() != rc.Window.Geom.Width() ||
		savedImg.Bounds().Dy() != rc.Window.Geom.Height() {
		savedImg = xgraphics.New(rc.Window.X, image.Rect(0, 0,
			rc.Window.Geom.Width(), rc.Window.Geom.Height()))
		if err := savedImg.XSurfaceSet(rc.Window.Id); err != nil {
			log.Fatal(err)
		}
	}
	return savedImg
}

// expose repaints the window without redrawing its contents.
func expose(rc *RenderContext) {
	img := getImg(rc)
	img.XPaint(rc.Window.Id)
}

// render updates the display.
func render(rc *RenderContext) {
	img := getImg(rc)
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)
	paneFocused := rc.Focus == rc.Pane.Buffer
	drawBuffer(rc.Pane, rc.Font, img, paneFocused)
	drawStatusLine(img, rc.Font, rc.Status, rc.Input, rc.Pane, !paneFocused)
	img.XDraw()
	img.XPaint(rc.Window.Id)
}
