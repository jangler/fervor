package main

import (
	"os"
	"path/filepath"
	"strings"
)

// completePath completes the typed path if there is exactly one match in the
// directory.
func completePath(path string) string {
	// read filenames from dir
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	dir, file := filepath.Split(absPath)
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
	var match string
	for _, name := range names {
		if strings.HasPrefix(name, file) {
			if match == "" {
				match = name
			} else {
				match = ""
				break
			}
		}
	}
	if match != "" {
		path = dir + match
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			return minPath(path) + "/"
		}
	}
	return minPath(path)
}
