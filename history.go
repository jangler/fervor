package main

import "container/list"

// history represents a command history.
type history struct {
	entries *list.List // list of string
	current *list.Element
}

// appendString adds s to the list of history entries if s is not already the
// last entry.
func (h *history) appendString(s string) {
	if h.entries == nil {
		h.entries = list.New()
	}
	h.current = nil
	if e := h.entries.Back(); s == "" || (e != nil && e.Value.(string) == s) {
		return
	}
	h.entries.PushBack(s)
}

// prev returns the previous history entry, or s if there is no such entry.
func (h *history) prev(s string) string {
	if h.entries == nil {
		h.entries = list.New()
	}
	if h.current == nil {
		current := h.entries.Back()
		h.appendString(s)
		h.current = current
	} else if h.current.Prev() != nil {
		h.current = h.current.Prev()
	}
	if h.current != nil {
		s = h.current.Value.(string)
	}
	return s
}

// next returns the next history entry, or an empty string if there is no such
// entry.
func (h *history) next() string {
	if h.entries == nil {
		h.entries = list.New()
	}
	if h.current == nil {
		return ""
	} else {
		h.current = h.current.Next()
		if h.current == nil {
			return ""
		}
	}
	if h.current != nil {
		return h.current.Value.(string)
	}
	return ""
}
