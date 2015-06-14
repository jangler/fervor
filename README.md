Fervor
======

![screenshot](http://jangler.info/dl/fervor-screenshot.png 'screenshot')

A graphical, modeless, cross-platform text editor influenced by Acme, vi, and
even Notepad. Features include:

- Infinite undo
- Unicode (UTF-8) support
- Regular expression search
- Acme-like right-click find
- No GUI toolkit dependencies
- Quick startup and low memory footprint
- Asynchronous shell command execution and selection filtering
- Syntax highlighting (currently for: C, Go, JSON, Make, Python, and Bash)

Not included (yet):

- Support for Windows-style (CRLF) line endings
- Any form of configuration
- A scrollbar

Not included (ever):

- Tabs or panes. Use a good window manager instead!

This project is in alpha status--use at your own risk!

Installation
------------
Install via the [go command](http://golang.org/cmd/go/):

	go get -u github.com/jangler/fervor

The SDL2 and SDL2\_ttf libraries are required.

Usage
-----
	Usage: fervor [<option> ...] [<file>]

	Options:
	  -version=false: print version information and exit

Key bindings
------------
	Ctrl+A           Move cursor to beginning of line
	Ctrl+C           Copy (in buffer), cancel (in prompt)
	Ctrl+D           Change directory...
	Ctrl+E           Move cursor to end of line
	Ctrl+F           Find regexp forward...
	Ctrl+Shift+F     Find regexp backward...
	Ctrl+G           Go to line...
	Ctrl+H           Delete character backward
	Ctrl+N           Next match
	Ctrl+Shift+N     Previous match
	Ctrl+O           Open...
	Ctrl+Shift+O     Open in new window...
	Ctrl+P           Pipe selection through command...
	Ctrl+Q           Quit
	Ctrl+Shift+Q     Quit without confirmation
	Ctrl+R           Run command...
	Ctrl+Shift+R     Reload font (fixes missing glyphs)
	Ctrl+S           Save
	Ctrl+Shift+S     Save as...
	Ctrl+U           Delete line backward
	Ctrl+V           Paste
	Ctrl+W           Delete word backward
	Ctrl+X           Cut
	Ctrl+Y           Redo
	Ctrl+Z           Undo
	Tab (in prompt)  Complete command name or file path

Holding Shift makes a cursor motion select text from the previous cursor
position to the resulting position. Enter, Backspace, Delete, Home, End, PgUp,
PgDn, Up, Down, Left, Right, Esc, Ctrl+Backspace, Ctrl+Delete, Ctrl+Home,
Ctrl+End, Ctrl+Left, and Ctrl+Right should also work as expected.

Mouse bindings
--------------
	Left click   Position cursor
	Left drag    Select text
	Right click  Find next instance of clicked word or selection
	Right drag   Find next instance of selection

Holding Shift makes a left click select text from the previous cursor position
to the clicked position, and makes a right click or drag search backward
instead of forward.
