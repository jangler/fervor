package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"unsafe"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

// getFont loads the default TTF from memory and returns it.
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

// initFlag processes command-line flags and arguments.
func initFlag() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<file>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}
}

// openFile attempts to open the file given by path and return a new buffer
// containing the contents of that file. If an error is encountered, it returns
// a nil buffer and the error instead.
func openFile(path string) (*edit.Buffer, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	buf := edit.NewBuffer()
	buf.Insert(buf.End(), string(contents))
	return buf, nil
}

func main() {
	initFlag()
	log.SetFlags(log.Lshortfile)

	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()
	ttf.Init()
	defer ttf.Quit()

	font := getFont()

	win := createWindow(os.Args[0], font)
	defer win.Destroy()

	var buf *edit.Buffer
	if flag.NArg() == 1 {
		var err error
		if buf, err = openFile(flag.Arg(0)); err == nil {
			go func() { status <- fmt.Sprintf(`Opened "%s".`, flag.Arg(0)) }()
		} else {
			go func() { status <- err.Error() }()
		}
	} else {
		go func() { status <- "New file." }()
	}
	if buf == nil {
		buf = edit.NewBuffer()
	}

	go renderLoop(buf, font, win)
	eventLoop(buf, font, win)
}
