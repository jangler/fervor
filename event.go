package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_ttf"
)

// user event types. these are only vars in order to be addressable.
var (
	pipeEvent   = 1
	statusEvent = 2
)

var userEventType uint32 // set at beginning of event loop

// colRowFromXY converts (x, y) coordinates in a window to a row and column.
func colRowFromXY(winHeight, x, y int) (col, row int) {
	ps := paneSpace(winHeight)
	y = y % ps
	x -= padPx - fontWidth/2
	y /= fontHeight
	x /= fontWidth
	return x, y
}

// click processes a left mouse click at the given coordinates.
func click(rc *RenderContext, x, y, times int, shift bool) {
	pane := rc.Pane
	_, winHeight := rc.Window.GetSize()
	x, y = colRowFromXY(winHeight, x, y)

	switch times {
	case 1: // place cursor
		pane.Mark(pane.IndexFromCoords(x, y), insMark)
		if !shift {
			pane.Mark(pane.IndexFromMark(insMark), selMark)
		}
	case 2: // select word
		selectWord(pane.Buffer, pane.IndexFromCoords(x, y))
	case 3: // select line
		index := pane.IndexFromCoords(x, y)
		selectLine(rc.Pane.Buffer, index.Line)
	}
}

// clickFind moves the cursor and selection to the next or previous instance of
// the selected text.
func clickFind(rc *RenderContext, shift bool, x, y int) {
	pane := rc.Pane
	_, winHeight := rc.Window.GetSize()
	x, y = colRowFromXY(winHeight, x, y)

	// get selection
	selIndex := pane.IndexFromMark(selMark)
	insertIndex := pane.IndexFromMark(insMark)
	selIndex, insertIndex = order(selIndex, insertIndex)

	// reposition cursor if click is outside selection
	clickIndex := pane.IndexFromCoords(x, y)
	if clickIndex.Less(selIndex) || insertIndex.Less(clickIndex) {
		pane.Mark(clickIndex, selMark)
		pane.Mark(clickIndex, insMark)
		selIndex = pane.IndexFromMark(selMark)
		insertIndex = pane.IndexFromMark(insMark)
	}

	// select word if selection is nil
	if selIndex == insertIndex {
		selectWord(pane.Buffer, selIndex)
		selIndex = pane.IndexFromMark(selMark)
		insertIndex = pane.IndexFromMark(insMark)
	}
	selection := pane.Get(selIndex, insertIndex)

	if shift { // search backwards
		index := selIndex
		text := pane.Get(edit.Index{1, 0}, index)
		if pos := strings.LastIndex(text, selection); pos >= 0 {
			pane.Mark(pane.ShiftIndex(index, pos-utf8.RuneCountInString(text)),
				selMark)
			pane.Mark(pane.ShiftIndex(pane.IndexFromMark(selMark),
				utf8.RuneCountInString(selection)), insMark)
		} else {
			rc.Status = "No backward match."
		}
	} else { // search forwards
		index := insertIndex
		text := pane.Get(index, pane.End())
		if pos := strings.Index(text, selection); pos >= 0 {
			pane.Mark(pane.ShiftIndex(index, pos), selMark)
			pane.Mark(pane.ShiftIndex(pane.IndexFromMark(selMark),
				utf8.RuneCountInString(selection)), insMark)
		} else {
			rc.Status = "No forward match."
		}
	}
}

// textInput inserts text into the focus, or performs another action depending
// on the contents of the string.
func textInput(buf *edit.Buffer, s string) {
	index := buf.IndexFromMark(insMark)
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
func resize(pane *Pane, width, height int) {
	cols, rows := bufSize(width, height)
	pane.Cols, pane.Rows = cols, rows
	pane.SetSize(cols, rows)
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

// warpMouseToSel warps the mouse to the center of the buffer selection.
func warpMouseToSel(w *sdl.Window, b *edit.Buffer) {
	sel, ins := b.IndexFromMark(selMark), b.IndexFromMark(insMark)
	selCol, selRow := b.CoordsFromIndex(sel)
	insCol, insRow := b.CoordsFromIndex(ins)
	x := (float64(selCol)+float64(insCol))*float64(fontWidth)/2 + padPx
	y := (float64(selRow)+float64(insRow)+1)*float64(fontHeight)/2 + padPx
	w.WarpMouseInWindow(int(x), int(y))
}

// eventLoop handles SDL events until quit is requested.
func eventLoop(pane *Pane, status string, font *ttf.Font, win *sdl.Window) {
	userEventType = sdl.RegisterEvents(1)
	rc := &RenderContext{pane, edit.NewBuffer(), pane.Buffer, status, font,
		win, nil}
	rc.Input.Mark(edit.Index{1, 0}, insMark)
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
			prevIns := rc.Pane.IndexFromMark(insMark)

			switch event.Keysym.Sym {
			case sdl.K_BACKSPACE:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end := rc.Focus.IndexFromMark(insMark)
					begin := shiftIndexByWord(rc.Focus, end, -1)
					rc.Focus.Delete(begin, end)
				} else {
					index := rc.Focus.IndexFromMark(insMark)
					if sel := rc.Focus.IndexFromMark(selMark); sel != index {
						rc.Focus.Delete(order(sel, index))
					} else {
						rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
					}
				}
			case sdl.K_DELETE:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					begin := rc.Focus.IndexFromMark(insMark)
					end := shiftIndexByWord(rc.Focus, begin, 1)
					rc.Focus.Delete(begin, end)
				} else {
					index := rc.Focus.IndexFromMark(insMark)
					if sel := rc.Focus.IndexFromMark(selMark); sel != index {
						rc.Focus.Delete(order(sel, index))
					} else {
						rc.Focus.Delete(index, rc.Focus.ShiftIndex(index, 1))
					}
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.See(insMark)
				}
			case sdl.K_DOWN:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Focus.IndexFromMark(insMark)
					col, row := rc.Focus.CoordsFromIndex(index)
					rc.Focus.Mark(rc.Focus.IndexFromCoords(col, row+1),
						insMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insMark),
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
					rc.Focus.Mark(rc.Focus.End(), insMark)
				} else {
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Mark(edit.Index{index.Line, 1 << 30}, insMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_LEFT:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insMark)
					index = shiftIndexByWord(rc.Focus, index, -1)
					rc.Focus.Mark(index, insMark)
				} else {
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Mark(rc.Focus.ShiftIndex(index, -1), insMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_HOME:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					rc.Focus.Mark(edit.Index{1, 0}, insMark)
				} else {
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Mark(edit.Index{index.Line, 0}, insMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insMark), selMark)
				}
				if rc.Focus == rc.Pane.Buffer {
					rc.Pane.Separate()
				}
			case sdl.K_PAGEDOWN:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Pane.IndexFromMark(insMark)
					col, row := rc.Pane.CoordsFromIndex(index)
					rc.Pane.Mark(rc.Pane.IndexFromCoords(col,
						row+rc.Pane.Rows), insMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insMark),
							selMark)
					}
					rc.Pane.Separate()
				}
			case sdl.K_PAGEUP:
				if rc.Focus == rc.Pane.Buffer {
					index := rc.Pane.IndexFromMark(insMark)
					col, row := rc.Pane.CoordsFromIndex(index)
					rc.Pane.Mark(rc.Pane.IndexFromCoords(col,
						row-rc.Pane.Rows), insMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insMark),
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
					index := rc.Focus.IndexFromMark(insMark)
					index = shiftIndexByWord(rc.Focus, index, 1)
					rc.Focus.Mark(index, insMark)
				} else {
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Mark(rc.Focus.ShiftIndex(index, 1), insMark)
				}
				if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
					rc.Focus.Mark(rc.Focus.IndexFromMark(insMark), selMark)
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
					case pipePrompt, runPrompt:
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
					index := rc.Focus.IndexFromMark(insMark)
					col, row := rc.Focus.CoordsFromIndex(index)
					rc.Focus.Mark(rc.Focus.IndexFromCoords(col, row-1),
						insMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insMark),
							selMark)
					}
					rc.Pane.Separate()
				}
			case sdl.K_a:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Mark(edit.Index{index.Line, 0}, insMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insMark),
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
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Mark(edit.Index{index.Line, 1 << 30}, insMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insMark),
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
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
				}
			case sdl.K_n:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						rc.Status = find(rc.Pane.Buffer, rc.Regexp, false,
							rc.Status)
					} else {
						rc.Status = find(rc.Pane.Buffer, rc.Regexp, true,
							rc.Status)
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
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						rc.Font = getFont()
						rc.Status = "Reloaded font."
					} else if rc.Focus != rc.Input {
						rc.Prompt(runPrompt)
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
					index := rc.Focus.IndexFromMark(insMark)
					rc.Focus.Delete(edit.Index{index.Line, 0}, index)
				}
			case sdl.K_v:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					sel := rc.Focus.IndexFromMark(selMark)
					insert := rc.Focus.IndexFromMark(insMark)
					if sel != insert {
						rc.Focus.Delete(order(sel, insert))
						insert, _ = order(sel, insert)
					}
					rc.Focus.Insert(insert, sdl.GetClipboardText())
				}
			case sdl.K_w:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					end := rc.Focus.IndexFromMark(insMark)
					begin := shiftIndexByWord(rc.Focus, end, -1)
					rc.Focus.Delete(begin, end)
				}
			case sdl.K_x:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					sel := rc.Focus.IndexFromMark(selMark)
					insert := rc.Focus.IndexFromMark(insMark)
					sdl.SetClipboardText(rc.Focus.Get(order(sel, insert)))
					rc.Focus.Delete(order(sel, insert))
				}

			case sdl.K_y:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if !rc.Pane.Redo(selMark, insMark) {
						rc.Status = "Nothing to redo."
					}
				}
			case sdl.K_z:
				if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
					if event.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
						if !rc.Pane.Redo(selMark, insMark) {
							rc.Status = "Nothing to redo."
						}
					} else {
						if !rc.Pane.Undo(selMark, insMark) {
							rc.Status = "Nothing to undo."
						}
					}
				}
			default:
				recognized = false
			}
			if recognized {
				if prevSel != rc.Pane.IndexFromMark(selMark) ||
					prevIns != rc.Pane.IndexFromMark(insMark) {
					rc.Pane.See(insMark)
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
					_, winHeight := rc.Window.GetSize()
					x, y := colRowFromXY(winHeight, int(event.X), int(event.Y))
					rightClickIndex = rc.Pane.IndexFromCoords(x, y)
				}
			} else if event.Type == sdl.MOUSEBUTTONUP &&
				event.Button == sdl.BUTTON_RIGHT {
				clickFind(rc, shift, int(event.X), int(event.Y))
				rc.Pane.See(insMark)
				warpMouseToSel(rc.Window, rc.Pane.Buffer)
				render(rc)
			}
			rc.Pane.Separate()
		case *sdl.MouseMotionEvent:
			if event.State&sdl.ButtonLMask() != 0 {
				click(rc, int(event.X), int(event.Y), 1, true)
				render(rc)
			} else if event.State&sdl.ButtonRMask() != 0 {
				_, winHeight := rc.Window.GetSize()
				x, y := colRowFromXY(winHeight, int(event.X), int(event.Y))
				index := rc.Pane.IndexFromCoords(x, y)
				if index != rightClickIndex {
					rc.Pane.Mark(rightClickIndex, selMark)
					rc.Pane.Mark(index, insMark)
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
					rc.Pane.See(insMark)
				}
				render(rc)
			}
		case *sdl.UserEvent:
			switch *(*int)(event.Data1) {
			case pipeEvent:
				sel := rc.Focus.IndexFromMark(selMark)
				insert := rc.Focus.IndexFromMark(insMark)
				rc.Pane.Delete(order(sel, insert))
				insert, _ = order(sel, insert)
				rc.Pane.Insert(insert, *(*string)(event.Data2))
				rc.Pane.See(insMark)
				render(rc)
			case statusEvent:
				if rc.Focus != rc.Input {
					rc.Status = *(*string)(event.Data2)
					render(rc)
				}
			}
			enableGC()
		case *sdl.WindowEvent:
			switch event.Event {
			case sdl.WINDOWEVENT_EXPOSED, sdl.WINDOWEVENT_SHOWN:
				win.UpdateSurface()
			case sdl.WINDOWEVENT_RESIZED:
				resize(rc.Pane, int(event.Data1), int(event.Data2))
				render(rc)
			}
		}
	}
}
