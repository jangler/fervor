package main

import (
	"bytes"
	"log"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

// TextInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func TextInput(buf *edit.Buffer, s string) {
	buf.Insert(buf.End(), s)
}

// EventLoop handles SDL events until quit is requested.
func EventLoop(buf *edit.Buffer, font *ttf.Font, win *sdl.Window) {
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				buf.Delete(buf.ShiftIndex(buf.End(), -1), buf.End())
			case sdl.K_RETURN:
				TextInput(buf, "\n")
			case sdl.K_TAB:
				TextInput(buf, "\t")
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					return
				}
			}
			Render <- 1
		case *sdl.QuitEvent:
			return
		case *sdl.TextInputEvent:
			if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
				TextInput(buf, string(event.Text[:n]))
				Render <- 1
			}
		case *sdl.WindowEvent:
			switch event.Event {
			case sdl.WINDOWEVENT_EXPOSED:
				win.UpdateSurface()
			case sdl.WINDOWEVENT_RESIZED:
				fontWidth, _, err := font.SizeUTF8("0")
				if err != nil {
					log.Fatal(err)
				}
				buf.SetSize(int(event.Data1)/fontWidth,
					int(event.Data2)/font.Height())
				Render <- 1
			}
		}
	}
}
