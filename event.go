package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const (
	openPrompt = "Open: "
)

var (
	spaceRegexp = regexp.MustCompile(`\s`) // matches whitespace characters
	wordRegexp  = regexp.MustCompile(`\w`) // matches word characters
)

// singleClick processes a single left mouse click at the given coordinates.
func singleClick(pane *Pane, win *sdl.Window, font *ttf.Font, x, y int) {
	_, height := win.GetSize()
	ps := paneSpace(height, 1, font)
	y = y % ps
	x -= padPx - fontWidth/2
	y /= font.Height()
	x /= fontWidth
	pane.Mark(pane.IndexFromCoords(x, y), insertMark)
}

// textInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func textInput(buf *edit.Buffer, s string) {
	index := buf.IndexFromMark(insertMark)
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
		// TODO
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
	text := pane.Get(edit.Index{1, 0}, pane.End())
	return ioutil.WriteFile(pane.Title, []byte(text), 0664)
}

// Prompt enters into prompt mode, prompting for input with the given string.
func (rc *RenderContext) Prompt(s string) {
	rc.Status = s
	rc.Focus = rc.Input
	rc.Input.Delete(edit.Index{1, 0}, rc.Input.End())
}

// EnterInput exits prompt mode, taking action based on the prompt string and
// input text.
func (rc *RenderContext) EnterInput() {
	input := rc.Input.Get(edit.Index{1, 0}, rc.Input.End())
	switch rc.Status {
	case openPrompt:
		if contents, err := ioutil.ReadFile(input); err == nil {
			rc.Pane.Delete(edit.Index{1, 0}, rc.Pane.End())
			rc.Pane.Insert(edit.Index{1, 0}, string(contents))
			rc.Status = fmt.Sprintf(`Opened "%s".`, input)
			rc.Pane.Mark(edit.Index{1, 0}, insertMark)
			rc.Pane.Title = input
			rc.Pane.SetSyntax()
		} else {
			rc.Status = err.Error()
		}
	}
	rc.Focus = rc.Pane.Buffer
}

// eventLoop handles SDL events until quit is requested.
func eventLoop(pane *Pane, status string, font *ttf.Font, win *sdl.Window) {
	rc := &RenderContext{pane, edit.NewBuffer(), pane.Buffer, status, font,
		win}
	rc.Input.Mark(edit.Index{1, 0}, insertMark)
	render(rc)
	for {
		switch event := sdl.WaitEvent().(type) {
		case *sdl.KeyDownEvent:
			if rc.Focus == rc.Pane.Buffer {
				rc.Status = rc.Pane.Title
			}
			recognized := true
			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
			case sdl.K_DELETE:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Delete(index, rc.Focus.ShiftIndex(index, 1))
			case sdl.K_DOWN:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Focus.IndexFromMark(insertMark)
					col, row := rc.Focus.CoordsFromIndex(index)
					rc.Focus.Mark(rc.Focus.IndexFromCoords(col, row+1),
						insertMark)
				}
			case sdl.K_END:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
			case sdl.K_LEFT:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(rc.Focus.ShiftIndex(index, -1), insertMark)
			case sdl.K_HOME:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
			case sdl.K_PAGEDOWN:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Pane.IndexFromMark(insertMark)
					col, row := rc.Pane.CoordsFromIndex(index)
					rc.Pane.Mark(rc.Pane.IndexFromCoords(col,
						row+rc.Pane.Rows), insertMark)
				}
			case sdl.K_PAGEUP:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Pane.IndexFromMark(insertMark)
					col, row := rc.Pane.CoordsFromIndex(index)
					rc.Pane.Mark(rc.Pane.IndexFromCoords(col,
						row-rc.Pane.Rows), insertMark)
				}
			case sdl.K_RETURN:
				if rc.Focus == rc.Pane.Buffer {
					textInput(rc.Focus, "\n")
				} else {
					rc.EnterInput()
				}
			case sdl.K_RIGHT:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(rc.Focus.ShiftIndex(index, 1), insertMark)
			case sdl.K_TAB:
				if rc.Focus == rc.Pane.Buffer {
					textInput(rc.Focus, "\t")
				}
			case sdl.K_UP:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Focus.IndexFromMark(insertMark)
					col, row := rc.Focus.CoordsFromIndex(index)
					rc.Focus.Mark(rc.Focus.IndexFromCoords(col, row-1),
						insertMark)
				}
			case sdl.K_a:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
				}
			case sdl.K_e:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
				}
			case sdl.K_h:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
				}
			case sdl.K_o:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					rc.Prompt(openPrompt)
				}
			case sdl.K_q:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if rc.Focus == rc.Pane.Buffer {
						return
					} else {
						rc.Status = rc.Pane.Title
						rc.Focus = rc.Pane.Buffer
					}
				}
			case sdl.K_s:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if rc.Focus != rc.Pane.Buffer {
						break
					}
					if err := saveFile(rc.Pane); err == nil {
						rc.Status = fmt.Sprintf(`Saved "%s".`, rc.Pane.Title)
					} else {
						rc.Status = err.Error()
					}
				}
			case sdl.K_u:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Delete(edit.Index{index.Line, 0}, index)
				}
			case sdl.K_w:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end := rc.Focus.IndexFromMark(insertMark)
					begin := shiftIndexByWord(rc.Focus, end, -1)
					rc.Focus.Delete(begin, end)
				}
			default:
				recognized = false
			}
			if recognized {
				rc.Pane.See(insertMark)
				render(rc)
			}
		case *sdl.MouseButtonEvent:
			if event.Type == sdl.MOUSEBUTTONDOWN &&
				event.Button == sdl.BUTTON_LEFT {
				singleClick(rc.Pane, win, font, int(event.X), int(event.Y))
				render(rc)
			}
		case *sdl.MouseWheelEvent:
			rc.Pane.Scroll(int(event.Y) * -3)
			render(rc)
		case *sdl.QuitEvent:
			return
		case *sdl.TextInputEvent:
			if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
				textInput(rc.Focus, string(event.Text[:n]))
				render(rc)
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
