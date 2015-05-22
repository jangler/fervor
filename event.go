package main

import (
	"bytes"
	"log"
	"regexp"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

var (
	spaceRegexp = regexp.MustCompile(`\s`) // matches whitespace characters
	wordRegexp  = regexp.MustCompile(`\w`) // matches word characters
)

// focusedPane returns the focused pane index out of a slice of panes.
func focusedPane(panes []Pane) int {
	for i, pane := range panes {
		if pane.Focused {
			return i
		}
	}
	log.Fatal("No focused pane")
	return 0 // unreachable
}

// change pane focus based on mouse position
func refocus(panes []Pane, win *sdl.Window, font *ttf.Font, x, y int) {
	_, height := win.GetSize()
	ps := paneSpace(height, len(panes), font)
	i := y / ps
	if i < len(panes) {
		for j := range panes {
			panes[j].Focused = false
		}
		panes[i].Focused = true
	}
}

// singleClick processes a single left mouse click at the given coordinates.
func singleClick(panes []Pane, win *sdl.Window, font *ttf.Font, x, y int) {
	_, height := win.GetSize()
	ps := paneSpace(height, len(panes), font)
	i := y / ps                        // clicked pane index
	y = y%ps - font.Height() - padPx*3 // subtract pane header
	x -= padPx - fontWidth/2
	y /= font.Height()
	x /= fontWidth
	if i < len(panes) {
		panes[i].Mark(panes[i].IndexFromCoords(x, y), insertMark)
	}
}

// textInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func textInput(buf *edit.Buffer, s string) {
	index, _ := buf.IndexFromMark(insertMark)
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

// resize resizes the panes in the display and requests a render.
func resize(panes []Pane, font *ttf.Font, width, height int) {
	cols, rows := bufSize(width, height, len(panes), font)
	for i := range panes {
		panes[i].Cols, panes[i].Rows = cols, rows
		panes[i].SetSize(cols, rows)
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
	pane := panes[focusedPane(panes)]
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
		case *sdl.MouseButtonEvent:
			if event.Type == sdl.MOUSEBUTTONDOWN &&
				event.Button == sdl.BUTTON_LEFT {
				singleClick(panes, win, font, int(event.X), int(event.Y))
				pane = panes[focusedPane(panes)]
				paneSet <- panes
			}
		case *sdl.MouseMotionEvent:
			refocus(panes, win, font, int(event.X), int(event.Y))
			pane = panes[focusedPane(panes)]
			paneSet <- panes
		case *sdl.MouseWheelEvent:
			pane.Scroll(int(event.Y) * -3)
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
