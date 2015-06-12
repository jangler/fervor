package main

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const (
	cdPrompt            = "Change directory to: "
	findBackwardPrompt  = "Find backward: "
	findForwardPrompt   = "Find forward: "
	goToLinePrompt      = "Go to line: "
	openNewPrompt       = "Open in new window: "
	openPrompt          = "Open: "
	pipePrompt          = "Pipe selection through: "
	reallyOpenPrompt    = "Really open (y/n)? "
	reallyQuitPrompt    = "Really quit (y/n)? "
	runPrompt           = "Run: "
	saveAsPrompt        = "Save as: "
)

// user event types. these are only vars in order to be addressable.
var (
	pipeEvent   = 1
	statusEvent = 2
)

var (
	spaceRegexp = regexp.MustCompile(`\s`) // matches whitespace characters
	wordRegexp  = regexp.MustCompile(`\w`) // matches word characters
)

var userEventType uint32 // set at beginning of event loop

var shellName, shellOpt string

func init() {
	if runtime.GOOS == "windows" {
		shellName, shellOpt = "cmd", "/c"
	} else {
		shellName, shellOpt = "/bin/sh", "-c"
	}
}

// minPath returns the shortest valid representation of the given file path.
func minPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	if wd, err := os.Getwd(); err == nil {
		if relWd, err := filepath.Rel(wd, abs); err == nil {
			if len(relWd) < len(path) {
				path = relWd
			}
		}
	}
	if curUser, err := user.Current(); err == nil {
		if relHome, err := filepath.Rel(curUser.HomeDir, abs); err == nil {
			relHome = "~/" + relHome
			if len(relHome) < len(path) {
				path = relHome
			}
		}
	}

	return filepath.Clean(path)
}

// selectWord selects the word at the given index in the pane.
func selectWord(pane *Pane, index edit.Index) {
	selIndex, insertIndex := index, index
	for wordRegexp.MatchString(pane.Get(pane.ShiftIndex(selIndex, -1),
		selIndex)) {
		selIndex = pane.ShiftIndex(selIndex, -1)
	}
	for wordRegexp.MatchString(pane.Get(insertIndex,
		pane.ShiftIndex(insertIndex, 1))) {
		insertIndex = pane.ShiftIndex(insertIndex, 1)
	}
	pane.Mark(selIndex, selMark)
	pane.Mark(insertIndex, insertMark)
}

// colRowFromXY converts (x, y) coordinates in a window to a row and column.
func colRowFromXY(rc *RenderContext, x, y int) (col, row int) {
	_, height := rc.Window.GetSize()
	ps := paneSpace(height, 1, rc.Font)
	y = y % ps
	x -= padPx - fontWidth/2
	y /= rc.Font.Height()
	x /= fontWidth
	return x, y
}

// click processes a left mouse click at the given coordinates.
func click(rc *RenderContext, x, y, times int, shift bool) {
	pane := rc.Pane
	x, y = colRowFromXY(rc, x, y)

	switch times {
	case 1: // place cursor
		pane.Mark(pane.IndexFromCoords(x, y), insertMark)
		if !shift {
			pane.Mark(pane.IndexFromMark(insertMark), selMark)
		}
	case 2: // select word
		selectWord(pane, pane.IndexFromCoords(x, y))
	case 3: // select line
		index := pane.IndexFromCoords(x, y)
		pane.Mark(edit.Index{index.Line, 0}, selMark)
		pane.Mark(edit.Index{index.Line, 1 << 30}, insertMark)
	}
}

// clickFind moves the cursor and selection to the next or previous instance of
// the selected text.
func clickFind(rc *RenderContext, shift bool, x, y int) {
	pane := rc.Pane
	x, y = colRowFromXY(rc, x, y)

	// get selection
	selIndex := pane.IndexFromMark(selMark)
	insertIndex := pane.IndexFromMark(insertMark)
	selIndex, insertIndex = order(selIndex, insertIndex)

	// reposition cursor if click is outside selection
	clickIndex := pane.IndexFromCoords(x, y)
	if clickIndex.Less(selIndex) || insertIndex.Less(clickIndex) {
		pane.Mark(clickIndex, selMark)
		pane.Mark(clickIndex, insertMark)
		selIndex = pane.IndexFromMark(selMark)
		insertIndex = pane.IndexFromMark(insertMark)
	}

	// select word if selection is nil
	if selIndex == insertIndex {
		selectWord(pane, selIndex)
		selIndex = pane.IndexFromMark(selMark)
		insertIndex = pane.IndexFromMark(insertMark)
	}
	selection := pane.Get(selIndex, insertIndex)

	if shift { // search backwards
		index := selIndex
		text := pane.Get(edit.Index{1, 0}, index)
		if pos := strings.LastIndex(text, selection); pos >= 0 {
			pane.Mark(pane.ShiftIndex(index, pos-len(text)), selMark)
			pane.Mark(pane.ShiftIndex(pane.IndexFromMark(selMark),
				len(selection)), insertMark)
		} else {
			rc.Status = "No backward match."
		}
	} else { // search forwards
		index := insertIndex
		text := pane.Get(index, pane.End())
		if pos := strings.Index(text, selection); pos >= 0 {
			pane.Mark(pane.ShiftIndex(index, pos), selMark)
			pane.Mark(pane.ShiftIndex(pane.IndexFromMark(selMark),
				len(selection)), insertMark)
		} else {
			rc.Status = "No forward match."
		}
	}
}

// order returns index1 and index2 in buffer order.
func order(index1, index2 edit.Index) (first, second edit.Index) {
	if index1.Less(index2) {
		return index1, index2
	}
	return index2, index1
}

// textInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func textInput(buf *edit.Buffer, s string) {
	index := buf.IndexFromMark(insertMark)
	if sel := buf.IndexFromMark(selMark); sel != index {
		buf.Delete(order(sel, index))
		index, _ = order(sel, index)
	}
	if s == "\n" {
		// autoindent
		indent := buf.Get(edit.Index{index.Line, 0},
			edit.Index{index.Line, 0xffff})
		i := 0
		for i < len(indent) && (indent[i] == ' ' || indent[i] == '\t') {
			i++
		}
		if i <= index.Char {
			s += indent[:i]

			// delete lines containing only whitespace
			if i == len(indent) {
				buf.Delete(edit.Index{index.Line, 0}, index)
				index.Char = 0
			}
		}
	}
	buf.Insert(index, s)
}

// resize resizes the pane in the display
func resize(pane *Pane, font *ttf.Font, width, height int) {
	cols, rows := bufSize(width, height, 1, font)
	pane.Cols, pane.Rows = cols, rows
	pane.SetSize(cols, rows)
}

// shiftIndexByWord returns the given index shifted forward by n words. A
// negative value for n will shift backwards.
func shiftIndexByWord(b *edit.Buffer, index edit.Index, n int) edit.Index {
	for n > 0 {
		text := []rune(b.Get(index, edit.Index{index.Line, 1 << 30}))
		if len(text) == 0 {
			index = b.ShiftIndex(index, 1)
		} else {
			i := 0
			for i < len(text) && spaceRegexp.MatchString(string(text[i])) {
				i++
			}
			if i < len(text) && wordRegexp.MatchString(string(text[i])) {
				for i < len(text) && wordRegexp.MatchString(string(text[i])) {
					i++
				}
			} else {
				for i < len(text) &&
					!wordRegexp.MatchString(string(text[i])) &&
					!spaceRegexp.MatchString(string(text[i])) {
					i++
				}
			}
			index.Char += i
		}
		n--
	}
	for n < 0 {
		if index.Char == 0 {
			index = b.ShiftIndex(index, -1)
		} else {
			text := []rune(b.Get(edit.Index{index.Line, 0}, index))
			i := len(text) - 1
			for i >= 0 && spaceRegexp.MatchString(string(text[i])) {
				i--
			}
			if i >= 0 && wordRegexp.MatchString(string(text[i])) {
				for i >= 0 && wordRegexp.MatchString(string(text[i])) {
					i--
				}
			} else {
				for i >= 0 && !wordRegexp.MatchString(string(text[i])) &&
					!spaceRegexp.MatchString(string(text[i])) {
					i--
				}
			}
			index.Char = i + 1
		}
		n++
	}
	return index
}

// saveFile writes the contents of pane to a file with the name of the pane's
// title.
func saveFile(pane *Pane) error {
	text := pane.Get(edit.Index{1, 0}, pane.End()) + "\n"
	path, err := filepath.Abs(expandVars(pane.Title))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, []byte(text), 0664)
	if err == nil {
		pane.ResetModified()
	}
	return err
}

// Prompt enters into prompt mode, prompting for input with the given string.
func (rc *RenderContext) Prompt(s string) {
	rc.Input.ResetUndo()
	rc.Pane.Separate()
	rc.Status = s
	rc.Focus = rc.Input
	rc.Input.Delete(edit.Index{1, 0}, rc.Input.End())
}

// find attempts a regex search in the buffer.
func find(rc *RenderContext, forward bool) {
	if rc.Regexp == nil {
		rc.Status = "No pattern to find."
		return
	}
	rc.Status = rc.Pane.Title

	pane := rc.Pane
	selIndex := pane.IndexFromMark(selMark)
	insIndex := pane.IndexFromMark(insertMark)

	if forward {
		_, index := order(selIndex, insIndex)
		text := pane.Get(index, pane.End())
		if loc := rc.Regexp.FindStringIndex(text); loc != nil {
			pane.Mark(pane.ShiftIndex(index, loc[0]), selMark)
			pane.Mark(pane.ShiftIndex(index, loc[1]), insertMark)
			pane.Separate()
		} else {
			rc.Status = "No forward match."
		}
	} else {
		index, _ := order(selIndex, insIndex)
		text := pane.Get(edit.Index{1, 0}, index)
		if locs := rc.Regexp.FindAllStringIndex(text, -1); locs != nil {
			loc := locs[len(locs)-1]
			pane.Mark(pane.ShiftIndex(edit.Index{1, 0}, loc[0]), selMark)
			pane.Mark(pane.ShiftIndex(edit.Index{1, 0}, loc[1]), insertMark)
			pane.Separate()
		} else {
			rc.Status = "No backward match."
		}
	}
}

// expandVars returns a version of path with environment variables expanded.
func expandVars(path string) string {
	path = os.ExpandEnv(path)
	if curUser, err := user.Current(); err == nil {
		path = strings.Replace(path, "~/", curUser.HomeDir+"/", -1)
	}
	return path
}

// reportExitStatus pushes a status message to the SDL event queue depending
// on err (which may be nil).
func reportExitStatus(cmd string, err error) {
	var event sdl.UserEvent
	var msg string
	if err == nil {
		msg = fmt.Sprintf(`Command "%s" exited successfully.`, cmd)
	} else {
		msg = fmt.Sprintf(`Command "%s" exited with error: %v`,
			cmd, err)
	}
	event.Type, event.Data1 = userEventType, unsafe.Pointer(&statusEvent)
	event.Data2 = unsafe.Pointer(&msg)
	sdl.PushEvent(&event)
}

// getSelection returns the selected text in the buffer.
func getSelection(b *edit.Buffer) string {
	sel, ins := b.IndexFromMark(selMark), b.IndexFromMark(insertMark)
	return b.Get(order(sel, ins))
}

// EnterInput exits prompt mode, taking action based on the prompt string and
// input text. Returns false if the application should quit.
func (rc *RenderContext) EnterInput() bool {
	input := rc.Input.Get(edit.Index{1, 0}, rc.Input.End())
	switch rc.Status {
	case cdPrompt:
		input = expandVars(input)
		if abs, err := filepath.Abs(input); err == nil {
			input = abs
		}
		name := expandVars(rc.Pane.Title)
		if abs, err := filepath.Abs(name); err == nil {
			name = abs
		}
		if err := os.Chdir(input); err == nil {
			rc.Status = fmt.Sprintf(`Working dir is "%s".`, input)
			rc.Pane.Title = minPath(name)
			rc.Window.SetTitle(rc.Pane.Title)
		} else {
			rc.Status = err.Error()
		}
	case findBackwardPrompt:
		if re, err := regexp.Compile(input); err == nil {
			rc.Regexp = re
			find(rc, false)
		} else {
			rc.Status = err.Error()
		}
	case findForwardPrompt:
		if re, err := regexp.Compile(input); err == nil {
			rc.Regexp = re
			find(rc, true)
		} else {
			rc.Status = err.Error()
		}
	case goToLinePrompt:
		if n, err := strconv.ParseInt(input, 0, 0); err == nil {
			rc.Status = rc.Pane.Title
			rc.Pane.Mark(edit.Index{int(n), 0}, selMark)
			rc.Pane.Mark(edit.Index{int(n), 1 << 30}, insertMark)
		} else {
			rc.Status = err.Error()
		}
	case openPrompt:
		if input == "" {
			rc.Status = rc.Pane.Title
			break
		}
		rc.Pane.Delete(edit.Index{1, 0}, rc.Pane.End())
		input = expandVars(input)
		if contents, err := ioutil.ReadFile(input); err == nil {
			rc.Pane.Insert(edit.Index{1, 0}, string(contents))
			penult := rc.Pane.ShiftIndex(rc.Pane.End(), -1)
			if rc.Pane.Get(penult, rc.Pane.End()) == "\n" {
				rc.Pane.Delete(penult, rc.Pane.End())
			}
			rc.Status = fmt.Sprintf(`Opened "%s".`, minPath(input))
		} else {
			rc.Status = fmt.Sprintf(`New file: "%s".`, minPath(input))
		}
		rc.Pane.Mark(edit.Index{1, 0}, insertMark)
		rc.Pane.Mark(edit.Index{1, 0}, selMark)
		rc.Pane.Title = minPath(input)
		rc.Window.SetTitle(rc.Pane.Title)
		rc.Pane.SetSyntax()
		rc.Pane.ResetModified()
		rc.Pane.ResetUndo()
	case openNewPrompt:
		cmd := exec.Command(os.Args[0], expandVars(input))
		if err := cmd.Start(); err == nil {
			rc.Status = rc.Pane.Title
		} else {
			rc.Status = err.Error()
		}
	case pipePrompt:
		rc.Status = rc.Pane.Title
		if input == "" {
			break
		}

		// initialize command
		cmd := exec.Command(shellName, shellOpt, input)
		inPipe, err := cmd.StdinPipe()
		if err != nil {
			rc.Status = err.Error()
			break
		}
		outPipe, err := cmd.StdoutPipe()
		if err != nil {
			rc.Status = err.Error()
			break
		}
		if err := cmd.Start(); err != nil {
			rc.Status = err.Error()
			break
		}

		go func() {
			// write to stdin and read from stdout
			go func() {
				io.WriteString(inPipe, getSelection(rc.Pane.Buffer))
				inPipe.Close()
			}()
			var outBytes []byte
			go func() {
				outBytes, _ = ioutil.ReadAll(outPipe)
			}()

			reportExitStatus(input, cmd.Wait())
			if outBytes != nil {
				// strip trailing newline
				if len(outBytes) > 0 && outBytes[len(outBytes)-1] == '\n' {
					outBytes = outBytes[:len(outBytes)-1]
				}

				// push pipe event
				output := string(outBytes)
				var event sdl.UserEvent
				event.Type = userEventType
				event.Data1 = unsafe.Pointer(&pipeEvent)
				event.Data2 = unsafe.Pointer(&output)
				sdl.PushEvent(&event)
			}
		}()
	case reallyOpenPrompt:
		if input == "y" || input == "yes" {
			rc.Prompt(openPrompt)
			return true // so that main buffer isn't focused
		} else {
			rc.Status = rc.Pane.Title
		}
	case reallyQuitPrompt:
		if input == "y" || input == "yes" {
			return false
		} else {
			rc.Status = rc.Pane.Title
		}
	case runPrompt:
		rc.Status = rc.Pane.Title
		if input == "" {
			break
		}
		cmd := exec.Command(shellName, shellOpt, input)

		go func() {
			output, err := cmd.CombinedOutput()
			reportExitStatus(input, err)

			if output != nil && len(output) > 0 {
				// generate a random filename
				src := make([]byte, 8)
				for i := range src {
					src[i] = byte(rand.Intn(256))
				}
				name := base32.StdEncoding.EncodeToString(src)

				// write command output to temp file
				path := filepath.Join(os.TempDir(), name)
				file, err := os.Create(path)
				if err != nil {
					return
				}
				file.Write(output)
				file.Close()

				// open temp file in new window, then clean up
				exec.Command(os.Args[0], path).Run()
				defer os.Remove(path)
			}
		}()
	case saveAsPrompt:
		prevTitle := rc.Pane.Title
		input = expandVars(input)
		rc.Pane.Title = input
		if err := saveFile(rc.Pane); err == nil {
			rc.Pane.Title = minPath(input)
			rc.Status = fmt.Sprintf(`Saved "%s".`, rc.Pane.Title)
			rc.Window.SetTitle(rc.Pane.Title)
			rc.Pane.SetSyntax()
			rc.Pane.ResetModified()
		} else {
			rc.Status = err.Error()
			rc.Pane.Title = prevTitle
		}
	}
	rc.Focus = rc.Pane.Buffer
	return true
}

// warpMouseToSel warps the mouse to the center of the buffer selection.
func warpMouseToSel(w *sdl.Window, b *edit.Buffer, fontHeight int) {
	sel, ins := b.IndexFromMark(selMark), b.IndexFromMark(insertMark)
	selCol, selRow := b.CoordsFromIndex(sel)
	insCol, insRow := b.CoordsFromIndex(ins)
	x := (float64(selCol)+float64(insCol))*float64(fontWidth)/2 + padPx
	y := (float64(selRow)+float64(insRow)+1)*float64(fontHeight)/2 + padPx
	w.WarpMouseInWindow(int(x), int(y))
}

// eventLoop handles SDL events until quit is requested.
func eventLoop(pane *Pane, status string, font *ttf.Font, win *sdl.Window) {
	rand.Seed(time.Now().UnixNano())
	userEventType = sdl.RegisterEvents(1)
	rc := &RenderContext{pane, edit.NewBuffer(), pane.Buffer, status, font,
		win, nil}
	rc.Input.Mark(edit.Index{1, 0}, insertMark)
	rc.Input.Mark(edit.Index{1, 0}, selMark)
	render(rc)
	w, h := win.GetSize()
	win.SetSize(w, h)
	clickCount := 0
	lastClick := time.Now()
	var rightClickIndex edit.Index
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			if rc.Focus == rc.Pane.Buffer {
				rc.Status = rc.Pane.Title
			}
			recognized := true

			// get current marks to see if they change based on the event
			prevSel := rc.Pane.IndexFromMark(selMark)
			prevIns := rc.Pane.IndexFromMark(insertMark)

			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end := rc.Focus.IndexFromMark(insertMark)
					begin := shiftIndexByWord(rc.Focus, end, -1)
					rc.Focus.Delete(begin, end)
				} else {
					index := rc.Focus.IndexFromMark(insertMark)
					if sel := rc.Focus.IndexFromMark(selMark); sel != index {
						rc.Focus.Delete(order(sel, index))
					} else {
						rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
					}
				}
			case sdl.K_DELETE:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					begin := rc.Focus.IndexFromMark(insertMark)
					end := shiftIndexByWord(rc.Focus, begin, 1)
					rc.Focus.Delete(begin, end)
				} else {
					index := rc.Focus.IndexFromMark(insertMark)
					if sel := rc.Focus.IndexFromMark(selMark); sel != index {
						rc.Focus.Delete(order(sel, index))
					} else {
						rc.Focus.Delete(index, rc.Focus.ShiftIndex(index, 1))
					}
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.See(insertMark)
				}
			case sdl.K_DOWN:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Focus.IndexFromMark(insertMark)
					col, row := rc.Focus.CoordsFromIndex(index)
					rc.Focus.Mark(rc.Focus.IndexFromCoords(col, row+1),
						insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
							selMark)
					}
					rc.Pane.Separate()
				}
			case sdl.K_ESCAPE:
				if rc.Focus == rc.Input {
					rc.Status = rc.Pane.Title
					rc.Focus = rc.Pane.Buffer
				}
			case sdl.K_END:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					rc.Focus.Mark(rc.Focus.End(), insertMark)
				} else {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 1 << 30}, insertMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_LEFT:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					index = shiftIndexByWord(rc.Focus, index, -1)
					rc.Focus.Mark(index, insertMark)
				} else {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(rc.Focus.ShiftIndex(index, -1), insertMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_HOME:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					rc.Focus.Mark(edit.Index{1, 0}, insertMark)
				} else {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_PAGEDOWN:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Pane.IndexFromMark(insertMark)
					col, row := rc.Pane.CoordsFromIndex(index)
					rc.Pane.Mark(rc.Pane.IndexFromCoords(col,
						row+rc.Pane.Rows), insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
							selMark)
					}
					rc.Pane.Separate()
				}
			case sdl.K_PAGEUP:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Pane.IndexFromMark(insertMark)
					col, row := rc.Pane.CoordsFromIndex(index)
					rc.Pane.Mark(rc.Pane.IndexFromCoords(col,
						row-rc.Pane.Rows), insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
							selMark)
					}
					rc.Pane.Separate()
				}
			case sdl.K_RETURN:
				if rc.Focus == rc.Pane.Buffer {
					textInput(rc.Focus, "\n")
				} else {
					if !rc.EnterInput() {
						return
					}
				}
			case sdl.K_RIGHT:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					index = shiftIndexByWord(rc.Focus, index, 1)
					rc.Focus.Mark(index, insertMark)
				} else {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(rc.Focus.ShiftIndex(index, 1), insertMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_TAB:
				if rc.Focus == rc.Pane.Buffer {
					textInput(rc.Focus, "\t")
				} else {
					input := rc.Input.Get(edit.Index{1, 0}, rc.Input.End())
					input = expandVars(input)
					switch rc.Status {
					case cdPrompt:
						input = completePath(input, true)
					case openPrompt, openNewPrompt, saveAsPrompt:
						input = completePath(input, false)
					case runPrompt:
						tokens := strings.Split(input, " ")
						for i, token := range tokens {
							if i == 0 {
								tokens[i] = completeCmd(token)
							} else {
								tokens[i] = completePath(token, false)
							}
						}
						input = strings.Join(tokens, " ")
					}
					rc.Input.Delete(edit.Index{1, 0}, rc.Input.End())
					rc.Input.Insert(edit.Index{1, 0}, input)
				}
			case sdl.K_UP:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Focus.IndexFromMark(insertMark)
					col, row := rc.Focus.CoordsFromIndex(index)
					rc.Focus.Mark(rc.Focus.IndexFromCoords(col, row-1),
						insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
							selMark)
					}
					rc.Pane.Separate()
				}
			case sdl.K_a:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
							selMark)
					}
					if rc.Focus == rc.Pane.Buffer {
						rc.Pane.Separate()
					}
				}
			case sdl.K_c:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if rc.Focus == rc.Input {
						rc.Status = rc.Pane.Title
						rc.Focus = rc.Pane.Buffer
					} else {
						sdl.SetClipboardText(getSelection(rc.Pane.Buffer))
						rc.Status = "Copied text."
					}
				}
			case sdl.K_d:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					rc.Prompt(cdPrompt)
				}
			case sdl.K_e:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 1 << 30}, insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
							selMark)
					}
					if rc.Focus == rc.Pane.Buffer {
						rc.Pane.Separate()
					}
				}
			case sdl.K_f:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						rc.Prompt(findBackwardPrompt)
					} else {
						rc.Prompt(findForwardPrompt)
					}
				}
			case sdl.K_g:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					rc.Prompt(goToLinePrompt)
				}
			case sdl.K_h:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
				}
			case sdl.K_n:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						find(rc, false)
					} else {
						find(rc, true)
					}
				}
			case sdl.K_o:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						rc.Prompt(openNewPrompt)
					} else {
						if rc.Pane.Modified() {
							rc.Prompt(reallyOpenPrompt)
						} else {
							rc.Prompt(openPrompt)
						}
					}
				}
			case sdl.K_p:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					rc.Prompt(pipePrompt)
				}
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if rc.Focus == rc.Pane.Buffer {
						if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 &&
							rc.Pane.Modified() {
							rc.Prompt(reallyQuitPrompt)
						} else {
							return
						}
					} else {
						rc.Status = rc.Pane.Title
						rc.Focus = rc.Pane.Buffer
					}
				}
			case sdl.K_r:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					rc.Prompt(runPrompt)
				}
			case sdl.K_s:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 &&
					rc.Focus != rc.Input {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						rc.Prompt(saveAsPrompt)
					} else {
						if err := saveFile(rc.Pane); err == nil {
							rc.Status = fmt.Sprintf(`Saved "%s".`,
								rc.Pane.Title)
						} else {
							rc.Status = err.Error()
						}
					}
				}
			case sdl.K_u:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Delete(edit.Index{index.Line, 0}, index)
				}
			case sdl.K_v:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					sel := rc.Focus.IndexFromMark(selMark)
					insert := rc.Focus.IndexFromMark(insertMark)
					if sel != insert {
						rc.Focus.Delete(order(sel, insert))
						insert, _ = order(sel, insert)
					}
					rc.Focus.Insert(insert, sdl.GetClipboardText())
				}
			case sdl.K_w:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end := rc.Focus.IndexFromMark(insertMark)
					begin := shiftIndexByWord(rc.Focus, end, -1)
					rc.Focus.Delete(begin, end)
				}
			case sdl.K_x:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					sel := rc.Focus.IndexFromMark(selMark)
					insert := rc.Focus.IndexFromMark(insertMark)
					sdl.SetClipboardText(rc.Focus.Get(order(sel, insert)))
					rc.Focus.Delete(order(sel, insert))
				}

			case sdl.K_y:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if !rc.Pane.Redo(selMark, insertMark) {
						rc.Status = "Nothing to redo."
					}
				}
			case sdl.K_z:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						if !rc.Pane.Redo(selMark, insertMark) {
							rc.Status = "Nothing to redo."
						}
					} else {
						if !rc.Pane.Undo(selMark, insertMark) {
							rc.Status = "Nothing to undo."
						}
					}
				}
			default:
				recognized = false
			}
			if recognized {
				if prevSel != rc.Pane.IndexFromMark(selMark) ||
					prevIns != rc.Pane.IndexFromMark(insertMark) {
					rc.Pane.See(insertMark)
				}
				render(rc)
			}
		case *sdl.MouseButtonEvent:
			if rc.Focus == rc.Pane.Buffer {
				rc.Status = rc.Pane.Title
			}
			state := sdl.GetKeyboardState()
			shift := state[sdl.SCANCODE_LSHIFT]|state[sdl.SCANCODE_RSHIFT] != 0
			if event.Type == sdl.MOUSEBUTTONDOWN {
				if event.Button == sdl.BUTTON_LEFT {
					if time.Since(lastClick) < time.Second/4 {
						clickCount = clickCount%3 + 1
					} else {
						clickCount = 1
					}
					lastClick = time.Now()
					click(rc, int(event.X), int(event.Y), clickCount, shift)
					render(rc)
				} else if event.Button == sdl.BUTTON_RIGHT {
					x, y := colRowFromXY(rc, int(event.X), int(event.Y))
					rightClickIndex = rc.Pane.IndexFromCoords(x, y)
				}
			} else if event.Type == sdl.MOUSEBUTTONUP &&
				event.Button == sdl.BUTTON_RIGHT {
				clickFind(rc, shift, int(event.X), int(event.Y))
				rc.Pane.See(insertMark)
				warpMouseToSel(rc.Window, rc.Pane.Buffer, rc.Font.Height())
				render(rc)
			}
			rc.Pane.Separate()
		case *sdl.MouseMotionEvent:
			if event.State&sdl.ButtonLMask() != 0 {
				click(rc, int(event.X), int(event.Y), 1, true)
				render(rc)
			} else if event.State&sdl.ButtonRMask() != 0 {
				x, y := colRowFromXY(rc, int(event.X), int(event.Y))
				index := rc.Pane.IndexFromCoords(x, y)
				if index != rightClickIndex {
					rc.Pane.Mark(rightClickIndex, selMark)
					rc.Pane.Mark(index, insertMark)
					render(rc)
				}
			}
		case *sdl.MouseWheelEvent:
			rc.Pane.Scroll(int(event.Y) * -3)
			render(rc)
		case *sdl.QuitEvent:
			return
		case *sdl.TextInputEvent:
			if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
				textInput(rc.Focus, string(event.Text[:n]))
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.See(insertMark)
				}
				render(rc)
			}
		case *sdl.UserEvent:
			switch *(*int)(event.Data1) {
			case pipeEvent:
				sel := rc.Focus.IndexFromMark(selMark)
				insert := rc.Focus.IndexFromMark(insertMark)
				rc.Pane.Delete(order(sel, insert))
				insert, _ = order(sel, insert)
				rc.Pane.Insert(insert, *(*string)(event.Data2))
				rc.Pane.See(insertMark)
				render(rc)
			case statusEvent:
				if rc.Focus != rc.Input {
					rc.Status = *(*string)(event.Data2)
					render(rc)
				}
			}
		case *sdl.WindowEvent:
			switch event.Event {
			case sdl.WINDOWEVENT_EXPOSED:
				win.UpdateSurface()
			case sdl.WINDOWEVENT_RESIZED:
				resize(rc.Pane, font, int(event.Data1), int(event.Data2))
				render(rc)
			}
		}
	}
}
