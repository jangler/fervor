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

var (
	goRules = []edit.Rule{
		mustCompile(`//.+?$`, commentId),
		mustCompile(`/\*.*?\*/`, commentId),
		mustCompile(`\b(break|case|chan|const|continue|default|defer|else|`+
			`fallthrough|for|func|go|goto|if|import|interface|map|package|`+
			`range|return|select|struct|switch|type|var)\b`, keywordId),
		mustCompile(`\b(append|cap|close|complex|copy|delete|imag|len|make|`+
			`new|panic|print|println|real|recover)\b`, keywordId),
		mustCompile(`\b(bool|byte|complex(64|128)|error|float(32|64)|`+
			`u?int(8|16|32|64)?|rune|string|uintptr)\b`, keywordId),
		mustCompile(`'([^']|\\')*?'`, literalId),
		mustCompile(`"([^"]|\\")*?"`, literalId),
		mustCompile("`.*?`", literalId),
		mustCompile(`\b(0[bx])?(\d*\.)?\d+\b`, literalId),
		mustCompile(`\btrue|false|iota|nil\b`, literalId),
	}
	jsonRules = []edit.Rule{
		mustCompile(`"([^"]|\\")*?":`, keywordId),
		mustCompile(`"([^"]|\\")*?"`, literalId),
		mustCompile(`\b(\d*\.)?\d+\b`, literalId),
	}
)
