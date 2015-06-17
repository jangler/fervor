package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const version = "0.1.0"

const (
	insMark = iota // ID of the cursor/insertion mark
	selMark        // ID of the selection anchor mark
)

var (
	expandtabFlag = false
	fontFlag      = ""
	ptsizeFlag    = 12
	tabstopFlag   = 8
	versionFlag   = false
)

// readIni reads option defaults from the .ini file, if one exists.
func readIni() {
	if curUser, err := user.Current(); err == nil {
		paths := []string{
			filepath.Join(curUser.HomeDir, "fervor.ini"),
			filepath.Join(curUser.HomeDir, ".config", "fervor.ini"),
		}
		for _, path := range paths {
			if contents, err := ioutil.ReadFile(path); err == nil {
				for _, line := range strings.Split(string(contents), "\n") {
					// ignore comment lines
					if strings.HasPrefix(line, ";") {
						continue
					}

					tokens := strings.SplitN(line, "=", 2)
					if len(tokens) == 2 {
						flag.Set(tokens[0], tokens[1]) // ignore errors
					}
				}
				break
			}
		}
	} else {
		log.Print(err)
	}
}

// initFlags initializes the flag package.
func initFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<option> ...] [<file>]\n",
			os.Args[0])
		fmt.Fprintln(os.Stderr, "\nOptions:")
		flag.PrintDefaults()
	}
	flag.BoolVar(&expandtabFlag, "expandtab", expandtabFlag,
		"insert spaces using the Tab key")
	flag.StringVar(&fontFlag, "font", fontFlag,
		"use the TTF at the given path")
	flag.IntVar(&ptsizeFlag, "ptsize", ptsizeFlag, "set point size of font")
	flag.IntVar(&tabstopFlag, "tabstop", tabstopFlag,
		"set width of tab stops, in columns")
	flag.BoolVar(&versionFlag, "version", versionFlag,
		"print version information and exit")
}

// parseFlags processes command-line flags.
func parseFlags() {
	flag.Parse()

	// you're joking, right?
	if ptsizeFlag < 8 {
		ptsizeFlag = 8
	}
	if tabstopFlag < 1 {
		tabstopFlag = 1
	}

	if versionFlag {
		fmt.Printf("%s version %s %s/%s\n", os.Args[0], version, runtime.GOOS,
			runtime.GOARCH)
		os.Exit(0)
	}

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
	log.SetFlags(log.Lshortfile)
	initFlags()
	readIni()
	parseFlags()

	// init SDL
	runtime.LockOSThread()
	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()
	if err := ttf.Init(); err != nil {
		log.Fatal(err)
	}
	defer ttf.Quit()
	font := getFont()

	// init buffer
	var pane *Pane
	var arg, status string
	if flag.NArg() == 0 || flag.Arg(0) == "" {
		arg = os.DevNull
	} else {
		arg = flag.Arg(0)
	}
	if buf, err := openFile(arg); err == nil {
		status = fmt.Sprintf(`Opened "%s".`, minPath(arg))
		pane = &Pane{buf, minPath(arg), tabstopFlag, 80, 25}
	} else {
		status = fmt.Sprintf(`New file: "%s".`, minPath(arg))
		pane = &Pane{edit.NewBuffer(), minPath(arg), tabstopFlag, 80, 25}
	}
	pane.SetTabWidth(tabstopFlag)
	pane.SetSyntax()
	pane.Mark(edit.Index{1, 0}, selMark, insMark)
	win := createWindow(minPath(arg), font)
	defer win.Destroy()
	w, h := win.GetSize()
	resize(pane, w, h)

	eventLoop(pane, status, font, win)
}
