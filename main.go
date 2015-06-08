package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

// SetSyntax automatically sets the syntax rules for p.
func (p *Pane) SetSyntax() {
	if strings.HasSuffix(p.Title, ".go") {
		p.Buffer.SetSyntax(goRules)
	} else if strings.HasSuffix(p.Title, ".json") {
		p.Buffer.SetSyntax(jsonRules)
	} else {
		p.Buffer.SetSyntax([]edit.Rule{})
	}
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

	var pane *Pane
	var arg, status string
	if flag.NArg() == 0 {
		arg = os.DevNull
	} else {
		arg = flag.Arg(0)
	}
	if buf, err := openFile(arg); err == nil {
		status = fmt.Sprintf(`Opened "%s".`, arg)
		pane = &Pane{buf, arg, 4, 80, 25}
	} else {
		status = fmt.Sprintf(`New file: "%s".`, arg)
		pane = &Pane{edit.NewBuffer(), arg, 4, 80, 25}
	}
	pane.SetTabWidth(4)
	pane.SetSyntax()
	pane.Mark(edit.Index{1, 0}, insertMark)

	win := createWindow(arg, font)
	defer win.Destroy()

	w, h := win.GetSize()
	resize(pane, font, w, h)
	win.SetSize(w, h) // force correct window size
	eventLoop(pane, status, font, win)
}
