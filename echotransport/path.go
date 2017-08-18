package echotransport

import (
	"regexp"
	"strings"
)

var replacePathRegex = regexp.MustCompile(`{[^}]+}`)

func replacePathParameters(path string) string {
	return replacePathRegex.ReplaceAllStringFunc(path, func(p string) string {
		p = strings.TrimPrefix(p, "{")
		p = strings.TrimSuffix(p, "}")
		return ":" + p
	})
}
