package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/jangler/edit"
)

const (
	cdPrompt           = "Change directory to: "
	findBackwardPrompt = "Find backward: "
	findForwardPrompt  = "Find forward: "
	goToLinePrompt     = "Go to line: "
	openNewPrompt      = "Open in new window: "
	openPrompt         = "Open: "
	pipePrompt         = "Pipe selection through: "
	reallyOpenPrompt   = "Really open (y/n)? "
	reallyQuitPrompt   = "Really quit (y/n)? "
	runPrompt          = "Run: "
	saveAsPrompt       = "Save as: "
)

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
			rc.Status = find(rc.Pane.Buffer, rc.Regexp, false, rc.Status)
		} else {
			rc.Status = err.Error()
		}
	case findForwardPrompt:
		if re, err := regexp.Compile(input); err == nil {
			rc.Regexp = re
			rc.Status = find(rc.Pane.Buffer, rc.Regexp, true, rc.Status)
		} else {
			rc.Status = err.Error()
		}
	case goToLinePrompt:
		if n, err := strconv.ParseInt(input, 0, 0); err == nil {
			rc.Status = rc.Pane.Title
			selectLine(rc.Pane.Buffer, int(n))
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
		rc.Pane.Mark(edit.Index{1, 0}, selMark, insMark)
		rc.Pane.Title = minPath(input)
		rc.Window.SetTitle(rc.Pane.Title)
		rc.Pane.SetSyntax()
		rc.Pane.ResetModified()
		rc.Pane.ResetUndo()
	case openNewPrompt:
		rc.Status = newInstance(expandVars(input), rc.Pane.Title)
	case pipePrompt:
		rc.Status = rc.Pane.Title
		if input == "" {
			break
		}
		rc.Status = pipeCmd(input, getSelection(rc.Pane.Buffer), rc.Status)
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
		runCmd(input)
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

// Prompt enters into prompt mode, prompting for input with the given string.
func (rc *RenderContext) Prompt(s string) {
	rc.Input.ResetUndo()
	rc.Pane.Separate()
	rc.Status = s
	rc.Focus = rc.Input
	rc.Input.Delete(edit.Index{1, 0}, rc.Input.End())
}
