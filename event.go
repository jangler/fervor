package main

import (
	"bytes"
	"log"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

func textInput(buf *edit.Buffer, s string) {
	buf.Insert(buf.End(), s)
}

func eventLoop(buf *edit.Buffer, font *ttf.Font, win *sdl.Window) {
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				buf.Delete(buf.ShiftIndex(buf.End(), -1), buf.End())
			case sdl.K_RETURN:
				textInput(buf, "\n")
			case sdl.K_TAB:
				textInput(buf, "\t")
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					return
				}
			}
			render <- 1
		case *sdl.QuitEvent:
			quit <- 1
			return
		case *sdl.TextInputEvent:
			if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
				textInput(buf, string(event.Text[:n]))
				render <- 1
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
				render <- 1
			}
		}
	}
}
