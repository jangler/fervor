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

// textInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func textInput(buf *edit.Buffer, s string) {
	index, _ := buf.IndexFromMark(insertMark)
	buf.Insert(index, s)
}

// resize resizes the panes in the display and requests a render.
func resize(panes []Pane, font *ttf.Font, width, height int) {
	cols, rows := bufSize(width, height, len(panes), font)
	for _, pane := range panes {
		pane.Cols, pane.Rows = cols, rows
		pane.SetSize(cols, rows)
	}
	render <- 1
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
func eventLoop(panes []Pane, font *ttf.Font, win *sdl.Window) {
	pane := panes[0]
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Delete(pane.ShiftIndex(index, -1), index)
			case sdl.K_DELETE:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Delete(index, pane.ShiftIndex(index, 1))
			case sdl.K_DOWN:
				index, _ := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row+1), insertMark)
			case sdl.K_END:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Mark(edit.Index{index.Line, 0xffff}, insertMark)
			case sdl.K_LEFT:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Mark(pane.ShiftIndex(index, -1), insertMark)
			case sdl.K_HOME:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Mark(edit.Index{index.Line, 0}, insertMark)
			case sdl.K_PAGEDOWN:
				index, _ := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row+pane.Rows), insertMark)
			case sdl.K_PAGEUP:
				index, _ := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row-pane.Rows), insertMark)
			case sdl.K_RETURN:
				textInput(pane.Buffer, "\n")
			case sdl.K_RIGHT:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Mark(pane.ShiftIndex(index, 1), insertMark)
			case sdl.K_TAB:
				textInput(pane.Buffer, "\t")
			case sdl.K_UP:
				index, _ := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row-1), insertMark)
			case sdl.K_a:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index, _ := pane.IndexFromMark(insertMark)
					pane.Mark(edit.Index{index.Line, 0}, insertMark)
				}
			case sdl.K_e:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index, _ := pane.IndexFromMark(insertMark)
					pane.Mark(edit.Index{index.Line, 0xffff}, insertMark)
				}
			case sdl.K_h:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index, _ := pane.IndexFromMark(insertMark)
					pane.Delete(pane.ShiftIndex(index, -1), index)
				}
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					return
				}
			case sdl.K_u:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index, _ := pane.IndexFromMark(insertMark)
					pane.Delete(edit.Index{index.Line, 0}, index)
				}
			case sdl.K_w:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end, _ := pane.IndexFromMark(insertMark)
					begin := pane.ShiftIndexByWord(end, -1)
					pane.Delete(begin, end)
				}
			}
			pane.See(insertMark)
			render <- 1
		case *sdl.QuitEvent:
			return
		case *sdl.TextInputEvent:
			if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
				textInput(pane.Buffer, string(event.Text[:n]))
				render <- 1
			}
		case *sdl.WindowEvent:
			switch event.Event {
			case sdl.WINDOWEVENT_EXPOSED:
				win.UpdateSurface()
			case sdl.WINDOWEVENT_RESIZED:
				resize(panes, font, int(event.Data1), int(event.Data2))
			}
		}
	}
}
