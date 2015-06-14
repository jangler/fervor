package main

import (
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

var shellName, shellOpt string

func init() {
	rand.Seed(time.Now().UnixNano())

	if runtime.GOOS == "windows" {
		shellName, shellOpt = "cmd", "/c"
	} else {
		shellName, shellOpt = "/bin/sh", "-c"
	}
}

// newInstance opens a new instane of Fervor editing the given filename and
// returns a status message.
func newInstance(filename, defaultStatus string) string {
	cmd := exec.Command(os.Args[0], filename)
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
			exec.Command(os.Args[0], path).Run()
			defer os.Remove(path)
		}
	}()
}
