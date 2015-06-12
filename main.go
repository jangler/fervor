package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const (
	insertMark = iota // ID of the cursor/insertion mark
	selMark           // ID of the selection anchor mark
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
	if buf.Get(buf.ShiftIndex(buf.End(), -1), buf.End()) == "\n" {
		buf.Delete(buf.ShiftIndex(buf.End(), -1), buf.End())
	}
	buf.ResetModified()
	buf.ResetUndo()
	return buf, nil
}

// SetSyntax automatically sets the syntax rules for p.
func (p *Pane) SetSyntax() {
	if strings.HasSuffix(p.Title, ".c") || strings.HasSuffix(p.Title, ".h") {
		p.Buffer.SetSyntax(cRules())
	} else if strings.HasSuffix(p.Title, ".go") {
		p.Buffer.SetSyntax(goRules())
	} else if strings.HasSuffix(p.Title, ".json") {
		p.Buffer.SetSyntax(jsonRules())
	} else if strings.ToLower(p.Title) == "makefile" {
		p.Buffer.SetSyntax(makefileRules())
	} else if strings.HasSuffix(p.Title, ".py") {
		p.Buffer.SetSyntax(pythonRules())
	} else if strings.HasSuffix(p.Title, ".sh") {
		p.Buffer.SetSyntax(pythonRules())
	} else {
		firstLine := p.Buffer.Get(edit.Index{1, 0}, edit.Index{1, 1 << 30})
		if strings.HasPrefix(firstLine, "#!") {
			if strings.Contains(firstLine, "python") {
				p.Buffer.SetSyntax(pythonRules())
			} else if strings.Contains(firstLine, "sh") {
				p.Buffer.SetSyntax(bashRules())
			} else {
				p.Buffer.SetSyntax([]edit.Rule{})
			}
		} else {
			p.Buffer.SetSyntax([]edit.Rule{})
		}
	}
}

func main() {
	initFlag()
	log.SetFlags(log.Lshortfile)

	runtime.LockOSThread()
	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()
	if err := ttf.Init(); err != nil {
		log.Fatal(err)
	}
	defer ttf.Quit()

	font := getFont()
	var err error
	fontWidth, _, err = font.SizeUTF8("0")
	if err != nil {
		log.Fatal(err)
	}

	var pane *Pane
	var arg, status string
	if flag.NArg() == 0 || flag.Arg(0) == "" {
		arg = os.DevNull
	} else {
		arg = flag.Arg(0)
	}
	if buf, err := openFile(arg); err == nil {
		status = fmt.Sprintf(`Opened "%s".`, minPath(arg))
		pane = &Pane{buf, minPath(arg), 4, 80, 25}
	} else {
		status = fmt.Sprintf(`New file: "%s".`, minPath(arg))
		pane = &Pane{edit.NewBuffer(), minPath(arg), 4, 80, 25}
	}
	pane.SetTabWidth(4)
	pane.SetSyntax()
	pane.Mark(edit.Index{1, 0}, insertMark)
	pane.Mark(edit.Index{1, 0}, selMark)

	win := createWindow(minPath(arg), font)
	defer win.Destroy()

	w, h := win.GetSize()
	resize(pane, font, w, h)
	eventLoop(pane, status, font, win)
}
