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
		fmt.Fprintf(os.Stderr, "Usage: %s [<file> ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
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

	panes := make([]Pane, 0)
	if flag.NArg() > 0 {
		for _, arg := range flag.Args() {
			if buf, err := openFile(arg); err == nil {
				go func() {
					status <- fmt.Sprintf(`Opened "%s".`, flag.Arg(0))
				}()
				panes = append(panes, Pane{buf, arg})
			} else {
				go func() { status <- err.Error() }()
			}
		}
	} else {
		go func() { status <- "New file." }()
	}
	if len(panes) == 0 {
		panes = append(panes, Pane{edit.NewBuffer(), "[new file]"})
	}

	go renderLoop(font, win)
	paneSet <- panes
	w, h := win.GetSize()
	resize(panes, font, w, h)
	eventLoop(panes, font, win)
}
