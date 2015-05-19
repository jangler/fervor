package main

import (
	"log"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

var (
	bgColor uint32 = 0xffe0e0e0

	sdlBgColor = sdl.Color{0xe0, 0xe0, 0xe0, 0xff}
	sdlFgColor = sdl.Color{0x20, 0x20, 0x20, 0xff}
)

var render = make(chan int)

func drawString(font *ttf.Font, s string, fg, bg sdl.Color, dst *sdl.Surface,
	x, y int) {
	if s != "" {
		surf, err := font.RenderUTF8_Shaded(s, fg, bg)
		if err != nil {
			log.Fatal(err)
		}
		defer surf.Free()
		err = surf.Blit(&sdl.Rect{0, 0, surf.W, surf.H}, dst,
			&sdl.Rect{int32(x), int32(y), 0, 0})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func renderLoop(win *sdl.Window, font *ttf.Font) {
	for {
		<-render
		surf, err := win.GetSurface()
		if err != nil {
			log.Fatal(err)
		}
		surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor)
		drawString(font, "Hello, world!", sdlFgColor, sdlBgColor, surf, 0, 0)
		win.UpdateSurface()
	}
}
