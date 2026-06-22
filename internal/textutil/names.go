package textutil

import (
	"strings"
	"unicode"
)

var lowercaseNameParticles = map[string]struct{}{
	"da":  {},
	"das": {},
	"de":  {},
	"do":  {},
	"dos": {},
	"e":   {},
}

// TitleCaseName formats a person name with the first letter of each word uppercase,
// keeping Portuguese particles (da, de, do, etc.) lowercase when not first.
// Example: "JOAO DA SILVA" → "Joao da Silva"
func TitleCaseName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	words := strings.Fields(strings.ToLower(value))
	for i, word := range words {
		if i > 0 {
			if _, isParticle := lowercaseNameParticles[word]; isParticle {
				words[i] = word
				continue
			}
		}

		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}

func TitleCaseNamePtr(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return value
	}

	formatted := TitleCaseName(*value)
	return &formatted
}
