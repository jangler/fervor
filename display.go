package main

import (
	"log"

	"github.com/jangler/edit"
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
	x, y int) int {
	if s != "" {
		surf, err := font.RenderUTF8_Shaded(s, fg, bg)
		if err != nil {
			log.Fatal(err, s)
		}
		defer surf.Free()
		err = surf.Blit(&sdl.Rect{0, 0, surf.W, surf.H}, dst,
			&sdl.Rect{int32(x), int32(y), 0, 0})
		if err != nil {
			log.Fatal(err)
		}
		x += int(surf.W)
	}
	return x
}

func renderLoop(buf *edit.Buffer, font *ttf.Font, win *sdl.Window) {
	for {
		<-render
		surf, err := win.GetSurface()
		if err != nil {
			log.Fatal(err)
		}
		surf.FillRect(&sdl.Rect{0, 0, surf.W, surf.H}, bgColor)
		x, y := 0, 0
		for _, line := range buf.DisplayLines() {
			for e := line.Front(); e != nil; e = e.Next() {
				text := e.Value.(edit.Fragment).Text
				x = drawString(font, text, sdlFgColor, sdlBgColor, surf, x, y)
			}
			y += font.Height()
			x = 0
		}
		win.UpdateSurface()
	}
}
