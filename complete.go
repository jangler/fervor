package main

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

var pathSep string // separates entries in PATH environment variable

func init() {
	if runtime.GOOS == "windows" {
		pathSep = ";"
	} else {
		pathSep = ":"
	}
}

// completeCmd completes the typed command name if there is exactly one match
// in $PATH.
func completeCmd(cmd string) string {
	paths := strings.Split(os.Getenv("PATH"), pathSep)
	var match string
	for _, path := range paths {
		if f, err := os.Open(path); err == nil {
			if names, err := f.Readdirnames(0); err == nil {
				for _, name := range names {
					if strings.HasPrefix(name, cmd) {
						if match == "" || match == name { // links are OK
							match = name
						} else {
							return cmd
						}
					}
				}
			}
			f.Close()
		}
	}
	if match != "" {
		return match
	}
	return cmd
}

// completePath completes the typed path if there is exactly one match in the
// directory.
func completePath(path string, dirsOnly bool) string {
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
		if strings.HasPrefix(name, file) &&
			(!dirsOnly || isDir(filepath.Join(dir, name))) {
			if match == "" {
				match = name
			} else {
				return minPath(path)
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
	} else {
		return false
	}
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
