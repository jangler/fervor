package main

import "github.com/veandco/go-sdl2/sdl"

func eventLoop(win *sdl.Window) {
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			switch event.Keysym.Sym {
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					return
				}
			}
		case *sdl.QuitEvent:
			return
		case *sdl.WindowEvent:
			switch event.Event {
			case sdl.WINDOWEVENT_EXPOSED:
				win.UpdateSurface()
			case sdl.WINDOWEVENT_RESIZED:
				render <- 1
			}
		}
	}
}
