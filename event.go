package main

import (
	"bytes"
	"regexp"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

var (
	spaceRegexp = regexp.MustCompile(`\s`) // matches whitespace characters
	wordRegexp  = regexp.MustCompile(`\w`) // matches word characters
)

// singleClick processes a single left mouse click at the given coordinates.
func singleClick(pane *Pane, win *sdl.Window, font *ttf.Font, x, y int) {
	_, height := win.GetSize()
	ps := paneSpace(height, 1, font)
	y = y%ps
	x -= padPx - fontWidth/2
	y /= font.Height()
	x /= fontWidth
	pane.Mark(pane.IndexFromCoords(x, y), insertMark)
}

// textInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func textInput(buf *edit.Buffer, s string) {
	index := buf.IndexFromMark(insertMark)
	if s == "\n" {
		// autoindent
		indent := buf.Get(edit.Index{index.Line, 0},
			edit.Index{index.Line, 0xffff})
		i := 0
		for i < len(indent) && (indent[i] == ' ' || indent[i] == '\t') {
			i++
		}
		if i <= index.Char {
			s += indent[:i]

			// delete lines containing only whitespace
			if i == len(indent) {
				buf.Delete(edit.Index{index.Line, 0}, index)
				index.Char = 0
			}
		}
	}
	buf.Insert(index, s)
}

// resize resizes the pane in the display
func resize(pane *Pane, font *ttf.Font, width, height int) {
	cols, rows := bufSize(width, height, 1, font)
	pane.Cols, pane.Rows = cols, rows
	pane.SetSize(cols, rows)
}

// ShiftIndexByWord returns the given index shifted forward by n words. A
// negative value for n will shift backwards.
func (p *Pane) ShiftIndexByWord(index edit.Index, n int) edit.Index {
	for n > 0 {
		// TODO
		n--
	}
	for n < 0 {
		if index.Char == 0 {
			index = p.ShiftIndex(index, -1)
		} else {
			text := []rune(p.Get(edit.Index{index.Line, 0}, index))
			i := len(text) - 1
			for i >= 0 && spaceRegexp.MatchString(string(text[i])) {
				i--
			}
			if i >= 0 && wordRegexp.MatchString(string(text[i])) {
				for i >= 0 && wordRegexp.MatchString(string(text[i])) {
					i--
				}
			} else {
				for i >= 0 && !wordRegexp.MatchString(string(text[i])) &&
					!spaceRegexp.MatchString(string(text[i])) {
					i--
				}
			}
			index.Char = i + 1
		}
		n++
	}
	return index
}

// eventLoop handles SDL events until quit is requested.
func eventLoop(pane *Pane, status string, font *ttf.Font, win *sdl.Window) {
	rc := &RenderContext{pane, status, font, win}
	render(rc)
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				index := pane.IndexFromMark(insertMark)
				pane.Delete(pane.ShiftIndex(index, -1), index)
			case sdl.K_DELETE:
				index := pane.IndexFromMark(insertMark)
				pane.Delete(index, pane.ShiftIndex(index, 1))
			case sdl.K_DOWN:
				index := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row+1), insertMark)
			case sdl.K_END:
				index := pane.IndexFromMark(insertMark)
				pane.Mark(edit.Index{index.Line, 0xffff}, insertMark)
			case sdl.K_LEFT:
				index := pane.IndexFromMark(insertMark)
				pane.Mark(pane.ShiftIndex(index, -1), insertMark)
			case sdl.K_HOME:
				index := pane.IndexFromMark(insertMark)
				pane.Mark(edit.Index{index.Line, 0}, insertMark)
			case sdl.K_PAGEDOWN:
				index := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row+pane.Rows),
					insertMark)
			case sdl.K_PAGEUP:
				index := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row-pane.Rows), insertMark)
			case sdl.K_RETURN:
				textInput(pane.Buffer, "\n")
			case sdl.K_RIGHT:
				index := pane.IndexFromMark(insertMark)
				pane.Mark(pane.ShiftIndex(index, 1), insertMark)
			case sdl.K_TAB:
				textInput(pane.Buffer, "\t")
			case sdl.K_UP:
				index := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row-1), insertMark)
			case sdl.K_a:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := pane.IndexFromMark(insertMark)
					pane.Mark(edit.Index{index.Line, 0}, insertMark)
				}
			case sdl.K_e:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := pane.IndexFromMark(insertMark)
					pane.Mark(edit.Index{index.Line, 0xffff}, insertMark)
				}
			case sdl.K_h:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := pane.IndexFromMark(insertMark)
					pane.Delete(pane.ShiftIndex(index, -1), index)
				}
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					return
				}
			case sdl.K_u:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := pane.IndexFromMark(insertMark)
					pane.Delete(edit.Index{index.Line, 0}, index)
				}
			case sdl.K_w:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end := pane.IndexFromMark(insertMark)
					begin := pane.ShiftIndexByWord(end, -1)
					pane.Delete(begin, end)
				}
			}
			pane.See(insertMark)
			render(rc)
		case *sdl.MouseButtonEvent:
			if event.Type == sdl.MOUSEBUTTONDOWN &&
				event.Button == sdl.BUTTON_LEFT {
				singleClick(pane, win, font, int(event.X), int(event.Y))
				render(rc)
			}
		case *sdl.MouseWheelEvent:
			pane.Scroll(int(event.Y) * -3)
			render(rc)
		case *sdl.QuitEvent:
			return
		case *sdl.TextInputEvent:
			if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
				textInput(pane.Buffer, string(event.Text[:n]))
				render(rc)
			}
		case *sdl.WindowEvent:
			switch event.Event {
			case sdl.WINDOWEVENT_EXPOSED:
				win.UpdateSurface()
			case sdl.WINDOWEVENT_RESIZED:
				resize(pane, font, int(event.Data1), int(event.Data2))
				render(rc)
			}
		}
	}
}
