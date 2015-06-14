package main

import (
	"regexp"

	"github.com/jangler/edit"
)

var (
	spaceRegexp = regexp.MustCompile(`\s`)
	wordRegexp  = regexp.MustCompile(`\w`)
)

// find attempts a regexp search in the buffer and returns a status message.
func find(b *edit.Buffer, re *regexp.Regexp, forward bool,
	defaultStatus string) string {
	if re == nil {
		return "No pattern to find."
	}

	selIndex := b.IndexFromMark(selMark)
	insIndex := b.IndexFromMark(insMark)

	if forward {
		_, index := order(selIndex, insIndex)
		text := b.Get(index, b.End())
		if loc := re.FindStringIndex(text); loc != nil {
			b.Mark(b.ShiftIndex(index, loc[0]), selMark)
			b.Mark(b.ShiftIndex(index, loc[1]), insMark)
			b.Separate()
		} else {
			return "No forward match."
		}
	} else {
		index, _ := order(selIndex, insIndex)
		text := b.Get(edit.Index{1, 0}, index)
		if locs := re.FindAllStringIndex(text, -1); locs != nil {
			loc := locs[len(locs)-1]
			b.Mark(b.ShiftIndex(edit.Index{1, 0}, loc[0]), selMark)
			b.Mark(b.ShiftIndex(edit.Index{1, 0}, loc[1]), insMark)
			b.Separate()
		} else {
			return "No backward match."
		}
	}

	return defaultStatus
}

// getSelection returns the selected text in the buffer.
func getSelection(b *edit.Buffer) string {
	return b.Get(order(b.IndexFromMark(selMark), b.IndexFromMark(insMark)))
}

// order returns index1 and index2 in buffer order.
func order(index1, index2 edit.Index) (first, second edit.Index) {
	if index1.Less(index2) {
		return index1, index2
	}
	return index2, index1
}

// select line changes the selection to the entirety of a line, sans leading
// whitespace.
func selectLine(b *edit.Buffer, line int) {
	selIndex := edit.Index{line, 0}
	for spaceRegexp.MatchString(b.Get(selIndex,
		edit.Index{line, selIndex.Char + 1})) {
		selIndex.Char++
	}
	b.Mark(selIndex, selMark)
	b.Mark(edit.Index{line, 1 << 30}, insMark)
}

// selectWord selects the word at the given index in the buffer.
func selectWord(b *edit.Buffer, index edit.Index) {
	selIndex, insIndex := index, index
	for wordRegexp.MatchString(b.Get(
		edit.Index{selIndex.Line, selIndex.Char - 1}, selIndex)) {
		selIndex.Char -= 1
	}
	for wordRegexp.MatchString(b.Get(
		insIndex, edit.Index{insIndex.Line, insIndex.Char + 1})) {
		insIndex.Char += 1
	}
	b.Mark(selIndex, selMark)
	b.Mark(insIndex, insMark)
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
