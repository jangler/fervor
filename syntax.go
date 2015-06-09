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

// rule sets for various languages. maybe it would be better to return these
// sets as the results of functions instead of having them all compiled and
// stored at startup.
var (
	goRules = []edit.Rule{ // complete
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
	jsonRules = []edit.Rule{ // complete
		mustCompile(`"(\\.|[^"])*?":`, keywordId),
		mustCompile(`"(\\.|[^"])*?"`, literalId),
		mustCompile(`\b(true|false|null)\b`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([Ee][+-]?\d+)?\b`, literalId),
	}
	pythonRules = []edit.Rule{ // complete
		mustCompile(`#.*$`, commentId),
		mustCompile(`\b(and|as|assert|break|class|continue|def|del|elif|else|`+
			`except|finally|for|from|global|if|import|in|is|lambda|nonlocal|`+
			`not|or|pass|raise|return|try|while|with|yield)\b`, keywordId),
		mustCompile(`\b(False|None|True)\b`, literalId),
		mustCompile(`(\b([rR][bB]|[bB][rR]|\b[uUrR]))?('''(\\?.)*?'''|`+
			`"""(\\?.)*?"""|"(\.|[^"])*?"|'(\.|[^'])*?')`, literalId),
		mustCompile(`\b0[bB][01]+\b`, literalId),
		mustCompile(`\b0[oO]?[0-7]+\b`, literalId),
		mustCompile(`\b0[xX][0-9a-fA-F]+\b`, literalId),
		mustCompile(`\b(\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?`+
			`([jJ](\d+\.\d*|\d*\.\d+|\d+)([eE][+-]?\d+)?)?\b`, literalId),
	}
	bashRules = []edit.Rule{ // ?
		mustCompile(`\$#`, -1),
		mustCompile(`#.*$`, commentId),
		mustCompile(`[!:.]| \[\[? | \]\]?|\b(alias|bg|bind|break|builtin|`+
			`caller|case|cd|command|compgen|complete|compopt|continue|`+
			`declare|dirs|disown|do|done|echo|enable|esac|eval|exec|exit|`+
			`export|false|fc|fg|fi|for|function|getopts|hash|help|history|`+
			`if|in|jobs|kill|let|local|logout|mapfile|popd|printf|pushd|`+
			`pwd|read|readarray|readonly|return|select|set|shift|shopt|`+
			`source|suspend|test|time|times|trap|true|type|typeset|ulimit|`+
			`umask|unalias|unset|until|wait|while)\b`, keywordId),
		mustCompile(`\$?("(\\.|[^"])*?"|'(\\.|[^'])*?')`, literalId),
		mustCompile("`(\\.|[^`])*?`", literalId),
	}
)
