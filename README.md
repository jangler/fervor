Fervor
======
A graphical cross-platform text editor (alpha status).

![screenshot](http://jangler.info/dl/2015-06-08-195034_fervor.png 'screenshot')

Installation
------------
	go get -u github.com/jangler/fervor

Usage
-----
	Usage: fervor [<file>]

Key bindings
------------
	Ctrl+A        Home
	Ctrl+C        Copy
	Ctrl+E        End
	Ctrl+F        Find regexp...
	Ctrl+Shift+F  Reverse find regexp...
	Ctrl+H        Backspace
	Ctrl+N        Next match
	Ctrl+Shift+N  Previous match
	Ctrl+O        Open...
	Ctrl+Q        Quit
	Ctrl+Shift+Q  Force quit
	Ctrl+S        Save
	Ctrl+Shift+S  Save as...
	Ctrl+U        Delete line backward
	Ctrl+V        Paste
	Ctrl+W        Delete word backward
	Ctrl+X        Cut
	Ctrl+Y        Redo
	Ctrl+Z        Undo

Holding Shift makes any cursor motion select text from the previous cursor
position to the resulting position. The Backspace, Delete, Home, End, PgUp, and
PgDn keys should work as expected.

Mouse bindings
--------------
	Left click   Position cursor
	Left drag    Select text
	Right click  Find next instance of clicked word or selection
	Right drag   Find next instance of selection

Holding Shift makes a left click select text from the previous cursor position
to the clicked position, and makes a right click or drag search backward
instead of forward.
