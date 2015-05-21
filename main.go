package main

import (
	"log"
	"os"
	"unsafe"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

var quit = make(chan int)

func getFont() *ttf.Font {
	data, err := Asset("data/DejaVuSansMono.ttf")
	if err != nil {
		log.Fatal(err)
	}
	rw := sdl.RWFromMem(unsafe.Pointer(&data[0]), len(data))
	font, err := ttf.OpenFontRW(rw, 1, 12)
	if err != nil {
		log.Fatal(err)
	}
	font.SetHinting(ttf.HINTING_MONO)
	return font
}

func main() {
	log.SetFlags(log.Lshortfile)
	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()
	ttf.Init()

	defer ttf.Quit()
	font := getFont()
	fontWidth, _, err := font.SizeUTF8("0")
	if err != nil {
		log.Fatal(err)
	}

	buf := edit.NewBuffer()

	win, err := sdl.CreateWindow(os.Args[0], sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, fontWidth*80, font.Height()*25,
		sdl.WINDOW_RESIZABLE)
	if err != nil {
		log.Fatal(err)
	}
	defer win.Destroy()

	go eventLoop(buf, font, win)
	go renderLoop(buf, font, win)
	render <- 1
	<-quit
}
