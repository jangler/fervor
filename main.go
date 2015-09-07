package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const version = "0.3.0"

const (
	insMark = iota // ID of the cursor/insertion mark
	selMark        // ID of the selection anchor mark
)

var (
	darkFlag      = false
	expandtabFlag = false
	fontFlag      = ""
	ptsizeFlag    = 12
	tabstopFlag   = 8
	versionFlag   = false
)

var sectionFlags = make(map[string]map[string]string)
var shebangRegexp = regexp.MustCompile(`^#!(/usr/bin/env |/.+/)(.+)( |$)`)

// readIni reads option defaults from the .ini file, if one exists.
func readIni() {
	if curUser, err := user.Current(); err == nil {
		paths := []string{
			filepath.Join(curUser.HomeDir, "fervor.ini"),
			filepath.Join(curUser.HomeDir, ".config", "fervor.ini"),
		}
		for _, path := range paths {
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				continue
			}
			section := ""
			sectionFlags[""] = make(map[string]string)
			for _, line := range strings.Split(string(contents), "\n") {
				// ignore comment lines
				if strings.HasPrefix(line, ";") {
					continue
				}

				if match, _ := regexp.MatchString(`^\[.+\]$`, line); match {
					section = line
					sectionFlags[section] = make(map[string]string)
				} else {
					tokens := strings.SplitN(line, "=", 2)
					if len(tokens) == 2 {
						if section == "" {
							flag.Set(tokens[0], tokens[1]) // ignore errors
						}
						sectionFlags[section][tokens[0]] = tokens[1]
					}
				}
			}
			break // successfully read .ini file
		}
	} else {
		log.Print(err)
	}
}

// setFileFlags sets flags based on INI settings, the current file path, and
// the first line of the buffer, and returns syntax rules to be used for the
// file.
func setFileFlags(path, line string) []edit.Rule {
	fn := filepath.Base(path)
	var sb string
	if subs := shebangRegexp.FindStringSubmatch(line); subs != nil {
		sb = subs[2]
	}

	// first, reset flags to defaults
	for k, v := range sectionFlags[""] {
		flag.Set(k, v) // ignore errors
	}

	for section, flags := range sectionFlags {
		match := false
		if globs, ok := flags["filename"]; ok {
			for _, glob := range strings.Split(globs, ";") {
				if match, _ = filepath.Match(glob, fn); match {
					break
				}
			}
		}
		if shebangs, ok := flags["shebang"]; ok && !match && sb != "" {
			for _, shebang := range strings.Split(shebangs, ";") {
				if shebang == sb {
					match = true
					break
				}
			}
		}
		if match {
			for k, v := range flags {
				flag.Set(k, v) // ignore errors
			}
			clampFlags()
			if syntaxFunc, ok := syntaxMap[section]; ok {
				return syntaxFunc()
			}
			break
		}
	}

	return []edit.Rule{}
}

// clampFlags keeps flags within reasonable bounds.
func clampFlags() {
	if ptsizeFlag < 8 {
		ptsizeFlag = 8
	}
	if tabstopFlag < 1 {
		tabstopFlag = 1
	}
}

// initFlags initializes the flag package.
func initFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<option> ...] [<file> ...]\n",
			os.Args[0])
		fmt.Fprintln(os.Stderr, "\nOptions:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Global and file-specific default options can be specified in either
~/fervor.ini or ~/.config/fervor.ini.`)
	}
	flag.BoolVar(&darkFlag, "dark", darkFlag, "use dark color scheme")
	flag.BoolVar(&expandtabFlag, "expandtab", expandtabFlag,
		"insert spaces using the Tab key")
	flag.StringVar(&fontFlag, "font", fontFlag,
		"use the font at the given path")
	flag.IntVar(&ptsizeFlag, "ptsize", ptsizeFlag, "set point size of font")
	flag.IntVar(&tabstopFlag, "tabstop", tabstopFlag,
		"set width of tab stops, in columns")
	flag.BoolVar(&versionFlag, "version", versionFlag,
		"print version information and exit")
}

// parseFlags processes command-line flags.
func parseFlags() {
	flag.Parse()
	clampFlags()

	if versionFlag {
		fmt.Printf("%s version %s %s/%s\n", os.Args[0], version, runtime.GOOS,
			runtime.GOARCH)
		os.Exit(0)
	}

	sectionFlags[""] = map[string]string{
		"dark":      fmt.Sprintf("%v", darkFlag),
		"expandtab": fmt.Sprintf("%v", expandtabFlag),
		"font":      fmt.Sprintf("%v", fontFlag),
		"ptsize":    fmt.Sprintf("%v", ptsizeFlag),
		"tabstop":   fmt.Sprintf("%v", tabstopFlag),
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
	syntaxRules := setFileFlags(path,
		buf.Get(edit.Index{1, 0}, edit.Index{1, 1 << 30}))
	buf.SetSyntax(syntaxRules)
	return buf, nil
}

// lineEnding returns the line ending string used for a buffer, and converts
// the buffer to Unix line endings if it uses DOS line endings.
func lineEnding(b *edit.Buffer) string {
	if strings.Contains(b.Get(edit.Index{1, 0}, edit.Index{2, 0}), "\r\n") {
		text := b.Get(edit.Index{1, 0}, b.End())
		b.Delete(edit.Index{1, 0}, b.End())
		b.Insert(edit.Index{1, 0}, strings.Replace(text, "\r", "", -1))
		return "\r\n"
	}
	return "\n"
}

func main() {
	log.SetFlags(log.Lshortfile)
	initFlags()
	readIni()
	parseFlags()
	setColorScheme()

	// init SDL
	runtime.LockOSThread()
	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()
	if err := ttf.Init(); err != nil {
		log.Fatal(err)
	}
	defer ttf.Quit()

	// open new instances for other file args
	if flag.NArg() > 1 {
		for _, arg := range flag.Args()[1:] {
			newInstance(arg, "")
		}
	}

	// init buffer
	var arg, status string
	if flag.NArg() == 0 || flag.Arg(0) == "" {
		arg = os.DevNull
	} else {
		arg = flag.Arg(0)
	}
	var buf *edit.Buffer
	var err error
	if buf, err = openFile(arg); err == nil {
		status = fmt.Sprintf(`Opened "%s".`, minPath(arg))
	} else {
		status = fmt.Sprintf(`New file: "%s".`, minPath(arg))
		buf = edit.NewBuffer()
		buf.SetSyntax(setFileFlags(arg, ""))
	}
	pane := &Pane{buf, minPath(arg), tabstopFlag, 80, 25, lineEnding(buf)}
	if pane.LineEnding == "\r\n" {
		status += " [DOS]"
	}
	pane.SetTabWidth(tabstopFlag)
	pane.Mark(edit.Index{1, 0}, selMark, insMark)
	font := getFont()
	win := createWindow(minPath(arg), font)
	defer win.Destroy()
	w, h := win.GetSize()
	resize(pane, w, h)

	eventLoop(pane, status, font, win)
}
