TODO
====

0.3.0
-----
- Fix the window expose issue
- Look into pipe command text not coming through
- Plugin help
	- Used for displaying things like method signatures in the status line
	- Key binding - Ctrl+K ("keyword lookup")
	- How can the plugins get enough information, without passing the entire
	  buffer to them? Maybe the entire buffer should be passed on stdin, and
	  the cursor position can be passed as command-line arguments.
	- Maybe this is a bad idea.
- Repeat last edit sequence (like . in Vim)
- Syntax highlighting
	- Tcl
	- Markdown
	- ReStructured Text
- Use as filter (-filter flag?)
- In-editor help?
- Set options dynamically?

0.4.0
-----
- Terminal-ish support (interact with launched processes)
- Edit remote files
- Custom color schemes?
