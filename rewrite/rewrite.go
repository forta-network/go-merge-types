package rewrite

import (
	"regexp"
	"strings"
)

type Rule struct {
	Match     string `yaml:"match"`
	Transform string `yaml:"transform"`

	pattern  *regexp.Regexp
	compiled bool
}

func (rule *Rule) init() {
	if rule.compiled {
		return
	}
	rule.pattern = regexp.MustCompile(rule.Match)
	rule.compiled = true
}

type Rewriter []*Rule

func (rules Rewriter) Rewrite(input string) string {
	for _, rule := range rules {
		rule.init()

		results := rule.pattern.FindStringSubmatch(input)
		if len(results) == 2 {
			return strings.Replace(rule.Transform, "$", results[1], -1)
		}
	}
	return input
}
