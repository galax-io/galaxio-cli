package codegen

import (
	"strings"
	"unicode"
)

// LowerCamel converts human-readable identifiers into lowerCamelCase.
func LowerCamel(value string) string {
	words := SplitWords(value)
	if len(words) == 0 {
		return ""
	}

	for i := range words {
		words[i] = strings.ToLower(words[i])
	}

	result := words[0]
	for _, word := range words[1:] {
		result += strings.ToUpper(word[:1]) + word[1:]
	}

	return result
}

// DeriveNameWord normalizes a name into a compact lower-case token for paths.
func DeriveNameWord(value string) string {
	words := SplitWords(value)
	if len(words) == 0 {
		return "generated"
	}

	var builder strings.Builder
	for _, word := range words {
		builder.WriteString(strings.ToLower(word))
	}
	return builder.String()
}

// SplitWords breaks identifiers like "PetAdmin", "pet-admin", or "pet_admin" into words.
func SplitWords(value string) []string {
	replacer := strings.NewReplacer("/", " ", "-", " ", "_", " ", ".", " ")
	value = replacer.Replace(value)

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	words := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}

		var builder strings.Builder
		runes := []rune(part)
		for i, r := range runes {
			if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) || i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				words = appendWord(words, builder.String())
				builder.Reset()
			}
			builder.WriteRune(r)
		}
		words = appendWord(words, builder.String())
	}

	return words
}

func appendWord(words []string, word string) []string {
	word = strings.TrimSpace(word)
	if word == "" {
		return words
	}
	if strings.HasPrefix(word, "{") && strings.HasSuffix(word, "}") {
		return words
	}
	return append(words, word)
}
