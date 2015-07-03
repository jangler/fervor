package main

import (
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jangler/edit"
)

var nonWordRegexp = regexp.MustCompile(`\W`)

// commonPrefix returns the longest common prefix of two strings.
func commonPrefix(a, b string) string {
	var prefix string
	if len(b) < len(a) {
		a, b = b, a
	}
	for i := len(a); i >= 0; i-- {
		if a[:i] == b[:i] {
			prefix = a[:i]
			break
		}
	}
	return prefix
}

// completeCmd completes the typed command name to the longest common prefix of
// matches in $PATH.
func completeCmd(cmd string) string {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	var prefix string
	for _, path := range paths {
		if f, err := os.Open(path); err == nil {
			if names, err := f.Readdirnames(0); err == nil {
				for _, name := range names {
					if strings.HasPrefix(name, cmd) {
						if prefix == "" {
							prefix = name
						} else {
							prefix = commonPrefix(prefix, name)
							if prefix == "" {
								return cmd
							}
						}
					}
				}
			}
			f.Close()
		}
	}
	if prefix == "" {
		return cmd
	}
	return prefix
}

// completePath completes the typed path to the longest common prefix of paths
// in the directory.
func completePath(path string, dirsOnly bool) string {
	// read filenames from dir
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	var dir, file string
	if strings.HasSuffix(path, "/") {
		dir = absPath
	} else {
		dir, file = filepath.Split(absPath)
	}
	f, err := os.Open(dir)
	if err != nil {
		return path
	}
	names, err := f.Readdirnames(0)
	f.Close()
	if err != nil {
		return path
	}

	// return a match if there is exactly one
	var prefix string
	for _, name := range names {
		if strings.HasPrefix(name, file) &&
			(!dirsOnly || isDir(filepath.Join(dir, name))) {
			if prefix == "" {
				prefix = name
			} else {
				if prefix = commonPrefix(prefix, name); prefix == "" {
					break
				}
			}
		}
	}
	if prefix != "" {
		path = filepath.Join(dir, prefix)
	}
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		return minPath(path) + "/"
	}
	return minPath(path)
}

// completeWord completes the typed word to the first match in the buffer.
func completeWord(b *edit.Buffer, prefix string, forward bool) string {
	endLine := b.End().Line
	startLine := b.IndexFromMark(insMark).Line
	line := startLine
	for {
		// search for completion in line
		s := b.Get(edit.Index{line, 0}, edit.Index{line, 1 << 30})
		words := nonWordRegexp.Split(s, -1)
		if forward {
			for _, word := range words {
				if strings.HasPrefix(word, prefix) && word != prefix {
					return word
				}
			}
		} else {
			for i := len(words) - 1; i >= 0; i-- {
				if strings.HasPrefix(words[i], prefix) && words[i] != prefix {
					return words[i]
				}
			}
		}

		// go to next line
		if forward {
			line++
			if line > endLine {
				line = 0
			}
		} else {
			line--
			if line < 1 {
				line = endLine
			}
		}
		if line == startLine { // search failed
			break
		}
	}
	return ""
}

// expandVars returns a version of path with environment variables and ~/
// expanded.
func expandVars(path string) string {
	path = os.ExpandEnv(path)
	if curUser, err := user.Current(); err == nil {
		path = strings.Replace(path, "~/", curUser.HomeDir+"/", -1)
	}
	return path
}

// isDir returns true if and only if the given path represents a directory.
func isDir(path string) bool {
	if fi, err := os.Stat(path); err == nil {
		return fi.IsDir()
	}
	return false
}

// minPath returns the shortest valid representation of the given file path.
func minPath(path string) string {
	// try absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	// try path relative to working directory
	if wd, err := os.Getwd(); err == nil {
		if relWd, err := filepath.Rel(wd, abs); err == nil {
			if utf8.RuneCountInString(relWd) < utf8.RuneCountInString(path) {
				path = relWd
			}
		}
	}

	// try path relative to home directory
	if curUser, err := user.Current(); err == nil {
		if relHome, err := filepath.Rel(curUser.HomeDir, abs); err == nil {
			relHome = "~/" + relHome
			if utf8.RuneCountInString(relHome) < utf8.RuneCountInString(path) {
				path = relHome
			}
		}
	}

	return filepath.Clean(path)
}
