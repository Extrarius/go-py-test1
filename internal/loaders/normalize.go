package loaders

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Normalize приводит текст к единому виду: UTF-8, без лишних пробелов.
func Normalize(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "\uFFFD")
	}
	// Схлопываем повторяющиеся пробелы и переносы в один пробел
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
