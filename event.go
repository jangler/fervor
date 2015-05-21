package main

import (
	"bytes"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
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
		pane.SetSize(cols, rows)
	}
	render <- 1
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
			case sdl.K_DOWN:
				index, _ := pane.IndexFromMark(insertMark)
				col, row := pane.CoordsFromIndex(index)
				pane.Mark(pane.IndexFromCoords(col, row+1), insertMark)
			case sdl.K_LEFT:
				index, _ := pane.IndexFromMark(insertMark)
				pane.Mark(pane.ShiftIndex(index, -1), insertMark)
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
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					return
				}
			}
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
