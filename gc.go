package main

import "runtime/debug"

// In order to pass data to the main event loop from a goroutine, a pointer to
// the data is passed through a sdl.UserEvent. Between pushing the event onto
// the event queue and receiving the event, the memory that is pointed to could
// be garbage collected, since no references to it exist. Therefore, to avoid
// segfaults, the sender must call disableGC() before sending, and the receiver
// must call enableGC() after receiving.

// gcDisables and initialGCPercent must not be read or written without
// receiving from gcMutex first.
var gcMutex = make(chan int, 1)
var gcDisables, initialGCPercent int

func init() {
	gcMutex <- 1
}

// disableGC disables garbage collection until enableGC is called once for each
// call to disableGC.
func disableGC() {
	<-gcMutex
	if gcDisables == 0 {
		initialGCPercent = debug.SetGCPercent(-1)
	}
	gcDisables++
	gcMutex <- 1
}

// enableGC negates a call to disableGC, re-enabling garbage collection if all
// calls have been negated.
func enableGC() {
	<-gcMutex
	if gcDisables > 0 {
		gcDisables--
		if gcDisables == 0 {
			debug.SetGCPercent(initialGCPercent)
		}
	}
	gcMutex <- 1
}
