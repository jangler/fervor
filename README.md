Fervor
======
A graphical cross-platform text editor (alpha status).

![screenshot](http://jangler.info/dl/2015-06-08-195034_fervor.png 'screenshot')

Installation
------------
Install via the [go command](http://golang.org/cmd/go/):

	go get -u github.com/jangler/fervor

The SDL2 and SDL2\_ttf libraries are required.

Usage
-----
	Usage: fervor [<file>]

Key bindings
------------
	Ctrl+A           Move cursor to beginning of line
	Ctrl+C           Copy
	Ctrl+D           Cancel prompt
	Ctrl+E           Move cursor to end of line
	Ctrl+F           Find regexp forward...
	Ctrl+Shift+F     Find regexp backward...
	Ctrl+H           Delete character backward
	Ctrl+N           Next match
	Ctrl+Shift+N     Previous match
	Ctrl+O           Open...
	Ctrl+Q           Quit
	Ctrl+Shift+Q     Force quit
	Ctrl+S           Save
	Ctrl+Shift+S     Save as...
	Ctrl+U           Delete line backward
	Ctrl+V           Paste
	Ctrl+W           Delete word backward
	Ctrl+X           Cut
	Ctrl+Y           Redo
	Ctrl+Z           Undo
	Ctrl+End         Move cursor to end of buffer
	Ctrl+Home        Move cursor to beginning of buffer
	Tab (in prompt)  Expand environment variables

Holding Shift makes any cursor motion select text from the previous cursor
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
