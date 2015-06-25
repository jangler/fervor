package main

import (
	"encoding/base32"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/jangler/edit"
	"github.com/veandco/go-sdl2/sdl"
)

var shellName, shellOpt string // command interpreter invocation

func init() {
	rand.Seed(time.Now().UnixNano())

	if runtime.GOOS == "windows" {
		shellName, shellOpt = "cmd", "/c"
	} else {
		shellName, shellOpt = "/bin/sh", "-c"
	}
}

// flags returns a []string of command line flags that have been set in a form
// that can be passed as arguments to exec.Command().
func flags() []string {
	args := make([]string, flag.NFlag())
	i := 0
	flag.Visit(func(f *flag.Flag) {
		args[i] = fmt.Sprintf("-%s=%v", f.Name, f.Value)
		i++
	})
	return args
}

// newInstance opens a new instane of Fervor editing the given filename and
// returns a status message.
func newInstance(filename, defaultStatus string) string {
	cmd := exec.Command(os.Args[0], append(flags(), filename)...)
	if err := cmd.Start(); err != nil {
		return err.Error()
	}
	return defaultStatus
}

// reportExitStatus pushes a status message to the SDL event queue depending
// on err (which may be nil).
func reportExitStatus(cmd string, err error) {
	var event sdl.UserEvent
	var msg string
	if err == nil {
		msg = fmt.Sprintf(`Command "%s" exited successfully.`, cmd)
	} else {
		msg = fmt.Sprintf(`Command "%s" exited with error: %v`, cmd, err)
	}
	event.Type, event.Data1 = userEventType, unsafe.Pointer(&statusEvent)
	disableGC()
	event.Data2 = unsafe.Pointer(&msg)
	sdl.PushEvent(&event)
}

// keywordLookup runs the current keyword program with the buffer contents
// written to standard input and cursor position passed as two command-line
// arguments. Status events are pushed to the event queue on completion.
func keywordLookup(b *edit.Buffer, defaultStatus string) string {
	// TODO: Refactor this. It's very similar to pipeCmd().

	// initialize command
	ins := b.IndexFromMark(insMark)
	cmd := exec.Command(kwprogFlag, fmt.Sprintf("%d", ins.Line),
		fmt.Sprintf("%d", ins.Char))
	inPipe, err := cmd.StdinPipe()
	if err != nil {
		return err.Error()
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err.Error()
	}
	if err := cmd.Start(); err != nil {
		return err.Error()
	}

	go func() {
		// write to stdin and read from stdout
		go func() {
			if _, err := io.WriteString(inPipe,
				b.Get(edit.Index{1, 0}, b.End())); err != nil {
				log.Print(err)
			}
			inPipe.Close()
		}()
		var outBytes []byte
		go func() {
			if outBytes, err = ioutil.ReadAll(outPipe); err != nil {
				log.Print(err)
			}
		}()

		if err := cmd.Wait(); err != nil {
			reportExitStatus(kwprogFlag, err)
		} else if outBytes != nil {
			// strip trailing newline
			if len(outBytes) > 0 && outBytes[len(outBytes)-1] == '\n' {
				outBytes = outBytes[:len(outBytes)-1]
			}

			// push status event
			output := string(outBytes)
			var event sdl.UserEvent
			event.Type = userEventType
			event.Data1 = unsafe.Pointer(&statusEvent)
			disableGC()
			event.Data2 = unsafe.Pointer(&output)
			sdl.PushEvent(&event)
		}
	}()

	return defaultStatus
}

// pipeCmd pipes selection through cmdString asynchronously and returns a
// status message. Results are returned on the SDL event queue.
func pipeCmd(cmdString, selection, defaultStatus string) string {
	// initialize command
	cmd := exec.Command(shellName, shellOpt, cmdString)
	inPipe, err := cmd.StdinPipe()
	if err != nil {
		return err.Error()
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err.Error()
	}
	if err := cmd.Start(); err != nil {
		return err.Error()
	}

	go func() {
		// write to stdin and read from stdout
		go func() {
			if _, err := io.WriteString(inPipe, selection); err != nil {
				log.Print(err)
			}
			inPipe.Close()
		}()
		var outBytes []byte
		go func() {
			if outBytes, err = ioutil.ReadAll(outPipe); err != nil {
				log.Print(err)
			}
		}()

		reportExitStatus(cmdString, cmd.Wait())

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
			disableGC()
			event.Data2 = unsafe.Pointer(&output)
			sdl.PushEvent(&event)
		}
	}()

	return defaultStatus
}

// runCmd executes cmdString asynchronously. Results are returned on the SDL
// event queue.
func runCmd(cmdString string) {
	cmd := exec.Command(shellName, shellOpt, cmdString)

	go func() {
		output, err := cmd.CombinedOutput()
		reportExitStatus(cmdString, err)

		if output != nil && len(output) > 0 {
			// generate a random filename
			src := make([]byte, 8)
			for i := range src {
				src[i] = byte(rand.Intn(256))
			}
			name := base32.StdEncoding.EncodeToString(src)
			name = strings.TrimRight(name, "=")

			// write command output to temp file
			path := filepath.Join(os.TempDir(), name)
			file, err := os.Create(path)
			if err != nil {
				log.Print(err)
				return
			}
			file.Write(output)
			file.Close()

			// open temp file in new window, then clean up
			exec.Command(os.Args[0], append(flags(), path)...).Run()
			defer os.Remove(path)
		}
	}()
}
