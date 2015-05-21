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

const insertMark = iota // ID of the insertion (cursor) mark

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
	buf.Mark(edit.Index{1, 0}, insertMark)
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
	var err error
	fontWidth, _, err = font.SizeUTF8("0")
	if err != nil {
		log.Fatal(err)
	}

	win := createWindow(os.Args[0], font)
	defer win.Destroy()

	panes := make([]Pane, 0)
	args := flag.Args()
	if len(args) == 0 {
		args = []string{os.DevNull}
	}
	for _, arg := range args {
		if buf, err := openFile(arg); err == nil {
			go func() {
				status <- fmt.Sprintf(`Opened "%s".`, arg)
			}()
			buf.SetTabWidth(4)
			panes = append(panes, Pane{buf, arg, 4, 80, 25})
		} else {
			go func() { status <- err.Error() }()
		}
	}

	go renderLoop(font, win)
	paneSet <- panes
	w, h := win.GetSize()
	resize(panes, font, w, h)
	eventLoop(panes, font, win)
}
