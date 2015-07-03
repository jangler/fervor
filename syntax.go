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
	commentID = iota
	keywordID
	literalID
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
	"[svg]":        svgRules,
}

// bashRules returns syntax highlighting rules for Bash.
func bashRules() []edit.Rule {
	// Not sure how "complete" this is. All the builtins are accounted for, but
	// I might check out how other programs syntax highlight bash to see if
	// other things are usually highlighted.
	return []edit.Rule{
		mustCompile(`\$#`, -1),
		mustCompile(`#.*$`, commentID),
		mustCompile(`[!:.]| \[\[? | \]\]?|\b(alias|bg|bind|break|builtin|`+
			`caller|case|cd|command|compgen|complete|compopt|continue|`+
			`declare|dirs|disown|do|done|echo|elif|else|enable|esac|eval|`+
			`exec|exit|export|false|fc|fg|fi|for|function|getopts|hash|help|`+
			`history|if|in|jobs|kill|let|local|logout|mapfile|popd|printf|`+
			`pushd|pwd|read|readarray|readonly|return|select|set|shift|shopt|`+
			`source|suspend|test|then|time|times|trap|true|type|typeset|`+
			`ulimit|umask|unalias|unset|until|wait|while)\b`, keywordID),
		mustCompile(`\$?("(\\.|[^"])*?"|'(\\.|[^'])*?')`, literalID),
		mustCompile("`(\\.|[^`])*?`", literalID),
	}
}

// cRules returns syntax highlighting rules for C.
func cRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`^#(define|undef|include|if|elif|else|endif|ifdef|ifndef|`+
			`line|pragma).*$`, commentID),
		mustCompile(`//.+?$`, commentID),
		mustCompile(`/\*.*?\*/`, commentID),
		mustCompile(`\b(auto|break|case|char|const|continue|default|do|`+
			`double|else|enum|extern|float|for|goto|if|inline|int|long|`+
			`register|restrict|return|short|signed|sizeof|static|struct|`+
			`switch|typedef|union|unsigned|void|volatile|while|_Bool|`+
			`_Complex|_Imaginary)\b`, keywordID),
		mustCompile(`\b(bool|true|false|NULL)\b`, literalID),
		mustCompile(`L?'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eEpP][+-]?\d+)?`+
			`[fFlL]?\b`, literalID),
		mustCompile(`\b(0([bB][01]+|[0-7]+|[xX][0-9a-fA-F]+)|\d+)`+
			`([uU]?[lL]?|[lL]?[uU]?)\b`, literalID),
	}
}

// cssRules returns syntax highlighting rules for CSS.
func cssRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`/\*.*?\*/`, commentID),
		mustCompile(`^\S*[^:] `, keywordID),
		mustCompile(`\S+;`, literalID),
	}
}

// goRules returns syntax highlighting rules for Go.
func goRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`//.+?$`, commentID),
		mustCompile(`/\*.*?\*/`, commentID),
		mustCompile(`\b(break|case|chan|const|continue|default|defer|else|`+
			`fallthrough|for|func|go|goto|if|import|interface|map|package|`+
			`range|return|select|struct|switch|type|var)\b`, keywordID),
		mustCompile(`\b(append|cap|close|complex|copy|delete|imag|len|make|`+
			`new|panic|print|println|real|recover)\b`, keywordID),
		mustCompile(`\b(bool|byte|complex(64|128)|error|float(32|64)|`+
			`u?int(8|16|32|64)?|rune|string|uintptr)\b`, keywordID),
		mustCompile(`\b(true|false|iota|nil)\b`, literalID),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
		mustCompile("`.*?`", literalID),
		mustCompile(`\b0[bB][01]+\b`, literalID),
		mustCompile(`\b0[0-7]+\b`, literalID),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalID),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?i?\b`, literalID),
		mustCompile(`\b\d+\bi`, literalID),
	}
}

// htmlRules returns syntax highlighting rules for HTML.
func htmlRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`<!(--.*--|DOCTYPE.*)>`, commentID),
		mustCompile(`\b(a|abbr|address|area|article|aside|audio|b|base|`+
			`bd[io]|blockquote|body|br|button|canvas|caption|cite|code|`+
			`col(group)?|datalist|dd|del|details|dfn|dialog|div|dl|dt|`+
			`em(bed)?|fieldset|fig(caption|ure)|footer|form|h[123456]|`+
			`head|header|hr|html|i|iframe|img|input|ins|kbd|keygen|label|`+
			`legend|li|link|main|map|mark|menu(item)?|meta|meter|nav|`+
			`noscript|object|ol|opt(group|tion)|output|p|param|pre|progress|`+
			`q|rp|rt|ruby|s|samp|script|section|select|small|source|span|`+
			`strong|style|sub|summary|sup|table|tbody|td|textarea|tfoot|th|`+
			`thead|time|title|tr|track|u|ul|var|video|wbr)\b`, keywordID),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
	}
}

// iniRules returns syntax highlighting rules for INI files.
func iniRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`(^| )[;#].*$`, commentID),
		mustCompile(`^\[.*\]$`, keywordID),
	}
}

// javaScriptRules returns syntax highlighting rules for JavaScript.
func javaScriptRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`//.+?$`, commentID),
		mustCompile(`/\*.*?\*/`, commentID),
		mustCompile(`\b(break|case|catch|continue|debugger|default|delete|`+
			`do|else|finally|for|function|if|in|instanceof|new|return|switch|`+
			`this|throw|try|typeof|var|void|while|with)\b`, keywordID),
		mustCompile(`\b(false|NaN|null|true|undefined)\b`, literalID),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalID),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([Ee][+-]?\d+)?\b`, literalID),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
		// missing: regexp literals, because of confusion with division
	}
}

// jsonRules returns syntax highlighting rules for JSON.
func jsonRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`"(\\.|[^"])*?":`, keywordID),
		mustCompile(`"(\\.|[^"])*?"`, literalID),
		mustCompile(`\b(true|false|null)\b`, literalID),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([Ee][+-]?\d+)?\b`, literalID),
	}
}

// luaRules returns syntax highlighting rules for Lua.
func luaRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`(^#!|--).*$`, commentID),
		mustCompile(`\b(and|break|do|else|elseif|end|for|function|goto|`+
			`if|in|local|not|or|repeat|return|then|until|while)\b`, keywordID),
		mustCompile(`\b(assert|collectgarbage|dofile|error|_G|`+
			`(get|set)metatable|ipairs|load(file)?|next|pairs|pcall|print|`+
			`raw(equal|get|len|set)|select|to(number|string)|type|_VERSION|`+
			`xpcall|require)\b`, keywordID),
		mustCompile(`\b(false|nil|true)\b`, literalID),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
		mustCompile(`\b(0[xX][0-9a-fA-F]+(\.[0-9a-fA-F]+)?|\d+(\.\d+)?)`+
			`([EePp][+-]?\d+)?\b`, literalID),
	}
}

// makefileRules returns syntax highlighting rules for makefiles.
func makefileRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`#.*$`, commentID),
		mustCompile(`\b(else|end[ei]f|ifn?def|ifn?eq|(-|s)?include|load|`+
			`override|private|(un)?export|(un)?define|vpath)\b`, keywordID),
		mustCompile(`\b(abspath|addprefix|(add)?suffix|and|basename|call|`+
			`error|eval|file|filter(-out)?|findstring|firstword|flavor|`+
			`foreach|guile|if|info|join|lastword|(not)?dir|or(igin)?|`+
			`patsubst|realpath|shell|sort|strip|subst|value|warning|wildcard|`+
			`word(s|list)?)\b`, keywordID),
		mustCompile(`\b\.(DEFAULT|DELETE_ON_ERROR|EXPORT_ALL_VARIABLES|`+
			`IGNORE|INTERMEDIATE|LOW_RESOLUTION_TIME|NOTPARALLEL|ONESHELL|`+
			`PHONY|POSIX|PRECIOUS|SECONDARY|SECONDEXPANSION|SILENT|`+
			`SUFFIXES)\b`, keywordID),
		mustCompile(`\b(DEFAULT_GOAL|\.FEATURES|\.INCLUDE_DIRS|MAKEFILE_LIST|`+
			`MAKE_RESTARTS|MAKE_TERMERR|MAKE_TERMOUT|\.RECIPEPREFIX|`+
			`\.VARIABLES)\b`, keywordID),
		mustCompile(`\b(CURDIR|MAKE(CMDGOALS|FILES|FLAGS|_HOST|LEVEL|`+
			`\.LIBPATTERNS|SHELL|SUFFIXES|_VERSION)?|SHELL|VPATH)\b`,
			keywordID),
		mustCompile(`\$(\(.+\)|\{.+\}|.)`, literalID),
	}
}

// pythonRules returns syntax highlighting rules for Python.
func pythonRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`#.*$`, commentID),
		mustCompile(`\b(and|as|assert|break|class|continue|def|del|elif|else|`+
			`except|finally|for|from|global|if|import|in|is|lambda|nonlocal|`+
			`not|or|pass|raise|return|try|while|with|yield)\b`, keywordID),
		mustCompile(`\b(abs|all|any|ascii|bin|bool|bytearray|bytes|callable|`+
			`chr|classmethod|compile|complex|delattr|dict|dir|divmod|`+
			`enumerate|eval|exec|filter|float|format|frozenset|getattr|`+
			`globals|hasattr|hash|help|hex|id|input|int|isinstance|`+
			`issubclass|iter|len|list|locals|map|max|memoryview|min|next|`+
			`object|oct|open|ord|pow|print|property|range|repr|reversed|`+
			`round|set|setattr|slice|sorted|staticmethod|str|sum|super|`+
			`tuple|type|vars|zip|__import__)\b`, keywordID),
		mustCompile(`\b(False|None|True)\b`, literalID),
		mustCompile(`(\b([rR][bB]|[bB][rR]|\b[uUrR]))?('''(\\?.)*?'''|`+
			`"""(\\?.)*?"""|"(\.|[^"])*?"|'(\.|[^'])*?')`, literalID),
		mustCompile(`\b0[bB][01]+\b`, literalID),
		mustCompile(`\b0[oO]?[0-7]+\b`, literalID),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalID),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?`+
			`([jJ](\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?)?\b`, literalID),
	}
}

// rubyRules returns syntax highlighting rules for Ruby.
func rubyRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`#.*$`, commentID),
		mustCompile(`/\*.*?\*/`, commentID), // does anyone actually use these?
		mustCompile(`\b(__ENCODING__|__LINE__|__FILE__|BEGIN|END|alias|and|`+
			`begin|break|case|class|def|defined\?|do|else|elsif|end|ensure|`+
			`for|if|in|module|next|not|or|redo|rescue|retry|return|self|`+
			`super|then|undef|unless|until|when|while|yield)\b`, keywordID),
		mustCompile(`\b(__callee__|__dir__|__method__|abort|at_exit|`+
			`autoload\??|binding|block_given\?|callcc|caller(_locations)?|`+
			`catch|chomp|chop|eval|exec|exit|fail|fork|format|gets|`+
			`(global|local)_variables|g?sub|iterator\?|lambda|load|loop|open|`+
			`p|print|s?printf|proc|put[cs]|raise|s?rand|readlines?|`+
			`require(_relative)?|select|set_trace_func|sleep|spawn|`+
			`sys(call|tem)|test|throw|(un)?trace_var|trap|warn)\b`, keywordID),
		mustCompile(`\b(false|nil|true)\b`, literalID),
		mustCompile(`\b0[bB][01_]+\b`, literalID),
		mustCompile(`\b0[oO]?[0-7_]+\b`, literalID),
		mustCompile(`\b0[dD][0-9_]+\b`, literalID),
		mustCompile(`\b0[xX][0-9a-fA-F_]+\b`, literalID),
		mustCompile(`\b([0-9][0-9_]*\.([0-9][0-9_]*)?|`+
			`([0-9][0-9_]*)?\.[0-9][0-9_]*|[[0-9][0-9_]*)`+
			`([Ee][+-]?[0-9][0-9_]*)?\b`, literalID),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
		mustCompile("`(\\.|[^`])*?`", literalID),
		mustCompile(`%[iIqQrRsSwWxX](\(.*?\)|\{.*?\})`, literalID),
		// missing: regexp literals, because of confusion with division
		// missing: recognition of string interpolation: #{...}
		// should symbols be highlighted as literals?
	}
}

// svgRules returns syntax highlighting rules for SVG.
func svgRules() []edit.Rule {
	return []edit.Rule{
		mustCompile(`<!--.*-->`, commentID),
		mustCompile(`\b(a|altGlyph(Def|Item)?|animate(Motion|Transform)?|`+
			`circle|clipPath|color-profile|cursor|defs|desc|ellipse|`+
			`fe(Blend|(Color|Convolve)Matrix|ComponentTransfer|Composite|`+
			`(Diffuse|Specular)Lighting|DisplacementMap|`+
			`(Distant|Point|Spot)Light|Flood|Func[ABGR]|GaussianBlur|Image`+
			`|Merge(Node)?|Morphology|Offset|Tile|Turbulence)|filter|`+
			`font(-face(-format|-name|-src|-uri)?)?|foreignObject|g|`+
			`glyph(Ref)?|hkern|image|line(arGradient)?|marker|mask|metadata|`+
			`missing-glyph|m?path|pattern|poly(gon|line)|radialGradient|rect|`+
			`script|set|stop|style|svg|switch|symbol|text(Path)?|title|tref|`+
			`tspan|use|view|vkern)\b`, keywordID),
		mustCompile(`'(\\.|[^'])*?'|"(\\.|[^"])*?"`, literalID),
	}
}
