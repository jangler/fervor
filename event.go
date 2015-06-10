package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

const (
	findBackwardPrompt = "Find backward: "
	findForwardPrompt  = "Find forward: "
	openPrompt         = "Open: "
	reallyOpenPrompt   = "Really open (y/n)? "
	reallyQuitPrompt   = "Really quit (y/n)? "
	saveAsPrompt       = "Save as: "
)

var (
	spaceRegexp = regexp.MustCompile(`\s`) // matches whitespace characters
	wordRegexp  = regexp.MustCompile(`\w`) // matches word characters
)

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
		pane.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
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
		panic("forward shift not implemented")
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
	err := ioutil.WriteFile(pane.Title, []byte(text), 0664)
	if err == nil {
		pane.ResetModified()
	}
	return err
}

// Prompt enters into prompt mode, prompting for input with the given string.
func (rc *RenderContext) Prompt(s string) {
	rc.Pane.ResetUndo()
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

// EnterInput exits prompt mode, taking action based on the prompt string and
// input text. Returns false if the application should quit.
func (rc *RenderContext) EnterInput() bool {
	input := rc.Input.Get(edit.Index{1, 0}, rc.Input.End())
	switch rc.Status {
	case findBackwardPrompt:
		if re, err := regexp.Compile(input); err == nil {
			rc.Regexp = re
			find(rc, false)
		} else {
			rc.Status = err.Error()
			break
		}
	case findForwardPrompt:
		if re, err := regexp.Compile(input); err == nil {
			rc.Regexp = re
			find(rc, true)
		} else {
			rc.Status = err.Error()
			break
		}
	case openPrompt:
		rc.Pane.Delete(edit.Index{1, 0}, rc.Pane.End())
		if contents, err := ioutil.ReadFile(input); err == nil {
			rc.Pane.Insert(edit.Index{1, 0}, string(contents))
			penult := rc.Pane.ShiftIndex(rc.Pane.End(), -1)
			if rc.Pane.Get(penult, rc.Pane.End()) == "\n" {
				rc.Pane.Delete(penult, rc.Pane.End())
			}
			rc.Status = fmt.Sprintf(`Opened "%s".`, input)
		} else {
			rc.Status = fmt.Sprintf(`New file: "%s".`, input)
		}
		rc.Pane.Mark(edit.Index{1, 0}, insertMark)
		rc.Pane.Mark(edit.Index{1, 0}, selMark)
		rc.Pane.Title = input
		rc.Window.SetTitle(input)
		rc.Pane.SetSyntax()
		rc.Pane.ResetModified()
		rc.Pane.ResetUndo()
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
	case saveAsPrompt:
		prevTitle := rc.Pane.Title
		rc.Pane.Title = input
		if err := saveFile(rc.Pane); err == nil {
			rc.Status = fmt.Sprintf(`Saved "%s".`, input)
			rc.Window.SetTitle(input)
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
			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				index := rc.Focus.IndexFromMark(insertMark)
				if sel := rc.Focus.IndexFromMark(selMark); sel != index {
					rc.Focus.Delete(order(sel, index))
				} else {
					rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
				}
			case sdl.K_DELETE:
				index := rc.Focus.IndexFromMark(insertMark)
				if sel := rc.Focus.IndexFromMark(selMark); sel != index {
					rc.Focus.Delete(order(sel, index))
				} else {
					rc.Focus.Delete(index, rc.Focus.ShiftIndex(index, 1))
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
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_LEFT:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(rc.Focus.ShiftIndex(index, -1), insertMark)
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_HOME:
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
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
				index := rc.Focus.IndexFromMark(insertMark)
				rc.Focus.Mark(rc.Focus.ShiftIndex(index, 1), insertMark)
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
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
					sel := rc.Focus.IndexFromMark(selMark)
					insert := rc.Focus.IndexFromMark(insertMark)
					sdl.SetClipboardText(rc.Focus.Get(order(sel, insert)))
				}
			case sdl.K_e:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
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
					if rc.Pane.Modified() {
						rc.Prompt(reallyOpenPrompt)
					} else {
						rc.Prompt(openPrompt)
					}
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
				rc.Pane.See(insertMark)
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
