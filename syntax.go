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
		mustCompile(`/*.*?\*/`, commentId),
		mustCompile(`\b(break|case|chan|const|continue|default|defer|else|`+
			`fallthrough|for|func|go|goto|if|import|interface|map|package|`+
			`range|return|select|struct|switch|type|var)\b`, keywordId),
		mustCompile(`\b(close|len|cap|new|make|append|copy|delete|complex|`+
			`real|imag|panic|recover|print|println)\b`, keywordId),
		mustCompile(`'.*?'`, literalId),
		mustCompile(`".*?"`, literalId),
		mustCompile("`.*?`", literalId),
		mustCompile(`\b\d+\b`, literalId),
		mustCompile(`\bnil\b`, literalId),
	}
	jsonRules = []edit.Rule{
		mustCompile(`"([^"]|\\")*?":`, keywordId),
		mustCompile(`"([^"]|\\")*?"`, literalId),
		mustCompile(`\b(\d*\.)?\d+\b`, literalId),
	}
)
