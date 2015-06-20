package main

import "github.com/jangler/edit"

func mustCompile(pattern string, id int) edit.Rule {
	rule, err := edit.NewRule(pattern, id)
	if err != nil {
		panic(err)
	}
	return rule
}

const (
	commentId = iota
	keywordId
	literalId
)

var syntaxMap = map[string]func() []edit.Rule{
	"[bash]":       bashRules,
	"[c]":          cRules,
	"[css]":        cssRules,
	"[go]":         goRules,
	"[html]":       htmlRules,
	"[ini]":        iniRules,
	"[javascript]": javaScriptRules,
	"[json]":       jsonRules,
	"[lua]":        luaRules,
	"[make]":       makefileRules,
	"[python]":     pythonRules,
	"[ruby]":       rubyRules,
}

// bashRules returns syntax highlighting rules for Bash.
func bashRules() []edit.Rule {
	// Not sure how "complete" this is. All the builtins are accounted for, but
	// I might check out how other programs syntax highlight bash to see if
	// other things are usually highlighted.
	return []edit.Rule{
		mustCompile(`\$#`, -1),
		mustCompile(`#.*$`, commentId),
		mustCompile(`[!:.]| \[\[? | \]\]?|\b(alias|bg|bind|break|builtin|`+
			`caller|case|cd|command|compgen|complete|compopt|continue|`+
			`declare|dirs|disown|do|done|echo|elif|else|enable|esac|eval|`+
			`exec|exit|export|false|fc|fg|fi|for|function|getopts|hash|help|`+
			`history|if|in|jobs|kill|let|local|logout|mapfile|popd|printf|`+
			`pushd|pwd|read|readarray|readonly|return|select|set|shift|shopt|`+
			`source|suspend|test|then|time|times|trap|true|type|typeset|`+
			`ulimit|umask|unalias|unset|until|wait|while)\b`, keywordId),
		mustCompile(`\$?("(\\.|[^"])*?"|'(\\.|[^'])*?')`, literalId),
		mustCompile("`(\\.|[^`])*?`", literalId),
	}
}

// cRules returns syntax highlighting rules for C.
func cRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`^#(define|undef|include|if|elif|else|endif|ifdef|ifndef|`+
			`line|pragma).*$`, commentId),
		mustCompile(`//.+?$`, commentId),
		mustCompile(`/\*.*?\*/`, commentId),
		mustCompile(`\b(auto|break|case|char|const|continue|default|do|`+
			`double|else|enum|extern|float|for|goto|if|inline|int|long|`+
			`register|restrict|return|short|signed|sizeof|static|struct|`+
			`switch|typedef|union|unsigned|void|volatile|while|_Bool|`+
			`_Complex|_Imaginary)\b`, keywordId),
		mustCompile(`\b(bool|true|false|NULL)\b`, literalId),
		mustCompile(`L?'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eEpP][+-]?\d+)?`+
			`[fFlL]?\b`, literalId),
		mustCompile(`\b(0([bB][01]+|[0-7]+|[xX][0-9a-fA-F]+)|\d+)`+
			`([uU]?[lL]?|[lL]?[uU]?)\b`, literalId),
	}
}

// cssRules returns syntax highlighting rules for CSS.
func cssRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`/\*.*?\*/`, commentId),
		mustCompile(`^\S*[^:] `, keywordId),
		mustCompile(`\S+;`, literalId),
	}
}

// goRules returns syntax highlighting rules for Go.
func goRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`//.+?$`, commentId),
		mustCompile(`/\*.*?\*/`, commentId),
		mustCompile(`\b(break|case|chan|const|continue|default|defer|else|`+
			`fallthrough|for|func|go|goto|if|import|interface|map|package|`+
			`range|return|select|struct|switch|type|var)\b`, keywordId),
		mustCompile(`\b(append|cap|close|complex|copy|delete|imag|len|make|`+
			`new|panic|print|println|real|recover)\b`, keywordId),
		mustCompile(`\b(bool|byte|complex(64|128)|error|float(32|64)|`+
			`u?int(8|16|32|64)?|rune|string|uintptr)\b`, keywordId),
		mustCompile(`\b(true|false|iota|nil)\b`, literalId),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalId),
		mustCompile("`.*?`", literalId),
		mustCompile(`\b0[bB][01]+\b`, literalId),
		mustCompile(`\b0[0-7]+\b`, literalId),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?i?\b`, literalId),
		mustCompile(`\b\d+\bi`, literalId),
	}
}

// htmlRules returns syntax highlighting rules for HTML.
func htmlRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`<!(--.*--|DOCTYPE.*)>`, commentId),
		mustCompile(`\b(a|abbr|address|area|article|aside|audio|b|base|`+
			`bd[io]|blockquote|body|br|button|canvas|caption|cite|code|`+
			`col(group)?|datalist|dd|del|details|dfn|dialog|div|dl|dt|`+
			`em(bed)?|fieldset|fig(caption|ure)|footer|form|h[123456]|`+
			`head|header|hr|html|i|iframe|img|input|ins|kbd|keygen|label|`+
			`legend|li|link|main|map|mark|menu(item)?|meta|meter|nav|`+
			`noscript|object|ol|opt(group|tion)|output|p|param|pre|progress|`+
			`q|rp|rt|ruby|s|samp|script|section|select|small|source|span|`+
			`strong|style|sub|summary|sup|table|tbody|td|textarea|tfoot|th|`+
			`thead|time|title|tr|track|u|ul|var|video|wbr)\b`, keywordId),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalId),
	}
}

// iniRules returns syntax highlighting rules for INI files.
func iniRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`(^| )[;#].*$`, commentId),
		mustCompile(`^\[.*\]$`, keywordId),
	}
}

// javaScriptRules returns syntax highlighting rules for JavaScript.
func javaScriptRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`//.+?$`, commentId),
		mustCompile(`/\*.*?\*/`, commentId),
		mustCompile(`\b(break|case|catch|continue|debugger|default|delete|`+
			`do|else|finally|for|function|if|in|instanceof|new|return|switch|`+
			`this|throw|try|typeof|var|void|while|with)\b`, keywordId),
		mustCompile(`\b(false|NaN|null|true|undefined)\b`, literalId),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([Ee][+-]?\d+)?\b`, literalId),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalId),
		// missing: regexp literals, because of confusion with division
	}
}

// jsonRules returns syntax highlighting rules for JSON.
func jsonRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`"(\\.|[^"])*?":`, keywordId),
		mustCompile(`"(\\.|[^"])*?"`, literalId),
		mustCompile(`\b(true|false|null)\b`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([Ee][+-]?\d+)?\b`, literalId),
	}
}

// luaRules returns syntax highlighting rules for Lua.
func luaRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`(^#!|--).*$`, commentId),
		mustCompile(`\b(and|break|do|else|elseif|end|for|function|goto|`+
			`if|in|local|not|or|repeat|return|then|until|while)\b`, keywordId),
		mustCompile(`\b(assert|collectgarbage|dofile|error|_G|`+
			`(get|set)metatable|ipairs|load(file)?|next|pairs|pcall|print|`+
			`raw(equal|get|len|set)|select|to(number|string)|type|_VERSION|`+
			`xpcall|require)\b`, keywordId),
		mustCompile(`\b(false|nil|true)\b`, literalId),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalId),
		mustCompile(`\b(0[xX][0-9a-fA-F]+(\.[0-9a-fA-F]+)?|\d+(\.\d+)?)`+
			`([EePp][+-]?\d+)?\b`, literalId),
	}
}

// makefileRules returns syntax highlighting rules for makefiles.
func makefileRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`#.*$`, commentId),
		mustCompile(`\b(else|end[ei]f|ifn?def|ifn?eq|(-|s)?include|load|`+
			`override|private|(un)?export|(un)?define|vpath)\b`, keywordId),
		mustCompile(`\b(abspath|addprefix|(add)?suffix|and|basename|call|`+
			`error|eval|file|filter(-out)?|findstring|firstword|flavor|`+
			`foreach|guile|if|info|join|lastword|(not)?dir|or(igin)?|`+
			`patsubst|realpath|shell|sort|strip|subst|value|warning|wildcard|`+
			`word(s|list)?)\b`, keywordId),
		mustCompile(`\b\.(DEFAULT|DELETE_ON_ERROR|EXPORT_ALL_VARIABLES|`+
			`IGNORE|INTERMEDIATE|LOW_RESOLUTION_TIME|NOTPARALLEL|ONESHELL|`+
			`PHONY|POSIX|PRECIOUS|SECONDARY|SECONDEXPANSION|SILENT|`+
			`SUFFIXES)\b`, keywordId),
		mustCompile(`\b(DEFAULT_GOAL|\.FEATURES|\.INCLUDE_DIRS|MAKEFILE_LIST|`+
			`MAKE_RESTARTS|MAKE_TERMERR|MAKE_TERMOUT|\.RECIPEPREFIX|`+
			`\.VARIABLES)\b`, keywordId),
		mustCompile(`\b(CURDIR|MAKE(CMDGOALS|FILES|FLAGS|_HOST|LEVEL|`+
			`\.LIBPATTERNS|SHELL|SUFFIXES|_VERSION)?|SHELL|VPATH)\b`,
			keywordId),
		mustCompile(`\$(\(.+\)|\{.+\}|.)`, literalId),
	}
}

// pythonRules returns syntax highlighting rules for Python.
func pythonRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`#.*$`, commentId),
		mustCompile(`\b(and|as|assert|break|class|continue|def|del|elif|else|`+
			`except|finally|for|from|global|if|import|in|is|lambda|nonlocal|`+
			`not|or|pass|raise|return|try|while|with|yield)\b`, keywordId),
		mustCompile(`\b(abs|all|any|ascii|bin|bool|bytearray|bytes|callable|`+
			`chr|classmethod|compile|complex|delattr|dict|dir|divmod|`+
			`enumerate|eval|exec|filter|float|format|frozenset|getattr|`+
			`globals|hasattr|hash|help|hex|id|input|int|isinstance|`+
			`issubclass|iter|len|list|locals|map|max|memoryview|min|next|`+
			`object|oct|open|ord|pow|print|property|range|repr|reversed|`+
			`round|set|setattr|slice|sorted|staticmethod|str|sum|super|`+
			`tuple|type|vars|zip|__import__)\b`, keywordId),
		mustCompile(`\b(False|None|True)\b`, literalId),
		mustCompile(`(\b([rR][bB]|[bB][rR]|\b[uUrR]))?('''(\\?.)*?'''|`+
			`"""(\\?.)*?"""|"(\.|[^"])*?"|'(\.|[^'])*?')`, literalId),
		mustCompile(`\b0[bB][01]+\b`, literalId),
		mustCompile(`\b0[oO]?[0-7]+\b`, literalId),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?`+
			`([jJ](\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?)?\b`, literalId),
	}
}

// rubyRules returns syntax highlighting rules for Ruby.
func rubyRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`#.*$`, commentId),
		mustCompile(`/\*.*?\*/`, commentId), // does anyone actually use these?
		mustCompile(`\b(__ENCODING__|__LINE__|__FILE__|BEGIN|END|alias|and|`+
			`begin|break|case|class|def|defined\?|do|else|elsif|end|ensure|`+
			`for|if|in|module|next|not|or|redo|rescue|retry|return|self|`+
			`super|then|undef|unless|until|when|while|yield)\b`, keywordId),
		mustCompile(`\b(__callee__|__dir__|__method__|abort|at_exit|`+
			`autoload\??|binding|block_given\?|callcc|caller(_locations)?|`+
			`catch|chomp|chop|eval|exec|exit|fail|fork|format|gets|`+
			`(global|local)_variables|g?sub|iterator\?|lambda|load|loop|open|`+
			`p|print|s?printf|proc|put[cs]|raise|s?rand|readlines?|`+
			`require(_relative)?|select|set_trace_func|sleep|spawn|`+
			`sys(call|tem)|test|throw|(un)?trace_var|trap|warn)\b`, keywordId),
		mustCompile(`\b(false|nil|true)\b`, literalId),
		mustCompile(`\b0[bB][01_]+\b`, literalId),
		mustCompile(`\b0[oO]?[0-7_]+\b`, literalId),
		mustCompile(`\b0[dD][0-9_]+\b`, literalId),
		mustCompile(`\b0[xX][0-9a-fA-F_]+\b`, literalId),
		mustCompile(`\b([0-9][0-9_]*\.([0-9][0-9_]*)?|`+
			`([0-9][0-9_]*)?\.[0-9][0-9_]*|[[0-9][0-9_]*)`+
			`([Ee][+-]?[0-9][0-9_]*)?\b`, literalId),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalId),
		mustCompile("`(\\.|[^`])*?`", literalId),
		mustCompile(`%[iIqQrRsSwWxX](\(.*?\)|\{.*?\})`, literalId),
		// missing: regexp literals, because of confusion with division
		// missing: recognition of string interpolation: #{...}
		// should symbols be highlighted as literals?
	}
}
