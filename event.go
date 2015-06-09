package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"time"

	"code.google.com/p/jamslam-freetype-go/freetype/truetype"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/jangler/edit"
)

const (
	openPrompt       = "Open: "
	reallyOpenPrompt = "Really open (y/n)? "
	reallyQuitPrompt = "Really quit (y/n)? "
	saveAsPrompt     = "Save as: "
)

var (
	spaceRegexp = regexp.MustCompile(`\s`) // matches whitespace characters
	wordRegexp  = regexp.MustCompile(`\w`) // matches word characters
)

// click processes a left mouse click at the given coordinates.
func click(pane *Pane, win *xwindow.Window, font *truetype.Font, x, y,
	times int, shift bool) {
	height := win.Geom.Height()
	ps := paneSpace(height, 1, font)
	y = y % ps
	x -= padPx - fontWidth/2
	y /= fontHeight
	x /= fontWidth
	switch times {
	case 1: // place cursor
		pane.Mark(pane.IndexFromCoords(x, y), insertMark)
		if !shift {
			pane.Mark(pane.IndexFromMark(insertMark), selMark)
		}
	case 2: // select word
		selIndex := pane.IndexFromCoords(x, y)
		insertIndex := selIndex
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
	case 3: // select line
		index := pane.IndexFromCoords(x, y)
		pane.Mark(edit.Index{index.Line, 0}, selMark)
		pane.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
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
func resize(pane *Pane, font *truetype.Font, width, height int) {
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
	text := pane.Get(edit.Index{1, 0}, pane.End()) + "\n"
	err := ioutil.WriteFile(pane.Title, []byte(text), 0664)
	if err == nil {
		pane.ResetModified()
	}
	return err
}

// Prompt enters into prompt mode, prompting for input with the given string.
func (rc *RenderContext) Prompt(s string) {
	rc.Status = s
	rc.Focus = rc.Input
	rc.Input.Delete(edit.Index{1, 0}, rc.Input.End())
}

// EnterInput exits prompt mode, taking action based on the prompt string and
// input text. Returns false if the application should quit.
func (rc *RenderContext) EnterInput() bool {
	input := rc.Input.Get(edit.Index{1, 0}, rc.Input.End())
	switch rc.Status {
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
		rc.Pane.SetSyntax()
		rc.Pane.ResetModified()
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
			ewmh.WmNameSet(rc.Window.X, rc.Window.Id, input)
			icccm.WmNameSet(rc.Window.X, rc.Window.Id, input)
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

// eventLoop handles SDL events until quit is requested.
func eventLoop(pane *Pane, status string, font *truetype.Font,
	win *xwindow.Window) {
	ewmh.WmNameSet(win.X, win.Id, pane.Title)
	icccm.WmNameSet(win.X, win.Id, pane.Title)
	rc := &RenderContext{pane, edit.NewBuffer(), pane.Buffer, status, font,
		win}
	rc.Input.Mark(edit.Index{1, 0}, insertMark)
	rc.Input.Mark(edit.Index{1, 0}, selMark)
	render(rc)
	w, h := win.Geom.Width(), win.Geom.Height()
	win.Resize(w, h)
	clickCount := 0
	lastClick := time.Now()
	var shift bool

	wmDeleteWindow := make(chan int, 1)
	win.WMGracefulClose(
		func(w *xwindow.Window) {
			wmDeleteWindow <- 1
		})

	win.Listen(xproto.EventMaskKeyPress | xproto.EventMaskKeyRelease |
		xproto.EventMaskButtonPress | xproto.EventMaskExposure |
		xproto.EventMaskStructureNotify)
	xevent.KeyPressFun(
		func(xu *xgbutil.XUtil, event xevent.KeyPressEvent) {
			log.Print("key press", event)
			//render(rc)
		}).Connect(win.X, win.Id)
	xevent.KeyReleaseFun(
		func(xu *xgbutil.XUtil, event xevent.KeyReleaseEvent) {
			log.Print("key release", event)
			//render(rc)
		}).Connect(win.X, win.Id)
	xevent.ButtonPressFun(
		func(xu *xgbutil.XUtil, event xevent.ButtonPressEvent) {
			switch event.Detail {
			case 1: // left click
				if time.Since(lastClick) < time.Second/4 {
					clickCount = clickCount%3 + 1
				} else {
					clickCount = 1
				}
				lastClick = time.Now()
				click(rc.Pane, win, font, int(event.EventX), int(event.EventY),
					clickCount, shift)
				render(rc)
			case 4: // mouse wheel up
				rc.Pane.Scroll(-3)
				render(rc)
			case 5: // mouse wheel down
				rc.Pane.Scroll(3)
				render(rc)
			}
		}).Connect(win.X, win.Id)
	xevent.ExposeFun(
		func(xu *xgbutil.XUtil, event xevent.ExposeEvent) {
			expose(rc)
		}).Connect(win.X, win.Id)
	xevent.ConfigureNotifyFun(
		func(xu *xgbutil.XUtil, event xevent.ConfigureNotifyEvent) {
			w, h := rc.Window.Geom.Width(), rc.Window.Geom.Height()
			rc.Window.Geometry()
			if w != rc.Window.Geom.Width() || h != rc.Window.Geom.Height() {
				resize(rc.Pane, font, rc.Window.Geom.Width(),
					rc.Window.Geom.Height())
				render(rc)
			}
		}).Connect(win.X, win.Id)
	pingBefore, pingAfter, pingQuit := xevent.MainPing(win.X)

	for {
		select {
		case <-wmDeleteWindow:
			return
		case <-pingBefore:
			<-pingAfter
		case <-pingQuit:
			return
		}
	}
	/*
		for {
			xevent.Read(win.X, true)
		}
			for !xevent.Empty(win.X) {
				if event, err := xevent.Dequeue(win.X); event != nil {
					switch event.(type) {
					case xproto.KeyPressEvent:
						if rc.Focus == rc.Pane.Buffer {
							rc.Status = rc.Pane.Title
						}
						recognized := true
					}
				} else {
					log.Print(err)
				}
			}
			switch event := sdl.WaitEvent().(type) {
			case *sdl.KeyDownEvent:
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
					}
				case sdl.K_END:
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 2 << 30}, insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
					}
				case sdl.K_LEFT:
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(rc.Focus.ShiftIndex(index, -1), insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
					}
				case sdl.K_HOME:
					index := rc.Focus.IndexFromMark(insertMark)
					rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
					if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
						rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark), selMark)
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
					}
				case sdl.K_a:
					if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
						index := rc.Focus.IndexFromMark(insertMark)
						rc.Focus.Mark(edit.Index{index.Line, 0}, insertMark)
						if event.Keysym.Mod&sdl.KMOD_SHIFT == 0 {
							rc.Focus.Mark(rc.Focus.IndexFromMark(insertMark),
								selMark)
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
					}
				case sdl.K_h:
					if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
						index := rc.Focus.IndexFromMark(insertMark)
						rc.Focus.Delete(rc.Focus.ShiftIndex(index, -1), index)
					}
				case sdl.K_o:
					if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
						if rc.Focus != rc.Pane.Buffer {
							break
						}
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
					if event.Keysym.Mod&sdl.KMOD_CTRL != 0 {
						if rc.Focus != rc.Pane.Buffer {
							break
						}
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
				default:
					recognized = false
				}
				if recognized {
					rc.Pane.See(insertMark)
					render(rc)
				}
			case *sdl.MouseMotionEvent:
				if event.State == sdl.ButtonLMask() {
					click(rc.Pane, win, font, int(event.X), int(event.Y), 1, true)
					render(rc)
				}
			case *sdl.TextInputEvent:
				if n := bytes.Index(event.Text[:], []byte{0}); n > 0 {
					textInput(rc.Focus, string(event.Text[:n]))
					render(rc)
				}
		}
	*/
}
